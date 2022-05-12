/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package main

import (
	"errors"
	"fmt"
	"io/fs"
	"io/ioutil"
	"os"
	"strings"

	"github.com/sirupsen/logrus"
	logger "github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
	"github.com/spf13/viper"
)

type config struct {
	Delete       bool `mapstructure:"delete"`
	Add          bool `mapstructure:"add"`
	List         bool `mapstructure:"list"`
	ConfigName   string
	ConfigPath   string
	PreviousPath string
	CurrentPath  string
}

var co *config = &config{}

var (
	home     string
	dotKube  string
	confDir  string
	prevLn   string
	kubeConf string
)

var version = "0.1-development"

const (
	viperKeyList    = "list"
	viperKeyDelete  = "delete"
	viperKeyAdd     = "add"
	viperKeyHelp    = "help"
	viperKeyVersion = "version"
)

func init() {
	var err error
	home, err = os.UserHomeDir()
	if err != nil {
		panic(err)
	}
	dotKube = fmt.Sprintf("%s/.kube", home)
	confDir = fmt.Sprintf("%s/co", dotKube)
	kubeConf = fmt.Sprintf("%s/config", dotKube)
	prevLn = fmt.Sprintf("%s/previous", confDir)

	if _, err = os.Stat(confDir); errors.Is(err, fs.ErrNotExist) {
		err := os.Mkdir(confDir, 0700)
		CheckError(err, logger.Fatalf)
	} else if err != nil {
		CheckError(err, logrus.Fatalf)
	}

	flag.BoolP(viperKeyDelete, "d", false, "Delete the config with the given name. Usage: kubectl co --delete [configname]")
	flag.BoolP(viperKeyAdd, "a", false, "Add a new given config providing the path and the name. Usage: kubectl co --add [configpath] [configname]")
	flag.BoolP(viperKeyList, "l", false, "List all available config files")

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
  '

Flags:`)

		flag.PrintDefaults()
	}

	flag.Parse()
	err = viper.BindPFlags(flag.CommandLine)
	CheckError(err, logger.Fatalf)
	err = viper.Unmarshal(co)
	CheckError(err, logger.Fatalf)
	logger.SetLevel(logger.DebugLevel)
}

func main() {
	if viper.GetBool(viperKeyVersion) {
		fmt.Printf("kubectl-co version: %s\n", version)
	} else if viper.GetBool(viperKeyHelp) {
		flag.Usage()
	} else {
		parseFlags(os.Args)
	}
}

func CheckError(err error, loggerFunc func(format string, args ...interface{})) (wasError bool) {
	wasError = false

	if err != nil {
		wasError = true
		loggerFunc("%s\n", err)
	}
	return wasError
}

func parseFlags(args []string) {
	logger.Debug("args", args)
	logger.Debug("config", co)
	var clArgs []string = []string{}
	var err error

	for _, arg := range args {
		if !strings.HasPrefix(arg, "-") && !strings.Contains(arg, "kubectl-co") {
			clArgs = append(clArgs, arg)
		}
	}

	co.PreviousPath, err = os.Readlink(prevLn)
	CheckError(err, logger.Debugf)

	co.CurrentPath, err = os.Readlink(kubeConf)
	CheckError(err, logger.Debugf)

	if co.Delete && co.Add {
		panic(fmt.Sprintf("%s and %s can't be used togeether!", viperKeyAdd, viperKeyDelete))
	} else if co.Delete && len(clArgs) != 1 {
		panic(fmt.Sprintf("When using %s you must only provide the name of the config to be deleted!", viperKeyDelete))
	} else if co.Delete && len(clArgs) == 1 {
		co.ConfigName = clArgs[0]
		deleteKubeConfig()
	} else if co.Add && (len(clArgs) == 0 || len(clArgs) > 2) {
		panic(fmt.Sprintf("When using %s you must provide the path as first argument and the name of the config as second argument!", viperKeyAdd))
	} else if co.Add && len(clArgs) == 2 {
		co.ConfigPath = clArgs[0]
		co.ConfigName = clArgs[1]
		addConfig()
	} else if co.Add && len(clArgs) == 1 {
		co.ConfigName = clArgs[0]
		addConfig()
	} else if co.List {
		listConfigs()
	} else {
		if len(clArgs) == 1 {
			co.ConfigName = clArgs[0]
		}
		linkKubeConfig()
	}
}

func addConfig() {
	configToWrite := fmt.Sprintf("%s/.kube/co/%s", home, co.ConfigName)

	if co.ConfigPath == "" {
		os.Create(configToWrite)
		fmt.Printf("Created new config file %s. You may need to initalize it.", configToWrite)
	} else {
		input, err := ioutil.ReadFile(co.ConfigPath)
		CheckError(err, logrus.Fatalf)

		err = ioutil.WriteFile(configToWrite, input, 0600)
		CheckError(err, logrus.Fatalf)
		fmt.Println("Added", configToWrite)
	}
}

func linkKubeConfig() {
	var configToUse string

	err := os.Remove(kubeConf)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		CheckError(err, logrus.Fatalf)
	}

	err = os.Remove(prevLn)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		CheckError(err, logrus.Fatalf)
	}

	if co.ConfigName != "" {
		configToUse = fmt.Sprintf("%s/%s", confDir, co.ConfigName)
	} else if co.PreviousPath != "" {
		configToUse = co.PreviousPath
	} else {
		panic("Don't know what to do. Need a configname to configure")
	}

	err = os.Symlink(configToUse, kubeConf)
	CheckError(err, logrus.Fatalf)

	err = os.Symlink(co.CurrentPath, prevLn)
	CheckError(err, logrus.Fatalf)

	fmt.Printf("Linked %s to %s\n", kubeConf, configToUse)
	logger.Debugf("Linked %s to %s", prevLn, co.CurrentPath)
}

func deleteKubeConfig() {
	configToUse := fmt.Sprintf("%s/%s", confDir, co.ConfigName)
	if _, err := os.Stat(configToUse); err != nil {
		panic(err)
	}
	fmt.Println("Deleted", configToUse)
}

func listConfigs() {
	entries, err := os.ReadDir(confDir)
	CheckError(err, logger.Fatalf)
	for _, entry := range entries {
		fmt.Println(entry.Name())
	}
}
