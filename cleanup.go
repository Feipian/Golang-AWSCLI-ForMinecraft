package main

import (
	"context"
	"fmt"
	"log"
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

	fmt.Println("🧹 Starting cleanup of Minecraft AWS resources...")

	// 1. Find and terminate EC2 instance (release any EIP first)
	instName := "minecraft-ec2"
	instDesc, err := ec2Client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
		Filters: []ec2types.Filter{
			{Name: aws.String("tag:Name"), Values: []string{instName}},
		},
	})
	if err != nil {
		log.Printf("failed to describe instances: %v", err)
	} else {
		for _, r := range instDesc.Reservations {
			for _, inst := range r.Instances {
				state := inst.State
				if state != nil && state.Name != ec2types.InstanceStateNameTerminated {
					instanceID := aws.ToString(inst.InstanceId)
					// Disassociate and release any Elastic IP attached to this instance
					if addrDesc, aerr := ec2Client.DescribeAddresses(ctx, &ec2.DescribeAddressesInput{Filters: []ec2types.Filter{{Name: aws.String("instance-id"), Values: []string{instanceID}}}}); aerr == nil {
						for _, a := range addrDesc.Addresses {
							if a.AssociationId != nil {
								_, _ = ec2Client.DisassociateAddress(ctx, &ec2.DisassociateAddressInput{AssociationId: a.AssociationId})
							}
							if a.AllocationId != nil {
								_, _ = ec2Client.ReleaseAddress(ctx, &ec2.ReleaseAddressInput{AllocationId: a.AllocationId})
							}
						}
					}

					fmt.Printf("Terminating instance: %s\n", instanceID)
					_, err = ec2Client.TerminateInstances(ctx, &ec2.TerminateInstancesInput{InstanceIds: []string{instanceID}})
					if err != nil {
						log.Printf("failed to terminate instance %s: %v", instanceID, err)
					} else {
						fmt.Printf("✅ Instance %s termination initiated\n", instanceID)
						waiter := ec2.NewInstanceTerminatedWaiter(ec2Client)
						_ = waiter.Wait(ctx, &ec2.DescribeInstancesInput{InstanceIds: []string{instanceID}}, 5*time.Minute)
					}
				}
			}
		}
	}

	// 2. Route tables (disassociate and delete default routes), then delete
	rtDesc, err := ec2Client.DescribeRouteTables(ctx, &ec2.DescribeRouteTablesInput{
		Filters: []ec2types.Filter{
			{Name: aws.String("tag:Name"), Values: []string{"minecraft-rt"}},
		},
	})
	if err != nil {
		log.Printf("failed to describe route tables: %v", err)
	} else {
		for _, rt := range rtDesc.RouteTables {
			rtID := aws.ToString(rt.RouteTableId)
			// Disassociate non-main associations
			for _, assoc := range rt.Associations {
				if assoc.RouteTableAssociationId != nil && !aws.ToBool(assoc.Main) {
					_, _ = ec2Client.DisassociateRouteTable(ctx, &ec2.DisassociateRouteTableInput{AssociationId: assoc.RouteTableAssociationId})
				}
			}
			// Remove default route if present
			for _, r := range rt.Routes {
				if aws.ToString(r.DestinationCidrBlock) == "0.0.0.0/0" {
					_, _ = ec2Client.DeleteRoute(ctx, &ec2.DeleteRouteInput{RouteTableId: rt.RouteTableId, DestinationCidrBlock: aws.String("0.0.0.0/0")})
				}
			}
			// Skip main route table deletion
			if len(rt.Associations) > 0 && aws.ToBool(rt.Associations[0].Main) {
				fmt.Printf("Skipping main route table: %s\n", rtID)
				continue
			}
			fmt.Printf("Deleting route table: %s\n", rtID)
			_, err = ec2Client.DeleteRouteTable(ctx, &ec2.DeleteRouteTableInput{RouteTableId: rt.RouteTableId})
			if err != nil {
				log.Printf("failed to delete route table %s: %v", rtID, err)
			} else {
				fmt.Printf("✅ Route table %s deleted\n", rtID)
			}
		}
	}

	// 3. Delete subnets
	subnetDesc, err := ec2Client.DescribeSubnets(ctx, &ec2.DescribeSubnetsInput{
		Filters: []ec2types.Filter{
			{Name: aws.String("tag:Name"), Values: []string{"minecraft-subnet"}},
		},
	})
	if err != nil {
		log.Printf("failed to describe subnets: %v", err)
	} else {
		for _, subnet := range subnetDesc.Subnets {
			subnetID := aws.ToString(subnet.SubnetId)
			fmt.Printf("Deleting subnet: %s\n", subnetID)
			_, err = ec2Client.DeleteSubnet(ctx, &ec2.DeleteSubnetInput{SubnetId: aws.String(subnetID)})
			if err != nil {
				log.Printf("failed to delete subnet %s: %v", subnetID, err)
			} else {
				fmt.Printf("✅ Subnet %s deleted\n", subnetID)
			}
		}
	}

	// 4. Delete security group(s)
	sgName := "minecraft-sg"
	sgDesc, err := ec2Client.DescribeSecurityGroups(ctx, &ec2.DescribeSecurityGroupsInput{
		Filters: []ec2types.Filter{{Name: aws.String("group-name"), Values: []string{sgName}}},
	})
	if err != nil {
		log.Printf("failed to describe security groups: %v", err)
	} else {
		for _, sg := range sgDesc.SecurityGroups {
			sgID := aws.ToString(sg.GroupId)
			fmt.Printf("Deleting security group: %s\n", sgID)
			_, err = ec2Client.DeleteSecurityGroup(ctx, &ec2.DeleteSecurityGroupInput{GroupId: aws.String(sgID)})
			if err != nil {
				log.Printf("failed to delete security group %s: %v", sgID, err)
			} else {
				fmt.Printf("✅ Security group %s deleted\n", sgID)
			}
		}
	}

	// 5. Detach and delete Internet Gateway(s)
	igwDesc, err := ec2Client.DescribeInternetGateways(ctx, &ec2.DescribeInternetGatewaysInput{
		Filters: []ec2types.Filter{{Name: aws.String("tag:Name"), Values: []string{"minecraft-igw"}}},
	})
	if err != nil {
		log.Printf("failed to describe internet gateways: %v", err)
	} else {
		for _, igw := range igwDesc.InternetGateways {
			igwID := aws.ToString(igw.InternetGatewayId)
			for _, attachment := range igw.Attachments {
				if attachment.VpcId != nil {
					vpcID := aws.ToString(attachment.VpcId)
					fmt.Printf("Detaching IGW %s from VPC %s\n", igwID, vpcID)
					_, _ = ec2Client.DetachInternetGateway(ctx, &ec2.DetachInternetGatewayInput{InternetGatewayId: igw.InternetGatewayId, VpcId: aws.String(vpcID)})
				}
			}
			fmt.Printf("Deleting internet gateway: %s\n", igwID)
			_, err = ec2Client.DeleteInternetGateway(ctx, &ec2.DeleteInternetGatewayInput{InternetGatewayId: igw.InternetGatewayId})
			if err != nil {
				log.Printf("failed to delete internet gateway %s: %v", igwID, err)
			} else {
				fmt.Printf("✅ Internet gateway %s deleted\n", igwID)
			}
		}
	}

	// 6. Delete VPC(s)
	vpcDesc, err := ec2Client.DescribeVpcs(ctx, &ec2.DescribeVpcsInput{
		Filters: []ec2types.Filter{{Name: aws.String("tag:Name"), Values: []string{"minecraft-vpc"}}},
	})
	if err != nil {
		log.Printf("failed to describe VPCs: %v", err)
	} else {
		for _, vpc := range vpcDesc.Vpcs {
			vpcID := aws.ToString(vpc.VpcId)
			fmt.Printf("Deleting VPC: %s\n", vpcID)
			_, err = ec2Client.DeleteVpc(ctx, &ec2.DeleteVpcInput{VpcId: aws.String(vpcID)})
			if err != nil {
				log.Printf("failed to delete VPC %s: %v", vpcID, err)
			} else {
				fmt.Printf("✅ VPC %s deleted\n", vpcID)
			}
		}
	}

	fmt.Println("🎉 Cleanup completed! All Minecraft resources should be deleted.")
	fmt.Println("💡 Note: EC2 instances may take a few minutes to fully terminate.")
}
