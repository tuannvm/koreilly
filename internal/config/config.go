package config

import (
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

type Config struct {
	Debug     bool   `mapstructure:"debug"`
	LogLevel  string `mapstructure:"log_level"`
	APIKey    string `mapstructure:"api_key"`
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
	// Set default values
	viper.SetDefault("debug", false)
	viper.SetDefault("log_level", "info")
	viper.SetDefault("output_dir", "books")

	// Read from config file if exists
	const (
		DefaultConfigDir  = ".config/goreilly"
		DefaultConfigFile = "config.yaml"
	)
	configDir, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}

	appDir := filepath.Join(configDir, "koreilly")
	if err := os.MkdirAll(appDir, 0755); err != nil {
		return nil, err
	}

	viper.AddConfigPath(appDir)
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")

	// Ignore if config file doesn't exist
	_ = viper.ReadInConfig()

	// Read from environment variables
	viper.SetEnvPrefix("KOREILLY")
	viper.AutomaticEnv()

	// Bind environment variables
	viper.BindEnv("api_key", "KOREILLY_API_KEY")
	viper.BindEnv("gmail.email", "KOREILLY_GMAIL_EMAIL")
	viper.BindEnv("gmail.password", "KOREILLY_GMAIL_PASSWORD")
	viper.BindEnv("kindle.email", "KOREILLY_KINDLE_EMAIL")

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// Save saves the configuration to file
func (c *Config) Save() error {
	viper.Set("api_key", c.APIKey)
	viper.Set("gmail.email", c.Gmail.Email)
	viper.Set("kindle.email", c.Kindle.Email)

	configDir, err := os.UserConfigDir()
	if err != nil {
		return err
	}

	appDir := filepath.Join(configDir, "koreilly")
	if err := os.MkdirAll(appDir, 0755); err != nil {
		return err
	}

	configFile := filepath.Join(appDir, "config.yaml")
	return viper.WriteConfigAs(configFile)
}
