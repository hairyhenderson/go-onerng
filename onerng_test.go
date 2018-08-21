package onerng

import (
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

// func TestNoiseCommand(t *testing.T) {
// 	fmt.Println(DisableWhitener)
// 	fmt.Println(EnableRF)
// 	fmt.Println(DisableAvalanche)

// 	fmt.Println(Default)
// 	fmt.Println(Silent)

// 	testdata := []struct {
// 		flags NoiseMode
// 		cmd   string
// 	}{
// 		{Default, "cmd0\n"},
// 		{DisableWhitener, "cmd1\n"},
// 		{EnableRF, "cmd2\n"},
// 		{EnableRF | DisableWhitener, "cmd3\n"},
// 		{DisableAvalanche, "cmd4\n"},
// 		{DisableAvalanche | DisableWhitener, "cmd5\n"},
// 		{DisableAvalanche | EnableRF, "cmd6\n"},
// 		{DisableAvalanche | EnableRF | DisableWhitener, "cmd7\n"},
// 	}
// 	for _, d := range testdata {
// 		assert.Equal(t, d.cmd, noiseCommand(d.flags), d.cmd, d.flags)
// 	}
// }

type fakeDev struct {
	closed bool
	rbuf   *bytes.Buffer
	wbuf   *bytes.Buffer
}

func (d *fakeDev) reset() {
	d.closed = false
	d.rbuf.Reset()
	d.wbuf.Reset()
}

func (d *fakeDev) Close() error {
	d.closed = true
	return nil
}

func (d *fakeDev) Read(b []byte) (int, error) {
	return d.rbuf.Read(b)
}

func (d *fakeDev) Write(b []byte) (int, error) {
	return d.wbuf.Write(b)
}

func TestCmd(t *testing.T) {
	d := &fakeDev{rbuf: &bytes.Buffer{}, wbuf: &bytes.Buffer{}}
	o := &OneRNG{Path: "/dev/null", device: d}
	ctx, cancel := context.WithCancel(context.Background())
	err := o.cmd(ctx, "foo", "bar")
	assert.NoError(t, err)
	assert.Equal(t, "foobar", d.wbuf.String())

	d.reset()
	cancel()
	err = o.cmd(ctx, "foo", "bar")
	assert.NoError(t, err)
	assert.Equal(t, "foo", d.wbuf.String())
}

func TestClose(t *testing.T) {
	d := &fakeDev{}
	o := &OneRNG{Path: "/dev/null", device: d}
	err := o.close()
	assert.NoError(t, err)
	assert.True(t, d.closed)

	o = &OneRNG{Path: "/dev/null", device: nil}
	err = o.close()
	assert.NoError(t, err)
	assert.Nil(t, o.device)
}

func TestVersion(t *testing.T) {
	d := &fakeDev{
		wbuf: &bytes.Buffer{},
		rbuf: bytes.NewBufferString("dfoawiuhf98h9inf2oifoi2jr\n" +
			"dfkjawflihjwfoiuh2rliu13he487631487645t98y23rtoqu3rbno9q34htgfv\n" +
			"\r\nVersion 3\r\nas;dlfjaw;oihf2ih2o3iuf2ofnlo2jnlfuhf2iou\n\n"),
	}
	o := &OneRNG{Path: "/dev/null", device: d}
	ctx := context.Background()
	v, err := o.Version(ctx)
	assert.NoError(t, err)
	assert.Equal(t, "cmdo\ncmd4\ncmdv\ncmdO\ncmdo\n", d.wbuf.String())
	assert.Equal(t, 3, v)
}
