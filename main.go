/*
Copyright © 2022 steffakasid
*/
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
	logger "github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/steffakasid/kubectl-co/internal"
)

type cmdCfg struct {
	Delete   bool `mapstructure:"delete"`
	Debug    bool `mapstructure:"debug"`
	Add      bool `mapstructure:"add"`
	Previous bool `mapstructure:"previous"`
	Current  bool `mapstructure:"current"`
}

var c *cmdCfg = &cmdCfg{}
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
	logger.SetLevel(logger.InfoLevel)

	flag.BoolP(viperKeyDelete, "d", false, "Delete the config with the given name. Usage: kubectl co --delete [configname]")
	flag.BoolP(viperKeyAdd, "a", false, "Add a new given config providing the path and the name. Usage: kubectl co --add [configpath] [configname]")
	flag.BoolP(viperKeyPrevious, "p", false, "Switch to previous config")
	flag.BoolP(viperKeyCurrent, "c", false, "Show the current config path")
	flag.BoolP(viperKeyHelp, "h", false, "Show help")
	flag.Bool(viperKeyDebug, false, "Turn on debug output")

	flag.Usage = func() {
		w := os.Stderr

		fmt.Fprintf(w, "Usage of %s: \n", os.Args[0])
		fmt.Fprintln(w, `
This tool can be used to work with multiple kube configs. It allows to
add, delete and switch config files.

NOTE: If you set the KUBECONFIG environment var this will always take precedence before the config file.

Preqrequisites:
  kubectl should be installed (even if the application would also run for it own as 'kubectl-co')

Examples:
  kubectl co --add new-config ~/.kube/config    - adds your current kubeconfig to be used by co with the name 'new-config'
  kubectl co --add completly-new                - adds a plain new config file which must be inialised afterwards
  kubectl co --previous                         - switch to previous config and set current config to previous
  kubectl co --delete new-config                - delete config with name 'new-config'
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
	err = viper.BindPFlags(flag.CommandLine)
	CheckError(err, logger.Fatalf)
	err = viper.Unmarshal(c)
	CheckError(err, logger.Fatalf)

	if c.Debug {
		logger.SetLevel(logger.DebugLevel)
	}

	home, err := os.UserHomeDir()
	CheckError(err, logger.Fatalf)
	co, err = internal.NewCO(home)
	CheckError(err, logger.Fatalf)
}

func main() {
	if viper.GetBool(viperKeyVersion) {
		fmt.Printf("kubectl-co version: %s\n", version)
	} else if viper.GetBool(viperKeyHelp) {
		flag.Usage()
	} else {
		args := flag.Args()
		err := validateFlags(args)

		CheckError(err, logger.Fatalf)

		if len(args) > 0 {
			co.ConfigName = args[0]
		}
		execute(args)
	}
}

func validateFlags(args []string) error {
	logger.Debug("config", c)

	if (c.Current && c.Previous) || (c.Delete && c.Previous) || (c.Delete && c.Current) || (c.Add && c.Previous) || (c.Add && c.Current) || (c.Add && c.Delete) {
		return fmt.Errorf("%s, %s, %s and %s are exklusiv just use one at a time", viperKeyAdd, viperKeyDelete, viperKeyPrevious, viperKeyCurrent)
	} else if c.Delete && len(args) != 1 {
		return fmt.Errorf("When using %s you must only provide the name of the config to be deleted!", viperKeyDelete)
	} else if c.Add && (len(args) == 0 || len(args) > 2) {
		return fmt.Errorf("When using %s you must provide the path as first argument and the name of the config as second argument!", viperKeyAdd)
	} else if c.Previous && len(args) != 0 {
		return fmt.Errorf("%s doesn't take any arguments", viperKeyPrevious)
	}
	return nil
}

func execute(args []string) {
	var configs []string
	var err error

	if c.Add {
		copyConfigFrom := ""
		if len(args) == 2 {
			copyConfigFrom = args[1]
		}
		err = co.AddConfig(copyConfigFrom)
		co.LinkKubeConfig()
	} else if c.Delete {
		err = co.DeleteConfig()
	} else if c.Previous || len(args) == 1 {
		err = co.LinkKubeConfig()
	} else {
		configs, err = co.ListConfigs()

		red := color.New(color.FgRed)

		for _, config := range configs {
			if strings.Contains(co.CurrentConfigPath, config) {
				red.Println(config)
			} else {
				fmt.Println(config)
			}
		}
	}
	CheckError(err, logger.Fatalf)
}

func CheckError(err error, loggerFunc func(format string, args ...interface{})) (wasError bool) {
	wasError = false

	if err != nil {
		wasError = true
		loggerFunc("%s\n", err)
	}
	return wasError
}
