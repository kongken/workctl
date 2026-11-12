package cmdmultica

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	clierrors "github.com/kongken/workctl/internal/errors"
	"github.com/kongken/workctl/internal/linear"
	"github.com/kongken/workctl/internal/multicacli"
	"github.com/spf13/cobra"
)

// New creates the Multica integration command group.
func New() *cobra.Command {
	root := &cobra.Command{
		Use:   "multica",
		Short: "Delegate Linear work to Multica",
	}
	root.AddCommand(newPromptCommand(), newDelegateCommand())
	return root
}

func newPromptCommand() *cobra.Command {
	root := &cobra.Command{Use: "prompt", Short: "Manage local Multica agent prompts"}
	var file string
	var command string
	sync := &cobra.Command{
		Use:   "sync <agent-id>",
		Short: "Sync a local prompt file to a Multica agent",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(file) == "" {
				return clierrors.NewValidationError("--file is required")
			}
			instructions, err := os.ReadFile(file)
			if err != nil {
				return fmt.Errorf("read prompt file: %w", err)
			}
			if strings.TrimSpace(string(instructions)) == "" {
				return clierrors.NewValidationError("prompt file is empty")
			}
			result, err := multicacli.New(command).UpdateAgentInstructions(cmd.Context(), args[0], string(instructions))
			if err != nil {
				return err
			}
			_, err = cmd.OutOrStdout().Write(append(result, '\n'))
			return err
		},
	}
	sync.Flags().StringVar(&file, "file", "", "Local UTF-8 prompt file")
	sync.Flags().StringVar(&command, "multica-command", "", "Multica executable (default: multica)")
	root.AddCommand(sync)
	return root
}

func newDelegateCommand() *cobra.Command {
	var agentID, projectID, status, promptFile, command string
	cmd := &cobra.Command{
		Use:   "delegate <linear-issue>",
		Short: "Create a Multica issue from a Linear issue",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if strings.TrimSpace(agentID) == "" {
				return clierrors.NewValidationError("--agent-id is required")
			}
			issueID, err := linear.IssueIdentifier(args[0])
			if err != nil {
				return err
			}
			if issueID == "" {
				return clierrors.NewValidationError("could not determine Linear issue identifier")
			}
			raw, err := linear.FetchIssueDetailsJSON(cmd.Context(), issueID)
			if err != nil {
				return err
			}
			issue, err := parseIssue(raw)
			if err != nil {
				return err
			}

			client := multicacli.New(command)
			promptName := ""
			if promptFile != "" {
				instructions, readErr := os.ReadFile(promptFile)
				if readErr != nil {
					return fmt.Errorf("read prompt file: %w", readErr)
				}
				if strings.TrimSpace(string(instructions)) == "" {
					return clierrors.NewValidationError("prompt file is empty")
				}
				if _, err := client.UpdateAgentInstructions(cmd.Context(), agentID, string(instructions)); err != nil {
					return err
				}
				promptName = promptFile
			}

			result, err := client.CreateIssue(cmd.Context(), multicacli.CreateIssueInput{
				Title:       issue.Title,
				Description: handoffDescription(issue, promptName),
				AgentID:     agentID,
				ProjectID:   projectID,
				Status:      status,
			})
			if err != nil {
				return err
			}
			return writeJSON(cmd.OutOrStdout(), map[string]any{
				"linear_issue": issue.Identifier,
				"linear_url":   issue.URL,
				"prompt_file":  promptName,
				"multica":      json.RawMessage(result),
			})
		},
	}
	cmd.Flags().StringVar(&agentID, "agent-id", "", "Multica agent UUID")
	cmd.Flags().StringVar(&projectID, "project", "", "Multica project UUID")
	cmd.Flags().StringVar(&status, "status", "todo", "Initial Multica issue status")
	cmd.Flags().StringVar(&promptFile, "prompt-file", "", "Sync this local prompt before delegation")
	cmd.Flags().StringVar(&command, "multica-command", "", "Multica executable (default: multica)")
	return cmd
}

type linearIssue struct {
	Identifier  string  `json:"identifier"`
	Title       string  `json:"title"`
	Description *string `json:"description"`
	URL         string  `json:"url"`
}

func parseIssue(raw json.RawMessage) (linearIssue, error) {
	var issue linearIssue
	if err := json.Unmarshal(raw, &issue); err != nil {
		return linearIssue{}, fmt.Errorf("decode Linear issue: %w", err)
	}
	if issue.Identifier == "" || issue.Title == "" {
		return linearIssue{}, fmt.Errorf("Linear issue response is missing identifier or title")
	}
	return issue, nil
}

func handoffDescription(issue linearIssue, promptFile string) string {
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

func writeJSON(out io.Writer, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	_, err = out.Write(data)
	return err
}
