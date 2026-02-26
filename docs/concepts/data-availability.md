# Data Availability

Data availability (DA) ensures that all transaction data required to verify the chain's state is accessible to anyone.

## Why DA Matters

Without data availability guarantees:

- Nodes can't verify state transitions
- Users can't prove their balances
- The chain's security model breaks down

Evolve uses external DA layers to provide these guarantees, rather than storing all data on L1.

## How Evolve Handles Data Availability

Evolve currently supports two DA modes:

### Local DA

- **Use case**: Development and testing
- **Guarantee**: None (operator can withhold data)
- **Latency**: Instant

### Celestia

- **Use case**: Production deployments
- **Guarantee**: Data availability sampling (DAS)
- **Latency**: ~12 seconds to finality

## DA Flow

```
Block Produced
      │
      ▼
┌─────────────────┐
│   Submitter     │  Queues block for DA
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│   DA Layer      │  Stores and orders data
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│   Full Nodes    │  Retrieve and verify
└─────────────────┘
```

## Namespaces

Evolve uses DA namespaces to organize data:

| Namespace | Purpose |
|-----------|---------|
| Header | Block headers |
| Data | Transaction data |
| Forced Inclusion | User-submitted transactions |

## Best Practices

- **Development**: Use Local DA for fast iteration
- **Testnet**: Use Celestia testnet (Mocha or Arabica)
- **Production**: Use Celestia mainnet or equivalent

## Learn More

- [Local DA Guide](/guides/da-layers/local-da)
- [Celestia Guide](/guides/da-layers/celestia)
- [DA Interface Reference](/reference/interfaces/da)
