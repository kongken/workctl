package cmdmultica

import "testing"

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
