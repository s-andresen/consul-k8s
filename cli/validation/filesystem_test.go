package validation

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsPath(t *testing.T) {
	tests := map[string]struct {
		path        string
		expectValid bool
	}{
		"Valid Linux path": {"/home/user/.kube/config", true},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			err := IsPath(tt.path)

			if tt.expectValid {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}
