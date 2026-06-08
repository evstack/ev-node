---
description: This page provides an overview of some common ways to deploy chains.
---

# 🚀 Deploying Your Chain

One of the benefits of building chains with Evolve is the flexibility you have as a developer to choose things like the DA layer, the settlement scheme, and the execution environment.

You can learn more in the [Evolve architecture overview](../../learn/specs/overview.md).

The challenge that comes with this flexibility is that there are more services that now need to be deployed and managed while running your chain.

In the tutorials so far, you've seen various helper scripts used to make things easier. While great for tutorials, there are better ways to deploy and manage chains than using various bash scripts.

## 🏗️ Deployment Scales

Depending on your needs and the stage of your chain development, there are different deployment approaches you can take:

### 🏠 Local Development

For development and testing purposes, you can deploy your chain locally using containerized environments. This approach provides:

- Quick iteration and testing
- No external dependencies
- Full control over the environment
- Cost-effective development

### 🌐 Testnet Deployment

When you're ready to test with real network conditions, you can deploy to testnet environments. This includes:

- Integration with testnet DA networks
- Real network latency and conditions
- Multi-node testing scenarios
- Pre-production validation

## 📚 Available Deployment Guides

Choose the deployment approach that matches your current needs:

- [🏠 Local Development](./local.md) - Run everything on your machine with local-da (no external dependencies)
- [🌐 Testnet Deployment](./testnet.md) - Deploy on testnet with external DA networks

:::warning Disclaimer
These examples are for educational purposes only. Before deploying your chain for production use you should fully understand the services you are deploying and your choice in deployment method.
:::

## 🎉 Next Steps

For production mainnet deployments, consider additional requirements such as monitoring, security audits, infrastructure hardening, and operational procedures that go beyond the scope of these tutorials.
