package celestia

import (
	"encoding/json"
	"fmt"
)

// Namespace represents a Celestia namespace (29 bytes: 1 version + 28 ID)
type Namespace []byte

// Commitment represents a blob commitment (merkle root)
type Commitment []byte

// Blob represents a Celestia blob with namespace and commitment
type Blob struct {
	Namespace  Namespace  `json:"namespace"`
	Data       []byte     `json:"data"`
	ShareVer   uint32     `json:"share_version"`
	Commitment Commitment `json:"commitment"`
	Index      int        `json:"index"`
}

// Proof represents a Celestia inclusion proof
type Proof struct {
	Data []byte `json:"data"`
}

// SubmitOptions contains options for blob submission
type SubmitOptions struct {
	Fee           float64 `json:"fee,omitempty"`
	GasLimit      uint64  `json:"gas_limit,omitempty"`
	SignerAddress string  `json:"signer_address,omitempty"`
}

// MarshalJSON implements json.Marshaler for Proof
func (p *Proof) MarshalJSON() ([]byte, error) {
	return json.Marshal(p.Data)
}

// UnmarshalJSON implements json.Unmarshaler for Proof
func (p *Proof) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &p.Data)
}

// ValidateNamespace validates that a namespace is properly formatted (29 bytes).
func ValidateNamespace(ns Namespace) error {
	const NamespaceSize = 29
	if len(ns) != NamespaceSize {
		return fmt.Errorf("invalid namespace size: got %d, expected %d", len(ns), NamespaceSize)
	}
	return nil
}
