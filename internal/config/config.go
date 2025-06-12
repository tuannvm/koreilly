package config

import (
	"fmt"
	"log"
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
	log.Printf("Loading configuration...")
	c := &Config{}

	// Set default values
	c.Debug = false
	c.LogLevel = "info"
	c.OutputDir = "books"
	log.Printf("Default values - Debug: %v, LogLevel: %s, OutputDir: %s", c.Debug, c.LogLevel, c.OutputDir)

	// Bind environment variables with GOREILLY_ prefix
	viper.SetEnvPrefix("GOREILLY")
	viper.AutomaticEnv()

	// Set default values in viper
	viper.SetDefault("debug", false)
	viper.SetDefault("log_level", "info")
	viper.SetDefault("output_dir", "books")

	// Always use ~/.config/goreilly/config.yaml (cross-platform, not OS default)
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}
	configPath := filepath.Join(home, ".config", "goreilly", "config.yaml")
	log.Printf("Looking for config file at: %s", configPath)

	viper.SetConfigFile(configPath)

	if err := viper.ReadInConfig(); err != nil {
		if os.IsNotExist(err) {
			log.Printf("Config file not found at %s, using defaults", configPath)
		} else {
			log.Printf("Error reading config file: %v", err)
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
	} else {
		log.Printf("Successfully read config from %s", configPath)
	}

	// Debug: Print all settings before unmarshaling
	log.Printf("All settings before unmarshal: %+v", viper.AllSettings())

	// Unmarshal config
	if err := viper.Unmarshal(c); err != nil {
		log.Printf("Error unmarshaling config: %v", err)
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	log.Printf("Final config - Debug: %v, LogLevel: %s, OutputDir: %s", c.Debug, c.LogLevel, c.OutputDir)
	return c, nil
}

// Save saves the configuration to file
func (c *Config) Save() error {
	viper.Set("username", c.Username)
	viper.Set("password", c.Password)
	viper.Set("gmail.email", c.Gmail.Email)
	viper.Set("kindle.email", c.Kindle.Email)

	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}
	appDir := filepath.Join(home, ".config", "goreilly")
	if err := os.MkdirAll(appDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}
	configFile := filepath.Join(appDir, "config.yaml")
	return viper.WriteConfigAs(configFile)
}
