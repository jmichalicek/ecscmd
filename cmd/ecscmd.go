/*
Copyright © 2019 Justin Michalicek

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
  "os"
  "github.com/spf13/cobra"
	"strings"
  homedir "github.com/mitchellh/go-homedir"
	"github.com/jmichalicek/ecscmd/taskdef"
	"github.com/knadh/koanf"
	"github.com/knadh/koanf/parsers/toml"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/providers/env"
)


var cfgFile string
var k = koanf.New(".") // TODO: just following the docs/examples for now. Not a fan of the global


// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
  Use:   "ecscmd",
  Short: "Easily update and deploy ECS services and tasks",
  Long: `ecscmd is a tool for managing AWS ECS to make deployments and updates simpler.

  ecscmd enables temlated updates to task definitions, updates to services, and running
  one off ECS tasks. Separating these out into a single, simple application allows easy use
  of this for deployment situations so that ECS deployments may be more easily decoupled from
  infrastructure tooling such as Terraform or CloudFormation.`,
  // Uncomment the following line if your bare application
  // has an action associated with it:
  //	Run: func(cmd *cobra.Command, args []string) { },
}

// This probably belongs in its own file?

// task def example in config file will be like: maybe?
// [taskDefinitions]
//   [taskDefinitions.name1]
//     stuff here... container def variables
//   [taskDefinitions.name2]
//     stuff here
// [taskDefinition]
// And then can do ecscmd register-task-def name1 --containerDefs='path/to/template' [container def template vars here somehow?] --other-properties, etc.
// with defaults coming from config.  But might also just use list of [taskDefinition] blocks
var cmdRegisterTaskDef = &cobra.Command{
    Use:   "register-task-def taskDefName",
    Short: "Register an ECS task definition",
    Long: `Register a new task definition or update an existing task definition.
    A taskDefinition section should exist in the config file`,
    Args: cobra.MinimumNArgs(1),
    Run: func(cmd *cobra.Command, args []string) {
			// TODO: too much going on here... or will be
      fmt.Println("Print: " + strings.Join(args, " "))
			var taskDefName = args[0]
			var configKey = fmt.Sprintf("taskdef.%s", taskDefName)
			taskDefConfig := k.Get(configKey).(map[string]interface{})
			fmt.Println("Config: " + fmt.Sprintf("%v", taskDefConfig))
			// TODO: not certain this is the way to go given that aws-sdk-go doesn't use the json for this
			// but it's an easy-ish way to make it clear, modifiable, work with all kinds of vars
			// so maybe it will unmarshal to the types I need.
			taskdef, err := taskdef.ParseTemplate(taskDefConfig)
			fmt.Println("Taskdef: " + taskdef)
			fmt.Println("Err: " + fmt.Sprintf("%s", err))
    },
  }

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
  if err := rootCmd.Execute(); err != nil {
    fmt.Println(err)
    os.Exit(1)
  }
}

func init() {
  cobra.OnInitialize(initConfig)

  // Here you will define your flags and configuration settings.
  // Cobra supports persistent flags, which, if defined here,
  // will be global for your application.

  rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.ecscmd.yaml)")


  // Cobra also supports local flags, which will only run
  // when this action is called directly.
  rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")

	rootCmd.AddCommand(cmdRegisterTaskDef)
}


// initConfig reads in config file and ENV variables if set.
func initConfig() {
	home, err := homedir.Dir()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	// TODO: look in current dir first, then in home
	// TODO: other config file formats, custom config file path from command line
	k.Load(file.Provider(fmt.Sprintf("%s/.ecscmd.toml", home)), toml.Parser())
	k.Load(env.Provider("", ".", nil), nil)
}
