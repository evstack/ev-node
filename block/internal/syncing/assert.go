package syncing

import (
	"errors"
	"fmt"

	"github.com/libp2p/go-libp2p/core/crypto"

	"github.com/evstack/ev-node/pkg/genesis"
	"github.com/evstack/ev-node/types"
)

func assertExpectedProposer(genesis genesis.Genesis, height uint64, proposerAddr []byte, pubKey crypto.PubKey) error {
	if err := genesis.ValidateProposer(height, proposerAddr, pubKey); err != nil {
		return fmt.Errorf("unexpected proposer at height %d: %w", height, err)
	}

	return nil
}

func assertValidSignedData(signedData *types.SignedData, genesis genesis.Genesis) error {
	if signedData == nil || signedData.Txs == nil {
		return errors.New("empty signed data")
	}

	if err := assertExpectedProposer(genesis, signedData.Height(), signedData.Signer.Address, signedData.Signer.PubKey); err != nil {
		return err
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
