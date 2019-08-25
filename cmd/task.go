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

import (
	"fmt"

	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/jmichalicek/ecscmd/session"
	"github.com/jmichalicek/ecscmd/taskdef"
	"github.com/spf13/cobra"
	"log"
)

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

		result, err := taskdef.RegisterTaskDefinition(i, client)
		if err != nil {
			log.Fatalf("[ERROR] %s", err)
		}

		// not sure how I feel about using log vs fmt here. If actually going into a log, the timestamp is great
		// but for regular useful user output...meh. May just want to do both stdout and log
		log.Printf("[INFO] AWS Response:\n%v\n", result)
	},
}

func init() {
	rootCmd.AddCommand(taskCmd)
	taskCmd.AddCommand(cmdRegisterTaskDef)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// serviceCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// serviceCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}