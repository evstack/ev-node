package runner

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/evstack/ev-node/apps/loadgen/internal/matrix"
	"github.com/evstack/ev-node/apps/loadgen/internal/spamoor"

	spamoorapi "github.com/celestiaorg/tastora/framework/docker/evstack/spamoor"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/require"
)

func TestRunEntryUsesBaselineCounters(t *testing.T) {
	client := &fakeClient{
		spammers:  []spamoorapi.Spammer{{ID: 99}},
		createIDs: []int{11, 12},
		getSpammerByID: map[int]*spamoorapi.Spammer{
			11: {ID: 11, Name: "bench-baseline-0", Status: 1},
			12: {ID: 12, Name: "bench-baseline-1", Status: 1},
		},
		metricsSeq: []metricSnapshot{
			{sent: 100, failed: 7},
		},
		spammerNames: []string{"bench-baseline-0", "bench-baseline-1"},
	}

	var gotTarget int
	var gotBaselineSent float64
	var gotBaselineFailed float64

	err := runEntryWithWait(context.Background(), client, matrix.Entry{
		TestName:        "baseline",
		Scenario:        spamoorapi.ScenarioEOATX,
		Env:             map[string]string{"BENCH_COUNT_PER_SPAMMER": "5"},
		NumSpammers:     2,
		CountPerSpammer: 5,
	}, func(ctx context.Context, api spamoor.Client, targetCount int, baselineSent, baselineFailed float64, namePrefix string) (float64, float64, error) {
		gotTarget = targetCount
		gotBaselineSent = baselineSent
		gotBaselineFailed = baselineFailed
		return 10, 0, nil
	})

	require.NoError(t, err)
	require.Equal(t, 10, gotTarget)
	require.Equal(t, 100.0, gotBaselineSent)
	require.Equal(t, 7.0, gotBaselineFailed)
	require.Len(t, client.createCalls, 2)
}

func TestRunEntryFailsWhenSpammerDoesNotStart(t *testing.T) {
	client := &fakeClient{
		createIDs: []int{11},
		getSpammerByID: map[int]*spamoorapi.Spammer{
			11: {ID: 11, Name: "bench-fail-0", Status: 0},
		},
		metricsSeq: []metricSnapshot{
			{sent: 0, failed: 0},
		},
		spammerNames: []string{"bench-fail-0"},
	}

	err := runEntryWithWait(context.Background(), client, matrix.Entry{
		TestName:        "fail",
		Scenario:        spamoorapi.ScenarioEOATX,
		Env:             map[string]string{"BENCH_COUNT_PER_SPAMMER": "1"},
		NumSpammers:     1,
		CountPerSpammer: 1,
	}, func(ctx context.Context, api spamoor.Client, targetCount int, baselineSent, baselineFailed float64, namePrefix string) (float64, float64, error) {
		t.Fatal("wait function should not be called when spammer startup fails")
		return 0, 0, nil
	})

	require.ErrorContains(t, err, "failed to start")
}

func TestWaitForSpamoorDoneUsesDeltas(t *testing.T) {
	client := &fakeClient{
		metricsSeq: []metricSnapshot{
			{sent: 105, failed: 3},
			{sent: 108, failed: 4},
		},
		spammerNames: []string{"bench-test-0"},
	}

	sent, failed, err := waitForSpamoorDoneWithInterval(context.Background(), client, 8, 100, 2, "", time.Millisecond)
	require.NoError(t, err)
	require.Equal(t, 8.0, sent)
	require.Equal(t, 2.0, failed)
}

func TestWaitForSpamoorDoneHonorsContext(t *testing.T) {
	client := &fakeClient{
		metricsSeq: []metricSnapshot{
			{sent: 100, failed: 0},
		},
		spammerNames: []string{"bench-test-0"},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
	defer cancel()

	_, _, err := waitForSpamoorDoneWithInterval(ctx, client, 2, 100, 0, "", 10*time.Millisecond)
	require.ErrorContains(t, err, "timed out waiting for 2 txs")
}

func TestWaitForSyncReturnsOnceHeightDeltaSettles(t *testing.T) {
	client := &fakeClient{
		clientsSeq: [][]spamoorapi.Client{
			{{Height: 100}},
			{{Height: 108}},
		},
	}

	err := waitForSync(context.Background(), client, time.Millisecond)
	require.NoError(t, err)
}

func TestWaitForSyncHonorsContext(t *testing.T) {
	client := &fakeClient{
		getClientsErr: errors.New("boom"),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
	defer cancel()

	err := waitForSync(ctx, client, 10*time.Millisecond)
	require.ErrorContains(t, err, "cancelled waiting for sync")
}

func TestSumCounterWithPrefixFiltersCorrectly(t *testing.T) {
	family := labeledCounterFamily("spamoor_transactions_sent_total", map[string]float64{
		"bench-EOATransferBurst-0": 1000,
		"bench-EOATransferBurst-1": 1000,
		"bench-EOATransfer-0":      500,
		"bench-EOATransfer-1":      500,
	})

	t.Run("no prefix sums all", func(t *testing.T) {
		total := sumCounterWithPrefix(family, "")
		require.Equal(t, 3000.0, total)
	})

	t.Run("burst prefix sums only burst spammers", func(t *testing.T) {
		total := sumCounterWithPrefix(family, "bench-EOATransferBurst-")
		require.Equal(t, 2000.0, total)
	})

	t.Run("baseline prefix does not match burst spammers", func(t *testing.T) {
		total := sumCounterWithPrefix(family, "bench-EOATransfer-")
		require.Equal(t, 1000.0, total)
	})

	t.Run("nil family returns zero", func(t *testing.T) {
		require.Equal(t, 0.0, sumCounterWithPrefix(nil, "anything"))
	})

	t.Run("unmatched prefix returns zero", func(t *testing.T) {
		total := sumCounterWithPrefix(family, "bench-Nonexistent-")
		require.Equal(t, 0.0, total)
	})
}

type metricSnapshot struct {
	sent   float64
	failed float64
}

type fakeClient struct {
	spammers       []spamoorapi.Spammer
	createIDs      []int
	createCalls    []createCall
	getSpammerByID map[int]*spamoorapi.Spammer
	metricsSeq     []metricSnapshot
	metricsIndex   int
	clientsSeq     [][]spamoorapi.Client
	clientsIndex   int
	getClientsErr  error
	spammerNames   []string
}

type createCall struct {
	name     string
	scenario string
	config   any
	start    bool
}

func (f *fakeClient) URL() string { return "http://spamoor.test" }

func (f *fakeClient) ListSpammers() ([]spamoorapi.Spammer, error) {
	return append([]spamoorapi.Spammer(nil), f.spammers...), nil
}

func (f *fakeClient) DeleteSpammer(id int) error {
	return nil
}

func (f *fakeClient) CreateSpammer(name, scenario string, config any, start bool) (int, error) {
	f.createCalls = append(f.createCalls, createCall{name: name, scenario: scenario, config: config, start: start})
	if len(f.createIDs) == 0 {
		return 0, fmt.Errorf("unexpected CreateSpammer call")
	}
	id := f.createIDs[0]
	f.createIDs = f.createIDs[1:]
	return id, nil
}

func (f *fakeClient) GetSpammer(id int) (*spamoorapi.Spammer, error) {
	sp, ok := f.getSpammerByID[id]
	if !ok {
		return nil, fmt.Errorf("spammer %d not found", id)
	}
	return sp, nil
}

func (f *fakeClient) GetMetrics() (map[string]*dto.MetricFamily, error) {
	snapshot := metricSnapshot{}
	if len(f.metricsSeq) > 0 {
		if f.metricsIndex >= len(f.metricsSeq) {
			snapshot = f.metricsSeq[len(f.metricsSeq)-1]
		} else {
			snapshot = f.metricsSeq[f.metricsIndex]
			f.metricsIndex++
		}
	}

	perSpammer := snapshot.sent / float64(len(f.spammerNames))
	perFailed := snapshot.failed / float64(len(f.spammerNames))
	sentMap := make(map[string]float64, len(f.spammerNames))
	failedMap := make(map[string]float64, len(f.spammerNames))
	for _, name := range f.spammerNames {
		sentMap[name] = perSpammer
		failedMap[name] = perFailed
	}
	return map[string]*dto.MetricFamily{
		"spamoor_transactions_sent_total":   labeledCounterFamily("spamoor_transactions_sent_total", sentMap),
		"spamoor_transactions_failed_total": labeledCounterFamily("spamoor_transactions_failed_total", failedMap),
	}, nil
}

func (f *fakeClient) GetClients() ([]spamoorapi.Client, error) {
	if f.getClientsErr != nil {
		return nil, f.getClientsErr
	}
	if len(f.clientsSeq) == 0 {
		return nil, nil
	}
	if f.clientsIndex >= len(f.clientsSeq) {
		return f.clientsSeq[len(f.clientsSeq)-1], nil
	}
	clients := f.clientsSeq[f.clientsIndex]
	f.clientsIndex++
	return clients, nil
}

func labeledCounterFamily(name string, spammerValues map[string]float64) *dto.MetricFamily {
	counterType := dto.MetricType_COUNTER
	labelName := "spammer_name"
	var metrics []*dto.Metric
	for spammerName, value := range spammerValues {
		metrics = append(metrics, &dto.Metric{
			Label: []*dto.LabelPair{
				{Name: &labelName, Value: &spammerName},
			},
			Counter: &dto.Counter{Value: &value},
		})
	}
	return &dto.MetricFamily{
		Name:   &name,
		Type:   &counterType,
		Metric: metrics,
	}
}
