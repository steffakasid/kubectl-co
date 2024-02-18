package internal

import (
	"errors"
	"fmt"
	"io/fs"
	"os"

	extendedslog "github.com/steffakasid/extended-slog"
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

const onlyOwnerAccess = 0700

func NewCO(home string) (*CO, error) {
	var err error
	var co *CO = &CO{}

	kubeHome := fmt.Sprintf("%s/%s", home, dotKube)

	co.CObasePath = fmt.Sprintf("%s/%s", kubeHome, COfolderName)
	co.KubeConfigPath = fmt.Sprintf("%s/%s/config", home, dotKube)
	co.PreviousConfigLink = fmt.Sprintf("%s/previous", co.CObasePath)

	if _, err = os.Stat(kubeHome); errors.Is(err, fs.ErrNotExist) {
		err := os.Mkdir(kubeHome, onlyOwnerAccess)
		if err != nil {
			return nil, err
		}
	}

	if _, err = os.Stat(co.CObasePath); errors.Is(err, fs.ErrNotExist) {
		err := os.Mkdir(co.CObasePath, onlyOwnerAccess)
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
		err = os.Chmod(configToWrite, onlyOwnerAccess)
		if err != nil {
			return err
		}
		extendedslog.Logger.Infof("Created new config file %s. You may need to initalize it.", configToWrite)
	} else {
		input, err := os.ReadFile(newConfigPath)
		if err != nil {
			return err
		}

		err = os.WriteFile(configToWrite, input, onlyOwnerAccess)
		if err != nil {
			return err
		}
		extendedslog.Logger.Infof("Added %s", configToWrite)
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
	// chmod on symlink to avoid kubectl warnings.
	if err := os.Chmod(co.KubeConfigPath, onlyOwnerAccess); err != nil {
		return err
	}

	if co.CurrentConfigPath != "" {
		if err := os.Symlink(co.CurrentConfigPath, co.PreviousConfigLink); err != nil {
			return err
		}
		extendedslog.Logger.Debugf("Linked %s to %s", co.PreviousConfigLink, co.CurrentConfigPath)
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
