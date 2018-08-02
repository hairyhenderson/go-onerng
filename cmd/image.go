package cmd

import (
	"context"
	"fmt"

	"github.com/hairyhenderson/go-onerng"
	"github.com/spf13/cobra"
)

// imageCmd represents the image command
func imageCmd(ctx context.Context) *cobra.Command {
	return &cobra.Command{
		Use:   "image",
		Short: "Dump the OneRNG's firmware image",
		RunE: func(cmd *cobra.Command, args []string) error {
			o := onerng.OneRNG{Path: opts.Device}
			image, err := o.Image(ctx)
			if err != nil {
				return err
			}
			fmt.Printf("%q\n", image)
			return nil
		},
	}
}
