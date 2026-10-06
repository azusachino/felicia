package main

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	journeypackage "github.com/azusachino/felicia/apps/felicia-core/journeypackage"
	"github.com/azusachino/felicia/apps/felicia-runtime/importer"
)

func newPackageCmd() *cobra.Command {
	packageCmd := &cobra.Command{
		Use:   "package",
		Short: "Validate and inspect journey packages",
		RunE: func(_ *cobra.Command, _ []string) error {
			return errors.New("usage: felicia-cli package validate <journey.zip>")
		},
	}

	validateCmd := &cobra.Command{
		Use:   "validate <journey.zip>",
		Short: "Validate a journey package archive and its manifest",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			pkg, err := journeypackage.Read(args[0])
			if err != nil {
				return err
			}
			document, err := importer.DecodePackage(pkg)
			if err != nil {
				return fmt.Errorf("validate package contents: %w", err)
			}
			registry, err := importer.DefaultRegistry()
			if err != nil {
				return fmt.Errorf("load kind registry: %w", err)
			}
			if err := importer.ValidatePackageDocument(document, registry); err != nil {
				return fmt.Errorf("validate package contents: %w", err)
			}
			return writeJSON(cmd.OutOrStdout(), map[string]any{
				"package_id":     pkg.Manifest.PackageID,
				"schema_version": pkg.Manifest.SchemaVersion,
				"files":          len(pkg.Files),
				"journeys":       1,
				"candidates":     len(document.Stops),
				"mementos":       len(document.Mementos),
				"photos":         len(document.Photos),
			})
		},
	}

	packageCmd.AddCommand(validateCmd)
	return packageCmd
}
