package env

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"
)

const (
	// Environment variable keys
	GithubURLKey      = "GITHUB_REPO_URL"
	GithubUsernameKey = "GITHUB_USERNAME"
	GithubTokenKey    = "GITHUB_TOKEN"
)

// Config holds the application configuration loaded from environment
type Config struct {
	GithubRepoURL  string
	GithubUsername string
	GithubToken    string
	EnvFilePath    string
}

// LoadConfig loads configuration from .env file stored in user's home directory
func LoadConfig() (*Config, error) {

	// Set environment file path in app directory
	envPath := filepath.Join(".env")

	// Create a new config with the env file path
	config := &Config{
		EnvFilePath: envPath,
	}

	// Check if .env file exists
	_, err := os.Stat(envPath)
	if os.IsNotExist(err) {
		// .env file doesn't exist, return empty config
		return config, nil
	}

	// Load .env file
	err = godotenv.Load(envPath)
	if err != nil {
		return nil, fmt.Errorf("error loading .env file: %w", err)
	}

	// Get environment variables
	config.GithubRepoURL = os.Getenv(GithubURLKey)
	config.GithubUsername = os.Getenv(GithubUsernameKey)
	config.GithubToken = os.Getenv(GithubTokenKey)

	return config, nil
}

// SaveConfig saves configuration to .env file
func (c *Config) SaveConfig() error {
	// Create content for .env file
	content := fmt.Sprintf("%s=%s\n%s=%s\n%s=%s\n",
		GithubURLKey, c.GithubRepoURL,
		GithubUsernameKey, c.GithubUsername,
		GithubTokenKey, c.GithubToken,
	)

	// Check if directory exists, create if not
	dir := filepath.Dir(c.EnvFilePath)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err = os.MkdirAll(dir, 0755)
		if err != nil {
			return fmt.Errorf("failed to create directory for .env file: %w", err)
		}
	}

	// Write content to .env file
	err := os.WriteFile(c.EnvFilePath, []byte(content), 0600) // 0600 = read/write for owner only
	if err != nil {
		return fmt.Errorf("failed to write .env file: %w", err)
	}

	return nil
}

// IsConfigComplete checks if all required configuration values are set
func (c *Config) IsConfigComplete() bool {
	return c.GithubRepoURL != "" &&
		c.GithubUsername != "" &&
		c.GithubToken != ""
}

// UpdateFromUserInput updates config with user-provided values if they're not empty
func (c *Config) UpdateFromUserInput(repoURL, username, token string) {
	if strings.TrimSpace(repoURL) != "" {
		c.GithubRepoURL = strings.TrimSpace(repoURL)
	}

	if strings.TrimSpace(username) != "" {
		c.GithubUsername = strings.TrimSpace(username)
	}

	if strings.TrimSpace(token) != "" {
		c.GithubToken = strings.TrimSpace(token)
	}
}
