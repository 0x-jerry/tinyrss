import { useIntervalFn } from '@vueuse/core'
import { onScopeDispose } from 'vue'
import type { FeedsTreeProvider } from '../providers/feedsTree'
import type { ItemsProvider } from '../providers/items'

/**
 * Polls feed tree (unread badges) and the item list every intervalMs seconds.
 * Handlers swallow errors — a 401 already bounces to login via the api client.
 */
export function useAutoRefresh(feedsTree: FeedsTreeProvider, items?: ItemsProvider, intervalMs = 60_000) {
  const { pause, resume, isActive } = useIntervalFn(
    () => {
      feedsTree.reload().catch(() => {})
      if (items) items.load().catch(() => {})
    },
    intervalMs,
    { immediate: false },
  )
  onScopeDispose(pause)
  return { pause, resume, isActive }
}
