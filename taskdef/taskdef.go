package taskdef

// might just make this package "tasks" and include running tasks in here, etc.

import (
	"github.com/aws/aws-sdk-go/aws"
	// "github.com/aws/aws-sdk-go/aws/awserr"
	// "github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecs"
	"text/template"
	// "fmt"
	"bytes"
	"encoding/json"
	// "log"
)

const fargate = "FARGATE"
const ec2 = "EC2"

// register-task-def stuff

// super lazy here on what will get returned for now. Should possibly return a proper object.
// the aws packages have structs for task defs, etc.
func ParseContainerDefTemplate(config map[string]interface{}) ([]byte, error) {
	templateFile := config["template"].(string)
	templateVars := config["templatevars"]
	t := template.Must(template.ParseFiles(templateFile))

	var tpl bytes.Buffer
	if err := t.Execute(&tpl, templateVars); err != nil {
  	return tpl.Bytes(), err
	}
	result := tpl.Bytes()
	return result, nil
}
// end resgieter-task-def stuff

func MakeContainerDefinitions(containerDefs []byte) ([]*ecs.ContainerDefinition, error) {
	// TODO: is this useful or should this just be what ParseContainerDefTemplate() does?
	var cdefs []*ecs.ContainerDefinition
	err := json.Unmarshal(containerDefs, &cdefs)
	return cdefs, err
}

/*
 * Takes the dict for config for a taskdefinition + a slice of *ecs.ContainerDefinition to build
 * the ecs.TaskDefinitionInput.
 *
 * Possibly should be renamed - would like one function which does all of this for ease of use
 * TODO: make config more concrete?
 */
func NewTaskDefinitionInput(config map[string]interface{}, containerDefs []*ecs.ContainerDefinition) (*ecs.RegisterTaskDefinitionInput, error){
	family := config["family"].(string)
	input := ecs.RegisterTaskDefinitionInput{ContainerDefinitions: containerDefs, Family: &family}

	if compats, ok := config["requiresCompatibilities"]; ok {
		cl := compats.([]interface{})
		requiredCompats := make([]*string, len(cl))
		for i := range cl {
			v := compats.([]interface{})[i].(string)
		  requiredCompats[i] = &v
		}
		input.RequiresCompatibilities = requiredCompats
	}

	if val, ok := config["cpu"]; ok {
		input.Cpu = aws.String(val.(string))
	}

	if val, ok := config["memory"]; ok {
		input.Memory = aws.String(val.(string))
	}

	if len(input.RequiresCompatibilities) == 1 && fargate == *input.RequiresCompatibilities[0] {
		input.NetworkMode = aws.String("awsvpc")
	} else {
		if val, ok := config["networkMode"]; ok {
			input.NetworkMode = aws.String(val.(string))
		}
	}

	if val, ok := config["executionRoleArn"]; ok {
		// required for Fargate
		input.ExecutionRoleArn = aws.String(val.(string))
	}


	// fmt.Printf("\n\nINPUT: %v\n\n", input)
	return &input, input.Validate()
}

func RegisterTaskDefinition(input *ecs.RegisterTaskDefinitionInput, client *ecs.ECS) (*ecs.RegisterTaskDefinitionOutput, error) {
	// TODO: do I even need this function? it's not actually doing anything.
	// Perhaps it should implement the full workflow which currently is in the anonymous func in ecscmd.go

	return client.RegisterTaskDefinition(input)
	// fmt.Printf("%T", result)
	// fmt.Printf("%v\n", result)
	// if err != nil {
	// 	fmt.Printf("%s", err)
	// }
}