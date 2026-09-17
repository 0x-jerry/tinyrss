import { inject, reactive, readonly, provide, type DeepReadonly } from 'vue'
import { itemsKey } from './keys'
import type { SelectionProvider } from './selection'
import type { FeedsTreeProvider } from './feedsTree'
import { api, type ItemsApi } from '../api/endpoints'
import type { Item, ItemDetail } from '../types/models'

export type Filter = 'all' | 'unread' | 'starred'

export interface ItemsState {
  items: Item[]
  total: number
  page: number
  limit: number
  loading: boolean
  filter: Filter
  search: string
  selectedItem: ItemDetail | null
}

export interface ItemsDeps {
  apiObj?: ItemsApi
  getSelection: () => { feedId: number | null; folderId: number | null }
  onItemsChanged?: () => void
}

export interface ItemsProvider {
  state: DeepReadonly<ItemsState>
  load: () => Promise<void>
  nextPage: () => Promise<void>
  setFilter: (f: Filter) => Promise<void>
  setSearch: (s: string) => Promise<void>
  openItem: (id: number) => Promise<void>
  markRead: (id: number) => Promise<void>
  markUnread: (id: number) => Promise<void>
  star: (id: number) => Promise<void>
  unstar: (id: number) => Promise<void>
  markAllRead: () => Promise<void>
}

export function buildItemQuery(
  selection: { feedId: number | null; folderId: number | null },
  filter: Filter,
  search: string,
  page: number,
  limit: number,
): string {
  const p = new URLSearchParams()
  if (selection.feedId != null) p.set('feed_id', String(selection.feedId))
  else if (selection.folderId != null) p.set('folder_id', String(selection.folderId))
  if (filter === 'unread') p.set('unread', 'true')
  if (filter === 'starred') p.set('starred', 'true')
  if (search) p.set('search', search)
  p.set('page', String(page))
  p.set('limit', String(limit))
  return p.toString()
}

export function createItemsProvider(deps: ItemsDeps): ItemsProvider {
  const apiObj = deps.apiObj ?? api
  const raw = reactive<ItemsState>({
    items: [],
    total: 0,
    page: 1,
    limit: 50,
    loading: false,
    filter: 'all',
    search: '',
    selectedItem: null,
  })

  function notify() {
    deps.onItemsChanged?.()
  }

  async function fetchPage(page: number, replace: boolean) {
    raw.loading = true
    try {
      const res = await apiObj.listItems(buildItemQuery(deps.getSelection(), raw.filter, raw.search, page, raw.limit))
      raw.items = replace ? res.items : raw.items.concat(res.items)
      raw.total = res.total
      raw.page = res.page
      raw.limit = res.limit
    } finally {
      raw.loading = false
    }
  }

  function patch(id: number, patch: Partial<Item>) {
    const item = raw.items.find((i) => i.id === id)
    if (item) Object.assign(item, patch)
    if (raw.selectedItem?.id === id) Object.assign(raw.selectedItem, patch)
  }

  return {
    state: readonly(raw),
    load: () => fetchPage(1, true),
    nextPage: async () => {
      if (raw.page * raw.limit >= raw.total) return
      await fetchPage(raw.page + 1, false)
    },
    setFilter: async (f) => {
      raw.filter = f
      await fetchPage(1, true)
    },
    setSearch: async (s) => {
      raw.search = s
      await fetchPage(1, true)
    },
    openItem: async (id) => {
      const detail = await apiObj.getItem(id)
      raw.selectedItem = detail
      // Mark read optimistically; the local list + unread counts refresh right away.
      patch(id, { is_read: true })
      notify()
      await apiObj.setItemState(id, 'read')
    },
    markRead: async (id) => {
      patch(id, { is_read: true })
      notify()
      await apiObj.setItemState(id, 'read')
    },
    markUnread: async (id) => {
      patch(id, { is_read: false })
      notify()
      await apiObj.setItemState(id, 'unread')
    },
    star: async (id) => {
      patch(id, { is_starred: true })
      await apiObj.setItemState(id, 'star')
    },
    unstar: async (id) => {
      patch(id, { is_starred: false })
      await apiObj.setItemState(id, 'unstar')
    },
    markAllRead: async () => {
      await apiObj.readAll(deps.getSelection().feedId, deps.getSelection().folderId)
      raw.items.forEach((i) => (i.is_read = true))
      notify()
    },
  }
}

export function provideItems(deps: { selection: SelectionProvider; feedsTree: FeedsTreeProvider }): ItemsProvider {
  const provider = createItemsProvider({
    getSelection: () => ({
      feedId: deps.selection.state.feedId,
      folderId: deps.selection.state.folderId,
    }),
    onItemsChanged: () => deps.feedsTree.reload(),
  })
  provide(itemsKey, provider)
  return provider
}

export function injectItems(): ItemsProvider {
  const p = inject<ItemsProvider>(itemsKey)
  if (!p) throw new Error('items provider not provided')
  return p
}
