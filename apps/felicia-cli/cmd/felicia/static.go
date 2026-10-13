package main

import (
	"context"
	"errors"

	"github.com/spf13/cobra"

	"github.com/azusachino/felicia/apps/felicia-providers/sqlite"
	publication "github.com/azusachino/felicia/apps/felicia-publication"
)

func newStaticCmd() *cobra.Command {
	staticCmd := &cobra.Command{
		Use:   "static",
		Short: "Static site compilation",
		RunE: func(_ *cobra.Command, _ []string) error {
			return errors.New("usage: felicia-cli static compile [options]")
		},
	}

	var (
		database  string
		mediaRoot string
		out       string
	)

	compileCmd := &cobra.Command{
		Use:   "compile",
		Short: "Compile static site publication from SQLite and media root",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := resolveWorkspaceDefaults(&database, &mediaRoot); err != nil {
				return err
			}
			repo, err := sqlite.Open(database)
			if err != nil {
				return err
			}
			defer func() { _ = repo.Close() }()
			report, err := publication.Publish(
				context.Background(),
				publication.StaticCompiler{},
				publication.Input{},
				repo,
				publication.FileMediaSource{Root: mediaRoot},
				out,
			)
			if err != nil {
				return err
			}
			return writeJSON(cmd.OutOrStdout(), report)
		},
	}

	compileCmd.Flags().StringVar(&database, "db", "", "SQLite database path")
	compileCmd.Flags().StringVar(&mediaRoot, "media-root", "", "private local media root")
	compileCmd.Flags().StringVar(&out, "out", "site", "static output directory")

	staticCmd.AddCommand(compileCmd)
	return staticCmd
}
