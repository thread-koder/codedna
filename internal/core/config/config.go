package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

const (
	// ConfigPathEnv is the environment variable name for the config file path
	ConfigPathEnv = "CODEDNA_CONFIG"
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
	v := viper.New()

	// Configure environment variable support
	v.SetEnvPrefix("CODEDNA")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	setDefaults(v)

	// Check for config path in environment variable
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
			return nil, fmt.Errorf("failed to read config: %w", err)
		}
		// Config file not found, but that's okay as we have defaults
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &cfg, nil
}
