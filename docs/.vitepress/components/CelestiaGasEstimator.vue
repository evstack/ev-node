<template>
  <div class="calculator">
    <section class="panel">
      <h2>Network cadence</h2>
      <div class="field-row">
        <label for="header-bytes">Header size (bytes)</label>
        <input id="header-bytes" type="number" :value="HEADER_BYTES" readonly />
      </div>
      <div class="field-row">
        <label for="header-count">Headers per submission</label>
        <input
          id="header-count"
          type="number"
          min="0"
          step="1"
          v-model.number="headerCount"
          @blur="normalizeHeaderCount"
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
          <span>Headers per submission</span>
          <strong>{{ formatInteger(normalizedHeaderCount) }}</strong>
        </li>
        <li>
          <span>Header bytes / submission</span>
          <strong>{{ formatInteger(headerBytesTotal) }}</strong>
        </li>
        <li>
          <span>Submission interval (s)</span>
          <strong>{{ formatNumber(submissionIntervalSeconds, 3) }}</strong>
        </li>
        <li>
          <span>Submissions / second</span>
          <strong>{{ formatNumber(submissionsPerSecond, 4) }}</strong>
        </li>
        <li>
          <span>Submissions / minute</span>
          <strong>{{ formatNumber(submissionsPerMinute, 2) }}</strong>
        </li>
        <li>
          <span>Blocks / second</span>
          <strong>{{ formatNumber(blocksPerSecond, 4) }}</strong>
        </li>
        <li>
          <span>Header bytes / second</span>
          <strong>{{ formatNumber(headerBytesPerSecond, 2) }}</strong>
        </li>
      </ul>
    </section>

    <section class="panel">
      <h2>Gas parameters</h2>
      <p class="hint">
        Locked to Celestia mainnet defaults until live parameter fetching and
        manual overrides ship.
      </p>
      <ul class="param-list">
        <li>
          <span>Fixed cost</span>
          <strong>{{ formatInteger(GAS_PARAMS.fixedCost) }} gas</strong>
        </li>
        <li>
          <span>Gas per blob byte</span>
          <strong>{{ formatInteger(GAS_PARAMS.gasPerBlobByte) }} gas / byte</strong>
        </li>
        <li>
          <span>Share size</span>
          <strong>{{ formatInteger(GAS_PARAMS.shareSizeBytes) }} bytes</strong>
        </li>
        <li>
          <span>Per-blob static gas</span>
          <strong>{{ formatInteger(GAS_PARAMS.perBlobStaticGas) }} gas</strong>
        </li>
      </ul>
      <div class="toggle-row">
        <label>
          <input type="checkbox" v-model="firstTx" />
          First transaction for this account (adds 10,000 gas)
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

    <section class="panel">
      <h2>Blobs</h2>
      <p class="hint">
        Describe each blob batch with a payload size (bytes) and how many blobs
        of that size you plan to include. Empty or zero-count rows are ignored.
      </p>
      <div class="blob-row" v-for="(blob, index) in blobs" :key="blob.id">
        <div class="blob-label">Batch {{ index + 1 }}</div>
        <div class="blob-inputs">
          <label :for="`blob-size-${blob.id}`">
            Size (bytes)
            <input
              :id="`blob-size-${blob.id}`"
              type="number"
              min="0"
              step="1"
              v-model.number="blob.size"
            />
          </label>
          <label :for="`blob-count-${blob.id}`">
            Count
            <input
              :id="`blob-count-${blob.id}`"
              type="number"
              min="1"
              step="1"
              v-model.number="blob.count"
              @blur="normalizeCount(blob)"
            />
          </label>
        </div>
        <button
          type="button"
          class="ghost"
          @click="removeBlob(blob.id)"
          v-if="blobs.length > 1"
        >
          Remove
        </button>
      </div>
      <button type="button" class="secondary" @click="addBlob">
        Add blob
      </button>
    </section>

    <section class="panel results">
      <h2>Estimation</h2>
      <div class="summary">
        <div class="summary-item">
          <span>Total gas limit (per block)</span>
          <strong>{{ formatInteger(totalGasLimit) }}</strong>
        </div>
        <div class="summary-item">
          <span>Fee per block (TIA)</span>
          <strong>{{ formatNumber(totalFeeTIA, 6) }}</strong>
        </div>
        <div class="summary-item">
          <span>Fee per second (TIA)</span>
          <strong>{{ formatNumber(feePerSecondTIA, 6) }}</strong>
        </div>
        <div class="summary-item">
          <span>Projected daily fee (TIA)</span>
          <strong>{{ formatNumber(feePerDayTIA, 4) }}</strong>
        </div>
      </div>

      <details open>
        <summary>Contribution breakdown</summary>
        <ul class="breakdown">
          <li>
            <span>Fixed cost</span>
            <strong>{{ formatInteger(fixedCost) }}</strong>
          </li>
          <li v-if="headerGas > 0">
            <span>Header data cost ({{ formatInteger(normalizedHeaderCount) }}
              header<span v-if="normalizedHeaderCount !== 1">s</span>,
              {{ headerShares }}
              share<span v-if="headerShares !== 1">s</span>)</span
            >
            <strong>{{ formatInteger(headerGas) }}</strong>
          </li>
          <li>
            <span>Blob data cost</span>
            <strong>{{ formatInteger(blobGasTotal) }}</strong>
          </li>
          <li>
            <span>Per-blob static gas</span>
            <strong>{{ formatInteger(perBlobStaticGasTotal) }}</strong>
          </li>
          <li v-if="firstTx">
            <span>First transaction surcharge</span>
            <strong>{{ FIRST_TX_SURCHARGE }}</strong>
          </li>
        </ul>
      </details>

      <details :open="activeBlobBreakdown.length > 0">
        <summary>Per-blob breakdown</summary>
        <p v-if="activeBlobBreakdown.length === 0" class="hint">
          Add a blob size above to see individual share usage.
        </p>
        <table v-else>
          <thead>
            <tr>
              <th>#</th>
              <th>Size (bytes)</th>
              <th>Count</th>
              <th>Shares / blob</th>
              <th>Total shares</th>
              <th>Gas / blob</th>
              <th>Total gas</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="item in activeBlobBreakdown" :key="item.id">
              <td>{{ item.index + 1 }}</td>
              <td>{{ formatInteger(item.size) }}</td>
              <td>{{ formatInteger(item.count) }}</td>
              <td>{{ formatInteger(item.sharesPerBlob) }}</td>
              <td>{{ formatInteger(item.totalShares) }}</td>
              <td>{{ formatInteger(item.gasPerBlob) }}</td>
              <td>{{ formatInteger(item.totalGas) }}</td>
            </tr>
          </tbody>
        </table>
      </details>
    </section>
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, ref } from "vue";

const HEADER_BYTES = 175;
const FIRST_TX_SURCHARGE = 10_000;
const GAS_PARAMS = Object.freeze({
  fixedCost: 65_000,
  gasPerBlobByte: 8,
  perBlobStaticGas: 0,
  shareSizeBytes: 480,
});

type BlobInput = {
  id: number;
  size: number;
  count: number;
};

const headerCount = ref(1);
const blockTime = ref(2);
const blockTimeUnit = ref<"s" | "ms">("s");
const firstTx = ref(false);
const gasPriceValue = ref(0.004);
const blobs = reactive<BlobInput[]>([{ id: 1, size: 0, count: 1 }]);

let nextBlobId = 2;

const blockTimeSeconds = computed(() => {
  const value = blockTime.value;
  if (!isFinite(value) || value <= 0) {
    return NaN;
  }
  return blockTimeUnit.value === "ms" ? value / 1000 : value;
});

const normalizedHeaderCount = computed(() =>
  Math.max(0, Math.round(isFinite(headerCount.value) ? headerCount.value : 0)),
);

const headerBytesTotal = computed(
  () => normalizedHeaderCount.value * HEADER_BYTES,
);

const submissionIntervalSeconds = computed(() => {
  const blockSeconds = blockTimeSeconds.value;
  const count = normalizedHeaderCount.value;
  if (!isFinite(blockSeconds) || blockSeconds <= 0 || count <= 0) {
    return NaN;
  }
  return blockSeconds * count;
});

const submissionsPerSecond = computed(() => {
  const interval = submissionIntervalSeconds.value;
  if (!isFinite(interval) || interval <= 0) {
    return 0;
  }
  return 1 / interval;
});

const submissionsPerMinute = computed(
  () => submissionsPerSecond.value * 60,
);

const blocksPerSecond = computed(() => {
  const seconds = blockTimeSeconds.value;
  if (!seconds || !isFinite(seconds) || seconds <= 0) {
    return 0;
  }
  return 1 / seconds;
});

const headerBytesPerSecond = computed(() => {
  const totalBytes = headerBytesTotal.value;
  if (totalBytes === 0) {
    return 0;
  }
  const seconds = submissionIntervalSeconds.value;
  if (!seconds || !isFinite(seconds) || seconds <= 0) {
    return 0;
  }
  return totalBytes / seconds;
});

const headerShares = computed(() => {
  const shareSize = Math.max(GAS_PARAMS.shareSizeBytes, 1);
  const totalBytes = headerBytesTotal.value;
  if (totalBytes <= 0) {
    return 0;
  }
  return Math.max(1, Math.ceil(totalBytes / shareSize));
});

const headerGas = computed(() => {
  const shareSize = Math.max(GAS_PARAMS.shareSizeBytes, 1);
  const gasPerByte = Math.max(GAS_PARAMS.gasPerBlobByte, 0);
  if (headerShares.value === 0) {
    return 0;
  }
  return Math.round(headerShares.value * shareSize * gasPerByte);
});

const activeBlobs = computed(() =>
  blobs.filter(
    (blob) =>
      blob.size !== undefined &&
      blob.size > 0 &&
      blob.count !== undefined &&
      blob.count > 0,
  ),
);

const activeBlobBreakdown = computed(() => {
  const shareSize = Math.max(GAS_PARAMS.shareSizeBytes, 1);
  const gasPerByte = Math.max(GAS_PARAMS.gasPerBlobByte, 0);

  return activeBlobs.value.map((blob, index) => {
    const shares = Math.max(1, Math.ceil(blob.size / shareSize));
    const gas = Math.round(shares * shareSize * gasPerByte);
    const count = Math.max(1, Math.floor(blob.count));
    return {
      id: blob.id,
      index,
      size: blob.size,
      count,
      sharesPerBlob: shares,
      totalShares: shares * count,
      gasPerBlob: gas,
      totalGas: gas * count,
    };
  });
});

const blobGasTotal = computed(() =>
  activeBlobBreakdown.value.reduce((sum, blob) => sum + blob.totalGas, 0),
);

const perBlobStaticGasTotal = computed(() => {
  const perBlobStaticGasValue = Math.max(GAS_PARAMS.perBlobStaticGas, 0);
  return activeBlobBreakdown.value.reduce(
    (sum, blob) => sum + blob.count * perBlobStaticGasValue,
    0,
  );
});

const fixedCost = computed(() => Math.max(GAS_PARAMS.fixedCost, 0));

const firstTxSurcharge = computed(() =>
  firstTx.value ? FIRST_TX_SURCHARGE : 0,
);

const totalGasLimit = computed(() =>
  Math.round(
    fixedCost.value +
      headerGas.value +
      blobGasTotal.value +
      perBlobStaticGasTotal.value +
      firstTxSurcharge.value,
  ),
);

const gasPriceUTIA = computed(() => Math.max(gasPriceValue.value, 0));

const totalFeeUTIA = computed(() => totalGasLimit.value * gasPriceUTIA.value);
const totalFeeTIA = computed(() => totalFeeUTIA.value / 1_000_000);

const feePerSecondTIA = computed(
  () => {
    const interval = submissionIntervalSeconds.value;
    if (!isFinite(interval) || interval <= 0) {
      return 0;
    }
    return totalFeeTIA.value / interval;
  },
);

const feePerDayTIA = computed(() => feePerSecondTIA.value * 86_400);

function addBlob() {
  blobs.push({ id: nextBlobId++, size: 0, count: 1 });
}

function removeBlob(id: number) {
  if (blobs.length === 1) {
    blobs[0].size = 0;
    blobs[0].count = 1;
    return;
  }
  const index = blobs.findIndex((blob) => blob.id === id);
  if (index !== -1) {
    blobs.splice(index, 1);
  }
}

function normalizeHeaderCount() {
  if (!Number.isFinite(headerCount.value) || headerCount.value < 0) {
    headerCount.value = 0;
    return;
  }
  headerCount.value = Math.round(headerCount.value);
}

function normalizeCount(blob: BlobInput) {
  if (blob.count === undefined || blob.count < 1 || !Number.isFinite(blob.count)) {
    blob.count = 1;
    return;
  }
  blob.count = Math.round(blob.count);
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
.calculator {
  display: grid;
  gap: 1.75rem;
  max-width: 1100px;
  margin: 0 auto;
}

@media (min-width: 900px) {
  .calculator {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

.results {
  grid-column: 1 / -1;
}

.panel {
  background: var(--vp-c-bg-soft);
  border-radius: 12px;
  padding: 1.75rem;
  box-shadow: 0 1px 2px rgba(15, 23, 42, 0.05);
}

.panel h2 {
  margin-top: 0;
  margin-bottom: 1rem;
  font-size: 1.125rem;
}

.field-row {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
  margin-bottom: 1.25rem;
}

.field-row label {
  font-weight: 600;
}

.field-group {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
}

.field-group input {
  flex: 1;
}

.field-group select {
  min-width: 130px;
}

input,
select {
  border: 1px solid var(--vp-c-divider);
  border-radius: 8px;
  padding: 0.5rem 0.75rem;
  font-size: 0.95rem;
  background: var(--vp-c-bg);
  color: inherit;
}

input[readonly] {
  background: var(--vp-c-bg-soft);
  cursor: not-allowed;
}

.derived {
  list-style: none;
  padding: 0;
  margin: 0;
  display: grid;
  gap: 0.5rem;
}

.derived li {
  display: flex;
  align-items: center;
  justify-content: space-between;
  font-size: 0.95rem;
  gap: 1rem;
}

.toggle-row {
  margin: 1rem 0;
}

.toggle-row label {
  display: inline-flex;
  align-items: center;
  gap: 0.5rem;
  font-weight: 500;
}

.param-list {
  list-style: none;
  margin: 0 0 1.5rem;
  padding: 0;
  display: grid;
  gap: 0.75rem;
}

.param-list li {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 1rem;
  font-size: 0.95rem;
}

.param-list strong {
  font-variant-numeric: tabular-nums;
}

.hint {
  margin-top: 0;
  margin-bottom: 1.25rem;
  font-size: 0.85rem;
  color: var(--vp-c-text-2);
}

.blob-row {
  display: flex;
  flex-wrap: wrap;
  align-items: flex-start;
  gap: 1rem;
  margin-bottom: 1rem;
  padding: 1rem;
  border: 1px solid var(--vp-c-divider);
  border-radius: 10px;
  background: var(--vp-c-bg);
}

.blob-label {
  font-weight: 600;
  min-width: 90px;
}

.blob-inputs {
  flex: 1;
  display: grid;
  gap: 0.75rem;
  grid-template-columns: repeat(auto-fit, minmax(160px, 1fr));
}

.blob-inputs label {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
  font-weight: 600;
}

.blob-inputs input {
  width: 100%;
}

button {
  border: none;
  border-radius: 8px;
  padding: 0.5rem 0.75rem;
  font-weight: 600;
  cursor: pointer;
  transition: background 0.15s ease;
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

.results .summary {
  display: grid;
  gap: 1rem;
  margin-bottom: 1.5rem;
  grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
}

.summary-item {
  padding: 1rem;
  border-radius: 8px;
  background: var(--vp-c-bg);
  border: 1px solid var(--vp-c-divider);
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
  font-variant-numeric: tabular-nums;
}

.summary-item span {
  font-size: 0.85rem;
  color: var(--vp-c-text-2);
}

details summary {
  cursor: pointer;
  font-weight: 600;
  margin-bottom: 0.75rem;
}

.breakdown {
  list-style: none;
  padding: 0;
  margin: 0 0 1.5rem;
  display: grid;
  gap: 0.5rem;
}

.breakdown li {
  display: flex;
  align-items: center;
  justify-content: space-between;
  font-size: 0.95rem;
}

table {
  width: 100%;
  border-collapse: collapse;
  font-size: 0.9rem;
}

.results table {
  min-width: 420px;
}

.results details {
  overflow-x: auto;
}

th,
td {
  padding: 0.5rem;
  border-bottom: 1px solid var(--vp-c-divider);
  text-align: left;
}

thead {
  background: var(--vp-c-bg);
}
</style>
