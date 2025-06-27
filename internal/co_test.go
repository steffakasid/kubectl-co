package internal

import (
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func initCO(t *testing.T) (co *CO) {
	home := t.TempDir()

	kubeHome := path.Join(home, ".kube")
	coHome := path.Join(kubeHome, "co")
	previousFile := path.Join(coHome, "previousconfig")
	previousLink := path.Join(coHome, "previous")

	err := os.Mkdir(kubeHome, 0777)
	require.NoError(t, err)

	err = os.Mkdir(coHome, 0777)
	require.NoError(t, err)

	_, err = os.Create(previousFile)
	require.NoError(t, err)

	err = os.Symlink(previousFile, previousLink)
	require.NoError(t, err)

	co, err = NewCO(home)
	require.NoError(t, err)
	return co
}

func TestNewCO(t *testing.T) {
	t.Run("Init", func(t *testing.T) {
		home := t.TempDir()
		co, err := NewCO(home)
		require.NoError(t, err)
		assert.NotNil(t, co)
	})
}

func TestInitKubeHome(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		home := t.TempDir()
		kubeHome := path.Join(home, ".kube")
		err := initKubeHome(kubeHome)
		require.NoError(t, err)
	})
}

func TestInitCOHome(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		co := CO{
			CObasePath: path.Join(t.TempDir(), "co"),
		}
		err := co.initCOHome()
		require.NoError(t, err)
		assert.DirExists(t, co.CObasePath)
	})
	t.Run("Parent does not exist", func(t *testing.T) {
		co := CO{
			CObasePath: path.Join(t.TempDir(), "parent", "co"),
		}
		err := co.initCOHome()
		require.Error(t, err)
		require.Contains(t, err.Error(), "no such file or directory")
		assert.NoDirExists(t, co.CObasePath)
	})
}

func TestAddConfig(t *testing.T) {

	t.Run("Add new config", func(t *testing.T) {
		co := initCO(t)
		co.ConfigName = "testconfig"
		expectedConfigFile := path.Join(co.CObasePath, "testconfig")
		err := co.AddConfig("")
		require.NoError(t, err)
		assert.FileExists(t, expectedConfigFile)
	})

	t.Run("Add new config 2", func(t *testing.T) {
		co := initCO(t)
		co.ConfigName = "testconfig"
		expectedConfigFile := path.Join(co.CObasePath, "testconfig")
		err := co.AddConfig("../test/test.yml")
		require.NoError(t, err)
		assert.FileExists(t, expectedConfigFile)
	})

	t.Run("Add not existing config", func(t *testing.T) {
		co := initCO(t)
		co.ConfigName = "testconfig"
		err := co.AddConfig("not-existing.yml")
		require.Error(t, err)
		require.Contains(t, err.Error(), "no such file or directory")
	})

	t.Run("Add missconfigured co name", func(t *testing.T) {
		co := initCO(t)
		co.ConfigName = "testconfig"
		co.CObasePath = "../no-existing-path"
		err := co.AddConfig("../test/test.yml")
		require.Error(t, err)
		require.Contains(t, err.Error(), "no such file or directory")
	})
}

func TestLinkKubeConfig(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		co := initCO(t)

		co.ConfigName = "testconfig"
		configFile := path.Join(co.CObasePath, co.ConfigName)

		_, err := os.Create(configFile)
		require.NoError(t, err)

		err = co.LinkKubeConfig()

		require.NoError(t, err)
		assert.FileExists(t, co.KubeConfigPath)

		expectedLink, err := os.Readlink(co.KubeConfigPath)
		require.NoError(t, err)
		assert.Equal(t, configFile, expectedLink)
	})
}

func TestCleanup(t *testing.T) {
	t.Run("Cleanup only kube config", func(t *testing.T) {
		co := initCO(t)

		_, err := os.Create(co.KubeConfigPath)
		require.NoError(t, err)

		err = co.cleanup()
		require.NoError(t, err)
		assert.NoFileExists(t, co.KubeConfigPath)
	})
}

func TestDeleteConfig(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		co := initCO(t)
		co.ConfigName = "testconfig"
		configFile := path.Join(co.CObasePath, co.ConfigName)

		_, err := os.Create(configFile)
		require.NoError(t, err)

		err = co.DeleteConfig()
		require.NoError(t, err)
		assert.NoFileExists(t, configFile)
	})
}

func TestListConfigs(t *testing.T) {

	t.Run("Success", func(t *testing.T) {
		co := initCO(t)
		co.ConfigName = "testconfig"
		anotherConfig := "anotherone"
		configFile := path.Join(co.CObasePath, co.ConfigName)
		_, err := os.Create(configFile)
		require.NoError(t, err)

		anotherFile := path.Join(co.CObasePath, anotherConfig)
		_, err = os.Create(anotherFile)
		require.NoError(t, err)

		configs, err := co.ListConfigs()
		require.NoError(t, err)
		assert.Equal(t, []string{anotherConfig, "previousconfig", co.ConfigName}, configs)
	})

	t.Run("Error", func(t *testing.T) {
		co := initCO(t)
		co.CObasePath = "not-existing-path"

		configs, err := co.ListConfigs()
		require.Error(t, err)
		require.Len(t, configs, 0)
		require.Contains(t, err.Error(), "no such file or directory")
	})
}
