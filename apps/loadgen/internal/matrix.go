package internal

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"github.com/celestiaorg/tastora/framework/docker/evstack/spamoor"
)

var validScenarios = map[string]bool{
	spamoor.ScenarioEOATX:            true,
	spamoor.ScenarioERC20TX:          true,
	spamoor.ScenarioERC721TX:         true,
	spamoor.ScenarioERC1155TX:        true,
	spamoor.ScenarioCallTX:           true,
	spamoor.ScenarioDeployTX:         true,
	spamoor.ScenarioDeployDestruct:   true,
	spamoor.ScenarioSetCodeTX:        true,
	spamoor.ScenarioUniswapSwaps:     true,
	spamoor.ScenarioBlobs:            true,
	spamoor.ScenarioBlobAverage:      true,
	spamoor.ScenarioBlobReplacements: true,
	spamoor.ScenarioBlobConflicting:  true,
	spamoor.ScenarioBlobCombined:     true,
	spamoor.ScenarioGasBurnerTX:      true,
	spamoor.ScenarioStorageSpam:      true,
	spamoor.ScenarioGeasTX:           true,
	spamoor.ScenarioXenToken:         true,
	spamoor.ScenarioTaskRunner:       true,
}

// Matrix is the top-level structure of a benchmark matrix JSON file.
type Matrix struct {
	Entries []Entry `json:"entries"`
}

// Entry is a single benchmark scenario in a matrix file. NumSpammers and
// CountPerSpammer are derived from Env during validation.
type Entry struct {
	TestName        string            `json:"test_name"`
	Scenario        string            `json:"scenario"`
	Timeout         string            `json:"timeout"`
	Probability     *float64          `json:"probability,omitempty"`
	Env             map[string]string `json:"env"`
	NumSpammers     int               `json:"-"`
	CountPerSpammer int               `json:"-"`
}

// LoadMatrix reads and validates a matrix JSON file from disk.
func LoadMatrix(path string) (*Matrix, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read matrix file: %w", err)
	}
	var m Matrix
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parse matrix JSON: %w", err)
	}
	if err := validateMatrix(&m); err != nil {
		return nil, err
	}
	return &m, nil
}

func validateMatrix(m *Matrix) error {
	if len(m.Entries) == 0 {
		return fmt.Errorf("matrix has no entries")
	}
	for i := range m.Entries {
		if err := m.Entries[i].validate(); err != nil {
			return fmt.Errorf("entry %d (%s): %w", i, m.Entries[i].TestName, err)
		}
	}
	return nil
}

func (e *Entry) validate() error {
	if e.Scenario == "" {
		return fmt.Errorf("missing scenario")
	}
	if !validScenarios[e.Scenario] {
		return fmt.Errorf("unknown scenario %q", e.Scenario)
	}

	e.NumSpammers = envInt(e.Env, "BENCH_NUM_SPAMMERS", 1)
	if e.NumSpammers <= 0 {
		return fmt.Errorf("BENCH_NUM_SPAMMERS must be > 0")
	}

	e.CountPerSpammer = envInt(e.Env, "BENCH_COUNT_PER_SPAMMER", 0)
	if e.CountPerSpammer <= 0 {
		return fmt.Errorf("BENCH_COUNT_PER_SPAMMER must be > 0")
	}

	if e.Probability != nil && (*e.Probability < 0 || *e.Probability > 1) {
		return fmt.Errorf("probability must be between 0.0 and 1.0, got %f", *e.Probability)
	}

	return nil
}

func envInt(env map[string]string, key string, fallback int) int {
	v, ok := env[key]
	if !ok {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}
