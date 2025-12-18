package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/charmbracelet/log"
	"github.com/hayasedb/hayase/internal/cache"
	"github.com/hayasedb/hayase/internal/history"
	"github.com/hayasedb/hayase/internal/paths"
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
	verbose    bool
	noCache    bool
	clearCache bool
	reset      bool
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
	cmd.Flags().BoolVarP(&c.noCache, "no-cache", "n", false, "Bypass cache and fetch fresh data")
	cmd.Flags().BoolVar(&c.clearCache, "clear-cache", false, "Clear all cached data and exit")
	cmd.Flags().BoolVar(&c.reset, "reset", false, "Delete all user data and exit")

	return cmd
}

func (c *CLI) run() error {
	if c.clearCache {
		if err := cache.Clear(); err != nil {
			return fmt.Errorf("clear cache: %w", err)
		}
		fmt.Println("Cache cleared")
		return nil
	}

	if c.reset {
		if err := os.RemoveAll(paths.DataDir()); err != nil {
			return fmt.Errorf("reset data: %w", err)
		}
		fmt.Println("User data reset")
		return nil
	}

	s := cache.NewScraper(scraper.New(), c.noCache)
	h := history.New()
	return tui.Run(s, h)
}
