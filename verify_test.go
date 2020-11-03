package onerng

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReadMagic(t *testing.T) {
	r := &bytes.Buffer{}
	err := readMagic(r)
	assert.Error(t, err)

	r = bytes.NewBufferString("abcdefg")
	err = readMagic(r)
	assert.Error(t, err)

	r = bytes.NewBuffer([]byte{0x00, 0x01, 0x02, 0xfe, 0xed, 0xbe, 0xee, 0xff})
	err = readMagic(r)
	assert.Error(t, err)

	r = bytes.NewBuffer([]byte{0x00, 0x01, 0x02, 0xfe, 0xed, 0xbe, 0xef, 0x20, 0x14})
	err = readMagic(r)
	assert.NoError(t, err)
}

func TestReadHeader(t *testing.T) {
	r := &bytes.Buffer{}
	_, _, err := readHeader(r)
	assert.Error(t, err)

	r = bytes.NewBuffer([]byte{0x00, 0x00, 0x00})
	_, _, err = readHeader(r)
	assert.Error(t, err)

	r = bytes.NewBuffer([]byte{0x00, 0x00, 0x00, 0x00})
	_, _, err = readHeader(r)
	assert.Error(t, err)

	r = bytes.NewBuffer([]byte{0x0f, 0x00, 0x00, 0x07, 0x00, 0xff, 0xee})
	l, v, err := readHeader(r)
	assert.NoError(t, err)
	assert.Equal(t, 15, l)
	assert.Equal(t, 7, v)

	r = bytes.NewBuffer([]byte{0x0f, 0xf0, 0x01, 0x07, 0x70, 0xff, 0xee, 0x11, 0x42})
	l, v, err = readHeader(r)
	assert.NoError(t, err)
	assert.Equal(t, 0x01f00f, l)
	assert.Equal(t, 0x7007, v)
}
