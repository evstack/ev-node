package config

import (
	"errors"
	"strings"
)

// InstrumentationConfig defines the configuration for metrics reporting.
type InstrumentationConfig struct {
	// When true, Prometheus metrics are served under /metrics on
	// PrometheusListenAddr.
	// Check out the documentation for the list of available metrics.
	Prometheus bool `yaml:"prometheus" comment:"Enable Prometheus metrics"` // When true, Prometheus metrics are served

	// Address to listen for Prometheus collector(s) connections.
	PrometheusListenAddr string `mapstructure:"prometheus_listen_addr" yaml:"prometheus_listen_addr" comment:"Address to listen for Prometheus metrics"`

	// Maximum number of simultaneous connections.
	// If you want to accept a larger number than the default, make sure
	// you increase your OS limits.
	// 0 - unlimited.
	MaxOpenConnections int `mapstructure:"max_open_connections" yaml:"max_open_connections" comment:"Maximum number of simultaneous connections"`

	// Instrumentation namespace.
	Namespace string `yaml:"namespace" comment:"Namespace for metrics"` // Instrumentation namespace

	// When true, pprof endpoints are served under /debug/pprof/ on
	// PprofListenAddr. This enables runtime profiling of the application.
	// Available endpoints include:
	// - /debug/pprof/          - Index page
	// - /debug/pprof/cmdline   - Command line arguments
	// - /debug/pprof/profile   - CPU profile
	// - /debug/pprof/symbol    - Symbol lookup
	// - /debug/pprof/trace     - Execution trace
	// - /debug/pprof/goroutine - Goroutine stack dumps
	// - /debug/pprof/heap      - Heap memory profile
	// - /debug/pprof/mutex     - Mutex contention profile
	// - /debug/pprof/block     - Block profile
	// - /debug/pprof/allocs    - Allocation profile
	Pprof bool `yaml:"pprof" comment:"Enable pprof profiling server"` // When true, pprof endpoints are served

	// Address to listen for pprof connections.
	// Default is ":6060" which is the standard port for pprof.
	PprofListenAddr string `mapstructure:"pprof_listen_addr" yaml:"pprof_listen_addr" comment:"Address to listen for pprof connections"`

	// Tracing enables OpenTelemetry tracing when true.
	Tracing bool `mapstructure:"tracing" yaml:"tracing" comment:"Enable OpenTelemetry tracing"`

	// TracingEndpoint is the OTLP endpoint (host:port) for exporting traces.
	TracingEndpoint string `mapstructure:"tracing_endpoint" yaml:"tracing_endpoint" comment:"OTLP endpoint for traces (host:port)"`

	// TracingServiceName is the service.name resource attribute for this node.
	TracingServiceName string `mapstructure:"tracing_service_name" yaml:"tracing_service_name" comment:"OpenTelemetry service.name for this process"`

	// TracingSampleRate is the TraceID ratio-based sampling rate (0.0 - 1.0).
	TracingSampleRate float64 `mapstructure:"tracing_sample_rate" yaml:"tracing_sample_rate" comment:"Sampling rate for traces (0.0-1.0)"`
}

// DefaultInstrumentationConfig returns a default configuration for metrics
// reporting.
func DefaultInstrumentationConfig() *InstrumentationConfig {
	return &InstrumentationConfig{
		Prometheus:           false,
		PrometheusListenAddr: ":26660",
		MaxOpenConnections:   3,
		Namespace:            "evnode",
		Pprof:                false,
		PprofListenAddr:      ":6060",
		Tracing:              false,
		TracingEndpoint:      "localhost:4318",
		TracingServiceName:   "ev-node",
		TracingSampleRate:    0.1,
	}
}

// ValidateBasic performs basic validation (checking param bounds, etc.) and
// returns an error if any check fails.
func (cfg *InstrumentationConfig) ValidateBasic() error {
	if cfg.MaxOpenConnections < 0 {
		return errors.New("max_open_connections can't be negative")
	}
	if cfg.IsTracingEnabled() {
		if cfg.TracingSampleRate < 0 || cfg.TracingSampleRate > 1 {
			return errors.New("tracing_sample_rate must be between 0 and 1")
		}
		if strings.TrimSpace(cfg.TracingEndpoint) == "" {
			return errors.New("tracing_endpoint cannot be empty")
		}
	}
	return nil
}

// IsPrometheusEnabled returns true if Prometheus metrics are enabled.
func (cfg *InstrumentationConfig) IsPrometheusEnabled() bool {
	return cfg.Prometheus && cfg.PrometheusListenAddr != ""
}

// IsPprofEnabled returns true if pprof endpoints are enabled.
func (cfg *InstrumentationConfig) IsPprofEnabled() bool {
	return cfg.Pprof
}

// GetPprofListenAddr returns the address to listen for pprof connections.
// If PprofListenAddr is empty, it returns the default pprof port ":6060".
func (cfg *InstrumentationConfig) GetPprofListenAddr() string {
	if cfg.PprofListenAddr == "" {
		return DefaultInstrumentationConfig().PprofListenAddr
	}

	return cfg.PprofListenAddr
}

// IsTracingEnabled returns true if OpenTelemetry tracing is enabled.
func (cfg *InstrumentationConfig) IsTracingEnabled() bool {
	return cfg != nil && cfg.Tracing && cfg.TracingEndpoint != ""
}
