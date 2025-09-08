package main

import (
	"context"
	"fmt"
	"log"

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

	// // 2. Create Internet Gateway
	// igwOut, err := ec2Client.CreateInternetGateway(ctx, &ec2.CreateInternetGatewayInput{})
	// if err != nil {
	// 	log.Fatalf("failed to create IGW: %v", err)
	// }
	// igwID := *igwOut.InternetGateway.InternetGatewayId
	// fmt.Println("Created Internet Gateway:", igwID)

	// // Attach IGW to VPC
	// _, err = ec2Client.AttachInternetGateway(ctx, &ec2.AttachInternetGatewayInput{
	// 	InternetGatewayId: aws.String(igwID),
	// 	VpcId:             aws.String(vpcID),
	// })
	// if err != nil {
	// 	log.Fatalf("failed to attach IGW: %v", err)
	// }

	// // 3. Create Subnet (public)
	// subnetOut, err := ec2Client.CreateSubnet(ctx, &ec2.CreateSubnetInput{
	// 	VpcId:            aws.String(vpcID),
	// 	CidrBlock:        aws.String("10.0.1.0/24"), // Subnet range
	// 	AvailabilityZone: aws.String("us-east-1a"), // Change region/zone
	// 	TagSpecifications: []ec2types.TagSpecification{
	// 		{
	// 			ResourceType: ec2types.ResourceTypeSubnet,
	// 			Tags: []ec2types.Tag{
	// 				{Key: aws.String("Name"), Value: aws.String("minecraft-subnet")},
	// 			},
	// 		},
	// 	},
	// })
	// if err != nil {
	// 	log.Fatalf("failed to create subnet: %v", err)
	// }
	// subnetID := *subnetOut.Subnet.SubnetId
	// fmt.Println("Created Subnet:", subnetID)

	// // 4. Create Route Table
	// rtOut, err := ec2Client.CreateRouteTable(ctx, &ec2.CreateRouteTableInput{
	// 	VpcId: aws.String(vpcID),
	// })
	// if err != nil {
	// 	log.Fatalf("failed to create route table: %v", err)
	// }
	// rtID := *rtOut.RouteTable.RouteTableId
	// fmt.Println("Created Route Table:", rtID)

	// // Add route to Internet
	// _, err = ec2Client.CreateRoute(ctx, &ec2.CreateRouteInput{
	// 	RouteTableId:         aws.String(rtID),
	// 	DestinationCidrBlock: aws.String("0.0.0.0/0"),
	// 	GatewayId:            aws.String(igwID),
	// })
	// if err != nil {
	// 	log.Fatalf("failed to create route: %v", err)
	// }

	// // Associate route table with subnet
	// _, err = ec2Client.AssociateRouteTable(ctx, &ec2.AssociateRouteTableInput{
	// 	RouteTableId: aws.String(rtID),
	// 	SubnetId:     aws.String(subnetID),
	// })
	// if err != nil {
	// 	log.Fatalf("failed to associate route table: %v", err)
	// }

	// fmt.Println("✅ VPC setup complete. Use VPC:", vpcID, "Subnet:", subnetID)
}
