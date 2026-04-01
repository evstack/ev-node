//go:build evm

package benchmark

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDistributeSpammers(t *testing.T) {
	tests := []struct {
		name     string
		total    int
		pcts     [4]int
		expected [4]int
	}{
		{
			name:     "minimum total equals types",
			total:    4,
			pcts:     [4]int{40, 30, 20, 10},
			expected: [4]int{1, 1, 1, 1},
		},
		{
			name:     "equal percentages",
			total:    8,
			pcts:     [4]int{25, 25, 25, 25},
			expected: [4]int{2, 2, 2, 2},
		},
		{
			name:     "default mix 8 spammers",
			total:    8,
			pcts:     [4]int{40, 30, 20, 10},
			expected: [4]int{3, 2, 2, 1},
		},
		{
			name:     "large total distributes proportionally",
			total:    104,
			pcts:     [4]int{40, 30, 20, 10},
			expected: [4]int{41, 31, 21, 11},
		},
		{
			name:     "sum of counts equals total",
			total:    7,
			pcts:     [4]int{40, 30, 20, 10},
			expected: [4]int{2, 2, 2, 1},
		},
		{
			name:     "5 spammers with uneven split",
			total:    5,
			pcts:     [4]int{40, 30, 20, 10},
			expected: [4]int{2, 1, 1, 1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := distributeSpammers(tt.total, tt.pcts)
			require.Equal(t, tt.expected, result)

			sum := result[0] + result[1] + result[2] + result[3]
			require.Equal(t, tt.total, sum, "sum of distributed spammers must equal total")

			for i, c := range result {
				require.GreaterOrEqual(t, c, 1, "type %d must have at least 1 spammer", i)
			}
		})
	}
}

func TestDistributeSpammersPanicsOnLowTotal(t *testing.T) {
	require.Panics(t, func() {
		distributeSpammers(3, [4]int{40, 30, 20, 10})
	})
	require.Panics(t, func() {
		distributeSpammers(0, [4]int{25, 25, 25, 25})
	})
}
