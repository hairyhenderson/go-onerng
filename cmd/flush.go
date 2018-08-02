package cmd

import (
	"context"

	"github.com/hairyhenderson/go-onerng"
	"github.com/spf13/cobra"
)

// flushCmd represents the flush command
func flushCmd(ctx context.Context) *cobra.Command {
	return &cobra.Command{
		Use:   "flush",
		Short: "Flush the OneRNG's entropy pool",
		RunE: func(cmd *cobra.Command, args []string) error {
			o := onerng.OneRNG{Path: opts.Device}
			return o.Flush(ctx)
		},
	}
}
