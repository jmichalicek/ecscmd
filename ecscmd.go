package "ecscmd"

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecs"

	"fmt"
	// "log"
)

// TODO: not so sure I will need these. I can probably just
// use the structs from the aws sdk, but may need them for reading in
// the toml config
type containerDefinition struct {
	image string
}

type taskDefinition struct {
	name string  // name from toml config
	family string // may not need this as aws sdk will have a struct with it.
	containerDefinitions []containerDefinition
	// or containerDefTemplate string ??
}

type service {
	taskDefinition string // really just the arn, etc.
	clusterId string
}

func fake_func() {
	// TODO: region as var and from settings
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String("us-east-1")},
	)

	// Create EC2 service client
	svc := ecs.New(sess)

	input := &ecs.DescribeTaskDefinitionInput{
		TaskDefinition: aws.String("hello_world:8"),
	}

	result, err := svc.DescribeTaskDefinition(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case ecs.ErrCodeServerException:
				fmt.Println(ecs.ErrCodeServerException, aerr.Error())
			case ecs.ErrCodeClientException:
				fmt.Println(ecs.ErrCodeClientException, aerr.Error())
			case ecs.ErrCodeInvalidParameterException:
				fmt.Println(ecs.ErrCodeInvalidParameterException, aerr.Error())
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			fmt.Println(err.Error())
		}
		return
	}

	fmt.Println(result)
}
