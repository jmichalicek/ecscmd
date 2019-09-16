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
	"github.com/jmichalicek/ecscmd/service"
	"github.com/jmichalicek/ecscmd/session"
	"github.com/spf13/cobra"
	"log"
)

// should/could this be a pflag thing? that might actually clean up the cobra + koanf stuff
// TODO: would like these to be pointers to be clear between "not set" and set to something falsey or empty
// maybe can use this https://github.com/spf13/cobra/issues/434#issuecomment-299128976
// TODO: this currently is to hold the options passed in on the command line and parsed by cobra
// so that the values can then be merged into the map created by koanf. But MAYBE this should be the final target
// with koanf values put here, THEN command line options parsed to override (bool values may cause issues) and then
// THIS passed to NewUpdateServiceInput but then these all need to be nillable to tell that they are unset
// which causes extra work in setting them.
type serviceCommandOptions struct {
	// updateservice specific
	assignPublicIp bool // or string since real value is ENABLED or DISABLED ?
	taskDefinition string
	subnets        []string
	securityGroups []string
	cluster        string
	desiredCount   int64 // aws sdk takes int64, so just use that all the way through
}

type updateServiceCommandOptions struct {
	serviceCommandOptions
	forceDeployment bool
}

type createServiceCommandOptions struct {
	serviceCommandOptions
	launchType string // create service specific
	name       string // create service specific
}

// TODO: this is gross and clunky. I may have to look into something other than cobra... or maybe
// I am doing something wrong. Some of their examples were like this.
var createServiceOptions createServiceCommandOptions = createServiceCommandOptions{}
var updateServiceOptions updateServiceCommandOptions = updateServiceCommandOptions{}

// serviceCmd represents the service command
var serviceCmd = &cobra.Command{
	Use:   "service",
	Short: "Commands for managing ECS services",
	// Run: func(cmd *cobra.Command, args []string) {
	// 	fmt.Println("service called")
	// },
}

var createServiceCmd = &cobra.Command{
	Use:   "create <serviceName>",
	Short: "Create a new ECS service using defaults from the configuration name provided",
	Args:  cobra.MinimumNArgs(1), // maybe make this an optional flag --config-profile ?
	Run: func(cmd *cobra.Command, args []string) {
		// TODO: if service already exists we get a not so clear error "InvalidParameterException: Creation of service was not idempotent."
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
		if createServiceOptions.taskDefinition != "" {
			serviceConfig["taskDefinition"] = createServiceOptions.taskDefinition
		}

		if createServiceOptions.launchType != "" {
			serviceConfig["launchType"] = createServiceOptions.launchType
		}

		_, ok := serviceConfig["desiredCount"]
		if !ok || cmd.Flags().Changed("desired-count") {
			// user set the flag as opposed to being 0 by default
			serviceConfig["desiredCount"] = createServiceOptions.desiredCount
		} else {
			// this is gross. koanf is reading the integer from toml as a float64
			serviceConfig["desiredCount"] = int64(serviceConfig["desiredCount"].(float64))
		}

		// ideally could just pass taskDefConfig and get this back with something else wrapping the above stuff
		// and this.
		i, err := service.NewCreateServiceInput(serviceConfig)
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
		if baseConfig.dryRun {
			fmt.Printf("%v", i)
		} else {
			result, err := client.CreateService(i)
			if err != nil {
				log.Fatalf("[ERROR] %s", err)
			}
			log.Printf("[INFO] AWS Response:\n%v\n", result)
		}
	},
}

// should this instaed be `ecscmd service udpate` and `ecscmd service create`, etc?
var updateServiceCmd = &cobra.Command{
	Use:   "update <serviceName>",
	Short: "Update an existing ECS Service",
	Long:  `Update an existing ECS Service`,
	Args:  cobra.MinimumNArgs(1), // maybe make this an optional flag --config-profile ?
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
		if updateServiceOptions.taskDefinition != "" {
			serviceConfig["taskDefinition"] = updateServiceOptions.taskDefinition
		}

		if updateServiceOptions.forceDeployment {
			serviceConfig["forceDeployment"] = true
		}
		if cmd.Flags().Changed("desired-count") {
			fmt.Println("setting desired-count!!")
			// user set the flag as opposed to being 0 by default
			// TODO: replace cobra? forcing a default value makes things super unclear and clunky. The command help
			// now shows a default of 0, but really default is to keep it as is...
			serviceConfig["desiredCount"] = updateServiceOptions.desiredCount
		} else {
			// this is gross. koanf is reading the integer from toml as a float64
			if dc, ok := serviceConfig["desiredCount"]; ok {
				serviceConfig["desiredCount"] = int64(dc.(float64))
			}
		}

		// ideally could just pass taskDefConfig and get this back with something else wrapping the above stuff
		// and this.
		i, err := service.NewUpdateServiceInput(serviceConfig)
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
		if baseConfig.dryRun {
			fmt.Printf("%v", i)
		} else {
			result, err := client.UpdateService(i)
			if err != nil {
				log.Fatalf("[ERROR] %s", err)
			}

			log.Printf("[INFO] AWS Response:\n%v\n", result)
		}

	},
}

func init() {
	rootCmd.AddCommand(serviceCmd)
	serviceCmd.AddCommand(createServiceCmd)
	serviceCmd.AddCommand(updateServiceCmd)

	// variables for viper to store command line flag values to. this feels incredibly clunky and inelegant.
	// the mixing of cobra/viper/koanf is gross, too.
	updateServiceCmd.Flags().StringVarP(&updateServiceOptions.taskDefinition, "task-definition", "t", "", "Task definition arn for the service to use.")
	updateServiceCmd.Flags().BoolVar(&updateServiceOptions.forceDeployment, "force-deployment", false, "Task definition arn for the service to use.")
	updateServiceCmd.Flags().Int64Var(&updateServiceOptions.desiredCount, "desired-count", 0, "Desired number of the service to run. Default is really to keep the same, not 0, but cobra is dumb about this.")

	// TODO: task-definition as a persistent flag on the service command?
	createServiceCmd.Flags().StringVar(&createServiceOptions.taskDefinition, "task-definition", "", "Task definition arn for the service to use.")
	createServiceCmd.Flags().Int64Var(&createServiceOptions.desiredCount, "desired-count", 0, "Desired number of the service to run.")
	createServiceCmd.Flags().StringVar(&createServiceOptions.launchType, "launch-type", "FARGATE", "Desired number of the service to run.")
	createServiceCmd.Flags().StringVar(&createServiceOptions.name, "name", "", "Name of the service to create")
}
