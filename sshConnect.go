package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

// sshConnect prints the SSH command for the minecraft instance and optionally runs it.
func main() {
	keyPath := flag.String("key", "", "Path to your .pem private key (required to run automatically)")
	name := flag.String("name", "minecraft-ec2", "EC2 instance Name tag to connect to")
	auto := flag.Bool("run", false, "If set, execute the SSH command")
	user := flag.String("user", "ec2-user", "SSH username (Amazon Linux = ec2-user)")
	flag.Parse()

	ctx := context.TODO()
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		log.Fatalf("failed to load AWS config: %v", err)
	}

	ec2Client := ec2.NewFromConfig(cfg)

	// Locate instance by Name tag
	instDesc, err := ec2Client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
		Filters: []ec2types.Filter{{Name: aws.String("tag:Name"), Values: []string{*name}}},
	})
	if err != nil {
		log.Fatalf("describe instances failed: %v", err)
	}

	var publicIP string
	var instanceID string
	for _, r := range instDesc.Reservations {
		for _, inst := range r.Instances {
			if inst.State != nil && inst.State.Name == ec2types.InstanceStateNameRunning {
				instanceID = aws.ToString(inst.InstanceId)
				if inst.PublicIpAddress != nil {
					publicIP = aws.ToString(inst.PublicIpAddress)
				}
				break
			}
		}
	}
	if publicIP == "" {
		log.Fatalf("could not find running instance %q with a public IP", *name)
	}

	fmt.Printf("Instance ID: %s\n", instanceID)
	cmd := fmt.Sprintf("ssh -o StrictHostKeyChecking=accept-new -i %q %s@%s", *keyPath, *user, publicIP)
	fmt.Println("SSH command:")
	fmt.Println(cmd)

	if *auto {
		if *keyPath == "" {
			log.Fatalf("--key is required when using --run")
		}
		// Execute user's local SSH client (auto-accept host key on first connect)
		c := exec.Command("ssh", "-o", "StrictHostKeyChecking=accept-new", "-i", *keyPath, fmt.Sprintf("%s@%s", *user, publicIP))
		// Attach to current console for interactive session
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		c.Stdin = os.Stdin
		if err := c.Run(); err != nil {
			log.Fatalf("ssh exited with error: %v", err)
		}
	}
}
