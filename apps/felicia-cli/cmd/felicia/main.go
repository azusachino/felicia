// Package main implements the user-facing Felicia CLI.
package main

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
)

func main() {
	if err := execute(os.Args[1:], os.Stdout); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "felicia:", err)
		os.Exit(1)
	}
}

func execute(args []string, output io.Writer) error {
	cmd := newRootCmd(output)
	cmd.SetArgs(args)
	return cmd.Execute()
}

func newRootCmd(output io.Writer) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:           "felicia-cli",
		Short:         "Felicia CLI for travel journal intake, publication, and package management",
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(_ *cobra.Command, _ []string) error {
			return errors.New("usage: felicia-cli package|import|journey|static")
		},
	}
	rootCmd.SetOut(output)
	rootCmd.SetErr(output)

	rootCmd.AddCommand(newPackageCmd())
	rootCmd.AddCommand(newImportCmd())
	rootCmd.AddCommand(newJourneyCmd())
	rootCmd.AddCommand(newStaticCmd())

	return rootCmd
}
