package main

import (
	"context"
	"fmt"
	"os"

	"github.com/pkg/errors"

	"github.com/hairyhenderson/go-onerng"
	"github.com/spf13/cobra"
)

// imageCmd represents the image command
func imageCmd(ctx context.Context) *cobra.Command {
	var imgOut string
	cmd := &cobra.Command{
		Use:   "image",
		Short: "Dump the OneRNG's firmware image",
		RunE: func(cmd *cobra.Command, args []string) error {
			o := onerng.OneRNG{Path: opts.Device}
			err := o.Init(ctx)
			if err != nil {
				return errors.Wrapf(err, "init failed before image extraction")
			}
			image, err := o.Image(ctx)
			if err != nil {
				return err
			}
			var out *os.File
			if imgOut == "-" {
				out = os.Stdout
			} else {
				out, err = os.OpenFile(imgOut, os.O_RDWR|os.O_CREATE, 0644)
				if err != nil {
					return err
				}
			}
			n, err := out.Write(image)
			fmt.Fprintf(os.Stderr, "Wrote %db to %s\n", n, imgOut)
			return err
		},
	}
	cmd.Flags().StringVarP(&imgOut, "out", "o", "onerng.img", "output file for image (use - for stdout)")
	return cmd
}
