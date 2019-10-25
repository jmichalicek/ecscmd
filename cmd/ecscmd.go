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
	"github.com/knadh/koanf"
	"github.com/knadh/koanf/parsers/toml"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"os"
	"path"
)

// type rootConfig struct {
// 	ConfigFile string
// 	AwsProfile *string
// 	AwsRegion *string
// }

// mauy make these public like my initial example above
type rootConfig struct {
	configFile string
	dryRun     bool
}

var baseConfig rootConfig

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

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		exitWithError(err)
	}
}

func init() {
	// TODO: is there a good way I can pull this stuff out of init() and do it more like
	// https://www.netlify.com/blog/2016/09/06/creating-a-microservice-boilerplate-in-go/ but with all
	// the subcommands, etc?
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVar(&baseConfig.configFile, "config", "", "config file (default is $HOME/.ecscmd.toml)")
	// TODO: Make this per command just to provide more specific help/description for how it affects that command?
	rootCmd.PersistentFlags().BoolVar(&baseConfig.dryRun, "dry-run", false, "Perform dry-run. Does not actually send command. Output info about what would have been performed.")
	// rootCmd.PersistentFlags().StringVar(rconf.AwsProfile, "profile", "", "profile to use from ~/.aws/config and ~/.aws/credentials")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	// This might be built into golang 1.12
	home, err := homedir.Dir()
	if err != nil {
		exitWithError(err)
	}
	// TODO: look in current dir first, then in home
	// TODO: other config file formats, custom config file path from command line
	if baseConfig.configFile != "" {
		if canUseFile(baseConfig.configFile) {
			k.Load(file.Provider(baseConfig.configFile), toml.Parser())
		} else {
			exitWithError(fmt.Errorf("Cannot load specified config file: %s", baseConfig.configFile))
		}
	} else {
		// TODO: which should take precedence? ~/.ecscmd.toml FIRST to load defaults and then override project specific
		// or local dir first (as is now) to provide general, in code repo default, and let user override with ~/.ecscmd.toml
		// TODO: load other config file formats... .yml, etc.
		// TODO: may switch to yaml by default (or only) - I like it better for the config structure ecscmd needs.
		// TODO: for configs -- allow multiple --config="foo.toml" and parse in order, later ones overriding earlier ones?
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

func exitWithError(err error) {
	fmt.Println(err.Error())
	os.Exit(1)
}
