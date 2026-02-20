package node

// NOTE: This mirrors the exported JSON shape of celestia-node/state/tx_config.go
// at release v0.28.4, pared down to avoid importing Cosmos-SDK and
// celestia-app packages. See pkg/da/jsonrpc/README.md for rationale and sync tips.

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
