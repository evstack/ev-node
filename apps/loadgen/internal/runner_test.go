package internal

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/celestiaorg/tastora/framework/docker/evstack/spamoor"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/require"
)

func TestRunEntryUsesBaselineCounters(t *testing.T) {
	client := &fakeSpamoorClient{
		spammers:  []spamoor.Spammer{{ID: 99}},
		createIDs: []int{11, 12},
		getSpammerByID: map[int]*spamoor.Spammer{
			11: {ID: 11, Name: "bench-baseline-0", Status: 1},
			12: {ID: 12, Name: "bench-baseline-1", Status: 1},
		},
		metricsSeq: []metricSnapshot{
			{sent: 100, failed: 7},
		},
	}

	var gotTarget int
	var gotBaselineSent float64
	var gotBaselineFailed float64

	err := runEntryWithWait(context.Background(), client, Entry{
		TestName:        "baseline",
		Scenario:        spamoor.ScenarioEOATX,
		Env:             map[string]string{"BENCH_COUNT_PER_SPAMMER": "5"},
		NumSpammers:     2,
		CountPerSpammer: 5,
	}, func(ctx context.Context, api SpamoorClient, targetCount int, baselineSent, baselineFailed float64) (float64, float64, error) {
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
	client := &fakeSpamoorClient{
		createIDs: []int{11},
		getSpammerByID: map[int]*spamoor.Spammer{
			11: {ID: 11, Name: "bench-fail-0", Status: 0},
		},
		metricsSeq: []metricSnapshot{
			{sent: 0, failed: 0},
		},
	}

	err := runEntryWithWait(context.Background(), client, Entry{
		TestName:        "fail",
		Scenario:        spamoor.ScenarioEOATX,
		Env:             map[string]string{"BENCH_COUNT_PER_SPAMMER": "1"},
		NumSpammers:     1,
		CountPerSpammer: 1,
	}, func(ctx context.Context, api SpamoorClient, targetCount int, baselineSent, baselineFailed float64) (float64, float64, error) {
		t.Fatal("wait function should not be called when spammer startup fails")
		return 0, 0, nil
	})

	require.ErrorContains(t, err, "failed to start")
}

func TestWaitForSpamoorDoneUsesDeltas(t *testing.T) {
	client := &fakeSpamoorClient{
		metricsSeq: []metricSnapshot{
			{sent: 105, failed: 3},
			{sent: 108, failed: 4},
		},
	}

	sent, failed, err := waitForSpamoorDoneWithInterval(context.Background(), client, 8, 100, 2, time.Millisecond)
	require.NoError(t, err)
	require.Equal(t, 8.0, sent)
	require.Equal(t, 2.0, failed)
}

func TestWaitForSpamoorDoneHonorsContext(t *testing.T) {
	client := &fakeSpamoorClient{
		metricsSeq: []metricSnapshot{
			{sent: 100, failed: 0},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
	defer cancel()

	_, _, err := waitForSpamoorDoneWithInterval(ctx, client, 2, 100, 0, 10*time.Millisecond)
	require.ErrorContains(t, err, "timed out waiting for 2 txs")
}

func TestWaitForSyncReturnsOnceHeightDeltaSettles(t *testing.T) {
	client := &fakeSpamoorClient{
		clientsSeq: [][]spamoor.Client{
			{{Height: 100}},
			{{Height: 108}},
		},
	}

	err := waitForSync(context.Background(), client, time.Millisecond)
	require.NoError(t, err)
}

func TestWaitForSyncHonorsContext(t *testing.T) {
	client := &fakeSpamoorClient{
		getClientsErr: errors.New("boom"),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
	defer cancel()

	err := waitForSync(ctx, client, 10*time.Millisecond)
	require.ErrorContains(t, err, "cancelled waiting for sync")
}

type metricSnapshot struct {
	sent   float64
	failed float64
}

type fakeSpamoorClient struct {
	spammers       []spamoor.Spammer
	createIDs      []int
	createCalls    []createCall
	getSpammerByID map[int]*spamoor.Spammer
	metricsSeq     []metricSnapshot
	metricsIndex   int
	clientsSeq     [][]spamoor.Client
	clientsIndex   int
	getClientsErr  error
}

type createCall struct {
	name     string
	scenario string
	config   any
	start    bool
}

func (f *fakeSpamoorClient) URL() string { return "http://spamoor.test" }

func (f *fakeSpamoorClient) ListSpammers() ([]spamoor.Spammer, error) {
	return append([]spamoor.Spammer(nil), f.spammers...), nil
}

func (f *fakeSpamoorClient) DeleteSpammer(id int) error {
	return nil
}

func (f *fakeSpamoorClient) CreateSpammer(name, scenario string, config any, start bool) (int, error) {
	f.createCalls = append(f.createCalls, createCall{name: name, scenario: scenario, config: config, start: start})
	if len(f.createIDs) == 0 {
		return 0, fmt.Errorf("unexpected CreateSpammer call")
	}
	id := f.createIDs[0]
	f.createIDs = f.createIDs[1:]
	return id, nil
}

func (f *fakeSpamoorClient) GetSpammer(id int) (*spamoor.Spammer, error) {
	sp, ok := f.getSpammerByID[id]
	if !ok {
		return nil, fmt.Errorf("spammer %d not found", id)
	}
	return sp, nil
}

func (f *fakeSpamoorClient) GetMetrics() (map[string]*dto.MetricFamily, error) {
	snapshot := metricSnapshot{}
	if len(f.metricsSeq) > 0 {
		if f.metricsIndex >= len(f.metricsSeq) {
			snapshot = f.metricsSeq[len(f.metricsSeq)-1]
		} else {
			snapshot = f.metricsSeq[f.metricsIndex]
			f.metricsIndex++
		}
	}

	return map[string]*dto.MetricFamily{
		"spamoor_transactions_sent_total":   counterFamily("spamoor_transactions_sent_total", snapshot.sent),
		"spamoor_transactions_failed_total": counterFamily("spamoor_transactions_failed_total", snapshot.failed),
	}, nil
}

func (f *fakeSpamoorClient) GetClients() ([]spamoor.Client, error) {
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

func counterFamily(name string, value float64) *dto.MetricFamily {
	counterType := dto.MetricType_COUNTER
	return &dto.MetricFamily{
		Name: &name,
		Type: &counterType,
		Metric: []*dto.Metric{
			{
				Counter: &dto.Counter{
					Value: &value,
				},
			},
		},
	}
}
