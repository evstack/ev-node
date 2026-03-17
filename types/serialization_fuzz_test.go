package types

import (
	"testing"

	"google.golang.org/protobuf/proto"

	pb "github.com/evstack/ev-node/types/pb/evnode/v1"
)

func FuzzHeaderUnmarshalBinary(f *testing.F) {
	// Seed with a valid marshalled header
	h := Header{}
	h.BaseHeader.Height = 1
	h.BaseHeader.ChainID = "test"
	h.BaseHeader.Time = 1000
	if data, err := h.MarshalBinary(); err == nil {
		f.Add(data)
	}
	f.Add([]byte{})
	f.Add([]byte{0xff, 0xff, 0xff})

	f.Fuzz(func(t *testing.T, data []byte) {
		var h Header
		if err := h.UnmarshalBinary(data); err != nil {
			return
		}
		// Round-trip: if unmarshal succeeds, marshal should not panic
		_, _ = h.MarshalBinary()
	})
}

func FuzzDataUnmarshalBinary(f *testing.F) {
	d := Data{Txs: Txs{[]byte("tx1"), []byte("tx2")}}
	if data, err := d.MarshalBinary(); err == nil {
		f.Add(data)
	}
	f.Add([]byte{})

	f.Fuzz(func(t *testing.T, data []byte) {
		var d Data
		if err := d.UnmarshalBinary(data); err != nil {
			return
		}
		_, _ = d.MarshalBinary()
	})
}

func FuzzSignedHeaderUnmarshalBinary(f *testing.F) {
	f.Add([]byte{})
	f.Add([]byte{0x0a, 0x00})

	f.Fuzz(func(t *testing.T, data []byte) {
		var sh SignedHeader
		if err := sh.UnmarshalBinary(data); err != nil {
			return
		}
		_, _ = sh.MarshalBinary()
	})
}

func FuzzSignedDataUnmarshalBinary(f *testing.F) {
	f.Add([]byte{})
	f.Add([]byte{0x0a, 0x00})

	f.Fuzz(func(t *testing.T, data []byte) {
		var sd SignedData
		if err := sd.UnmarshalBinary(data); err != nil {
			return
		}
		_, _ = sd.MarshalBinary()
	})
}

func FuzzDAEnvelopeUnmarshalBinary(f *testing.F) {
	f.Add([]byte{})

	f.Fuzz(func(t *testing.T, data []byte) {
		var sh SignedHeader
		if _, err := sh.UnmarshalDAEnvelope(data); err != nil {
			return
		}
		_, _ = sh.MarshalDAEnvelope(nil)
	})
}

func FuzzStateFromProto(f *testing.F) {
	s := State{ChainID: "test", LastBlockHeight: 1}
	if p, err := s.ToProto(); err == nil {
		if data, err := proto.Marshal(p); err == nil {
			f.Add(data)
		}
	}
	f.Add([]byte{})

	f.Fuzz(func(t *testing.T, data []byte) {
		var ps pb.State
		if err := proto.Unmarshal(data, &ps); err != nil {
			return
		}
		var s State
		_ = s.FromProto(&ps)
	})
}
