package main

import (
	"fmt"
	"os"
	"strings"

	"config_handler/cli"
	"config_handler/config"
	"config_handler/env"
	"config_handler/git"
	"config_handler/notification"
	"config_handler/ui"
)

func main() {
	// Parse command-line flags and load configuration
	appConfig, err := cli.ParseFlags()
	if err != nil {
		ui.PrintError("Failed to parse command-line arguments: " + err.Error())
		os.Exit(1)
	}

	// Display application logo and title
	ui.PrintLogo()
	ui.PrintTitle("Linux Configuration Manager")

	// Initialize notification system
	ui.PrintInfo("Initializing notification system...")
	notifyConfig := notification.DefaultConfig()
	notifyManager, err := notification.NewManager(notifyConfig)
	if err != nil {
		ui.PrintWarning("Could not initialize notification system: " + err.Error())
		// Continue without notifications
	} else {
		ui.PrintSuccess("Notification system initialized")
	}

	// Load GitHub credentials from .env file
	ui.PrintSection("Loading Credentials")
	envConfig, err := env.LoadConfig()
	if err != nil {
		ui.PrintWarning("Could not load credentials: " + err.Error())
	}

	// Initialize git repository if it doesn't exist
	ui.PrintInfo("Initializing local repository...")
	gitRepo, err := git.InitOrOpenRepo(appConfig.RepoDir)
	if err != nil {
		ui.PrintError("Failed to initialize repository: " + err.Error())
		os.Exit(1)
	}
	ui.PrintSuccess("Repository initialized successfully")

	// Set up GitHub repository if configuration is available
	configChanged := false

	if !envConfig.IsConfigComplete() {
		ui.PrintSection("GitHub Configuration")
		ui.PrintInfo("Please provide your GitHub credentials")

		// Ask for missing information using Bubble Tea prompts
		if envConfig.GithubRepoURL == "" {
			repoURL := ui.PromptInput("GitHub repository URL (e.g., https://github.com/username/repo.git)", "")
			envConfig.UpdateFromUserInput(repoURL, "", "")
			configChanged = true
		}

		if envConfig.GithubUsername == "" {
			username := ui.PromptInput("GitHub username", "")
			envConfig.UpdateFromUserInput("", username, "")
			configChanged = true
		}

		if envConfig.GithubToken == "" {
			token := ui.PromptInput("GitHub personal access token", "")
			envConfig.UpdateFromUserInput("", "", token)
			configChanged = true
		}

		// Save updated configuration
		if configChanged {
			ui.PrintInfo("Saving credentials...")
			err = envConfig.SaveConfig()
			if err != nil {
				ui.PrintWarning("Could not save credentials: " + err.Error())
			} else {
				ui.PrintSuccess("Credentials saved successfully")
			}
		}
	} else {
		ui.PrintSuccess("Loaded GitHub credentials from .env file")
	}

	// Apply configuration to Git repository
	if envConfig.GithubRepoURL != "" {
		ui.PrintInfo("Setting up GitHub remote...")
		err = gitRepo.SetRemote(envConfig.GithubRepoURL)
		if err != nil {
			ui.PrintError("Failed to set remote URL: " + err.Error())
			os.Exit(1)
		}
	} else {
		ui.PrintError("GitHub repository URL is required for synchronization")
		os.Exit(1)
	}

	if envConfig.GithubUsername != "" && envConfig.GithubToken != "" {
		gitRepo.SetCredentials(envConfig.GithubUsername, envConfig.GithubToken)
		ui.PrintSuccess("GitHub credentials configured successfully")
	} else {
		ui.PrintError("GitHub credentials are required for synchronization")
		os.Exit(1)
	}

	// Save configuration to file
	ui.PrintInfo("Saving application configuration...")
	err = cli.SaveConfig(appConfig)
	if err != nil {
		ui.PrintWarning("Could not save application configuration: " + err.Error())
	} else {
		ui.PrintSuccess("Application configuration saved successfully")
	}

	// Show effective configuration if verbose
	if appConfig.Verbose {
		ui.PrintSection("Effective Configuration")
		ui.PrintInfo("Configuration Directory: " + appConfig.ConfigDir)
		ui.PrintInfo("Repository Directory: " + appConfig.RepoDir)
		ui.PrintInfo("Sync Interval: " + appConfig.SyncInterval.String())

		if len(appConfig.IncludePatterns) > 0 {
			ui.PrintInfo("Include Patterns: " + strings.Join(appConfig.IncludePatterns, ", "))
		}

		if len(appConfig.ExcludePatterns) > 0 {
			ui.PrintInfo("Exclude Patterns: " + strings.Join(appConfig.ExcludePatterns, ", "))
		}

		ui.PrintInfo(fmt.Sprintf("Operation Mode: %s", getOperationMode(appConfig)))
	}

	// Setup config manager
	configManager := config.NewManager(
		appConfig.ConfigDir,
		appConfig.RepoDir,
		gitRepo,
		appConfig.IncludePatterns,
		appConfig.ExcludePatterns,
		appConfig.SyncInterval,
		appConfig.Verbose,
		notifyManager,
	)

	// Do initial sync
	ui.PrintSection("Initial Synchronization")
	ui.PrintProgress("Performing initial sync of configuration files", 3)

	err = configManager.InitialSync()
	if err != nil {
		ui.PrintError("Failed during initial sync: " + err.Error())
		os.Exit(1)
	}
	ui.PrintSuccess("Initial sync completed successfully!")

	// If run-once flag is set, exit after initial sync
	if appConfig.RunOnce {
		ui.PrintInfo("Run-once flag set. Exiting after initial sync.")
		os.Exit(0)
	}

	// If sync-only flag is set, exit after initial sync
	if appConfig.SyncOnly {
		ui.PrintInfo("Sync-only flag set. Exiting after initial sync.")
		os.Exit(0)
	}

	// Start file watcher to detect changes
	ui.PrintSection("File Monitoring")
	ui.PrintInfo("Starting file watcher for configuration changes...")
	err = configManager.StartWatcher()
	if err != nil {
		ui.PrintError("Failed to start file watcher: " + err.Error())
		os.Exit(1)
	}
	ui.PrintSuccess("File watcher started successfully")
	ui.PrintSeparator()
	ui.PrintInfo("Now monitoring for configuration changes. Press Ctrl+C to exit.")

	// Keep the application running
	select {}
}

// getOperationMode returns a string describing the current operation mode
func getOperationMode(config *cli.AppConfig) string {
	if config.RunOnce {
		return "Run Once (sync and exit)"
	} else if config.SyncOnly {
		return "Sync Only (no monitoring)"
	} else {
		return "Normal (sync and monitor)"
	}
}
