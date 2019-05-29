/*
Package onerng provides functions to help interface with the OneRNG hardware RNG.

See http://onerng.info for information about the device, and see especially
http://www.moonbaseotago.com/onerng/theory.html for the theory of operation.

To use this package, you must first plug the OneRNG into an available USB port,
and your OS should auto-detect the device as a USB serial modem. On Linux, you
may need to load the cdc_acm module.

Once you know which device file points to the OneRNG, you can instantiate a
*OneRNG struct instance. All communication with the OneRNG is done through
this instance.

	o := &OneRNG{Path: "/dev/ttyACM0"}
	version, err := o.Version(context.TODO())
	if err != nil {
		return err
	}
	fmt.Printf("version is %d\n", version)

Reading data from the OneRNG can be done with the Read function:

	o := &OneRNG{Path: "/dev/ttyACM0"}
	_, err = o.Read(context.TODO(), os.Stdout, -1, EnableRF | DisableWhitener)
	if err != nil {
		return err
	}
*/
package onerng

import (
	"bufio"
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
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
	device io.ReadWriteCloser
}

// cmd sends one or more commands to the OneRNG. The device is not closed on
// completion, as it's usually being read from simultaneously.
func (o *OneRNG) cmd(ctx context.Context, c ...string) error {
	err := o.open()
	if err != nil {
		return err
	}
	for _, v := range c {
		_, err = o.device.Write([]byte(v))
		if err != nil {
			return errors.Wrapf(err, "Errored on command %q", v)
		}
		select {
		case <-ctx.Done():
			return nil
		default:
		}
	}
	return nil
}

// open the OneRNG device for read/write, if it hasn't already been opened.
// Access it as o.device.
func (o *OneRNG) open() (err error) {
	if o.device != nil {
		return nil
	}
	o.device, err = os.OpenFile(o.Path, os.O_RDWR, 0600)
	return err
}

// close the OneRNG device if it hasn't already been closed
func (o *OneRNG) close() error {
	if o.device == nil {
		return nil
	}
	err := o.device.Close()
	o.device = nil
	return err
}

// Version - query the OneRNG for its hardware version
func (o *OneRNG) Version(ctx context.Context) (int, error) {
	err := o.open()
	if err != nil {
		return 0, err
	}
	defer o.close()

	err = o.cmd(ctx, cmdPause)
	if err != nil {
		return 0, err
	}

	_, cancel := context.WithCancel(ctx)
	defer cancel()
	buf := make(chan string)
	errc := make(chan error, 1)
	go o.scan(ctx, buf, errc)

	err = o.cmd(ctx, noiseCommand(Silent), cmdVersion, cmdRun)
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

	err = o.cmd(ctx, cmdPause)
	if err != nil {
		return 0, err
	}

	n := strings.Replace(verString, "Version ", "", 1)
	version, err := strconv.Atoi(n)
	return version, err
}

// Identify - query the OneRNG for its ID
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

	err = o.cmd(ctx, noiseCommand(Silent), cmdID, cmdRun)
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

	err = o.cmd(ctx, cmdPause)
	if err != nil {
		return "", err
	}

	return idString, err
}

// Flush the OneRNG's entropy pool
func (o *OneRNG) Flush(ctx context.Context) error {
	err := o.open()
	if err != nil {
		return err
	}
	defer o.close()

	_, cancel := context.WithCancel(ctx)
	defer cancel()
	err = o.cmd(ctx, cmdFlush)
	return err
}

// Image extracts the firmware image. This image is padded with random data to
// either 128Kb or 256Kb (depending on hardware), and signed.
//
// See also the Verify function.
func (o *OneRNG) Image(ctx context.Context) ([]byte, error) {
	err := o.open()
	if err != nil {
		return nil, err
	}
	defer o.close()

	err = o.cmd(ctx, cmdPause, noiseCommand(Silent))
	if err != nil {
		return nil, err
	}

	time.Sleep(2 * time.Second)

	_, cancel := context.WithCancel(ctx)
	defer cancel()
	buf := make(chan []byte)
	errc := make(chan error, 1)
	go o.stream(ctx, 4, buf, errc)

	err = o.cmd(ctx, noiseCommand(Silent), cmdImage, cmdRun)
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

	err = o.cmd(ctx, cmdPause)
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
	// fmt.Fprintf(os.Stderr, "Initialized after %d loops\n", i)
	return nil
}

// Read n bytes of data from the OneRNG into the given Writer. Set flags to
// configure the OneRNG's. Set n to -1 to continuously read until an error is
// encountered, or the context is cancelled.
//
// The OneRNG device will be closed when the operation completes.
func (o *OneRNG) Read(ctx context.Context, out io.Writer, n int64, flags NoiseMode) (written int64, err error) {
	err = o.open()
	if err != nil {
		return 0, err
	}
	defer o.close()

	err = o.cmd(ctx, noiseCommand(flags), cmdRun)
	if err != nil {
		return 0, err
	}

	defer o.cmd(ctx, cmdPause)

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

	err = o.cmd(ctx, noiseCommand(Default), cmdRun)
	if err != nil {
		return 0, err
	}

	// make sure we always end with a pause/silence/flush
	defer o.cmd(ctx, cmdPause, noiseCommand(Silent), cmdFlush)

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

const (
	// cmdVersion - print firmware version (as "Version n")
	cmdVersion = "cmdv\n"
	// cmdFlush - flush entropy pool
	cmdFlush = "cmdw\n"
	// cmdImage - extract the signed firmware image for verification
	cmdImage = "cmdX\n"
	// cmdID - print hardware ID
	cmdID = "cmdI\n"
	// cmdRun - start the task
	cmdRun = "cmdO\n"
	// cmdPause - stop/pause the task
	cmdPause = "cmdo\n"
)

// AESWhitener creates a "whitener" that wraps the provided writer. The random
// data that the OneRNG generates is sometimes a little "too" random for some
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

	err = o.cmd(ctx, noiseCommand(Default), cmdRun)
	if err != nil {
		return []byte{}, err
	}
	defer o.cmd(ctx, cmdPause)
	// 16 bytes == AES-128
	_, err = copyWithContext(ctx, buf, o.device, 16)
	k := buf.Bytes()

	return k, err
}

// NoiseMode represents the different noise-generation modes available to the OneRNG
type NoiseMode uint32

const (
	// DisableWhitener - Disable the on-board CRC16 generator - no effect if both noise generators are disabled
	DisableWhitener NoiseMode = 1 << iota
	// EnableRF - Enable noise generation from RF
	EnableRF
	// DisableAvalanche - Disable noise generation from the Avalanche Diode
	DisableAvalanche

	// Default mode - Avalanche enabled, RF disabled, Whitener enabled.
	Default NoiseMode = 0
	// Silent - a convenience - everything disabled
	Silent NoiseMode = 4
)

// noiseCommand converts the given mode to the appropriate command to send to the OneRNG
func noiseCommand(flags NoiseMode) string {
	num := strconv.Itoa(int(flags))
	return "cmd" + num + "\n"
}

type readerFunc func(p []byte) (n int, err error)

func (rf readerFunc) Read(p []byte) (n int, err error) { return rf(p) }

// io.CopyN/io.Copy with cancellation support
func copyWithContext(ctx context.Context, dst io.Writer, src io.Reader, n int64) (int64, error) {
	// allow 10 500ms timeouts, for a total of 5s. After this, it's probably worth just giving up
	allowedTimeouts := 10

	rf := func(p []byte) (int, error) {
		if f, ok := src.(*os.File); ok {
			// I don't want reads to block forever, but I also don't want to time out immediately
			err := f.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
			if err != nil {
				return 0, err
			}
		}

		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		default:
			n, err := src.Read(p)
			if allowedTimeouts > 0 {
				if err != nil && os.IsTimeout(err) {
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
