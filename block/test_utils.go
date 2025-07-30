package block

import (
	"crypto/sha256"
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// GenerateHeaderHash creates a deterministic hash for a test header based on height and proposer.
// This is useful for predicting expected hashes in tests without needing full header construction.
func GenerateHeaderHash(t *testing.T, height uint64, proposer []byte) []byte {
	t.Helper()
	// Create a simple deterministic representation of the header's identity
	heightBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(heightBytes, height)

	hasher := sha256.New()
	_, err := hasher.Write([]byte("testheader:")) // Prefix to avoid collisions
	require.NoError(t, err)
	_, err = hasher.Write(heightBytes)
	require.NoError(t, err)
	_, err = hasher.Write(proposer)
	require.NoError(t, err)

	return hasher.Sum(nil)
}

// MockZapLogger is a mock implementation of *zap.Logger for testing
type MockZapLogger struct {
	mock.Mock
}

// Mock the core zap.Logger methods
func (m *MockZapLogger) Debug(msg string, fields ...zap.Field) { m.Called(msg, fields) }
func (m *MockZapLogger) Info(msg string, fields ...zap.Field)  { m.Called(msg, fields) }
func (m *MockZapLogger) Warn(msg string, fields ...zap.Field)  { m.Called(msg, fields) }
func (m *MockZapLogger) Error(msg string, fields ...zap.Field) { m.Called(msg, fields) }
func (m *MockZapLogger) Fatal(msg string, fields ...zap.Field) {
	m.Called(msg, fields)
	panic("fatal error logged")
}
func (m *MockZapLogger) Panic(msg string, fields ...zap.Field) {
	m.Called(msg, fields)
	panic("panic error logged")
}

// Mock additional methods that might be used
func (m *MockZapLogger) With(fields ...zap.Field) *zap.Logger {
	args := m.Called(fields)
	if logger, ok := args.Get(0).(*zap.Logger); ok {
		return logger
	}
	return zap.NewNop() // Return a no-op logger as fallback
}

func (m *MockZapLogger) Named(name string) *zap.Logger {
	args := m.Called(name)
	if logger, ok := args.Get(0).(*zap.Logger); ok {
		return logger
	}
	return zap.NewNop()
}

func (m *MockZapLogger) WithOptions(opts ...zap.Option) *zap.Logger {
	args := m.Called(opts)
	if logger, ok := args.Get(0).(*zap.Logger); ok {
		return logger
	}
	return zap.NewNop()
}

func (m *MockZapLogger) Core() zapcore.Core {
	args := m.Called()
	if core, ok := args.Get(0).(zapcore.Core); ok {
		return core
	}
	return zapcore.NewNopCore()
}

func (m *MockZapLogger) Sync() error {
	args := m.Called()
	return args.Error(0)
}
