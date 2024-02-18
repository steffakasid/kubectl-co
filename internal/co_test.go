package internal

import (
	"os"
	"path"
	"testing"

	extendedslog "github.com/steffakasid/extended-slog"
	"github.com/stretchr/testify/assert"
)

func init() {
	extendedslog.InitLogger()
}

func initCO(t *testing.T) (co *CO, home string, kubeHome string, coHome string, previousFile string, previousLink string) {
	home = t.TempDir()

	kubeHome = path.Join(home, ".kube")
	coHome = path.Join(kubeHome, "co")
	previousFile = path.Join(coHome, "previousconfig")
	previousLink = path.Join(coHome, "previous")

	err := os.Mkdir(kubeHome, 0777)
	assert.NoError(t, err)

	err = os.Mkdir(coHome, 0777)
	assert.NoError(t, err)

	_, err = os.Create(previousFile)
	assert.NoError(t, err)

	err = os.Symlink(previousFile, previousLink)
	assert.NoError(t, err)

	co, err = NewCO(home)
	assert.NoError(t, err)
	return co, home, kubeHome, coHome, previousFile, previousLink
}

func TestNewCO(t *testing.T) {
	t.Run("Init", func(t *testing.T) {
		home := t.TempDir()
		co, err := NewCO(home)
		assert.NoError(t, err)
		assert.NotNil(t, co)
	})
}

func TestAddConfig(t *testing.T) {

	t.Run("Add new config", func(t *testing.T) {
		co, _, _, coHome, _, _ := initCO(t)
		co.ConfigName = "testconfig"
		expectedConfigFile := path.Join(coHome, "testconfig")
		err := co.AddConfig("")
		assert.NoError(t, err)
		assert.FileExists(t, expectedConfigFile)
	})

	t.Run("Add new config", func(t *testing.T) {
		co, _, _, coHome, _, _ := initCO(t)
		co.ConfigName = "testconfig"
		expectedConfigFile := path.Join(coHome, "testconfig")
		err := co.AddConfig("../test/test.yml")
		assert.NoError(t, err)
		assert.FileExists(t, expectedConfigFile)
	})
}

func TestLinkKubeConfig(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		co, _, kubeHome, coHome, _, _ := initCO(t)

		co.ConfigName = "testconfig"
		configFile := path.Join(coHome, co.ConfigName)
		expectedFile := path.Join(kubeHome, "config")
		os.Create(configFile)

		err := co.LinkKubeConfig()

		assert.NoError(t, err)
		assert.FileExists(t, expectedFile)

		expectedLink, err := os.Readlink(expectedFile)
		assert.NoError(t, err)
		assert.Equal(t, configFile, expectedLink)
	})
}

func TestCleanup(t *testing.T) {
	t.Run("Cleanup only kube config", func(t *testing.T) {
		co, _, kubeHome, _, _, _ := initCO(t)

		kubeConfig := path.Join(kubeHome, "config")
		os.Create(kubeConfig)

		err := co.cleanup()
		assert.NoError(t, err)
		assert.NoFileExists(t, kubeConfig)
	})
}

func TestDeleteConfig(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		co, _, _, coHome, _, _ := initCO(t)
		co.ConfigName = "testconfig"
		configFile := path.Join(coHome, co.ConfigName)

		_, err := os.Create(configFile)
		assert.NoError(t, err)

		err = co.DeleteConfig()
		assert.NoError(t, err)
		assert.NoFileExists(t, configFile)
	})
}

func TestListConfigs(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		co, _, _, coHome, _, _ := initCO(t)
		co.ConfigName = "testconfig"
		anotherConfig := "anotherone"
		configFile := path.Join(coHome, co.ConfigName)
		os.Create(configFile)
		anotherFile := path.Join(coHome, anotherConfig)
		os.Create(anotherFile)

		configs, err := co.ListConfigs()
		assert.NoError(t, err)
		assert.Equal(t, []string{anotherConfig, "previousconfig", co.ConfigName}, configs)
	})

}
