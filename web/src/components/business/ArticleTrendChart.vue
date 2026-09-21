<script setup lang="ts">
import { computed, ref } from 'vue'
import type { FeedStat } from '../../types/models'

export interface ArticleTrendChartProps {
  feeds: FeedStat[]
  days: number
}

const props = defineProps<ArticleTrendChartProps>()

const hidden = ref<Set<number>>(new Set())

const palette = [
  'var(--accent)',
  '#22c55e',
  '#eab308',
  '#3b82f6',
  '#ec4899',
  '#8b5cf6',
  '#f97316',
]

function feedIndex(id: number): number {
  return props.feeds.findIndex((f) => f.feed_id === id)
}

function colorOf(id: number): string {
  const i = feedIndex(id)
  return palette[(i < 0 ? 0 : i) % palette.length]
}

// Shared x timeline: every date seen across all feeds, ascending.
const timeline = computed<string[]>(() => {
  const dates = new Set<string>()
  for (const f of props.feeds) for (const d of f.series) dates.add(d.date)
  return [...dates].sort()
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
  const idxs = n <= 2 ? timeline.value.map((_, i) => i) : [0, Math.floor((n - 1) / 2), n - 1]
  return idxs.map((i) => ({ x: xFor(i), label: timeline.value[i] }))
})

function toggle(id: number) {
  const next = new Set(hidden.value)
  if (next.has(id)) next.delete(id)
  else next.add(id)
  hidden.value = next
}
</script>

<template>
  <div class="chart">
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
    </svg>

    <ul class="legend">
      <li v-for="f in feeds" :key="f.feed_id" class="legend__item">
        <button
          type="button"
          class="legend__chip"
          :class="{ 'legend__chip--off': hidden.has(f.feed_id) }"
          :aria-pressed="!hidden.has(f.feed_id)"
          @click="toggle(f.feed_id)"
        >
          <span class="legend__swatch" :style="{ background: colorOf(f.feed_id) }" />
          <span class="legend__label">{{ f.title }}</span>
        </button>
      </li>
    </ul>
  </div>
</template>

<style scoped>
.chart {
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.chart__svg {
  width: 100%;
  height: auto;
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
.legend {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  list-style: none;
  margin: 0;
  padding: 0;
}
.legend__chip {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 4px 8px;
  border: 1px solid var(--border-strong);
  border-radius: 6px;
  background: var(--surface);
  color: var(--text-secondary);
  font: inherit;
  font-size: 12px;
  line-height: 1;
  cursor: pointer;
}
.legend__chip:hover {
  background: var(--bg-hover);
}
.legend__chip--off {
  opacity: 0.45;
}
.legend__swatch {
  width: 8px;
  height: 8px;
  border-radius: 2px;
}
.legend__label {
  max-width: 180px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
</style>
