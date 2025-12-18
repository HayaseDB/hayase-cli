package cmd

import (
	"context"
	"fmt"

	"github.com/charmbracelet/log"
	"github.com/hayasedb/hayase/internal/scraper"
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
	verbose bool
}

func ExecuteContext(ctx context.Context) error {
	return newCLI().execute(ctx)
}

func newCLI() *CLI {
	return &CLI{}
}

func (c *CLI) execute(_ context.Context) error {
	return c.buildCommand().Execute()
}

func (c *CLI) buildCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "hayase",
		Short:         "Anime streaming CLI",
		Version:       versionString,
		SilenceErrors: true,
		SilenceUsage:  true,
		PersistentPreRun: func(_ *cobra.Command, _ []string) {
			if c.verbose {
				log.SetLevel(log.DebugLevel)
			}
		},
		RunE: func(_ *cobra.Command, _ []string) error {
			return c.run()
		},
	}

	cmd.PersistentFlags().BoolVarP(&c.verbose, "verbose", "v", false, "Enable verbose output")

	return cmd
}

func (c *CLI) run() error {
	s := scraper.New()
	return tui.Run(s)
}
