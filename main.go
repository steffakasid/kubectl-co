/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package main

import (
	"fmt"
	"os"
	"strings"

	logger "github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/steffakasid/kubectl-co/internal"
)

type cmdCfg struct {
	Delete bool `mapstructure:"delete"`
	Debug  bool `mapstructure:"debug"`
	Add    bool `mapstructure:"add"`
	List   bool `mapstructure:"list"`
}

var c *cmdCfg = &cmdCfg{}
var co *internal.CO

var version = "0.1-development"

const (
	viperKeyList    = "list"
	viperKeyDebug   = "debug"
	viperKeyDelete  = "delete"
	viperKeyAdd     = "add"
	viperKeyHelp    = "help"
	viperKeyVersion = "version"
)

func init() {
	var err error
	logger.SetLevel(logger.InfoLevel)

	flag.BoolP(viperKeyDelete, "d", false, "Delete the config with the given name. Usage: kubectl co --delete [configname]")
	flag.BoolP(viperKeyAdd, "a", false, "Add a new given config providing the path and the name. Usage: kubectl co --add [configpath] [configname]")
	flag.BoolP(viperKeyList, "l", false, "List all available config files")
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
	} else {
		var err error
		var configs []string

		clArgs, err := parseFlags(os.Args)
		CheckError(err, logger.Fatalf)

		if len(clArgs) > 0 {
			co.ConfigName = clArgs[0]
		}

		if c.Add {
			copyConfigFrom := ""
			if len(clArgs) == 2 {
				copyConfigFrom = clArgs[1]
			}
			err = co.AddConfig(copyConfigFrom)
		} else if c.Delete {
			err = co.DeleteConfig()
		} else if c.List {
			configs, err = co.ListConfigs()
			for _, config := range configs {
				fmt.Println(config)
			}
		} else {
			err = co.LinkKubeConfig()
		}
		CheckError(err, logger.Fatalf)
	}
}

func parseFlags(args []string) ([]string, error) {
	logger.Debug("args", args)
	logger.Debug("config", c)

	var clArgs []string = []string{}
	for _, arg := range args {
		if !strings.HasPrefix(arg, "-") && !strings.Contains(arg, "kubectl-co") {
			clArgs = append(clArgs, arg)
		}
	}

	if (c.Delete && c.Add) || (c.Delete && c.List) || (c.Add && c.List) {
		return nil, fmt.Errorf("%s, %s and %s are exklusiv just use one at a time", viperKeyAdd, viperKeyDelete, viperKeyList)
	} else if c.Delete && len(clArgs) != 1 {
		return nil, fmt.Errorf("When using %s you must only provide the name of the config to be deleted!", viperKeyDelete)
	} else if c.Add && (len(clArgs) == 0 || len(clArgs) > 2) {
		return nil, fmt.Errorf("When using %s you must provide the path as first argument and the name of the config as second argument!", viperKeyAdd)
	} else if c.List && len(clArgs) != 0 {
		return nil, fmt.Errorf("%s doesn't take any arguments", viperKeyList)
	}
	return clArgs, nil
}

func CheckError(err error, loggerFunc func(format string, args ...interface{})) (wasError bool) {
	wasError = false

	if err != nil {
		wasError = true
		loggerFunc("%s\n", err)
	}
	return wasError
}
