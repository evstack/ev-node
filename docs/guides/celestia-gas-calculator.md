---
title: Celestia Gas Calculator
description: Interactive estimator that mirrors Celestia MsgPayForBlobs gas logic with a focus on header sizing.
pageClass: gas-calculator
---

# Celestia Gas Calculator

Interactive calculator to estimate Celestia DA costs based on your rollup's block production rate and transaction throughput. All calculations mirror Celestia's `DefaultEstimateGas` logic, with fees reported in `TIA` based on your specified gas price in `uTIA / gas`.

> **Important**: These are estimates only. Actual costs may vary based on network conditions, gas price fluctuations, and blob size optimizations. Use these projections as a planning guide, not exact values.

## How it works

The calculator is organized into five sections:

### 1. Block production

Configure your rollup's block production rate. Set your block time (e.g., 250ms, 1s) to establish the base cadence of block production.

### 2. Batching strategy

Select how blocks are batched before submission to the DA layer. Four strategies are available:

- **Immediate**: Submits as soon as any blocks are available. Best for low-latency requirements where cost is not a concern.
- **Size-based**: Waits until the batch reaches a size threshold (fraction of max blob size). Best for maximizing blob utilization and minimizing costs when latency is flexible.
- **Time-based**: Waits for a time interval before submitting. Provides predictable submission timing aligned with DA block times.
- **Adaptive** (Recommended): Balances between size and time constraints—submits when either the size threshold is reached OR the max delay expires.

Configure strategy parameters:

- **DA block time**: The block time of the DA chain (default: 6s for Celestia)
- **Batch size threshold**: For size/adaptive strategies, the fraction of max blob size to fill before submitting (default: 80%)
- **Batch max delay**: For time/adaptive strategies, the maximum wait time before submitting (default: DA block time)
- **Batch minimum items**: Minimum number of blocks to accumulate before submission

### 3. Data workload

Model your transaction throughput and calldata usage:

- **EVM mode**: Customize your transaction mix across common ERC-20, ERC-721, ERC-1155, and native transfers. The visual donut chart shows the weighted distribution of transaction types and calculates the average calldata bytes per transaction. Use "Randomize configuration" for quick testing or manually adjust weights in the customization panel.
- **Cosmos SDK mode**: Coming soon

The calculator translates your transaction rate and calldata into Celestia blob gas requirements, projecting costs per submission, per second, and annually.

For EVM workloads, data submissions are chunked into 500 KiB blobs (mirroring the batching logic in `da_submitter.go`). If a cadence produces more than 500 KiB of calldata in a window, the tool automatically simulates multiple blobs—and therefore multiple PayForBlobs transactions—so base gas and data gas scale accordingly.

### 4. Gas parameters

Review the Celestia mainnet gas parameters used for calculations:

- **Fixed cost**: 65,000 gas per submission
- **Gas per blob byte**: 8 gas per byte
- **Share size**: 480 bytes
- **Per-blob static gas**: 0 gas

Set your expected gas price and optionally account for the one-time 10,000 gas surcharge if this is the first transaction for the account.

> **Note**: Gas parameters are currently locked to Celestia mainnet defaults. Live parameter fetching and manual overrides will be added in a future update.

### 5. Estimation

View comprehensive cost breakdowns including:

- Total gas per submission and corresponding fees
- Detailed breakdown of header costs, data costs, and baseline gas
- Annual cost projections
- Throughput metrics (transactions per second, month, and year)

<CelestiaGasEstimator />
