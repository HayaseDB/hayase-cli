package storage

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/spf13/viper"

	"github.com/hayasedb/hayase-cli/internal/models"
)

type Config struct {
	v *viper.Viper
}

func NewConfig() (*Config, error) {
	v := viper.New()

	setDefaults(v)

	configDir, err := getConfigDir()
	if err != nil {
		return nil, err
	}

	v.AddConfigPath(configDir)
	v.SetConfigName("config")
	v.SetConfigType("yaml")

	v.SetEnvPrefix("HAYASE")
	v.AutomaticEnv()

	if err := os.MkdirAll(configDir, 0755); err != nil {
		return nil, err
	}

	config := &Config{v: v}

	if err := config.Load(); err != nil {
		var configFileNotFoundError viper.ConfigFileNotFoundError
		if errors.As(err, &configFileNotFoundError) {
			_ = config.Save()
		}
	}

	return config, nil
}

func setDefaults(v *viper.Viper) {
	v.SetDefault("provider", "aniworld")
	v.SetDefault("language", "ger-sub")
	v.SetDefault("quality", "1080p")

	v.SetDefault("player", "mpv")

	v.SetDefault("instantSearch", true)

	v.SetDefault("timeout", 10)
}

func getConfigDir() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "hayase-cli"), nil
}

func (c *Config) Load() error {
	return c.v.ReadInConfig()
}

func (c *Config) Save() error {
	if err := c.v.WriteConfig(); err != nil {
		var configFileNotFoundError viper.ConfigFileNotFoundError
		if errors.As(err, &configFileNotFoundError) {
			return c.v.SafeWriteConfig()
		}
		return err
	}
	return nil
}

func (c *Config) GetString(key string) string {
	return c.v.GetString(key)
}

func (c *Config) GetInt(key string) int {
	return c.v.GetInt(key)
}

func (c *Config) GetBool(key string) bool {
	return c.v.GetBool(key)
}

func (c *Config) Set(key string, value interface{}) {
	c.v.Set(key, value)
}

func (c *Config) GetProvider() string {
	return c.GetString("provider")
}

func (c *Config) GetPlayer() string {
	return c.GetString("player")
}

func (c *Config) GetLanguage() models.Language {
	lang := c.GetString("language")
	if parsed, err := models.ParseLanguage(lang); err == nil {
		return parsed
	}
	return models.GerSub
}

func (c *Config) GetQuality() models.Quality {
	quality := c.GetString("quality")
	switch quality {
	case "720p":
		return models.Quality720p
	case "1080p":
		return models.Quality1080p
	case "1440p":
		return models.Quality1440p
	case "2160p":
		return models.Quality2160p
	default:
		return models.Quality1080p
	}
}

func (c *Config) GetTimeout() int {
	return c.GetInt("timeout")
}

func (c *Config) GetInstantSearch() bool {
	return c.GetBool("instantSearch")
}
