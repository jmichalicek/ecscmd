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
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/jmichalicek/ecscmd/service"
	"github.com/jmichalicek/ecscmd/session"
	"github.com/jmichalicek/ecscmd/taskdef"

	"github.com/hashicorp/logutils"
	"github.com/knadh/koanf"
	"github.com/knadh/koanf/parsers/toml"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"log"
	"os"
	"path"
	"strings"
)

// type rootConfig struct {
// 	ConfigFile string
// 	AwsProfile *string
// 	AwsRegion *string
// }

type rootConfig struct {
	configFile string
	logLevel   string
}

var baseConfig rootConfig

var k = koanf.New(".") // TODO: just following the docs/examples for now. Not a fan of the global

// variables for viper to store command line flag values to... this feels incredibly clunky and inelegant.
var serviceTaskDef string

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

// TODO: may go back to full taskdef as json template. awscli allows for using a json file so
// hopefully I can just unmarshal the whole thing to RegistTaskDefinitionInput, maybe.
// which could simplify the config structure to being almost all template vars + a few aws session details like region, profile, etc
// need to see how this works as is with creating a new taskdef and as far as updating existing taskdef - want to be able
// to easily just update the bare minimum which most of the time will just be container defs to update an image
var cmdRegisterTaskDef = &cobra.Command{
	Use:   "register-task-def taskDefName",
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

// should this instaed be `ecscmd service udpate` and `ecscmd service create`, etc?
var cmdUpdateService = &cobra.Command{
	Use:   "update-service <serviceName>",
	Short: "Update an existing ECS Service",
	Long:  `Update an existing ECS Service`,
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// TODO: too much going on here... or will be. This needs to be its own function defined elsewhere
		// TODO: skip grouping as sevice.* and taskdef.* and just use name?
		var configName = args[0]
		var configKey = fmt.Sprintf("service.%s", configName)
		// k2 := k.Cut(configKey)
		// serviceConfig := k2.Raw()
		serviceConfig := k.Cut(configKey).Raw()

		// TODO: again, super clunky and inelegant... there must be a better way, but mixing Cobra for its nested commands
		// with koanf for its better parsing of everything else seems to leave few options here and they all kind of suck.
		// taking command line options here and using them to override settings from configs
		if &serviceTaskDef != nil {
			serviceConfig["taskDefinition"] = serviceTaskDef
		}

		// ideally could just pass taskDefConfig and get this back with something else wrapping the above stuff
		// and this.
		i, err := service.NewFargateUpdateServiceInput(serviceConfig)
		if err != nil {
			log.Fatalf("[ERROR] %s", err)
		}
		log.Printf("[DEBUG] %v", i)

		session, err := session.NewAwsSession(serviceConfig)
		if err != nil {
			log.Fatalf("[ERROR] %s", err)
		}
		// TODO: look at source for how this is implemented to handle both this OR with extra config
		// both on ecs.New()
		client := ecs.New(session)

		// result, err := taskdef.RegisterTaskDefinition(i, client)
		// TODO: updateservice call
		result, err := client.UpdateService(i)
		if err != nil {
			log.Fatalf("[ERROR] %s", err)
		}

		log.Printf("[INFO] AWS Response:\n%v\n", result)
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
	rootCmd.PersistentFlags().StringVar(&baseConfig.configFile, "config", "", "config file (default is $HOME/.ecscmd.toml)")
	rootCmd.PersistentFlags().StringVar(&baseConfig.logLevel, "log-level", "INFO", "Minimum level for log messages. Default is INFO.")
	// rootCmd.PersistentFlags().StringVar(rconf.AwsProfile, "profile", "", "profile to use from ~/.aws/config and ~/.aws/credentials")

	// TODO? Make deeper subcommands like below?
	// ecscmd taskdef register
	// ecscmd service update
	rootCmd.AddCommand(cmdRegisterTaskDef)
	rootCmd.AddCommand(cmdUpdateService)

	// variables for viper to store command line flag values to... this feels incredibly clunky and inelegant.
	// the mixing of cobra/viper/koanf is gross, too.
	cmdUpdateService.Flags().StringVarP(&serviceTaskDef, "taskdef", "t", "", "Task definition arn for the service to use.")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	// This is when cobra has initialized and so logLevel has been properly set
	filter := &
logutils.LevelFilter{
		Levels:   []logutils.LogLevel{"DEBUG", "INFO", "WARN", "ERROR"},
		MinLevel: logutils.LogLevel(strings.ToUpper(baseConfig.logLevel)),
		Writer:   os.Stderr,
	}
	log.SetOutput(filter)

	// This might be built into golang 1.12
	home, err := homedir.Dir()
	if err != nil {
		log.Fatal(err)
	}
	// TODO: look in current dir first, then in home
	// TODO: other config file formats, custom config file path from command line
	if baseConfig.configFile != "" {
		if canUseFile(baseConfig.configFile) {
			k.Load(file.Provider(baseConfig.configFile), toml.Parser())
		} else {
			log.Fatalf("[ERROR] Cannot load specified config file: %s", baseConfig.configFile)
		}
	} else {
		// TODO: which should take precedence? ~/.ecscmd.toml FIRST to load defaults and then override project specific
		// or local dir first (as is now) to provide general, in code repo default, and let user override with ~/.ecscmd.toml
		// TODO: load other config file formats... .yml, etc.
		// TODO: may switch to yaml by default (or only) - I like it better for the config structure ecscmd needs.
		projectConfig := path.Join(".", ".ecscmd.toml")
		if canUseFile(projectConfig) {
			k.Load(file.Provider(projectConfig), toml.Parser())
		}

		defaultConfig := path.Join(home, ".ecscmd.toml")
		if canUseFile(defaultConfig) {
			k.Load(file.Provider(defaultConfig), toml.Parser())
		}
	}

	k.Load(env.Provider("", ".", nil), nil)
	// TODO: override further via command line
}

// TODO: this should live somewhere more reusable
// TODO: does this expand ~ or $HOME?
func canUseFile(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	// there could be other errors wher the file exists but is not usable still for some reason.
	return err == nil && !info.IsDir()
}
