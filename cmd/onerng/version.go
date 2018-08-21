package main

import (
	"context"
	"fmt"

	"github.com/hairyhenderson/go-onerng"
	"github.com/spf13/cobra"
)

// versionCmd represents the version command
func versionCmd(ctx context.Context) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Display the OneRNG's hardware version",
		RunE: func(cmd *cobra.Command, args []string) error {
			o := &onerng.OneRNG{Path: opts.Device}
			version, err := o.Version(ctx)
			if err != nil {
				return err
			}
			fmt.Printf("OneRNG Hardware Version: %d\n", version)
			return nil
		},
	}
}
