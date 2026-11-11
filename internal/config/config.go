package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/BurntSushi/toml"
)

var cliWorkspace string

// SetCLIWorkspace records the --workspace global flag value.
func SetCLIWorkspace(slug string) {
	cliWorkspace = slug
}

// CLIWorkspace returns the --workspace global flag value, if set.
func CLIWorkspace() string {
	return cliWorkspace
}

// userConfig holds keys loaded from the user-level config file.
var userConfig map[string]any

// projectConfig holds keys loaded from the nearest project .linear.toml / linear.toml.
var projectConfig map[string]any

// UserConfigPath returns the user-level Linear config file. It lives next to the
// Linear credentials so it does not collide with the separate Plane config:
// %APPDATA%/linear/config.toml on Windows, otherwise
// $XDG_CONFIG_HOME/linear/config.toml (defaulting to ~/.config/linear/config.toml).
func UserConfigPath() (string, error) {
	if runtime.GOOS == "windows" {
		appData := os.Getenv("APPDATA")
		if appData == "" {
			return "", fmt.Errorf("APPDATA is not set")
		}
		return filepath.Join(appData, "linear", "config.toml"), nil
	}
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "linear", "config.toml"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "linear", "config.toml"), nil
}

// LoadUserConfig searches paths in order and keeps the first existing file.
// Callers in tests may pass paths; production should call LoadUserConfigFromHome.
func LoadUserConfig(paths ...string) error {
	userConfig = nil
	return loadConfig(&userConfig, paths)
}

// LoadUserConfigFromHome loads the user-level config file when present. A missing
// home directory simply means there is no user config.
func LoadUserConfigFromHome() error {
	path, err := UserConfigPath()
	if err != nil {
		return nil
	}
	return LoadUserConfig(path)
}

// Load resolves the user-level and project-level config in precedence order,
// returning a descriptive error for the first invalid file instead of silently
// falling back to other sources.
func Load() error {
	if err := LoadUserConfigFromHome(); err != nil {
		return fmt.Errorf("Invalid user configuration: %w", err)
	}
	if err := LoadProjectConfigFromCwd(); err != nil {
		return fmt.Errorf("Invalid project configuration: %w", err)
	}
	return nil
}

// LoadProjectConfig searches cwd (and optional explicit paths) for project config.
// Callers in tests may pass paths; production should call LoadProjectConfigFromCwd.
func LoadProjectConfig(paths ...string) error {
	projectConfig = nil
	return loadConfig(&projectConfig, paths)
}

// loadConfig reads the first existing file in paths into target. A missing file
// is not an error; a parse failure names the offending file.
func loadConfig(target *map[string]any, paths []string) error {
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return err
		}
		var parsed map[string]any
		if _, err := toml.Decode(string(data), &parsed); err != nil {
			// TOML errors report the line and key only, not the values, so the
			// wrapped error is safe to show even when the file holds an api_key.
			return fmt.Errorf("invalid TOML in %s: %w", path, err)
		}
		*target = parsed
		return nil
	}
	return nil
}

// LoadProjectConfigFromCwd loads ./linear.toml or ./.linear.toml if present.
func LoadProjectConfigFromCwd() error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	return LoadProjectConfig(
		filepath.Join(cwd, "linear.toml"),
		filepath.Join(cwd, ".linear.toml"),
	)
}

// GetOption returns a string option (team_id, workspace, api_key, …).
// The LINEAR_<KEY> environment variable (e.g. LINEAR_TEAM_ID) takes precedence
// over project config, which takes precedence over the user-level config.
func GetOption(key string) (string, bool) {
	if v := os.Getenv("LINEAR_" + strings.ToUpper(key)); v != "" {
		return v, true
	}
	if s, ok := stringOption(projectConfig, key); ok {
		return s, true
	}
	return stringOption(userConfig, key)
}

func stringOption(cfg map[string]any, key string) (string, bool) {
	if cfg == nil {
		return "", false
	}
	v, ok := cfg[key]
	if !ok || v == nil {
		return "", false
	}
	s, ok := v.(string)
	if !ok || s == "" {
		return "", false
	}
	return s, true
}
