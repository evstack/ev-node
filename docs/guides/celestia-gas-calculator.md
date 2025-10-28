---
title: Celestia Gas Calculator
description: Interactive estimator that mirrors Celestia MsgPayForBlobs gas logic with a focus on header sizing.
pageClass: gas-calculator
---

# Celestia Gas Calculator

Fine-tune the inputs below to see how the 175-byte block header, block cadence, and blob payload batches contribute to the final `gasLimit` and transaction fee. All calculations mirror the structure of Celestia’s `DefaultEstimateGas`. Enter gas prices in `uTIA / gas` (the chain’s native denomination) and the estimator will report fees in `TIA`.

Set the header count to match how many headers you plan to batch per submission; the tool multiplies that count by the block time to derive the submission cadence (e.g., 15 headers at 250 ms ⇒ one submission every 3.75 s). Switch the execution environment to EVM to model data submissions: choose a transaction mix, randomize it if you want a quick sample, and the calculator will translate the calldata bytes into Celestia blob gas, projecting costs per submission, per second, per month, and per year. Cosmos SDK presets are coming soon.

For this first iteration the gas parameters are locked to the current Celestia mainnet defaults: fixed cost `65,000`, `8` gas per blob byte, effective share size `480` bytes, and zero static per-blob gas. Parameter overrides and live chain lookups will land in a follow-up.

<CelestiaGasEstimator />
