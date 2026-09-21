<script setup lang="ts">
import { computed, reactive, watch } from 'vue'
import { useFeedStats } from '../composables/useFeedStats'
import ArticleTrendChart from '../components/business/ArticleTrendChart.vue'

const ranges = [7, 30, 90] as const

const { state, load } = useFeedStats()

watch(
  () => state.days,
  (d) => load(d),
)
load(state.days)

// Sortable summary table. "Latest article" uses the raw timestamp string; feeds
// that never published sort last regardless of direction.
type SortKey = 'feed' | 'total' | 'latest'
const sort = reactive<{ key: SortKey; dir: 1 | -1 }>({ key: 'feed', dir: 1 })

const sortedFeeds = computed(() => {
  const feeds = state.data?.feeds ?? []
  const { key, dir } = sort
  return [...feeds].sort((a, b) => {
    let cmp = 0
    if (key === 'total') {
      cmp = a.total - b.total
    } else if (key === 'latest') {
      if (a.latest_at === b.latest_at) cmp = 0
      else if (!a.latest_at) cmp = 1
      else if (!b.latest_at) cmp = -1
      else cmp = a.latest_at < b.latest_at ? -1 : 1
    } else {
      cmp = a.title.localeCompare(b.title)
    }
    return cmp * dir
  })
})

function setSort(key: SortKey) {
  if (sort.key === key) {
    sort.dir = sort.dir === 1 ? -1 : 1
  } else {
    sort.key = key
    sort.dir = key === 'feed' ? 1 : -1
  }
}

function formatLatest(s: string): string {
  return s || '—'
}

</script>

<template>
  <div class="stats">
    <nav class="stats__nav" aria-label="Page navigation">
      <router-link to="/" class="stats__back">
        <span aria-hidden="true" class="i-lucide-chevron-left text-[14px]" />
        All articles
      </router-link>
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
    </nav>

    <!-- Content stays mounted across range changes; a translucent mask covers it
         while refetching instead of unmounting/replacing the rendered content. -->
    <div v-if="state.data && state.data.feeds.length > 0" class="stats__body">
      <div class="stats__chart">
        <ArticleTrendChart :key="state.days" :feeds="state.data.feeds" :days="state.days" />
        <div v-if="state.loading" class="mask" role="status" aria-live="polite">
          <span aria-hidden="true" class="mask__spinner i-lucide-loader-circle" />
          Loading statistics…
        </div>
      </div>
      <table class="table">
        <thead>
          <tr>
            <th
              class="table__feed"
              :aria-sort="sort.key === 'feed' ? (sort.dir === 1 ? 'ascending' : 'descending') : 'none'"
            >
              <button type="button" class="th-btn" :class="{ 'th-btn--active': sort.key === 'feed' }" @click="setSort('feed')">
                Feed
                <span
                  v-if="sort.key === 'feed'"
                  aria-hidden="true"
                  class="th-btn__arrow"
                  :class="sort.dir === 1 ? 'i-lucide-chevron-up' : 'i-lucide-chevron-down'"
                />
              </button>
            </th>
            <th
              class="table__num"
              :aria-sort="sort.key === 'total' ? (sort.dir === 1 ? 'ascending' : 'descending') : 'none'"
            >
              <button type="button" class="th-btn" :class="{ 'th-btn--active': sort.key === 'total' }" @click="setSort('total')">
                Articles
                <span
                  v-if="sort.key === 'total'"
                  aria-hidden="true"
                  class="th-btn__arrow"
                  :class="sort.dir === 1 ? 'i-lucide-chevron-up' : 'i-lucide-chevron-down'"
                />
              </button>
            </th>
            <th
              class="table__num"
              :aria-sort="sort.key === 'latest' ? (sort.dir === 1 ? 'ascending' : 'descending') : 'none'"
            >
              <button type="button" class="th-btn" :class="{ 'th-btn--active': sort.key === 'latest' }" @click="setSort('latest')">
                Latest article
                <span
                  v-if="sort.key === 'latest'"
                  aria-hidden="true"
                  class="th-btn__arrow"
                  :class="sort.dir === 1 ? 'i-lucide-chevron-up' : 'i-lucide-chevron-down'"
                />
              </button>
            </th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="f in sortedFeeds" :key="f.feed_id">
            <td class="table__feed">{{ f.title }}</td>
            <td class="table__num">{{ f.total }}</td>
            <td class="table__num">{{ formatLatest(f.latest_at) }}</td>
          </tr>
        </tbody>
      </table>
    </div>

    <div v-else-if="state.loading" class="state" aria-live="polite">
      <span aria-hidden="true" class="state__icon i-lucide-loader-circle" />
      Loading statistics…
    </div>
    <div v-else-if="state.error" class="state state--error" role="alert">
      <span aria-hidden="true" class="state__icon i-lucide-circle-alert" />
      {{ state.error }}
    </div>
    <div v-else class="state">
      <span aria-hidden="true" class="state__icon i-lucide-inbox" />
      No feeds yet.
    </div>
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
.stats__body {
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.stats__chart {
  position: relative;
  border-radius: 8px;
}
.mask {
  position: absolute;
  inset: 0;
  z-index: 8;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  background: color-mix(in srgb, var(--bg) 62%, transparent);
  border-radius: 8px;
  color: var(--text-muted);
  font-size: 13px;
}
.mask__spinner {
  font-size: 16px;
  animation: mask-spin 0.8s linear infinite;
}
@keyframes mask-spin {
  to {
    transform: rotate(360deg);
  }
}
.stats__nav {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 12px;
  padding: 10px 14px;
  border: 1px solid var(--border);
  border-radius: 8px;
  background: var(--surface);
}
.stats__back {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  color: var(--accent);
  font-size: 12.5px;
  font-weight: 500;
  text-decoration: none;
  white-space: nowrap;
}
.stats__back:hover {
  text-decoration: underline;
}
.stats__title {
  margin: 0;
  font-size: 18px;
  font-weight: 700;
  color: var(--text);
  flex: 1;
  min-width: 0;
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
.th-btn {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 0;
  border: 0;
  background: none;
  color: inherit;
  font: inherit;
  font-weight: 600;
  cursor: pointer;
}
.th-btn:hover {
  color: var(--text);
}
.th-btn--active {
  color: var(--accent-soft-text);
}
.th-btn__arrow {
  font-size: 11px;
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
</style>
