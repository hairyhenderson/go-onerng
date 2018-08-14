package onerng

import (
	"bufio"
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"fmt"
	"io"
	mrand "math/rand"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
)

// OneRNG - a OneRNG device
type OneRNG struct {
	Path   string
	device *os.File
}

// ReadMode -
type ReadMode uint32

func (o *OneRNG) cmd(ctx context.Context, c ...string) error {
	err := o.open()
	if err != nil {
		return err
	}
	for _, v := range c {
		_, err = o.device.WriteString(v)
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

func (o *OneRNG) open() (err error) {
	if o.device != nil {
		return nil
	}
	o.device, err = os.OpenFile(o.Path, os.O_RDWR, 0600)
	return err
}

func (o *OneRNG) close() error {
	if o.device == nil {
		return nil
	}
	err := o.device.Close()
	o.device = nil
	return err
}

// Version -
func (o *OneRNG) Version(ctx context.Context) (int, error) {
	err := o.open()
	if err != nil {
		return 0, err
	}
	defer o.close()

	err = o.cmd(ctx, CmdPause)
	if err != nil {
		return 0, err
	}

	_, cancel := context.WithCancel(ctx)
	defer cancel()
	buf := make(chan string)
	errc := make(chan error, 1)
	go o.scan(ctx, buf, errc)

	err = o.cmd(ctx, CmdSilent, CmdVersion, CmdRun)
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

	err = o.cmd(ctx, CmdPause)
	if err != nil {
		return 0, err
	}

	n := strings.Replace(verString, "Version ", "", 1)
	version, err := strconv.Atoi(n)
	return version, err
}

// Identify -
func (o *OneRNG) Identify(ctx context.Context) (string, error) {
	err := o.open()
	if err != nil {
		return "", err
	}
	defer o.close()

	_, cancel := context.WithCancel(ctx)
	defer cancel()
	buf := make(chan string)
	errc := make(chan error, 1)
	go o.scan(ctx, buf, errc)

	err = o.cmd(ctx, CmdSilent, CmdID, CmdRun)
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

	err = o.cmd(ctx, CmdPause)
	if err != nil {
		return "", err
	}

	return idString, err
}

// Flush -
func (o *OneRNG) Flush(ctx context.Context) error {
	err := o.open()
	if err != nil {
		return err
	}
	defer o.close()

	_, cancel := context.WithCancel(ctx)
	defer cancel()
	err = o.cmd(ctx, CmdFlush)
	return err
}

// Image -
func (o *OneRNG) Image(ctx context.Context) ([]byte, error) {
	err := o.open()
	if err != nil {
		return nil, err
	}
	defer o.close()

	err = o.cmd(ctx, CmdPause, CmdSilent)
	if err != nil {
		return nil, err
	}

	time.Sleep(2 * time.Second)

	_, cancel := context.WithCancel(ctx)
	defer cancel()
	buf := make(chan []byte)
	errc := make(chan error, 1)
	go o.stream(ctx, 4, buf, errc)

	err = o.cmd(ctx, CmdSilent, CmdImage, CmdRun)
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

	err = o.cmd(ctx, CmdPause)
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

// Read -
func (o *OneRNG) Read(ctx context.Context, out io.WriteCloser, n int64, flags ReadMode) (written int64, err error) {
	err = o.open()
	if err != nil {
		return 0, err
	}
	defer o.close()

	err = o.cmd(ctx, NoiseCommand(flags), CmdRun)
	if err != nil {
		return 0, err
	}

	defer o.cmd(ctx, CmdPause)

	written, err = copyWithContext(ctx, out, o.device, n)
	return written, err
}

// readData - try to read some data from the RNG
func (o *OneRNG) readData(ctx context.Context) (int, error) {
	err := o.open()
	if err != nil {
		return 0, err
	}
	defer o.close()

	_, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
	defer cancel()

	buf := make(chan []byte)
	errc := make(chan error, 1)
	go o.stream(ctx, 1, buf, errc)

	err = o.cmd(ctx, CmdAvalanche, CmdRun)
	if err != nil {
		return 0, err
	}

	// make sure we always end with a pause/silence/flush
	defer o.cmd(ctx, CmdPause, CmdSilent, CmdFlush)

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
func (o *OneRNG) stream(ctx context.Context, bs int, buf chan []byte, errc chan error) {
	err := o.open()
	if err != nil {
		errc <- err
		return
	}

	defer close(buf)
	defer close(errc)
	for {
		b := make([]byte, bs)
		n, err := io.ReadAtLeast(o.device, b, len(b))
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

func (o *OneRNG) scan(ctx context.Context, buf chan string, errc chan error) {
	err := o.open()
	if err != nil {
		errc <- err
		return
	}
	defer close(buf)
	defer close(errc)
	scanner := bufio.NewScanner(o.device)
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return
		case buf <- scanner.Text():
		}
	}
}

// NoiseCommand - returns the appropriate noise-generation command for the given flags
func NoiseCommand(flags ReadMode) string {
	num := strconv.Itoa(int(flags))
	return "cmd" + num + "\n"
}

type aesWhitener struct {
	out io.WriteCloser
}

// AESWhitener creates a "whitener" that wraps the provided writer. The random
// data that the OneRNG generates is sometimes a little  "too" random for some
// purposes (i.e. rngd), so this can be used to further mangle that data in non-
// predictable ways.
//
// This uses AES-128.
func (o *OneRNG) AESWhitener(ctx context.Context, out io.WriteCloser) (io.WriteCloser, error) {
	k, err := o.key(ctx)
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(k)
	if err != nil {
		return nil, err
	}

	// create a random IV with math/rand - doesn't need to be cryptographically-random
	iv := make([]byte, aes.BlockSize)
	_, err = mrand.Read(iv)
	if err != nil {
		return nil, err
	}

	stream := cipher.NewCFBEncrypter(block, iv)
	s := &cipher.StreamWriter{S: stream, W: out}
	return s, nil
}

func (o *OneRNG) key(ctx context.Context) ([]byte, error) {
	err := o.open()
	if err != nil {
		return []byte{}, err
	}

	buf := &bytes.Buffer{}

	err = o.cmd(ctx, CmdAvalanche, CmdRun)
	if err != nil {
		return []byte{}, err
	}
	defer o.cmd(ctx, CmdPause)
	// 16 bytes == AES-128
	_, err = copyWithContext(ctx, buf, o.device, 16)
	k := buf.Bytes()

	return k, err
}
