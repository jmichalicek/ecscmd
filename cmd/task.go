/*
Copyright © 2019 Justin Michalicek <jmichalicek@gmail.com>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

// cloudwatchlogs import should maybe be somewhere else, but dropping it here for now for initial
// implementation in the first place I am using it
import (
	"fmt"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/jmichalicek/ecscmd/session"
	"github.com/jmichalicek/ecscmd/taskdef"
	"github.com/spf13/cobra"
	"strings"
	"time"
)

// Limited subset of functionality for now
// TODO: Need to see if this will work well to set with koanf and then override from command line
// or do I need two separate structs - one for koanf to set and a separate one for cobra to save command
// line options to.
type runTaskCommandOptions struct {
	Cluster         string `koanf:"cluster"`
	Count           int64  `koanf:"count"`
	Group           string `koanf:"group"`
	TaskDefinition  string `koanf:"family"`
	WaitForComplete bool
	StreamOutput    bool
	Fargate         bool
	// EnableECSManagedTags bool
	// The name of the task group to associate with the task. The default value
	// is the family name of the task definition (for example, family:my-family-name).
}

// TODO: this is gross and clunky. I may have to look into something other than cobra... or maybe
// I am doing something wrong. Some of their examples were like this.
var runTaskOptions runTaskCommandOptions = runTaskCommandOptions{Fargate: true}

type registerTaskDefCommandOptions struct {
	TemplateVars            []string // can be used to set template vars which are not in the koanf settings or to override them
	Family                  string
	RequiresCompatibilities []string
	cpu                     int64
	memory                  int64
	template                string
}

var registerTaskOptions registerTaskDefCommandOptions = registerTaskDefCommandOptions{}

// serviceCmd represents the service command
var taskCmd = &cobra.Command{
	Use:   "task",
	Short: "commands for managing tasks and task definitions",
	// Run: func(cmd *cobra.Command, args []string) {
	// 	fmt.Println("service called")
	// },
}

// TODO: may go back to full taskdef as json template. awscli allows for using a json file so
// hopefully I can just unmarshal the whole thing to RegistTaskDefinitionInput, maybe.
// which could simplify the config structure to being almost all template vars + a few aws session details like region, profile, etc
// need to see how this works as is with creating a new taskdef and as far as updating existing taskdef - want to be able
// to easily just update the bare minimum which most of the time will just be container defs to update an image
var cmdRegisterTaskDef = &cobra.Command{
	Use:   "register taskDefName",
	Short: "Register an ECS task definition",
	Long: `Register a new task definition or update an existing task definition.
    A taskDefinition section should exist in the config file`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// TODO: too much going on here... or will be. This needs to be its own function defined elsewhere
		var taskDefName = args[0]
		var configKey = fmt.Sprintf("taskdef.%s", taskDefName)
		if !k.Exists(configKey) {
			return fmt.Errorf("No task definition configuration named %s", taskDefName)
		}
		taskDefConfig := k.Cut(configKey).Raw()
		// Get a correctly typed template variables so that it can be accessed via index and updated
		var tvars map[string]interface{} = taskDefConfig["templatevars"].(map[string]interface{})
		for _, tvar := range registerTaskOptions.TemplateVars {
			// very naive, but should work for 99% of cases... a key with an = in it would be weird
			s := strings.SplitN(tvar, "=", 2)
			tvars[s[0]] = s[1]
		}
		// TODO: not certain this is the way to go given that aws-sdk-go doesn't use the json for this
		// but it's an easy-ish way to make it clear, modifiable, work with all kinds of vars
		containerDefBytes, err := taskdef.ParseContainerDefTemplate(taskDefConfig)
		cdef, err := taskdef.MakeContainerDefinitions(containerDefBytes)

		// ideally could just pass taskDefConfig and get this back with something else wrapping the above stuff
		// and this.
		i, err := taskdef.NewTaskDefinitionInput(taskDefConfig, cdef)
		if err != nil {
			return err
		}

		session, err := session.NewAwsSession(taskDefConfig)
		if err != nil {
			return err
		}
		// TODO: look at source for how this is implemented to handle both this OR with extra config
		// both on ecs.New()
		client := ecs.New(session)

		if baseConfig.dryRun {
			fmt.Printf("%v", i)
		} else {
			result, err := taskdef.RegisterTaskDefinition(i, client)
			if err != nil {
				return err
			}
			// TODO: Abstract output sooner for various verbosities and formats.
			fmt.Printf("Registered task defintion: %s\n", *result.TaskDefinition.TaskDefinitionArn)
		}
		return nil
	},
}

var cmdRunTask = &cobra.Command{
	// TODO: deal with using either the actual task def name/family/arn OR our local config name
	Use:   "run taskDefName",
	Short: "Run an ECS task. Currently assumes a Fargate launch type.",
	Long:  `Run an ECS task. Currently assumes a Fargate launch type.`,
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// TODO: Wow, this has gotten messy after starting to add logstream watching... needs cleaned up even more than I feared it would
		// TODO: really should make the name of the config optional - may want to run a task
		// which there is no local config for
		var taskDefName = args[0]
		var configKey = fmt.Sprintf("taskdef.%s", taskDefName)
		if !k.Exists(configKey) {
			return fmt.Errorf("No task definition configuration named %s", taskDefName)
		}
		config := k.Cut(configKey).Raw()
		session, err := session.NewAwsSession(config)
		if err != nil {
			return err
		}

		// TODO: clunky and gross
		if runTaskOptions.TaskDefinition != "" {
			config["family"] = runTaskOptions.TaskDefinition
		}

		config["launchType"] = taskdef.FARGATE
		// TODO: look at source for how this is implemented to handle both this OR with extra config
		// both on ecs.New()
		client := ecs.New(session)
		// START GET TASK DEF AND LOG CONFIGS
		// Make sure task def exists
		// TODO: Put this in a function somewhere or really I think I do need wrapper Task and TaskDef structs maybe
		taskDefFam := config["family"].(string)
		describeTaskDefInput := &ecs.DescribeTaskDefinitionInput{
			TaskDefinition: &taskDefFam,
		}
		descTaskDef, err := client.DescribeTaskDefinition(describeTaskDefInput)
		if err != nil {
			return err
		}
		// TODO: assuming a single container task. Handle multi-container later... fancier logging, specify the container, etc.
		logConfig := descTaskDef.TaskDefinition.ContainerDefinitions[0].LogConfiguration.Options
		containerName := descTaskDef.TaskDefinition.ContainerDefinitions[0].Name
		logGroup := logConfig["awslogs-group"]
		logStreamPrefix := logConfig["awslogs-stream-prefix"]
		logStreamName := *logStreamPrefix + "/" + *containerName + "/"
		// END GET TASK DEF AND LOG CONFIGS

		// TODO: look up task def, get log stream info
		runTaskInput, err := taskdef.NewRunTaskInput(config)
		if err != nil {
			return err
		}

		// handle the output.
		// TODO: Make a nice interface to use for these and a single "dostuff()"
		// function which takes that interface - then could call mything.DryRun()
		// or MyConfig.Execute() and those could return a standard interface and errors
		// and then the if err != nil stuff could live
		// in one place instead of every Run function
		// Unsure how to abstract stuff like waiting on a waiter, though.
		if baseConfig.dryRun {
			// TODO: better output here - really should try to look up the task def on aws
			fmt.Printf("Would run task %v\n", config)
			return nil
		}

		runTaskResponse, err := client.RunTask(&runTaskInput)
		// result, err := taskdef.RegisterTaskDefinition(i, client)
		if err != nil {
			return err
		}

		if runTaskOptions.WaitForComplete || runTaskOptions.StreamOutput {
			// TODO: could be interesting to do this with waitgroup rather than status channels
			// to know when to move on, stop the cloudwatch goroutine, etc.
			taskStatusChannel := make(chan RunTaskResult)
			// TODO: not actually making use of cloudWatchChannel right now. A better use would be
			// to return an object with the cloudwatch info from it, allowing streamCloudwatchLogs() to be more generic
			// and whatever uses it to print that data or otherwise as needed.
			cloudWatchChannel := make(chan string)
			// need a separate channel for cloudwatch goroutine to know when to quit
			// rather than having both the main loop and the cloudwatch goroutine listen on the same channel. In the latter
			// case, only one will actually get the message to stop and things go wonky
			cloudWatchDoneChan := make(chan bool)
			taskArn := runTaskResponse.Tasks[0].TaskArn
			taskStatus := TaskPending
			go WaitForTask(client, *runTaskInput.Cluster, *taskArn, taskStatusChannel)

			if runTaskOptions.StreamOutput {
				// TODO: abstract this into a function somewhere
				taskArn := runTaskResponse.Tasks[0].TaskArn
				parts := strings.Split(*taskArn, "/")
				taskId := parts[len(parts)-1]
				go streamCloudwatchLogs(cloudwatchlogs.New(session), *logGroup, logStreamName+taskId, cloudWatchDoneChan)
			}

			done := false
			// kludge because the goroutine sends the pending status
			// before this is listening for it
			fmt.Printf("Waiting for task")
			for {
				select {
				case s := <-cloudWatchChannel:
					fmt.Printf("%s", s)
				case s := <-taskStatusChannel:
					if s.Error != nil {
						// TODO: Do I really want to always exit now? For now, go with yes.
						return s.Error
					}
					taskStatus = s.Status
					switch s.Status {
					case TaskStopped:
						done = true
						cloudWatchDoneChan <- done
					case TaskRunning:
						// TODO: differentiate between "still running" and "just started"?
						fmt.Println("Task started")
					}
				default:
					// If waiting for task without streaming logs or if output will be streamed, but the task has not
					// started show "working" dots
					if !runTaskOptions.StreamOutput || taskStatus == TaskPending {
						fmt.Printf(".")
						time.Sleep(time.Second)
					}
				}
				if done {
					fmt.Println("")
					break
				}
			}
			close(taskStatusChannel)
			close(cloudWatchChannel)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(taskCmd)
	taskCmd.AddCommand(cmdRegisterTaskDef)
	taskCmd.AddCommand(cmdRunTask)

	// TODO: still gross. I want these processed AFTER reading options from config, anyway.
	cmdRegisterTaskDef.Flags().StringArrayVar(&registerTaskOptions.TemplateVars, "template-var", []string{}, "Specify a template variable name and value for use in the task definition template. --template-var\"name=value\". May be specified multiple times to set multiple variables.")

	cmdRunTask.Flags().StringVar(&runTaskOptions.TaskDefinition, "task-definition", "", "Task definition arn for the task to run. This could be full arn, family, or family:revision")
	cmdRunTask.Flags().StringVar(&runTaskOptions.Cluster, "cluster", "", "Cluster to run the task on. Defaults to AWS default cluster.")
	cmdRunTask.Flags().Int64Var(&runTaskOptions.Count, "count", 1, "How many of this task to run.")
	cmdRunTask.Flags().BoolVar(&runTaskOptions.WaitForComplete, "wait-for-stop", false, "Wait for the task to complete before continuing.")
	cmdRunTask.Flags().BoolVar(&runTaskOptions.StreamOutput, "stream-logs", false, "Stream cloudwatch logs for the task.")
}

// TODO: Move everything below here to the taskdef and a new cloudwatch package or to a central
// main ecscmd package.
type TaskStatus int

/* AWS has more statuses than this */
const (
	TaskPending TaskStatus = iota
	TaskRunning
	TaskStopped
)

/* include more data about the task? */
// Not sure Result is really correct/accurate here.
// This type is for communicating current status and any errors across the channel
type RunTaskResult struct {
	Error  error
	Status TaskStatus
}

// WaitForTask uses the passed in *ecs.ECS client to wait for the task specified by taskArn on cluster to stop.
// The current task status at each change and any error are returned in the taskStatusChannel.
//
// I suspect this function currently has a race condition where if the task is already
// running (as opposed to pending or provisioning) or possibly only already stopped and may get hung up
// or may return an error..
func WaitForTask(client *ecs.ECS, cluster string, taskArn string, taskStatusChannel chan<- RunTaskResult) {
	describeTaskInput := &ecs.DescribeTasksInput{
		Cluster: &cluster,
		Tasks:   []*string{&taskArn},
	}

	taskStatusChannel <- RunTaskResult{Error: nil, Status: TaskPending}
	if err := client.WaitUntilTasksRunning(describeTaskInput); err != nil {
		fmt.Println(err.Error())
		// TODO: Is it really definitely stopped yet?
		taskStatusChannel <- RunTaskResult{Error: err, Status: TaskStopped}
		return
	}
	taskStatusChannel <- RunTaskResult{Error: nil, Status: TaskRunning}
	// I have seen these just take longer than the allowed timeout on the waiter and needed to wait in a loop
	err := client.WaitUntilTasksStopped(describeTaskInput)
	taskStatusChannel <- RunTaskResult{Error: err, Status: TaskStopped}
	return
}

func streamCloudwatchLogs(client *cloudwatchlogs.CloudWatchLogs, logGroup string, logStreamName string, done <-chan bool) {
	// TODO: this runs as a goroutine... does it NEED the done chan? It will just exit when the main program loop
	// exits anyway with the current use case. It does give a way to ensure it stops if I want to stop it early, though.
	logEventsInput := &cloudwatchlogs.GetLogEventsInput{}
	logEventsInput.SetStartFromHead(true)
	logEventsInput.SetLimit(10)
	logEventsInput.SetLogGroupName(logGroup)
	logEventsInput.SetLogStreamName(logStreamName)

	for {
		output, _ := client.GetLogEvents(logEventsInput)
		// TODO: actually do something with the error but many of these errors are just temporary while waiting
		for _, event := range output.Events {
			// event.Timetstamp is unix epoch MILLISECONDS
			// TODO: allow structured output - convert the whole event to json and dump it
			// TODO: allow timestamp in desired timezone
			// TODO: abstract this elsewhere - will need reused for a general stream task logs command
			//       and have cloudWatchChannel allow the message to be passed back - maybe just the message
			//			 and do the rest of the formatting elsewhere? or pass back the formatted message?
			//			 Probably abstract into my own struct with a PrettyPrint() receiver
			// TODO: return this data + more over a channel rather than printing here so that receiver
			// can do whatever with it
			fmt.Printf("[%s] %s\n", time.Unix(*event.Timestamp/1000, 0).In(time.UTC), *event.Message)
		}
		logEventsInput.NextToken = output.NextForwardToken

		// check to see if the task has completed so we can exit or sleep before the next api call
		select {
		case <-done:
				return
		default:
			// TODO: configurable sleep time?
			time.Sleep(time.Second * 3) // Randomly selected sleep time
		}

	}
}
