//go:build evm

package e2e

import (
	"context"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tastoradocker "github.com/celestiaorg/tastora/framework/docker"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/stretchr/testify/require"

	"github.com/evstack/ev-node/execution/evm"
)

// TestEvmSequencerWithAWSKMSSignerE2E validates an EVM sequencer using AWS KMS
// as the block signer. The test is skipped unless KMS env vars are provided.
// export EVNODE_E2E_AWS_KMS_KEY_ID=
// export EVNODE_E2E_AWS_KMS_REGION=
// go test -v -tags e2e,evm -run TestEvmSequencerWithAWSKMSSignerE2E -count=1 --evm-binary=$(pwd)/../../build/testapp
func TestEvmSequencerWithAWSKMSSignerE2E(t *testing.T) {
	if testing.Short() {
		t.Skip("skip e2e in short mode")
	}
	flag.Parse()

	kmsKeyID := os.Getenv("EVNODE_E2E_AWS_KMS_KEY_ID")
	if kmsKeyID == "" {
		t.Skip("set EVNODE_E2E_AWS_KMS_KEY_ID to run AWS KMS EVM e2e test")
	}

	kmsRegion := firstNonEmptyEVMKMS(
		os.Getenv("EVNODE_E2E_AWS_KMS_REGION"),
		os.Getenv("AWS_REGION"),
		os.Getenv("AWS_DEFAULT_REGION"),
	)
	kmsProfile := os.Getenv("EVNODE_E2E_AWS_KMS_PROFILE")

	workDir := t.TempDir()
	sequencerHome := filepath.Join(workDir, "evm-kms-agg")
	sut := NewSystemUnderTest(t)
	evmBinary := requireEVMBinary(t, sut)

	dockerClient, networkID := tastoradocker.Setup(t)
	evmEnv := SetupCommonEVMEnv(t, sut, dockerClient, networkID)

	jwtSecretFile := createJWTSecretFile(t, sequencerHome, evmEnv.SequencerJWT)

	initArgs := []string{
		"init",
		"--home", sequencerHome,
		"--evnode.node.aggregator=true",
		"--evnode.signer.signer_type=awskms",
		"--evnode.signer.kms_key_id=" + kmsKeyID,
		"--evnode.signer.kms_timeout=10s",
		"--evnode.signer.kms_max_retries=3",
	}
	if kmsRegion != "" {
		initArgs = append(initArgs, "--evnode.signer.kms_region="+kmsRegion)
	}
	if kmsProfile != "" {
		initArgs = append(initArgs, "--evnode.signer.kms_profile="+kmsProfile)
	}

	output, err := sut.RunCmd(evmBinary, initArgs...)
	require.NoError(t, err, "failed to init evm sequencer with awskms signer: %s", output)

	startArgs := []string{
		"start",
		"--evnode.log.format", "json",
		"--home", sequencerHome,
		"--evnode.node.aggregator=true",
		"--evnode.node.block_time", DefaultBlockTime,
		"--evnode.da.block_time", DefaultDABlockTime,
		"--evnode.da.address", evmEnv.Endpoints.GetDAAddress(),
		"--evnode.da.namespace", DefaultDANamespace,
		"--evnode.da.batching_strategy", "immediate",
		"--evnode.rpc.address", evmEnv.Endpoints.GetRollkitRPCListen(),
		"--evnode.p2p.listen_address", evmEnv.Endpoints.GetRollkitP2PAddress(),
		"--evnode.signer.signer_type=awskms",
		"--evnode.signer.kms_key_id=" + kmsKeyID,
		"--evnode.signer.kms_timeout=10s",
		"--evnode.signer.kms_max_retries=3",
		"--evm.jwt-secret-file", jwtSecretFile,
		"--evm.genesis-hash", evmEnv.GenesisHash,
		"--evm.engine-url", evmEnv.Endpoints.GetSequencerEngineURL(),
		"--evm.eth-url", evmEnv.Endpoints.GetSequencerEthURL(),
	}
	if kmsRegion != "" {
		startArgs = append(startArgs, "--evnode.signer.kms_region="+kmsRegion)
	}
	if kmsProfile != "" {
		startArgs = append(startArgs, "--evnode.signer.kms_profile="+kmsProfile)
	}

	sut.ExecCmd(evmBinary, startArgs...)
	sut.AwaitNodeUp(t, evmEnv.Endpoints.GetRollkitRPCAddress(), NodeStartupTimeout)

	client, err := ethclient.Dial(evmEnv.Endpoints.GetSequencerEthURL())
	require.NoError(t, err, "should connect to evm endpoint")
	defer client.Close()

	var nonce uint64
	tx := evm.GetRandomTransaction(t, TestPrivateKey, TestToAddress, DefaultChainID, DefaultGasLimit, &nonce)
	require.NoError(t, client.SendTransaction(context.Background(), tx), "failed to submit tx")

	require.Eventually(t, func() bool {
		return evm.CheckTxIncluded(client, tx.Hash())
	}, 20*time.Second, 500*time.Millisecond, "tx should be included in a block")

	t.Logf("KMS-backed EVM tx included: %s", tx.Hash().Hex())
}

func firstNonEmptyEVMKMS(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

func requireEVMBinary(t testing.TB, sut *SystemUnderTest) string {
	t.Helper()

	helpOutput, err := sut.RunCmd(evmSingleBinaryPath, "start", "--help")
	require.NoError(t, err, "failed to run start --help on -evm-binary=%q", evmSingleBinaryPath)

	if !strings.Contains(helpOutput, "--evm.jwt-secret-file") {
		t.Skipf(
			"-evm-binary=%q does not look like the EVM binary (missing --evm.jwt-secret-file). Pass the correct binary path via -evm-binary",
			evmSingleBinaryPath,
		)
	}

	return evmSingleBinaryPath
}
