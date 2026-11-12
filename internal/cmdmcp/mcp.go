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
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Serve workctl tools over MCP stdio",
		Long: `Serve workctl tools over the Model Context Protocol on stdio.

The server exposes read tools for Linear and Multica plus the delegation
bridge (delegate_to_multica, sync_agent_prompt). Multica operations shell out
to the installed Multica CLI, so log in and select a workspace first.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			server := newServer(multicaCommand)
			return server.Serve(cmd.Context(), cmd.InOrStdin(), cmd.OutOrStdout())
		},
	}
	cmd.Flags().StringVar(&multicaCommand, "multica-command", "", "Multica executable (default: multica)")
	return cmd
}

func newServer(multicaCommand string) *mcpserver.Server {
	return &mcpserver.Server{
		Name:    "workctl",
		Version: version.Version,
		Tools:   toolset{multicaCommand: multicaCommand}.tools(),
	}
}
