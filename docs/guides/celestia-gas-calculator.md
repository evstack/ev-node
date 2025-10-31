---
title: Celestia Gas Calculator
description: Interactive estimator that mirrors Celestia MsgPayForBlobs gas logic with a focus on header sizing.
pageClass: gas-calculator
---

# Celestia Gas Calculator

Interactive calculator to estimate Celestia DA costs based on your rollup's block production rate and transaction throughput. All calculations mirror Celestia's `DefaultEstimateGas` logic, with fees reported in `TIA` based on your specified gas price in `uTIA / gas`.

> **Important**: These are estimates only. Actual costs may vary based on network conditions, gas price fluctuations, and blob size optimizations. Use these projections as a planning guide, not exact values.

## How it works

The calculator is organized into four sections:

### 1. Header cadence

Configure your rollup's block production rate and header batching strategy. Set how many headers you batch per submission—the tool automatically calculates the submission interval. For example, 15 headers at 250 ms block time means one submission every 3.75 seconds.

### 2. Data workload

Model your transaction throughput and calldata usage:

- **EVM mode**: Customize your transaction mix across common ERC-20, ERC-721, ERC-1155, and native transfers. The visual donut chart shows the weighted distribution of transaction types and calculates the average calldata bytes per transaction. Use "Randomize configuration" for quick testing or manually adjust weights in the customization panel.
- **Cosmos SDK mode**: Coming soon

The calculator translates your transaction rate and calldata into Celestia blob gas requirements, projecting costs per submission, per second, and annually.

For EVM workloads, data submissions are chunked into 500 KiB blobs (mirroring the batching logic in `da_submitter.go`). If a cadence produces more than 500 KiB of calldata in a window, the tool automatically simulates multiple blobs—and therefore multiple PayForBlobs transactions—so base gas and data gas scale accordingly.

### 3. Gas parameters

Review the Celestia mainnet gas parameters used for calculations:
- **Fixed cost**: 65,000 gas per submission
- **Gas per blob byte**: 8 gas per byte
- **Share size**: 480 bytes
- **Per-blob static gas**: 0 gas

Set your expected gas price and optionally account for the one-time 10,000 gas surcharge if this is the first transaction for the account.

> **Note**: Gas parameters are currently locked to Celestia mainnet defaults. Live parameter fetching and manual overrides will be added in a future update.

### 4. Estimation

View comprehensive cost breakdowns including:
- Total gas per submission and corresponding fees
- Detailed breakdown of header costs, data costs, and baseline gas
- Annual cost projections
- Throughput metrics (transactions per second, month, and year)

<CelestiaGasEstimator />
