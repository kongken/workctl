package cmdmultica

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestHandoffDescriptionIncludesSourceAndPrompt(t *testing.T) {
	description := "Acceptance criteria"
	got := handoffDescription(linearIssue{
		Identifier:  "ENG-42",
		Title:       "Fix login",
		Description: &description,
		URL:         "https://linear.app/acme/issue/ENG-42",
	}, ".workctl/agent-prompts/default.md")
	want := "## Source\n- Linear issue: ENG-42\n- URL: https://linear.app/acme/issue/ENG-42\n- Prompt file: .workctl/agent-prompts/default.md\n\nAcceptance criteria\n"
	if got != want {
		t.Fatalf("description = %q, want %q", got, want)
	}
}

func TestHandoffDescriptionAllowsMissingDescription(t *testing.T) {
	got := handoffDescription(linearIssue{Identifier: "ENG-42", Title: "Fix login"}, "")
	want := "## Source\n- Linear issue: ENG-42\n\n"
	if got != want {
		t.Fatalf("description = %q, want %q", got, want)
	}
}

func TestAgentCommandsExposeCRUD(t *testing.T) {
	root := New()
	var agent *cobra.Command
	for _, command := range root.Commands() {
		if command.Name() == "agent" {
			agent = command
			break
		}
	}
	if agent == nil {
		t.Fatal("agent command is not registered")
	}
	want := map[string]bool{"list": false, "get": false, "create": false, "update": false, "delete": false, "restore": false}
	for _, command := range agent.Commands() {
		if _, ok := want[command.Name()]; ok {
			want[command.Name()] = true
		}
	}
	for name, found := range want {
		if !found {
			t.Errorf("agent %s command is not registered", name)
		}
	}
}
