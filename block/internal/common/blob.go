package common

import (
	"encoding/binary"
	"fmt"

	"github.com/evstack/ev-node/types"
)

const blobMagic uint32 = 0x45564E44

// MarshalBlockBlob encodes a signed header, data and envelope signature into a
// single binary blob using a custom length-prefixed format.
// Layout: [magic 4B][header_len 4B][header][data_len 4B][data][sig_len 4B][sig]
func MarshalBlockBlob(header *types.SignedHeader, data *types.Data, envelopeSig []byte) ([]byte, error) {
	headerBz, err := header.MarshalBinary()
	if err != nil {
		return nil, fmt.Errorf("marshal header: %w", err)
	}
	dataBz, err := data.MarshalBinary()
	if err != nil {
		return nil, fmt.Errorf("marshal data: %w", err)
	}

	size := 4 + 4 + len(headerBz) + 4 + len(dataBz) + 4 + len(envelopeSig)
	buf := make([]byte, 0, size)

	buf = binary.BigEndian.AppendUint32(buf, blobMagic)
	buf = binary.BigEndian.AppendUint32(buf, uint32(len(headerBz)))
	buf = append(buf, headerBz...)
	buf = binary.BigEndian.AppendUint32(buf, uint32(len(dataBz)))
	buf = append(buf, dataBz...)
	buf = binary.BigEndian.AppendUint32(buf, uint32(len(envelopeSig)))
	buf = append(buf, envelopeSig...)

	return buf, nil
}

// UnmarshalBlockBlob decodes a combined block blob produced by MarshalBlockBlob.
func UnmarshalBlockBlob(bz []byte) (*types.SignedHeader, *types.Data, []byte, error) {
	if len(bz) < 4 {
		return nil, nil, nil, fmt.Errorf("blob too short: %d", len(bz))
	}
	if magic := binary.BigEndian.Uint32(bz[:4]); magic != blobMagic {
		return nil, nil, nil, fmt.Errorf("invalid blob magic: %x", magic)
	}
	off := 4

	header, off, err := readBlobField(bz, off, "header")
	if err != nil {
		return nil, nil, nil, err
	}
	var signedHeader types.SignedHeader
	if err := signedHeader.UnmarshalBinary(header); err != nil {
		return nil, nil, nil, fmt.Errorf("unmarshal header: %w", err)
	}

	dataBz, off, err := readBlobField(bz, off, "data")
	if err != nil {
		return nil, nil, nil, err
	}
	var data types.Data
	if err := data.UnmarshalBinary(dataBz); err != nil {
		return nil, nil, nil, fmt.Errorf("unmarshal data: %w", err)
	}

	var envelopeSig []byte
	if off+4 <= len(bz) {
		sig, _, sigErr := readBlobField(bz, off, "signature")
		if sigErr == nil {
			envelopeSig = sig
		}
	}

	return &signedHeader, &data, envelopeSig, nil
}

func readBlobField(bz []byte, off int, name string) ([]byte, int, error) {
	if off+4 > len(bz) {
		return nil, off, fmt.Errorf("truncated %s length at offset %d", name, off)
	}
	fieldLen := int(binary.BigEndian.Uint32(bz[off : off+4]))
	off += 4
	if off+fieldLen > len(bz) {
		return nil, off, fmt.Errorf("truncated %s data: need %d, have %d", name, fieldLen, len(bz)-off)
	}
	field := bz[off : off+fieldLen]
	off += fieldLen
	return field, off, nil
}
