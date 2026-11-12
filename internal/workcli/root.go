package workcli

import (
	"context"
	"errors"
	"io"
	"strings"

	"github.com/kongken/workctl/internal/cli"
	"github.com/kongken/workctl/internal/cmdmcp"
	"github.com/kongken/workctl/internal/cmdmultica"
	"github.com/kongken/workctl/internal/cmdplane"
	clierrors "github.com/kongken/workctl/internal/errors"
	"github.com/spf13/cobra"
)

// NewRoot creates the single workctl command tree: Linear commands at the root
// plus the Plane group. No separate linear entrypoint is registered.
func NewRoot() *cobra.Command {
	root := cli.NewRoot()
	root.AddCommand(cli.NewLinearRoot(), cmdplane.New(), cmdmultica.New(), cmdmcp.New())
	return root
}

// Run renders one error at the process boundary. Plane command actions return errors.
func Run(ctx context.Context, args []string, in io.Reader, out, errOut io.Writer) int {
	root := NewRoot()
	root.SetArgs(args)
	root.SetIn(in)
	root.SetOut(out)
	root.SetErr(errOut)
	err := root.ExecuteContext(ctx)
	if err == nil {
		return 0
	}
	clierrors.HandleErrorTo(errOut, err, "")
	if errors.Is(err, context.Canceled) {
		return 130
	}
	var validation *clierrors.ValidationError
	if errors.As(err, &validation) || strings.HasPrefix(err.Error(), "unknown command ") {
		return 2
	}
	return 1
}
