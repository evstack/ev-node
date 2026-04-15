package types

import "unsafe"

// Tx represents transaction.
type Tx []byte

// Txs represents a slice of transactions.
type Txs []Tx

// Compile-time guard for the unsafe []Tx <-> [][]byte reinterpretation used in
// hot serialization paths.
var (
	_ [unsafe.Sizeof(Tx{}) - unsafe.Sizeof([]byte{})]struct{}
	_ [unsafe.Sizeof([]byte{}) - unsafe.Sizeof(Tx{})]struct{}
)
