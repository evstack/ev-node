package sync

import (
	"encoding/binary"
	"fmt"
	"time"

	"github.com/celestiaorg/go-header"
	"github.com/evstack/ev-node/types"
)

type SignedHeaderWithDAHint = DAHeightHintContainer[*types.SignedHeader]
type DataWithDAHint = DAHeightHintContainer[*types.Data]

type DAHeightHintContainer[H header.Header[H]] struct {
	Entry        H
	DAHeightHint uint64
}

func (s *DAHeightHintContainer[H]) ChainID() string {
	return s.Entry.ChainID()
}

func (s *DAHeightHintContainer[H]) Hash() header.Hash {
	return s.Entry.Hash()
}

func (s *DAHeightHintContainer[H]) Height() uint64 {
	return s.Entry.Height()
}

func (s *DAHeightHintContainer[H]) LastHeader() header.Hash {
	return s.Entry.LastHeader()
}

func (s *DAHeightHintContainer[H]) Time() time.Time {
	return s.Entry.Time()
}

func (s *DAHeightHintContainer[H]) Validate() error {
	return s.Entry.Validate()
}

func (s *DAHeightHintContainer[H]) New() *DAHeightHintContainer[H] {
	var empty H
	return &DAHeightHintContainer[H]{Entry: empty.New()}
}

func (sh *DAHeightHintContainer[H]) Verify(untrstH *DAHeightHintContainer[H]) error {
	return sh.Entry.Verify(untrstH.Entry)
}

func (s *DAHeightHintContainer[H]) SetDAHint(daHeight uint64) {
	s.DAHeightHint = daHeight
}
func (s *DAHeightHintContainer[H]) DAHint() uint64 {
	return s.DAHeightHint
}

func (s *DAHeightHintContainer[H]) IsZero() bool {
	return s == nil
}

func (s *DAHeightHintContainer[H]) MarshalBinary() ([]byte, error) {
	bz, err := s.Entry.MarshalBinary()
	if err != nil {
		return nil, err
	}
	out := make([]byte, 8+len(bz))
	binary.BigEndian.PutUint64(out, s.DAHeightHint)
	copy(out[8:], bz)
	return out, nil
}

func (s *DAHeightHintContainer[H]) UnmarshalBinary(data []byte) error {
	if len(data) < 8 {
		return fmt.Errorf("invalid length: %d", len(data))
	}
	s.DAHeightHint = binary.BigEndian.Uint64(data)
	return s.Entry.UnmarshalBinary(data[8:])
}
