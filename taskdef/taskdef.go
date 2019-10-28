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
		// not []map[string]string because the labels key is a list and driveropts is another map
		// TODO: if I have koanf just deserialize to a struct, can I have it just deserialize to
		// the aws-sdk-go types or do I need my own struct in the middle?
		volumeConfigs := vols.([]interface{})
		volumes := make([]*ecs.Volume, len(volumeConfigs))
		for i, conf := range volumeConfigs {
			volume := makeEcsVolume(conf.(map[string]interface{}))
			volumes[i] = &volume
		}
		// input.RequiresCompatibilities = requiredCompats
		input = input.SetVolumes(volumes)
	}

	// fmt.Printf("\n\nINPUT: %v\n\n", input)
	return input, input.Validate()
}

/*
 * newEcsVolume retuns a *ecs.Volume from the slightly flatter ecscmd volume configuration data
 * see https://docs.aws.amazon.com/sdk-for-go/api/service/ecs/#DockerVolumeConfiguration
 * and https://docs.aws.amazon.com/sdk-for-go/api/service/ecs/#Volume
 */
func makeEcsVolume(volumeConfig map[string]interface{}) ecs.Volume {
	// TODO: Support HostVolumeProperties on the ecs.Volume
	// TODO: Support DriverOpts on the DockerVolumeConfiguration, autoprovision
	// TODO: really need to make a proper struct for all this I think and unmarshal from koanf to that

	scope, _ := volumeConfig["scope"].(string)
	driver, _ := volumeConfig["driver"].(string)
	name := volumeConfig["name"].(string)
	dvc := ecs.DockerVolumeConfiguration{Scope: &scope, Driver: &driver}
	v := ecs.Volume{Name: &name, DockerVolumeConfiguration: &dvc}
	// cannot assert to map[string]string ? unsure why not.
	// l, ok := volumeConfig["labels"].(map[string]string)
	l, _ := volumeConfig["labels"].(map[string]interface{}) // ensuring we have a list to iterate over here
	if len(l) > 0 {
		// the if is mostly to avoid the call to SetLabels
		labels := make(map[string]*string, len(l))
		for k, v := range l {
				label := v.(string)
				// label := v // if I could just assert to map[string]string
				labels[k] = &label
		}
		v.DockerVolumeConfiguration.SetLabels(labels)
	}
	return v
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
