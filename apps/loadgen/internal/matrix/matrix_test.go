package matrix

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	spamoorapi "github.com/celestiaorg/tastora/framework/docker/evstack/spamoor"
	"github.com/stretchr/testify/require"
)

func TestValidate(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		m := Matrix{
			Entries: []Entry{
				{
					TestName: "EOA",
					Scenario: spamoorapi.ScenarioEOATX,
					Timeout:  "2m",
					Probability: func() *float64 {
						v := 0.5
						return &v
					}(),
					Env: map[string]string{
						"BENCH_NUM_SPAMMERS":      "2",
						"BENCH_COUNT_PER_SPAMMER": "7",
					},
				},
			},
		}

		require.NoError(t, Validate(&m))
		require.Equal(t, 2, m.Entries[0].NumSpammers)
		require.Equal(t, 7, m.Entries[0].CountPerSpammer)
	})

	t.Run("empty entries", func(t *testing.T) {
		err := Validate(&Matrix{})
		require.ErrorContains(t, err, "matrix has no entries")
	})

	t.Run("invalid scenario", func(t *testing.T) {
		m := Matrix{
			Entries: []Entry{
				{
					TestName: "Bad",
					Scenario: "unknown",
					Env: map[string]string{
						"BENCH_COUNT_PER_SPAMMER": "1",
					},
				},
			},
		}

		err := Validate(&m)
		require.ErrorContains(t, err, `unknown scenario "unknown"`)
	})

	t.Run("invalid spammer count", func(t *testing.T) {
		m := Matrix{
			Entries: []Entry{
				{
					TestName: "Bad",
					Scenario: spamoorapi.ScenarioEOATX,
					Env: map[string]string{
						"BENCH_NUM_SPAMMERS":      "0",
						"BENCH_COUNT_PER_SPAMMER": "1",
					},
				},
			},
		}

		err := Validate(&m)
		require.ErrorContains(t, err, "BENCH_NUM_SPAMMERS must be > 0")
	})

	t.Run("invalid tx count", func(t *testing.T) {
		m := Matrix{
			Entries: []Entry{
				{
					TestName: "Bad",
					Scenario: spamoorapi.ScenarioEOATX,
					Env: map[string]string{
						"BENCH_COUNT_PER_SPAMMER": "0",
					},
				},
			},
		}

		err := Validate(&m)
		require.ErrorContains(t, err, "BENCH_COUNT_PER_SPAMMER must be > 0")
	})

	t.Run("invalid probability", func(t *testing.T) {
		v := 1.5
		m := Matrix{
			Entries: []Entry{
				{
					TestName:    "Bad",
					Scenario:    spamoorapi.ScenarioEOATX,
					Probability: &v,
					Env: map[string]string{
						"BENCH_COUNT_PER_SPAMMER": "1",
					},
				},
			},
		}

		err := Validate(&m)
		require.ErrorContains(t, err, "probability must be between 0.0 and 1.0")
	})
}

func TestLoad(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		m := Matrix{
			Entries: []Entry{
				{
					TestName: "EOA",
					Scenario: spamoorapi.ScenarioEOATX,
					Env: map[string]string{
						"BENCH_NUM_SPAMMERS":      "2",
						"BENCH_COUNT_PER_SPAMMER": "7",
					},
				},
			},
		}
		data, err := json.Marshal(m)
		require.NoError(t, err)

		path := filepath.Join(t.TempDir(), "matrix.json")
		require.NoError(t, os.WriteFile(path, data, 0o600))

		loaded, err := Load(path)
		require.NoError(t, err)
		require.Equal(t, m.Entries[0].TestName, loaded.Entries[0].TestName)
		require.Equal(t, 2, loaded.Entries[0].NumSpammers)
		require.Equal(t, 7, loaded.Entries[0].CountPerSpammer)
	})

	t.Run("invalid json", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "invalid.json")
		require.NoError(t, os.WriteFile(path, []byte(`{`), 0o600))

		_, err := Load(path)
		require.ErrorContains(t, err, "parse matrix JSON")
	})
}
