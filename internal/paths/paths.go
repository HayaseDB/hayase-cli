package paths

import "github.com/adrg/xdg"

const appName = "hayase"

func ConfigFile(name string) (string, error) {
	return xdg.ConfigFile(appName + "/" + name)
}

func DataFile(name string) (string, error) {
	return xdg.DataFile(appName + "/" + name)
}

func CacheFile(name string) (string, error) {
	return xdg.CacheFile(appName + "/" + name)
}
