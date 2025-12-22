package types

import (
	"fmt"
	"time"

	"github.com/celestiaorg/go-header"
	"google.golang.org/protobuf/proto"

	pb "github.com/evstack/ev-node/types/pb/evnode/v1"
)

type (
	P2PSignedHeader = P2PEnvelope[*SignedHeader]
	P2PData         = P2PEnvelope[*Data]
)

var (
	_ header.Header[*P2PData]         = &P2PData{}
	_ header.Header[*P2PSignedHeader] = &P2PSignedHeader{}
)

// P2PEnvelope is a generic envelope for P2P messages that includes a DA height hint.
type P2PEnvelope[H header.Header[H]] struct {
	Message      H
	DAHeightHint uint64
}

// New creates a new P2PEnvelope.
func (e *P2PEnvelope[H]) New() *P2PEnvelope[H] {
	var empty H
	return &P2PEnvelope[H]{Message: empty.New()}
}

// IsZero checks if the envelope or its message is zero.
func (e *P2PEnvelope[H]) IsZero() bool {
	return e == nil || e.Message.IsZero()
}

// SetDAHint sets the DA height hint.
func (e *P2PEnvelope[H]) SetDAHint(daHeight uint64) {
	e.DAHeightHint = daHeight
}

// DAHint returns the DA height hint.
func (e *P2PEnvelope[H]) DAHint() uint64 {
	return e.DAHeightHint
}

// Verify verifies the envelope message against an untrusted envelope.
func (e *P2PEnvelope[H]) Verify(untrst *P2PEnvelope[H]) error {
	return e.Message.Verify(untrst.Message)
}

// ChainID returns the ChainID of the message.
func (e *P2PEnvelope[H]) ChainID() string {
	return e.Message.ChainID()
}

// Height returns the Height of the message.
func (e *P2PEnvelope[H]) Height() uint64 {
	return e.Message.Height()
}

// LastHeader returns the LastHeader hash of the message.
func (e *P2PEnvelope[H]) LastHeader() Hash {
	return e.Message.LastHeader()
}

// Time returns the Time of the message.
func (e *P2PEnvelope[H]) Time() time.Time {
	return e.Message.Time()
}

// Hash returns the hash of the message.
func (e *P2PEnvelope[H]) Hash() Hash {
	return e.Message.Hash()
}

// Validate performs basic validation on the message.
func (e *P2PEnvelope[H]) Validate() error {
	return e.Message.Validate()
}

// MarshalBinary marshals the envelope to binary.
func (e *P2PEnvelope[H]) MarshalBinary() ([]byte, error) {
	var mirrorPb proto.Message

	switch msg := any(e.Message).(type) {
	case *Data:
		pData := msg.ToProto()
		mirrorPb = &pb.P2PData{
			Metadata:     pData.Metadata,
			Txs:          pData.Txs,
			DaHeightHint: &e.DAHeightHint,
		}
	case *SignedHeader:
		psh, err := msg.ToProto()
		if err != nil {
			return nil, err
		}
		mirrorPb = &pb.P2PSignedHeader{
			Header:       psh.Header,
			Signature:    psh.Signature,
			Signer:       psh.Signer,
			DaHeightHint: &e.DAHeightHint,
		}
	default:
		return nil, fmt.Errorf("unsupported type for toProto: %T", msg)
	}
	return proto.Marshal(mirrorPb)
}

// UnmarshalBinary unmarshals the envelope from binary.
func (e *P2PEnvelope[H]) UnmarshalBinary(data []byte) error {
	switch target := any(e.Message).(type) {
	case *Data:
		var pData pb.P2PData
		if err := proto.Unmarshal(data, &pData); err != nil {
			return err
		}
		mirrorData := &pb.Data{
			Metadata: pData.Metadata,
			Txs:      pData.Txs,
		}
		if err := target.FromProto(mirrorData); err != nil {
			return err
		}
		if pData.DaHeightHint != nil {
			e.DAHeightHint = *pData.DaHeightHint
		}
		return nil
	case *SignedHeader:
		var pHeader pb.P2PSignedHeader
		if err := proto.Unmarshal(data, &pHeader); err != nil {
			return err
		}
		psh := &pb.SignedHeader{
			Header:    pHeader.Header,
			Signature: pHeader.Signature,
			Signer:    pHeader.Signer,
		}
		if err := target.FromProto(psh); err != nil {
			return err
		}
		if pHeader.DaHeightHint != nil {
			e.DAHeightHint = *pHeader.DaHeightHint
		}
		return nil
	default:
		return fmt.Errorf("unsupported type for UnmarshalBinary: %T", target)
	}
}
