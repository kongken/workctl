package cli

import (
	"fmt"

	"github.com/kongken/workctl/internal/cmdapi"
	"github.com/kongken/workctl/internal/cmdauth"
	"github.com/kongken/workctl/internal/cmdconfig"
	"github.com/kongken/workctl/internal/cmdcycle"
	"github.com/kongken/workctl/internal/cmddocument"
	"github.com/kongken/workctl/internal/cmdinitiative"
	"github.com/kongken/workctl/internal/cmdinitiativeupdate"
	"github.com/kongken/workctl/internal/cmdissue"
	"github.com/kongken/workctl/internal/cmdlabel"
	"github.com/kongken/workctl/internal/cmdmarkdown"
	"github.com/kongken/workctl/internal/cmdmilestone"
	"github.com/kongken/workctl/internal/cmdproject"
	"github.com/kongken/workctl/internal/cmdprojectupdate"
	"github.com/kongken/workctl/internal/cmdschema"
	"github.com/kongken/workctl/internal/cmdteam"
	"github.com/kongken/workctl/internal/cmdtemplate"
	"github.com/kongken/workctl/internal/cmduser"
	"github.com/kongken/workctl/internal/config"
	"github.com/kongken/workctl/internal/credentials"
	clierrors "github.com/kongken/workctl/internal/errors"
	"github.com/kongken/workctl/internal/keyring"
	"github.com/kongken/workctl/internal/version"
	"github.com/spf13/cobra"
)

// NewRoot builds the single workctl command tree. Linear commands live at the
// root for compatibility; the workctl wrapper adds the explicit service groups.
func NewRoot() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "workctl",
		Version:       version.Version,
		Short:         "Manage Linear and Plane work from the command line.",
		SilenceErrors: true,
		SilenceUsage:  true,
		Long: `Manage Linear and Plane work from the command line.

Linear commands are available directly (for example "workctl issue view").
Plane commands are grouped under "plane".

Environment Variables:
  LINEAR_DEBUG=1              Show full error details including stack traces
  LINEAR_IGNORE_ENV_FILE=1    Skip loading .env files`,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Fprintln(cmd.OutOrStdout(), "Use --help to see available commands")
		},
	}
	configureRoot(cmd)
	registerLinearCommands(cmd)
	return cmd
}

// NewLinearRoot builds an explicit Linear command group for the workctl
// wrapper. It shares the same command constructors as the compatibility root.
func NewLinearRoot() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "linear",
		Short:         "Manage Linear workspaces and work items",
		SilenceErrors: true,
		SilenceUsage:  true,
	}
	registerLinearCommands(cmd)
	return cmd
}

func configureRoot(cmd *cobra.Command) {
	var workspace string

	cmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		// Plane has an independent configuration and credential store, so a
		// broken Linear config must not affect Plane commands.
		if inPlaneTree(cmd) {
			return nil
		}
		config.SetCLIWorkspace(workspace)
		if err := config.Load(); err != nil {
			return clierrors.NewValidationError(err.Error(), clierrors.WithCause(err))
		}
		keyring.UseOS()
		_, _ = credentials.Load()
		return nil
	}
	cmd.PersistentFlags().StringVar(&workspace, "workspace", "", "Target workspace (uses credentials)")
	cmd.SetFlagErrorFunc(func(_ *cobra.Command, err error) error {
		return clierrors.NewValidationError(err.Error())
	})
}

func registerLinearCommands(cmd *cobra.Command) {
	issueCmd := cmdissue.New()
	issueCmd.Aliases = []string{"i"}

	teamCmd := cmdteam.New()
	teamCmd.Aliases = []string{"t"}

	userCmd := cmduser.New()
	userCmd.Aliases = []string{"u"}

	projectCmd := cmdproject.New()
	projectCmd.Aliases = []string{"p"}

	projectUpdateCmd := cmdprojectupdate.New()
	projectUpdateCmd.Aliases = []string{"pu"}

	cycleCmd := cmdcycle.New()
	cycleCmd.Aliases = []string{"cy"}

	milestoneCmd := cmdmilestone.New()
	milestoneCmd.Aliases = []string{"m"}

	initiativeCmd := cmdinitiative.New()
	initiativeCmd.Aliases = []string{"init"}

	initiativeUpdateCmd := cmdinitiativeupdate.New()
	initiativeUpdateCmd.Aliases = []string{"iu"}

	labelCmd := cmdlabel.New()
	labelCmd.Aliases = []string{"l"}

	configCmd := cmdconfig.New()
	configCmd.Aliases = []string{"configure"}

	cmd.AddCommand(
		cmdauth.New(),
		issueCmd,
		teamCmd,
		userCmd,
		projectCmd,
		projectUpdateCmd,
		cycleCmd,
		milestoneCmd,
		initiativeCmd,
		initiativeUpdateCmd,
		labelCmd,
		cmdtemplate.New(),
		cmddocument.New(),
		configCmd,
		cmdschema.New(),
		cmdapi.New(),
		cmdmarkdown.New(),
	)
}

// inPlaneTree reports whether cmd is the plane group or one of its descendants.
func inPlaneTree(cmd *cobra.Command) bool {
	for c := cmd; c != nil; c = c.Parent() {
		if c.Name() == "plane" {
			return true
		}
	}
	return false
}
