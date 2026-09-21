import { reactive } from 'vue'
import { api } from '../api/endpoints'
import type { FeedStatsResponse } from '../types/models'

export function useFeedStats(initialDays = 30) {
  const state = reactive<{
    days: number
    data: FeedStatsResponse | null
    loading: boolean
    error: string
  }>({
    days: initialDays,
    data: null,
    loading: false,
    error: '',
  })

  async function load(days: number) {
    state.loading = true
    state.error = ''
    try {
      state.data = await api.feedStats(days)
    } catch (e) {
      state.error = e instanceof Error ? e.message : 'Could not load statistics'
    } finally {
      state.loading = false
    }
  }

  return { state, load }
}
