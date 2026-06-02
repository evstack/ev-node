package internal

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadMatrix(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		path := writeMatrixFile(t, `{
		  "entries": [
		    {
		      "test_name": "EOA",
		      "scenario": "eoatx",
		      "timeout": "2m",
		      "probability": 0.5,
		      "env": {
		        "BENCH_NUM_SPAMMERS": "2",
		        "BENCH_COUNT_PER_SPAMMER": "7"
		      }
		    }
		  ]
		}`)

		matrix, err := LoadMatrix(path)
		require.NoError(t, err)
		require.Len(t, matrix.Entries, 1)
		require.Equal(t, 2, matrix.Entries[0].NumSpammers)
		require.Equal(t, 7, matrix.Entries[0].CountPerSpammer)
	})

	t.Run("empty entries", func(t *testing.T) {
		path := writeMatrixFile(t, `{"entries":[]}`)
		_, err := LoadMatrix(path)
		require.ErrorContains(t, err, "matrix has no entries")
	})

	t.Run("invalid json", func(t *testing.T) {
		path := writeMatrixFile(t, `{`)
		_, err := LoadMatrix(path)
		require.ErrorContains(t, err, "parse matrix JSON")
	})

	t.Run("invalid scenario", func(t *testing.T) {
		path := writeMatrixFile(t, `{
		  "entries": [
		    {
		      "test_name": "Bad",
		      "scenario": "unknown",
		      "env": {
		        "BENCH_COUNT_PER_SPAMMER": "1"
		      }
		    }
		  ]
		}`)
		_, err := LoadMatrix(path)
		require.ErrorContains(t, err, `unknown scenario "unknown"`)
	})

	t.Run("invalid spammer count", func(t *testing.T) {
		path := writeMatrixFile(t, `{
		  "entries": [
		    {
		      "test_name": "Bad",
		      "scenario": "eoatx",
		      "env": {
		        "BENCH_NUM_SPAMMERS": "0",
		        "BENCH_COUNT_PER_SPAMMER": "1"
		      }
		    }
		  ]
		}`)
		_, err := LoadMatrix(path)
		require.ErrorContains(t, err, "BENCH_NUM_SPAMMERS must be > 0")
	})

	t.Run("invalid tx count", func(t *testing.T) {
		path := writeMatrixFile(t, `{
		  "entries": [
		    {
		      "test_name": "Bad",
		      "scenario": "eoatx",
		      "env": {
		        "BENCH_COUNT_PER_SPAMMER": "0"
		      }
		    }
		  ]
		}`)
		_, err := LoadMatrix(path)
		require.ErrorContains(t, err, "BENCH_COUNT_PER_SPAMMER must be > 0")
	})

	t.Run("invalid probability", func(t *testing.T) {
		path := writeMatrixFile(t, `{
		  "entries": [
		    {
		      "test_name": "Bad",
		      "scenario": "eoatx",
		      "probability": 1.5,
		      "env": {
		        "BENCH_COUNT_PER_SPAMMER": "1"
		      }
		    }
		  ]
		}`)
		_, err := LoadMatrix(path)
		require.ErrorContains(t, err, "probability must be between 0.0 and 1.0")
	})
}

func writeMatrixFile(t *testing.T, contents string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "matrix.json")
	require.NoError(t, os.WriteFile(path, []byte(contents), 0o600))
	return path
}
