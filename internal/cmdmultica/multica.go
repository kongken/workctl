package cmdmultica

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/kongken/workctl/internal/delegation"
	clierrors "github.com/kongken/workctl/internal/errors"
	"github.com/kongken/workctl/internal/multicacli"
	"github.com/spf13/cobra"
)

// New creates the Multica integration command group.
func New() *cobra.Command {
	root := &cobra.Command{
		Use:   "multica",
		Short: "Delegate Linear work to Multica",
	}
	root.AddCommand(newAgentCommand(), newPromptCommand(), newDelegateCommand())
	return root
}

func newAgentCommand() *cobra.Command {
	root := &cobra.Command{
		Use:   "agent",
		Short: "Manage Multica agents",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}
	root.AddCommand(
		newAgentListCommand(),
		newAgentGetCommand(),
		newAgentCreateCommand(),
		newAgentUpdateCommand(),
		newAgentDeleteCommand(),
		newAgentRestoreCommand(),
	)
	return root
}

func newAgentListCommand() *cobra.Command {
	var includeArchived bool
	var command string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List Multica agents",
		RunE: func(cmd *cobra.Command, _ []string) error {
			result, err := multicacli.New(command).ListAgents(cmd.Context(), includeArchived)
			if err != nil {
				return err
			}
			return writeRaw(cmd.OutOrStdout(), result)
		},
	}
	cmd.Flags().BoolVar(&includeArchived, "include-archived", false, "Include archived agents")
	cmd.Flags().StringVar(&command, "multica-command", "", "Multica executable (default: multica)")
	return cmd
}

func newAgentGetCommand() *cobra.Command {
	var command string
	cmd := &cobra.Command{
		Use:   "get <agent-id>",
		Short: "Get a Multica agent",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := multicacli.New(command).GetAgent(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			return writeRaw(cmd.OutOrStdout(), result)
		},
	}
	cmd.Flags().StringVar(&command, "multica-command", "", "Multica executable (default: multica)")
	return cmd
}

func newAgentCreateCommand() *cobra.Command {
	var input multicacli.AgentCreateInput
	var promptFile, command string
	var maxConcurrentTasks int32
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a Multica agent",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if strings.TrimSpace(input.Name) == "" {
				return clierrors.NewValidationError("--name is required")
			}
			if strings.TrimSpace(input.RuntimeID) == "" {
				return clierrors.NewValidationError("--runtime-id is required")
			}
			if promptFile != "" {
				if cmd.Flags().Changed("instructions") {
					return clierrors.NewValidationError("--instructions and --instructions-file are mutually exclusive")
				}
				instructions, err := delegation.ReadPromptFile(promptFile)
				if err != nil {
					return err
				}
				input.Instructions = instructions
			}
			if cmd.Flags().Changed("max-concurrent-tasks") {
				input.MaxConcurrentTasks = &maxConcurrentTasks
			}
			result, err := multicacli.New(command).CreateAgent(cmd.Context(), input)
			if err != nil {
				return err
			}
			return writeRaw(cmd.OutOrStdout(), result)
		},
	}
	cmd.Flags().StringVar(&input.Name, "name", "", "Agent name (required)")
	cmd.Flags().StringVar(&input.RuntimeID, "runtime-id", "", "Runtime UUID (required)")
	cmd.Flags().StringVar(&input.Description, "description", "", "Agent description")
	cmd.Flags().StringVar(&input.Instructions, "instructions", "", "Agent instructions")
	cmd.Flags().StringVar(&promptFile, "instructions-file", "", "Read agent instructions from a UTF-8 file")
	cmd.Flags().StringVar(&input.Model, "model", "", "Model identifier")
	cmd.Flags().Int32Var(&maxConcurrentTasks, "max-concurrent-tasks", 0, "Maximum concurrent runs")
	cmd.Flags().StringVar(&command, "multica-command", "", "Multica executable (default: multica)")
	return cmd
}

func newAgentUpdateCommand() *cobra.Command {
	var name, description, instructions, promptFile, model, status, runtimeID, command string
	var maxConcurrentTasks int32
	cmd := &cobra.Command{
		Use:   "update <agent-id>",
		Short: "Update a Multica agent",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			input := multicacli.AgentUpdateInput{}
			if cmd.Flags().Changed("name") {
				input.Name = &name
			}
			if cmd.Flags().Changed("description") {
				input.Description = &description
			}
			if cmd.Flags().Changed("instructions") && promptFile != "" {
				return clierrors.NewValidationError("--instructions and --instructions-file are mutually exclusive")
			}
			if cmd.Flags().Changed("instructions") {
				input.Instructions = &instructions
			}
			if promptFile != "" {
				value, err := delegation.ReadPromptFile(promptFile)
				if err != nil {
					return err
				}
				input.Instructions = &value
			}
			if cmd.Flags().Changed("model") {
				input.Model = &model
			}
			if cmd.Flags().Changed("status") {
				input.Status = &status
			}
			if cmd.Flags().Changed("runtime-id") {
				input.RuntimeID = &runtimeID
			}
			if cmd.Flags().Changed("max-concurrent-tasks") {
				input.MaxConcurrentTasks = &maxConcurrentTasks
			}
			if input.Name == nil && input.Description == nil && input.Instructions == nil && input.Model == nil && input.Status == nil && input.RuntimeID == nil && input.MaxConcurrentTasks == nil {
				return clierrors.NewValidationError("at least one update field is required")
			}
			result, err := multicacli.New(command).UpdateAgent(cmd.Context(), args[0], input)
			if err != nil {
				return err
			}
			return writeRaw(cmd.OutOrStdout(), result)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "New agent name")
	cmd.Flags().StringVar(&description, "description", "", "New agent description")
	cmd.Flags().StringVar(&instructions, "instructions", "", "New agent instructions")
	cmd.Flags().StringVar(&promptFile, "instructions-file", "", "Read new instructions from a UTF-8 file")
	cmd.Flags().StringVar(&model, "model", "", "New model identifier (empty clears it)")
	cmd.Flags().StringVar(&status, "status", "", "New agent status")
	cmd.Flags().StringVar(&runtimeID, "runtime-id", "", "New runtime UUID")
	cmd.Flags().Int32Var(&maxConcurrentTasks, "max-concurrent-tasks", 0, "New maximum concurrent runs")
	cmd.Flags().StringVar(&command, "multica-command", "", "Multica executable (default: multica)")
	return cmd
}

func newAgentDeleteCommand() *cobra.Command {
	var command string
	cmd := &cobra.Command{
		Use:     "delete <agent-id>",
		Aliases: []string{"archive"},
		Short:   "Archive (delete) a Multica agent",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := multicacli.New(command).ArchiveAgent(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			return writeRaw(cmd.OutOrStdout(), result)
		},
	}
	cmd.Flags().StringVar(&command, "multica-command", "", "Multica executable (default: multica)")
	return cmd
}

func newAgentRestoreCommand() *cobra.Command {
	var command string
	cmd := &cobra.Command{
		Use:   "restore <agent-id>",
		Short: "Restore an archived Multica agent",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := multicacli.New(command).RestoreAgent(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			return writeRaw(cmd.OutOrStdout(), result)
		},
	}
	cmd.Flags().StringVar(&command, "multica-command", "", "Multica executable (default: multica)")
	return cmd
}

func writeRaw(out io.Writer, result json.RawMessage) error {
	_, err := out.Write(append(result, '\n'))
	return err
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
			result, err := delegation.New(command).Delegate(cmd.Context(), delegation.DelegateInput{
				Issue:      args[0],
				AgentID:    agentID,
				ProjectID:  projectID,
				Status:     status,
				PromptFile: promptFile,
			})
			if err != nil {
				return err
			}
			return writeJSON(cmd.OutOrStdout(), result)
		},
	}
	cmd.Flags().StringVar(&agentID, "agent-id", "", "Multica agent UUID")
	cmd.Flags().StringVar(&projectID, "project", "", "Multica project UUID")
	cmd.Flags().StringVar(&status, "status", "todo", "Initial Multica issue status")
	cmd.Flags().StringVar(&promptFile, "prompt-file", "", "Sync this local prompt before delegation")
	cmd.Flags().StringVar(&command, "multica-command", "", "Multica executable (default: multica)")
	return cmd
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
