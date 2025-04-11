# Config Handler

A Golang application to automatically synchronize Linux configuration files with a GitHub repository.

## Overview

Config Handler monitors your Linux configuration files (by default in `~/.config`) and automatically synchronizes any changes with a specified GitHub repository. This allows you to:

- Back up your configuration files
- Sync configurations across multiple machines
- Keep a history of configuration changes
- Easily restore configurations on a new system

## Features

- Automatic monitoring of configuration files
- Real-time synchronization with GitHub
- Intelligent detection of file changes
- Support for all types of configuration files
- Secure credentials storage in a protected `.env` file
- Command-line arguments for customization
- Configuration file support
- Include/exclude patterns using glob syntax

## Prerequisites

- Go 1.16 or higher
- Git installed on your system
- A GitHub account
- A GitHub repository to store your configurations
- A GitHub personal access token with repo permissions

## Installation

1. Clone this repository:

   ```
   git clone https://github.com/IvanDec0/dotconfig_automatic.git
   ```

2. Build the application:

   ```
   cd dotconfig_automatic
   go build -o dotconfig_automatic
   ```

3. Run the application:
   ```
   ./dotconfig_automatic
   ```

## Command-Line Arguments

Config Handler supports the following command-line arguments:

```
Usage:
  ./config_handler [flags]

Flags:
  -c, --config-file string       Configuration file path (default "~/.config_handler/config.yaml")
      --config-dir string        Directory containing configuration files to sync (default "~/.config")
      --exclude strings          Directories/files to exclude (comma-separated)
      --github-repo string       GitHub repository URL
      --github-token string      GitHub personal access token
      --github-user string       GitHub username
      --include strings          Directories/files to include (comma-separated)
      --repo-dir string          Directory for the git repository (default "~/.config_sync_repo")
      --run-once                 Sync once and exit
  -i, --sync-interval duration   Interval between checking for changes (default 5s)
      --sync-only                Only perform sync without starting watcher
  -v, --verbose                  Enable verbose logging
      --version                  Show version information
```

Examples:

```bash
# Sync specific directories only
./config_handler --include="i3,polybar,nvim"

# Exclude certain directories
./config_handler --exclude="*cache*,*.log"

# Custom sync interval
./config_handler --sync-interval=10s

# Run once without continuous monitoring
./config_handler --run-once

# Verbose mode
./config_handler -v
```

## Configuration File

Config Handler supports a YAML configuration file for persistent settings. By default, it's located at `~/.config_handler/config.yaml` but can be specified with the `--config-file` flag.

Example configuration file:

```yaml
# Paths
config_dir: "/home/username/.config"
repo_dir: "/home/username/.config_sync_repo"

# GitHub settings
github_repo_url: "https://github.com/username/configs.git"
github_username: "yourusername"
github_token: "your_token"

# Sync settings
sync_interval: "5s"

# Include/Exclude patterns
include:
  - "i3"
  - "polybar"
  - "nvim/init.vim"

exclude:
  - "**/cache/**"
  - "**/*.log"
```

## Setup

When you run the application for the first time, it will prompt you for:

1. GitHub repository URL (e.g., https://github.com/username/configs.git)
2. GitHub username
3. GitHub personal access token

After providing this information, the application will:

- Save your credentials securely in a `.env` file in `dotconfig_automatic/`
- Create a local git repository to store your configurations
- Perform an initial sync of your configuration files
- Start monitoring for changes

On subsequent runs, the application will load your credentials and configuration, so you won't need to enter them again.

## Security

The application stores your GitHub credentials in a protected `.env` file in the current directory
with permissions set to 0600 (readable only by the owner). This ensures that:

- Credentials are stored outside the Git repository to prevent accidental exposure
- Only the account owner can access the file
- The application's `.gitignore` excludes `.env` files as an additional safeguard

## Include/Exclude Patterns

You can use glob patterns to specify which files to include or exclude:

- `*` matches any sequence of non-separator characters
- `**` matches any sequence of characters, including separator characters
- `?` matches any single non-separator character
- `[seq]` matches any character in the sequence
- `{s1,s2}` matches either string s1 or s2

Examples:

- `i3/**` - Include all files in the i3 directory
- `*.conf` - Include all .conf files
- `**/cache/**` - Exclude all cache directories

## How It Works

1. Config Handler creates a local git repository at the specified repo directory
2. It copies your configuration files from the config directory to this repository
3. It sets up a file watcher to monitor for changes in your configuration directory
4. When a change is detected, it automatically:
   - Copies the changed file to the repository
   - Commits the change
   - Pushes to GitHub

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License - see the LICENSE file for details.
