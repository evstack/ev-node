package celestia

import (
	"encoding/json"
	"fmt"

	"github.com/celestiaorg/nmt"

	"github.com/evstack/ev-node/da"
)

// Namespace represents a Celestia namespace (29 bytes: 1 version + 28 ID)
type Namespace []byte

// Commitment represents a blob commitment (merkle root)
type Commitment []byte

// Blob represents a Celestia blob with namespace and commitment
type Blob struct {
	Namespace  Namespace  `json:"namespace"`
	Data       []byte     `json:"data"`
	ShareVer   uint8      `json:"share_version"`
	Commitment Commitment `json:"commitment"`
	Signer     []byte     `json:"signer,omitempty"`
	Index      int        `json:"index"`
}

// Proof represents a Celestia inclusion proof
type Proof []*nmt.Proof

// SubmitOptions contains options for blob submission
type SubmitOptions struct {
	GasPrice          float64 `json:"gas_price,omitempty"`
	IsGasPriceSet     bool    `json:"is_gas_price_set,omitempty"`
	MaxGasPrice       float64 `json:"max_gas_price,omitempty"`
	Gas               uint64  `json:"gas,omitempty"`
	TxPriority        int     `json:"tx_priority,omitempty"`
	KeyName           string  `json:"key_name,omitempty"`
	SignerAddress     string  `json:"signer_address,omitempty"`
	FeeGranterAddress string  `json:"fee_granter_address,omitempty"`
}

// MarshalJSON implements json.Marshaler for Proof
func (p Proof) MarshalJSON() ([]byte, error) {
	return json.Marshal([]*nmt.Proof(p))
}

// UnmarshalJSON implements json.Unmarshaler for Proof
func (p *Proof) UnmarshalJSON(data []byte) error {
	var proofs []*nmt.Proof
	if err := json.Unmarshal(data, &proofs); err != nil {
		return err
	}
	*p = proofs
	return nil
}

// ValidateNamespace validates that a namespace is properly formatted (29 bytes).
func ValidateNamespace(ns Namespace) error {
	if len(ns) != da.NamespaceSize {
		return fmt.Errorf("invalid namespace size: got %d, expected %d", len(ns), da.NamespaceSize)
	}

	parsed, err := da.NamespaceFromBytes(ns)
	if err != nil {
		return fmt.Errorf("invalid namespace: %w", err)
	}

	if parsed.Version != da.NamespaceVersionZero || !parsed.IsValidForVersion0() {
		return fmt.Errorf("invalid namespace: only version 0 namespaces with first %d zero bytes are supported", da.NamespaceVersionZeroPrefixSize)
	}
	return nil
}
