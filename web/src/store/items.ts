import { reactive, readonly } from 'vue'
import { useLocalStorage } from '@vueuse/core'
import { api } from '../api/endpoints'
import type { Item, ItemDetail } from '../types/models'

export type Filter = 'all' | 'unread' | 'starred'

const FILTERS: readonly Filter[] = ['all', 'unread', 'starred']
const SCOPE_FILTERS_KEY = 'tinyrss.scopeFilters'

export type ScopeKey = 'all' | `feed:${number}`

export function scopeKeyOf(selection: { feedId: number | null }): ScopeKey {
  if (selection.feedId != null) return `feed:${selection.feedId}`
  return 'all'
}

export function parseScopeFilters(raw: string): Partial<Record<ScopeKey, Filter>> {
  try {
    const v = JSON.parse(raw)
    if (typeof v !== 'object' || v === null) return {}
    const out: Partial<Record<ScopeKey, Filter>> = {}
    for (const [k, val] of Object.entries(v)) {
      if (FILTERS.includes(val as Filter)) (out as Record<string, Filter>)[k] = val as Filter
    }
    return out
  } catch {
    return {}
  }
}

const scopeFiltersSerializer = {
  read: (raw: string) => parseScopeFilters(raw),
  write: (v: Partial<Record<ScopeKey, Filter>>) => JSON.stringify(v),
}

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
  getSelection: () => { feedId: number | null }
  selectItem?: (id: number) => void
  onItemsChanged?: () => void
  /** Apply a ±1 unread-count change to the tree badge for the item's feed. */
  adjustUnread?: (feedId: number, delta: number) => void
}

export interface ItemsStore {
  state: Readonly<ItemsState>
  load: () => Promise<void>
  nextPage: () => Promise<void>
  setFilter: (f: Filter) => Promise<void>
  setSearch: (s: string) => Promise<void>
  openItem: (id: number) => Promise<void>
  toggleRead: (id: number) => Promise<void>
  toggleStar: (id: number) => Promise<void>
  markRead: (id: number) => Promise<void>
  markUnread: (id: number) => Promise<void>
  star: (id: number) => Promise<void>
  unstar: (id: number) => Promise<void>
  markAllRead: () => Promise<void>
}

export function buildItemQuery(
  selection: { feedId: number | null },
  filter: Filter,
  search: string,
  page: number,
  limit: number,
): string {
  const p = new URLSearchParams()
  if (selection.feedId != null) p.set('feed_id', String(selection.feedId))
  if (filter === 'unread') p.set('unread', 'true')
  if (filter === 'starred') p.set('starred', 'true')
  if (search) p.set('search', search)
  p.set('page', String(page))
  p.set('limit', String(limit))
  return p.toString()
}

export function createItemsStore(deps: ItemsDeps): ItemsStore {
  // Each feed (and "all articles") remembers its own filter.
  const filters = useLocalStorage<Partial<Record<ScopeKey, Filter>>>(SCOPE_FILTERS_KEY, {}, {
    serializer: scopeFiltersSerializer,
  })

  function currentScope(): ScopeKey {
    return scopeKeyOf(deps.getSelection())
  }

  const raw = reactive<ItemsState>({
    items: [],
    total: 0,
    page: 1,
    limit: 50,
    loading: false,
    // The active filter derives from the current scope's stored preference and is
    // recomputed on access, so it follows feed changes without an observer.
    get filter() {
      const f = filters.value[currentScope()]
      return f && FILTERS.includes(f) ? f : 'unread'
    },
    search: '',
    selectedItem: null,
  })

  function notify() {
    deps.onItemsChanged?.()
  }

  async function fetchPage(page: number, replace: boolean, defaultLoad = false) {
    raw.loading = true
    try {
      const currentFilter = raw.filter
      const res = await api.listItems(buildItemQuery(deps.getSelection(), currentFilter, raw.search, page, raw.limit))
      // Unread is the default filter; on a default load, when nothing is unread
      // in the current scope, fall back to showing everything instead of an
      // empty list. An explicit filter choice (setFilter) is never overridden.
      if (defaultLoad && replace && currentFilter === 'unread' && res.total === 0) {
        filters.value[currentScope()] = 'all'
        await fetchPage(1, true)
        return
      }
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

  // Every read/star mutation goes through these two: optimistic patch (plus a
  // tree-badge unread adjust for reads), then the API call. Toggles reuse them
  // to avoid re-duplicating the read/unread/star/unstar switching in every component.
  async function setItemRead(id: number, read: boolean) {
    const item = findItem(id)
    patch(id, { is_read: read })
    if (item) deps.adjustUnread?.(item.feed_id, read ? -1 : 1)
    await api.setItemState(id, read ? 'read' : 'unread')
  }
  async function setItemStarred(id: number, starred: boolean) {
    patch(id, { is_starred: starred })
    await api.setItemState(id, starred ? 'star' : 'unstar')
  }

  function findItem(id: number) {
    return raw.items.find((i) => i.id === id)
  }

  return {
    // read-only runtime guard from Vue's readonly(); items stays mutable-typed so
    // the virtual list (useVirtualList) receives Item[] without a component-side cast.
    state: readonly(raw) as Readonly<ItemsState>,
    load: () => fetchPage(1, true, true),
    nextPage: async () => {
      if (raw.page * raw.limit >= raw.total) return
      await fetchPage(raw.page + 1, false)
    },
    setFilter: async (f) => {
      filters.value[currentScope()] = f
      await fetchPage(1, true)
    },
    setSearch: async (s) => {
      raw.search = s
      await fetchPage(1, true)
    },
    openItem: async (id) => {
      // select + open in a single action: selecting drives the highlight/persist,
      // then the detail is fetched and marked read optimistically.
      deps.selectItem?.(id)
      const detail = await api.getItem(id)
      raw.selectedItem = detail

      if (detail.is_read) return
      patch(id, { is_read: true })
      deps.adjustUnread?.(detail.feed_id, -1)
      await api.setItemState(id, 'read')
    },
    toggleRead: async (id) => {
      const item = findItem(id)
      if (!item) return
      await setItemRead(id, !item.is_read)
    },
    toggleStar: async (id) => {
      const item = findItem(id)
      if (!item) return
      await setItemStarred(id, !item.is_starred)
    },
    markRead: (id) => setItemRead(id, true),
    markUnread: (id) => setItemRead(id, false),
    star: (id) => setItemStarred(id, true),
    unstar: (id) => setItemStarred(id, false),
    markAllRead: async () => {
      await api.readAll(deps.getSelection().feedId)
      raw.items.forEach((i) => (i.is_read = true))
      notify()
    },
  }
}

