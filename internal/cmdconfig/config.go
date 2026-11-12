package cmdconfig

import (
	"fmt"

	"github.com/kongken/workctl/internal/config"
	"github.com/kongken/workctl/internal/errors"
	"github.com/spf13/cobra"
)

// New returns the config command group.
func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Configure the CLI",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
	cmd.AddCommand(newGetCommand())
	return cmd
}

func newGetCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "get <key>",
		Short: "Print a configuration value",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Surface the same parse errors as the root pre-run, including when the
			// command is invoked without it.
			if err := config.Load(); err != nil {
				return errors.NewValidationError(err.Error(), errors.WithCause(err))
			}
			val, ok := config.GetOption(args[0])
			if !ok {
				return errors.NewCliError(
					fmt.Sprintf("Failed to get config: %s", errors.NewNotFoundError("Config key", args[0]).Error()),
				)
			}
			fmt.Fprintln(cmd.OutOrStdout(), val)
			return nil
		},
	}
}
