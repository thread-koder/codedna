package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/spf13/viper"
)

const (
	// ConfigPathEnv is the environment variable name for the config file path
	ConfigPathEnv = "CODEDNA_CONFIG"
)

var (
	cfg    *Config
	cfgErr error
	once   sync.Once
)

type Config struct {
	Log struct {
		Global struct {
			Level  string `mapstructure:"level"`
			Format string `mapstructure:"format"`
			Output string `mapstructure:"output"`
			File   string `mapstructure:"file"`
		} `mapstructure:"global"`
	} `mapstructure:"log"`
}

func setDefaults(v *viper.Viper) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}

	// Global logging defaults
	v.SetDefault("log.global.level", "info")
	v.SetDefault("log.global.format", "console")
	v.SetDefault("log.global.output", "stdout")
	v.SetDefault("log.global.file", filepath.Join(homeDir, ".codedna", "logs", "codedna.log"))
}

func Load() (*Config, error) {
	once.Do(func() {
		v := viper.New()

		// Configure environment variable support
		v.SetEnvPrefix("CODEDNA")
		v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
		v.AutomaticEnv()

		setDefaults(v)

		if configPath := v.GetString(ConfigPathEnv); configPath != "" {
			v.SetConfigFile(configPath)
		} else {
			v.SetConfigName("config")
			v.SetConfigType("yaml")
			v.AddConfigPath(".")                                                    // Current directory
			v.AddConfigPath(filepath.Join(os.Getenv("HOME"), ".codedna"))           // User's home directory
			v.AddConfigPath(filepath.Join(os.Getenv("XDG_CONFIG_HOME"), "codedna")) // XDG config directory
		}

		if err := v.ReadInConfig(); err != nil {
			if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
				cfgErr = fmt.Errorf("failed to read config: %w", err)
				return
			}
			// Config file not found, but that's okay as we have defaults
		}

		var config Config
		if err := v.Unmarshal(&config); err != nil {
			cfgErr = fmt.Errorf("failed to unmarshal config: %w", err)
			return
		}

		cfg = &config
	})

	return cfg, cfgErr
}
