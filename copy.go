package onerng

import (
	"context"
	"io"
	"os"
	"strings"
	"time"
)

type readerFunc func(p []byte) (n int, err error)

func (rf readerFunc) Read(p []byte) (n int, err error) { return rf(p) }

// io.CopyN/io.Copy with context support
func copyWithContext(ctx context.Context, dst io.Writer, src *os.File, n int64) (int64, error) {
	// allow 10 500ms timeouts, for a total of 5s. After this, it's probably worth just giving up
	allowedTimeouts := 10

	rf := func(p []byte) (int, error) {
		// we don't want reads to block forever, but we also don't want to time out immediately
		err := src.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
		if err != nil {
			return 0, err
		}

		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		default:
			n, err := src.Read(p)
			if allowedTimeouts > 0 {
				if err != nil && strings.HasSuffix(err.Error(), "i/o timeout") {
					allowedTimeouts--
					return n, nil
				}
			}
			return n, err
		}
	}

	if n < 0 {
		return io.Copy(dst, readerFunc(rf))
	}
	return io.CopyN(dst, readerFunc(rf), n)
}
