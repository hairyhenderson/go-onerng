package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/pkg/errors"

	"github.com/dustin/go-humanize"
	"github.com/hairyhenderson/go-onerng"
	"github.com/spf13/cobra"
)

func readCmd(ctx context.Context) *cobra.Command {
	readOut := ""
	disableAvalanche := false
	enableRF := false
	disableWhitener := false
	count := int64(-1)
	cmd := &cobra.Command{
		Use:   "read",
		Short: "read some random data from the OneRNG",
		RunE: func(cmd *cobra.Command, args []string) error {
			o := onerng.OneRNG{Path: opts.Device}
			err := o.Init(ctx)
			if err != nil {
				return errors.Wrapf(err, "init failed")
			}
			var out *os.File
			if readOut == "-" {
				out = os.Stdout
			} else {
				out, err = os.OpenFile(readOut, os.O_RDWR|os.O_CREATE, 0644)
				if err != nil {
					return err
				}
			}
			flags := onerng.ReadMode(onerng.Default)
			if disableAvalanche {
				flags |= onerng.DisableAvalanche
			}
			if enableRF {
				flags |= onerng.EnableRF
			}
			if disableWhitener {
				flags |= onerng.DisableWhitener
			}
			start := time.Now()
			written, err := o.Read(ctx, out, count, flags)
			delta := time.Since(start)
			rate := float64(written) / delta.Seconds()
			fmt.Fprintf(os.Stderr, "%s written in %s (%s/s)\n", humanize.Bytes(uint64(written)), delta, humanize.Bytes(uint64(rate)))
			return err
		},
	}
	cmd.Flags().StringVarP(&readOut, "out", "o", "-", "output file for data (use - for stdout)")
	cmd.Flags().BoolVar(&disableAvalanche, "disable-avalanche", false, "Disable noise generation from the Avalanche Diode")
	cmd.Flags().BoolVar(&enableRF, "enable-rf", false, "Enable noise generation from RF")
	cmd.Flags().BoolVar(&disableWhitener, "disable-whitener", false, "Disable the on-board CRC16 generator")
	cmd.Flags().Int64VarP(&count, "count", "n", -1, "Read only N bytes (use -1 for unlimited)")
	return cmd
}
