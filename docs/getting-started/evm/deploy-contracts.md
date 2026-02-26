# Deploy Contracts

Deploy smart contracts to your Evolve EVM chain using Foundry or Hardhat.

## Network Configuration

| Setting | Local | Testnet (example) |
|---------|-------|-------------------|
| RPC URL | <http://localhost:8545> | <https://rpc.your-chain.com> |
| Chain ID | 1337 | Your chain ID |
| Currency | ETH | Your native token |

## Foundry

### Install

```bash
curl -L https://foundry.paradigm.xyz | bash
foundryup
```

### Configure

Create or update `foundry.toml`:

```toml
[profile.default]
src = "src"
out = "out"
libs = ["lib"]

[rpc_endpoints]
local = "http://localhost:8545"
```

### Deploy

```bash
# Deploy a contract
forge create src/MyContract.sol:MyContract \
  --rpc-url local \
  --private-key $PRIVATE_KEY

# Deploy with constructor args
forge create src/Token.sol:Token \
  --rpc-url local \
  --private-key $PRIVATE_KEY \
  --constructor-args "MyToken" "MTK" 18

# Deploy and verify (if explorer supports it)
forge create src/MyContract.sol:MyContract \
  --rpc-url local \
  --private-key $PRIVATE_KEY \
  --verify
```

### Interact

```bash
# Call a read function
cast call $CONTRACT_ADDRESS "balanceOf(address)" $WALLET_ADDRESS --rpc-url local

# Send a transaction
cast send $CONTRACT_ADDRESS "transfer(address,uint256)" $TO_ADDRESS 1000 \
  --rpc-url local \
  --private-key $PRIVATE_KEY
```

## Hardhat

### Install

```bash
npm init -y
npm install --save-dev hardhat @nomicfoundation/hardhat-toolbox
npx hardhat init
```

### Configure

Update `hardhat.config.js`:

```javascript
require("@nomicfoundation/hardhat-toolbox");

module.exports = {
  solidity: "0.8.24",
  networks: {
    local: {
      url: "http://localhost:8545",
      accounts: [process.env.PRIVATE_KEY],
    },
  },
};
```

### Deploy

Create `scripts/deploy.js`:

```javascript
const hre = require("hardhat");

async function main() {
  const Contract = await hre.ethers.getContractFactory("MyContract");
  const contract = await Contract.deploy();
  await contract.waitForDeployment();

  console.log("Deployed to:", await contract.getAddress());
}

main().catch((error) => {
  console.error(error);
  process.exit(1);
});
```

Run:

```bash
npx hardhat run scripts/deploy.js --network local
```

## Prefunded Accounts

The default chainspec includes prefunded accounts for testing. Check your `genesis.json` `alloc` section for available addresses.

To add your own:

```json
{
  "alloc": {
    "0xYourAddress": {
      "balance": "0x200000000000000000000000000000000000000000000000000000000000000"
    }
  }
}
```

## Next Steps

- [Configure ev-reth](/getting-started/evm/setup-ev-reth) — Chainspec customization
- [Base Fee Redirect](/ev-reth/features/base-fee-redirect) — Send fees to treasury
- [Deploy Allowlist](/ev-reth/features/deploy-allowlist) — Restrict contract deployment
