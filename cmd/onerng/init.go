package main

import (
	"context"

	"github.com/hairyhenderson/go-onerng"
	"github.com/spf13/cobra"
)

// initCmd represents the init command
func initCmd(ctx context.Context) *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Initialize the RNG",
		RunE: func(cmd *cobra.Command, args []string) error {
			o := onerng.OneRNG{Path: opts.Device}
			err := o.Init(ctx)
			return err
		},
	}
}
