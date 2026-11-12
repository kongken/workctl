package delegation_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/kongken/workctl/internal/delegation"
	"github.com/kongken/workctl/internal/multicacli"
)

func TestHandoffDescriptionIncludesSourceAndPrompt(t *testing.T) {
	description := "Acceptance criteria"
	got := delegation.HandoffDescription(delegation.LinearIssue{
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
	got := delegation.HandoffDescription(delegation.LinearIssue{Identifier: "ENG-42", Title: "Fix login"}, "")
	want := "## Source\n- Linear issue: ENG-42\n\n"
	if got != want {
		t.Fatalf("description = %q, want %q", got, want)
	}
}

func TestParseIssueRejectsMissingIdentity(t *testing.T) {
	if _, err := delegation.ParseIssue(json.RawMessage(`{"title":"Fix login"}`)); err == nil {
		t.Fatal("expected an error for a missing identifier")
	}
}

type fakeMultica struct {
	instructions string
	createInput  multicacli.CreateIssueInput
}

func (f *fakeMultica) UpdateAgentInstructions(_ context.Context, _ string, instructions string) (json.RawMessage, error) {
	f.instructions = instructions
	return json.RawMessage(`{"id":"agent-1"}`), nil
}

func (f *fakeMultica) CreateIssue(_ context.Context, input multicacli.CreateIssueInput) (json.RawMessage, error) {
	f.createInput = input
	return json.RawMessage(`{"identifier":"MUL-1"}`), nil
}

func TestDelegateSyncsPromptBeforeCreatingIssue(t *testing.T) {
	promptFile := filepath.Join(t.TempDir(), "agent.md")
	if err := os.WriteFile(promptFile, []byte("Use the repo standards."), 0o600); err != nil {
		t.Fatal(err)
	}
	fake := &fakeMultica{}
	service := delegation.Service{
		ResolveIssue: func(string) (string, error) { return "ENG-42", nil },
		FetchIssue: func(context.Context, string) (json.RawMessage, error) {
			return json.RawMessage(`{"identifier":"ENG-42","title":"Fix login","url":"https://linear.app/acme/issue/ENG-42","description":"Body"}`), nil
		},
		Multica: fake,
	}

	result, err := service.Delegate(context.Background(), delegation.DelegateInput{
		Issue:      "ENG-42",
		AgentID:    "agent-1",
		ProjectID:  "project-1",
		Status:     "todo",
		PromptFile: promptFile,
	})
	if err != nil {
		t.Fatal(err)
	}
	if fake.instructions != "Use the repo standards." {
		t.Fatalf("instructions = %q", fake.instructions)
	}
	if fake.createInput.Title != "Fix login" || fake.createInput.AssigneeID != "agent-1" || fake.createInput.ProjectID != "project-1" {
		t.Fatalf("create input = %+v", fake.createInput)
	}
	if result.LinearIssue != "ENG-42" || result.PromptFile != promptFile {
		t.Fatalf("result = %+v", result)
	}
}

func TestDelegateRequiresAgent(t *testing.T) {
	service := delegation.Service{
		ResolveIssue: func(string) (string, error) { return "ENG-42", nil },
		FetchIssue: func(context.Context, string) (json.RawMessage, error) {
			return json.RawMessage(`{"identifier":"ENG-42","title":"Fix login"}`), nil
		},
	}
	if _, err := service.Delegate(context.Background(), delegation.DelegateInput{Issue: "ENG-42"}); err == nil {
		t.Fatal("expected an error when no agent is given")
	}
}
