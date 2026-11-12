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

func SetCLIWorkspace(slug string) { cliWorkspace = slug }
func CLIWorkspace() string        { return cliWorkspace }

var userConfig map[string]any
var projectConfig map[string]any

// UserConfigPath is the shared user configuration file used by Linear and Plane.
// It is ~/.config/workctl.toml (or XDG_CONFIG_HOME/workctl.toml); Windows uses
// %APPDATA%/workctl.toml.
func UserConfigPath() (string, error) {
	if runtime.GOOS == "windows" {
		if appData := os.Getenv("APPDATA"); appData != "" {
			return filepath.Join(appData, "workctl.toml"), nil
		}
		return "", fmt.Errorf("APPDATA is not set")
	}
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "workctl.toml"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "workctl.toml"), nil
}

// LoadUserConfig is retained for callers and tests that provide an explicit
// file. Both the new [linear] section and the old flat Linear format work.
func LoadUserConfig(paths ...string) error {
	userConfig = nil
	return loadLinearConfig(&userConfig, paths)
}

func LoadUserConfigFromHome() error {
	path, err := UserConfigPath()
	if err != nil {
		return nil
	}
	legacy := filepath.Join(filepath.Dir(path), "linear", "config.toml")
	return LoadUserConfig(path, legacy)
}

func Load() error {
	if err := LoadUserConfigFromHome(); err != nil {
		return fmt.Errorf("Invalid user configuration: %w", err)
	}
	if err := LoadProjectConfigFromCwd(); err != nil {
		return fmt.Errorf("Invalid project configuration: %w", err)
	}
	return nil
}

func LoadProjectConfig(paths ...string) error {
	projectConfig = nil
	return loadConfig(&projectConfig, paths)
}

func loadLinearConfig(target *map[string]any, paths []string) error {
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return err
		}
		var root map[string]any
		if _, err := toml.Decode(string(data), &root); err != nil {
			return fmt.Errorf("invalid TOML in %s: %w", path, err)
		}
		if section, ok := root["linear"]; ok {
			m, ok := section.(map[string]any)
			if !ok {
				return fmt.Errorf("linear configuration in %s must be a table", path)
			}
			*target = m
		} else {
			// Legacy ~/.config/linear/config.toml and flat explicit files.
			*target = root
		}
		return nil
	}
	return nil
}

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
			return fmt.Errorf("invalid TOML in %s: %w", path, err)
		}
		*target = parsed
		return nil
	}
	return nil
}

func LoadProjectConfigFromCwd() error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	return LoadProjectConfig(filepath.Join(cwd, "linear.toml"), filepath.Join(cwd, ".linear.toml"))
}

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
	return s, ok && s != ""
}
