package taskdef

// might just make this package "tasks" and include running tasks in here, etc.

import (
	"github.com/aws/aws-sdk-go/service/ecs"
	"text/template"
	netconfig "github.com/jmichalicek/ecscmd/network_configuration"
	// "fmt"
	"bytes"
	"encoding/json"
	// "log"
)

const FARGATE = "FARGATE"
const EC2 = "EC2"

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
func NewTaskDefinitionInput(config map[string]interface{}, containerDefs []*ecs.ContainerDefinition) (*ecs.RegisterTaskDefinitionInput, error) {
	family := config["family"].(string)
	input := &ecs.RegisterTaskDefinitionInput{ContainerDefinitions: containerDefs, Family: &family}

	if compats, ok := config["requiresCompatibilities"]; ok {
		cl := compats.([]interface{})
		requiredCompats := make([]*string, len(cl))
		for i := range cl {
			v := compats.([]interface{})[i].(string)
			requiredCompats[i] = &v
		}
		// input.RequiresCompatibilities = requiredCompats
		input = input.SetRequiresCompatibilities(requiredCompats)
	}

	if val, ok := config["cpu"]; ok {
		input = input.SetCpu(val.(string))
	}

	if val, ok := config["memory"]; ok {
		input = input.SetMemory(val.(string))
	}

	if len(input.RequiresCompatibilities) == 1 && FARGATE == *input.RequiresCompatibilities[0] {
		// Fargate requires awsvpc network mode
		input = input.SetNetworkMode("awsvpc")
	} else {
		if val, ok := config["networkMode"]; ok {
			input = input.SetNetworkMode(val.(string))
		}
	}

	if val, ok := config["executionRoleArn"]; ok {
		// required for Fargate
		input = input.SetExecutionRoleArn(val.(string))
	}

	if val, ok := config["taskRoleArn"]; ok {
		// The taskRole so that you do not have to pass aws creds around
		input = input.SetTaskRoleArn(val.(string))
	}

	// fmt.Printf("\n\nINPUT: %v\n\n", input)
	return input, input.Validate()
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


// NewRunTaskInput creates an ecs.RunTaskInput and returns it.
func NewRunTaskInput(config map[string]interface{}) (ecs.RunTaskInput, error) {
	// TODO: Take a typed config struct rather than this generic options or along with it
	// and put together the RunTaskInput... but at that point I have basically mirrored
	// ecs.RunTaskInput and could just use that unless my own struct could abstract it
	// in some useful manner for register, deregister, run, etc.

	input := ecs.RunTaskInput{}

	if cluster, ok := config["cluster"]; ok {
		input.SetCluster(cluster.(string))
	}

	if count, ok := config["count"]; ok {
		input.SetCount(count.(int64))
	}
	// Not sure I care for this. the config read in will have family, which is what we want to run
	// but if user specifies more specifically on the command line such as to run a specific revision,
	// then that's no good - that is where
	// a separate args struct becomes good
	// taskDefinition := config["taskDefinition"].(string)
	if taskDefinition, ok := config["family"]; ok {
		input.SetTaskDefinition(taskDefinition.(string))
	}

	if launchType, ok := config["launchType"]; ok {
		input.SetLaunchType(launchType.(string))
	}

	networkConfig, err := netconfig.NewNetworkConfiguration(config)

	input.SetNetworkConfiguration(&networkConfig)
	if err != nil {
		return input, err
	}

	return input, input.Validate()
}
