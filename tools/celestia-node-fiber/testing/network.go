//go:build fibre

package cnfibertest

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	sdkmath "cosmossdk.io/math"
	"github.com/cometbft/cometbft/privval"
	core "github.com/cometbft/cometbft/types"
	"github.com/cosmos/cosmos-sdk/client/grpc/cmtservice"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/stretchr/testify/require"

	"github.com/celestiaorg/celestia-app/v8/app"
	"github.com/celestiaorg/celestia-app/v8/app/encoding"
	appfibre "github.com/celestiaorg/celestia-app/v8/fibre"
	"github.com/celestiaorg/celestia-app/v8/pkg/appconsts"
	"github.com/celestiaorg/celestia-app/v8/pkg/user"
	"github.com/celestiaorg/celestia-app/v8/test/util/testnode"
	fibretypes "github.com/celestiaorg/celestia-app/v8/x/fibre/types"
	valtypes "github.com/celestiaorg/celestia-app/v8/x/valaddr/types"
)

const (
	// defaultChainID matches the celestia-node bridge test helper so the
	// bridge and consensus node agree on network identity.
	defaultChainID = "private"

	// escrowDeposit is a generous initial escrow. Uploads consume from
	// escrow for gas + blob fees, so leave headroom for multiple runs.
	escrowDeposit = 50_000_000 // 50 TIA in utia

	// clientAccount is the keyring account the adapter uses to sign
	// payment promises and MsgPayForFibre. Pre-funded in genesis; also
	// has a funded escrow after StartNetwork returns.
	clientAccount = appfibre.DefaultKeyName
)

// Network bundles a single-validator celestia-app chain, an in-process
// Fibre server registered via valaddr, and a funded escrow account. The
// caller's *testing.T cleanup stops everything.
type Network struct {
	// Consensus is the celestia-app testnode context (keyring, gRPC
	// client, home dir). Use Consensus.GRPCClient for consensus gRPC
	// access and Consensus.Keyring for signing.
	Consensus testnode.Context

	// FibreServer is the in-process Fibre gRPC server registered with
	// the chain's valaddr module. The adapter (via appfibre.Client's
	// default host registry) discovers it through valaddr.
	FibreServer *appfibre.Server

	// ChainID matches what the bridge node needs to be configured with.
	ChainID string

	// ClientAccount is the keyring name for the pre-funded + pre-escrowed
	// account the showcase uses. Expose it so the test wires SubmitConfig.
	ClientAccount string
}

// StartNetwork boots a single-validator Fibre chain + server and returns
// a ready-to-use Network. Registration + escrow funding is complete when
// this returns.
func StartNetwork(t *testing.T, ctx context.Context) *Network {
	t.Helper()

	cfg := testnode.DefaultConfig().
		WithChainID(defaultChainID).
		WithFundedAccounts(clientAccount).
		WithDelayedPrecommitTimeout(50 * time.Millisecond)

	cctx, _, grpcAddr := testnode.NewNetwork(t, cfg)
	_, err := cctx.WaitForHeight(1)
	require.NoError(t, err, "waiting for first block")

	server := startFibreServer(t, ctx, cctx, grpcAddr)
	// The Fibre client's gRPC dialer expects URI-style targets, so prefix
	// the host with the dns:/// scheme before registering — matches how
	// talis provisions real Fibre networks (tools/talis/fibre_setup.go).
	registerValidator(t, ctx, cctx, "dns:///"+server.ListenAddress())
	fundEscrow(t, ctx, cctx)

	return &Network{
		Consensus:     cctx,
		FibreServer:   server,
		ChainID:       defaultChainID,
		ClientAccount: clientAccount,
	}
}

// ConsensusGRPCAddr returns the host:port of the chain's gRPC endpoint
// that the adapter's SubmitConfig.CoreGRPCConfig should point at.
func (n *Network) ConsensusGRPCAddr() string {
	return n.Consensus.GRPCClient.Target()
}

// startFibreServer spins up an in-process Fibre gRPC server bound to an
// ephemeral localhost port. The server uses the testnode's private
// validator key for BLS signing, and keeps blob data in memory.
func startFibreServer(
	t *testing.T,
	ctx context.Context,
	cctx testnode.Context,
	appGRPCAddr string,
) *appfibre.Server {
	t.Helper()

	pvKey := filepath.Join(cctx.HomeDir, "config", "priv_validator_key.json")
	pvState := filepath.Join(cctx.HomeDir, "data", "priv_validator_state.json")
	filePV := privval.LoadFilePV(pvKey, pvState)

	serverCfg := appfibre.DefaultServerConfig()
	serverCfg.AppGRPCAddress = appGRPCAddr
	serverCfg.ServerListenAddress = "127.0.0.1:0"
	serverCfg.SignerFn = func(string) (core.PrivValidator, error) {
		return filePV, nil
	}
	serverCfg.StoreFn = func(sc appfibre.StoreConfig) (*appfibre.Store, error) {
		return appfibre.NewMemoryStore(sc), nil
	}

	server, err := appfibre.NewServer(serverCfg)
	require.NoError(t, err, "creating fibre server")
	require.NoError(t, server.Start(ctx), "starting fibre server")
	t.Cleanup(func() {
		stopCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = server.Stop(stopCtx)
	})
	return server
}

// registerValidator submits MsgSetFibreProviderInfo so the chain's
// valaddr module maps the validator's consensus address to the Fibre
// server's listen address. Without this the client's host registry
// cannot locate any FSPs.
func registerValidator(
	t *testing.T,
	ctx context.Context,
	cctx testnode.Context,
	fibreAddr string,
) {
	t.Helper()

	stakingClient := stakingtypes.NewQueryClient(cctx.GRPCClient)
	validators, err := stakingClient.Validators(ctx, &stakingtypes.QueryValidatorsRequest{})
	require.NoError(t, err)
	require.Len(t, validators.Validators, 1, "single-validator testnode expected")
	valOperator := validators.Validators[0].OperatorAddress

	txClient, err := testnode.NewTxClientFromContext(cctx)
	require.NoError(t, err)

	msg := &valtypes.MsgSetFibreProviderInfo{
		Signer: valOperator,
		Host:   fibreAddr,
	}
	resp, err := txClient.SubmitTx(ctx, []sdk.Msg{msg}, user.SetGasLimit(200_000), user.SetFee(5_000))
	require.NoError(t, err, "registering validator fibre host")
	require.Equal(t, uint32(0), resp.Code, "register validator tx failed")
	require.NoError(t, cctx.WaitForNextBlock())

	// Sanity-check the registration landed.
	tmClient := cmtservice.NewServiceClient(cctx.GRPCClient)
	valSet, err := tmClient.GetLatestValidatorSet(ctx, &cmtservice.GetLatestValidatorSetRequest{})
	require.NoError(t, err)
	require.Len(t, valSet.Validators, 1)
	consAddr, err := sdk.ConsAddressFromBech32(valSet.Validators[0].Address)
	require.NoError(t, err)

	valAddrClient := valtypes.NewQueryClient(cctx.GRPCClient)
	info, err := valAddrClient.FibreProviderInfo(ctx, &valtypes.QueryFibreProviderInfoRequest{
		ValidatorConsensusAddress: consAddr.String(),
	})
	require.NoError(t, err)
	require.True(t, info.Found, "fibre provider info not registered")
	require.Equal(t, fibreAddr, info.Info.Host)
}

// fundEscrow deposits enough utia into the client's escrow account to
// cover payment promises for several blob uploads. The async PFF
// settlement kicked off by adapter.Upload debits this account.
func fundEscrow(t *testing.T, ctx context.Context, cctx testnode.Context) {
	t.Helper()

	info, err := cctx.Keyring.Key(clientAccount)
	require.NoError(t, err, "loading client keyring entry")
	addr, err := info.GetAddress()
	require.NoError(t, err)

	ecfg := encoding.MakeConfig(app.ModuleEncodingRegisters...)
	txClient, err := user.SetupTxClient(
		ctx, cctx.Keyring, cctx.GRPCClient, ecfg,
		user.WithDefaultAccount(clientAccount),
	)
	require.NoError(t, err, "setting up funded-account tx client")

	amount := sdk.NewCoin(appconsts.BondDenom, sdkmath.NewInt(escrowDeposit))
	msg := &fibretypes.MsgDepositToEscrow{
		Signer: addr.String(),
		Amount: amount,
	}
	resp, err := txClient.SubmitTx(ctx, []sdk.Msg{msg}, user.SetGasLimit(200_000), user.SetFee(5_000))
	require.NoError(t, err, "depositing to escrow")
	require.Equal(t, uint32(0), resp.Code, "deposit tx failed")
	require.NoError(t, cctx.WaitForNextBlock())

	// Sanity: escrow is now visible.
	queryClient := fibretypes.NewQueryClient(cctx.GRPCClient)
	escrow, err := queryClient.EscrowAccount(ctx, &fibretypes.QueryEscrowAccountRequest{
		Signer: addr.String(),
	})
	require.NoError(t, err)
	require.True(t, escrow.Found, "escrow account not found after deposit")
	require.Equal(t, amount, escrow.EscrowAccount.Balance, "escrow balance mismatch")
}
