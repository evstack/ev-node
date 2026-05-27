# Sponsored Batch Transactions

ev-reth supports an EIP-2718 transaction type, `0x76`, for application-sponsored gas and atomic batches of EVM operations.

Use this transaction type when an application wants users to submit intents without holding the native gas token, or when a product flow needs several calls to succeed or fail together.

## What It Adds

| Capability | Description |
|------------|-------------|
| Gas sponsorship | A sponsor account can pay gas for a transaction signed by another account |
| Batch calls | One transaction can execute multiple `Call` operations |
| Atomic execution | If any call reverts, the whole transaction reverts |
| Standard submission | Signed transactions are submitted through `eth_sendRawTransaction` |
| RPC visibility | Sponsored transactions expose a `feePayer` field in transaction and receipt responses |

Regular Ethereum transactions still work. Use type `0x76` only for sponsored or batched flows.

## Transaction Model

An EvNode transaction replaces the standard single `to`, `value`, and `input` fields with a list of calls:

```typescript
type Call = {
  to: Address | null
  value: bigint
  data: Hex
}
```

The transaction also keeps EIP-1559-style fee fields:

```typescript
type EvNodeTransaction = {
  chainId: bigint
  nonce: bigint
  maxPriorityFeePerGas: bigint
  maxFeePerGas: bigint
  gasLimit: bigint
  calls: Call[]
  accessList: AccessList
  feePayerSignature?: Signature
}
```

The executor is the account that signs the transaction intent. The executor remains the transaction `from` address and is visible to contracts as `tx.origin`.

The sponsor is optional. When a sponsor signs the transaction, ev-reth charges gas to the sponsor instead of the executor. Value transfers still come from the executor.

## Batch Calls

Each call contains:

- `to`: target contract or account. Use `null` for contract creation.
- `value`: native token value sent with that call.
- `data`: calldata or contract creation bytecode.

Rules:

- `calls` must contain at least one item.
- Calls execute in order.
- Only the first call may be a contract creation (`to: null`).
- If any call fails, all state changes from earlier calls in the same transaction are reverted.

Example batch:

```typescript
await evnode.send({
  calls: [
    { to: token, value: 0n, data: approveData },
    { to: router, value: 0n, data: swapData },
  ],
})
```

This pattern is useful for application flows such as approve-then-call, multi-step account setup, onboarding actions, and batched admin operations.

## Sponsorship Flow

Sponsorship separates authorization from gas payment:

1. The executor builds the transaction with `feePayerSignature` empty.
2. The executor signs the transaction intent using the `0x76` signature domain.
3. The application validates the intent against its sponsorship policy.
4. The sponsor signs the same intent using the `0x78` sponsor domain.
5. The final signed transaction is submitted with `eth_sendRawTransaction`.

The sponsor signature binds to the executor address. A sponsor cannot reuse the same sponsor signature for a different executor.

For a user-facing application, the usual architecture is:

```text
User wallet / app client
  signs executor intent
        |
        v
Application sponsor service
  checks policy and sponsor balance
  adds feePayerSignature
        |
        v
ev-reth JSON-RPC
  eth_sendRawTransaction
```

The sponsor pays gas up to `gasLimit * maxFeePerGas`. Any gas refund goes back to the sponsor. The executor only needs enough balance for value transfers in the batch.

## JSON-RPC Behavior

Submit type `0x76` transactions with standard Ethereum JSON-RPC:

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "eth_sendRawTransaction",
  "params": ["0x76..."]
}
```

When a sponsored transaction is returned by `eth_getTransactionByHash`, `eth_getBlockByNumber`, or related transaction APIs, ev-reth includes the recovered sponsor address as `feePayer`.

Receipts also include `feePayer` when sponsorship was used.

## Validation

ev-reth validates type `0x76` transactions before admitting them to the txpool:

- The executor signature must be valid.
- The sponsor signature must be valid when present.
- The call list must be non-empty.
- Only the first call may create a contract.
- Sponsored transactions require the sponsor to have enough balance for gas.
- Unsponsored transactions require the executor to have enough balance for gas and value.

## Client Support

Most Ethereum wallets and CLIs do not know how to encode transaction type `0x76`. Use the JavaScript client for application code:

```bash
pnpm add @evstack/evnode-viem viem
```

See [JavaScript Client](/ev-reth/js-client) for signing, batching, sponsorship, and serialization examples.
