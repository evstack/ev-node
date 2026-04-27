package main

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/spf13/cobra"
)

// openKeyring opens (or creates if missing) a test-backend keyring at
// keyringDir. The "test" backend is unencrypted on disk — fine for a
// bench account, not fine for anything mainnet.
func openKeyring(keyringDir string) (keyring.Keyring, error) {
	interfaceRegistry := types.NewInterfaceRegistry()
	cryptocodec.RegisterInterfaces(interfaceRegistry)
	cdc := codec.NewProtoCodec(interfaceRegistry)
	return keyring.New(
		"fiber-bench",
		keyring.BackendTest,
		keyringDir,
		nil,
		cdc,
	)
}

func keysCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "keys",
		Short: "Manage the cosmos keyring used to sign Fibre payment promises",
	}
	cmd.AddCommand(keysAddCmd(), keysShowCmd(), keysListCmd())
	return cmd
}

func keysAddCmd() *cobra.Command {
	var keyringDir string
	cmd := &cobra.Command{
		Use:   "add <name>",
		Short: "Create a new key in the bench keyring (test backend, unencrypted)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			kr, err := openKeyring(keyringDir)
			if err != nil {
				return fmt.Errorf("open keyring: %w", err)
			}

			if rec, _ := kr.Key(name); rec != nil {
				return fmt.Errorf("key %q already exists in keyring %s", name, keyringDir)
			}

			rec, mnemonic, err := kr.NewMnemonic(
				name,
				keyring.English,
				sdk.FullFundraiserPath,
				keyring.DefaultBIP39Passphrase,
				hd.Secp256k1,
			)
			if err != nil {
				return fmt.Errorf("create key: %w", err)
			}

			addr, err := rec.GetAddress()
			if err != nil {
				return fmt.Errorf("get address: %w", err)
			}

			fmt.Printf("name:    %s\n", name)
			fmt.Printf("address: %s\n", addr.String())
			fmt.Printf("keyring: %s (backend=test)\n", keyringDir)
			fmt.Printf("\nmnemonic (back this up — printed once, never stored elsewhere):\n%s\n", mnemonic)
			fmt.Printf("\nNext steps:\n")
			fmt.Printf("  1. Top up the address above with utia on the chain.\n")
			fmt.Printf("  2. Deposit into the Fibre escrow with celestia-appd or your tooling, e.g.\n")
			fmt.Printf("     celestia-appd tx fibre deposit-escrow <utia-amount> --from %s --keyring-backend test --keyring-dir %s --chain-id <chain-id> --node tcp://<rpc>\n", name, keyringDir)
			return nil
		},
	}
	cmd.Flags().StringVar(&keyringDir, "keyring-dir", defaultKeyringDir(), "directory to store keyring files (test backend)")
	return cmd
}

func keysShowCmd() *cobra.Command {
	var keyringDir string
	cmd := &cobra.Command{
		Use:   "show <name>",
		Short: "Print the address of an existing key",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			kr, err := openKeyring(keyringDir)
			if err != nil {
				return fmt.Errorf("open keyring: %w", err)
			}
			rec, err := kr.Key(args[0])
			if err != nil {
				return fmt.Errorf("get key: %w", err)
			}
			addr, err := rec.GetAddress()
			if err != nil {
				return fmt.Errorf("get address: %w", err)
			}
			fmt.Printf("name:    %s\n", args[0])
			fmt.Printf("address: %s\n", addr.String())
			fmt.Printf("keyring: %s (backend=test)\n", keyringDir)
			return nil
		},
	}
	cmd.Flags().StringVar(&keyringDir, "keyring-dir", defaultKeyringDir(), "directory holding keyring files (test backend)")
	return cmd
}

func keysListCmd() *cobra.Command {
	var keyringDir string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all keys in the bench keyring",
		RunE: func(cmd *cobra.Command, args []string) error {
			kr, err := openKeyring(keyringDir)
			if err != nil {
				return fmt.Errorf("open keyring: %w", err)
			}
			records, err := kr.List()
			if err != nil {
				return fmt.Errorf("list keys: %w", err)
			}
			if len(records) == 0 {
				fmt.Printf("(empty — keyring at %s)\n", keyringDir)
				return nil
			}
			for _, rec := range records {
				addr, err := rec.GetAddress()
				if err != nil {
					return err
				}
				fmt.Printf("%-20s %s\n", rec.Name, addr.String())
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&keyringDir, "keyring-dir", defaultKeyringDir(), "directory holding keyring files (test backend)")
	return cmd
}

// silenceUnusedClient keeps the SDK client package referenced even if a
// future refactor stops using it directly — convenient when wiring a
// proper send/escrow command.
var _ = client.Context{}
