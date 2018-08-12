package onerng

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNoiseCommand(t *testing.T) {
	fmt.Println(DisableWhitener)
	fmt.Println(EnableRF)
	fmt.Println(DisableAvalanche)

	fmt.Println(Default)
	fmt.Println(Silent)

	testdata := []struct {
		flags ReadMode
		cmd   string
	}{
		{Default, "cmd0\n"},
		{DisableWhitener, "cmd1\n"},
		{EnableRF, "cmd2\n"},
		{EnableRF | DisableWhitener, "cmd3\n"},
		{DisableAvalanche, "cmd4\n"},
		{DisableAvalanche | DisableWhitener, "cmd5\n"},
		{DisableAvalanche | EnableRF, "cmd6\n"},
		{DisableAvalanche | EnableRF | DisableWhitener, "cmd7\n"},
	}
	for _, d := range testdata {
		assert.Equal(t, d.cmd, NoiseCommand(d.flags), d.cmd, d.flags)
	}
}
