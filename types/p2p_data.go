package types

import (
	"errors"

	"github.com/celestiaorg/go-header"
	"google.golang.org/protobuf/proto"

	pb "github.com/evstack/ev-node/types/pb/evnode/v1"
)

var _ header.Header[*P2PData] = &P2PData{}

type P2PData struct {
	Data
	DAHeightHint uint64
}

func (d *P2PData) New() *P2PData {
	return new(P2PData)
}

func (d *P2PData) IsZero() bool {
	return d == nil || d.Data.IsZero()
}

func (d *P2PData) Verify(untrstD *P2PData) error {
	return d.Data.Verify(&untrstD.Data)
}

func (d *P2PData) SetDAHint(daHeight uint64) {
	d.DAHeightHint = daHeight
}

func (d *P2PData) DAHint() uint64 {
	return d.DAHeightHint
}

func (d *P2PData) MarshalBinary() ([]byte, error) {
	msg, err := d.ToProto()
	if err != nil {
		return nil, err
	}
	return proto.Marshal(msg)
}

func (d *P2PData) UnmarshalBinary(data []byte) error {
	var pData pb.P2PData
	if err := proto.Unmarshal(data, &pData); err != nil {
		return err
	}
	return d.FromProto(&pData)
}

func (d *P2PData) ToProto() (*pb.P2PData, error) {
	pData := d.Data.ToProto()
	return &pb.P2PData{
		Metadata:     pData.Metadata,
		Txs:          pData.Txs,
		DaHeightHint: &d.DAHeightHint,
	}, nil
}

func (d *P2PData) FromProto(other *pb.P2PData) error {
	if other == nil {
		return errors.New("P2PData is nil")
	}

	pData := &pb.Data{
		Metadata: other.Metadata,
		Txs:      other.Txs,
	}
	if err := d.Data.FromProto(pData); err != nil {
		return err
	}
	if other.DaHeightHint != nil {
		d.DAHeightHint = *other.DaHeightHint
	}
	return nil
}
