package types

import (
	"errors"
	"fmt"
	"time"

	"github.com/libp2p/go-libp2p/core/crypto"
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/evstack/ev-node/types/pb/evnode/v1"
)

// MarshalBinary encodes Metadata into binary form and returns it.
func (m *Metadata) MarshalBinary() ([]byte, error) {
	return proto.Marshal(m.ToProto())
}

// UnmarshalBinary decodes binary form of Metadata into object.
func (m *Metadata) UnmarshalBinary(metadata []byte) error {
	var pMetadata pb.Metadata
	err := proto.Unmarshal(metadata, &pMetadata)
	if err != nil {
		return err
	}
	return m.FromProto(&pMetadata)
}

// MarshalBinary encodes Header into binary form and returns it.
func (h *Header) MarshalBinary() ([]byte, error) {
	return proto.Marshal(h.ToProto())
}

// MarshalBinaryLegacy returns the legacy header encoding that includes the
// deprecated fields.
func (h *Header) MarshalBinaryLegacy() ([]byte, error) {
	return marshalLegacyHeader(h)
}

// UnmarshalBinary decodes binary form of Header into object.
func (h *Header) UnmarshalBinary(data []byte) error {
	var pHeader pb.Header
	err := proto.Unmarshal(data, &pHeader)
	if err != nil {
		return err
	}
	err = h.FromProto(&pHeader)
	return err
}

// MarshalBinary encodes Data into binary form and returns it.
func (d *Data) MarshalBinary() ([]byte, error) {
	return proto.Marshal(d.ToProto())
}

// UnmarshalBinary decodes binary form of Data into object.
func (d *Data) UnmarshalBinary(data []byte) error {
	var pData pb.Data
	err := proto.Unmarshal(data, &pData)
	if err != nil {
		return err
	}
	err = d.FromProto(&pData)
	return err
}

// ToProto converts SignedHeader into protobuf representation and returns it.
func (sh *SignedHeader) ToProto() (*pb.SignedHeader, error) {
	if sh.Signer.PubKey == nil {
		return &pb.SignedHeader{
			Header:    sh.Header.ToProto(),
			Signature: sh.Signature[:],
			Signer:    &pb.Signer{},
		}, nil
	}

	pubKey, err := crypto.MarshalPublicKey(sh.Signer.PubKey)
	if err != nil {
		return nil, err
	}
	return &pb.SignedHeader{
		Header:    sh.Header.ToProto(),
		Signature: sh.Signature[:],
		Signer: &pb.Signer{
			Address: sh.Signer.Address,
			PubKey:  pubKey,
		},
	}, nil
}

// FromProto fills SignedHeader with data from protobuf representation. The contained
// Signer can only be used to verify signatures, not to sign messages.
func (sh *SignedHeader) FromProto(other *pb.SignedHeader) error {
	if other == nil {
		return errors.New("signed header is nil")
	}
	if other.Header == nil {
		return errors.New("signed header's Header is nil")
	}
	if err := sh.Header.FromProto(other.Header); err != nil {
		return err
	}
	if other.Signature != nil {
		sh.Signature = make([]byte, len(other.Signature))
		copy(sh.Signature, other.Signature)
	} else {
		sh.Signature = nil
	}
	if other.Signer != nil && len(other.Signer.PubKey) > 0 {
		pubKey, err := crypto.UnmarshalPublicKey(other.Signer.PubKey)
		if err != nil {
			return err
		}
		sh.Signer = Signer{
			Address: append([]byte(nil), other.Signer.Address...),
			PubKey:  pubKey,
		}
	} else {
		sh.Signer = Signer{}
	}
	return nil
}

// MarshalBinary encodes SignedHeader into binary form and returns it.
func (sh *SignedHeader) MarshalBinary() ([]byte, error) {
	hp, err := sh.ToProto()
	if err != nil {
		return nil, err
	}
	return proto.Marshal(hp)
}

// UnmarshalBinary decodes binary form of SignedHeader into object.
func (sh *SignedHeader) UnmarshalBinary(data []byte) error {
	var pHeader pb.SignedHeader
	err := proto.Unmarshal(data, &pHeader)
	if err != nil {
		return err
	}
	err = sh.FromProto(&pHeader)
	if err != nil {
		return err
	}
	return nil
}

// ToDAEnvelopeProto converts SignedHeader into DAHeaderEnvelope protobuf representation.
func (sh *SignedHeader) ToDAEnvelopeProto(envelopeSignature []byte) (*pb.DAHeaderEnvelope, error) {
	if sh.Signer.PubKey == nil {
		return &pb.DAHeaderEnvelope{
			Header:            sh.Header.ToProto(),
			Signature:         sh.Signature[:],
			Signer:            &pb.Signer{},
			EnvelopeSignature: envelopeSignature,
		}, nil
	}

	pubKey, err := crypto.MarshalPublicKey(sh.Signer.PubKey)
	if err != nil {
		return nil, err
	}
	return &pb.DAHeaderEnvelope{
		Header:    sh.Header.ToProto(),
		Signature: sh.Signature[:],
		Signer: &pb.Signer{
			Address: sh.Signer.Address,
			PubKey:  pubKey,
		},
		EnvelopeSignature: envelopeSignature,
	}, nil
}

// FromDAEnvelopeProto fills SignedHeader with data from DAHeaderEnvelope protobuf representation.
// It assumes the envelope signature has already been extracted/verified if needed.
func (sh *SignedHeader) FromDAEnvelopeProto(envelope *pb.DAHeaderEnvelope) error {
	if envelope == nil {
		return errors.New("da header envelope is nil")
	}
	if envelope.Header == nil {
		return errors.New("da header envelope's Header is nil")
	}
	if err := sh.Header.FromProto(envelope.Header); err != nil {
		return err
	}
	if envelope.Signature != nil {
		sh.Signature = make([]byte, len(envelope.Signature))
		copy(sh.Signature, envelope.Signature)
	} else {
		sh.Signature = nil
	}
	if envelope.Signer != nil && len(envelope.Signer.PubKey) > 0 {
		pubKey, err := crypto.UnmarshalPublicKey(envelope.Signer.PubKey)
		if err != nil {
			return err
		}
		sh.Signer = Signer{
			Address: append([]byte(nil), envelope.Signer.Address...),
			PubKey:  pubKey,
		}
	} else {
		sh.Signer = Signer{}
	}
	return nil
}

// MarshalDAEnvelope encodes SignedHeader into DAHeaderEnvelope binary form.
func (sh *SignedHeader) MarshalDAEnvelope(envelopeSignature []byte) ([]byte, error) {
	envelope, err := sh.ToDAEnvelopeProto(envelopeSignature)
	if err != nil {
		return nil, err
	}
	return proto.Marshal(envelope)
}

// UnmarshalDAEnvelope decodes binary form of DAHeaderEnvelope into object and returns the envelope signature.
func (sh *SignedHeader) UnmarshalDAEnvelope(data []byte) ([]byte, error) {
	var envelope pb.DAHeaderEnvelope
	err := proto.Unmarshal(data, &envelope)
	if err != nil {
		return nil, err
	}
	err = sh.FromDAEnvelopeProto(&envelope)
	if err != nil {
		return nil, err
	}
	return envelope.EnvelopeSignature, nil
}

// ToProto converts Header into protobuf representation and returns it.
func (h *Header) ToProto() *pb.Header {
	pHeader := &pb.Header{
		Version: &pb.Version{
			Block: h.Version.Block,
			App:   h.Version.App,
		},
		Height:          h.BaseHeader.Height,
		Time:            h.BaseHeader.Time,
		LastHeaderHash:  h.LastHeaderHash[:],
		DataHash:        h.DataHash[:],
		AppHash:         h.AppHash[:],
		ProposerAddress: h.ProposerAddress[:],
		ChainId:         h.BaseHeader.ChainID,
		ValidatorHash:   h.ValidatorHash,
	}
	if unknown := encodeLegacyUnknownFields(h.Legacy); len(unknown) > 0 {
		pHeader.ProtoReflect().SetUnknown(unknown)
	}
	return pHeader
}

// FromProto fills Header with data from its protobuf representation.
func (h *Header) FromProto(other *pb.Header) error {
	if other == nil {
		return errors.New("header is nil")
	}
	if other.Version != nil {
		h.Version.Block = other.Version.Block
		h.Version.App = other.Version.App
	} else {
		h.Version = Version{}
	}
	h.BaseHeader.ChainID = other.GetChainId()
	h.BaseHeader.Height = other.GetHeight()
	h.BaseHeader.Time = other.GetTime()
	if other.LastHeaderHash != nil {
		h.LastHeaderHash = append([]byte(nil), other.LastHeaderHash...)
	} else {
		h.LastHeaderHash = nil
	}

	if other.DataHash != nil {
		h.DataHash = append([]byte(nil), other.DataHash...)
	} else {
		h.DataHash = nil
	}

	if other.AppHash != nil {
		h.AppHash = append([]byte(nil), other.AppHash...)
	} else {
		h.AppHash = nil
	}

	if other.ProposerAddress != nil {
		h.ProposerAddress = append([]byte(nil), other.ProposerAddress...)
	} else {
		h.ProposerAddress = nil
	}
	if other.ValidatorHash != nil {
		h.ValidatorHash = append([]byte(nil), other.ValidatorHash...)
	} else {
		h.ValidatorHash = nil
	}

	legacy, err := decodeLegacyHeaderFields(other)
	if err != nil {
		return err
	}
	h.Legacy = legacy

	return nil
}

// ToProto converts Metadata into protobuf representation and returns it.
func (m *Metadata) ToProto() *pb.Metadata {
	return &pb.Metadata{
		ChainId:      m.ChainID,
		Height:       m.Height,
		Time:         m.Time,
		LastDataHash: m.LastDataHash[:],
	}
}

// FromProto fills Metadata with data from its protobuf representation.
func (m *Metadata) FromProto(other *pb.Metadata) error {
	if other == nil {
		return errors.New("metadata is nil")
	}
	m.ChainID = other.GetChainId()
	m.Height = other.GetHeight()
	m.Time = other.GetTime()
	if other.LastDataHash != nil {
		m.LastDataHash = append([]byte(nil), other.LastDataHash...)
	} else {
		m.LastDataHash = nil
	}
	return nil
}

// ToProto converts Data into protobuf representation and returns it.
func (d *Data) ToProto() *pb.Data {
	var mProto *pb.Metadata
	if d.Metadata != nil {
		mProto = d.Metadata.ToProto()
	}
	return &pb.Data{
		Metadata: mProto,
		Txs:      txsToByteSlices(d.Txs),
	}
}

// FromProto fills the Data with data from its protobuf representation
func (d *Data) FromProto(other *pb.Data) error {
	if other == nil {
		return errors.New("data is nil")
	}
	if other.Metadata != nil {
		if d.Metadata == nil {
			d.Metadata = &Metadata{}
		}
		if err := d.Metadata.FromProto(other.Metadata); err != nil {
			return err
		}
	} else {
		d.Metadata = nil
	}
	d.Txs = byteSlicesToTxs(other.GetTxs())
	return nil
}

// ToProto converts State into protobuf representation and returns it.
func (s *State) ToProto() (*pb.State, error) {

	return &pb.State{
		Version: &pb.Version{
			Block: s.Version.Block,
			App:   s.Version.App,
		},
		ChainId:         s.ChainID,
		InitialHeight:   s.InitialHeight,
		LastBlockHeight: s.LastBlockHeight,
		LastBlockTime:   timestamppb.New(s.LastBlockTime),
		DaHeight:        s.DAHeight,
		AppHash:         s.AppHash[:],
		LastHeaderHash:  s.LastHeaderHash[:],
	}, nil
}

// FromProto fills State with data from its protobuf representation.
func (s *State) FromProto(other *pb.State) error {
	if other == nil {
		return errors.New("state is nil")
	}
	if other.Version != nil {
		s.Version = Version{
			Block: other.Version.Block,
			App:   other.Version.App,
		}
	} else {
		s.Version = Version{}
	}
	s.ChainID = other.GetChainId()
	s.InitialHeight = other.GetInitialHeight()
	s.LastBlockHeight = other.GetLastBlockHeight()
	if other.LastBlockTime != nil {
		s.LastBlockTime = other.LastBlockTime.AsTime()
	} else {
		s.LastBlockTime = time.Time{}
	}
	if other.AppHash != nil {
		s.AppHash = append([]byte(nil), other.AppHash...)
	} else {
		s.AppHash = nil
	}
	if other.LastHeaderHash != nil {
		s.LastHeaderHash = append([]byte(nil), other.LastHeaderHash...)
	} else {
		s.LastHeaderHash = nil
	}
	s.DAHeight = other.GetDaHeight()
	return nil
}

func txsToByteSlices(txs Txs) [][]byte {
	if txs == nil {
		return nil
	}
	bytes := make([][]byte, len(txs))
	for i := range txs {
		bytes[i] = txs[i]
	}
	return bytes
}

func byteSlicesToTxs(bytes [][]byte) Txs {
	if len(bytes) == 0 {
		return Txs{}
	}
	txs := make(Txs, len(bytes))
	for i := range txs {
		txs[i] = bytes[i]
	}
	return txs
}

// ToProto converts SignedData into protobuf representation and returns it.
func (sd *SignedData) ToProto() (*pb.SignedData, error) {
	dataProto := sd.Data.ToProto()

	var signerProto *pb.Signer
	if sd.Signer.PubKey != nil {
		pubKey, err := crypto.MarshalPublicKey(sd.Signer.PubKey)
		if err != nil {
			return nil, err
		}
		signerProto = &pb.Signer{
			Address: sd.Signer.Address,
			PubKey:  pubKey,
		}
	} else {
		signerProto = &pb.Signer{}
	}

	return &pb.SignedData{
		Data:      dataProto,
		Signature: sd.Signature[:],
		Signer:    signerProto,
	}, nil
}

// FromProto fills SignedData with data from protobuf representation.
func (sd *SignedData) FromProto(other *pb.SignedData) error {
	if other == nil {
		return errors.New("signed data is nil")
	}
	if other.Data != nil {
		if err := sd.Data.FromProto(other.Data); err != nil {
			return err
		}
	}
	if other.Signature != nil {
		sd.Signature = make([]byte, len(other.Signature))
		copy(sd.Signature, other.Signature)
	} else {
		sd.Signature = nil
	}
	if other.Signer != nil && len(other.Signer.PubKey) > 0 {
		pubKey, err := crypto.UnmarshalPublicKey(other.Signer.PubKey)
		if err != nil {
			return err
		}
		sd.Signer = Signer{
			Address: append([]byte(nil), other.Signer.Address...),
			PubKey:  pubKey,
		}
	} else {
		sd.Signer = Signer{}
	}
	return nil
}

// MarshalBinary encodes SignedData into binary form and returns it.
func (sd *SignedData) MarshalBinary() ([]byte, error) {
	p, err := sd.ToProto()
	if err != nil {
		return nil, err
	}
	return proto.Marshal(p)
}

// UnmarshalBinary decodes binary form of SignedData into object.
func (sd *SignedData) UnmarshalBinary(data []byte) error {
	var pData pb.SignedData
	err := proto.Unmarshal(data, &pData)
	if err != nil {
		return err
	}
	return sd.FromProto(&pData)
}

// Legacy protobuf field numbers for backwards compatibility
const (
	legacyLastCommitHashField  = 5
	legacyConsensusHashField   = 7
	legacyLastResultsHashField = 9
)

// Maximum size of unknown fields to prevent DoS attacks via malicious headers
// with excessive unknown field data. 1MB should be more than sufficient for
// legitimate legacy headers (typical header is ~500 bytes).
const maxUnknownFieldSize = 1024 * 1024 // 1MB

// Maximum size for individual legacy hash fields. Standard hashes are 32 bytes,
// but we allow up to 1KB for flexibility with different hash algorithms.
const maxLegacyHashSize = 1024 // 1KB

func decodeLegacyHeaderFields(pHeader *pb.Header) (*LegacyHeaderFields, error) {
	unknown := pHeader.ProtoReflect().GetUnknown()
	if len(unknown) == 0 {
		return nil, nil
	}

	// Protect against DoS attacks via headers with massive unknown field data
	if len(unknown) > maxUnknownFieldSize {
		return nil, fmt.Errorf("unknown fields too large: %d bytes (max %d)", len(unknown), maxUnknownFieldSize)
	}

	var legacy LegacyHeaderFields

	for len(unknown) > 0 {
		fieldNum, typ, n := protowire.ConsumeTag(unknown)
		if n < 0 {
			return nil, protowire.ParseError(n)
		}
		unknown = unknown[n:]

		switch fieldNum {
		case legacyLastCommitHashField, legacyConsensusHashField, legacyLastResultsHashField:
			if typ != protowire.BytesType {
				size := protowire.ConsumeFieldValue(fieldNum, typ, unknown)
				if size < 0 {
					return nil, protowire.ParseError(size)
				}
				unknown = unknown[size:]
				continue
			}

			value, size := protowire.ConsumeBytes(unknown)
			if size < 0 {
				return nil, protowire.ParseError(size)
			}
			unknown = unknown[size:]

			// Validate field size to prevent excessive memory allocation
			if len(value) > maxLegacyHashSize {
				return nil, fmt.Errorf("legacy hash field %d too large: %d bytes (max %d)",
					fieldNum, len(value), maxLegacyHashSize)
			}

			copied := append([]byte(nil), value...)

			switch fieldNum {
			case legacyLastCommitHashField:
				legacy.LastCommitHash = copied
			case legacyConsensusHashField:
				legacy.ConsensusHash = copied
			case legacyLastResultsHashField:
				legacy.LastResultsHash = copied
			}
		default:
			size := protowire.ConsumeFieldValue(fieldNum, typ, unknown)
			if size < 0 {
				return nil, protowire.ParseError(size)
			}
			unknown = unknown[size:]
		}
	}

	if legacy.IsZero() {
		return nil, nil
	}

	return &legacy, nil
}

func encodeLegacyUnknownFields(legacy *LegacyHeaderFields) []byte {
	if legacy == nil || legacy.IsZero() {
		return nil
	}

	var payload []byte

	if len(legacy.LastCommitHash) > 0 {
		payload = appendBytesField(payload, legacyLastCommitHashField, legacy.LastCommitHash)
	}

	if len(legacy.ConsensusHash) > 0 {
		payload = appendBytesField(payload, legacyConsensusHashField, legacy.ConsensusHash)
	}

	if len(legacy.LastResultsHash) > 0 {
		payload = appendBytesField(payload, legacyLastResultsHashField, legacy.LastResultsHash)
	}

	return payload
}

func appendBytesField(buf []byte, number protowire.Number, value []byte) []byte {
	buf = protowire.AppendTag(buf, number, protowire.BytesType)
	buf = protowire.AppendVarint(buf, uint64(len(value)))
	buf = append(buf, value...)
	return buf
}

func marshalLegacyHeader(h *Header) ([]byte, error) {
	if h == nil {
		return nil, errors.New("header is nil")
	}

	clone := h.Clone()
	clone.ApplyLegacyDefaults()

	var payload []byte

	// version
	versionBytes, err := proto.Marshal(&pb.Version{
		Block: clone.Version.Block,
		App:   clone.Version.App,
	})
	if err != nil {
		return nil, err
	}
	payload = protowire.AppendTag(payload, 1, protowire.BytesType)
	payload = protowire.AppendVarint(payload, uint64(len(versionBytes)))
	payload = append(payload, versionBytes...)

	// height
	payload = protowire.AppendTag(payload, 2, protowire.VarintType)
	payload = protowire.AppendVarint(payload, clone.BaseHeader.Height)

	// time
	payload = protowire.AppendTag(payload, 3, protowire.VarintType)
	payload = protowire.AppendVarint(payload, clone.BaseHeader.Time)

	// last header hash
	if len(clone.LastHeaderHash) > 0 {
		payload = appendBytesField(payload, 4, clone.LastHeaderHash)
	}

	// last commit hash (legacy)
	if len(clone.Legacy.LastCommitHash) > 0 {
		payload = appendBytesField(payload, legacyLastCommitHashField, clone.Legacy.LastCommitHash)
	}

	// data hash
	if len(clone.DataHash) > 0 {
		payload = appendBytesField(payload, 6, clone.DataHash)
	}

	// consensus hash (legacy)
	if len(clone.Legacy.ConsensusHash) > 0 {
		payload = appendBytesField(payload, legacyConsensusHashField, clone.Legacy.ConsensusHash)
	}

	// app hash
	if len(clone.AppHash) > 0 {
		payload = appendBytesField(payload, 8, clone.AppHash)
	}

	// last results hash (legacy)
	if len(clone.Legacy.LastResultsHash) > 0 {
		payload = appendBytesField(payload, legacyLastResultsHashField, clone.Legacy.LastResultsHash)
	}

	// proposer address
	if len(clone.ProposerAddress) > 0 {
		payload = appendBytesField(payload, 10, clone.ProposerAddress)
	}

	// validator hash
	if len(clone.ValidatorHash) > 0 {
		payload = appendBytesField(payload, 11, clone.ValidatorHash)
	}

	// chain ID
	if len(clone.BaseHeader.ChainID) > 0 {
		payload = protowire.AppendTag(payload, 12, protowire.BytesType)
		payload = protowire.AppendVarint(payload, uint64(len(clone.BaseHeader.ChainID)))
		payload = append(payload, clone.BaseHeader.ChainID...)
	}

	return payload, nil
}
