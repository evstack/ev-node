import{_ as s,c as n,ag as e,o as i}from"./chunks/framework.CmpABV1Y.js";const m=JSON.parse('{"title":"Rollkit Minimal Header","description":"","frontmatter":{"head":[["meta",{"name":"og:title","content":"Rollkit Minimal Header | Evolve"},{"name":"og:description","content":false}]]},"headers":[],"relativePath":"adr/adr-015-rollkit-minimal-header.md","filePath":"adr/adr-015-rollkit-minimal-header.md","lastUpdated":1754470767000}'),t={name:"adr/adr-015-rollkit-minimal-header.md"};function l(o,a,p,r,c,d){return i(),n("div",null,a[0]||(a[0]=[e(`<h1 id="rollkit-minimal-header" tabindex="-1">Rollkit Minimal Header <a class="header-anchor" href="#rollkit-minimal-header" aria-label="Permalink to &quot;Rollkit Minimal Header&quot;">​</a></h1><h2 id="abstract" tabindex="-1">Abstract <a class="header-anchor" href="#abstract" aria-label="Permalink to &quot;Abstract&quot;">​</a></h2><p>This document specifies a minimal header format for Rollkit, designed to eliminate the dependency on CometBFT&#39;s header format. This new format can then be used to produce an execution layer tailored header if needed. For example, the new ABCI Execution layer can have an ABCI-specific header for IBC compatibility. This allows Rollkit to define its own header structure while maintaining backward compatibility where necessary.</p><h2 id="protocol-component-description" tabindex="-1">Protocol/Component Description <a class="header-anchor" href="#protocol-component-description" aria-label="Permalink to &quot;Protocol/Component Description&quot;">​</a></h2><p>The Rollkit minimal header is a streamlined version of the traditional header, focusing on essential information required for block processing and state management for nodes. This header format is designed to be lightweight and efficient, facilitating faster processing and reduced overhead.</p><h3 id="rollkit-minimal-header-structure" tabindex="-1">Rollkit Minimal Header Structure <a class="header-anchor" href="#rollkit-minimal-header-structure" aria-label="Permalink to &quot;Rollkit Minimal Header Structure&quot;">​</a></h3><div class="language-txt vp-adaptive-theme"><button title="Copy Code" class="copy"></button><span class="lang">txt</span><pre class="shiki shiki-themes github-light github-dark vp-code" tabindex="0"><code><span class="line"><span>┌─────────────────────────────────────────────┐</span></span>
<span class="line"><span>│             Rollkit Minimal Header          │</span></span>
<span class="line"><span>├─────────────────────┬───────────────────────┤</span></span>
<span class="line"><span>│ ParentHash          │ Hash of previous block│</span></span>
<span class="line"><span>├─────────────────────┼───────────────────────┤</span></span>
<span class="line"><span>│ Height              │ Block number          │</span></span>
<span class="line"><span>├─────────────────────┼───────────────────────┤</span></span>
<span class="line"><span>│ Timestamp           │ Creation time         │</span></span>
<span class="line"><span>├─────────────────────┼───────────────────────┤</span></span>
<span class="line"><span>│ ChainID             │ Chain identifier      │</span></span>
<span class="line"><span>├─────────────────────┼───────────────────────┤</span></span>
<span class="line"><span>│ DataCommitment      │ Pointer to block data │</span></span>
<span class="line"><span>│                     │ on DA layer           │</span></span>
<span class="line"><span>├─────────────────────┼───────────────────────┤</span></span>
<span class="line"><span>│ StateRoot           │ State commitment      │</span></span>
<span class="line"><span>├─────────────────────┼───────────────────────┤</span></span>
<span class="line"><span>│ ExtraData           │ Additional metadata   │</span></span>
<span class="line"><span>│                     │ (e.g. sequencer info) │</span></span>
<span class="line"><span>└─────────────────────┴───────────────────────┘</span></span></code></pre></div><h2 id="message-structure-communication-format" tabindex="-1">Message Structure/Communication Format <a class="header-anchor" href="#message-structure-communication-format" aria-label="Permalink to &quot;Message Structure/Communication Format&quot;">​</a></h2><p>The header is defined in GoLang as follows:</p><div class="language-go vp-adaptive-theme"><button title="Copy Code" class="copy"></button><span class="lang">go</span><pre class="shiki shiki-themes github-light github-dark vp-code" tabindex="0"><code><span class="line"><span style="--shiki-light:#6A737D;--shiki-dark:#6A737D;">// Header struct focusing on header information</span></span>
<span class="line"><span style="--shiki-light:#D73A49;--shiki-dark:#F97583;">type</span><span style="--shiki-light:#6F42C1;--shiki-dark:#B392F0;"> Header</span><span style="--shiki-light:#D73A49;--shiki-dark:#F97583;"> struct</span><span style="--shiki-light:#24292E;--shiki-dark:#E1E4E8;"> {</span></span>
<span class="line"><span style="--shiki-light:#6A737D;--shiki-dark:#6A737D;">    // Hash of the previous block header.</span></span>
<span class="line"><span style="--shiki-light:#24292E;--shiki-dark:#E1E4E8;">    ParentHash </span><span style="--shiki-light:#6F42C1;--shiki-dark:#B392F0;">Hash</span></span>
<span class="line"><span style="--shiki-light:#6A737D;--shiki-dark:#6A737D;">    // Height represents the block height (aka block number) of a given header</span></span>
<span class="line"><span style="--shiki-light:#24292E;--shiki-dark:#E1E4E8;">    Height </span><span style="--shiki-light:#D73A49;--shiki-dark:#F97583;">uint64</span></span>
<span class="line"><span style="--shiki-light:#6A737D;--shiki-dark:#6A737D;">    // Block creation timestamp</span></span>
<span class="line"><span style="--shiki-light:#24292E;--shiki-dark:#E1E4E8;">    Timestamp </span><span style="--shiki-light:#D73A49;--shiki-dark:#F97583;">uint64</span></span>
<span class="line"><span style="--shiki-light:#6A737D;--shiki-dark:#6A737D;">    // The Chain ID</span></span>
<span class="line"><span style="--shiki-light:#24292E;--shiki-dark:#E1E4E8;">    ChainID </span><span style="--shiki-light:#D73A49;--shiki-dark:#F97583;">string</span></span>
<span class="line"><span style="--shiki-light:#6A737D;--shiki-dark:#6A737D;">    // Pointer to location of associated block data aka transactions in the DA layer</span></span>
<span class="line"><span style="--shiki-light:#24292E;--shiki-dark:#E1E4E8;">    DataCommitment []</span><span style="--shiki-light:#D73A49;--shiki-dark:#F97583;">byte</span></span>
<span class="line"><span style="--shiki-light:#6A737D;--shiki-dark:#6A737D;">    // Commitment representing the state linked to the header</span></span>
<span class="line"><span style="--shiki-light:#24292E;--shiki-dark:#E1E4E8;">    StateRoot </span><span style="--shiki-light:#6F42C1;--shiki-dark:#B392F0;">Hash</span></span>
<span class="line"><span style="--shiki-light:#6A737D;--shiki-dark:#6A737D;">    // Arbitrary field for additional metadata</span></span>
<span class="line"><span style="--shiki-light:#24292E;--shiki-dark:#E1E4E8;">    ExtraData []</span><span style="--shiki-light:#D73A49;--shiki-dark:#F97583;">byte</span></span>
<span class="line"><span style="--shiki-light:#24292E;--shiki-dark:#E1E4E8;">}</span></span></code></pre></div><p>In case the chain has a specific designated proposer or a proposer set, that information can be put in the <code>extraData</code> field. So in single sequencer mode, the <code>sequencerAddress</code> can live in <code>extraData</code>. For base sequencer mode, this information is not relevant.</p><p>This minimal Rollkit header can be transformed to be tailored to a specific execution layer as well by inserting additional information typically needed.</p><h3 id="evm-execution-client" tabindex="-1">EVM execution client <a class="header-anchor" href="#evm-execution-client" aria-label="Permalink to &quot;EVM execution client&quot;">​</a></h3><ul><li><code>transactionsRoot</code>: Merkle root of all transactions in the block. Can be constructed from unpacking the <code>DataCommitment</code> in Rollkit Header.</li><li><code>receiptsRoot</code>: Merkle root of all transaction receipts, which store the results of transaction execution. This can be inserted by the EVM execution client.</li><li><code>Gas Limit</code>: Max gas allowed in the block.</li><li><code>Gas Used</code>: Total gas consumed in this block.</li></ul><h4 id="transformation-to-evm-header" tabindex="-1">Transformation to EVM Header <a class="header-anchor" href="#transformation-to-evm-header" aria-label="Permalink to &quot;Transformation to EVM Header&quot;">​</a></h4><div class="language-txt vp-adaptive-theme"><button title="Copy Code" class="copy"></button><span class="lang">txt</span><pre class="shiki shiki-themes github-light github-dark vp-code" tabindex="0"><code><span class="line"><span>┌─────────────────────────────────────────────┐</span></span>
<span class="line"><span>│             Rollkit Minimal Header          │</span></span>
<span class="line"><span>└───────────────────┬─────────────────────────┘</span></span>
<span class="line"><span>                    │</span></span>
<span class="line"><span>                    ▼ Transform</span></span>
<span class="line"><span>┌─────────────────────────────────────────────┐</span></span>
<span class="line"><span>│               EVM Header                    │</span></span>
<span class="line"><span>├─────────────────────┬───────────────────────┤</span></span>
<span class="line"><span>│ ParentHash          │ From Rollkit Header   │</span></span>
<span class="line"><span>├─────────────────────┼───────────────────────┤</span></span>
<span class="line"><span>│ Height/Number       │ From Rollkit Header   │</span></span>
<span class="line"><span>├─────────────────────┼───────────────────────┤</span></span>
<span class="line"><span>│ Timestamp           │ From Rollkit Header   │</span></span>
<span class="line"><span>├─────────────────────┼───────────────────────┤</span></span>
<span class="line"><span>│ ChainID             │ From Rollkit Header   │</span></span>
<span class="line"><span>├─────────────────────┼───────────────────────┤</span></span>
<span class="line"><span>│ TransactionsRoot    │ Derived from          │</span></span>
<span class="line"><span>│                     │ DataCommitment        │</span></span>
<span class="line"><span>├─────────────────────┼───────────────────────┤</span></span>
<span class="line"><span>│ StateRoot           │ From Rollkit Header   │</span></span>
<span class="line"><span>├─────────────────────┼───────────────────────┤</span></span>
<span class="line"><span>│ ReceiptsRoot        │ Added by EVM client   │</span></span>
<span class="line"><span>├─────────────────────┼───────────────────────┤</span></span>
<span class="line"><span>│ GasLimit            │ Added by EVM client   │</span></span>
<span class="line"><span>├─────────────────────┼───────────────────────┤</span></span>
<span class="line"><span>│ GasUsed             │ Added by EVM client   │</span></span>
<span class="line"><span>├─────────────────────┼───────────────────────┤</span></span>
<span class="line"><span>│ ExtraData           │ From Rollkit Header   │</span></span>
<span class="line"><span>└─────────────────────┴───────────────────────┘</span></span></code></pre></div><h3 id="abci-execution" tabindex="-1">ABCI Execution <a class="header-anchor" href="#abci-execution" aria-label="Permalink to &quot;ABCI Execution&quot;">​</a></h3><p>This header can be transformed into an ABCI-specific header for IBC compatibility.</p><ul><li><code>Version</code>: Required by IBC clients to correctly interpret the block&#39;s structure and contents.</li><li><code>LastCommitHash</code>: The hash of the previous block&#39;s commit, used by IBC clients to verify the legitimacy of the block&#39;s state transitions.</li><li><code>DataHash</code>: A hash of the block&#39;s transaction data, enabling IBC clients to verify that the data has not been tampered with. Can be constructed from unpacking the <code>DataCommitment</code> in Rollkit header.</li><li><code>ValidatorHash</code>: Current validator set&#39;s hash, which IBC clients use to verify that the block was validated by the correct set of validators. This can be the IBC attester set of the chain for backward compatibility with the IBC Tendermint client, if needed.</li><li><code>NextValidatorsHash</code>: The hash of the next validator set, allowing IBC clients to anticipate and verify upcoming validators.</li><li><code>ConsensusHash</code>: Denotes the hash of the consensus parameters, ensuring that IBC clients are aligned with the consensus rules of the blockchain.</li><li><code>AppHash</code>: Same as the <code>StateRoot</code> in the Rollkit Header.</li><li><code>EvidenceHash</code>: A hash of evidence of any misbehavior by validators, which IBC clients use to assess the trustworthiness of the validator set.</li><li><code>LastResultsHash</code>: Root hash of all results from the transactions from the previous block.</li><li><code>ProposerAddress</code>: The address of the block proposer, allowing IBC clients to track and verify the entities proposing new blocks. Can be constructed from the <code>extraData</code> field in the Rollkit Header.</li></ul><h4 id="transformation-to-abci-header" tabindex="-1">Transformation to ABCI Header <a class="header-anchor" href="#transformation-to-abci-header" aria-label="Permalink to &quot;Transformation to ABCI Header&quot;">​</a></h4><div class="language-txt vp-adaptive-theme"><button title="Copy Code" class="copy"></button><span class="lang">txt</span><pre class="shiki shiki-themes github-light github-dark vp-code" tabindex="0"><code><span class="line"><span>┌─────────────────────────────────────────────┐</span></span>
<span class="line"><span>│             Rollkit Minimal Header          │</span></span>
<span class="line"><span>└───────────────────┬─────────────────────────┘</span></span>
<span class="line"><span>                    │</span></span>
<span class="line"><span>                    ▼ Transform</span></span>
<span class="line"><span>┌─────────────────────────────────────────────┐</span></span>
<span class="line"><span>│               ABCI Header                   │</span></span>
<span class="line"><span>├─────────────────────┬───────────────────────┤</span></span>
<span class="line"><span>│ Height              │ From Rollkit Header   │</span></span>
<span class="line"><span>├─────────────────────┼───────────────────────┤</span></span>
<span class="line"><span>│ Time                │ From Rollkit Header   │</span></span>
<span class="line"><span>├─────────────────────┼───────────────────────┤</span></span>
<span class="line"><span>│ ChainID             │ From Rollkit Header   │</span></span>
<span class="line"><span>├─────────────────────┼───────────────────────┤</span></span>
<span class="line"><span>│ AppHash             │ From StateRoot        │</span></span>
<span class="line"><span>├─────────────────────┼───────────────────────┤</span></span>
<span class="line"><span>│ DataHash            │ From DataCommitment   │</span></span>
<span class="line"><span>├─────────────────────┼───────────────────────┤</span></span>
<span class="line"><span>│ Version             │ Added for IBC         │</span></span>
<span class="line"><span>├─────────────────────┼───────────────────────┤</span></span>
<span class="line"><span>│ LastCommitHash      │ Added for IBC         │</span></span>
<span class="line"><span>├─────────────────────┼───────────────────────┤</span></span>
<span class="line"><span>│ ValidatorHash       │ Added for IBC         │</span></span>
<span class="line"><span>├─────────────────────┼───────────────────────┤</span></span>
<span class="line"><span>│ NextValidatorsHash  │ Added for IBC         │</span></span>
<span class="line"><span>├─────────────────────┼───────────────────────┤</span></span>
<span class="line"><span>│ ConsensusHash       │ Added for IBC         │</span></span>
<span class="line"><span>├─────────────────────┼───────────────────────┤</span></span>
<span class="line"><span>│ EvidenceHash        │ Added for IBC         │</span></span>
<span class="line"><span>├─────────────────────┼───────────────────────┤</span></span>
<span class="line"><span>│ LastResultsHash     │ Added for IBC         │</span></span>
<span class="line"><span>├─────────────────────┼───────────────────────┤</span></span>
<span class="line"><span>│ ProposerAddress     │ From ExtraData        │</span></span>
<span class="line"><span>└─────────────────────┴───────────────────────┘</span></span></code></pre></div><h2 id="assumptions-and-considerations" tabindex="-1">Assumptions and Considerations <a class="header-anchor" href="#assumptions-and-considerations" aria-label="Permalink to &quot;Assumptions and Considerations&quot;">​</a></h2><ul><li>The Rollkit minimal header is designed to be flexible and adaptable, allowing for integration with various execution layers such as EVM and ABCI, without being constrained by CometBFT&#39;s header format.</li><li>The <code>extraData</code> field provides a mechanism for including additional metadata, such as sequencer information, which can be crucial for certain chain configurations.</li><li>The transformation of the Rollkit header into execution layer-specific headers should be done carefully to ensure compatibility and correctness, especially for IBC and any other cross-chain communication protocols.</li></ul><h3 id="header-transformation-flow" tabindex="-1">Header Transformation Flow <a class="header-anchor" href="#header-transformation-flow" aria-label="Permalink to &quot;Header Transformation Flow&quot;">​</a></h3><div class="language-txt vp-adaptive-theme"><button title="Copy Code" class="copy"></button><span class="lang">txt</span><pre class="shiki shiki-themes github-light github-dark vp-code" tabindex="0"><code><span class="line"><span>┌─────────────────────────────────────────────┐</span></span>
<span class="line"><span>│             Rollkit Minimal Header          │</span></span>
<span class="line"><span>│                                             │</span></span>
<span class="line"><span>│  A lightweight, flexible header format      │</span></span>
<span class="line"><span>│  with essential fields for block processing │</span></span>
<span class="line"><span>└───────────┬─────────────────┬───────────────┘</span></span>
<span class="line"><span>            │                 │</span></span>
<span class="line"><span>            ▼                 ▼</span></span>
<span class="line"><span>┌───────────────────┐ ┌─────────────────────┐</span></span>
<span class="line"><span>│  EVM Header       │ │  ABCI Header        │</span></span>
<span class="line"><span>│                   │ │                     │</span></span>
<span class="line"><span>│  For EVM-based    │ │  For IBC-compatible │</span></span>
<span class="line"><span>│  execution layers │ │  execution layers   │</span></span>
<span class="line"><span>└───────────────────┘ └─────────────────────┘</span></span></code></pre></div><h2 id="implementation" tabindex="-1">Implementation <a class="header-anchor" href="#implementation" aria-label="Permalink to &quot;Implementation&quot;">​</a></h2><p>Pending implementation.</p><h2 id="references" tabindex="-1">References <a class="header-anchor" href="#references" aria-label="Permalink to &quot;References&quot;">​</a></h2><ul><li><a href="https://ethereum.org/en/developers/docs/" target="_blank" rel="noreferrer">Ethereum Developer Documentation</a>: Comprehensive resources for understanding Ethereum&#39;s architecture, including block and transaction structures.</li><li><a href="https://docs.tendermint.com/master/spec/" target="_blank" rel="noreferrer">Tendermint Core Documentation</a>: Detailed documentation on Tendermint, which includes information on ABCI and its header format.</li><li><a href="https://github.com/tendermint/spec/blob/master/spec/abci/abci.md" target="_blank" rel="noreferrer">ABCI Specification</a>: The official specification for the Application Blockchain Interface (ABCI), which describes how applications can interact with the Tendermint consensus engine.</li><li><a href="https://github.com/cosmos/ibc" target="_blank" rel="noreferrer">IBC Protocol Specification</a>: Documentation on the Inter-Blockchain Communication (IBC) protocol, which includes details on how headers are used for cross-chain communication.</li></ul>`,29)]))}const u=s(t,[["render",l]]);export{m as __pageData,u as default};
