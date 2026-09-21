import { describe, it, expect, vi } from 'vitest'
import { useFeedStats } from '../src/composables/useFeedStats'
import { api } from '../src/api/endpoints'
import type { FeedStatsResponse } from '../src/types/models'

vi.mock('../src/api/endpoints', () => ({
  api: { feedStats: vi.fn() },
}))

const sample: FeedStatsResponse = {
  days: 30,
  feeds: [
    {
      feed_id: 1,
      title: 'A',
      latest_at: '2024-05-02 09:00:00',
      total: 3,
      series: [
        { date: '2024-05-01', count: 2 },
        { date: '2024-05-02', count: 1 },
      ],
    },
    {
      feed_id: 2,
      title: 'B',
      latest_at: '',
      total: 0,
      series: [],
    },
  ],
}

describe('useFeedStats', () => {
  it('loads the feed statistics for the requested range', async () => {
    vi.mocked(api.feedStats).mockResolvedValue(sample)

    const { state, load } = useFeedStats()
    await load(30)

    expect(api.feedStats).toHaveBeenCalledWith(30)
    expect(state.loading).toBe(false)
    expect(state.error).toBe('')
    expect(state.data).toEqual(sample)
    // Both feeds drive the chart (series) and the summary table (total/latest).
    expect(state.data?.feeds).toHaveLength(2)
    expect(state.data?.feeds[0].series).toHaveLength(2)
    expect(state.data?.feeds[1].total).toBe(0)
  })

  it('reports an error when the request fails', async () => {
    vi.mocked(api.feedStats).mockRejectedValue(new Error('boom'))

    const { state, load } = useFeedStats()
    await load(7)

    expect(api.feedStats).toHaveBeenCalledWith(7)
    expect(state.data).toBeNull()
    expect(state.error).toBe('boom')
    expect(state.loading).toBe(false)
  })
})
