package proxyconfig

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateFlags(t *testing.T) {
	t.Parallel()

	cases := map[string][]string{
		"Extra args": {"extra"},
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			cmd := Command{}
			cmd.init()

			cmd.set.Parse(c)

			err := cmd.validateFlags()
			require.Error(t, err)
		})
	}
}
