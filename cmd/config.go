package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/samzong/cli-template/internal/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage configuration",
	Long:  "View and modify configuration options",
}

var configViewCmd = &cobra.Command{
	Use:   "view",
	Short: "View current configuration",
	Long:  "Display the contents of the configuration file",
	Run: func(cmd *cobra.Command, args []string) {
		configPath := config.GetConfigPath()
		if configPath == "" {
			fmt.Println("Configuration file: Not found")
		} else {
			fmt.Printf("Configuration file: %s\n", configPath)
		}

		if cfg == nil {
			cfg = &config.DefaultConfig
		}

		fmt.Printf("\nConfiguration values:\n")
		fmt.Printf("  Example: %s\n", cfg.Example)
	},
}

var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize configuration file",
	Long:  "Create a new configuration file with default values",
	Run: func(cmd *cobra.Command, args []string) {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: Failed to find home directory: %v\n", err)
			return
		}

		configDir := filepath.Join(homeDir, ".config", CLI_NAME)
		if err := os.MkdirAll(configDir, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "Error: Failed to create config directory: %v\n", err)
			return
		}

		configFile := filepath.Join(configDir, fmt.Sprintf(".%s.yaml", CLI_NAME))

		if _, err := os.Stat(configFile); err == nil {
			fmt.Printf("Configuration file already exists at: %s\n", configFile)
			return
		}

		viper.SetConfigFile(configFile)

		defaultConfig := config.DefaultConfig
		viper.Set("example", defaultConfig.Example)

		if err := viper.WriteConfig(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: Failed to write config file: %v\n", err)
			return
		}

		fmt.Printf("Configuration file created at: %s\n", configFile)
		fmt.Println("Default configuration has been initialized.")
	},
}

func init() {
	configCmd.AddCommand(configViewCmd)
	configCmd.AddCommand(configInitCmd)
	rootCmd.AddCommand(configCmd)
}
