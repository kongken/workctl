// Package cmdmcp exposes workctl's Linear and Multica bridge operations as a
// Model Context Protocol server. The command is a thin adapter: the protocol
// lives in internal/mcpserver and the tool handlers call existing packages.
package cmdmcp

import (
	"github.com/kongken/workctl/internal/mcpserver"
	"github.com/kongken/workctl/internal/version"
	"github.com/spf13/cobra"
)

// New creates the mcp command group.
func New() *cobra.Command {
	root := &cobra.Command{
		Use:   "mcp",
		Short: "Run a Model Context Protocol server",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}
	root.AddCommand(newServeCommand())
	return root
}

func newServeCommand() *cobra.Command {
	var multicaCommand string
	var readOnly bool
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Serve workctl tools over MCP stdio",
		Long: `Serve workctl tools over the Model Context Protocol on stdio.

The server exposes Linear reads, Multica agent and issue management, and the
delegation bridge (delegate_to_multica, sync_agent_prompt). Multica operations
shell out to the installed Multica CLI, so log in and select a workspace first.

Pass --read-only to expose only tools that read state.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			server := newServer(multicaCommand, readOnly)
			return server.Serve(cmd.Context(), cmd.InOrStdin(), cmd.OutOrStdout())
		},
	}
	cmd.Flags().StringVar(&multicaCommand, "multica-command", "", "Multica executable (default: multica)")
	cmd.Flags().BoolVar(&readOnly, "read-only", false, "Expose only tools that read state")
	return cmd
}

func newServer(multicaCommand string, readOnly bool) *mcpserver.Server {
	tools := toolset{multicaCommand: multicaCommand}.tools()
	if readOnly {
		visible := tools[:0]
		for _, tool := range tools {
			if !tool.Write {
				visible = append(visible, tool)
			}
		}
		tools = visible
	}
	return &mcpserver.Server{
		Name:    "workctl",
		Version: version.Version,
		Tools:   tools,
	}
}
