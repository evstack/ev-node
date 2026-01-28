# Fee Systems

Evolve chains have two layers of fees: execution fees (paid to process transactions) and DA fees (paid to post data).

## Execution Fees

### EVM (ev-reth)

Uses EIP-1559 fee model:

```
Transaction Fee = (Base Fee + Priority Fee) × Gas Used
```

| Component | Destination | Purpose |
|-----------|-------------|---------|
| Base Fee | Burned (or redirected) | Congestion pricing |
| Priority Fee | Sequencer | Incentive for inclusion |

#### Base Fee Redirect

By default, base fees are burned. ev-reth can redirect them to a treasury:

```json
{
  "config": {
    "evolve": {
      "baseFeeSink": "0xTREASURY",
      "baseFeeRedirectActivationHeight": 0
    }
  }
}
```

See [Base Fee Redirect](/ev-reth/features/base-fee-redirect) for details.

### Cosmos SDK (ev-abci)

Uses standard Cosmos SDK fee model:

```
Transaction Fee = Gas Price × Gas Used
```

Configure minimum gas prices:

```toml
# app.toml
minimum-gas-prices = "0.025stake"
```

Fees go to the fee collector module and can be distributed via standard Cosmos mechanisms.

## DA Fees

Both execution environments incur DA fees when blocks are posted to the DA layer.

### Cost Factors

| Factor | Impact |
|--------|--------|
| Block size | Linear cost increase |
| DA gas price | Market-driven, varies |
| Batching | Amortizes overhead |
| Compression | Reduces data size |

### Who Pays?

The sequencer pays DA fees from their own funds. They recover costs through:

- Priority fees from users
- Base fee redirect (if configured)
- External subsidy

### Optimization Strategies

#### Lazy Aggregation

Only produce blocks when there are transactions:

```yaml
node:
  lazy-aggregator: true
  lazy-block-time: 1s  # Max wait time
```

Reduces empty blocks and DA costs.

#### Batching

ev-node batches multiple blocks into single DA submissions:

```yaml
da:
  batch-size-threshold: 100000  # bytes
  batch-max-delay: 5s
```

#### Compression

Enable blob compression:

```yaml
da:
  compression: true
```

## Fee Flow Diagram

```
User Transaction
      │
      │ Pays: Gas Price × Gas
      ▼
┌─────────────────┐
│    Sequencer    │
│                 │
│ Receives:       │
│ - Priority fees │
│ - Base fees*    │
└────────┬────────┘
         │
         │ Pays: DA fees
         ▼
┌─────────────────┐
│    DA Layer     │
│   (Celestia)    │
└─────────────────┘

* If base fee redirect is enabled
```

## Estimating Costs

### Execution Costs

EVM:

```bash
cast estimate --rpc-url http://localhost:8545 <CONTRACT> "transfer(address,uint256)" <TO> <AMOUNT>
```

Cosmos:

```bash
appd tx bank send <FROM> <TO> 1000stake --gas auto --gas-adjustment 1.3
```

### DA Costs

Depends on:

- DA layer pricing (e.g., Celestia gas price)
- Data size per block
- Submission frequency

Use the [Celestia Gas Calculator](/guides/tools/celestia-gas-calculator) for estimates.
