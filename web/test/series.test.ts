import { describe, it, expect } from 'vitest'
import { seriesActivity, chartableFeeds, defaultVisibleIds, hoverRows } from '../src/helpers/series'
import type { FeedStat } from '../src/types/models'

function feed(id: number, title: string, series: { date: string; count: number }[]): FeedStat {
  return { feed_id: id, title, latest_at: '', total: 0, series }
}

const feeds: FeedStat[] = [
  feed(1, 'A', [
    { date: '2024-05-01', count: 2 },
    { date: '2024-05-02', count: 1 },
  ]),
  feed(2, 'B', []),
  feed(3, 'C', [{ date: '2024-05-02', count: 9 }]),
  feed(4, 'D', [{ date: '2024-05-03', count: 4 }]),
]

describe('seriesActivity', () => {
  it('sums the per-day counts of a feed', () => {
    expect(seriesActivity(feeds[0])).toBe(3)
    expect(seriesActivity(feeds[2])).toBe(9)
  })
  it('returns 0 for a feed with no series', () => {
    expect(seriesActivity(feeds[1])).toBe(0)
  })
})

describe('chartableFeeds', () => {
  it('drops feeds with an empty series (cannot be plotted)', () => {
    const chartable = chartableFeeds(feeds)
    expect(chartable.map((f) => f.feed_id)).toEqual([1, 3, 4])
  })
})

describe('defaultVisibleIds', () => {
  it('shows every chartable feed when the count is at or below the max', () => {
    expect(defaultVisibleIds(feeds, 3)).toEqual(new Set([1, 3, 4]))
    expect(defaultVisibleIds(feeds, 10)).toEqual(new Set([1, 3, 4]))
  })
  it('keeps the most active feeds by in-range series activity when over the max', () => {
    // Activity: C=9, D=4, A=3, B=0(not chartable) → keep C and D.
    expect(defaultVisibleIds(feeds, 2)).toEqual(new Set([3, 4]))
  })
  it('returns an empty set when no feeds are chartable', () => {
    expect(defaultVisibleIds([feeds[1]], 5).size).toBe(0)
  })
})

describe('hoverRows', () => {
  it('returns visible feeds with count > 0 on the given day, sorted desc', () => {
    // feeds: A(2 on 05-01, 1 on 05-02), B(empty), C(9 on 05-02), D(4 on 05-03)
    const rows = hoverRows(feeds, new Set(), '2024-05-02')
    expect(rows).toEqual([
      { feedId: 3, title: 'C', count: 9 },
      { feedId: 1, title: 'A', count: 1 },
    ])
  })
  it('excludes hidden feeds and zero-count feeds', () => {
    const rows = hoverRows(feeds, new Set([1, 3]), '2024-05-02')
    expect(rows).toEqual([])
    expect(hoverRows(feeds, new Set(), '2024-05-03')).toEqual([{ feedId: 4, title: 'D', count: 4 }])
  })
  it('returns [] for a day with no articles', () => {
    expect(hoverRows(feeds, new Set(), '2024-04-01')).toEqual([])
  })
})
