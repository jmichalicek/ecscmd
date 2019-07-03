package taskdef

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecs"
	"text/template"
	"fmt"
	"bytes"
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

type service struct {
	taskDefinition string // really just the arn, etc.
	clusterId string
}

// register-task-def stuff

// super lazy here on what will get returned for now. Should possibly return a proper object.
// the aws packages have structs for task defs, etc.
func ParseTemplate(config map[string]interface{}) (string, error) {
	templateFile := config["template"].(string)
	templateVars := config["templatevars"]
	fmt.Printf("%v", templateVars)
	t := template.Must(template.ParseFiles(templateFile))

	var tpl bytes.Buffer
	if err := t.Execute(&tpl, templateVars); err != nil {
  	return "", err
	}
	result := tpl.String()
	return result, nil
}
// end resgieter-task-def stuff

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
