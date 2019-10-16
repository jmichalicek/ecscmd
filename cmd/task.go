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
	"log"
	"os"
	"time"
	"strings"
)

// Limited subset of functionality for now
type runTaskCommandOptions struct {
	Cluster string `koanf:"cluster"`
	Count int64 `koanf:"count"`
	Group string `koanf:"group"`
	TaskDefinition string `koanf:"family"`
	WaitForComplete bool
	StreamOutput bool
	Fargate bool
	// EnableECSManagedTags bool
	// The name of the task group to associate with the task. The default value
  // is the family name of the task definition (for example, family:my-family-name).
}

// TODO: this is gross and clunky. I may have to look into something other than cobra... or maybe
// I am doing something wrong. Some of their examples were like this.
var runTaskOptions runTaskCommandOptions = runTaskCommandOptions{Fargate: true}

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
	Run: func(cmd *cobra.Command, args []string) {
		// TODO: too much going on here... or will be. This needs to be its own function defined elsewhere
		var taskDefName = args[0]
		var configKey = fmt.Sprintf("taskdef.%s", taskDefName)
		// taskDefConfig := k.Get(configKey).(map[string]interface{})
		taskDefConfig := k.Cut(configKey).Raw()
		// TODO: not certain this is the way to go given that aws-sdk-go doesn't use the json for this
		// but it's an easy-ish way to make it clear, modifiable, work with all kinds of vars
		containerDefBytes, err := taskdef.ParseContainerDefTemplate(taskDefConfig)
		cdef, err := taskdef.MakeContainerDefinitions(containerDefBytes)
		// ideally could just pass taskDefConfig and get this back with something else wrapping the above stuff
		// and this.
		i, err := taskdef.NewTaskDefinitionInput(taskDefConfig, cdef)
		if err != nil {
			log.Fatalf("[ERROR] %s", err)
		}

		session, err := session.NewAwsSession(taskDefConfig)
		if err != nil {
			log.Fatalf("[ERROR] %s", err)
		}
		// TODO: look at source for how this is implemented to handle both this OR with extra config
		// both on ecs.New()
		client := ecs.New(session)

		if baseConfig.dryRun {
			fmt.Printf("%v", i)
		} else {
			result, err := taskdef.RegisterTaskDefinition(i, client)
			if err != nil {
				log.Fatalf("[ERROR] %s", err)
			}
			// not sure how I feel about using log vs fmt here. If actually going into a log, the timestamp is great
			// but for regular useful user output...meh. May just want to do both stdout and log
			log.Printf("[INFO] AWS Response:\n%v\n", result)
		}

	},
}

var cmdRunTask = &cobra.Command{
	// TODO: deal with using either the actual task def name/family/arn OR our local config name
	Use:   "run taskDefName",
	Short: "Run an ECS task. Currently assumes a Fargate launch type.",
	Long: `Run an ECS task. Currently assumes a Fargate launch type.`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// TODO: Wow, this has gotten messy after starting to add logstream watching... needs cleaned up even more than I feared it would
		// TODO: really should make the name of the config optional - may want to run a task
		// which there is no local config for
		var taskDefName = args[0]
		var configKey = fmt.Sprintf("taskdef.%s", taskDefName)
		config := k.Cut(configKey).Raw()
		session, err := session.NewAwsSession(config)
		if err != nil {
			fmt.Printf("[ERROR] %s", err)
			os.Exit(1)
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
			fmt.Printf("%s", err)
			os.Exit(1)
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
			fmt.Printf("[ERROR] %s", err)
			os.Exit(1)
		}

		// handle the output.
		// TODO: Make a nice interface to use for these and a single "dostuff()"
		// function which takes that interface - then could call mything.DryRun()
		// or MyConfig.Execute() and those could return a standard interface and errors
		// and then the if err != nil stuff could live in one place instead of every Run function
		// Unsure how to abstract stuff like waiting on a waiter, though.
		if baseConfig.dryRun {
			// TODO: better output here - really should try to look up the task def on aws
			fmt.Printf("Would run task %v\n", config)
			os.Exit(0)
		}

		result, err := client.RunTask(&runTaskInput)
		// result, err := taskdef.RegisterTaskDefinition(i, client)
		if err != nil {
			fmt.Printf("[ERROR] %s\n", err)
			os.Exit(1)
		}

		if runTaskOptions.WaitForComplete || runTaskOptions.StreamOutput {
			taskCompleteChannel := make(chan bool)
			cloudWatchChannel := make(chan string)
			taskArn := result.Tasks[0].TaskArn
			describeTaskInput := &ecs.DescribeTasksInput{
				Cluster: runTaskInput.Cluster,
				Tasks: []*string{taskArn},
			}
			// TODO: I feel like these should be their own functions, not nested here.
			go func() {
				if err := client.WaitUntilTasksRunning(describeTaskInput); err != nil {
					fmt.Printf("\n[ERROR] %s\n", err)
					taskCompleteChannel <- false
					return
				}

				fmt.Println("Task Started")
				// stopping via the web console doesn't seem to cause this waiter to stop.
				if err := client.WaitUntilTasksStopped(describeTaskInput); err != nil {
					// I have actually seen these waiters take longer than the timeout but the thing actually work
					// but it's rare
					fmt.Printf("\n[ERROR] %s\n", err)
					taskCompleteChannel <- false
					return
				}
				fmt.Println("Task complete!")
				taskCompleteChannel <- true
			}()

			streamCloudwatchLogs := func() {
				taskArn := result.Tasks[0].TaskArn
				parts := strings.Split(*taskArn, "/")
				taskId := parts[2]
				cwclient := cloudwatchlogs.New(session)
				logStreamName = logStreamName + taskId
				logEventsInput := &cloudwatchlogs.GetLogEventsInput{}
				logEventsInput.SetStartFromHead(true)
				logEventsInput.SetLimit(10)
				logEventsInput.SetLogGroupName(*logGroup)
				logEventsInput.SetLogStreamName(logStreamName)
				// should maybe wait for task to start using a channel for that!

				for {
					output, _ := cwclient.GetLogEvents(logEventsInput)
					// TODO: actually do something with the error but many of these errors are just temporary while waiting
					for _, event := range output.Events {
							// event.Timetstamp is unix epoch MILLISECONDS
							// TODO: allow structured output - convert the whole event to json and dump it
							// TODO: allow timestamp in desired timezone
					    fmt.Printf("[%s] %s\n", time.Unix(*event.Timestamp / 1000, 0).In(time.UTC), *event.Message)
					}
					logEventsInput.NextToken = output.NextForwardToken

					// check to see if the task has completed so we can exit or sleep before the next api call
					select {
					case <- taskCompleteChannel:
						return
					default:
						// TODO: configurable sleep time?
						time.Sleep(time.Second * 3) // Randomly selected sleep time
					}

				}
			}

			if runTaskOptions.StreamOutput {
				go streamCloudwatchLogs()
			}

			done := false
			for {
				select {
					case s := <- cloudWatchChannel:
					fmt.Printf("%s", s)
					case s := <- taskCompleteChannel:
						// Relying on the waiter function to print the errors. Not so sure that's a great idea long term.
						// thinking maybe two channels... taskErrors, taskOutput which would keep all output here.
						if !s {
							os.Exit(1)
						}
						done = true
					default:
						if !runTaskOptions.StreamOutput {
							// TODO: print dots UNTIL it is running?
							fmt.Printf(".")
							time.Sleep(time.Second)
						}
				}
				if done {
					break
				}
			}
			close(taskCompleteChannel)
			close(cloudWatchChannel)
		}
	},
}

func init() {
	rootCmd.AddCommand(taskCmd)
	taskCmd.AddCommand(cmdRegisterTaskDef)
	taskCmd.AddCommand(cmdRunTask)

	// TODO: still gross. I want these processed AFTER reading options from config, anyway.
	cmdRunTask.Flags().StringVar(&runTaskOptions.TaskDefinition, "task-definition", "", "Task definition arn for the task to run. This could be full arn, family, or family:revision")
	cmdRunTask.Flags().StringVar(&runTaskOptions.Cluster, "cluster", "", "Cluster to run the task on. Defaults to AWS default cluster.")
	cmdRunTask.Flags().Int64Var(&runTaskOptions.Count, "count", 1, "How many of this task to run.")
	cmdRunTask.Flags().BoolVar(&runTaskOptions.WaitForComplete, "wait-for-stop", false, "Wait for the task to complete before continuing.")
	cmdRunTask.Flags().BoolVar(&runTaskOptions.StreamOutput, "stream-logs", false, "Stream cloudwatch logs for the task.")


	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// serviceCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// serviceCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
