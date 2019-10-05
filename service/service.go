package service

// might just make this package "tasks" and include running tasks in here, etc.

// TODO: function go get existing service config.  Will be used to create config map with defaults from
// the existing service values.

// TODO: auto create log groups for service? Seems out of scope for this, but useful for simple "just works" functionality

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ecs"
	netconfig "github.com/jmichalicek/ecscmd/network_configuration"
)

/*
 * Create an ecs.UpdateServiceInput for a Fargate ECS service
 */
func NewUpdateServiceInput(config map[string]interface{}) (*ecs.UpdateServiceInput, error){
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

	if desiredCount, ok := config["desiredCount"]; ok {
		input = input.SetDesiredCount(desiredCount.(int64))
	}

	// TODO: Unsure if this should come in on config or be passed in as a standalone arg. It feels like a standlone arg
	// and command line only flag rather than config file thing.
	if forceDeployment, ok := config["forceDeployment"]; ok {
		if forceDeployment.(bool) {
			input = input.SetForceNewDeployment(true)
		}
	}

	networkConfig, err := netconfig.NewNetworkConfiguration(config)
	input.SetNetworkConfiguration(&networkConfig)
	if err != nil {
		return input, err
	}

	// TODO: other otpions
	// fmt.Printf("\n\nINPUT: %v\n\n", input)
	return input, input.Validate()
}

/*
 * Create and return a pointer to an ecs.CreateServiceInput
 * https://docs.aws.amazon.com/sdk-for-go/api/service/ecs/#CreateServiceInput
 */
func NewCreateServiceInput(config map[string]interface{}) (*ecs.CreateServiceInput, error){
	// TODO: may make a struct for this to take... then again, that basically becomes UpdateServiceInput...
	// but maybe can simplify it slightly or something.
	// TODO: Should this be fargate specific or should this just be NewCreateServiceInput and the fargate or not
	// is more dynamic?

	// TODO: support ClientToken!

	serviceName := aws.String(config["name"].(string))
	// https://docs.aws.amazon.com/sdk-for-go/api/service/ecs/#UpdateServiceInput
	input := &ecs.CreateServiceInput{ServiceName: serviceName}

	if taskDef, ok := config["taskDefinition"]; ok {
		input = input.SetTaskDefinition(taskDef.(string)) // seems to be not required - perhaps can update things without changing the task def.
	}

	if cluster, ok := config["cluster"]; ok {
		input = input.SetCluster(cluster.(string))
	}

	if desiredCount, ok := config["desiredCount"]; ok {
		input = input.SetDesiredCount(desiredCount.(int64))
	}

	if launchType, ok := config["launchType"]; ok {
		input = input.SetLaunchType(launchType.(string))
	}

	deploymentController := &ecs.DeploymentController{}
	if dc, ok := config["deploymentController"]; ok {
		deploymentController = deploymentController.SetType(dc.(string))
	} else {
		deploymentController = deploymentController.SetType("ECS")
	}
	input = input.SetDeploymentController(deploymentController)

	awsVpcConfig := &ecs.AwsVpcConfiguration{}
	// Supported for fargate, not for ec2 launch type
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

	// TODO: assumptions made here about vpc config assuming that this is for a Fargate service
	// TODO: support non-fargate cleanly?
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
