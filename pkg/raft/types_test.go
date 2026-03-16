package raft

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAssertValid(t *testing.T) {
	specs := map[string]struct {
		s      *RaftBlockState
		next   *RaftBlockState
		expErr bool
	}{
		"ok": {
			s:    &RaftBlockState{Height: 10, Timestamp: 100},
			next: &RaftBlockState{Height: 11, Timestamp: 101},
		},
		"all equal": {
			s:      &RaftBlockState{Height: 10, Timestamp: 100},
			next:   &RaftBlockState{Height: 10, Timestamp: 100},
			expErr: true,
		},
		"equal height": {
			s:      &RaftBlockState{Height: 10, Timestamp: 100},
			next:   &RaftBlockState{Height: 10, Timestamp: 101},
			expErr: true,
		},
		"equal timestamp": {
			s:      &RaftBlockState{Height: 10, Timestamp: 100},
			next:   &RaftBlockState{Height: 11, Timestamp: 100},
			expErr: true,
		},
		"lower timestamp": {
			s:      &RaftBlockState{Height: 10, Timestamp: 101},
			next:   &RaftBlockState{Height: 11, Timestamp: 100},
			expErr: true,
		},
		"lower height": {
			s:      &RaftBlockState{Height: 11, Timestamp: 100},
			next:   &RaftBlockState{Height: 10, Timestamp: 101},
			expErr: true,
		},
	}

	for name, spec := range specs {
		t.Run(name, func(t *testing.T) {
			err := assertValid(spec.s, spec.next)
			if spec.expErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}
