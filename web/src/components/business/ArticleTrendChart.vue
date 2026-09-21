<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import type { FeedStat } from '../../types/models'
import { chartableFeeds, defaultVisibleIds, hoverRows, seriesActivity } from '../../helpers/series'

export interface ArticleTrendChartProps {
  feeds: FeedStat[]
  days: number
}

const props = defineProps<ArticleTrendChartProps>()

// Feeds the user has explicitly hidden. The default keeps only the most active
// feeds visible; toggling any chip, or the All/None/Top controls, edits this set
// for the current range.
const hidden = ref<Set<number>>(new Set())
// Legend search: narrows down which chips are listed, useful when there are 100+.
const query = ref('')

const DEFAULT_VISIBLE = 12

const palette = [
  'var(--accent)',
  '#22c55e',
  '#eab308',
  '#3b82f6',
  '#ec4899',
  '#8b5cf6',
  '#f97316',
  '#14b8a6',
  '#f43f5e',
  '#84cc16',
  '#06b6d4',
  '#a855f7',
]

// Reset visibility to the sane default (top DEFAULT_VISIBLE active feeds). Ran on
// mount and again whenever `days` changes so each range starts from a clean
// slate without clobbering in-range manual toggles.
function applyDefaults() {
  hidden.value = defaultVisibleIds(props.feeds, DEFAULT_VISIBLE)
}
applyDefaults()
watch(() => props.days, applyDefaults)

function feedIndex(id: number): number {
  return props.feeds.findIndex((f) => f.feed_id === id)
}

function colorOf(id: number): string {
  const i = feedIndex(id)
  return palette[(i < 0 ? 0 : i) % palette.length]
}

// "YYYY-MM-DD" in UTC, matching the calendar the backend buckets articles by.
function isoUTC(date: Date): string {
  const y = date.getUTCFullYear()
  const m = String(date.getUTCMonth() + 1).padStart(2, '0')
  const d = String(date.getUTCDate()).padStart(2, '0')
  return `${y}-${m}-${d}`
}

// Full day-by-day x timeline spanning the trailing `days` window (today going
// back days-1), even when feeds only published on a handful of dates. Being a
// real time axis, it visibly stretches/relabels as the range filter changes
// instead of collapsing to just the days that happen to have articles.
const timeline = computed<string[]>(() => {
  const n = Math.max(1, props.days)
  const now = new Date()
  const start = new Date(Date.UTC(now.getUTCFullYear(), now.getUTCMonth(), now.getUTCDate() - (n - 1)))
  const out: string[] = []
  for (let i = 0; i < n; i++) {
    const d = new Date(start)
    d.setUTCDate(start.getUTCDate() + i)
    out.push(isoUTC(d))
  }
  return out
})

const maxY = computed(() => {
  let m = 0
  for (const f of props.feeds) for (const d of f.series) if (d.count > m) m = d.count
  return m
})

const W = 800
const H = 240
const padL = 30
const padR = 10
const padT = 12
const padB = 24
const plotW = W - padL - padR
const plotH = H - padT - padB

const xFor = (index: number) =>
  timeline.value.length <= 1
    ? padL + plotW / 2
    : padL + (index / (timeline.value.length - 1)) * plotW

const yFor = (count: number) => {
  const scale = maxY.value > 0 ? maxY.value : 1
  return H - padB - (count / scale) * plotH
}

interface SeriesPath {
  feedId: number
  points: [number, number][]
  color: string
}

const visibleSeries = computed<SeriesPath[]>(() =>
  props.feeds
    .filter((f) => !hidden.value.has(f.feed_id))
    .map((f) => {
      const points = f.series
        .map((d) => [xFor(timeline.value.indexOf(d.date)), yFor(d.count)] as [number, number])
        .filter((p) => p[0] >= padL && p[0] <= W - padR)
      return { feedId: f.feed_id, points, color: colorOf(f.feed_id) }
    })
    .filter((s) => s.points.length > 0),
)

const linePath = (points: [number, number][]) =>
  points.map((p, i) => `${i === 0 ? 'M' : 'L'}${p[0]},${p[1]}`).join(' ')

const areaPath = (points: [number, number][]) =>
  points.length === 0
    ? ''
    : `${linePath(points)} L${points[points.length - 1][0]},${H - padB} L${points[0][0]},${H - padB} Z`

const yTicks = computed(() => {
  const ticks: { y: number; value: number }[] = []
  const steps = 4
  for (let i = 0; i <= steps; i++) {
    const value = Math.round((maxY.value / steps) * i)
    ticks.push({ y: yFor(value), value })
  }
  return ticks
})

const xLabels = computed(() => {
  const n = timeline.value.length
  if (n === 0) return []
  // Space out a handful of evenly distributed labels; more for wide ranges.
  const maxLabels = n <= 7 ? n : 5
  const idxs: number[] = []
  for (let k = 0; k < maxLabels; k++) idxs.push(Math.round((k * (n - 1)) / (maxLabels - 1)))
  const uniq = [...new Set(idxs)]
  return uniq.map((i) => ({ x: xFor(i), label: timeline.value[i] }))
})

// Legend only deals with feeds that can be drawn, filtered by the search query.
const legendItems = computed<FeedStat[]>(() => {
  const q = query.value.trim().toLowerCase()
  const items = chartableFeeds(props.feeds)
  if (!q) return items
  return items.filter((f) => f.title.toLowerCase().includes(q))
})

const totalCount = computed(() => legendItems.value.length)
const shownCount = computed(
  () => legendItems.value.filter((f) => !hidden.value.has(f.feed_id)).length,
)

function toggle(id: number) {
  const next = new Set(hidden.value)
  if (next.has(id)) next.delete(id)
  else next.add(id)
  hidden.value = next
}

function showAll() {
  hidden.value = new Set()
}

function hideAll() {
  hidden.value = new Set(chartableFeeds(props.feeds).map((f) => f.feed_id))
}

// --- Hover detail -----------------------------------------------------------

// Index into `timeline` for the day under the pointer (snapped to a day bucket).
const hoverIndex = ref<number | null>(null)

function onMove(e: PointerEvent) {
  const rect = (e.currentTarget as Element).getBoundingClientRect()
  if (rect.width === 0 || timeline.value.length === 0) return
  // Map the pointer X into viewBox coordinates, then to the nearest day index.
  const vx = ((e.clientX - rect.left) / rect.width) * plotW + padL
  const idx = Math.round(((vx - padL) / plotW) * (timeline.value.length - 1))
  hoverIndex.value = Math.max(0, Math.min(timeline.value.length - 1, idx))
}

function onLeave() {
  hoverIndex.value = null
}

const hoverDate = computed(() =>
  hoverIndex.value == null ? null : timeline.value[hoverIndex.value],
)
const hoverRowsList = computed(() =>
  hoverDate.value == null ? [] : hoverRows(props.feeds, hidden.value, hoverDate.value),
)
const hoverTotal = computed(() => hoverRowsList.value.reduce((sum, r) => sum + r.count, 0))
const crosshairX = computed(() => (hoverIndex.value == null ? 0 : xFor(hoverIndex.value)))
const tooltipStyle = computed(() =>
  hoverIndex.value == null ? null : { left: `${(crosshairX.value / W) * 100}%` },
)
const tooltipFlip = computed(
  () => hoverIndex.value != null && hoverIndex.value > Math.floor(timeline.value.length / 2),
)

// 'YYYY-MM-DD' → humanized, keeping UTC so the day matches the axis labels.
function formatDate(iso: string): string {
  const [y, m, d] = iso.split('-').map(Number)
  return new Date(Date.UTC(y, m - 1, d)).toLocaleDateString(undefined, {
    timeZone: 'UTC',
    month: 'short',
    day: 'numeric',
    year: 'numeric',
  })
}

</script>

<template>
  <div class="chart">
    <div class="chart__plot" @pointerleave="onLeave">
      <svg
        class="chart__svg"
        :viewBox="`0 0 ${W} ${H}`"
        role="img"
        :aria-label="`Article counts per feed over the last ${days} days`"
      >
        <title>Article counts per feed over the last {{ days }} days</title>
        <g v-for="t in yTicks" :key="`y${t.value}`">
          <line :x1="padL" :y1="t.y" :x2="W - padR" :y2="t.y" class="grid" />
          <text :x="padL - 6" :y="t.y + 3" class="axis" text-anchor="end">{{ t.value }}</text>
        </g>
        <g v-for="l in xLabels" :key="`x${l.label}`">
          <text :x="l.x" :y="H - 6" class="axis" text-anchor="middle">{{ l.label }}</text>
        </g>
        <g v-for="s in visibleSeries" :key="`s${s.feedId}`">
          <path :d="areaPath(s.points)" class="area" :style="{ fill: s.color }" />
          <path :d="linePath(s.points)" class="line" :style="{ stroke: s.color }" fill="none" />
        </g>
        <g v-if="hoverIndex != null">
          <line :x1="crosshairX" :x2="crosshairX" :y1="padT" :y2="H - padB" class="guide" />
        </g>
        <rect
          v-if="visibleSeries.length > 0"
          :x="padL"
          :y="padT"
          :width="plotW"
          :height="plotH"
          fill="transparent"
          class="hover"
          @pointermove="onMove"
          @pointerdown="onMove"
        />
      </svg>

      <div
        v-if="tooltipStyle"
        class="tooltip"
        :class="{ 'tooltip--flip': tooltipFlip }"
        :style="tooltipStyle"
      >
        <div class="tooltip__head">
          <span class="tooltip__date">{{ hoverDate ? formatDate(hoverDate) : '' }}</span>
          <span v-if="hoverRowsList.length" class="tooltip__total">Total {{ hoverTotal }}</span>
        </div>
        <div v-if="hoverRowsList.length" class="tooltip__rows">
          <div v-for="r in hoverRowsList" :key="r.feedId" class="tooltip__row">
            <span class="tooltip__swatch" :style="{ background: colorOf(r.feedId) }" />
            <span class="tooltip__title">{{ r.title }}</span>
            <span class="tooltip__count">{{ r.count }}</span>
          </div>
        </div>
        <div v-else class="tooltip__empty">No articles on this date.</div>
      </div>
    </div>

    <div v-if="visibleSeries.length === 0" class="chart__empty">
      No activity in this period.
    </div>

    <fieldset class="legend">
      <legend class="legend__title">
        Legend
        <span class="legend__count">{{ shownCount }} / {{ totalCount }}</span>
      </legend>

      <div class="legend__toolbar">
        <span aria-hidden="true" class="i-lucide-search text-[13px] legend__search-icon" />
        <input
          v-model="query"
          class="legend__search"
          type="search"
          placeholder="Filter feeds…"
          aria-label="Filter feeds"
        />
        <button type="button" class="legend__act" title="Show all feeds" @click="showAll">All</button>
        <button type="button" class="legend__act" title="Hide all feeds" @click="hideAll">None</button>
        <button type="button" class="legend__act" title="Show only the most active" @click="applyDefaults">Top</button>
      </div>

      <ul class="legend__list">
        <li v-for="f in legendItems" :key="f.feed_id" class="legend__item">
          <button
            type="button"
            class="legend__chip"
            :class="{ 'legend__chip--off': hidden.has(f.feed_id) }"
            :aria-pressed="!hidden.has(f.feed_id)"
            :title="f.title"
            @click="toggle(f.feed_id)"
          >
            <span class="legend__swatch" :style="{ background: colorOf(f.feed_id) }" />
            <span class="legend__label">{{ f.title }}</span>
            <span class="legend__value">{{ seriesActivity(f) }}</span>
          </button>
        </li>
        <li v-if="legendItems.length === 0" class="legend__none">No matching feeds.</li>
      </ul>
    </fieldset>
  </div>
</template>

<style scoped>
.chart {
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.chart__plot {
  position: relative;
}
.chart__svg {
  width: 100%;
  height: auto;
  display: block;
}
.chart__empty {
  margin-top: -4px;
  padding: 8px 12px;
  border: 1px dashed var(--border-strong);
  border-radius: 8px;
  color: var(--text-faint);
  font-size: 12px;
  text-align: center;
}
.grid {
  stroke: var(--border);
  stroke-width: 1;
}
.axis {
  fill: var(--text-faint);
  font-size: 9px;
}
.line {
  stroke-width: 2;
}
.area {
  opacity: 0.08;
}
.guide {
  stroke: var(--border-strong);
  stroke-width: 1;
  stroke-dasharray: 3 3;
}
.hover {
  cursor: crosshair;
}
.tooltip {
  position: absolute;
  top: 14%;
  z-index: 5;
  min-width: 140px;
  max-width: 260px;
  padding: 8px 10px;
  border: 1px solid var(--border);
  border-radius: 8px;
  background: var(--surface);
  box-shadow: 0 6px 20px rgba(0, 0, 0, 0.14);
  font-size: 11px;
  color: var(--text-secondary);
  white-space: nowrap;
  transform: translate(8px, -50%);
}
.tooltip--flip {
  transform: translate(calc(-100% - 8px), -50%);
}
.tooltip__head {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 6px;
}
.tooltip__date {
  font-size: 11px;
  font-weight: 600;
  color: var(--text);
}
.tooltip__total {
  flex: none;
  font-weight: 600;
  color: var(--text);
  font-variant-numeric: tabular-nums;
}
.tooltip__rows {
  display: flex;
  flex-direction: column;
  gap: 3px;
  max-height: 150px;
  overflow-y: auto;
  overscroll-behavior: contain;
  -webkit-overflow-scrolling: touch;
}
.tooltip__rows::-webkit-scrollbar {
  width: 6px;
}
.tooltip__rows::-webkit-scrollbar-thumb {
  background: var(--border-strong);
  border-radius: 3px;
}
.tooltip__rows::-webkit-scrollbar-track {
  background: transparent;
}
.tooltip__row {
  display: flex;
  align-items: center;
  gap: 6px;
  max-width: 240px;
  padding-right: 2px;
}
.tooltip__swatch {
  flex: none;
  width: 8px;
  height: 8px;
  border-radius: 2px;
}
.tooltip__title {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.tooltip__count {
  flex: none;
  margin-left: 8px;
  color: var(--text-faint);
  font-variant-numeric: tabular-nums;
}
.tooltip__empty {
  color: var(--text-faint);
}
.legend {
  margin: 0;
  padding: 8px;
  border: 1px solid var(--border);
  border-radius: 8px;
  background: var(--surface);
}
.legend__title {
  padding: 0 2px;
  font-size: 12px;
  font-weight: 600;
  color: var(--text-secondary);
}
.legend__count {
  margin-left: 6px;
  font-weight: 500;
  color: var(--text-faint);
}
.legend__toolbar {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-top: 8px;
}
.legend__search-icon {
  color: var(--text-faint);
}
.legend__search {
  flex: 1;
  min-width: 0;
  padding: 4px 8px;
  border: 1px solid var(--border-strong);
  border-radius: 6px;
  font: inherit;
  font-size: 12px;
  color: var(--text);
  background: var(--bg);
}
.legend__search:focus-visible {
  outline: 2px solid var(--accent);
  outline-offset: -1px;
}
.legend__act {
  padding: 4px 8px;
  border: 1px solid var(--border-strong);
  border-radius: 6px;
  background: var(--bg-subtle);
  color: var(--text-muted);
  font: inherit;
  font-size: 12px;
  line-height: 1;
  cursor: pointer;
}
.legend__act:hover {
  background: var(--bg-hover);
  color: var(--text);
}
.legend__list {
  max-height: 200px;
  overflow-y: auto;
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  list-style: none;
  margin: 8px 0 0;
  padding: 0;
}
.legend__chip {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 4px 8px;
  border: 1px solid var(--border-strong);
  border-radius: 6px;
  background: var(--bg-subtle);
  color: var(--text-secondary);
  font: inherit;
  font-size: 12px;
  line-height: 1;
  cursor: pointer;
  max-width: 220px;
}
.legend__chip:hover {
  background: var(--bg-hover);
}
.legend__chip--off {
  opacity: 0.45;
}
.legend__swatch {
  flex: none;
  width: 8px;
  height: 8px;
  border-radius: 2px;
}
.legend__label {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.legend__value {
  flex: none;
  color: var(--text-faint);
  font-variant-numeric: tabular-nums;
}
.legend__none {
  padding: 6px 2px;
  color: var(--text-faint);
  font-size: 12px;
}
</style>
