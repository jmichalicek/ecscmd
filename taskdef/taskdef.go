package taskdef

// might just make this package "tasks" and include running tasks in here, etc.

import (
	"github.com/aws/aws-sdk-go/service/ecs"
	"text/template"
	netconfig "github.com/jmichalicek/ecscmd/network_configuration"
	// "fmt"
	"bytes"
	"encoding/json"
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

// TODO the volumes stuff here is a hack to get stuff done at the moment. Not sure another template is really a good way to go.
// probably want volumes config in the toml
func ParseVolumeDefTemplate(config map[string]interface{}) ([]byte, error) {
	templateFile := config["volumetemplate"].(string)
	templateVars := config["templatevars"]
	t := template.Must(template.ParseFiles(templateFile))

	var tpl bytes.Buffer
	if err := t.Execute(&tpl, templateVars); err != nil {
		return tpl.Bytes(), err
	}
	result := tpl.Bytes()
	return result, nil
}

func MakeVolumesDefinitions(volumeDefs []byte) ([]*ecs.Volume, error) {
	// TODO: is this useful or should this just be what ParseContainerDefTemplate() does?
	var defs []*ecs.Volume
	err := json.Unmarshal(volumeDefs, &defs)
	return defs, err
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

	if vols, ok := config["volumes"]; ok {
		volumeConfigs := vols.([]interface{})
		volumes := make([]*ecs.Volume, len(volumeConfigs))
		for i := range volumeConfigs {
			volume, err := makeEcsVolume(volumeConfigs[i].(map[string]interface{}))
			if err != nil {
				return input, err // seems like this could be confusing
			}
			volumes[i] = volume
		}
		// input.RequiresCompatibilities = requiredCompats
		input = input.SetVolumes(volumes)
	}

	// fmt.Printf("\n\nINPUT: %v\n\n", input)
	return input, input.Validate()
}

// TODO: Or I could just use ecs.Volume and ecs.DockerVolumeConfiguration...
// That gets annoying with the []*string though for Labels and DriverOpts
// the json marshall/unmarshall is less code to look at
type dockerVolumeConfig struct {
	Labels []string
	Scope string
	Driver string
}

type volume struct {
	Name string
	DockerVolumeConfiguration dockerVolumeConfig
}
/* newEcsVolume retuns a *ecs.Volume from ecscmd volume configuration data */
func makeEcsVolume(volumeConfig map[string]interface{}) (*ecs.Volume, error) {
	// TODO: really need to make a proper struct for all this I think and unmarshal from koanf to that
	// rather than all this mucking around with map[string]interface{}
	// Can I just do this?

	// TODO: this is not quite right, need to shuffle some of this about to
	// {
  //     "name": "a_name",
  //     "host": null,
  //     "dockerVolumeConfiguration": {
  //         "autoprovision": null,
  //         "labels": null,
  //         "scope": "task",
  //         "driver": "local",
  //         "driverOpts": null
  //     }
  // }

	// todo: struct for this?
	// docker_volume_config := map[string]interface{}{"labels": volumeConfig["labels"], "scope": volumeConfig["scope"], "driver": volumeConfig["driver"]}
	dvc := dockerVolumeConfig{Labels: volumeConfig["labels"].([]string), Scope: volumeConfig["scope"].(string), Driver: volumeConfig["driver"].(string)}
	vol := volume{Name: volumeConfig["name"].(string), DockerVolumeConfiguration: dvc}
	configJson, err := json.Marshal(vol)
	if err != nil {
		return nil, err
	}
	var volume *ecs.Volume
	err = json.Unmarshal(configJson, volume)
	return volume, err


	// v := ecs.Volume{Name: volumeConfig["name"], Driver: volumeConfig["driver"], Scope: volumeConfig["scope"]}
	// if l, ok := volumeConfig["labels"]; ok {
	// 	ll := l.([]string)
	// 	labels := make([]*string, len(ll))
	// 	for i := range ll {
	// 		label := ll[i].(string)
	// 		labels[i] = &label
	// 	}
	// }
	// // TODO: DriverOpts... really starting to wonder if this should just be handled via json, even if it's just
	// // json in the toml as a string...
	// return v
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
