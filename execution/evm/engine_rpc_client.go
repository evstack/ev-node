package evm

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/beacon/engine"
	"github.com/ethereum/go-ethereum/rpc"
)

// engineErrUnsupportedFork is the Engine API error code for "Unsupported fork".
// Defined in the Engine API specification.
const engineErrUnsupportedFork = -38005

// Engine API method names for GetPayload versions.
const (
	forkchoiceUpdatedV3Method = "engine_forkchoiceUpdatedV3"
	forkchoiceUpdatedV4Method = "engine_forkchoiceUpdatedV4"
	getPayloadV4Method        = "engine_getPayloadV4"
	getPayloadV5Method        = "engine_getPayloadV5"
	getPayloadV6Method        = "engine_getPayloadV6"
	newPayloadV4Method        = "engine_newPayloadV4"
	newPayloadV5Method        = "engine_newPayloadV5"
)

var _ EngineRPCClient = (*engineRPCClient)(nil)

type enginePayloadVersion uint32

const (
	enginePayloadVersionV4 enginePayloadVersion = 4
	enginePayloadVersionV5 enginePayloadVersion = 5
	enginePayloadVersionV6 enginePayloadVersion = 6
)

type engineForkchoiceVersion uint32

const (
	engineForkchoiceVersionV3 engineForkchoiceVersion = 3
	engineForkchoiceVersionV4 engineForkchoiceVersion = 4
)

// engineRPCClient is the concrete implementation wrapping *rpc.Client.
// It auto-detects whether to use engine_getPayloadV4, engine_getPayloadV5, or
// engine_getPayloadV6 by caching the last successful version and falling back on
// "Unsupported fork" errors.
type engineRPCClient struct {
	client *rpc.Client
	// payloadVersion tracks the preferred GetPayload version. The zero value
	// means V4, which keeps the default Prague path unchanged.
	payloadVersion atomic.Uint32
	// forkchoiceVersion tracks the preferred ForkchoiceUpdated version. The zero
	// value means V3, which keeps the default pre-Amsterdam path unchanged.
	forkchoiceVersion atomic.Uint32
}

// NewEngineRPCClient creates a new Engine API client.
func NewEngineRPCClient(client *rpc.Client) EngineRPCClient {
	return &engineRPCClient{client: client}
}

func (e *engineRPCClient) ForkchoiceUpdated(ctx context.Context, state engine.ForkchoiceStateV1, args map[string]any) (*engine.ForkChoiceResponse, error) {
	preferredVersion := e.preferredForkchoiceVersion()
	versions := forkchoiceFallbackOrder(preferredVersion)

	var unsupportedMethods []string
	var lastUnsupportedErr error
	for _, version := range versions {
		method := version.forkchoiceUpdatedMethod()
		methodArgs, err := forkchoiceArgsForMethod(args, method)
		if err != nil {
			return nil, err
		}

		var result engine.ForkChoiceResponse
		err = e.client.CallContext(ctx, &result, method, state, methodArgs)
		if err == nil {
			e.forkchoiceVersion.Store(uint32(version))
			if version == engineForkchoiceVersionV4 {
				e.payloadVersion.Store(uint32(enginePayloadVersionV6))
			}
			return &result, nil
		}

		if !isUnsupportedForkErr(err) {
			if len(unsupportedMethods) == 0 {
				return nil, fmt.Errorf("%s failed: %w", method, err)
			}
			return nil, fmt.Errorf("%s fallback after unsupported fork from %v: %w", method, unsupportedMethods, err)
		}

		unsupportedMethods = append(unsupportedMethods, method)
		lastUnsupportedErr = err
	}

	return nil, fmt.Errorf("forkchoice update unsupported fork after trying %v: %w", unsupportedMethods, lastUnsupportedErr)
}

func (e *engineRPCClient) GetPayload(ctx context.Context, payloadID engine.PayloadID) (*EnginePayloadEnvelope, error) {
	preferredVersion := e.preferredPayloadVersion()
	versions := getPayloadFallbackOrder(preferredVersion)

	var unsupportedMethods []string
	var lastUnsupportedErr error
	for _, version := range versions {
		method := version.getPayloadMethod()

		var result EnginePayloadEnvelope
		err := e.client.CallContext(ctx, &result, method, payloadID)
		if err == nil {
			e.payloadVersion.Store(uint32(version))
			return &result, nil
		}

		if !isUnsupportedForkErr(err) {
			if len(unsupportedMethods) == 0 {
				return nil, fmt.Errorf("%s payload %s: %w", method, payloadID, err)
			}
			return nil, fmt.Errorf("%s fallback after unsupported fork from %v, payload %s: %w", method, unsupportedMethods, payloadID, err)
		}

		unsupportedMethods = append(unsupportedMethods, method)
		lastUnsupportedErr = err
	}

	return nil, fmt.Errorf("get payload unsupported fork for payload %s after trying %v: %w", payloadID, unsupportedMethods, lastUnsupportedErr)
}

// GetPayloadMethod returns the Engine API method name currently used by GetPayload.
// This allows wrappers (e.g. tracing) to report the resolved version.
func (e *engineRPCClient) GetPayloadMethod() string {
	return e.preferredPayloadVersion().getPayloadMethod()
}

// GetForkchoiceUpdatedMethod returns the currently preferred Engine API
// forkchoiceUpdated method. Tracing uses this to report the resolved version.
func (e *engineRPCClient) GetForkchoiceUpdatedMethod() string {
	return e.preferredForkchoiceVersion().forkchoiceUpdatedMethod()
}

func (e *engineRPCClient) NewPayload(ctx context.Context, payload *EnginePayloadEnvelope, blobHashes []string, parentBeaconBlockRoot string, executionRequests [][]byte) (*engine.PayloadStatusV1, error) {
	payloadParam, err := payload.executionPayloadParam()
	if err != nil {
		return nil, err
	}

	var result engine.PayloadStatusV1
	err = e.client.CallContext(ctx, &result, newPayloadMethod(payload), payloadParam, blobHashes, parentBeaconBlockRoot, executionRequests)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (e *engineRPCClient) preferredPayloadVersion() enginePayloadVersion {
	switch version := enginePayloadVersion(e.payloadVersion.Load()); version {
	case enginePayloadVersionV5, enginePayloadVersionV6:
		return version
	default:
		return enginePayloadVersionV4
	}
}

func (v enginePayloadVersion) getPayloadMethod() string {
	switch v {
	case enginePayloadVersionV5:
		return getPayloadV5Method
	case enginePayloadVersionV6:
		return getPayloadV6Method
	default:
		return getPayloadV4Method
	}
}

func getPayloadFallbackOrder(preferred enginePayloadVersion) []enginePayloadVersion {
	switch preferred {
	case enginePayloadVersionV5:
		return []enginePayloadVersion{enginePayloadVersionV5, enginePayloadVersionV6, enginePayloadVersionV4}
	case enginePayloadVersionV6:
		return []enginePayloadVersion{enginePayloadVersionV6, enginePayloadVersionV5, enginePayloadVersionV4}
	default:
		return []enginePayloadVersion{enginePayloadVersionV4, enginePayloadVersionV5, enginePayloadVersionV6}
	}
}

func (e *engineRPCClient) preferredForkchoiceVersion() engineForkchoiceVersion {
	switch version := engineForkchoiceVersion(e.forkchoiceVersion.Load()); version {
	case engineForkchoiceVersionV4:
		return version
	default:
		return engineForkchoiceVersionV3
	}
}

func (v engineForkchoiceVersion) forkchoiceUpdatedMethod() string {
	if v == engineForkchoiceVersionV4 {
		return forkchoiceUpdatedV4Method
	}
	return forkchoiceUpdatedV3Method
}

func forkchoiceFallbackOrder(preferred engineForkchoiceVersion) []engineForkchoiceVersion {
	if preferred == engineForkchoiceVersionV4 {
		return []engineForkchoiceVersion{engineForkchoiceVersionV4, engineForkchoiceVersionV3}
	}
	return []engineForkchoiceVersion{engineForkchoiceVersionV3, engineForkchoiceVersionV4}
}

func forkchoiceArgsForMethod(args map[string]any, method string) (map[string]any, error) {
	if args == nil {
		return nil, nil
	}

	methodArgs := cloneMap(args)
	switch method {
	case forkchoiceUpdatedV4Method:
		if _, ok := methodArgs["slotNumber"]; !ok {
			return nil, fmt.Errorf("%s requires slotNumber payload attribute", method)
		}
	case forkchoiceUpdatedV3Method:
		delete(methodArgs, "slotNumber")
	}
	return methodArgs, nil
}

func cloneMap(args map[string]any) map[string]any {
	cloned := make(map[string]any, len(args))
	for key, value := range args {
		cloned[key] = value
	}
	return cloned
}

func newPayloadMethod(payload *EnginePayloadEnvelope) string {
	if payload.hasAmsterdamFields() {
		return newPayloadV5Method
	}
	return newPayloadV4Method
}

// isUnsupportedForkErr reports whether err is an Engine API "Unsupported fork"
// JSON-RPC error (code -38005).
func isUnsupportedForkErr(err error) bool {
	var rpcErr rpc.Error
	return errors.As(err, &rpcErr) && rpcErr.ErrorCode() == engineErrUnsupportedFork
}
