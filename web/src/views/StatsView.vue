<script setup lang="ts">
import { watch } from 'vue'
import { useFeedStats } from '../composables/useFeedStats'
import ArticleTrendChart from '../components/business/ArticleTrendChart.vue'

const ranges = [7, 30, 90] as const

const { state, load } = useFeedStats()

watch(
  () => state.days,
  (d) => load(d),
)
load(state.days)

function formatLatest(s: string): string {
  return s || '—'
}
</script>

<template>
  <div class="stats">
    <header class="stats__header">
      <h1 class="stats__title">Statistics</h1>
      <div class="range" role="group" aria-label="Time range">
        <button
          v-for="r in ranges"
          :key="r"
          type="button"
          class="range__btn"
          :class="{ 'range__btn--active': state.days === r }"
          :aria-pressed="state.days === r"
          @click="state.days = r"
        >
          {{ r }}d
        </button>
      </div>
    </header>

    <div v-if="state.loading" class="state" aria-live="polite">
      <span aria-hidden="true" class="state__icon i-lucide-loader-circle" />
      Loading statistics…
    </div>
    <div v-else-if="state.error" class="state state--error" role="alert">
      <span aria-hidden="true" class="state__icon i-lucide-circle-alert" />
      {{ state.error }}
    </div>
    <div v-else-if="!state.data || state.data.feeds.length === 0" class="state">
      <span aria-hidden="true" class="state__icon i-lucide-inbox" />
      No feeds yet.
    </div>
    <template v-else>
      <ArticleTrendChart :feeds="state.data.feeds" :days="state.days" />
      <table class="table">
        <thead>
          <tr>
            <th class="table__feed">Feed</th>
            <th class="table__num">Articles</th>
            <th class="table__num">Latest article</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="f in state.data.feeds" :key="f.feed_id">
            <td class="table__feed">{{ f.title }}</td>
            <td class="table__num">{{ f.total }}</td>
            <td class="table__num">{{ formatLatest(f.latest_at) }}</td>
          </tr>
        </tbody>
      </table>
    </template>

    <router-link to="/" class="back">← All articles</router-link>
  </div>
</template>

<style scoped>
.stats {
  max-width: 860px;
  margin: 0 auto;
  padding: 24px;
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.stats__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}
.stats__title {
  margin: 0;
  font-size: 18px;
  font-weight: 700;
  color: var(--text);
}
.range {
  display: flex;
  gap: 6px;
}
.range__btn {
  padding: 5px 10px;
  border: 1px solid var(--border-strong);
  border-radius: 6px;
  background: var(--surface);
  color: var(--text-muted);
  font: inherit;
  font-size: 12px;
  line-height: 1;
  cursor: pointer;
}
.range__btn:hover {
  background: var(--bg-hover);
}
.range__btn--active {
  border-color: var(--accent);
  background: var(--accent-soft);
  color: var(--accent-soft-text);
}
.state {
  display: flex;
  align-items: center;
  gap: 7px;
  padding: 12px;
  color: var(--text-faint);
  font-size: 12px;
}
.state--error {
  color: var(--danger);
}
.state__icon {
  font-size: 14px;
}
.table {
  width: 100%;
  border-collapse: collapse;
  border: 1px solid var(--border);
  border-radius: 8px;
  overflow: hidden;
  background: var(--surface);
}
.table th,
.table td {
  padding: 8px 12px;
  border-bottom: 1px solid var(--border-subtle);
  text-align: left;
  font-size: 12.5px;
}
.table thead th {
  color: var(--text-muted);
  font-weight: 600;
  background: var(--bg-subtle);
}
.table tbody tr:last-child td {
  border-bottom: 0;
}
.table__feed {
  color: var(--text-secondary);
}
.table__num {
  text-align: right;
  color: var(--text-muted);
}
.back {
  align-self: flex-start;
  color: var(--accent);
  font-size: 12.5px;
  text-decoration: none;
}
.back:hover {
  text-decoration: underline;
}
</style>
