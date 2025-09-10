package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

func main() {
	ctx := context.TODO()

	// Load AWS config
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		log.Fatalf("failed to load AWS config: %v", err)
	}

	ec2Client := ec2.NewFromConfig(cfg)

	fmt.Println("🔍 Checking Minecraft Server Status...")
	fmt.Println("==================================================")

	// 1. Find the Minecraft instance
	instName := "minecraft-ec2"
	instDesc, err := ec2Client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
		Filters: []ec2types.Filter{
			{Name: aws.String("tag:Name"), Values: []string{instName}},
		},
	})
	if err != nil {
		log.Fatalf("failed to describe instances: %v", err)
	}

	var instanceID, publicIP, privateIP string
	var instanceState string
	foundInstance := false

	for _, r := range instDesc.Reservations {
		for _, inst := range r.Instances {
			state := inst.State
			if state != nil {
				instanceID = aws.ToString(inst.InstanceId)
				instanceState = string(state.Name)
				if inst.PublicIpAddress != nil {
					publicIP = aws.ToString(inst.PublicIpAddress)
				}
				if inst.PrivateIpAddress != nil {
					privateIP = aws.ToString(inst.PrivateIpAddress)
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
		fmt.Println("❌ No Minecraft instance found!")
		return
	}

	// 2. Display instance information
	fmt.Printf("🖥️  Instance ID: %s\n", instanceID)
	fmt.Printf("🌐 Public IP: %s\n", publicIP)
	fmt.Printf("🏠 Private IP: %s\n", privateIP)
	fmt.Printf("📊 State: %s\n", instanceState)
	fmt.Println()

	// 3. Check instance status
	if instanceState != string(ec2types.InstanceStateNameRunning) {
		fmt.Printf("⚠️  Instance is not running (State: %s)\n", instanceState)
		return
	}

	// 4. Check Minecraft port connectivity
	fmt.Println("🎮 Testing Minecraft Server Connectivity...")
	if publicIP != "" {
		// Test port 25565
		conn, err := net.DialTimeout("tcp", publicIP+":25565", 5*time.Second)
		if err != nil {
			fmt.Printf("❌ Cannot connect to Minecraft server on %s:25565\n", publicIP)
			fmt.Printf("   Error: %v\n", err)
		} else {
			conn.Close()
			fmt.Printf("✅ Minecraft server is reachable on %s:25565\n", publicIP)
		}
	} else {
		fmt.Println("⚠️  No public IP address found")
	}

	// 5. Check security group rules
	fmt.Println("\n🔒 Checking Security Group Rules...")
	sgDesc, err := ec2Client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
		InstanceIds: []string{instanceID},
	})
	if err == nil && len(sgDesc.Reservations) > 0 && len(sgDesc.Reservations[0].Instances) > 0 {
		instance := sgDesc.Reservations[0].Instances[0]
		if len(instance.SecurityGroups) > 0 {
			sgID := aws.ToString(instance.SecurityGroups[0].GroupId)
			fmt.Printf("🛡️  Security Group: %s\n", sgID)

			// Get security group details
			sgDetails, err := ec2Client.DescribeSecurityGroups(ctx, &ec2.DescribeSecurityGroupsInput{
				GroupIds: []string{sgID},
			})
			if err == nil && len(sgDetails.SecurityGroups) > 0 {
				sg := sgDetails.SecurityGroups[0]
				fmt.Println("📋 Inbound Rules:")
				for _, rule := range sg.IpPermissions {
					fromPort := aws.ToInt32(rule.FromPort)
					toPort := aws.ToInt32(rule.ToPort)
					protocol := aws.ToString(rule.IpProtocol)

					if fromPort == 25565 && toPort == 25565 && protocol == "tcp" {
						fmt.Printf("   ✅ Minecraft (25565/tcp) - OPEN\n")
					} else if fromPort == 22 && toPort == 22 && protocol == "tcp" {
						fmt.Printf("   ✅ SSH (22/tcp) - OPEN\n")
					}
				}
			}
		}
	}

	// 6. Summary and connection info
	fmt.Println("\n==================================================")
	fmt.Println("📋 SUMMARY:")
	fmt.Printf("   Instance: %s (%s)\n", instanceID, instanceState)
	if publicIP != "" {
		fmt.Printf("   Minecraft: %s:25565\n", publicIP)
		fmt.Printf("   SSH: ssh ec2-user@%s\n", publicIP)
	}

	if instanceState == string(ec2types.InstanceStateNameRunning) {
		fmt.Println("   Status: ✅ Server should be accessible")
	} else {
		fmt.Println("   Status: ❌ Server is not running")
	}
}
