package executing

import "context"

// fetchIncludedTxs fetches transactions directly included in the DA.
func (e *Executor) fetchIncludedTxs(ctx context.Context) ([][]byte, error) {
	includedTxs := [][]byte{
		[]byte("tx1"),
		[]byte("tx2"),
	}

	return includedTxs, nil
}
