package onerng

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"

	//nolint:staticcheck
	"golang.org/x/crypto/openpgp"
)

// Verify reads a signed firmware image, extracts the signature, and verifies
// it against the given public key.
//
// Details are printed to Stderr on success, otherwise an error is returned.
//
// The general logic is ported from the official onerng_verify.py script
// distributed alongside the OneRNG package.
func Verify(_ context.Context, image io.Reader, pubkey string) error {
	if err := readMagic(image); err != nil {
		return fmt.Errorf("failed to find magic number: %w", err)
	}
	length, version, err := readHeader(image)
	if err != nil {
		return fmt.Errorf("failed to read header: %w", err)
	}

	return readAndVerify(image, version, length, pubkey)
}

func read(r io.Reader, p []byte) error {
	n, err := r.Read(p)
	if err != nil {
		return fmt.Errorf("read failed: %w", err)
	}
	if n == 0 {
		return fmt.Errorf("short read")
	}

	return nil
}

// readHeader reads the header and returns the length and the version
func readHeader(r io.Reader) (length, version int, err error) {
	// read the length
	l := make([]byte, 3)
	if err := read(r, l); err != nil {
		return 0, 0, fmt.Errorf("failed reading length: %w", err)
	}
	length = int(l[0])
	length |= int(l[1]) << 8
	length |= int(l[2]) << 16

	// read the version
	l = make([]byte, 2)
	if err := read(r, l); err != nil {
		return 0, 0, fmt.Errorf("failed reading version: %w", err)
	}
	version = int(l[0])
	version |= int(l[1]) << 8

	// read the actual code size - we don't use it yet
	if err := read(r, make([]byte, 2)); err != nil {
		return 0, 0, fmt.Errorf("failed reading actual code size: %w", err)
	}

	return length, version, nil
}

// readMagic reads the input until the magic sequence 0xfeedbeef2014 is found
func readMagic(r io.Reader) error {
	for i := int8(0); i < 6; i++ {
		c := make([]byte, 1)
		if err := read(r, c); err != nil {
			return fmt.Errorf("couldn't read header: %w", err)
		}
		x := c[0]

		switch {
		case i == 0 && x == 0xfe,
			i == 1 && x == 0xed,
			i == 2 && x == 0xbe,
			i == 3 && x == 0xef,
			i == 4 && x == 0x20,
			i == 5 && x == 0x14:
			// as long as this looks like the magic number, keep going!
			continue
		default:
			i = 0
		}
	}

	return nil
}

func readAndVerify(image io.Reader, version, length int, pubkey string) error {
	signed, signature, err := parseImage(image, version, length)
	if err != nil {
		return err
	}

	signer, err := verifyImage(signed, signature, pubkey)
	if err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "firmware verification passed OK - version=%d\n", version)
	for _, id := range signer.Identities {
		fmt.Fprintf(os.Stderr, "signed by: %#v\n", id.Name)
		fmt.Fprintf(os.Stderr, "\tcreated: %q\n", id.SelfSignature.CreationTime)
		fmt.Fprintf(os.Stderr, "\tfingerprint: %X\n", signer.PrimaryKey.Fingerprint)
	}

	return nil
}

func verifyImage(signed, sig []byte, pubkey string) (signer *openpgp.Entity, err error) {
	// read public key
	keyring, err := openpgp.ReadArmoredKeyRing(bytes.NewBufferString(pubkey))
	if err != nil {
		return nil, err
	}

	// verify
	signer, err = openpgp.CheckDetachedSignature(
		keyring,
		bytes.NewBuffer(signed),
		bytes.NewBuffer(sig),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to verify firmware signature: %w", err)
	}

	return signer, nil
}

func parseImage(image io.Reader, version, length int) (signed, sig []byte, err error) {
	c := make([]byte, length)
	n, err := io.ReadAtLeast(image, c, length)
	if err != nil {
		return nil, nil, err
	}
	if n != length {
		return nil, nil, fmt.Errorf("bad image: wrong length: was %d, expected %d", n, length)
	}

	// determine end offset
	endOff := 0
	if version >= 3 {
		endOff = 680
	} else {
		endOff = 600
	}

	// signature length - 2 bytes between image and signature
	slen := int(c[length-endOff])
	slen |= int(c[length-endOff+1]) << 8

	// split last part into image (signed part) & signature
	signed = c[0 : length-endOff]
	sig = c[length-endOff+2 : length-endOff+2+slen]

	return signed, sig, nil
}
