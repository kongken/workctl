package cli_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kongken/workctl/internal/cli"
	"github.com/kongken/workctl/internal/config"
)

func TestRootHelpListsWorkctl(t *testing.T) {
	buf := new(bytes.Buffer)
	cmd := cli.NewRoot()
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "workctl") {
		t.Fatalf("help missing name: %s", out)
	}
	if strings.Contains(out, "linear") {
		t.Fatalf("help still advertises a linear command: %s", out)
	}
}

func TestRootRegistersCommandGroups(t *testing.T) {
	cmd := cli.NewRoot()
	want := []string{
		"auth", "issue", "team", "user", "project", "project-update",
		"cycle", "milestone", "initiative", "initiative-update",
		"label", "template", "document", "config", "schema", "api", "markdown",
	}
	for _, name := range want {
		found := false
		for _, c := range cmd.Commands() {
			if c.Name() == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("missing command group %q", name)
		}
	}
	issue, _, err := cmd.Find([]string{"i"})
	if err != nil || issue.Name() != "issue" {
		t.Fatalf("alias i -> issue failed: %v name=%v", err, issue)
	}
}

func TestWorkspaceFlagSetsConfig(t *testing.T) {
	config.SetCLIWorkspace("")
	cmd := cli.NewRoot()
	cmd.SetOut(new(bytes.Buffer))
	cmd.SetArgs([]string{"--workspace", "acme"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if got := config.CLIWorkspace(); got != "acme" {
		t.Fatalf("CLIWorkspace = %q, want acme", got)
	}
}

func TestConfigGetReportsInvalidUserConfig(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	path := filepath.Join(dir, "linear", "config.toml")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("team_id = \"unterminated\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	cmd := cli.NewRoot()
	cmd.SetOut(new(bytes.Buffer))
	cmd.SetErr(new(bytes.Buffer))
	cmd.SetArgs([]string{"config", "get", "team_id"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected invalid user config to fail")
	}
	if !strings.Contains(err.Error(), "Invalid user configuration") || !strings.Contains(err.Error(), "toml:") {
		t.Fatalf("unexpected error: %v", err)
	}
}
