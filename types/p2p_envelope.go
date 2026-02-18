package types

import (
	"time"

	"github.com/celestiaorg/go-header"
	"google.golang.org/protobuf/proto"

	pb "github.com/evstack/ev-node/types/pb/evnode/v1"
)

var (
	_ header.Header[*P2PData]         = &P2PData{}
	_ header.Header[*P2PSignedHeader] = &P2PSignedHeader{}
)

// P2PSignedHeader wraps SignedHeader with an optional DA height hint for P2P sync optimization.
// PrevHeaderHash carries the producer's pre-computed hash of block N-1's header,
// allowing peers to skip re-computing trusted.Hash() during Verify.
type P2PSignedHeader struct {
	*SignedHeader
	DAHeightHint   uint64
	PrevHeaderHash Hash
}

// New creates a new P2PSignedHeader.
func (p *P2PSignedHeader) New() *P2PSignedHeader {
	return &P2PSignedHeader{SignedHeader: new(SignedHeader)}
}

// IsZero checks if the header is nil or zero.
func (p *P2PSignedHeader) IsZero() bool {
	return p == nil || p.SignedHeader == nil || p.SignedHeader.IsZero()
}

// SetDAHint sets the DA height hint.
func (p *P2PSignedHeader) SetDAHint(daHeight uint64) {
	p.DAHeightHint = daHeight
}

// DAHint returns the DA height hint.
func (p *P2PSignedHeader) DAHint() uint64 {
	return p.DAHeightHint
}

// Verify verifies against an untrusted header.
// If the untrusted header carries a PrevHeaderHash (from the producer),
// pre-populate the trusted header's hash cache to skip re-computation.
func (p *P2PSignedHeader) Verify(untrusted *P2PSignedHeader) error {
	if len(untrusted.PrevHeaderHash) > 0 {
		p.Header.SetCachedHash(untrusted.PrevHeaderHash)
	}
	return p.SignedHeader.Verify(untrusted.SignedHeader)
}

// MarshalBinary marshals the header to binary using P2P protobuf format.
func (p *P2PSignedHeader) MarshalBinary() ([]byte, error) {
	psh, err := p.ToProto()
	if err != nil {
		return nil, err
	}
	msg := &pb.P2PSignedHeader{
		Header:       psh.Header,
		Signature:    psh.Signature,
		Signer:       psh.Signer,
		DaHeightHint: &p.DAHeightHint,
	}
	if len(p.PrevHeaderHash) > 0 {
		msg.PrevHeaderHash = p.PrevHeaderHash
	}
	return proto.Marshal(msg)
}

// UnmarshalBinary unmarshals the header from binary using P2P protobuf format.
func (p *P2PSignedHeader) UnmarshalBinary(data []byte) error {
	var msg pb.P2PSignedHeader
	if err := proto.Unmarshal(data, &msg); err != nil {
		return err
	}
	psh := &pb.SignedHeader{
		Header:    msg.Header,
		Signature: msg.Signature,
		Signer:    msg.Signer,
	}
	if p.SignedHeader == nil {
		p.SignedHeader = new(SignedHeader)
	}
	if err := p.FromProto(psh); err != nil {
		return err
	}
	if msg.DaHeightHint != nil {
		p.DAHeightHint = *msg.DaHeightHint
	}
	if len(msg.PrevHeaderHash) > 0 {
		p.PrevHeaderHash = msg.PrevHeaderHash
	}
	return nil
}

// P2PData wraps Data with an optional DA height hint for P2P sync optimization.
// PrevDataHash carries the producer's pre-computed hash of block N-1's data,
// allowing peers to skip re-computing trusted.Hash() during Verify.
type P2PData struct {
	*Data
	DAHeightHint uint64
	PrevDataHash Hash
}

// New creates a new P2PData.
func (p *P2PData) New() *P2PData {
	return &P2PData{Data: new(Data)}
}

// IsZero checks if the data is nil or zero.
func (p *P2PData) IsZero() bool {
	return p == nil || p.Data == nil || p.Data.IsZero()
}

// SetDAHint sets the DA height hint.
func (p *P2PData) SetDAHint(daHeight uint64) {
	p.DAHeightHint = daHeight
}

// DAHint returns the DA height hint.
func (p *P2PData) DAHint() uint64 {
	return p.DAHeightHint
}

// Verify verifies against untrusted data.
// If the untrusted data carries a PrevDataHash (from the producer),
// pre-populate the trusted data's hash cache to skip re-computation.
func (p *P2PData) Verify(untrusted *P2PData) error {
	if len(untrusted.PrevDataHash) > 0 {
		p.Data.SetCachedHash(untrusted.PrevDataHash)
	}
	return p.Data.Verify(untrusted.Data)
}

// ChainID returns chain ID of the data.
func (p *P2PData) ChainID() string {
	return p.Data.ChainID()
}

// Height returns height of the data.
func (p *P2PData) Height() uint64 {
	return p.Data.Height()
}

// LastHeader returns last header hash of the data.
func (p *P2PData) LastHeader() Hash {
	return p.Data.LastHeader()
}

// Time returns time of the data.
func (p *P2PData) Time() time.Time {
	return p.Data.Time()
}

// Hash returns the hash of the data.
func (p *P2PData) Hash() Hash {
	return p.Data.Hash()
}

// Validate performs basic validation on the data.
func (p *P2PData) Validate() error {
	return p.Data.Validate()
}

// MarshalBinary marshals the data to binary using P2P protobuf format.
func (p *P2PData) MarshalBinary() ([]byte, error) {
	pData := p.ToProto()
	msg := &pb.P2PData{
		Metadata:     pData.Metadata,
		Txs:          pData.Txs,
		DaHeightHint: &p.DAHeightHint,
	}
	if len(p.PrevDataHash) > 0 {
		msg.PrevDataHash = p.PrevDataHash
	}
	return proto.Marshal(msg)
}

// UnmarshalBinary unmarshals the data from binary using P2P protobuf format.
func (p *P2PData) UnmarshalBinary(data []byte) error {
	var msg pb.P2PData
	if err := proto.Unmarshal(data, &msg); err != nil {
		return err
	}
	pData := &pb.Data{
		Metadata: msg.Metadata,
		Txs:      msg.Txs,
	}
	if p.Data == nil {
		p.Data = new(Data)
	}
	if err := p.FromProto(pData); err != nil {
		return err
	}
	if msg.DaHeightHint != nil {
		p.DAHeightHint = *msg.DaHeightHint
	}
	if len(msg.PrevDataHash) > 0 {
		p.PrevDataHash = msg.PrevDataHash
	}
	return nil
}
