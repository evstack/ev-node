package sync

import (
	"time"

	"github.com/celestiaorg/go-header"
)

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
	return s.Entry.MarshalBinary()
}

func (s *DAHeightHintContainer[H]) UnmarshalBinary(data []byte) error {
	return s.Entry.UnmarshalBinary(data)
}
