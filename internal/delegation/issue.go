package delegation

import (
	"encoding/json"
	"fmt"
	"strings"
)

// LinearIssue is the minimal Linear payload needed for a handoff.
type LinearIssue struct {
	Identifier  string  `json:"identifier"`
	Title       string  `json:"title"`
	Description *string `json:"description"`
	URL         string  `json:"url"`
}

// ParseIssue decodes a Linear issue JSON payload and validates its identity.
func ParseIssue(raw json.RawMessage) (LinearIssue, error) {
	var issue LinearIssue
	if err := json.Unmarshal(raw, &issue); err != nil {
		return LinearIssue{}, fmt.Errorf("decode Linear issue: %w", err)
	}
	if issue.Identifier == "" || issue.Title == "" {
		return LinearIssue{}, fmt.Errorf("Linear issue response is missing identifier or title")
	}
	return issue, nil
}

// HandoffDescription builds the Multica issue body, preserving the Linear
// source and the prompt file used for the handoff.
func HandoffDescription(issue LinearIssue, promptFile string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "## Source\n- Linear issue: %s\n", issue.Identifier)
	if issue.URL != "" {
		fmt.Fprintf(&b, "- URL: %s\n", issue.URL)
	}
	if promptFile != "" {
		fmt.Fprintf(&b, "- Prompt file: %s\n", promptFile)
	}
	b.WriteString("\n")
	if issue.Description != nil && strings.TrimSpace(*issue.Description) != "" {
		b.WriteString(*issue.Description)
		b.WriteString("\n")
	}
	return b.String()
}
