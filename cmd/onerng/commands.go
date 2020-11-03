package main

import (
	"bytes"
	"fmt"
	"io"
	"math"
	"os"
	"time"

	"github.com/hairyhenderson/go-onerng"
	"github.com/spf13/cobra"
)

func createORNG(cmd *cobra.Command) *onerng.OneRNG {
	return &onerng.OneRNG{Path: cmd.Flag("device").Value.String()}
}

func idCmd(cmd *cobra.Command, args []string) error {
	o := createORNG(cmd)
	id, err := o.Identify(cmd.Context())
	if err != nil {
		return err
	}
	fmt.Printf("OneRNG Hardware ID: %s\n", id)

	return nil
}

func versionCmd(cmd *cobra.Command, args []string) error {
	o := createORNG(cmd)
	version, err := o.Version(cmd.Context())
	if err != nil {
		return err
	}
	fmt.Printf("OneRNG Hardware Version: %d\n", version)

	return nil
}

func flushCmd(cmd *cobra.Command, args []string) error {
	o := createORNG(cmd)

	return o.Flush(cmd.Context())
}

func initCmd(cmd *cobra.Command, args []string) error {
	o := createORNG(cmd)

	return o.Init(cmd.Context())
}

func verifyCmd(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	o := createORNG(cmd)
	err := o.Init(ctx)
	if err != nil {
		return fmt.Errorf("init failed before image verification: %w", err)
	}
	image, err := o.Image(ctx)
	if err != nil {
		return fmt.Errorf("image extraction failed before verification: %w", err)
	}
	err = onerng.Verify(ctx, bytes.NewBuffer(image), publicKey)

	return err
}

func imageCmd(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	o := createORNG(cmd)

	err := o.Init(ctx)
	if err != nil {
		return fmt.Errorf("init failed before image extraction: %w", err)
	}
	image, err := o.Image(ctx)
	if err != nil {
		return err
	}

	out := os.Stdout
	imgOut := cmd.Flag("out").Value.String()
	if imgOut != "-" {
		out, err = os.OpenFile(imgOut, os.O_RDWR|os.O_CREATE, 0o644)
		if err != nil {
			return err
		}
	}
	n, err := out.Write(image)
	fmt.Fprintf(os.Stderr, "Wrote %db to %s\n", n, imgOut)

	return err
}

func readFlags(cmd *cobra.Command) (onerng.NoiseMode, error) {
	disableAvalanche, err := cmd.Flags().GetBool("disable-avalanche")
	if err != nil {
		return 0, err
	}

	enableRF, err := cmd.Flags().GetBool("enable-rf")
	if err != nil {
		return 0, err
	}

	disableWhitener, err := cmd.Flags().GetBool("disable-whitener")
	if err != nil {
		return 0, err
	}

	// set flags based on commandline options
	flags := onerng.Default
	if disableAvalanche {
		flags |= onerng.DisableAvalanche
	}
	if enableRF {
		flags |= onerng.EnableRF
	}
	if disableWhitener {
		flags |= onerng.DisableWhitener
	}

	return flags, nil
}

//nolint:gocyclo
func readCmd(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	o := createORNG(cmd)
	err := o.Init(ctx)
	if err != nil {
		return fmt.Errorf("init failed before read: %w", err)
	}

	enableAESWhiten, err := cmd.Flags().GetBool("aes-whitener")
	if err != nil {
		return err
	}
	count, err := cmd.Flags().GetInt64("count")
	if err != nil {
		return err
	}
	flags, err := readFlags(cmd)
	if err != nil {
		return err
	}

	// waste some entropy...
	devNull, err := os.OpenFile("/dev/null", os.O_WRONLY, 0o200)
	if err != nil {
		return err
	}
	wasteAmount := 10240
	_, err = o.Read(ctx, devNull, int64(wasteAmount), flags)
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: entropy wasteage failed or incomplete, continuing anyway\n")
	}

	out := io.WriteCloser(os.Stdout)
	readOut := cmd.Flag("out").Value.String()
	if readOut != "-" {
		out, err = os.OpenFile(readOut, os.O_RDWR|os.O_CREATE, 0o644)
		if err != nil {
			return err
		}
	}

	if enableAESWhiten {
		out, err = o.AESWhitener(ctx, out)
		if err != nil {
			return err
		}
	}

	start := time.Now()
	written, err := o.Read(ctx, out, count, flags)
	delta := time.Since(start)
	rate := float64(written) / delta.Seconds()
	fmt.Fprintf(os.Stderr, "%s written in %s (%s/s)\n", humanizeBytes(float64(written)), delta, humanizeBytes(rate))

	return err
}

// humanizeBytes produces a human readable representation of an IEC size.
// Taken from github.com/dustin/go-humanize
//nolint:gomnd
func humanizeBytes(s float64) string {
	base := 1024.0
	sizes := []string{"B", "KiB", "MiB", "GiB", "TiB", "PiB", "EiB"}
	if s < 10 {
		return fmt.Sprintf("%f B", s)
	}

	e := math.Floor(math.Log(s) / math.Log(base))
	suffix := sizes[int(e)]
	val := math.Floor(s/math.Pow(base, e)*10+0.5) / 10
	f := "%.0f %s"
	if val < 10 {
		f = "%.1f %s"
	}

	return fmt.Sprintf(f, val, suffix)
}
