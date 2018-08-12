package main

import (
	"context"
	"os"
	"os/signal"

	"github.com/hairyhenderson/go-onerng/cmd"
)

func main() {
	ctx := context.Background()

	ctx, cancel := context.WithCancel(ctx)
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	defer func() {
		signal.Stop(c)
		cancel()
	}()
	go func() {
		select {
		case <-c:
			cancel()
		case <-ctx.Done():
		}
	}()

	cmd.Execute(ctx)
}
