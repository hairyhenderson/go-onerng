package onerng

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
)

// OneRNG - a OneRNG device
type OneRNG struct {
	Path string
}

func (o *OneRNG) cmd(ctx context.Context, d *os.File, c ...string) (err error) {
	for _, v := range c {
		_, err = d.WriteString(v)
		if err != nil {
			return errors.Wrapf(err, "Errored on command %s", v)
		}
		select {
		case <-ctx.Done():
			return nil
		default:
		}
	}
	return nil
}

// Version -
func (o *OneRNG) Version(ctx context.Context) (int, error) {
	d, err := os.OpenFile(o.Path, os.O_RDWR, 0600)
	if err != nil {
		return 0, err
	}
	defer d.Close()

	err = o.cmd(ctx, d, CmdPause)
	if err != nil {
		return 0, err
	}

	_, cancel := context.WithCancel(ctx)
	defer cancel()
	buf := make(chan string)
	errc := make(chan error, 1)
	go scan(ctx, d, buf, errc)

	err = o.cmd(ctx, d, CmdSilent, CmdVersion, CmdRun)
	if err != nil {
		return 0, err
	}

	verString := ""
loop:
	for {
		var b string
		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		case b = <-buf:
			if strings.HasPrefix(b, "Version ") {
				verString = b
				cancel()
				break loop
			}
		case err := <-errc:
			return 0, err
		}
	}

	err = o.cmd(ctx, d, CmdPause)
	if err != nil {
		return 0, err
	}

	n := strings.Replace(verString, "Version ", "", 1)
	version, err := strconv.Atoi(n)
	return version, err
}

// Identify -
func (o *OneRNG) Identify(ctx context.Context) (string, error) {
	d, err := os.OpenFile(o.Path, os.O_RDWR, 0600)
	if err != nil {
		return "", err
	}
	defer d.Close()

	_, cancel := context.WithCancel(ctx)
	defer cancel()
	buf := make(chan string)
	errc := make(chan error, 1)
	go scan(ctx, d, buf, errc)

	err = o.cmd(ctx, d, CmdSilent, CmdID, CmdRun)
	if err != nil {
		return "", err
	}

	idString := ""
loop:
	for {
		var b string
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case b = <-buf:
			if strings.HasPrefix(b, "___") {
				idString = b
				cancel()
				break loop
			}
		case err := <-errc:
			return "", err
		}
	}

	err = o.cmd(ctx, d, CmdPause)
	if err != nil {
		return "", err
	}

	return idString, err
}

// Flush -
func (o *OneRNG) Flush(ctx context.Context) error {
	d, err := os.OpenFile(o.Path, os.O_RDWR, 0600)
	if err != nil {
		return err
	}
	defer d.Close()

	_, cancel := context.WithCancel(ctx)
	defer cancel()
	err = o.cmd(ctx, d, CmdFlush)
	return err
}

// Image -
func (o *OneRNG) Image(ctx context.Context) ([]byte, error) {
	d, err := os.OpenFile(o.Path, os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	defer d.Close()

	err = o.cmd(ctx, d, CmdPause, CmdSilent)
	if err != nil {
		return nil, err
	}

	time.Sleep(2 * time.Second)

	_, cancel := context.WithCancel(ctx)
	defer cancel()
	buf := make(chan []byte)
	errc := make(chan error, 1)
	go stream(ctx, d, 4, buf, errc)

	err = o.cmd(ctx, d, CmdSilent, CmdImage, CmdRun)
	if err != nil {
		return nil, err
	}

	image := []byte{}
	zeros := 0
loop:
	for {
		var b []byte
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case b = <-buf:
			image = append(image, b...)
			for _, v := range b {
				if v == 0 {
					zeros++
				} else {
					zeros = 0
				}
				if zeros > 200 {
					break loop
				}
			}
		case err := <-errc:
			return nil, err
		}
	}

	err = o.cmd(ctx, d, CmdPause)
	if err != nil {
		return nil, err
	}

	return image, err
}

// Init - wait for the device to finish initializing and start returning data
func (o *OneRNG) Init(ctx context.Context) error {
	i := 0
	for ; i < 200; i++ {
		n, err := o.readData(ctx)
		if err != nil {
			return err
		}
		if n > 0 {
			break
		}
	}
	fmt.Fprintf(os.Stderr, "Initialized after %d loops\n", i)
	return nil
}

// readData - try to read some data from the RNG
func (o *OneRNG) readData(ctx context.Context) (int, error) {
	d, err := os.OpenFile(o.Path, os.O_RDWR, 0600)
	if err != nil {
		return 0, err
	}
	defer d.Close()

	_, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
	defer cancel()

	buf := make(chan []byte)
	errc := make(chan error, 1)
	go stream(ctx, d, 1, buf, errc)

	err = o.cmd(ctx, d, CmdAvalanche, CmdRun)
	if err != nil {
		return 0, err
	}

	// make sure we always end with a pause/silence/flush
	defer o.cmd(ctx, d, CmdPause, CmdSilent, CmdFlush)

	// blocking read from the channel, with a timeout (from context)
	select {
	case <-ctx.Done():
		return 0, ctx.Err()
	case b := <-buf:
		return len(b), nil
	case err := <-errc:
		return 0, err
	}
}

// stream from a file into a channel until an error is encountered, the channel
// is closed, or the context is cancelled.
func stream(ctx context.Context, d *os.File, bs int, buf chan []byte, errc chan error) {
	defer close(buf)
	defer close(errc)
	for {
		b := make([]byte, bs)
		n, err := io.ReadAtLeast(d, b, len(b))
		if err != nil {
			errc <- err
			return
		}
		if n < len(b) {
			errc <- errors.Errorf("unexpected short read - wanted %db, read %db", len(b), n)
			return
		}

		select {
		case <-ctx.Done():
			return
		case buf <- b:
		}
	}
}

func scan(ctx context.Context, d *os.File, buf chan string, errc chan error) {
	defer close(buf)
	defer close(errc)
	scanner := bufio.NewScanner(d)
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return
		case buf <- scanner.Text():
		}
	}
}
