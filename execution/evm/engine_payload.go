package evm

import (
	"encoding/json"
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/beacon/engine"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

// EnginePayloadEnvelope mirrors engine.ExecutionPayloadEnvelope while preserving
// the raw executionPayload object for Amsterdam fields that go-ethereum does not
// yet expose on the typed envelope, such as blockAccessList.
type EnginePayloadEnvelope struct {
	ExecutionPayload    *engine.ExecutableData
	RawExecutionPayload json.RawMessage
	BlockValue          *big.Int
	BlobsBundle         *engine.BlobsBundle
	Requests            [][]byte
	Override            bool
	Witness             *hexutil.Bytes
}

func (e *EnginePayloadEnvelope) UnmarshalJSON(input []byte) error {
	type executionPayloadEnvelope struct {
		ExecutionPayload json.RawMessage `json:"executionPayload"`
		BlockValue       *hexutil.Big    `json:"blockValue"`
		BlobsBundle      *engine.BlobsBundle
		Requests         []hexutil.Bytes `json:"executionRequests"`
		Override         *bool           `json:"shouldOverrideBuilder"`
		Witness          *hexutil.Bytes  `json:"witness,omitempty"`
	}

	var dec executionPayloadEnvelope
	if err := json.Unmarshal(input, &dec); err != nil {
		return err
	}
	if len(dec.ExecutionPayload) == 0 {
		return errors.New("missing required field 'executionPayload' for EnginePayloadEnvelope")
	}

	var payload engine.ExecutableData
	if err := json.Unmarshal(dec.ExecutionPayload, &payload); err != nil {
		return err
	}
	e.ExecutionPayload = &payload
	e.RawExecutionPayload = append(e.RawExecutionPayload[:0], dec.ExecutionPayload...)

	if dec.BlockValue == nil {
		return errors.New("missing required field 'blockValue' for EnginePayloadEnvelope")
	}
	e.BlockValue = (*big.Int)(dec.BlockValue)
	e.BlobsBundle = dec.BlobsBundle

	if dec.Requests != nil {
		e.Requests = make([][]byte, len(dec.Requests))
		for i, request := range dec.Requests {
			e.Requests[i] = request
		}
	}
	if dec.Override != nil {
		e.Override = *dec.Override
	}
	e.Witness = dec.Witness
	return nil
}

func (e EnginePayloadEnvelope) MarshalJSON() ([]byte, error) {
	type executionPayloadEnvelope struct {
		ExecutionPayload json.RawMessage `json:"executionPayload"`
		BlockValue       *hexutil.Big    `json:"blockValue"`
		BlobsBundle      *engine.BlobsBundle
		Requests         []hexutil.Bytes `json:"executionRequests"`
		Override         bool            `json:"shouldOverrideBuilder"`
		Witness          *hexutil.Bytes  `json:"witness,omitempty"`
	}

	executionPayload, err := e.executionPayloadJSON()
	if err != nil {
		return nil, err
	}

	var requests []hexutil.Bytes
	if e.Requests != nil {
		requests = make([]hexutil.Bytes, len(e.Requests))
		for i, request := range e.Requests {
			requests[i] = request
		}
	}

	return json.Marshal(executionPayloadEnvelope{
		ExecutionPayload: executionPayload,
		BlockValue:       (*hexutil.Big)(e.BlockValue),
		BlobsBundle:      e.BlobsBundle,
		Requests:         requests,
		Override:         e.Override,
		Witness:          e.Witness,
	})
}

func (e *EnginePayloadEnvelope) executionPayloadParam() (any, error) {
	if e == nil {
		return nil, errors.New("nil EnginePayloadEnvelope")
	}
	payload, err := e.executionPayloadJSON()
	if err != nil {
		return nil, err
	}
	return payload, nil
}

func (e *EnginePayloadEnvelope) executionPayloadJSON() (json.RawMessage, error) {
	if e == nil {
		return nil, errors.New("nil EnginePayloadEnvelope")
	}
	if len(e.RawExecutionPayload) > 0 {
		return json.RawMessage(e.RawExecutionPayload), nil
	}
	if e.ExecutionPayload == nil {
		return nil, errors.New("nil execution payload")
	}
	payload, err := json.Marshal(e.ExecutionPayload)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(payload), nil
}

func (e *EnginePayloadEnvelope) hasAmsterdamFields() bool {
	if e == nil {
		return false
	}
	if e.ExecutionPayload != nil && e.ExecutionPayload.SlotNumber != nil {
		return true
	}
	return rawExecutionPayloadHasField(e.RawExecutionPayload, "slotNumber") ||
		rawExecutionPayloadHasField(e.RawExecutionPayload, "blockAccessList")
}

func rawExecutionPayloadHasField(payload json.RawMessage, field string) bool {
	if len(payload) == 0 {
		return false
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(payload, &fields); err != nil {
		return false
	}
	_, ok := fields[field]
	return ok
}
