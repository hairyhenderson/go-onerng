package onerng

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"

	"github.com/pkg/errors"
	"golang.org/x/crypto/openpgp"
)

//	looks like:
//		fe ed be ef 20 14	- magic number
//		00 00 04 		- len (little endian)
//		00 00			- version (little endian)
//		image			- len bytes of image
type fwImage struct {
	magic   [6]byte      // must be 0xfeedbeef2014
	length  [3]byte      // LE
	version [2]byte      // LE
	_       [1]byte      //
	fullimg [262144]byte // full image
	// image   [261536]byte // 256kb minus end offset
	// slen    [2]byte      // signature length
	// sig     [543]byte    // signature
	// endOff  [608]byte
}

// Verify - this is a more-or-less straight port from the onerng_verify.py script
// distributed with the OneRNG package
func Verify(ctx context.Context, image io.Reader, pubkey string) error {
	var x byte
	length := 0
	version := 0
	state := int8(0)
	for {
		c := make([]byte, 1)
		n, err := io.ReadAtLeast(image, c, len(c))
		if err != nil {
			return err
		}
		if n == 0 {
			return errors.Errorf("Short image")
		}
		x = c[0]

		// 1. read magic (6 bytes)
		if state == 0 && x == 0xfe {
			state++
		} else if state == 1 && x == 0xed {
			state++
		} else if state == 2 && x == 0xbe {
			state++
		} else if state == 3 && x == 0xef {
			state++
		} else if state == 4 && x == 0x20 {
			state++
		} else if state == 5 && x == 0x14 {
			state++
			// 2. read length (3 bytes)
		} else if state == 6 {
			length = int(x)
			state++
		} else if state == 7 {
			length = length | (int(x) << 8)
			state++
		} else if state == 8 {
			length = length | (int(x) << 16)
			state++
			// 3. read version (2 bytes)
		} else if state == 9 {
			version = int(x)
			state++
		} else if state == 10 {
			version = version | (int(x) << 8)
			state++
		} else if state == 11 {
			// skip a padding byte I guess...
			state++
		} else if state == 12 {
			// 4. read image
			c := make([]byte, length)
			n, err := io.ReadAtLeast(image, c, length)
			if err != nil {
				return err
			}
			if n != length {
				return errors.Errorf("Bad image")
			}

			// determine end offset
			endOff := 0
			if version >= 3 {
				endOff = 680
			} else {
				endOff = 600
			}

			// read length of signature - 2 bytes between image and signature
			x = c[length-endOff]
			klen := int(x)
			x = c[length-endOff+1]
			klen = klen | (int(x) << 8)

			// split last part into image (signed part) & signature
			signature := bytes.NewBuffer(c[length-endOff+2 : length-endOff+2+klen])
			signed := bytes.NewBuffer(c[0 : length-endOff])

			// leftovers := c[length-endOff+2+klen : length]
			// fmt.Fprintf(os.Stderr, "leftovers (%d):\n%#x\n", len(leftovers), leftovers)

			// read public key
			keyring, err := openpgp.ReadArmoredKeyRing(bytes.NewBufferString(pubkey))
			if err != nil {
				return err
			}

			// verify
			signer, err := openpgp.CheckDetachedSignature(keyring, signed, signature)
			if err != nil {
				return errors.Wrapf(err, "failed to verify firmware signature")
			}

			fmt.Fprintf(os.Stderr, "firmware verification passed OK - version=%d\n", version)
			for _, id := range signer.Identities {
				fmt.Fprintf(os.Stderr, "signed by: %#v\n", id.Name)
				fmt.Fprintf(os.Stderr, "\tcreated: %q\n", id.SelfSignature.CreationTime)
				fmt.Fprintf(os.Stderr, "\tfingerprint: %X\n", signer.PrimaryKey.Fingerprint)
			}

			break
		} else {
			// something didn't line up, so we need to begin again...
			state = 0
		}
	}
	return nil
}
