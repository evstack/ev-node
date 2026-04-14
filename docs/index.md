---
layout: doc
title: Evolve Documentation
titleTemplate: ':title'
---

<script setup>
import constants from './.vitepress/constants/constants.js'
</script>

# Evolve Documentation

Evolve is the fastest way to launch your own modular network — without validator overhead or token lock-in. Built on Celestia, fully open-source, production-ready.

## Get started

<div class="grid-cards">

<a class="card" href="/guides/quick-start">
<span class="card-title">Quick Start</span>
<span class="card-desc">Launch a node in minutes with the Testapp CLI.</span>
</a>

<a class="card" href="/learn/about">
<span class="card-title">What is Evolve?</span>
<span class="card-desc">Architecture, execution model, and why it exists.</span>
</a>

<a class="card" href="/guides/gm-world">
<span class="card-title">Build a Chain</span>
<span class="card-desc">Step-by-step tutorial to build your first chain.</span>
</a>

<a class="card" href="/api">
<span class="card-title">API Reference</span>
<span class="card-desc">gRPC and JSON-RPC endpoint documentation.</span>
</a>

</div>

## Explore

| Section | What you'll find |
|---------|-----------------|
| [Learn](/learn/about) | Core concepts — DA, sequencing, execution, specs |
| [How-To Guides](/guides/quick-start) | Tutorials for building, deploying, and operating chains |
| [EVM Integration](/guides/evm/single) | Run an EVM chain with Reth |
| [DA Layers](/guides/da/local-da) | Connect to Celestia or run a local DA |
| [Deploy](/guides/deploy/overview) | Local, testnet, and mainnet deployment |
| [API Docs](/api) | Full RPC reference |

<style>
.grid-cards {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 16px;
  margin: 24px 0;
}

@media (max-width: 640px) {
  .grid-cards {
    grid-template-columns: 1fr;
  }
}

.card {
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 20px 24px;
  border-radius: 16px;
  border: 1px solid #DAE4E7;
  background: rgba(255, 255, 255, 0.6);
  text-decoration: none !important;
  transition: border-color 0.2s ease;
}

.card:hover {
  border-color: #B8A6FF;
}

.card-title {
  font-weight: 500;
  font-size: 16px;
  color: #000000;
  letter-spacing: -0.02em;
}

.card-desc {
  font-size: 14px;
  color: #3C3C3C;
  line-height: 1.5;
}
</style>
