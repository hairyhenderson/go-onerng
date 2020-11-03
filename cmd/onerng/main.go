// onerng: OneRNG hardware random number generation utility

package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/hairyhenderson/go-onerng/version"
	"github.com/spf13/cobra"
)

func commands() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "onerng [opts] COMMAND",
		Short: "Tool for the OneRNG open source hardware entropy generator",
		Long: `OneRNG is an open source hardware entropy generator in a USB dongle.

This tool can be used to verify that the OneRNG device operates
correctly, and that the firmware has not been tampered with.`,
		Version: version.Version,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceErrors = true
			cmd.SilenceUsage = true

			return nil
		},
	}
	cmd.PersistentFlags().StringP("device", "d", "/dev/ttyACM0", "the OneRNG device")

	flush := &cobra.Command{
		Use:   "flush",
		Short: "Flush the OneRNG's entropy pool",
		RunE:  flushCmd,
	}
	id := &cobra.Command{
		Use:   "id",
		Short: "Display the OneRNG's hardware id",
		RunE:  idCmd,
	}
	init := &cobra.Command{
		Use:   "init",
		Short: "Initialize the RNG",
		RunE:  initCmd,
	}
	verify := &cobra.Command{
		Use:   "verify",
		Short: "Verify that OneRNG's firmware has not been tampered with.",
		RunE:  verifyCmd,
	}
	version := &cobra.Command{
		Use:   "version",
		Short: "Display the OneRNG's hardware version",
		RunE:  versionCmd,
	}
	image := &cobra.Command{
		Use:   "image",
		Short: "Dump the OneRNG's firmware image",
		RunE:  imageCmd,
	}
	image.Flags().StringP("out", "o", "onerng.img", "output file for image (use - for stdout)")

	read := &cobra.Command{
		Use:   "read",
		Short: "read some random data from the OneRNG",
		RunE:  readCmd,
	}
	read.Flags().StringP("out", "o", "-", "output file for data (use - for stdout)")
	read.Flags().Bool("disable-avalanche", false, "Disable noise generation from the Avalanche Diode")
	read.Flags().Bool("enable-rf", false, "Enable noise generation from RF")
	read.Flags().Bool("disable-whitener", false, "Disable the on-board CRC16 generator")
	read.Flags().Int64P("count", "n", -1, "Read only N bytes (use -1 for unlimited)")
	read.Flags().Bool("aes-whitener", true, "encrypt with AES-128 to 'whiten' the input stream with a random key obtained from the OneRNG")

	cmd.AddCommand(flush, id, init, image, read, verify, version)

	return cmd
}

func main() {
	returncode := 0
	defer func() { os.Exit(returncode) }()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	defer signal.Stop(c)

	go func() {
		select {
		case <-c:
			cancel()
		case <-ctx.Done():
		}
	}()

	cmd := commands()
	if err := cmd.ExecuteContext(ctx); err != nil {
		fmt.Fprintln(os.Stderr, err)
		returncode = 1
	}
}
