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
	Configs            []string
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

// initKubeHome creates the kube home directory if it does not exist.
// The directory is created with permissions that only allow owner access (0700).
// Returns an error if the directory creation fails.
func initKubeHome(kubeHome string) error {
	if _, err := os.Stat(kubeHome); errors.Is(err, fs.ErrNotExist) {
		err := os.Mkdir(kubeHome, onlyOwnerAccess)
		if err != nil {
			return fmt.Errorf("failed to create kube home directory: %w", err)
		}
	}
	return nil
}

// initCOHome initializes the CO home directory by creating it if it does not exist.
// It returns an error if the directory creation fails or if an unexpected error occurs
// when checking for the directory's existence.
func (co *CO) initCOHome() error {
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

// AddConfig creates or copies a kubeconfig file to the CO base path.
// If newConfigPath is empty, it creates a new empty config file with owner-only access permissions.
// If newConfigPath is provided, it reads the config from that path and writes it to the CO base path.
// The created or copied config file will be named according to co.ConfigName.
// Returns an error if file operations fail.
func (co *CO) AddConfig(newConfigPath string) error {
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

// LinkKubeConfig creates a symbolic link to the specified Kubernetes configuration file.
// It determines which configuration to use based on the following priority:
//  1. co.ConfigName - if provided, uses the config from co.CObasePath
//  2. co.PreviousConifgPath - if ConfigName is empty, falls back to the previous config path
//
// If the configuration file to use doesn't exist, the function returns nil without error.
// Otherwise, it performs the following steps:
//  1. Cleans up any previous Kubernetes configuration
//  2. Links the selected configuration file
//  3. Creates a link to the previous configuration for rollback purposes
//
// Returns an error if:
//   - Neither ConfigName nor PreviousConifgPath is set
//   - Cleanup of previous configuration fails
//   - Linking the configuration fails
//   - Linking the previous configuration fails
func (co *CO) LinkKubeConfig() error {
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

// linkConfigToUse creates a symbolic link from co.KubeConfigPath to the specified configToUse file.
// It first verifies that configToUse exists, then creates the symlink and sets its permissions to
// onlyOwnerAccess to avoid kubectl warnings. Returns an error if the config file doesn't exist,
// if symlink creation fails, or if setting permissions fails.
func (co *CO) linkConfigToUse(configToUse string) error {
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

// linkPreviousConfig creates a symbolic link to the current config file at the previous config link path.
// If CurrentConfigPath is set, it creates a symlink from PreviousConfigLink pointing to CurrentConfigPath.
// Returns an error if the symlink creation fails.
func (co *CO) linkPreviousConfig() error {
	if co.CurrentConfigPath != "" {
		if err := os.Symlink(co.CurrentConfigPath, co.PreviousConfigLink); err != nil {
			return fmt.Errorf("failed to create symlink for previous config: %w", err)
		}
		eslog.Debugf("Linked %s to %s", co.PreviousConfigLink, co.CurrentConfigPath)
	}
	return nil
}

// cleanup removes the kube config symlink and previous config symlink.
// It returns an error if either removal fails, unless the file does not exist.
func (co *CO) cleanup() error {
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

// DeleteConfig removes the configuration file associated with the CO instance.
// It first verifies that the config file exists, then clears the ConfigName,
// relinks the kubeconfig to remove the deleted config, and finally deletes the file.
// Returns an error if the config file does not exist, if relinking the kubeconfig fails,
// or if the file deletion fails.
func (co *CO) DeleteConfig() error {
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

// ListConfigs reads the CO base directory and populates the Configs field with
// all directory entries except "previous". It returns an error if the directory
// cannot be read.
func (co *CO) ListConfigs() error {
	entries, err := os.ReadDir(co.CObasePath)
	if err != nil {
		return fmt.Errorf("failed to read config directory %s: %w", co.CObasePath, err)
	}
	configs := []string{}
	for _, entry := range entries {
		if entry.Name() != "previous" {
			configs = append(configs, entry.Name())
		}
	}
	co.Configs = configs
	return nil
}
