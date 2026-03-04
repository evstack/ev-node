# Custom Precompiles

ev-reth supports custom EVM precompiled contracts for chain-specific functionality. This guide covers the built-in precompiles and how to add custom ones.

## What Are Precompiles?

Precompiles are special contracts at predefined addresses that execute native code instead of EVM bytecode. They're used for:

- Computationally expensive operations (cryptography, hashing)
- Chain-specific functionality (minting, governance)
- Operations impossible or inefficient in Solidity

## Built-in ev-reth Precompiles

### Mint Precompile

Allows an authorized address to mint native tokens. Useful for bridging scenarios.

**Address:** `0x0000000000000000000000000000000000000100`

**Configuration (chainspec):**

```json
{
  "config": {
    "evolve": {
      "mintPrecompile": {
        "admin": "0xBridgeContract",
        "address": "0x0000000000000000000000000000000000000100"
      }
    }
  }
}
```

**Interface:**

```solidity
interface IMint {
    /// @notice Mint native tokens to a recipient
    /// @param recipient Address to receive tokens
    /// @param amount Amount to mint (in wei)
    function mint(address recipient, uint256 amount) external;
}
```

**Usage:**

```solidity
// Only callable by admin address
IMint(0x0000000000000000000000000000000000000100).mint(
    0xRecipient,
    1 ether
);
```

See [Mint Precompile Reference](/ev-reth/features/mint-precompile) for details.

## Creating Custom Precompiles

Custom precompiles require modifying ev-reth source code.

### Step 1: Define the Precompile

Create a new precompile in `crates/precompiles/src/`:

```rust
// my_precompile.rs
use revm::precompile::{Precompile, PrecompileOutput, PrecompileResult};
use revm::primitives::{Bytes, U256};

pub const MY_PRECOMPILE_ADDRESS: Address = address!("0000000000000000000000000000000000000200");

pub fn my_precompile(input: &Bytes, gas_limit: u64) -> PrecompileResult {
    // Check gas
    let gas_used = 1000; // Base gas cost
    if gas_used > gas_limit {
        return Err(PrecompileError::OutOfGas);
    }

    // Parse input
    // input[0..4] = function selector
    // input[4..] = encoded arguments

    // Execute logic
    let result = process_input(input)?;

    Ok(PrecompileOutput {
        gas_used,
        bytes: result,
    })
}

fn process_input(input: &Bytes) -> Result<Bytes, PrecompileError> {
    // Your custom logic here
    Ok(Bytes::new())
}
```

### Step 2: Register the Precompile

Add the precompile to the precompile set:

```rust
// In precompiles/src/lib.rs
pub fn evolve_precompiles(chain_spec: &ChainSpec) -> PrecompileSet {
    let mut precompiles = standard_precompiles();

    // Add mint precompile if configured
    if let Some(mint_config) = &chain_spec.evolve.mint_precompile {
        precompiles.insert(mint_config.address, mint_precompile);
    }

    // Add your custom precompile
    if chain_spec.evolve.my_feature_enabled {
        precompiles.insert(MY_PRECOMPILE_ADDRESS, my_precompile);
    }

    precompiles
}
```

### Step 3: Add Chainspec Configuration

Define configuration structure:

```rust
// In chainspec types
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct MyPrecompileConfig {
    pub address: Address,
    pub admin: Option<Address>,
    pub some_parameter: u64,
}
```

Update chainspec parsing to include new config.

### Step 4: Build and Test

```bash
# Build ev-reth
cargo build --release

# Run tests
cargo test --package ev-reth-precompiles
```

## Precompile Best Practices

### Gas Metering

Charge gas proportional to computation:

```rust
fn my_precompile(input: &Bytes, gas_limit: u64) -> PrecompileResult {
    // Base cost
    let mut gas_used = 100;

    // Per-byte cost for input processing
    gas_used += input.len() as u64 * 3;

    // Additional cost for expensive operations
    if requires_crypto_operation(input) {
        gas_used += 10000;
    }

    if gas_used > gas_limit {
        return Err(PrecompileError::OutOfGas);
    }

    // Process...
}
```

### Access Control

For privileged operations, check caller:

```rust
fn admin_only_precompile(
    input: &Bytes,
    context: &PrecompileContext,
    config: &MyConfig,
) -> PrecompileResult {
    // Verify caller is admin
    if context.caller != config.admin {
        return Err(PrecompileError::Custom("unauthorized".into()));
    }

    // Process...
}
```

### Input Validation

Always validate input thoroughly:

```rust
fn my_precompile(input: &Bytes) -> PrecompileResult {
    // Check minimum length
    if input.len() < 36 {  // 4 byte selector + 32 byte arg
        return Err(PrecompileError::InvalidInput);
    }

    // Validate selector
    let selector = &input[0..4];
    if selector != MY_FUNCTION_SELECTOR {
        return Err(PrecompileError::InvalidInput);
    }

    // Parse and validate arguments
    let amount = U256::from_be_slice(&input[4..36]);
    if amount.is_zero() {
        return Err(PrecompileError::InvalidInput);
    }

    // Process...
}
```

### Determinism

Precompiles must be deterministic:

- No random number generation
- No external network calls
- No time-dependent logic
- Same input always produces same output

## Testing Precompiles

### Unit Tests

```rust
#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_my_precompile_success() {
        let input = encode_input(/* args */);
        let result = my_precompile(&input, 100000).unwrap();
        assert_eq!(result.bytes, expected_output());
    }

    #[test]
    fn test_my_precompile_out_of_gas() {
        let input = encode_input(/* args */);
        let result = my_precompile(&input, 10); // Too little gas
        assert!(matches!(result, Err(PrecompileError::OutOfGas)));
    }
}
```

### Integration Tests

Test precompile calls from Solidity:

```solidity
// test/MyPrecompile.t.sol
contract MyPrecompileTest is Test {
    address constant PRECOMPILE = 0x0000000000000000000000000000000000000200;

    function testPrecompileCall() public {
        (bool success, bytes memory result) = PRECOMPILE.call(
            abi.encodeWithSignature("myFunction(uint256)", 100)
        );
        assertTrue(success);
        // Assert result...
    }
}
```

## See Also

- [Mint Precompile](/ev-reth/features/mint-precompile) - Built-in minting
- [ev-reth Configuration](/ev-reth/configuration) - Chainspec setup
- [ev-reth Overview](/ev-reth/overview) - Architecture
