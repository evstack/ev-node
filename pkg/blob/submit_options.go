package blob

import "encoding/json"

// TxPriority mirrors celestia-node/state.TxPriority to preserve JSON compatibility.
type TxPriority int

const (
	TxPriorityLow TxPriority = iota + 1
	TxPriorityMedium
	TxPriorityHigh
)

// SubmitOptions is a pared-down copy of celestia-node/state.TxConfig JSON shape.
// Only exported fields are marshalled to match the RPC expectation of the blob service.
type SubmitOptions struct {
	GasPrice          float64    `json:"gas_price,omitempty"`
	IsGasPriceSet     bool       `json:"is_gas_price_set,omitempty"`
	MaxGasPrice       float64    `json:"max_gas_price,omitempty"`
	Gas               uint64     `json:"gas,omitempty"`
	TxPriority        TxPriority `json:"tx_priority,omitempty"`
	KeyName           string     `json:"key_name,omitempty"`
	SignerAddress     string     `json:"signer_address,omitempty"`
	FeeGranterAddress string     `json:"fee_granter_address,omitempty"`
}

// MarshalJSON ensures we only emit the fields the remote RPC understands.
func (cfg SubmitOptions) MarshalJSON() ([]byte, error) {
	type jsonSubmitOptions struct {
		GasPrice          float64    `json:"gas_price,omitempty"`
		IsGasPriceSet     bool       `json:"is_gas_price_set,omitempty"`
		MaxGasPrice       float64    `json:"max_gas_price,omitempty"`
		Gas               uint64     `json:"gas,omitempty"`
		TxPriority        TxPriority `json:"tx_priority,omitempty"`
		KeyName           string     `json:"key_name,omitempty"`
		SignerAddress     string     `json:"signer_address,omitempty"`
		FeeGranterAddress string     `json:"fee_granter_address,omitempty"`
	}
	return json.Marshal(jsonSubmitOptions(cfg))
}

// UnmarshalJSON decodes the RPC shape back into SubmitOptions.
func (cfg *SubmitOptions) UnmarshalJSON(data []byte) error {
	type jsonSubmitOptions struct {
		GasPrice          float64    `json:"gas_price,omitempty"`
		IsGasPriceSet     bool       `json:"is_gas_price_set,omitempty"`
		MaxGasPrice       float64    `json:"max_gas_price,omitempty"`
		Gas               uint64     `json:"gas,omitempty"`
		TxPriority        TxPriority `json:"tx_priority,omitempty"`
		KeyName           string     `json:"key_name,omitempty"`
		SignerAddress     string     `json:"signer_address,omitempty"`
		FeeGranterAddress string     `json:"fee_granter_address,omitempty"`
	}

	var j jsonSubmitOptions
	if err := json.Unmarshal(data, &j); err != nil {
		return err
	}
	*cfg = SubmitOptions(j)
	return nil
}
