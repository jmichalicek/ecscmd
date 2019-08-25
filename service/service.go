package service

// might just make this package "tasks" and include running tasks in here, etc.

// TODO: function go get existing service config.  Will be used to create config map with defaults from
// the existing service values.

// TODO: auto create log groups for service? Seems out of scope for this, but useful for simple "just works" functionality

import (
	"github.com/aws/aws-sdk-go/aws"
	// "github.com/aws/aws-sdk-go/aws/awserr"
	// "github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecs"
	// "fmt"
	// "bytes"
	// "encoding/json"
	// "log"
)

/*
 * Create an ecs.UpdateServiceInput for a Fargate ECS service
 */
func NewFargateUpdateServiceInput(config map[string]interface{}) (*ecs.UpdateServiceInput, error){
	// TODO: may make a struct for this to take... then again, that basically becomes UpdateServiceInput...
	// but maybe can simplify it slightly or something.

	serviceName := aws.String(config["name"].(string))
	// https://docs.aws.amazon.com/sdk-for-go/api/service/ecs/#UpdateServiceInput
	input := &ecs.UpdateServiceInput{Service: serviceName}

	if taskDef, ok := config["taskDefinition"]; ok {
		input = input.SetTaskDefinition(taskDef.(string)) // seems to be not required - perhaps can update things without changing the task def.
	}

	if cluster, ok := config["cluster"]; ok {
		input = input.SetCluster(cluster.(string))
	}

	// TODO: Unsure if this should come in on config or be passed in as a standalone arg. It feels like a standlone arg
	// and command line only flag rather than config file thing.
	if forceDeployment, ok := config["forceDeployment"]; ok {
		if forceDeployment.(bool) {
			input = input.SetForceNewDeployment(true)
		}
	}

	awsVpcConfig := &ecs.AwsVpcConfiguration{}
	if assignPublicIp, ok := config["assignPublicIp"]; ok {
		awsVpcConfig = awsVpcConfig.SetAssignPublicIp(assignPublicIp.(string))
	}

	if securityGroupIds, ok := config["securityGroups"]; ok {
		g := securityGroupIds.([]interface{})
		securityGroups := make([]*string, len(g))
		for i := range g {
			groupId := securityGroupIds.([]interface{})[i].(string)
		  securityGroups[i] = &groupId
		}
		awsVpcConfig = awsVpcConfig.SetSecurityGroups(securityGroups)
	}

	if subnetIds, ok := config["subnets"]; ok {
		s := subnetIds.([]interface{})
		subnets := make([]*string, len(s))
		for i := range s {
			subnetId := subnetIds.([]interface{})[i].(string)
		  subnets[i] = &subnetId
		}
		awsVpcConfig = awsVpcConfig.SetSubnets(subnets)
	}

	networkConfig := ecs.NetworkConfiguration{AwsvpcConfiguration: awsVpcConfig}
	input.SetNetworkConfiguration(&networkConfig)

	// TODO: other otpions
	// fmt.Printf("\n\nINPUT: %v\n\n", input)
	return input, input.Validate()
}
