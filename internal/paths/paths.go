package paths

import (
	"path/filepath"

	"github.com/adrg/xdg"
)

const appName = "hayase"

func ConfigFile(name string) (string, error) {
	return xdg.ConfigFile(filepath.Join(appName, name))
}

func DataFile(name string) (string, error) {
	return xdg.DataFile(filepath.Join(appName, name))
}

func CacheFile(name string) (string, error) {
	return xdg.CacheFile(filepath.Join(appName, name))
}

func CacheDir() string {
	return filepath.Join(xdg.CacheHome, appName)
}

func DataDir() string {
	return filepath.Join(xdg.DataHome, appName)
}

func ConfigDir() string {
	return filepath.Join(xdg.ConfigHome, appName)
}
