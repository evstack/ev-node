package types

import (
	"errors"

	"github.com/celestiaorg/go-header"
	"google.golang.org/protobuf/proto"

	pb "github.com/evstack/ev-node/types/pb/evnode/v1"
)

var _ header.Header[*P2PSignedHeader] = &P2PSignedHeader{}

type P2PSignedHeader struct {
	SignedHeader
	DAHeightHint uint64
}

func (sh *P2PSignedHeader) New() *P2PSignedHeader {
	return new(P2PSignedHeader)
}

func (sh *P2PSignedHeader) IsZero() bool {
	return sh == nil || sh.SignedHeader.IsZero()
}

func (sh *P2PSignedHeader) Verify(untrstH *P2PSignedHeader) error {
	return sh.SignedHeader.Verify(&untrstH.SignedHeader)
}

func (sh *P2PSignedHeader) SetDAHint(daHeight uint64) {
	sh.DAHeightHint = daHeight
}

func (sh *P2PSignedHeader) DAHint() uint64 {
	return sh.DAHeightHint
}

func (sh *P2PSignedHeader) MarshalBinary() ([]byte, error) {
	msg, err := sh.ToProto()
	if err != nil {
		return nil, err
	}
	return proto.Marshal(msg)
}

func (sh *P2PSignedHeader) UnmarshalBinary(data []byte) error {
	var pHeader pb.P2PSignedHeader
	if err := proto.Unmarshal(data, &pHeader); err != nil {
		return err
	}
	return sh.FromProto(&pHeader)
}

func (sh *P2PSignedHeader) ToProto() (*pb.P2PSignedHeader, error) {
	psh, err := sh.SignedHeader.ToProto()
	if err != nil {
		return nil, err
	}
	return &pb.P2PSignedHeader{
		Header:       psh.Header,
		Signature:    psh.Signature,
		Signer:       psh.Signer,
		DaHeightHint: &sh.DAHeightHint,
	}, nil
}

func (sh *P2PSignedHeader) FromProto(other *pb.P2PSignedHeader) error {
	if other == nil {
		return errors.New("P2PSignedHeader is nil")
	}
	// Reconstruct SignedHeader
	psh := &pb.SignedHeader{
		Header:    other.Header,
		Signature: other.Signature,
		Signer:    other.Signer,
	}
	if err := sh.SignedHeader.FromProto(psh); err != nil {
		return err
	}
	if other.DaHeightHint != nil {
		sh.DAHeightHint = *other.DaHeightHint
	}
	return nil
}
