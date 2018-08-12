package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile string
	opts    Config
)

func rootCmd(ctx context.Context) *cobra.Command {
	return &cobra.Command{
		Use:   "onerng [opts] COMMAND",
		Short: "Tool for the OneRNG open source hardware entropy generator",
		Long: `OneRNG is an open source hardware entropy generator in a USB dongle.

This tool can be used to verify that the OneRNG device operates
correctly, and that the firmware has not been tampered with.`,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceErrors = true
			cmd.SilenceUsage = true
			return nil
		},
	}
}

// Execute -
func Execute(ctx context.Context) {
	cmd := rootCmd(ctx)
	initConfig(ctx, cmd)
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func initConfig(ctx context.Context, cmd *cobra.Command) {
	cmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.go-onerng.yaml)")
	opts = Config{}
	cmd.PersistentFlags().StringVarP(&opts.Device, "device", "d", "/dev/ttyACM0", "the OneRNG device")

	cmd.AddCommand(
		verifyCmd(ctx),
		versionCmd(ctx),
		idCmd(ctx),
		flushCmd(ctx),
		imageCmd(ctx),
		initCmd(ctx),
		readCmd(ctx),
	)

	if cfgFile != "" { // enable ability to specify config file via flag
		viper.SetConfigFile(cfgFile)
	}

	viper.SetConfigName(".go-onerng")      // name of config file (without extension)
	viper.AddConfigPath(os.Getenv("HOME")) // adding home directory as first search path
	viper.AutomaticEnv()                   // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
}

// Config -
type Config struct {
	Device string
}
