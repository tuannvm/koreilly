package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

type Config struct {
	Username  string `mapstructure:"username"`
	Password  string `mapstructure:"password"`
	Debug     bool   `mapstructure:"debug"`
	LogLevel  string `mapstructure:"log_level"`
	OutputDir string `mapstructure:"output_dir"`
	Gmail     struct {
		Email    string `mapstructure:"email"`
		Password string `mapstructure:"password"`
	} `mapstructure:"gmail"`
	Kindle struct {
		Email string `mapstructure:"email"`
	} `mapstructure:"kindle"`
}

// Load loads the configuration from file and environment variables
func Load() (*Config, error) {
	c := &Config{}

	// Set default values
	c.Debug = false
	c.LogLevel = "info"
	c.OutputDir = "books"

	// Bind environment variables with GOREILLY_ prefix
	viper.SetEnvPrefix("GOREILLY")
	viper.AutomaticEnv()

	// Set default values in viper
	viper.SetDefault("debug", false)
	viper.SetDefault("log_level", "info")
	viper.SetDefault("output_dir", "books")

	// Read from config file if exists
	configDir, err := os.UserConfigDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get config directory: %w", err)
	}

	configPath := filepath.Join(configDir, "goreilly", "config.yaml")
	viper.SetConfigFile(configPath)

	if err := viper.ReadInConfig(); err != nil {
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
	}

	// Unmarshal config
	if err := viper.Unmarshal(c); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	return c, nil
}

// Save saves the configuration to file
func (c *Config) Save() error {
	viper.Set("username", c.Username)
	viper.Set("password", c.Password)
	viper.Set("gmail.email", c.Gmail.Email)
	viper.Set("kindle.email", c.Kindle.Email)

	configDir, err := os.UserConfigDir()
	if err != nil {
		return fmt.Errorf("failed to get config directory: %w", err)
	}

	appDir := filepath.Join(configDir, "goreilly")
	if err := os.MkdirAll(appDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	configFile := filepath.Join(appDir, "config.yaml")
	return viper.WriteConfigAs(configFile)
}
