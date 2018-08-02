package cmd

import (
	"context"
	"fmt"

	"github.com/hairyhenderson/go-onerng"
	"github.com/spf13/cobra"
)

// idCmd represents the id command
func idCmd(ctx context.Context) *cobra.Command {
	return &cobra.Command{
		Use:   "id",
		Short: "Display the OneRNG's hardware id",
		RunE: func(cmd *cobra.Command, args []string) error {
			o := onerng.OneRNG{Path: opts.Device}
			id, err := o.Identify(ctx)
			if err != nil {
				return err
			}
			fmt.Printf("OneRNG Hardware ID: %s\n", id)
			return nil
		},
	}
}
