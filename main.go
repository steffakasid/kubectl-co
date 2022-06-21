/*
Copyright © 2022 steffakasid
*/
package main

import (
	"fmt"
	"os"

	logger "github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/steffakasid/kubectl-co/internal"
)

type cmdCfg struct {
	Delete  bool `mapstructure:"delete"`
	Debug   bool `mapstructure:"debug"`
	Add     bool `mapstructure:"add"`
	List    bool `mapstructure:"list"`
	Current bool `mapstructure:"current"`
}

var c *cmdCfg = &cmdCfg{}
var co *internal.CO

var version = "0.1-development"

const (
	viperKeyList    = "list"
	viperKeyDebug   = "debug"
	viperKeyDelete  = "delete"
	viperKeyAdd     = "add"
	viperKeyCurrent = "current"
	viperKeyHelp    = "help"
	viperKeyVersion = "version"
)

func init() {
	var err error
	logger.SetLevel(logger.InfoLevel)

	flag.BoolP(viperKeyDelete, "d", false, "Delete the config with the given name. Usage: kubectl co --delete [configname]")
	flag.BoolP(viperKeyAdd, "a", false, "Add a new given config providing the path and the name. Usage: kubectl co --add [configpath] [configname]")
	flag.BoolP(viperKeyList, "l", false, "List all available config files")
	flag.BoolP(viperKeyCurrent, "c", false, "Show current config file")
	flag.BoolP(viperKeyHelp, "h", false, "Show help")
	flag.Bool(viperKeyDebug, false, "Turn on debug output")

	flag.Usage = func() {
		w := os.Stderr

		fmt.Fprintf(w, "Usage of %s: \n", os.Args[0])
		fmt.Fprintln(w, `
This tool can be used to work with multiple kube configs. It allows to
add, delete and switch config files.

NOTE: If you set the KUBECONFIG environment var this will always take precedence before the config file.

Usage:
  kubectl co [flags]
  kubectl-co [flags]

Preqrequisites:
  kubectl should be installed (even if the application would also run for it own as 'kubectl-co')

Examples:
  kubectl co --add ~/.kube/config new-config    - adds your current kubeconfig to be used by co with the name 'new-config'
  kubectl co --add completly-new                - adds a plain new config file which must be inialised afterwards
  kubectl co --list                             - list all available configs
  kubectl co --delete new-config                - delete config with name 'new-config'
  kubectl co new-config                         - switch to 'new-config' this will overwrite ~/.kube/config with a symbolic link
  kubectl co                                    - switch to previous config and set current config to previous

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
		var configs []string

		args := flag.Args()
		err := parseFlags(args)

		CheckError(err, logger.Fatalf)

		if len(args) > 0 {
			co.ConfigName = args[0]
		}

		if c.Add {
			copyConfigFrom := ""
			if len(args) == 2 {
				copyConfigFrom = args[1]
			}
			err = co.AddConfig(copyConfigFrom)
		} else if c.Delete {
			err = co.DeleteConfig()
		} else if c.List {
			configs, err = co.ListConfigs()
			for _, config := range configs {
				fmt.Println(config)
			}
		} else if c.Current {
			fmt.Println("Current config file:", co.CurrentConfigPath)
		} else {
			err = co.LinkKubeConfig()
		}
		CheckError(err, logger.Fatalf)
	}
}

func parseFlags(args []string) error {
	logger.Debug("config", c)

	if (c.Delete && c.Add) || (c.Delete && c.List) || (c.Add && c.List) {
		return fmt.Errorf("%s, %s and %s are exklusiv just use one at a time", viperKeyAdd, viperKeyDelete, viperKeyList)
	} else if c.Delete && len(args) != 1 {
		return fmt.Errorf("When using %s you must only provide the name of the config to be deleted!", viperKeyDelete)
	} else if c.Add && (len(args) == 0 || len(args) > 2) {
		return fmt.Errorf("When using %s you must provide the path as first argument and the name of the config as second argument!", viperKeyAdd)
	} else if c.List && len(args) != 0 {
		return fmt.Errorf("%s doesn't take any arguments", viperKeyList)
	}
	return nil
}

func CheckError(err error, loggerFunc func(format string, args ...interface{})) (wasError bool) {
	wasError = false

	if err != nil {
		wasError = true
		loggerFunc("%s\n", err)
	}
	return wasError
}
