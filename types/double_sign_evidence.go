package types

import (
	"bytes"
	"errors"
	"fmt"
	"time"

	"google.golang.org/protobuf/proto"

	pb "github.com/evstack/ev-node/types/pb/evnode/v1"
)

// Ingestion source identifying which path observed a SignedHeader.
const (
	EvidenceSourceP2P    = "p2p"
	EvidenceSourceDA     = "da"
	EvidenceSourceStored = "stored"
)

// DoubleSignEvidence records two validly-signed SignedHeaders at the same
// height produced by the sequencer. Persisted as proof of equivocation.
type DoubleSignEvidence struct {
	Height          uint64
	FirstHeader     *SignedHeader
	AlternateHeader *SignedHeader
	DetectedAt      time.Time
	FirstSource     string
	AlternateSource string
}

// ValidateBasic checks structural consistency of the evidence.
func (e *DoubleSignEvidence) ValidateBasic() error {
	if e == nil {
		return errors.New("evidence is nil")
	}
	if e.FirstHeader == nil || e.AlternateHeader == nil {
		return errors.New("evidence requires both first and alternate headers")
	}
	if e.FirstHeader.Height() != e.Height || e.AlternateHeader.Height() != e.Height {
		return fmt.Errorf("evidence height %d does not match both headers (%d, %d)",
			e.Height, e.FirstHeader.Height(), e.AlternateHeader.Height())
	}
	if bytes.Equal(e.FirstHeader.Hash(), e.AlternateHeader.Hash()) {
		return errors.New("evidence headers have identical hash — no equivocation")
	}
	if !bytes.Equal(e.FirstHeader.ProposerAddress, e.AlternateHeader.ProposerAddress) {
		return errors.New("evidence headers have different proposers — not an equivocation")
	}
	return nil
}

// ToProto converts DoubleSignEvidence to protobuf representation.
func (e *DoubleSignEvidence) ToProto() (*pb.DoubleSignEvidence, error) {
	if e == nil {
		return nil, errors.New("evidence is nil")
	}
	if e.FirstHeader == nil || e.AlternateHeader == nil {
		return nil, errors.New("evidence requires both first and alternate headers")
	}
	first, err := e.FirstHeader.ToProto()
	if err != nil {
		return nil, fmt.Errorf("marshal first header: %w", err)
	}
	alt, err := e.AlternateHeader.ToProto()
	if err != nil {
		return nil, fmt.Errorf("marshal alternate header: %w", err)
	}
	return &pb.DoubleSignEvidence{
		Height:          e.Height,
		FirstHeader:     first,
		AlternateHeader: alt,
		DetectedAt:      e.DetectedAt.UnixNano(),
		FirstSource:     e.FirstSource,
		AlternateSource: e.AlternateSource,
	}, nil
}

// FromProto fills DoubleSignEvidence from protobuf representation.
func (e *DoubleSignEvidence) FromProto(p *pb.DoubleSignEvidence) error {
	if p == nil {
		return errors.New("proto evidence is nil")
	}
	if p.FirstHeader == nil || p.AlternateHeader == nil {
		return errors.New("proto evidence missing first or alternate header")
	}
	first := new(SignedHeader)
	if err := first.FromProto(p.FirstHeader); err != nil {
		return fmt.Errorf("unmarshal first header: %w", err)
	}
	alt := new(SignedHeader)
	if err := alt.FromProto(p.AlternateHeader); err != nil {
		return fmt.Errorf("unmarshal alternate header: %w", err)
	}
	e.Height = p.Height
	e.FirstHeader = first
	e.AlternateHeader = alt
	e.DetectedAt = time.Unix(0, p.DetectedAt).UTC()
	e.FirstSource = p.FirstSource
	e.AlternateSource = p.AlternateSource
	return nil
}

// MarshalBinary encodes DoubleSignEvidence to protobuf bytes.
func (e *DoubleSignEvidence) MarshalBinary() ([]byte, error) {
	p, err := e.ToProto()
	if err != nil {
		return nil, err
	}
	return proto.Marshal(p)
}

// UnmarshalBinary decodes DoubleSignEvidence from protobuf bytes.
func (e *DoubleSignEvidence) UnmarshalBinary(data []byte) error {
	p := new(pb.DoubleSignEvidence)
	if err := proto.Unmarshal(data, p); err != nil {
		return fmt.Errorf("proto unmarshal double sign evidence: %w", err)
	}
	return e.FromProto(p)
}
