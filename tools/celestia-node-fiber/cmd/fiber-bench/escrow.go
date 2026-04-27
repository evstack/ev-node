package main

import (
	"context"
	"fmt"
	"time"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/celestiaorg/celestia-app/v9/app"
	"github.com/celestiaorg/celestia-app/v9/app/encoding"
	"github.com/celestiaorg/celestia-app/v9/pkg/appconsts"
	"github.com/celestiaorg/celestia-app/v9/pkg/user"
	fibretypes "github.com/celestiaorg/celestia-app/v9/x/fibre/types"
)

// escrowCmd groups Fibre-escrow operations. Uploads consume utia from
// the signer's escrow account; without a funded escrow, every Upload on
// the bench will fail at the chain.
func escrowCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "escrow",
		Short: "Manage Fibre escrow for the bench account",
	}
	cmd.AddCommand(escrowDepositCmd(), escrowQueryCmd())
	return cmd
}

func escrowDepositCmd() *cobra.Command {
	var (
		consensusGRPC string
		keyringDir    string
		keyName       string
		amountUtia    int64
		gasLimit      uint64
		feeUtia       uint64
	)
	cmd := &cobra.Command{
		Use:   "deposit",
		Short: "Deposit utia into the bench account's Fibre escrow",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(cmd.Context(), 60*time.Second)
			defer cancel()

			kr, err := openKeyring(keyringDir)
			if err != nil {
				return fmt.Errorf("open keyring: %w", err)
			}
			rec, err := kr.Key(keyName)
			if err != nil {
				return fmt.Errorf("key %q not found in %s: %w", keyName, keyringDir, err)
			}
			addr, err := rec.GetAddress()
			if err != nil {
				return fmt.Errorf("get address: %w", err)
			}

			conn, err := grpc.NewClient(consensusGRPC, grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				return fmt.Errorf("dial grpc: %w", err)
			}
			defer conn.Close()

			ecfg := encoding.MakeConfig(app.ModuleEncodingRegisters...)
			tc, err := user.SetupTxClient(ctx, kr, conn, ecfg, user.WithDefaultAccount(keyName))
			if err != nil {
				return fmt.Errorf("setup tx client: %w", err)
			}

			amount := sdk.NewCoin(appconsts.BondDenom, sdkmath.NewInt(amountUtia))
			msg := &fibretypes.MsgDepositToEscrow{
				Signer: addr.String(),
				Amount: amount,
			}
			fmt.Printf("submitting MsgDepositToEscrow: signer=%s amount=%s\n", addr.String(), amount.String())
			resp, err := tc.SubmitTx(ctx, []sdk.Msg{msg}, user.SetGasLimit(gasLimit), user.SetFee(feeUtia))
			if err != nil {
				return fmt.Errorf("submit tx: %w", err)
			}
			if resp.Code != 0 {
				return fmt.Errorf("deposit tx failed: code=%d codespace=%s", resp.Code, resp.Codespace)
			}
			fmt.Printf("deposit included: height=%d txhash=%s\n", resp.Height, resp.TxHash)

			// Sanity: read the escrow back so the operator sees the
			// new balance immediately.
			qc := fibretypes.NewQueryClient(conn)
			res, err := qc.EscrowAccount(ctx, &fibretypes.QueryEscrowAccountRequest{Signer: addr.String()})
			if err != nil {
				fmt.Printf("(could not query escrow back: %v)\n", err)
				return nil
			}
			if !res.Found {
				fmt.Println("(escrow not found after deposit — chain may need another block)")
				return nil
			}
			fmt.Printf("escrow balance: %s\n", res.EscrowAccount.Balance.String())
			return nil
		},
	}
	cmd.Flags().StringVar(&consensusGRPC, "consensus-grpc", "", "celestia-app gRPC address (host:port). Required.")
	cmd.Flags().StringVar(&keyringDir, "keyring-dir", defaultKeyringDir(), "directory holding the bench keyring")
	cmd.Flags().StringVar(&keyName, "key-name", "default", "key in the keyring to deposit from")
	cmd.Flags().Int64Var(&amountUtia, "amount", 50_000_000, "amount in utia to deposit (default 50 TIA)")
	cmd.Flags().Uint64Var(&gasLimit, "gas-limit", 200_000, "tx gas limit")
	cmd.Flags().Uint64Var(&feeUtia, "fee", 5_000, "fee in utia")
	_ = cobra.MarkFlagRequired(cmd.Flags(), "consensus-grpc")
	return cmd
}

func escrowQueryCmd() *cobra.Command {
	var (
		consensusGRPC string
		keyringDir    string
		keyName       string
	)
	cmd := &cobra.Command{
		Use:   "query",
		Short: "Print the current Fibre escrow balance for the bench account",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
			defer cancel()

			kr, err := openKeyring(keyringDir)
			if err != nil {
				return err
			}
			rec, err := kr.Key(keyName)
			if err != nil {
				return fmt.Errorf("key %q not found: %w", keyName, err)
			}
			addr, err := rec.GetAddress()
			if err != nil {
				return err
			}

			conn, err := grpc.NewClient(consensusGRPC, grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				return err
			}
			defer conn.Close()

			qc := fibretypes.NewQueryClient(conn)
			res, err := qc.EscrowAccount(ctx, &fibretypes.QueryEscrowAccountRequest{Signer: addr.String()})
			if err != nil {
				return err
			}
			if !res.Found {
				fmt.Printf("address: %s\nescrow:  not found (deposit first)\n", addr.String())
				return nil
			}
			fmt.Printf("address: %s\nescrow:  %s\n", addr.String(), res.EscrowAccount.Balance.String())
			return nil
		},
	}
	cmd.Flags().StringVar(&consensusGRPC, "consensus-grpc", "", "celestia-app gRPC address. Required.")
	cmd.Flags().StringVar(&keyringDir, "keyring-dir", defaultKeyringDir(), "keyring directory")
	cmd.Flags().StringVar(&keyName, "key-name", "default", "key name")
	_ = cobra.MarkFlagRequired(cmd.Flags(), "consensus-grpc")
	return cmd
}
