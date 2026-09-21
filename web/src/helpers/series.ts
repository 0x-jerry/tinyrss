import type { FeedStat } from '../types/models'

// Total articles a feed published inside the selected range (sum of its per-day
// series counts). Ranks feeds by how active they are in the charted period
// rather than across their whole lifetime, which is what the trend chart shows.
export function seriesActivity(f: FeedStat): number {
  return f.series.reduce((sum, d) => sum + d.count, 0)
}

// Feeds that actually contribute points to the chart (have at least one dated
// entry in range). Feeds with no series can't be drawn, so they don't belong in
// the legend's toggleable set.
export function chartableFeeds(feeds: FeedStat[]): FeedStat[] {
  return feeds.filter((f) => f.series.length > 0)
}

// The set of feed ids that should be visible by default. When there are few
// feeds they all render; past `max` we keep only the `max` most active (by
// in-range activity) so the chart stays readable instead of becoming a tangle
// of every line. The rest are hidden but still available in the legend.
export function defaultVisibleIds(feeds: FeedStat[], max: number): Set<number> {
  const chartable = chartableFeeds(feeds)
  if (chartable.length <= max) return new Set(chartable.map((f) => f.feed_id))
  return new Set(
    [...chartable]
      .sort((a, b) => seriesActivity(b) - seriesActivity(a))
      .slice(0, max)
      .map((f) => f.feed_id),
  )
}

export interface HoverRow {
  feedId: number
  title: string
  count: number
}

// The feeds to show in the hover detail box for a single day: every visible
// (not hidden) feed that published at least one article that day, sorted by
// count descending so the busiest feeds sit at the top.
export function hoverRows(
  feeds: FeedStat[],
  hiddenFeeds: Set<number>,
  date: string,
): HoverRow[] {
  const rows: HoverRow[] = []
  for (const f of feeds) {
    if (hiddenFeeds.has(f.feed_id)) continue
    const d = f.series.find((s) => s.date === date)
    if (d && d.count > 0) rows.push({ feedId: f.feed_id, title: f.title, count: d.count })
  }
  rows.sort((a, b) => b.count - a.count)
  return rows
}
