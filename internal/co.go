package internal

import (
	"errors"
	"fmt"
	"io/fs"
	"io/ioutil"
	"os"

	logger "github.com/sirupsen/logrus"
)

const (
	COfolderName = "co"
	dotKube      = ".kube"
)

func init() {}

type CO struct {
	ConfigName         string
	CObasePath         string
	KubeConfigPath     string
	PreviousConifgPath string
	PreviousConfigLink string
	CurrentConfigPath  string
}

func NewCO(home string) (*CO, error) {
	var err error
	var co *CO = &CO{}

	kubeHome := fmt.Sprintf("%s/%s", home, dotKube)

	co.CObasePath = fmt.Sprintf("%s/%s", kubeHome, COfolderName)
	co.KubeConfigPath = fmt.Sprintf("%s/%s/config", home, dotKube)
	co.PreviousConfigLink = fmt.Sprintf("%s/previous", co.CObasePath)

	if _, err = os.Stat(kubeHome); errors.Is(err, fs.ErrNotExist) {
		err := os.Mkdir(kubeHome, 0700)
		if err != nil {
			return nil, err
		}
	}

	if _, err = os.Stat(co.CObasePath); errors.Is(err, fs.ErrNotExist) {
		err := os.Mkdir(co.CObasePath, 0700)
		if err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}

	co.PreviousConifgPath, err = os.Readlink(co.PreviousConfigLink)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}

	fi, err := os.Lstat(co.KubeConfigPath)
	if err == nil && fi.Mode()&os.ModeSymlink != 0 {
		co.CurrentConfigPath, err = os.Readlink(co.KubeConfigPath)
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			return nil, err
		}
	}

	return co, nil
}

func (co CO) AddConfig(newConfigPath string) error {
	configToWrite := fmt.Sprintf("%s/%s", co.CObasePath, co.ConfigName)

	if newConfigPath == "" {
		_, err := os.Create(configToWrite)
		if err != nil {
			return err
		}
		err = os.Chmod(configToWrite, 0600)
		if err != nil {
			return err
		}
		fmt.Printf("Created new config file %s. You may need to initalize it.", configToWrite)
	} else {
		input, err := ioutil.ReadFile(newConfigPath)
		if err != nil {
			return err
		}

		err = ioutil.WriteFile(configToWrite, input, 0600)
		if err != nil {
			return err
		}
		fmt.Println("Added", configToWrite)
	}
	return nil
}

func (co CO) LinkKubeConfig() error {
	var configToUse string

	if err := co.cleanup(); err != nil {
		return err
	}

	if co.ConfigName != "" {
		configToUse = fmt.Sprintf("%s/%s", co.CObasePath, co.ConfigName)
	} else if co.PreviousConifgPath != "" {
		configToUse = co.PreviousConifgPath
	} else {
		return errors.New("don't know what to do. Need a configname to configure.")
	}

	if _, err := os.Stat(configToUse); errors.Is(err, fs.ErrNotExist) {
		return err
	}

	if err := os.Symlink(configToUse, co.KubeConfigPath); err != nil {
		return err
	}
	fmt.Printf("Linked %s to %s\n", co.KubeConfigPath, configToUse)

	if co.CurrentConfigPath != "" {
		if err := os.Symlink(co.CurrentConfigPath, co.PreviousConfigLink); err != nil {
			return err
		}
		logger.Debugf("Linked %s to %s", co.PreviousConfigLink, co.CurrentConfigPath)
	}

	return nil
}

func (co CO) cleanup() error {
	err := os.Remove(co.KubeConfigPath)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}

	err = os.Remove(co.PreviousConfigLink)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return nil
}

func (co CO) DeleteConfig() error {
	configToUse := fmt.Sprintf("%s/%s", co.CObasePath, co.ConfigName)
	if _, err := os.Stat(configToUse); err != nil {
		return err
	}
	co.ConfigName = ""
	err := co.LinkKubeConfig()
	if err != nil {
		return err
	}

	err = os.Remove(configToUse)
	if err != nil {
		return err
	}
	fmt.Println("Deleted", configToUse)
	return nil
}

func (co CO) ListConfigs() ([]string, error) {
	entries, err := os.ReadDir(co.CObasePath)
	if err != nil {
		return nil, err
	}
	configs := []string{}
	for _, entry := range entries {
		if entry.Name() != "previous" {
			configs = append(configs, entry.Name())
		}
	}
	return configs, nil
}
