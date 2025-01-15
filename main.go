/*
Copyright © 2022 steffakasid
*/
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/fatih/color"
	flag "github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/steffakasid/eslog"
	"github.com/steffakasid/kubectl-co/internal"
)

type cmdCfg struct {
	Delete   bool `mapstructure:"delete"`
	Debug    bool `mapstructure:"debug"`
	Add      bool `mapstructure:"add"`
	Previous bool `mapstructure:"previous"`
	Current  bool `mapstructure:"current"`
}

var config *cmdCfg = &cmdCfg{}
var co *internal.CO

var version = "0.1-development"

const (
	viperKeyPrevious = "previous"
	viperKeyCurrent  = "current"
	viperKeyDebug    = "debug"
	viperKeyDelete   = "delete"
	viperKeyAdd      = "add"
	viperKeyHelp     = "help"
	viperKeyVersion  = "version"
)

func init() {
	var err error

	flag.BoolP(viperKeyDelete, "d", false, "Delete the config with the given name. Usage: kubectl co --delete [configname]")
	flag.BoolP(viperKeyAdd, "a", false, "Add a new given config providing the path and the name. Usage: kubectl co --add [configpath] [configname]")
	flag.BoolP(viperKeyPrevious, "p", false, "Switch to previous config")
	flag.BoolP(viperKeyCurrent, "c", false, "Show the current config path")
	flag.BoolP(viperKeyHelp, "h", false, "Show help")
	flag.Bool(viperKeyDebug, false, "Turn on debug output")

	flag.Usage = func() {
		stdErr := os.Stderr

		fmt.Fprintf(stdErr, "Usage of %s: \n", os.Args[0])
		fmt.Fprintln(stdErr, `
This tool can be used to work with multiple kube configs. It allows to
add, delete and switch config files.

NOTE: If you set the KUBECONFIG environment var this will always take precedence before the config file.

Preqrequisites:
  kubectl should be installed (even if the application would also run for it own as 'kubectl-co')

Examples:
  kubectl co --add new-config ~/.kube/config    - adds your current kubeconfig to be used by co with the name 'new-config'
  kubectl co --add completly-new                - adds a plain new config file which must be inialised afterwards
  kubectl co --previous                         - switch to previous config and set current config to previous
  kubectl co --delete config-name               - delete config with name 'new-config'
  kubectl co --current                          - show the current config path
  kubectl co new-config                         - switch to 'new-config' this will overwrite ~/.kube/config with a symbolic link
  kubectl co                                    - list all available configs

Usage:
  kubectl co [flags]
  kubectl-co [flags]

Flags:`)

		flag.PrintDefaults()
	}

	flag.Parse()
	home, err := os.UserHomeDir()
	eslog.LogIfErrorf(err, eslog.Fatalf, "Can not get homedir: %s")

	viper.AddConfigPath(path.Join(home, ".config", "kubectl-co"))
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	err = viper.ReadInConfig()
	eslog.LogIfErrorf(err, eslog.Errorf, "Error reading config: %s")
	err = viper.BindPFlags(flag.CommandLine)
	eslog.LogIfErrorf(err, eslog.Fatalf, "Error binding flags: %s")
	err = viper.Unmarshal(config)
	eslog.LogIfErrorf(err, eslog.Fatalf, "Error unmarshal config: %s")

	if config.Debug {
		err = eslog.Logger.SetLogLevel("debug")
		eslog.LogIfErrorf(err, eslog.Fatalf, "Error SetLogLevel(debug): %s")
	} else {
		err = eslog.Logger.SetLogLevel("info")
		eslog.LogIfErrorf(err, eslog.Fatalf, "Error SetLogLevel(info): %s")
	}

	co, err = internal.NewCO(home)
	eslog.LogIfErrorf(err, eslog.Fatalf, "Error initializing co: %s")
}

func main() {
	if viper.GetBool(viperKeyVersion) {
		fmt.Printf("kubectl-co version: %s\n", version)
	} else if viper.GetBool(viperKeyHelp) {
		flag.Usage()
	} else {
		args := flag.Args()
		err := validateFlags(args)
		eslog.LogIfErrorf(err, eslog.Fatalf, "Error validating flags: %s")

		if len(args) > 0 {
			co.ConfigName = args[0]
		}
		execute(args)
	}
}

func validateFlags(args []string) error {
	eslog.Debugf("config %s", toString(config))

	if (config.Current && config.Previous) || (config.Delete && config.Previous) || (config.Delete && config.Current) || (config.Add && config.Previous) || (config.Add && config.Current) || (config.Add && config.Delete) {
		return fmt.Errorf("%s, %s, %s and %s are exklusiv just use one at a time", viperKeyAdd, viperKeyDelete, viperKeyPrevious, viperKeyCurrent)
	} else if config.Delete && len(args) != 1 {
		return fmt.Errorf("When using %s you must only provide the name of the config to be deleted!", viperKeyDelete)
	} else if config.Add && (len(args) == 0 || len(args) > 2) {
		return fmt.Errorf("When using %s you must provide the path as first argument and the name of the config as second argument!", viperKeyAdd)
	} else if config.Previous && len(args) != 0 {
		return fmt.Errorf("%s doesn't take any arguments", viperKeyPrevious)
	}
	return nil
}

func execute(args []string) {
	var configs []string
	var err error

	if config.Add {
		copyConfigFrom := ""
		if len(args) == 2 {
			copyConfigFrom = args[1]
		}
		err = co.AddConfig(copyConfigFrom)
		if err != nil {
			err = co.LinkKubeConfig()
		}
	} else if config.Delete {
		err = co.DeleteConfig()
	} else if config.Previous || len(args) == 1 {
		err = co.LinkKubeConfig()
	} else {
		configs, err = co.ListConfigs()

		red := color.New(color.FgRed)

		for _, config := range configs {
			if strings.HasSuffix(co.CurrentConfigPath, config) {
				red.Println(config)
			} else {
				fmt.Println(config)
			}
		}
	}
	eslog.LogIfErrorf(err, eslog.Fatalf, "Error on execute: %s")
}

func toString(obj any) string {

	bt, err := json.Marshal(obj)
	eslog.LogIfErrorf(err, eslog.Errorf, "error marshalling obj to json string: %s")

	return string(bt)
}
