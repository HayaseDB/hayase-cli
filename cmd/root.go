package cmd

import (
	"context"
	"fmt"

	"github.com/charmbracelet/log"
	"github.com/hayasedb/hayase/internal/tui"
	"github.com/spf13/cobra"
)

var versionString = "dev"

func SetVersionInfo(version, commit, date string) {
	versionString = version
	if commit != "none" {
		versionString = fmt.Sprintf("%s\n  commit: %s\n  built:  %s", version, commit, date)
	}
}

type CLI struct {
	ctx     context.Context
	verbose bool
}

func ExecuteContext(ctx context.Context) error {
	return newCLI().execute(ctx)
}

func newCLI() *CLI {
	return &CLI{}
}

func (c *CLI) execute(ctx context.Context) error {
	c.ctx = ctx
	return c.buildCommand().Execute()
}

func (c *CLI) buildCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "hayase",
		Short:         "Anime streaming CLI",
		Version:       versionString,
		SilenceErrors: true,
		SilenceUsage:  true,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			if c.verbose {
				log.SetLevel(log.DebugLevel)
			}
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return c.run(cmd, args)
		},
	}

	cmd.PersistentFlags().BoolVarP(&c.verbose, "verbose", "v", false, "Enable verbose output")

	return cmd
}

func (c *CLI) run(_ *cobra.Command, _ []string) error {
	return tui.Run()
}
