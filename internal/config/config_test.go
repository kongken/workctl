package config_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/kongken/workctl/internal/config"
)

func TestCLIWorkspaceRoundTrip(t *testing.T) {
	config.SetCLIWorkspace("")
	if config.CLIWorkspace() != "" {
		t.Fatal("expected empty")
	}
	config.SetCLIWorkspace("acme")
	if config.CLIWorkspace() != "acme" {
		t.Fatalf("got %q", config.CLIWorkspace())
	}
}

func TestGetOptionFromLinearToml(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".linear.toml")
	if err := os.WriteFile(path, []byte("team_id = \"ENG\"\nworkspace = \"acme\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := config.LoadProjectConfig(path); err != nil {
		t.Fatal(err)
	}
	team, ok := config.GetOption("team_id")
	if !ok || team != "ENG" {
		t.Fatalf("team_id = %q ok=%v", team, ok)
	}
	ws, ok := config.GetOption("workspace")
	if !ok || ws != "acme" {
		t.Fatalf("workspace = %q ok=%v", ws, ok)
	}
	if _, ok := config.GetOption("missing"); ok {
		t.Fatal("expected missing key")
	}
}

func TestGetOptionFromEnv(t *testing.T) {
	if err := config.LoadProjectConfig(); err != nil {
		t.Fatal(err)
	}
	t.Setenv("LINEAR_TEAM_ID", "OPS")
	team, ok := config.GetOption("team_id")
	if !ok || team != "OPS" {
		t.Fatalf("team_id = %q ok=%v", team, ok)
	}
}

func TestGetOptionEnvOverridesToml(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".linear.toml")
	if err := os.WriteFile(path, []byte("team_id = \"ENG\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := config.LoadProjectConfig(path); err != nil {
		t.Fatal(err)
	}
	t.Setenv("LINEAR_TEAM_ID", "OPS")
	team, ok := config.GetOption("team_id")
	if !ok || team != "OPS" {
		t.Fatalf("team_id = %q ok=%v", team, ok)
	}
}

func TestUserConfigPathUsesXDG(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("XDG_CONFIG_HOME is a Unix convention")
	}
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	path, err := config.UserConfigPath()
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(dir, "workctl.toml")
	if path != want {
		t.Fatalf("path = %q want %q", path, want)
	}
}

func writeUserConfig(t *testing.T, body string) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	path := filepath.Join(dir, "workctl.toml")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := config.LoadUserConfigFromHome(); err != nil {
		t.Fatal(err)
	}
}

func TestGetOptionFromUserConfig(t *testing.T) {
	// Reset the project config so this test only observes the user file.
	if err := config.LoadProjectConfig(); err != nil {
		t.Fatal(err)
	}
	writeUserConfig(t, "[linear]\nteam_id = \"HOME\"\nworkspace = \"home-ws\"\n")
	if team, ok := config.GetOption("team_id"); !ok || team != "HOME" {
		t.Fatalf("team_id = %q ok=%v", team, ok)
	}
	if ws, ok := config.GetOption("workspace"); !ok || ws != "home-ws" {
		t.Fatalf("workspace = %q ok=%v", ws, ok)
	}
}

func TestProjectConfigOverridesUserConfig(t *testing.T) {
	writeUserConfig(t, "[linear]\nteam_id = \"HOME\"\n")
	dir := t.TempDir()
	path := filepath.Join(dir, ".linear.toml")
	if err := os.WriteFile(path, []byte("team_id = \"PROJ\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := config.LoadProjectConfig(path); err != nil {
		t.Fatal(err)
	}
	team, ok := config.GetOption("team_id")
	if !ok || team != "PROJ" {
		t.Fatalf("team_id = %q ok=%v", team, ok)
	}
}

func TestEnvOverridesUserConfig(t *testing.T) {
	writeUserConfig(t, "[linear]\nteam_id = \"HOME\"\n")
	if err := config.LoadProjectConfig(); err != nil {
		t.Fatal(err)
	}
	t.Setenv("LINEAR_TEAM_ID", "OPS")
	team, ok := config.GetOption("team_id")
	if !ok || team != "OPS" {
		t.Fatalf("team_id = %q ok=%v", team, ok)
	}
}

func TestMissingUserConfigYieldsNoOption(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if err := config.LoadProjectConfig(); err != nil {
		t.Fatal(err)
	}
	if err := config.LoadUserConfigFromHome(); err != nil {
		t.Fatalf("missing user config should not error: %v", err)
	}
	if _, ok := config.GetOption("team_id"); ok {
		t.Fatal("expected no value from a missing user config")
	}
}

func TestInvalidUserConfigReturnsError(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	path := filepath.Join(dir, "workctl.toml")
	if err := os.WriteFile(path, []byte("team_id = \"unterminated\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	err := config.LoadUserConfigFromHome()
	if err == nil {
		t.Fatal("expected invalid TOML to fail")
	}
	if !strings.Contains(err.Error(), path) || !strings.Contains(err.Error(), "toml:") {
		t.Fatalf("error should name the file and parse failure, got %v", err)
	}
}

func TestLoadReportsInvalidUserConfig(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	path := filepath.Join(dir, "workctl.toml")
	if err := os.WriteFile(path, []byte("team_id = \"unterminated\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	err := config.Load()
	if err == nil {
		t.Fatal("expected Load to fail on invalid user config")
	}
	if !strings.Contains(err.Error(), "Invalid user configuration") {
		t.Fatalf("unexpected error: %v", err)
	}
}
