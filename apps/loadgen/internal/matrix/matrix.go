package matrix

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"time"

	spamoorapi "github.com/celestiaorg/tastora/framework/docker/evstack/spamoor"
)

var validScenarios = map[string]bool{
	spamoorapi.ScenarioEOATX:            true,
	spamoorapi.ScenarioERC20TX:          true,
	spamoorapi.ScenarioERC721TX:         true,
	spamoorapi.ScenarioERC1155TX:        true,
	spamoorapi.ScenarioCallTX:           true,
	spamoorapi.ScenarioDeployTX:         true,
	spamoorapi.ScenarioDeployDestruct:   true,
	spamoorapi.ScenarioSetCodeTX:        true,
	spamoorapi.ScenarioUniswapSwaps:     true,
	spamoorapi.ScenarioBlobs:            true,
	spamoorapi.ScenarioBlobAverage:      true,
	spamoorapi.ScenarioBlobReplacements: true,
	spamoorapi.ScenarioBlobConflicting:  true,
	spamoorapi.ScenarioBlobCombined:     true,
	spamoorapi.ScenarioGasBurnerTX:      true,
	spamoorapi.ScenarioStorageSpam:      true,
	spamoorapi.ScenarioGeasTX:           true,
	spamoorapi.ScenarioXenToken:         true,
	spamoorapi.ScenarioTaskRunner:       true,
}

// Matrix is the top-level structure of a benchmark matrix JSON file.
type Matrix struct {
	Entries []Entry `json:"entries"`
}

const (
	EnvNumSpammers     = "BENCH_NUM_SPAMMERS"
	EnvCountPerSpammer = "BENCH_COUNT_PER_SPAMMER"
)

// Entry is a single benchmark scenario in a matrix file. NumSpammers,
// CountPerSpammer, and ParsedTimeout are derived from Env/Timeout during validation.
type Entry struct {
	TestName        string            `json:"test_name"`
	Scenario        string            `json:"scenario"`
	Timeout         string            `json:"timeout"`
	Probability     *float64          `json:"probability,omitempty"`
	Env             map[string]string `json:"env"`
	NumSpammers     int               `json:"-"`
	CountPerSpammer int               `json:"-"`
	ParsedTimeout   time.Duration     `json:"-"`
}

// Load reads and validates a matrix JSON file from disk.
func Load(path string) (*Matrix, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read matrix file: %w", err)
	}
	var m Matrix
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parse matrix JSON: %w", err)
	}
	if err := Validate(&m); err != nil {
		return nil, err
	}
	return &m, nil
}

func Validate(m *Matrix) error {
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

	e.NumSpammers = envInt(e.Env, EnvNumSpammers, 1)
	if e.NumSpammers <= 0 {
		return fmt.Errorf("%s must be > 0", EnvNumSpammers)
	}

	e.CountPerSpammer = envInt(e.Env, EnvCountPerSpammer, 0)
	if e.CountPerSpammer <= 0 {
		return fmt.Errorf("%s must be > 0", EnvCountPerSpammer)
	}

	if e.Probability != nil && (*e.Probability < 0 || *e.Probability > 1) {
		return fmt.Errorf("probability must be between 0.0 and 1.0, got %f", *e.Probability)
	}

	if e.Timeout != "" {
		d, err := time.ParseDuration(e.Timeout)
		if err != nil {
			return fmt.Errorf("invalid timeout %q: %w", e.Timeout, err)
		}
		e.ParsedTimeout = d
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
