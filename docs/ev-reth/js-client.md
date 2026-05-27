# JavaScript Client

`@evstack/evnode-viem` is a Viem-based client for EvNode transaction type `0x76`. It builds, signs, sponsors, serializes, and sends sponsored batch transactions to ev-reth.

Use it when your application needs:

- Sponsored gas, where an application account pays transaction fees.
- Batched calls, where several EVM operations execute atomically.
- Direct encoding and decoding of `0x76` raw transactions.

## Install

```bash
pnpm add @evstack/evnode-viem viem
```

## Create a Client

`createEvnodeClient` wraps a Viem client. The signer must implement `signHash(hash)`, and that function must sign the raw 32-byte digest without adding an EIP-191 prefix.

```typescript
import { createClient, http } from 'viem'
import { privateKeyToAccount, sign } from 'viem/accounts'
import { createEvnodeClient } from '@evstack/evnode-viem'

const rpcUrl = 'http://localhost:8545'
const privateKey = '0x...' as const

const client = createClient({
  transport: http(rpcUrl),
})

const account = privateKeyToAccount(privateKey)

const evnode = createEvnodeClient({
  client,
  executor: {
    address: account.address,
    signHash: async (hash) => sign({ hash, privateKey }),
  },
})
```

## Send a Batch

Each call has `to`, `value`, and `data`. Use `to: null` only for the first call when deploying a contract.

```typescript
const txHash = await evnode.send({
  calls: [
    { to: recipient1, value: 1_000_000_000_000_000n, data: '0x' },
    { to: recipient2, value: 1_000_000_000_000_000n, data: '0x' },
  ],
})
```

If either transfer fails, ev-reth reverts the whole transaction.

## Sponsor a Transaction

Use `createIntent` when the executor signs first and a sponsor signs later.

```typescript
import { createClient, http } from 'viem'
import { privateKeyToAccount, sign } from 'viem/accounts'
import { createEvnodeClient } from '@evstack/evnode-viem'

const client = createClient({ transport: http('http://localhost:8545') })

const executorKey = '0x...' as const
const sponsorKey = '0x...' as const
const executor = privateKeyToAccount(executorKey)
const sponsor = privateKeyToAccount(sponsorKey)

const evnode = createEvnodeClient({
  client,
  executor: {
    address: executor.address,
    signHash: async (hash) => sign({ hash, privateKey: executorKey }),
  },
  sponsor: {
    address: sponsor.address,
    signHash: async (hash) => sign({ hash, privateKey: sponsorKey }),
  },
})

const intent = await evnode.createIntent({
  calls: [
    { to: executor.address, value: 0n, data: '0x' },
  ],
})

const txHash = await evnode.sponsorAndSend({ intent })
```

The executor remains the transaction sender. The sponsor pays gas and receives gas refunds.

## Manual Parameters

If omitted, the client resolves `chainId`, `nonce`, `maxFeePerGas`, `maxPriorityFeePerGas`, `gasLimit`, and `accessList` from the RPC endpoint or local defaults.

Override them when your application already has fee estimates or nonce management:

```typescript
await evnode.send({
  calls: [
    { to: account.address, value: 0n, data: '0x' },
  ],
  nonce: 12n,
  gasLimit: 100_000n,
  maxFeePerGas: 1_000_000_000n,
  maxPriorityFeePerGas: 0n,
  accessList: [],
})
```

## Application Sponsor Service

Applications can hide sponsor keys behind a JSON-RPC proxy. The proxy receives `eth_sendRawTransaction`, detects unsigned type `0x76` transactions, validates the app policy, adds `feePayerSignature`, and forwards the sponsored raw transaction to ev-reth.

The ev-reth repository includes this pattern under `bin/sponsor-service`.

Expected proxy behavior:

- Forward ordinary JSON-RPC requests unchanged.
- Intercept `eth_sendRawTransaction` only when the raw transaction starts with type byte `0x76`.
- Forward already-sponsored transactions unchanged.
- Reject intents that fail policy checks, such as wrong chain ID, high gas limit, high max fee, or low sponsor balance.

Client-side code can then point Viem at the sponsor service instead of the node:

```typescript
const client = createClient({
  transport: http('http://localhost:3000'),
})

const evnode = createEvnodeClient({
  client,
  executor: {
    address: executor.address,
    signHash: async (hash) => sign({ hash, privateKey: executorKey }),
  },
})

const txHash = await evnode.send({
  calls: [
    { to, value, data },
  ],
})
```

In this setup the executor signs the intent, while the sponsor service adds the sponsor signature before forwarding to ev-reth.

## Utilities

The package also exports lower-level helpers:

| Helper | Use |
|--------|-----|
| `encodeSignedTransaction` | Serialize an EvNode signed transaction |
| `decodeEvNodeTransaction` | Decode a type `0x76` raw transaction |
| `computeExecutorSigningHash` | Compute the executor digest |
| `computeSponsorSigningHash` | Compute the sponsor digest |
| `computeTxHash` | Compute the transaction hash |
| `recoverExecutor` | Recover the executor address |
| `recoverSponsor` | Recover the sponsor address |
| `estimateIntrinsicGas` | Estimate the minimum intrinsic gas for calls |
| `validateEvNodeTx` | Validate local call-list constraints |

## Related Docs

- [Sponsored Batch Transactions](/ev-reth/features/sponsored-transactions)
- [ev-reth Overview](/ev-reth/overview)
- [EVM Quickstart](/getting-started/evm/quickstart)
