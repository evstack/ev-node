<template>
    <div class="calculator">
        <section class="panel data-workload">
            <h2>Block production</h2>
            <div class="field-row">
                <label for="header-bytes">Header size (bytes)</label>
                <input
                    id="header-bytes"
                    type="number"
                    :value="HEADER_BYTES"
                    readonly
                />
            </div>
            <div class="field-row">
                <label for="block-time">Block time</label>
                <div class="field-group">
                    <input
                        id="block-time"
                        type="number"
                        min="0"
                        step="0.001"
                        v-model.number="blockTime"
                    />
                    <select v-model="blockTimeUnit">
                        <option value="s">seconds</option>
                        <option value="ms">milliseconds</option>
                    </select>
                </div>
            </div>
            <ul class="derived">
                <li>
                    <span>Blocks / second</span>
                    <strong>{{ formatNumber(blocksPerSecond, 4) }}</strong>
                </li>
            </ul>
        </section>

        <section class="panel batching-strategy">
            <h2>Batching strategy</h2>
            <p class="hint">
                Controls how blocks are batched before submission to the DA layer. Different strategies offer trade-offs between latency, cost efficiency, and throughput.
            </p>
            <div class="field-row">
                <label for="strategy">Strategy</label>
                <select id="strategy" v-model="batchingStrategy">
                    <option value="immediate">Immediate</option>
                    <option value="size">Size-based</option>
                    <option value="time">Time-based</option>
                    <option value="adaptive">Adaptive (Recommended)</option>
                </select>
            </div>
            <div class="strategy-description">
                <p v-if="batchingStrategy === 'immediate'">
                    <strong>Immediate:</strong> Submits as soon as any blocks are available. Best for low-latency requirements where cost is not a concern.
                </p>
                <p v-else-if="batchingStrategy === 'size'">
                    <strong>Size-based:</strong> Waits until the batch reaches a size threshold (fraction of max blob size). Best for maximizing blob utilization and minimizing costs when latency is flexible.
                </p>
                <p v-else-if="batchingStrategy === 'time'">
                    <strong>Time-based:</strong> Waits for a time interval before submitting. Provides predictable submission timing aligned with DA block times.
                </p>
                <p v-else-if="batchingStrategy === 'adaptive'">
                    <strong>Adaptive:</strong> Balances between size and time constraints—submits when either the size threshold is reached OR the max delay expires. Recommended for most production deployments.
                </p>
            </div>

            <div class="field-row">
                <label for="da-block-time">DA block time</label>
                <div class="field-group">
                    <input
                        id="da-block-time"
                        type="number"
                        min="1"
                        step="1"
                        v-model.number="daBlockTimeSeconds"
                    />
                    <span class="unit-label">seconds</span>
                </div>
            </div>

            <div v-if="batchingStrategy === 'size' || batchingStrategy === 'adaptive'" class="field-row">
                <label for="size-threshold">Batch size threshold (%)</label>
                <div class="field-group">
                    <input
                        id="size-threshold"
                        type="number"
                        min="10"
                        max="100"
                        step="5"
                        v-model.number="batchSizeThresholdPercent"
                    />
                    <span class="unit-label">% of 7 MB max blob</span>
                </div>
            </div>

            <div v-if="batchingStrategy === 'time' || batchingStrategy === 'adaptive'" class="field-row">
                <label for="max-delay">Batch max delay</label>
                <div class="field-group">
                    <input
                        id="max-delay"
                        type="number"
                        min="0"
                        step="1"
                        v-model.number="batchMaxDelaySeconds"
                    />
                    <span class="unit-label">seconds (0 = DA block time)</span>
                </div>
            </div>

            <div class="field-row">
                <label for="min-items">Batch minimum items</label>
                <input
                    id="min-items"
                    type="number"
                    min="1"
                    step="1"
                    v-model.number="batchMinItems"
                />
            </div>

            <p class="hint" style="margin-top: 1rem;">
                Header and data submission rates are shown in the Estimation section below, based on your data workload configuration.
            </p>
        </section>

        <section class="panel data-workload">
            <h2>Data workload</h2>
            <div class="env-toggle">
                <label>
                    <input type="radio" value="evm" v-model="executionEnv" />
                    EVM
                </label>
                <label>
                    <input type="radio" value="cosmos" v-model="executionEnv" />
                    Cosmos SDK
                </label>
            </div>

            <div v-if="executionEnv === 'cosmos'" class="coming-soon">
                Cosmos SDK transaction size presets are coming soon. Header
                costs still apply; data costs default to zero.
            </div>

            <div v-else class="evm-config">
                <div class="rate-grid">
                    <label for="tx-tps">
                        Transactions per second
                        <input
                            id="tx-tps"
                            type="number"
                            min="0"
                            step="0.01"
                            v-model.number="txPerSecondInput"
                        />
                    </label>
                    <label for="tx-month">
                        Transactions per month
                        <input
                            id="tx-month"
                            type="number"
                            min="0"
                            step="1"
                            v-model.number="txPerMonthInput"
                        />
                    </label>
                </div>

                <div class="mix-actions">
                    <button
                        type="button"
                        class="secondary"
                        @click="randomizeMix"
                    >
                        Randomize configuration
                    </button>
                    <button type="button" class="ghost" @click="resetMix">
                        Reset defaults
                    </button>
                </div>

                <div class="mix-grid">
                    <div class="mix-visual">
                        <template v-if="mixSegments.length">
                            <svg viewBox="0 0 36 36" class="mix-visual__chart">
                                <circle
                                    class="mix-visual__track"
                                    cx="18"
                                    cy="18"
                                    r="15.9155"
                                />
                                <circle
                                    v-for="segment in mixSegments"
                                    :key="segment.id"
                                    class="mix-visual__segment"
                                    cx="18"
                                    cy="18"
                                    r="15.9155"
                                    :stroke="segment.color"
                                    :stroke-dasharray="segment.dashArray"
                                    :stroke-dashoffset="segment.dashOffset"
                                />
                            </svg>
                            <div class="mix-visual__center">
                                <div class="mix-visual__value">
                                    {{ formatNumber(averageCalldataBytes, 1) }}
                                </div>
                                <div class="mix-visual__label">bytes avg</div>
                            </div>
                        </template>
                        <div v-else class="mix-chart__empty">
                            Enable at least one transaction type
                        </div>
                    </div>
                    <ul class="mix-legend">
                        <li v-for="slice in mixSlices" :key="slice.id">
                            <span
                                class="color-dot"
                                :style="{ background: slice.color }"
                            />
                            <div>
                                <strong>{{ slice.label }}</strong>
                                <span
                                    >{{ formatNumber(slice.percentage, 1) }}% •
                                    {{ slice.bytes }} bytes</span
                                >
                            </div>
                        </li>
                    </ul>
                </div>

                <details class="mix-table" open>
                    <summary>Customize transaction mix</summary>
                    <div class="mix-table__grid">
                        <label
                            v-for="entry in evmMix"
                            :key="entry.id"
                            class="mix-card"
                            :class="{ disabled: !entry.enabled }"
                        >
                            <div class="mix-card__header">
                                <input
                                    type="checkbox"
                                    v-model="entry.enabled"
                                />
                                <span>{{ entry.label }}</span>
                            </div>
                            <div class="mix-card__meta">
                                <span>{{ entry.bytes }} bytes</span>
                                <span
                                    class="color-dot"
                                    :style="{ background: entry.color }"
                                />
                            </div>
                            <div class="mix-card__control">
                                <span>Weight</span>
                                <input
                                    type="number"
                                    min="0"
                                    step="1"
                                    v-model.number="entry.weight"
                                    :disabled="!entry.enabled"
                                />
                            </div>
                        </label>
                    </div>
                </details>

                <div class="metrics">
                    <div>
                        <span>Average calldata bytes / tx</span>
                        <strong>{{
                            formatNumber(averageCalldataBytes, 2)
                        }}</strong>
                    </div>
                    <div>
                        <span>Transactions / submission</span>
                        <strong>{{
                            formatNumber(transactionsPerSubmission, 2)
                        }}</strong>
                    </div>
                    <div>
                        <span>Data blobs / submission</span>
                        <strong>{{ formatInteger(dataBlobCount) }}</strong>
                    </div>
                    <div>
                        <span>Average blob size (bytes)</span>
                        <strong>{{
                            formatNumber(averageDataBlobBytes, 2)
                        }}</strong>
                    </div>
                    <div>
                        <span>Data bytes / submission</span>
                        <strong>{{
                            formatNumber(dataBytesPerSubmission, 2)
                        }}</strong>
                    </div>
                    <div>
                        <span>Data shares / submission</span>
                        <strong>{{
                            formatInteger(dataSharesPerSubmission)
                        }}</strong>
                    </div>
                </div>
            </div>
        </section>

        <section class="panel">
            <h2>Gas parameters</h2>
            <p class="hint">
                Locked to Celestia mainnet defaults until live parameter
                fetching and manual overrides ship.
            </p>
            <ul class="param-list">
                <li>
                    <span>Fixed cost</span>
                    <strong
                        >{{ formatInteger(GAS_PARAMS.fixedCost) }} gas</strong
                    >
                </li>
                <li>
                    <span>Gas per blob byte</span>
                    <strong
                        >{{ formatInteger(GAS_PARAMS.gasPerBlobByte) }} gas /
                        byte</strong
                    >
                </li>
                <li>
                    <span>Share size</span>
                    <strong
                        >{{
                            formatInteger(GAS_PARAMS.shareSizeBytes)
                        }}
                        bytes</strong
                    >
                </li>
                <!-- <li>
                    <span>Per-blob static gas</span>
                    <strong
                        >{{
                            formatInteger(GAS_PARAMS.perBlobStaticGas)
                        }}
                        gas</strong
                    >
                </li> -->
            </ul>
            <div class="toggle-row">
                <label>
                    <input type="checkbox" v-model="firstTx" />
                    First transaction for this account (adds 10,000 gas once)
                </label>
            </div>
            <div class="field-row">
                <label for="gas-price">Gas price (uTIA / gas)</label>
                <input
                    id="gas-price"
                    type="number"
                    min="0"
                    step="0.001"
                    v-model.number="gasPriceValue"
                />
            </div>
        </section>

        <section class="panel results">
            <h2>Estimation</h2>
            <div class="summary">
                <div class="summary-item">
                    <span>Total gas / submission</span>
                    <strong>{{ formatInteger(totalGasPerSubmission) }}</strong>
                </div>
                <div class="summary-item">
                    <span>Fee / submission (TIA)</span>
                    <strong>{{
                        formatNumber(totalFeePerSubmissionTIA, 6)
                    }}</strong>
                </div>
                <div class="summary-item">
                    <span>Fee / second (TIA)</span>
                    <strong>{{ formatNumber(feePerSecondTIA, 6) }}</strong>
                </div>
                <div class="summary-item">
                    <span>Total yearly fee (TIA)</span>
                    <strong>{{
                        formatNumber(totalRecurringFeePerYearTIA, 4)
                    }}</strong>
                </div>
            </div>

            <details open>
                <summary>Header costs</summary>
                <ul class="breakdown">
                    <li>
                        <span>Header submission interval (s)</span>
                        <strong>{{ formatNumber(headerSubmissionIntervalSeconds, 2) }}</strong>
                    </li>
                    <li>
                        <span>Headers / submission</span>
                        <strong>{{ formatInteger(normalizedHeaderCount) }}</strong>
                    </li>
                    <li>
                        <span>Header bytes / submission</span>
                        <strong>{{ formatInteger(headerBytesTotal) }}</strong>
                    </li>
                    <li>
                        <span>Header submissions / year</span>
                        <strong>{{ formatNumber(headerSubmissionsPerYear, 0) }}</strong>
                    </li>
                    <li>
                        <span>Header gas / submission</span>
                        <strong>{{ formatInteger(headerGas) }}</strong>
                    </li>
                    <li>
                        <span>Header fee / submission (TIA)</span>
                        <strong>{{
                            formatNumber(headerFeePerSubmissionTIA, 6)
                        }}</strong>
                    </li>
                    <li>
                        <span>Header fee / year (TIA)</span>
                        <strong>{{
                            formatNumber(headerFeePerYearTIA, 4)
                        }}</strong>
                    </li>
                </ul>
            </details>

            <details :open="executionEnv === 'evm'">
                <summary>Data costs</summary>
                <p v-if="averageCalldataBytes === 0" class="hint">
                    Enable at least one transaction type to model calldata
                    usage.
                </p>
                <ul v-else class="breakdown">
                    <li>
                        <span>Data bytes / second</span>
                        <strong>{{
                            formatNumber(dataBytesPerSecond, 2)
                        }}</strong>
                    </li>
                    <li>
                        <span>Data submission interval (s)</span>
                        <strong>{{
                            formatNumber(dataSubmissionIntervalSeconds, 2)
                        }}</strong>
                    </li>
                    <li>
                        <span>Data submissions / year</span>
                        <strong>{{
                            formatNumber(dataSubmissionsPerYear, 0)
                        }}</strong>
                    </li>
                    <li>
                        <span>Average calldata bytes / tx</span>
                        <strong>{{
                            formatNumber(averageCalldataBytes, 2)
                        }}</strong>
                    </li>
                    <li>
                        <span>Transactions / data submission</span>
                        <strong>{{ formatNumber(transactionsPerSubmission, 0) }}</strong>
                    </li>
                    <li>
                        <span>Data bytes / submission</span>
                        <strong>{{
                            formatNumber(dataBytesPerSubmission, 0)
                        }}</strong>
                    </li>
                    <li>
                        <span>Data blobs / submission</span>
                        <strong>{{ formatInteger(dataBlobCount) }}</strong>
                    </li>
                    <li>
                        <span>Data gas / submission</span>
                        <strong>{{
                            formatInteger(dataGasPerSubmission)
                        }}</strong>
                    </li>
                    <li>
                        <span>Data fee / submission (TIA)</span>
                        <strong>{{
                            formatNumber(dataFeePerSubmissionTIA, 6)
                        }}</strong>
                    </li>
                    <li>
                        <span>Data fee / year (TIA)</span>
                        <strong>{{
                            formatNumber(dataFeePerYearTIA, 4)
                        }}</strong>
                    </li>
                </ul>
            </details>

            <details open>
                <summary>Baseline gas</summary>
                <ul class="breakdown">
                    <li>
                        <span>PFB transactions / submission</span>
                        <strong>{{
                            formatInteger(totalTransactionsPerSubmission)
                        }}</strong>
                    </li>
                    <li>
                        <span>Base gas / submission</span>
                        <strong>{{
                            formatInteger(fixedGasPerSubmission)
                        }}</strong>
                    </li>
                    <li>
                        <span>Base fee / submission (TIA)</span>
                        <strong>{{
                            formatNumber(fixedFeePerSubmissionTIA, 6)
                        }}</strong>
                    </li>
                    <li>
                        <span>Base fee / year (TIA)</span>
                        <strong>{{
                            formatNumber(fixedFeePerYearTIA, 4)
                        }}</strong>
                    </li>
                    <li v-if="firstTx">
                        <span>First transaction surcharge (TIA)</span>
                        <strong>{{ formatNumber(firstTxFeeTIA, 6) }}</strong>
                    </li>
                </ul>
            </details>

            <details>
                <summary>Throughput metrics</summary>
                <ul class="breakdown">
                    <li>
                        <span>Transactions per second</span>
                        <strong>{{ formatNumber(txPerSecond, 4) }}</strong>
                    </li>
                    <li>
                        <span>Transactions per month</span>
                        <strong>{{ formatNumber(txPerMonth, 0) }}</strong>
                    </li>
                    <li>
                        <span>Transactions per year</span>
                        <strong>{{ formatNumber(txPerYear, 0) }}</strong>
                    </li>
                    <li>
                        <span>Submissions per year</span>
                        <strong>{{
                            formatNumber(submissionsPerYear, 0)
                        }}</strong>
                    </li>
                </ul>
            </details>
        </section>
    </div>
</template>

<script setup lang="ts">
import { computed, reactive, ref } from "vue";

const HEADER_BYTES = 175;
const FIRST_TX_SURCHARGE = 10_000;
const SECONDS_PER_MONTH = 30 * 24 * 60 * 60;
const SECONDS_PER_YEAR = 365 * 24 * 60 * 60;
const DATA_CHUNK_BYTES = 500 * 1024; // 500 KiB chunk limit per blob
const MAX_BLOB_SIZE = 7 * 1024 * 1024; // 7 MB max blob size (from common/consts.go)

const GAS_PARAMS = Object.freeze({
    fixedCost: 65_000,
    gasPerBlobByte: 8,
    perBlobStaticGas: 0,
    shareSizeBytes: 482,
});

type BatchingStrategy = "immediate" | "size" | "time" | "adaptive";
type ExecutionEnv = "evm" | "cosmos";

type EvmTxType = {
    id: string;
    label: string;
    bytes: number;
    description?: string;
    defaultWeight: number;
};

type EvmMixEntry = EvmTxType & {
    enabled: boolean;
    weight: number;
    color: string;
};

const EVM_TX_TYPES: EvmTxType[] = [
    {
        id: "native-transfer",
        label: "Native value transfer",
        bytes: 0,
        defaultWeight: 2,
    },
    {
        id: "erc20-transfer",
        label: "ERC-20 transfer",
        bytes: 68,
        defaultWeight: 5,
    },
    {
        id: "erc20-approve",
        label: "ERC-20 approve",
        bytes: 68,
        defaultWeight: 3,
    },
    {
        id: "erc20-transferFrom",
        label: "ERC-20 transferFrom",
        bytes: 100,
        defaultWeight: 2,
    },
    {
        id: "erc721-transferFrom",
        label: "ERC-721 transferFrom",
        bytes: 100,
        defaultWeight: 1,
    },
    {
        id: "erc721-safeTransferFrom",
        label: "ERC-721 safeTransferFrom",
        bytes: 164,
        defaultWeight: 1,
    },
    {
        id: "erc721-mint",
        label: "ERC-721 mint",
        bytes: 68,
        defaultWeight: 1,
    },
    {
        id: "erc1155-transfer",
        label: "ERC-1155 safeTransferFrom",
        bytes: 196,
        defaultWeight: 1,
    },
    {
        id: "erc1155-batch",
        label: "ERC-1155 safeBatchTransferFrom",
        bytes: 228,
        defaultWeight: 1,
    },
    {
        id: "permit",
        label: "EIP-2612 permit",
        bytes: 228,
        defaultWeight: 1,
    },
];

const COLOR_PALETTE = [
    "#4263eb",
    "#f76707",
    "#0ca678",
    "#a61e4d",
    "#1098ad",
    "#5f3dc4",
    "#2d6a4f",
    "#ff922b",
    "#9c36b5",
    "#ffa94d",
];

const executionEnv = ref<ExecutionEnv>("evm");

const evmMix = reactive<EvmMixEntry[]>(
    EVM_TX_TYPES.map((type, index) => ({
        ...type,
        enabled: type.defaultWeight > 0,
        weight: type.defaultWeight,
        color: COLOR_PALETTE[index % COLOR_PALETTE.length],
    })),
);

// Batching strategy configuration
const batchingStrategy = ref<BatchingStrategy>("time");
const daBlockTimeSeconds = ref(6); // Celestia default block time
const batchSizeThreshold = ref(0.8); // 80% of max blob size (internal: 0.0-1.0)
const batchMaxDelaySeconds = ref(0); // 0 means use DA block time
const batchMinItems = ref(1);

// User-facing percentage (10-100) that syncs with internal threshold (0.1-1.0)
const batchSizeThresholdPercent = computed({
    get: () => Math.round(batchSizeThreshold.value * 100),
    set: (value: number) => {
        const clamped = Math.max(10, Math.min(100, value));
        batchSizeThreshold.value = clamped / 100;
    },
});

const blockTime = ref(0.25);
const blockTimeUnit = ref<"s" | "ms">("s");
const firstTx = ref(false);
const gasPriceValue = ref(0.004);

const txPerSecond = ref(10);
const txPerSecondInput = computed({
    get: () => txPerSecond.value,
    set: (value: number) => {
        txPerSecond.value = sanitizeNumber(value);
    },
});
const txPerMonthInput = computed({
    get: () => txPerSecond.value * SECONDS_PER_MONTH,
    set: (value: number) => {
        txPerSecond.value = sanitizeNumber(value) / SECONDS_PER_MONTH;
    },
});

const blockTimeSeconds = computed(() => {
    const value = blockTime.value;
    if (!isFinite(value) || value <= 0) {
        return NaN;
    }
    return blockTimeUnit.value === "ms" ? value / 1000 : value;
});

// Effective max delay: use DA block time if batchMaxDelaySeconds is 0
const effectiveMaxDelaySeconds = computed(() => {
    if (batchMaxDelaySeconds.value <= 0) {
        return daBlockTimeSeconds.value;
    }
    return batchMaxDelaySeconds.value;
});

// Target bytes for size-based batching
const targetBlobBytes = computed(() => MAX_BLOB_SIZE * batchSizeThreshold.value);

// ===== HEADER SUBMISSION INTERVAL =====
// Calculate header submission interval based on batching strategy
const headerSubmissionIntervalSeconds = computed(() => {
    const blockSeconds = blockTimeSeconds.value;
    if (!isFinite(blockSeconds) || blockSeconds <= 0) {
        return NaN;
    }

    const strategy = batchingStrategy.value;
    const minItems = Math.max(1, batchMinItems.value);
    const headerBytesPerBlock = HEADER_BYTES;

    if (strategy === "immediate") {
        return blockSeconds * minItems;
    }

    if (strategy === "time") {
        const delayBlocks = Math.ceil(effectiveMaxDelaySeconds.value / blockSeconds);
        return blockSeconds * Math.max(minItems, delayBlocks);
    }

    if (strategy === "size") {
        // How many blocks until headers fill the target blob size
        const blocksToThreshold = Math.ceil(targetBlobBytes.value / headerBytesPerBlock);
        return blockSeconds * Math.max(minItems, blocksToThreshold);
    }

    if (strategy === "adaptive") {
        const delayBlocks = Math.ceil(effectiveMaxDelaySeconds.value / blockSeconds);
        const blocksToThreshold = Math.ceil(targetBlobBytes.value / headerBytesPerBlock);
        return blockSeconds * Math.min(
            Math.max(minItems, delayBlocks),
            Math.max(minItems, blocksToThreshold)
        );
    }

    return blockSeconds * minItems;
});

// For backward compatibility, alias to submissionIntervalSeconds
const submissionIntervalSeconds = headerSubmissionIntervalSeconds;

// Calculate header count from submission interval
const normalizedHeaderCount = computed(() => {
    const blockSeconds = blockTimeSeconds.value;
    const interval = headerSubmissionIntervalSeconds.value;
    if (!isFinite(blockSeconds) || blockSeconds <= 0 || !isFinite(interval)) {
        return 1;
    }
    return Math.max(1, Math.round(interval / blockSeconds));
});

const headerBytesTotal = computed(
    () => normalizedHeaderCount.value * HEADER_BYTES,
);

const headerSubmissionsPerSecond = computed(() => {
    const interval = headerSubmissionIntervalSeconds.value;
    if (!isFinite(interval) || interval <= 0) {
        return 0;
    }
    return 1 / interval;
});

const submissionsPerSecond = headerSubmissionsPerSecond; // alias
const submissionsPerMinute = computed(() => headerSubmissionsPerSecond.value * 60);
const headerSubmissionsPerYear = computed(
    () => headerSubmissionsPerSecond.value * SECONDS_PER_YEAR,
);
const submissionsPerYear = headerSubmissionsPerYear; // alias

const blocksPerSecond = computed(() => {
    const seconds = blockTimeSeconds.value;
    if (!isFinite(seconds) || seconds <= 0) {
        return 0;
    }
    return 1 / seconds;
});

const headerBytesPerSecond = computed(() => {
    const interval = submissionIntervalSeconds.value;
    if (!isFinite(interval) || interval <= 0) {
        return 0;
    }
    return headerBytesTotal.value / interval;
});

const headerShares = computed(() => {
    const shareSize = Math.max(GAS_PARAMS.shareSizeBytes, 1);
    const totalBytes = headerBytesTotal.value;
    return Math.max(1, Math.ceil(totalBytes / shareSize));
});

const headerGas = computed(() => {
    const shareSize = Math.max(GAS_PARAMS.shareSizeBytes, 1);
    const gasPerByte = Math.max(GAS_PARAMS.gasPerBlobByte, 0);
    return Math.round(headerShares.value * shareSize * gasPerByte);
});

const enabledMix = computed(() =>
    evmMix.filter((entry) => entry.enabled && entry.weight > 0),
);

const totalMixWeight = computed(() =>
    enabledMix.value.reduce((sum, entry) => sum + entry.weight, 0),
);

const mixSlices = computed(() => {
    const total = totalMixWeight.value;
    if (total === 0) {
        return [];
    }
    return enabledMix.value.map((entry) => ({
        id: entry.id,
        label: entry.label,
        bytes: entry.bytes,
        percentage: (entry.weight / total) * 100,
        color: entry.color,
    }));
});

const mixSegments = computed(() => {
    const slices = mixSlices.value;
    if (!slices.length) {
        return [];
    }
    const total = slices.reduce((sum, slice) => sum + slice.percentage, 0);
    if (total === 0) {
        return [];
    }
    let cumulative = 0;
    return slices.map((slice) => {
        const value = (slice.percentage / total) * 100;
        const dashArray = `${value} ${100 - value}`;
        const dashOffset = 25 - cumulative;
        cumulative += value;
        return {
            ...slice,
            dashArray,
            dashOffset,
        };
    });
});

const averageCalldataBytes = computed(() => {
    if (executionEnv.value !== "evm") {
        return 0;
    }
    const total = totalMixWeight.value;
    if (total === 0) {
        return 0;
    }
    return (
        enabledMix.value.reduce(
            (sum, entry) => sum + (entry.weight / total) * entry.bytes,
            0,
        ) || 0
    );
});

const txPerMonth = computed(() => txPerSecond.value * SECONDS_PER_MONTH);
const txPerYear = computed(() => txPerSecond.value * SECONDS_PER_YEAR);

// Data bytes generated per second
const dataBytesPerSecond = computed(() => {
    if (executionEnv.value !== "evm") {
        return 0;
    }
    return txPerSecond.value * averageCalldataBytes.value;
});

// ===== DATA SUBMISSION INTERVAL =====
// Calculate data submission interval based on batching strategy
const dataSubmissionIntervalSeconds = computed(() => {
    const blockSeconds = blockTimeSeconds.value;
    const bytesPerSecond = dataBytesPerSecond.value;

    if (!isFinite(blockSeconds) || blockSeconds <= 0) {
        return NaN;
    }

    // If no data throughput, fall back to header interval
    if (bytesPerSecond <= 0) {
        return headerSubmissionIntervalSeconds.value;
    }

    const strategy = batchingStrategy.value;
    const minItems = Math.max(1, batchMinItems.value);
    const minInterval = blockSeconds * minItems;

    if (strategy === "immediate") {
        return minInterval;
    }

    if (strategy === "time") {
        return Math.max(minInterval, effectiveMaxDelaySeconds.value);
    }

    if (strategy === "size") {
        // Time to accumulate enough data to reach size threshold
        const timeToThreshold = targetBlobBytes.value / bytesPerSecond;
        return Math.max(minInterval, timeToThreshold);
    }

    if (strategy === "adaptive") {
        // Whichever comes first: size threshold or max delay
        const timeToThreshold = targetBlobBytes.value / bytesPerSecond;
        return Math.max(minInterval, Math.min(timeToThreshold, effectiveMaxDelaySeconds.value));
    }

    return minInterval;
});

const dataSubmissionsPerSecond = computed(() => {
    const interval = dataSubmissionIntervalSeconds.value;
    if (!isFinite(interval) || interval <= 0) {
        return 0;
    }
    return 1 / interval;
});

const dataSubmissionsPerYear = computed(
    () => dataSubmissionsPerSecond.value * SECONDS_PER_YEAR,
);

// Transactions included per data submission
const transactionsPerSubmission = computed(() => {
    const interval = dataSubmissionIntervalSeconds.value;
    if (!isFinite(interval) || interval <= 0) {
        return 0;
    }
    return txPerSecond.value * interval;
});

const dataBytesPerSubmission = computed(() => {
    if (executionEnv.value !== "evm") {
        return 0;
    }
    return averageCalldataBytes.value * transactionsPerSubmission.value;
});

const dataChunks = computed(() => {
    if (executionEnv.value !== "evm" || dataBytesPerSubmission.value <= 0) {
        return [] as Array<{ bytes: number; shares: number; gas: number }>;
    }
    const shareSize = Math.max(GAS_PARAMS.shareSizeBytes, 1);
    const gasPerByte = Math.max(GAS_PARAMS.gasPerBlobByte, 0);
    const chunks: Array<{ bytes: number; shares: number; gas: number }> = [];
    let remaining = dataBytesPerSubmission.value;
    while (remaining > 0) {
        const bytes = Math.min(DATA_CHUNK_BYTES, remaining);
        const shares = Math.max(1, Math.ceil(bytes / shareSize));
        const gas = shares * shareSize * gasPerByte;
        chunks.push({ bytes, shares, gas });
        remaining -= bytes;
    }
    return chunks;
});

const dataBlobCount = computed(() => dataChunks.value.length);

const dataSharesPerSubmission = computed(() =>
    dataChunks.value.reduce((sum, chunk) => sum + chunk.shares, 0),
);

const dataGasPerSubmission = computed(() =>
    dataChunks.value.reduce((sum, chunk) => sum + chunk.gas, 0),
);

const dataStaticGasPerSubmission = computed(
    () => dataBlobCount.value * Math.max(GAS_PARAMS.perBlobStaticGas, 0),
);

const averageDataBlobBytes = computed(() => {
    if (dataBlobCount.value === 0) {
        return 0;
    }
    return dataBytesPerSubmission.value / dataBlobCount.value;
});

const gasPriceUTIA = computed(() => Math.max(gasPriceValue.value, 0));
const gasPriceTIA = computed(() => gasPriceUTIA.value / 1_000_000);

// ===== HEADER COSTS =====
// 1 PFB transaction per header submission
const headerFixedGasPerSubmission = computed(() => GAS_PARAMS.fixedCost);

const headerFeePerSubmissionTIA = computed(
    () => (headerGas.value + headerFixedGasPerSubmission.value) * gasPriceTIA.value,
);

const headerFeePerYearTIA = computed(
    () => headerFeePerSubmissionTIA.value * headerSubmissionsPerYear.value,
);

// ===== DATA COSTS =====
// Each data submission may have multiple blobs (chunks), each is a separate PFB
const dataFixedGasPerSubmission = computed(
    () => Math.max(1, dataBlobCount.value) * GAS_PARAMS.fixedCost,
);

const dataRecurringGasPerSubmission = computed(
    () => dataGasPerSubmission.value + dataStaticGasPerSubmission.value + dataFixedGasPerSubmission.value,
);

const dataFeePerSubmissionTIA = computed(
    () => dataRecurringGasPerSubmission.value * gasPriceTIA.value,
);

const dataFeePerYearTIA = computed(
    () => dataFeePerSubmissionTIA.value * dataSubmissionsPerYear.value,
);

// ===== TOTALS =====
const firstTxGas = computed(() => (firstTx.value ? FIRST_TX_SURCHARGE : 0));
const firstTxFeeTIA = computed(() => firstTxGas.value * gasPriceTIA.value);

// Combined totals (for display, assumes one combined "submission" event)
const totalGasPerSubmission = computed(
    () => headerGas.value + headerFixedGasPerSubmission.value +
          dataRecurringGasPerSubmission.value + firstTxGas.value,
);

const totalFeePerSubmissionTIA = computed(
    () => totalGasPerSubmission.value * gasPriceTIA.value,
);

// Backward compat aliases
const fixedGasPerSubmission = computed(
    () => headerFixedGasPerSubmission.value + dataFixedGasPerSubmission.value,
);
const fixedFeePerSubmissionTIA = computed(
    () => fixedGasPerSubmission.value * gasPriceTIA.value,
);
const totalTransactionsPerSubmission = computed(
    () => 1 + dataBlobCount.value, // 1 header PFB + N data PFBs
);

const fixedFeePerYearTIA = computed(
    () => (headerFixedGasPerSubmission.value * headerSubmissionsPerYear.value +
           dataFixedGasPerSubmission.value * dataSubmissionsPerYear.value) * gasPriceTIA.value,
);

const totalRecurringFeePerYearTIA = computed(
    () => headerFeePerYearTIA.value + dataFeePerYearTIA.value,
);

const feePerSecondTIA = computed(() => {
    // Sum of header fee rate + data fee rate
    const headerFeePerSecond = headerFeePerSubmissionTIA.value * headerSubmissionsPerSecond.value;
    const dataFeePerSecond = dataFeePerSubmissionTIA.value * dataSubmissionsPerSecond.value;
    return headerFeePerSecond + dataFeePerSecond;
});

function randomizeMix() {
    let anyEnabled = false;
    evmMix.forEach((entry) => {
        entry.enabled = true;
        entry.weight = Math.max(1, Math.round(Math.random() * 100));
        anyEnabled = true;
    });
    if (!anyEnabled) {
        evmMix[0].enabled = true;
        evmMix[0].weight = 1;
    }
}

function resetMix() {
    evmMix.forEach((entry, index) => {
        entry.enabled = EVM_TX_TYPES[index].defaultWeight > 0;
        entry.weight = EVM_TX_TYPES[index].defaultWeight;
    });
}

function sanitizeNumber(value: number): number {
    if (!isFinite(value) || value < 0) {
        return 0;
    }
    return value;
}

function sanitizeInteger(value: number, min: number): number {
    if (!isFinite(value)) {
        return min;
    }
    return Math.max(min, Math.round(value));
}

function formatNumber(value: number, maximumFractionDigits = 2) {
    if (!isFinite(value)) {
        return "0";
    }
    return new Intl.NumberFormat("en-US", {
        minimumFractionDigits: 0,
        maximumFractionDigits,
    }).format(value);
}

function formatInteger(value: number) {
    if (!isFinite(value)) {
        return "0";
    }
    return new Intl.NumberFormat("en-US", {
        minimumFractionDigits: 0,
        maximumFractionDigits: 0,
    }).format(Math.round(value));
}
</script>

<style scoped>
/* ===== MAIN LAYOUT ===== */
.calculator {
    display: flex;
    flex-direction: column;
    gap: 2rem;
    padding: 0 0 3rem;
}

/* ===== PANELS ===== */
.panel {
    background: var(--vp-c-bg-soft);
    border-radius: 14px;
    padding: 2rem;
    box-shadow: 0 1px 2px rgba(15, 23, 42, 0.05);
}

.panel h2 {
    margin: 0 0 1rem;
    font-size: 1.125rem;
}

/* ===== FORM ELEMENTS ===== */
.field-row {
    display: flex;
    flex-direction: column;
    gap: 0.4rem;
    margin-bottom: 1.25rem;
}

.field-row label {
    font-weight: 600;
}

.field-group {
    display: flex;
    gap: 0.5rem;
}

.field-group input {
    flex: 1;
    min-width: 0;
}

.field-group select {
    min-width: 120px;
}

.field-group .unit-label {
    display: flex;
    align-items: center;
    font-size: 0.85rem;
    color: var(--vp-c-text-2);
    white-space: nowrap;
}

input,
select {
    border: 1px solid var(--vp-c-divider);
    border-radius: 8px;
    padding: 0.55rem 0.75rem;
    font-size: 0.95rem;
    background: var(--vp-c-bg);
    color: inherit;
}

input[readonly] {
    background: var(--vp-c-bg-soft);
    cursor: not-allowed;
}

/* ===== BUTTONS ===== */
button {
    border: none;
    border-radius: 8px;
    padding: 0.55rem 0.85rem;
    font-weight: 600;
    cursor: pointer;
    transition: all 0.15s ease;
}

button.secondary {
    background: var(--vp-c-brand-soft);
    color: var(--vp-c-brand-1);
}

button.secondary:hover {
    background: var(--vp-c-brand-1);
    color: var(--vp-c-bg);
}

button.ghost {
    background: transparent;
    border: 1px solid var(--vp-c-divider);
    color: inherit;
}

button.ghost:hover {
    border-color: var(--vp-c-brand-1);
    color: var(--vp-c-brand-1);
}

/* ===== LISTS & DATA DISPLAY ===== */
.derived,
.param-list,
.breakdown {
    list-style: none;
    margin: 0;
    padding: 0;
    display: grid;
    gap: 0.6rem;
}

.derived li,
.param-list li,
.breakdown li {
    display: flex;
    justify-content: space-between;
    gap: 1rem;
    font-size: 0.95rem;
}

.derived-header {
    font-weight: 600;
    font-size: 0.8rem;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    color: var(--vp-c-text-2);
    margin-top: 0.5rem;
    padding-top: 0.5rem;
    border-top: 1px solid var(--vp-c-divider);
}

.param-list {
    margin-bottom: 1.5rem;
}

.breakdown {
    margin-bottom: 1.5rem;
}

/* ===== TOGGLES ===== */
.env-toggle,
.toggle-row {
    margin-bottom: 1rem;
}

.env-toggle {
    display: flex;
    gap: 1rem;
    font-weight: 600;
}

.toggle-row label {
    display: inline-flex;
    align-items: center;
    gap: 0.5rem;
    font-weight: 500;
}

/* ===== TEXT STYLES ===== */
.hint {
    margin: 0 0 1.25rem;
    font-size: 0.85rem;
    color: var(--vp-c-text-2);
}

.strategy-description {
    margin: 0.5rem 0 1.25rem;
    padding: 0.75rem 1rem;
    background: var(--vp-c-bg);
    border-radius: 8px;
    border-left: 3px solid var(--vp-c-brand-1);
}

.strategy-description p {
    margin: 0;
    font-size: 0.9rem;
    line-height: 1.5;
}

strong {
    font-variant-numeric: tabular-nums;
}

/* ===== DATA WORKLOAD SECTION ===== */
.coming-soon {
    border: 1px dashed var(--vp-c-divider);
    border-radius: 10px;
    padding: 1rem;
    font-size: 0.95rem;
    background: rgba(148, 163, 184, 0.08);
}

.evm-config {
    display: flex;
    flex-direction: column;
    gap: 1.25rem;
}

/* Transaction rate inputs */
.rate-grid {
    display: grid;
    gap: 1rem;
    grid-template-columns: 1fr;
}

@media (min-width: 640px) {
    .rate-grid {
        grid-template-columns: 1fr 1fr;
    }
}

.rate-grid label {
    display: flex;
    flex-direction: column;
    gap: 0.35rem;
    font-weight: 600;
}

/* Action buttons */
.mix-actions {
    display: flex;
    gap: 0.75rem;
    flex-wrap: wrap;
}

/* Chart and legend container */
.mix-grid {
    display: grid;
    gap: 2rem;
}

@media (min-width: 768px) {
    .mix-grid {
        grid-template-columns: 280px 1fr;
    }
}

/* Donut chart */
.mix-visual {
    position: relative;
    width: 220px;
    height: 220px;
    margin: 0 auto;
}

.mix-visual__chart {
    width: 100%;
    height: 100%;
    transform: rotate(-90deg);
}

.mix-visual__track {
    fill: none;
    stroke: var(--vp-c-divider-soft);
    stroke-width: 3.2;
}

.mix-visual__segment {
    fill: none;
    stroke-width: 3.2;
    stroke-linecap: butt;
}

.mix-visual__center {
    position: absolute;
    top: 50%;
    left: 50%;
    transform: translate(-50%, -50%);
    text-align: center;
    background: var(--vp-c-bg);
    border-radius: 50%;
    width: 130px;
    height: 130px;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 0.15rem;
    box-shadow: inset 0 0 0 1px var(--vp-c-divider);
}

.mix-visual__value {
    font-weight: 700;
    font-size: 1.25rem;
    font-variant-numeric: tabular-nums;
}

.mix-visual__label {
    font-size: 0.75rem;
    text-transform: uppercase;
    letter-spacing: 0.08em;
    color: var(--vp-c-text-2);
}

.mix-chart__empty {
    text-align: center;
    font-size: 0.85rem;
    color: var(--vp-c-text-2);
    padding: 2rem 1rem;
}

/* Transaction type legend */
.mix-legend {
    list-style: none;
    margin: 0;
    padding: 0;
    display: flex;
    flex-direction: column;
    gap: 1rem;
}

.mix-legend li {
    display: flex;
    gap: 0.75rem;
    align-items: flex-start;
}

.color-dot {
    width: 12px;
    height: 12px;
    border-radius: 50%;
    flex-shrink: 0;
    margin-top: 0.25rem;
}

.mix-legend li > div {
    flex: 1;
    min-width: 0;
}

.mix-legend li strong {
    display: block;
    margin-bottom: 0.25rem;
}

.mix-legend li span {
    font-size: 0.9rem;
    color: var(--vp-c-text-2);
}

/* Transaction mix customization table */
.mix-table {
    border: 1px solid var(--vp-c-divider);
    border-radius: 12px;
    background: var(--vp-c-bg);
    padding: 1rem;
}

.mix-table summary {
    font-weight: 600;
    cursor: pointer;
    margin-bottom: 1rem;
}

.mix-table__grid {
    display: grid;
    gap: 0.75rem;
    grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
}

.mix-card {
    border: 1px solid var(--vp-c-divider);
    border-radius: 12px;
    padding: 1rem;
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
    background: var(--vp-c-bg-soft);
}

.mix-card.disabled {
    opacity: 0.5;
}

.mix-card__header {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    font-weight: 600;
}

.mix-card__meta {
    display: flex;
    justify-content: space-between;
    align-items: center;
    font-size: 0.85rem;
    color: var(--vp-c-text-2);
}

.mix-card__control {
    display: flex;
    justify-content: space-between;
    align-items: center;
    gap: 0.75rem;
}

.mix-card__control span {
    font-size: 0.9rem;
}

.mix-card__control input[type="number"] {
    width: 80px;
}

/* Summary metrics */
.metrics {
    display: flex;
    flex-direction: column;
    gap: 0.6rem;
    padding: 1rem;
    border-radius: 10px;
    background: rgba(76, 99, 235, 0.08);
}

.metrics div {
    display: flex;
    justify-content: space-between;
    gap: 1rem;
    font-size: 0.95rem;
}

/* ===== RESULTS SECTION ===== */
.results .summary {
    display: grid;
    gap: 1rem;
    margin-bottom: 1.5rem;
    grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
}

.summary-item {
    padding: 1rem;
    border-radius: 8px;
    background: var(--vp-c-bg);
    border: 1px solid var(--vp-c-divider);
    display: flex;
    flex-direction: column;
    gap: 0.3rem;
}

.summary-item span {
    font-size: 0.85rem;
    color: var(--vp-c-text-2);
}

.summary-item strong {
    font-size: 1.1rem;
}

/* Details/summary elements */
details {
    margin-bottom: 1.5rem;
}

details summary {
    cursor: pointer;
    font-weight: 600;
    margin-bottom: 0.75rem;
}
</style>
