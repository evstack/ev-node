package logging

import (
	"fmt"

	golog "github.com/ipfs/go-log/v2"
	"go.uber.org/zap"
)

// InfoWithKV logs an info message with key-value pairs, using structured zap logging when available,
// falling back to sprintf-style logging otherwise.
func InfoWithKV(logger golog.EventLogger, msg string, kvPairs ...interface{}) {
	if len(kvPairs)%2 != 0 {
		logger.Error("InfoWithKV called with odd number of key-value arguments")
		return
	}

	if zapLogger, ok := logger.(*golog.ZapEventLogger); ok {
		if zl := zapLogger.Desugar(); zl != nil {
			fields := make([]zap.Field, 0, len(kvPairs)/2)
			for i := 0; i < len(kvPairs); i += 2 {
				key := kvPairs[i].(string)
				value := kvPairs[i+1]
				fields = append(fields, createZapField(key, value))
			}
			zl.Info(msg, fields...)
			return
		}
	}

	// Fallback to sprintf-style logging
	logger.Info(formatWithKV(msg, kvPairs...))
}

// ErrorWithKV logs an error message with key-value pairs, using structured zap logging when available,
// falling back to sprintf-style logging otherwise.
func ErrorWithKV(logger golog.EventLogger, msg string, kvPairs ...interface{}) {
	if len(kvPairs)%2 != 0 {
		logger.Error("ErrorWithKV called with odd number of key-value arguments")
		return
	}

	if zapLogger, ok := logger.(*golog.ZapEventLogger); ok {
		if zl := zapLogger.Desugar(); zl != nil {
			fields := make([]zap.Field, 0, len(kvPairs)/2)
			for i := 0; i < len(kvPairs); i += 2 {
				key := kvPairs[i].(string)
				value := kvPairs[i+1]
				fields = append(fields, createZapField(key, value))
			}
			zl.Error(msg, fields...)
			return
		}
	}

	// Fallback to sprintf-style logging
	logger.Error(formatWithKV(msg, kvPairs...))
}

// WarnWithKV logs a warning message with key-value pairs, using structured zap logging when available,
// falling back to sprintf-style logging otherwise.
func WarnWithKV(logger golog.EventLogger, msg string, kvPairs ...interface{}) {
	if len(kvPairs)%2 != 0 {
		logger.Error("WarnWithKV called with odd number of key-value arguments")
		return
	}

	if zapLogger, ok := logger.(*golog.ZapEventLogger); ok {
		if zl := zapLogger.Desugar(); zl != nil {
			fields := make([]zap.Field, 0, len(kvPairs)/2)
			for i := 0; i < len(kvPairs); i += 2 {
				key := kvPairs[i].(string)
				value := kvPairs[i+1]
				fields = append(fields, createZapField(key, value))
			}
			zl.Warn(msg, fields...)
			return
		}
	}

	// Fallback to sprintf-style logging
	logger.Warn(formatWithKV(msg, kvPairs...))
}

// DebugWithKV logs a debug message with key-value pairs, using structured zap logging when available,
// falling back to sprintf-style logging otherwise.
func DebugWithKV(logger golog.EventLogger, msg string, kvPairs ...interface{}) {
	if len(kvPairs)%2 != 0 {
		logger.Error("DebugWithKV called with odd number of key-value arguments")
		return
	}

	if zapLogger, ok := logger.(*golog.ZapEventLogger); ok {
		if zl := zapLogger.Desugar(); zl != nil {
			fields := make([]zap.Field, 0, len(kvPairs)/2)
			for i := 0; i < len(kvPairs); i += 2 {
				key := kvPairs[i].(string)
				value := kvPairs[i+1]
				fields = append(fields, createZapField(key, value))
			}
			zl.Debug(msg, fields...)
			return
		}
	}

	// Fallback to sprintf-style logging
	logger.Debug(formatWithKV(msg, kvPairs...))
}

// createZapField creates the appropriate zap field based on the value type
func createZapField(key string, value interface{}) zap.Field {
	switch v := value.(type) {
	case string:
		return zap.String(key, v)
	case int:
		return zap.Int(key, v)
	case int64:
		return zap.Int64(key, v)
	case uint64:
		return zap.Uint64(key, v)
	case float64:
		return zap.Float64(key, v)
	case bool:
		return zap.Bool(key, v)
	case error:
		return zap.Error(v) // Note: zap.Error doesn't take a key, it uses "error" as key
	default:
		return zap.Any(key, v)
	}
}

// formatWithKV formats a message with key-value pairs for sprintf-style logging
func formatWithKV(msg string, kvPairs ...interface{}) string {
	if len(kvPairs) == 0 {
		return msg
	}

	format := msg
	args := make([]interface{}, 0, len(kvPairs)/2)
	for i := 0; i < len(kvPairs); i += 2 {
		key := kvPairs[i].(string)
		value := kvPairs[i+1]
		format += ", " + key + ": %v"
		args = append(args, value)
	}
	return fmt.Sprintf(format, args...)
}
