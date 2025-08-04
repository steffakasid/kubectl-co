package internal

import (
	"errors"
	"fmt"
	"io/fs"
	"os"

	"github.com/steffakasid/eslog"
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
	var co = &CO{}

	kubeHome := fmt.Sprintf("%s/%s", home, dotKube)
	if err := initKubeHome(kubeHome); err != nil {
		return nil, fmt.Errorf("failed to initialize kube home: %w", err)
	}

	co.CObasePath = fmt.Sprintf("%s/%s", kubeHome, COfolderName)
	co.KubeConfigPath = fmt.Sprintf("%s/%s/config", home, dotKube)
	co.PreviousConfigLink = fmt.Sprintf("%s/previous", co.CObasePath)

	if err := co.initCOHome(); err != nil {
		return nil, fmt.Errorf("failed to initialize CO home: %w", err)
	}

	co.PreviousConifgPath, err = os.Readlink(co.PreviousConfigLink)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("failed to read previous config link: %w", err)
	}

	fi, err := os.Lstat(co.KubeConfigPath)
	if err == nil && fi.Mode()&os.ModeSymlink != 0 {
		co.CurrentConfigPath, err = os.Readlink(co.KubeConfigPath)
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			return nil, fmt.Errorf("failed to read current config path: %w", err)
		}
	}

	return co, nil
}

func initKubeHome(kubeHome string) error {
	if _, err := os.Stat(kubeHome); errors.Is(err, fs.ErrNotExist) {
		err := os.Mkdir(kubeHome, onlyOwnerAccess)
		if err != nil {
			return fmt.Errorf("failed to create kube home directory: %w", err)
		}
	}
	return nil
}

func (co CO) initCOHome() error {
	if _, err := os.Stat(co.CObasePath); errors.Is(err, fs.ErrNotExist) {
		err := os.Mkdir(co.CObasePath, onlyOwnerAccess)
		if err != nil {
			return fmt.Errorf("failed to create CO home directory: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("error CO home directory already exists: %w", err)
	}
	return nil
}

func (co CO) AddConfig(newConfigPath string) error {
	configToWrite := fmt.Sprintf("%s/%s", co.CObasePath, co.ConfigName)

	if newConfigPath == "" {
		_, err := os.Create(configToWrite)
		if err != nil {
			return fmt.Errorf("failed to create new config file: %w", err)
		}
		err = os.Chmod(configToWrite, onlyOwnerAccess)
		if err != nil {
			return fmt.Errorf("failed to set permissions on new config file: %w", err)
		}
		eslog.Infof("Created new config file %s. You may need to initalize it.", configToWrite)
	} else {
		input, err := os.ReadFile(newConfigPath)
		if err != nil {
			return fmt.Errorf("failed to read input config file: %w", err)
		}

		err = os.WriteFile(configToWrite, input, onlyOwnerAccess)
		if err != nil {
			return fmt.Errorf("failed to write config file: %w", err)
		}
		eslog.Infof("Added %s", configToWrite)
	}
	return nil
}

func (co CO) LinkKubeConfig() error {
	var configToUse string

	if co.ConfigName != "" {
		configToUse = fmt.Sprintf("%s/%s", co.CObasePath, co.ConfigName)
	} else if co.PreviousConifgPath != "" {
		configToUse = co.PreviousConifgPath
	} else {
		return errors.New("don't know what to do. Need a configname to configure")
	}

	if _, err := os.Stat(configToUse); errors.Is(err, fs.ErrNotExist) {
		return nil
	}

	if err := co.cleanup(); err != nil {
		return fmt.Errorf("failed to cleanup previous kube config: %w", err)
	}

	if err := co.linkConfigToUse(configToUse); err != nil {
		return fmt.Errorf("failed to link kube config: %w", err)
	}

	return co.linkPreviousConfig()
}

func (co CO) linkConfigToUse(configToUse string) error {
	if _, err := os.Stat(configToUse); errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("config file %s does not exist", configToUse)
	}

	if err := os.Symlink(configToUse, co.KubeConfigPath); err != nil {
		return fmt.Errorf("failed to create symlink: %w", err)
	}
	fmt.Printf("Linked %s to %s\n", co.KubeConfigPath, configToUse)
	// chmod on symlink to avoid kubectl warnings.
	if err := os.Chmod(co.KubeConfigPath, onlyOwnerAccess); err != nil {
		return fmt.Errorf("failed to set permissions on kube config symlink: %w", err)
	}
	return nil
}

func (co CO) linkPreviousConfig() error {
	if co.CurrentConfigPath != "" {
		if err := os.Symlink(co.CurrentConfigPath, co.PreviousConfigLink); err != nil {
			return fmt.Errorf("failed to create symlink for previous config: %w", err)
		}
		eslog.Debugf("Linked %s to %s", co.PreviousConfigLink, co.CurrentConfigPath)
	}
	return nil
}

func (co CO) cleanup() error {
	err := os.Remove(co.KubeConfigPath)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("failed to remove kube config symlink: %w", err)
	}

	err = os.Remove(co.PreviousConfigLink)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("failed to remove previous config symlink: %w", err)
	}
	return nil
}

func (co CO) DeleteConfig() error {
	configToUse := fmt.Sprintf("%s/%s", co.CObasePath, co.ConfigName)
	if _, err := os.Stat(configToUse); err != nil {
		return fmt.Errorf("config file %s does not exist: %w", configToUse, err)
	}
	co.ConfigName = ""
	err := co.LinkKubeConfig()
	if err != nil {
		return fmt.Errorf("failed to link kube config after deletion: %w", err)
	}

	err = os.Remove(configToUse)
	if err != nil {
		return fmt.Errorf("failed to delete config file %s: %w", configToUse, err)
	}
	fmt.Println("Deleted", configToUse)
	return nil
}

func (co CO) ListConfigs() ([]string, error) {
	entries, err := os.ReadDir(co.CObasePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config directory %s: %w", co.CObasePath, err)
	}
	configs := []string{}
	for _, entry := range entries {
		if entry.Name() != "previous" {
			configs = append(configs, entry.Name())
		}
	}
	return configs, nil
}
