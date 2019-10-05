package network_configuration

import (
	"github.com/aws/aws-sdk-go/service/ecs"
)

func NewNetworkConfiguration(config map[string]interface{}) (ecs.NetworkConfiguration, error) {
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
	return networkConfig, networkConfig.Validate()
}
