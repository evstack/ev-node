// Package mocks provides mock implementations for testing.
// This file contains a manual mock that combines Executor and HeightProvider interfaces.
package mocks

import (
	"context"
	"time"

	"github.com/stretchr/testify/mock"
)

// NewMockHeightAwareExecutor creates a new instance of MockHeightAwareExecutor.
// It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
func NewMockHeightAwareExecutor(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockHeightAwareExecutor {
	mockExec := &MockHeightAwareExecutor{}
	mockExec.Mock.Test(t)

	t.Cleanup(func() { mockExec.AssertExpectations(t) })

	return mockExec
}

// MockHeightAwareExecutor is a mock that implements both Executor and HeightProvider interfaces.
// This allows testing code that needs an executor with height awareness capability.
type MockHeightAwareExecutor struct {
	mock.Mock
}

// InitChain implements the Executor interface.
func (m *MockHeightAwareExecutor) InitChain(ctx context.Context, genesisTime time.Time, initialHeight uint64, chainID string) ([]byte, uint64, error) {
	args := m.Called(ctx, genesisTime, initialHeight, chainID)
	return args.Get(0).([]byte), args.Get(1).(uint64), args.Error(2)
}

// GetTxs implements the Executor interface.
func (m *MockHeightAwareExecutor) GetTxs(ctx context.Context) ([][]byte, error) {
	args := m.Called(ctx)
	return args.Get(0).([][]byte), args.Error(1)
}

// ExecuteTxs implements the Executor interface.
func (m *MockHeightAwareExecutor) ExecuteTxs(ctx context.Context, txs [][]byte, blockHeight uint64, timestamp time.Time, prevStateRoot []byte) ([]byte, uint64, error) {
	args := m.Called(ctx, txs, blockHeight, timestamp, prevStateRoot)
	return args.Get(0).([]byte), args.Get(1).(uint64), args.Error(2)
}

// SetFinal implements the Executor interface.
func (m *MockHeightAwareExecutor) SetFinal(ctx context.Context, blockHeight uint64) error {
	args := m.Called(ctx, blockHeight)
	return args.Error(0)
}

// GetLatestHeight implements the HeightProvider interface.
func (m *MockHeightAwareExecutor) GetLatestHeight(ctx context.Context) (uint64, error) {
	args := m.Called(ctx)
	return args.Get(0).(uint64), args.Error(1)
}
