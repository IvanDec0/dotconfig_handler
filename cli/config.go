package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// AppConfig holds the application configuration
type AppConfig struct {
	// Paths
	ConfigDir  string `mapstructure:"config_dir"`
	RepoDir    string `mapstructure:"repo_dir"`
	ConfigFile string `mapstructure:"config_file"`

	// Sync settings
	SyncInterval time.Duration `mapstructure:"sync_interval"`

	// Include/Exclude patterns
	IncludePatterns []string `mapstructure:"include"`
	ExcludePatterns []string `mapstructure:"exclude"`

	// Operation modes
	RunOnce  bool `mapstructure:"run_once"`
	SyncOnly bool `mapstructure:"sync_only"`
	Verbose  bool `mapstructure:"verbose"`
}

// ParseFlags parses command-line flags and loads configuration from file
func ParseFlags() (*AppConfig, error) {
	// Set up configuration defaults
	config := &AppConfig{}

	// Get home directory for default config location
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("error getting home directory: %w", err)
	}

	// Default paths
	defaultConfigDir := filepath.Join(homeDir, ".config")
	defaultRepoDir := filepath.Join(homeDir, ".config_sync_repo")
	defaultConfigFile := filepath.Join("config.yml")

	// Set up command line flags
	pflag.StringVar(&config.ConfigDir, "config-dir", defaultConfigDir, "Directory containing configuration files to sync")
	pflag.StringVar(&config.RepoDir, "repo-dir", defaultRepoDir, "Directory for the git repository")
	pflag.StringVarP(&config.ConfigFile, "config-file", "c", defaultConfigFile, "Configuration file path")

	pflag.DurationVarP(&config.SyncInterval, "sync-interval", "i", 5*time.Second, "Interval between checking for changes")

	pflag.StringSliceVar(&config.IncludePatterns, "include", []string{}, "Directories/files to include (comma-separated)")
	pflag.StringSliceVar(&config.ExcludePatterns, "exclude", []string{}, "Directories/files to exclude (comma-separated)")

	pflag.BoolVar(&config.RunOnce, "run-once", false, "Sync once and exit")
	pflag.BoolVar(&config.SyncOnly, "sync-only", false, "Only perform sync without starting watcher")
	pflag.BoolVarP(&config.Verbose, "verbose", "v", false, "Enable verbose logging")

	// Parse the flags
	pflag.Parse()

	// Create a new Viper instance to avoid duplicated keys
	v := viper.New()
	v.SetConfigFile(config.ConfigFile)

	// Try to read configuration file
	if err := v.ReadInConfig(); err != nil {
		// It's okay if config file doesn't exist
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
	}

	// Use flag values if they were set, otherwise use values from config file
	if v.IsSet("config_dir") && !pflag.CommandLine.Changed("config-dir") {
		config.ConfigDir = v.GetString("config_dir")
	}

	if v.IsSet("repo_dir") && !pflag.CommandLine.Changed("repo-dir") {
		config.RepoDir = v.GetString("repo_dir")
	}

	if v.IsSet("sync_interval") && !pflag.CommandLine.Changed("sync-interval") {
		config.SyncInterval = v.GetDuration("sync_interval")
	}

	if v.IsSet("include") && !pflag.CommandLine.Changed("include") {
		config.IncludePatterns = v.GetStringSlice("include")
	}

	if v.IsSet("exclude") && !pflag.CommandLine.Changed("exclude") {
		config.ExcludePatterns = v.GetStringSlice("exclude")
	}

	if v.IsSet("run_once") && !pflag.CommandLine.Changed("run-once") {
		config.RunOnce = v.GetBool("run_once")
	}

	if v.IsSet("sync_only") && !pflag.CommandLine.Changed("sync-only") {
		config.SyncOnly = v.GetBool("sync_only")
	}

	if v.IsSet("verbose") && !pflag.CommandLine.Changed("verbose") {
		config.Verbose = v.GetBool("verbose")
	}

	return config, nil
}

// SaveConfig saves the configuration to a file
func SaveConfig(config *AppConfig) error {
	// Handle Viper config
	v := viper.New()
	v.SetConfigFile(config.ConfigFile)

	// Set the configuration values using only the underscore format
	// to prevent duplicate keys in the config file
	v.Set("config_dir", config.ConfigDir)
	v.Set("repo_dir", config.RepoDir)
	v.Set("sync_interval", config.SyncInterval)
	v.Set("include", config.IncludePatterns)
	v.Set("exclude", config.ExcludePatterns)
	v.Set("run_once", config.RunOnce)
	v.Set("sync_only", config.SyncOnly)
	v.Set("verbose", config.Verbose)

	// Ensure the config directory exists
	configDir := filepath.Dir(config.ConfigFile)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("error creating config directory: %w", err)
	}

	// Write the configuration to file
	if err := v.WriteConfigAs(config.ConfigFile); err != nil {
		return fmt.Errorf("error writing config file: %w", err)
	}

	return nil
}
