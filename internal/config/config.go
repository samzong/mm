package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

type Config struct {
	Example string `mapstructure:"example"`
}

var DefaultConfig = Config{
	Example: "default-value",
}

func LoadConfig(cfgFile string, cliName string) (*Config, error) {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to find home directory: %w", err)
		}

		configDir := filepath.Join(homeDir, ".config", cliName)
		if err := os.MkdirAll(configDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create config directory: %w", err)
		}

		viper.AddConfigPath(configDir)
		viper.AddConfigPath(homeDir)
		viper.SetConfigName(fmt.Sprintf(".%s", cliName))
		viper.SetConfigType("yaml")
	}

	viper.SetDefault("example", DefaultConfig.Example)

	viper.AutomaticEnv()
	viper.SetEnvPrefix(cliName)

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}

		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to find home directory: %w", err)
		}

		configDir := filepath.Join(homeDir, ".config", cliName)
		if err := os.MkdirAll(configDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create config directory: %w", err)
		}

		configFile := filepath.Join(configDir, fmt.Sprintf(".%s.yaml", cliName))
		viper.SetConfigFile(configFile)

		cfg := DefaultConfig
		if err := SaveConfig(&cfg); err != nil {
			return nil, fmt.Errorf("failed to save default config: %w", err)
		}
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &config, nil
}

func SaveConfig(config *Config) error {

	viper.Set("example", config.Example)

	return viper.WriteConfig()
}

func GetConfigPath() string {
	return viper.ConfigFileUsed()
}
