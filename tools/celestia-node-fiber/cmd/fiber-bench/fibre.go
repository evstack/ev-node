package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	appfibre "github.com/celestiaorg/celestia-app/v9/fibre"
	libshare "github.com/celestiaorg/go-square/v4/share"

	"github.com/celestiaorg/celestia-node/blob"
	"github.com/celestiaorg/celestia-node/fibre"
	blobapi "github.com/celestiaorg/celestia-node/nodebuilder/blob"
	nodebuilderfibre "github.com/celestiaorg/celestia-node/nodebuilder/fibre"
	"github.com/celestiaorg/celestia-node/state/txclient"

	"github.com/evstack/ev-node/block"
	cnfiber "github.com/evstack/ev-node/tools/celestia-node-fiber"
)

// buildFibreAdapter constructs a celestia-node-fiber Adapter that talks
// directly to consensus gRPC + FSPs — no bridge node hop. We do this by
// rebuilding only the submit-side wiring of celestia-node's api/client
// (which is otherwise eager about dialing BridgeDAAddr in NewReadClient).
//
// The returned adapter only supports Upload (and Download via FSPs).
// Listen would invoke a stub blob.Subscribe that returns an error;
// ev-node's aggregator-only setup never calls it (no syncer, no based
// sequencer), so this is fine.
//
// The returned closer releases the gRPC connection and stops the
// underlying app-level fibre client.
func buildFibreAdapter(
	ctx context.Context,
	consensusGRPC string,
	keyName string,
	kr keyring.Keyring,
) (block.FiberClient, func() error, error) {
	if consensusGRPC == "" {
		return nil, nil, errors.New("consensus gRPC address is required")
	}
	if keyName == "" {
		return nil, nil, errors.New("key name is required")
	}

	conn, err := grpc.NewClient(
		consensusGRPC,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("dial consensus grpc %q: %w", consensusGRPC, err)
	}

	tc, err := txclient.NewTxClient(kr, keyName, conn)
	if err != nil {
		_ = conn.Close()
		return nil, nil, fmt.Errorf("new tx client: %w", err)
	}
	if err := tc.Start(ctx); err != nil {
		_ = conn.Close()
		return nil, nil, fmt.Errorf("start tx client: %w", err)
	}

	appCfg := appfibre.DefaultClientConfig()
	appCfg.DefaultKeyName = keyName
	appCfg.StateAddress = conn.Target()
	appClient, err := appfibre.NewClient(kr, appCfg)
	if err != nil {
		_ = tc.Stop(ctx)
		_ = conn.Close()
		return nil, nil, fmt.Errorf("new app fibre client: %w", err)
	}
	if err := appClient.Start(ctx); err != nil {
		_ = tc.Stop(ctx)
		_ = conn.Close()
		return nil, nil, fmt.Errorf("start app fibre client: %w", err)
	}

	accClient := fibre.NewAccountClient(tc, conn)
	svc := fibre.NewService(appClient, tc, accClient)
	module := nodebuilderfibre.NewModule(svc)

	adapter := cnfiber.FromModules(module, noBridgeBlob{}, 0)

	closer := func() error {
		stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		var errs error
		if err := appClient.Stop(stopCtx); err != nil {
			errs = errors.Join(errs, err)
		}
		if err := tc.Stop(stopCtx); err != nil {
			errs = errors.Join(errs, err)
		}
		if err := conn.Close(); err != nil {
			errs = errors.Join(errs, err)
		}
		return errs
	}

	return adapter, closer, nil
}

// noBridgeBlob errors on every call. The only path that would invoke it
// is Listen→Subscribe, which our aggregator-only single-sequencer node
// never reaches. A clear error here surfaces an assumption break instead
// of a nil panic.
type noBridgeBlob struct{}

var _ blobapi.Module = noBridgeBlob{}

var errNoBridge = errors.New("fiber-bench: blob module not supported (running without a bridge node)")

func (noBridgeBlob) Submit(context.Context, []*blob.Blob, *blob.SubmitOptions) (uint64, error) {
	return 0, errNoBridge
}
func (noBridgeBlob) Get(context.Context, uint64, libshare.Namespace, blob.Commitment) (*blob.Blob, error) {
	return nil, errNoBridge
}
func (noBridgeBlob) GetAll(context.Context, uint64, []libshare.Namespace) ([]*blob.Blob, error) {
	return nil, errNoBridge
}
func (noBridgeBlob) GetProof(context.Context, uint64, libshare.Namespace, blob.Commitment) (*blob.Proof, error) {
	return nil, errNoBridge
}
func (noBridgeBlob) Included(context.Context, uint64, libshare.Namespace, *blob.Proof, blob.Commitment) (bool, error) {
	return false, errNoBridge
}
func (noBridgeBlob) GetCommitmentProof(context.Context, uint64, libshare.Namespace, []byte) (*blob.CommitmentProof, error) {
	return nil, errNoBridge
}
func (noBridgeBlob) Subscribe(context.Context, libshare.Namespace, uint64) (<-chan *blob.SubscriptionResponse, error) {
	return nil, errNoBridge
}
