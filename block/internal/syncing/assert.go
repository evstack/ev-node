package syncing

import (
	"errors"
	"fmt"

	"github.com/evstack/ev-node/pkg/genesis"
	"github.com/evstack/ev-node/types"
)

func assertValidSignedData(signedData *types.SignedData, genesis genesis.Genesis) error {
	if signedData == nil || signedData.Txs == nil {
		return errors.New("empty signed data")
	}
	if signedData.Signer.PubKey == nil {
		return errors.New("missing signer public key in signed data")
	}

	dataBytes, err := signedData.Data.MarshalBinary()
	if err != nil {
		return fmt.Errorf("failed to get signed data payload: %w", err)
	}

	valid, err := signedData.Signer.PubKey.Verify(dataBytes, signedData.Signature)
	if err != nil {
		return fmt.Errorf("failed to verify signature: %w", err)
	}

	if !valid {
		return fmt.Errorf("invalid signature")
	}

	return nil
}
