// Package delegation implements the synchronous Linear-to-Multica handoff and
// local prompt sync shared by the workctl CLI and the MCP server.
//
// The bridge reads a Linear issue through workctl's existing GraphQL client,
// optionally syncs a local UTF-8 prompt file to a Multica agent, then creates a
// Multica issue assigned to that agent. It is a synchronous handoff: it does
// not poll Multica runs or write status back to Linear.
package delegation

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	clierrors "github.com/kongken/workctl/internal/errors"
	"github.com/kongken/workctl/internal/linear"
	"github.com/kongken/workctl/internal/multicacli"
)

// MulticaClient is the Multica subset the delegation bridge needs.
type MulticaClient interface {
	UpdateAgentInstructions(ctx context.Context, agentID, instructions string) (json.RawMessage, error)
	CreateIssue(ctx context.Context, input multicacli.CreateIssueInput) (json.RawMessage, error)
}

// Service performs Linear-to-Multica handoffs. The function fields default to
// the real Linear client and are overridable in tests.
type Service struct {
	ResolveIssue func(provided string) (string, error)
	FetchIssue   func(ctx context.Context, issueID string) (json.RawMessage, error)
	Multica      MulticaClient
}

// New returns a Service wired to the real Linear client and the Multica CLI
// executable (default: "multica").
func New(multicaCommand string) Service {
	return Service{
		ResolveIssue: linear.IssueIdentifier,
		FetchIssue:   linear.FetchIssueDetailsJSON,
		Multica:      multicacli.New(multicaCommand),
	}
}

// DelegateInput describes one handoff request.
type DelegateInput struct {
	Issue      string
	AgentID    string
	ProjectID  string
	Status     string
	PromptFile string
}

// DelegateResult maps the Linear source to the created Multica issue.
type DelegateResult struct {
	LinearIssue string          `json:"linear_issue"`
	LinearURL   string          `json:"linear_url"`
	PromptFile  string          `json:"prompt_file"`
	Multica     json.RawMessage `json:"multica"`
}

// Delegate reads a Linear issue, optionally syncs a local prompt file, and
// creates an assigned Multica issue.
func (s Service) Delegate(ctx context.Context, in DelegateInput) (*DelegateResult, error) {
	if strings.TrimSpace(in.AgentID) == "" {
		return nil, clierrors.NewValidationError("agent id is required")
	}
	issueID, err := s.ResolveIssue(in.Issue)
	if err != nil {
		return nil, err
	}
	if issueID == "" {
		return nil, clierrors.NewValidationError("could not determine Linear issue identifier")
	}
	raw, err := s.FetchIssue(ctx, issueID)
	if err != nil {
		return nil, err
	}
	issue, err := ParseIssue(raw)
	if err != nil {
		return nil, err
	}

	promptName := ""
	if in.PromptFile != "" {
		instructions, err := ReadPromptFile(in.PromptFile)
		if err != nil {
			return nil, err
		}
		if _, err := s.Multica.UpdateAgentInstructions(ctx, in.AgentID, instructions); err != nil {
			return nil, err
		}
		promptName = in.PromptFile
	}

	result, err := s.Multica.CreateIssue(ctx, multicacli.CreateIssueInput{
		Title:       issue.Title,
		Description: HandoffDescription(issue, promptName),
		AssigneeID:  in.AgentID,
		ProjectID:   in.ProjectID,
		Status:      in.Status,
	})
	if err != nil {
		return nil, err
	}
	return &DelegateResult{
		LinearIssue: issue.Identifier,
		LinearURL:   issue.URL,
		PromptFile:  promptName,
		Multica:     result,
	}, nil
}

// ReadPromptFile loads a non-empty UTF-8 prompt file.
func ReadPromptFile(path string) (string, error) {
	instructions, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read prompt file: %w", err)
	}
	if strings.TrimSpace(string(instructions)) == "" {
		return "", clierrors.NewValidationError("prompt file is empty")
	}
	return string(instructions), nil
}
