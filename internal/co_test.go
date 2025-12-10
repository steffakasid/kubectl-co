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
	t.Run("Create new file", func(t *testing.T) {
		co := initCO(t)
		co.ConfigName = "newconfig"
		target := path.Join(co.CObasePath, co.ConfigName)

		err := co.AddConfig("")
		require.NoError(t, err)
		assert.FileExists(t, target)
	})

	t.Run("Copy from existing file", func(t *testing.T) {
		co := initCO(t)
		co.ConfigName = "copiedconfig"
		srcDir := t.TempDir()
		srcFile := path.Join(srcDir, "source.yml")
		content := []byte("kind: Config\n")
		err := os.WriteFile(srcFile, content, 0600)
		require.NoError(t, err)

		err = co.AddConfig(srcFile)
		require.NoError(t, err)

		target := path.Join(co.CObasePath, co.ConfigName)
		assert.FileExists(t, target)

		got, err := os.ReadFile(target)
		require.NoError(t, err)
		assert.Equal(t, content, got)
	})

	t.Run("NonExistingSource", func(t *testing.T) {
		co := initCO(t)
		co.ConfigName = "willfail"

		err := co.AddConfig("does-not-exist.yml")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no such file or directory")
	})

	t.Run("InvalidCObasePath", func(t *testing.T) {
		co := initCO(t)
		co.ConfigName = "badpath"
		co.CObasePath = path.Join(t.TempDir(), "non", "existent", "dir")

		err := co.AddConfig("")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no such file or directory")
	})
}

func TestLinkKubeConfig(t *testing.T) {
	tests := []struct {
		name         string
		setup        func(t *testing.T) *CO
		wantErr      bool
		wantLink     bool
		expectedLink string
	}{
		{
			name: "use previous",
			setup: func(t *testing.T) *CO {
				co := initCO(t)
				co.ConfigName = ""
				require.NotEmpty(t, co.PreviousConifgPath)
				return co
			},
			wantErr:      false,
			wantLink:     true,
			expectedLink: "", // validated from co.PreviousConifgPath inside subtest
		},
		{
			name: "no input",
			setup: func(t *testing.T) *CO {
				co := initCO(t)
				co.ConfigName = ""
				co.PreviousConifgPath = ""
				return co
			},
			wantErr:  true,
			wantLink: false,
		},
		{
			name: "target missing and kubeconfig absent",
			setup: func(t *testing.T) *CO {
				co := initCO(t)
				co.ConfigName = "not-existing-config"
				_ = os.Remove(co.KubeConfigPath)
				assert.NoFileExists(t, co.KubeConfigPath)
				return co
			},
			wantErr:  false,
			wantLink: false,
		},
		{
			name: "target exists",
			setup: func(t *testing.T) *CO {
				co := initCO(t)
				co.ConfigName = "existsconfig"
				target := path.Join(co.CObasePath, co.ConfigName)
				_, err := os.Create(target)
				require.NoError(t, err)
				return co
			},
			wantErr:  false,
			wantLink: true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			co := tc.setup(t)
			err := co.LinkKubeConfig()
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			if tc.wantLink {
				assert.FileExists(t, co.KubeConfigPath)
				linkTarget, err := os.Readlink(co.KubeConfigPath)
				require.NoError(t, err)
				if tc.name == "use previous" {
					assert.Equal(t, co.PreviousConifgPath, linkTarget)
				} else {
					expected := path.Join(co.CObasePath, co.ConfigName)
					assert.Equal(t, expected, linkTarget)
				}
			} else {
				assert.NoFileExists(t, co.KubeConfigPath)
			}
		})
	}
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
	t.Run("Non existing config", func(t *testing.T) {
		co := initCO(t)
		co.ConfigName = "does-not-exist"

		err := co.DeleteConfig()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "does not exist")
	})

	t.Run("Link error", func(t *testing.T) {
		co := initCO(t)
		co.ConfigName = "willlinkfail"
		target := path.Join(co.CObasePath, co.ConfigName)

		_, err := os.Create(target)
		require.NoError(t, err)

		// Ensure there's no previous config so after DeleteConfig clears ConfigName,
		// LinkKubeConfig will fail with "don't know what to do..."
		co.PreviousConifgPath = ""

		err = co.DeleteConfig()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to link kube config after deletion")
		// original config file should still exist on link failure
		assert.FileExists(t, target)
	})

	t.Run("Remove failure", func(t *testing.T) {
		co := initCO(t)
		co.ConfigName = "dirconfig"
		target := path.Join(co.CObasePath, co.ConfigName)

		// create a non-empty directory so os.Remove will fail
		require.NoError(t, os.Mkdir(target, 0700))
		_, err := os.Create(path.Join(target, "child"))
		require.NoError(t, err)

		err = co.DeleteConfig()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to delete config file")
		// directory should still exist after failed deletion
		assert.DirExists(t, target)
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
