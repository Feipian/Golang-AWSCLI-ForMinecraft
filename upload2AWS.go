package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

func base64Encode(s string) string {
	return base64.StdEncoding.EncodeToString([]byte(s))
}

// waitForActiveInternetRoute polls the route table until the 0.0.0.0/0 route
// referencing the given IGW is Active (not blackhole). If a blackhole is
// observed, it will try to delete and recreate the route. Times out after ~30s.
func waitForActiveInternetRoute(ctx context.Context, ec2Client *ec2.Client, rtID, igwID string) error {
	for i := 0; i < 10; i++ { // ~30s max
		desc, err := ec2Client.DescribeRouteTables(ctx, &ec2.DescribeRouteTablesInput{RouteTableIds: []string{rtID}})
		if err != nil || len(desc.RouteTables) == 0 {
			return fmt.Errorf("failed to read route table during wait: %v", err)
		}
		var route *ec2types.Route
		for idx := range desc.RouteTables[0].Routes {
			r := &desc.RouteTables[0].Routes[idx]
			if aws.ToString(r.DestinationCidrBlock) == "0.0.0.0/0" {
				route = r
				break
			}
		}
		if route != nil {
			// If already active and points at our IGW, we're done
			if route.State == ec2types.RouteStateActive && aws.ToString(route.GatewayId) == igwID {
				return nil
			}
			// If blackhole, try to recreate the route to force activation
			if route.State == ec2types.RouteStateBlackhole {
				_, _ = ec2Client.DeleteRoute(ctx, &ec2.DeleteRouteInput{
					RouteTableId:         aws.String(rtID),
					DestinationCidrBlock: aws.String("0.0.0.0/0"),
				})
				_, err := ec2Client.CreateRoute(ctx, &ec2.CreateRouteInput{
					RouteTableId:         aws.String(rtID),
					DestinationCidrBlock: aws.String("0.0.0.0/0"),
					GatewayId:            aws.String(igwID),
				})
				if err != nil {
					return fmt.Errorf("failed to recreate default route: %v", err)
				}
			}
		}
		time.Sleep(3 * time.Second)
	}
	return fmt.Errorf("route to IGW did not become Active in time")
}

// getPublicIPv4 queries an external service to discover the caller's public IPv4.
// Returns a plain IPv4 string like "203.0.113.10".
func getPublicIPv4(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://checkip.amazonaws.com", nil)
	if err != nil {
		return "", err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	ip := strings.TrimSpace(string(b))
	return ip, nil
}

func main() {
	ctx := context.TODO()

	// Load AWS config
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		log.Fatalf("failed to load AWS config: %v", err)
	}

	ec2Client := ec2.NewFromConfig(cfg)

	// 1. Find or create VPC tagged Name=minecraft-vpc
	var vpcID string
	existingVpcs, err := ec2Client.DescribeVpcs(ctx, &ec2.DescribeVpcsInput{
		Filters: []ec2types.Filter{
			{Name: aws.String("tag:Name"), Values: []string{"minecraft-vpc"}},
		},
	})
	if err != nil {
		log.Fatalf("failed to describe VPCs: %v", err)
	}

	if len(existingVpcs.Vpcs) > 0 {
		vpcID = *existingVpcs.Vpcs[0].VpcId
		fmt.Println("Found existing VPC:", vpcID)
	} else {
		vpcOut, err := ec2Client.CreateVpc(ctx, &ec2.CreateVpcInput{
			CidrBlock: aws.String("10.0.0.0/16"), // Private IP range
			TagSpecifications: []ec2types.TagSpecification{
				{
					ResourceType: ec2types.ResourceTypeVpc,
					Tags: []ec2types.Tag{
						{Key: aws.String("Name"), Value: aws.String("minecraft-vpc")},
					},
				},
			},
		})
		if err != nil {
			log.Fatalf("failed to create VPC: %v", err)
		}
		vpcID = *vpcOut.Vpc.VpcId
		fmt.Println("Created VPC:", vpcID)
	}

	// 2. Find or create Internet Gateway for this VPC and ensure it has a Name tag
	var igwID string
	igwDesc, err := ec2Client.DescribeInternetGateways(ctx, &ec2.DescribeInternetGatewaysInput{
		Filters: []ec2types.Filter{
			{Name: aws.String("attachment.vpc-id"), Values: []string{vpcID}},
		},
	})
	if err != nil {
		log.Fatalf("failed to describe IGWs: %v", err)
	}

	if len(igwDesc.InternetGateways) > 0 {
		igwID = *igwDesc.InternetGateways[0].InternetGatewayId
		fmt.Println("Found existing Internet Gateway:", igwID)
		// Ensure it has Name tag
		hasName := false
		for _, t := range igwDesc.InternetGateways[0].Tags {
			if aws.ToString(t.Key) == "Name" {
				hasName = true
				break
			}
		}
		if !hasName {
			_, err = ec2Client.CreateTags(ctx, &ec2.CreateTagsInput{
				Resources: []string{igwID},
				Tags:      []ec2types.Tag{{Key: aws.String("Name"), Value: aws.String("minecraft-igw")}},
			})
			if err != nil {
				log.Fatalf("failed to tag existing IGW: %v", err)
			}
		}
	} else {
		igwOut, err := ec2Client.CreateInternetGateway(ctx, &ec2.CreateInternetGatewayInput{
			TagSpecifications: []ec2types.TagSpecification{
				{
					ResourceType: ec2types.ResourceTypeInternetGateway,
					Tags:         []ec2types.Tag{{Key: aws.String("Name"), Value: aws.String("minecraft-igw")}},
				},
			},
		})
		if err != nil {
			log.Fatalf("failed to create IGW: %v", err)
		}
		igwID = *igwOut.InternetGateway.InternetGatewayId
		fmt.Println("Created Internet Gateway:", igwID)

		// Attach IGW to VPC
		_, err = ec2Client.AttachInternetGateway(ctx, &ec2.AttachInternetGatewayInput{
			InternetGatewayId: aws.String(igwID),
			VpcId:             aws.String(vpcID),
		})
		if err != nil {
			log.Fatalf("failed to attach IGW: %v", err)
		}
	}

	// 3. Find or create Subnet in this VPC; ensure Name tag
	var subnetID string
	subnetCidr := "10.0.1.0/24"
	// Try find by VPC + CIDR first
	existingSubnets, err := ec2Client.DescribeSubnets(ctx, &ec2.DescribeSubnetsInput{
		Filters: []ec2types.Filter{
			{Name: aws.String("vpc-id"), Values: []string{vpcID}},
			{Name: aws.String("cidr-block"), Values: []string{subnetCidr}},
		},
	})
	if err != nil {
		log.Fatalf("failed to describe subnets: %v", err)
	}

	if len(existingSubnets.Subnets) == 0 {
		// Fallback: search by Name tag within VPC
		existingSubnets, err = ec2Client.DescribeSubnets(ctx, &ec2.DescribeSubnetsInput{
			Filters: []ec2types.Filter{
				{Name: aws.String("vpc-id"), Values: []string{vpcID}},
				{Name: aws.String("tag:Name"), Values: []string{"minecraft-subnet"}},
			},
		})
		if err != nil {
			log.Fatalf("failed to describe subnets by tag: %v", err)
		}
	}

	if len(existingSubnets.Subnets) > 0 {
		subnet := existingSubnets.Subnets[0]
		subnetID = *subnet.SubnetId
		fmt.Println("Found existing Subnet:", subnetID)
		// Ensure Name tag
		hasName := false
		for _, t := range subnet.Tags {
			if aws.ToString(t.Key) == "Name" {
				hasName = true
				break
			}
		}
		if !hasName {
			_, err = ec2Client.CreateTags(ctx, &ec2.CreateTagsInput{
				Resources: []string{subnetID},
				Tags:      []ec2types.Tag{{Key: aws.String("Name"), Value: aws.String("minecraft-subnet")}},
			})
			if err != nil {
				log.Fatalf("failed to tag existing subnet: %v", err)
			}
		}
	} else {
		subnetOut, err := ec2Client.CreateSubnet(ctx, &ec2.CreateSubnetInput{
			VpcId:            aws.String(vpcID),
			CidrBlock:        aws.String(subnetCidr),
			AvailabilityZone: aws.String("ap-northeast-2c"),
			TagSpecifications: []ec2types.TagSpecification{
				{
					ResourceType: ec2types.ResourceTypeSubnet,
					Tags:         []ec2types.Tag{{Key: aws.String("Name"), Value: aws.String("minecraft-subnet")}},
				},
			},
		})
		if err != nil {
			log.Fatalf("failed to create subnet: %v", err)
		}
		subnetID = *subnetOut.Subnet.SubnetId
		fmt.Println("Created Subnet:", subnetID)
	}

	// 4. Find or create Route Table; ensure default route to IGW; associate to subnet
	var rtID string
	// Try find by VPC + Name tag
	rtDesc, err := ec2Client.DescribeRouteTables(ctx, &ec2.DescribeRouteTablesInput{
		Filters: []ec2types.Filter{
			{Name: aws.String("vpc-id"), Values: []string{vpcID}},
			{Name: aws.String("tag:Name"), Values: []string{"minecraft-rt"}},
		},
	})
	if err != nil {
		log.Fatalf("failed to describe route tables: %v", err)
	}
	if len(rtDesc.RouteTables) > 0 {
		rt := rtDesc.RouteTables[0]
		rtID = *rt.RouteTableId
		fmt.Println("Found existing Route Table:", rtID)
		// Ensure Name tag exists
		hasName := false
		for _, t := range rt.Tags {
			if aws.ToString(t.Key) == "Name" {
				hasName = true
				break
			}
		}
		if !hasName {
			_, err = ec2Client.CreateTags(ctx, &ec2.CreateTagsInput{
				Resources: []string{rtID},
				Tags:      []ec2types.Tag{{Key: aws.String("Name"), Value: aws.String("minecraft-rt")}},
			})
			if err != nil {
				log.Fatalf("failed to tag existing route table: %v", err)
			}
		}
	} else {
		rtOut, err := ec2Client.CreateRouteTable(ctx, &ec2.CreateRouteTableInput{
			VpcId: aws.String(vpcID),
			TagSpecifications: []ec2types.TagSpecification{
				{
					ResourceType: ec2types.ResourceTypeRouteTable,
					Tags:         []ec2types.Tag{{Key: aws.String("Name"), Value: aws.String("minecraft-rt")}},
				},
			},
		})
		if err != nil {
			log.Fatalf("failed to create route table: %v", err)
		}
		rtID = *rtOut.RouteTable.RouteTableId
		fmt.Println("Created Route Table:", rtID)
	}

	// Ensure default route 0.0.0.0/0 via IGW exists
	rtSingleDesc, err := ec2Client.DescribeRouteTables(ctx, &ec2.DescribeRouteTablesInput{
		RouteTableIds: []string{rtID},
	})
	if err != nil || len(rtSingleDesc.RouteTables) == 0 {
		log.Fatalf("failed to read route table %s: %v", rtID, err)
	}
	hasDefault := false
	for _, r := range rtSingleDesc.RouteTables[0].Routes {
		if aws.ToString(r.DestinationCidrBlock) == "0.0.0.0/0" {
			hasDefault = true
			break
		}
	}
	if !hasDefault {
		// Ensure IGW is attached before creating route
		_, err = ec2Client.AttachInternetGateway(ctx, &ec2.AttachInternetGatewayInput{
			InternetGatewayId: aws.String(igwID),
			VpcId:             aws.String(vpcID),
		})
		if err != nil {
			log.Printf("IGW may already be attached: %v", err)
		}

		// Create the default route
		_, err = ec2Client.CreateRoute(ctx, &ec2.CreateRouteInput{
			RouteTableId:         aws.String(rtID),
			DestinationCidrBlock: aws.String("0.0.0.0/0"),
			GatewayId:            aws.String(igwID),
		})
		if err != nil {
			log.Fatalf("failed to create default route: %v", err)
		}
		fmt.Println("Added default route to:", rtID)
	}
	// Extra safety: wait until route is Active; recreate if blackhole
	if err := waitForActiveInternetRoute(ctx, ec2Client, rtID, igwID); err != nil {
		log.Fatalf("default route did not become active: %v", err)
	}

	// Ensure association with subnet
	assocDesc, err := ec2Client.DescribeRouteTables(ctx, &ec2.DescribeRouteTablesInput{
		Filters: []ec2types.Filter{
			{Name: aws.String("association.subnet-id"), Values: []string{subnetID}},
		},
	})
	if err != nil {
		log.Fatalf("failed to describe route table associations: %v", err)
	}
	if len(assocDesc.RouteTables) == 0 {
		// Not associated yet → associate
		_, err = ec2Client.AssociateRouteTable(ctx, &ec2.AssociateRouteTableInput{
			RouteTableId: aws.String(rtID),
			SubnetId:     aws.String(subnetID),
		})
		if err != nil {
			log.Fatalf("failed to associate route table: %v", err)
		}
		fmt.Println("Associated subnet with Route Table:", rtID)
	} else {
		// Already associated. Replace if different
		currentRT := assocDesc.RouteTables[0]
		currentAssocId := aws.ToString(currentRT.Associations[0].RouteTableAssociationId)
		if aws.ToString(currentRT.RouteTableId) != rtID {
			_, err = ec2Client.ReplaceRouteTableAssociation(ctx, &ec2.ReplaceRouteTableAssociationInput{
				AssociationId: aws.String(currentAssocId),
				RouteTableId:  aws.String(rtID),
			})
			if err != nil {
				log.Fatalf("failed to replace route table association: %v", err)
			}
			fmt.Println("Replaced Route Table association to:", rtID)
		}
	}

	// 5. Create or reuse Security Group with ports for SSH (22) and Minecraft (25565)
	var sgID string
	sgName := "minecraft-sg"
	sgDesc := "Security group for PaperMC and SSH"

	// Detect caller public IPv4 and build CIDR /32
	clientIP := ""
	if ip, err := getPublicIPv4(ctx); err == nil && ip != "" {
		clientIP = ip + "/32"
		fmt.Println("Restricting access to:", clientIP)
	} else {
		fmt.Println("Warning: could not detect public IP; will fallback to 0.0.0.0/0 rules")
	}
	sgDescOut, err := ec2Client.DescribeSecurityGroups(ctx, &ec2.DescribeSecurityGroupsInput{
		Filters: []ec2types.Filter{
			{Name: aws.String("vpc-id"), Values: []string{vpcID}},
			{Name: aws.String("group-name"), Values: []string{sgName}},
		},
	})
	if err != nil {
		log.Fatalf("failed to describe security groups: %v", err)
	}
	if len(sgDescOut.SecurityGroups) > 0 {
		sgID = aws.ToString(sgDescOut.SecurityGroups[0].GroupId)
		fmt.Println("Found existing Security Group:", sgID)
	} else {
		createSgOut, err := ec2Client.CreateSecurityGroup(ctx, &ec2.CreateSecurityGroupInput{
			VpcId:       aws.String(vpcID),
			GroupName:   aws.String(sgName),
			Description: aws.String(sgDesc),
			TagSpecifications: []ec2types.TagSpecification{
				{ResourceType: ec2types.ResourceTypeSecurityGroup, Tags: []ec2types.Tag{{Key: aws.String("Name"), Value: aws.String(sgName)}}},
			},
		})
		if err != nil {
			log.Fatalf("failed to create security group: %v", err)
		}
		sgID = aws.ToString(createSgOut.GroupId)
		fmt.Println("Created Security Group:", sgID)
	}

	// Authorize needed ingress rules, prefer restricting to caller IP if available
	needSSHv4 := true
	needMCv4 := true
	needHTTPv4 := true
	// Read current rules
	sgRead, err := ec2Client.DescribeSecurityGroups(ctx, &ec2.DescribeSecurityGroupsInput{GroupIds: []string{sgID}})
	if err == nil && len(sgRead.SecurityGroups) > 0 {
		for _, p := range sgRead.SecurityGroups[0].IpPermissions {
			from := aws.ToInt32(p.FromPort)
			to := aws.ToInt32(p.ToPort)
			proto := aws.ToString(p.IpProtocol)
			if proto == "tcp" && from <= 22 && to >= 22 {
				for _, r := range p.IpRanges {
					cidr := aws.ToString(r.CidrIp)
					if clientIP != "" && cidr == clientIP {
						needSSHv4 = false
					}
					if clientIP == "" && cidr == "0.0.0.0/0" {
						needSSHv4 = false
					}
				}
			}
			if proto == "tcp" && from <= 25565 && to >= 25565 {
				for _, r := range p.IpRanges {
					cidr := aws.ToString(r.CidrIp)
					if clientIP != "" && cidr == clientIP {
						needMCv4 = false
					}
					if clientIP == "" && cidr == "0.0.0.0/0" {
						needMCv4 = false
					}
				}
			}
			if proto == "tcp" && from <= 80 && to >= 80 {
				for _, r := range p.IpRanges {
					cidr := aws.ToString(r.CidrIp)
					if clientIP != "" && cidr == clientIP {
						needHTTPv4 = false
					}
					if clientIP == "" && cidr == "0.0.0.0/0" {
						needHTTPv4 = false
					}
				}
			}
		}
	}

	// If we detected caller IP, revoke any wide-open rules for 22/80/25565 (IPv4 and IPv6)
	if clientIP != "" && err == nil && len(sgRead.SecurityGroups) > 0 {
		revokePerms := []ec2types.IpPermission{}
		for _, p := range sgRead.SecurityGroups[0].IpPermissions {
			from := aws.ToInt32(p.FromPort)
			to := aws.ToInt32(p.ToPort)
			proto := aws.ToString(p.IpProtocol)
			if proto == "tcp" && ((from <= 22 && to >= 22) || (from <= 80 && to >= 80) || (from <= 25565 && to >= 25565)) {
				perm := ec2types.IpPermission{IpProtocol: aws.String("tcp"), FromPort: p.FromPort, ToPort: p.ToPort}
				for _, r := range p.IpRanges {
					if aws.ToString(r.CidrIp) == "0.0.0.0/0" {
						perm.IpRanges = append(perm.IpRanges, ec2types.IpRange{CidrIp: aws.String("0.0.0.0/0")})
					}
				}
				for _, r := range p.Ipv6Ranges {
					if aws.ToString(r.CidrIpv6) == "::/0" {
						perm.Ipv6Ranges = append(perm.Ipv6Ranges, ec2types.Ipv6Range{CidrIpv6: aws.String("::/0")})
					}
				}
				if len(perm.IpRanges) > 0 || len(perm.Ipv6Ranges) > 0 {
					revokePerms = append(revokePerms, perm)
				}
			}
		}
		if len(revokePerms) > 0 {
			_, rerr := ec2Client.RevokeSecurityGroupIngress(ctx, &ec2.RevokeSecurityGroupIngressInput{
				GroupId:       aws.String(sgID),
				IpPermissions: revokePerms,
			})
			if rerr != nil {
				log.Printf("warning: failed to revoke wide-open ingress: %v", rerr)
			} else {
				fmt.Println("Revoked wide-open rules (0.0.0.0/0, ::/0) for 22/80/25565")
			}
		}
	}
	var ipPerms []ec2types.IpPermission
	if needSSHv4 {
		perm := ec2types.IpPermission{IpProtocol: aws.String("tcp"), FromPort: aws.Int32(22), ToPort: aws.Int32(22)}
		cidr := "0.0.0.0/0"
		if clientIP != "" {
			cidr = clientIP
		}
		perm.IpRanges = append(perm.IpRanges, ec2types.IpRange{CidrIp: aws.String(cidr), Description: aws.String("SSH IPv4")})
		ipPerms = append(ipPerms, perm)
	}
	if needMCv4 {
		perm := ec2types.IpPermission{IpProtocol: aws.String("tcp"), FromPort: aws.Int32(25565), ToPort: aws.Int32(25565)}
		cidr := "0.0.0.0/0"
		if clientIP != "" {
			cidr = clientIP
		}
		perm.IpRanges = append(perm.IpRanges, ec2types.IpRange{CidrIp: aws.String(cidr), Description: aws.String("Minecraft IPv4")})
		ipPerms = append(ipPerms, perm)
	}
	if needHTTPv4 {
		perm := ec2types.IpPermission{IpProtocol: aws.String("tcp"), FromPort: aws.Int32(80), ToPort: aws.Int32(80)}
		cidr := "0.0.0.0/0"
		if clientIP != "" {
			cidr = clientIP
		}
		perm.IpRanges = append(perm.IpRanges, ec2types.IpRange{CidrIp: aws.String(cidr), Description: aws.String("HTTP IPv4")})
		ipPerms = append(ipPerms, perm)
	}
	if len(ipPerms) > 0 {
		_, err = ec2Client.AuthorizeSecurityGroupIngress(ctx, &ec2.AuthorizeSecurityGroupIngressInput{
			GroupId:       aws.String(sgID),
			IpPermissions: ipPerms,
		})
		if err != nil {
			log.Fatalf("failed to update SG ingress: %v", err)
		}
		fmt.Println("Updated SG ingress rules on:", sgID)
	}

	// 6. Use the same AMI as your working instance
	amiID := "ami-0ae2c887094315bed" // al2023-ami-2023.8.20250818.0-kernel-6.1-x86_64
	fmt.Println("Using AMI:", amiID)

	// 7. Launch or reuse EC2 instance
	instName := "minecraft-ec2"
	// Try to find existing running/stopped instance with tag Name
	instDesc, err := ec2Client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
		Filters: []ec2types.Filter{
			{Name: aws.String("tag:Name"), Values: []string{instName}},
			{Name: aws.String("vpc-id"), Values: []string{vpcID}},
		},
	})
	if err != nil {
		log.Fatalf("failed to describe instances: %v", err)
	}
	var instanceID string
	var publicIP string
	foundInstance := false
	for _, r := range instDesc.Reservations {
		for _, inst := range r.Instances {
			state := inst.State
			if state != nil && (state.Name == ec2types.InstanceStateNameRunning || state.Name == ec2types.InstanceStateNameStopped) {
				instanceID = aws.ToString(inst.InstanceId)
				if inst.PublicIpAddress != nil {
					publicIP = aws.ToString(inst.PublicIpAddress)
				}
				foundInstance = true
				break
			}
		}
		if foundInstance {
			break
		}
	}

	if !foundInstance {
		userData := `#!/bin/bash
set -euxo pipefail

# Step 1: Update system and install Java
dnf update -y
dnf install -y java-21-amazon-corretto-headless --allowerasing

# Step 2: Create PaperMC directory
mkdir -p /opt/papermc
cd /opt/papermc

# Step 3: Download your specific PaperMC jar
echo "Downloading PaperMC 1.21.4-232..."
curl -L -o paper.jar "https://fill-data.papermc.io/v1/objects/5ee4f542f628a14c644410b08c94ea42e772ef4d29fe92973636b6813d4eaffc/paper-1.21.4-232.jar"

# Step 4: Accept EULA
echo "eula=true" > eula.txt

# Step 5: Create systemd service
cat > /etc/systemd/system/papermc.service << 'UNIT'
[Unit]
Description=PaperMC Server
After=network.target

[Service]
WorkingDirectory=/opt/papermc
ExecStart=/usr/bin/java -Xms1G -Xmx1536M -jar /opt/papermc/paper.jar nogui
Restart=always
User=root

[Install]
WantedBy=multi-user.target
UNIT

# Step 6: Start the service
systemctl daemon-reload
systemctl enable --now papermc

echo "PaperMC server setup complete!"
`

		// Set key pair name for SSH access (change to your key pair name)
		keyName := "minecraft-key" // Use the EXACT name from AWS Console

		runOut, err := ec2Client.RunInstances(ctx, &ec2.RunInstancesInput{
			ImageId:      aws.String(amiID),
			InstanceType: ec2types.InstanceTypeT3Small,
			MinCount:     aws.Int32(1),
			MaxCount:     aws.Int32(1),
			NetworkInterfaces: []ec2types.InstanceNetworkInterfaceSpecification{
				{
					AssociatePublicIpAddress: aws.Bool(true),
					DeviceIndex:              aws.Int32(0),
					SubnetId:                 aws.String(subnetID),
					Groups:                   []string{sgID},
				},
			},
			UserData: aws.String(base64Encode(userData)),
			TagSpecifications: []ec2types.TagSpecification{{
				ResourceType: ec2types.ResourceTypeInstance,
				Tags:         []ec2types.Tag{{Key: aws.String("Name"), Value: aws.String(instName)}},
			}},
			KeyName: func() *string {
				if keyName == "" {
					return nil
				}
				return aws.String(keyName)
			}(),
		})
		if err != nil || len(runOut.Instances) == 0 {
			log.Fatalf("failed to run instance: %v", err)
		}
		instanceID = aws.ToString(runOut.Instances[0].InstanceId)
		fmt.Println("Launched EC2 instance:", instanceID)
		// Wait until running
		waiter := ec2.NewInstanceRunningWaiter(ec2Client)
		if err := waiter.Wait(ctx, &ec2.DescribeInstancesInput{InstanceIds: []string{instanceID}}, 5*time.Minute); err != nil {
			log.Fatalf("instance did not enter running state: %v", err)
		}
	}

	// Ensure the instance has an Elastic IP associated
	// 1) Check if an Elastic IP is already associated with this instance
	addrDesc, err := ec2Client.DescribeAddresses(ctx, &ec2.DescribeAddressesInput{
		Filters: []ec2types.Filter{
			{Name: aws.String("instance-id"), Values: []string{instanceID}},
		},
	})
	if err != nil {
		log.Printf("warning: failed to describe addresses: %v", err)
	}

	var allocationID string
	var elasticIP string
	if err == nil && len(addrDesc.Addresses) > 0 {
		// Already has an EIP
		allocationID = aws.ToString(addrDesc.Addresses[0].AllocationId)
		elasticIP = aws.ToString(addrDesc.Addresses[0].PublicIp)
		fmt.Println("Found existing Elastic IP:", elasticIP)
	} else {
		// 2) Allocate a new Elastic IP in the VPC domain
		allocOut, aerr := ec2Client.AllocateAddress(ctx, &ec2.AllocateAddressInput{Domain: ec2types.DomainTypeVpc})
		if aerr != nil {
			log.Fatalf("failed to allocate Elastic IP: %v", aerr)
		}
		allocationID = aws.ToString(allocOut.AllocationId)
		elasticIP = aws.ToString(allocOut.PublicIp)
		fmt.Println("Allocated Elastic IP:", elasticIP)

		// 3) Associate the Elastic IP to our instance
		_, asErr := ec2Client.AssociateAddress(ctx, &ec2.AssociateAddressInput{
			InstanceId:   aws.String(instanceID),
			AllocationId: aws.String(allocationID),
		})
		if asErr != nil {
			log.Fatalf("failed to associate Elastic IP: %v", asErr)
		}
		fmt.Println("Associated Elastic IP to instance:", instanceID)
	}

	// Fetch public IP (prefer the Elastic IP if present)
	instOut, err := ec2Client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{InstanceIds: []string{instanceID}})
	if err == nil && len(instOut.Reservations) > 0 && len(instOut.Reservations[0].Instances) > 0 {
		if instOut.Reservations[0].Instances[0].PublicIpAddress != nil {
			publicIP = aws.ToString(instOut.Reservations[0].Instances[0].PublicIpAddress)
		}
	}
	if elasticIP != "" {
		publicIP = elasticIP
	}
	if publicIP != "" {
		fmt.Println("PaperMC public address:", publicIP+":25565")
		fmt.Println("SSH:", "ssh ec2-user@"+publicIP)
	} else {
		fmt.Println("Instance public IP not yet available. Check AWS console.")
	}

	// fmt.Println("✅ VPC setup complete. Use VPC:", vpcID, "Subnet:", subnetID)
}
