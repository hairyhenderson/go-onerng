package onerng

import (
	"bufio"
	"context"
	"os"
	"strconv"
	"strings"

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

	_, cancel := context.WithCancel(ctx)
	defer cancel()
	buf := make(chan string)
	errc := make(chan error, 1)
	go func() {
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
	}()

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
	go func() {
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
	}()

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

	_, cancel := context.WithCancel(ctx)
	defer cancel()
	buf := make(chan []byte)
	errc := make(chan error, 1)
	go func() {
		defer close(buf)
		defer close(errc)
		b := make([]byte, 128)
		for {
			n, err := d.Read(b)
			if err != nil {
				errc <- err
				return
			}
			select {
			case <-ctx.Done():
				return
			case buf <- b:
			}
			if n == 0 {
				return
			}
		}
	}()

	err = o.cmd(ctx, d, CmdSilent, CmdImage, CmdRun)
	if err != nil {
		return nil, err
	}

	image := []byte{}
loop:
	for {
		var b []byte
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case b = <-buf:
			copy(image, b)
			if len(b) == 0 {
				break loop
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
