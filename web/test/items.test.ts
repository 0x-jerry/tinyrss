import { describe, it, expect, vi, beforeEach } from 'vitest'
import { buildItemQuery, createItemsProvider, scopeKeyOf, parseScopeFilters } from '../src/providers/items'
import type { ItemsApi } from '../src/api/endpoints'
import type { ItemsResponse } from '../src/types/models'

const item = (id: number) => ({
  id,
  feed_id: 1,
  feed_title: 'f',
  title: `t${id}`,
  url: '',
  author: '',
  published_at: '',
  is_read: false,
  is_starred: false,
  created_at: '',
})

function makeApi(respond: (query: string) => Promise<ItemsResponse>): ItemsApi {
  return {
    listItems: respond,
    getItem: async () => ({ ...item(1), summary: '', content: '' }),
    setItemState: async () => {},
    readAll: async () => ({ count: 0 }),
  }
}

describe('buildItemQuery', () => {
  it('prefers feed_id over folder_id', () => {
    const q = buildItemQuery({ feedId: 5, folderId: 3 }, 'all', '', 1, 50)
    expect(q).toContain('feed_id=5')
    expect(q).not.toContain('folder_id')
  })

  it('uses folder_id when no feed is selected', () => {
    const q = buildItemQuery({ feedId: null, folderId: 3 }, 'all', '', 1, 50)
    expect(q).toContain('folder_id=3')
  })

  it('renders filter, search, pagination and limits', () => {
    const q = buildItemQuery({ feedId: null, folderId: null }, 'unread', 'hello', 2, 50)
    expect(q).toContain('unread=true')
    expect(q).toContain('search=hello')
    expect(q).toContain('page=2')
    expect(q).toContain('limit=50')
  })
})

describe('items provider', () => {
  const selection = { feedId: null, folderId: null }
  let calls: string[]
  let onItemsChanged: ReturnType<typeof vi.fn>

  beforeEach(() => {
    calls = []
    onItemsChanged = vi.fn()
  })

  function makeProvider() {
    const apiObj = makeApi(async (query) => {
      calls.push(query)
      const page = Number(/page=(\d+)/.exec(query)?.[1] ?? 1)
      return { items: [item(page)], total: 150, page, limit: 50 } as ItemsResponse
    })
    return createItemsProvider({ apiObj, getSelection: () => selection, onItemsChanged })
  }

  it('loads the first page', async () => {
    const p = makeProvider()
    await p.load()
    expect(p.state.items.length).toBe(1)
    expect(p.state.page).toBe(1)
    expect(p.state.total).toBe(150)
    expect(calls[0]).toContain('page=1')
    expect(calls[0]).not.toContain('feed_id')
    expect(calls[0]).not.toContain('folder_id')
  })

  it('appends the next page and advances pagination', async () => {
    const p = makeProvider()
    await p.load()
    await p.nextPage()
    expect(p.state.page).toBe(2)
    expect(p.state.items.length).toBe(2)
  })

  it('selects a feed and rebuilds the query on reload', async () => {
    selection.feedId = 7
    const p = makeProvider()
    await p.load()
    expect(calls[0]).toContain('feed_id=7')
  })

  it('defaults to unread and falls back to all when nothing is unread', async () => {
    const queries: string[] = []
    const apiObj = makeApi(async (query) => {
      queries.push(query)
      return query.includes('unread=true')
        ? ({ items: [], total: 0, page: 1, limit: 50 } as ItemsResponse)
        : ({ items: [item(1)], total: 1, page: 1, limit: 50 } as ItemsResponse)
    })
    const p = createItemsProvider({ apiObj, getSelection: () => selection, onItemsChanged })
    expect(p.state.filter).toBe('unread')
    await p.load()
    expect(p.state.filter).toBe('all')
    expect(p.state.items.length).toBe(1)
    expect(queries[0]).toContain('unread=true')
    expect(queries[1]).not.toContain('unread=true')
  })

  it('honors a filter remembered per feed scope', async () => {
    const sel = { feedId: null as number | null, folderId: null as number | null }
    const queries: string[] = []
    const apiObj = makeApi(async (query) => {
      queries.push(query)
      return { items: [item(1)], total: 1, page: 1, limit: 50 } as ItemsResponse
    })
    const p = createItemsProvider({ apiObj, getSelection: () => sel, onItemsChanged })
    // Choose starred on feed 7...
    sel.feedId = 7
    await p.setFilter('starred')
    expect(queries[queries.length - 1]).toContain('starred=true')
    // ...feed 8 keeps its own default (unread), not feed 7's starred...
    sel.feedId = 8
    await p.load()
    expect(queries[queries.length - 1]).toContain('unread=true')
    expect(queries[queries.length - 1]).not.toContain('starred=true')
    // ...and switching back to feed 7 restores starred.
    sel.feedId = 7
    await p.load()
    expect(queries[queries.length - 1]).toContain('starred=true')
  })

  it('keys the scope by feed, then folder, then all articles', () => {
    expect(scopeKeyOf({ feedId: 7, folderId: 3 })).toBe('feed:7')
    expect(scopeKeyOf({ feedId: null, folderId: 3 })).toBe('folder:3')
    expect(scopeKeyOf({ feedId: null, folderId: null })).toBe('all')
  })

  it('parses scope filters, dropping malformed entries', () => {
    expect(parseScopeFilters('{"feed:1":"starred","folder:2":"unread","feed:3":"nope","all":true}')).toEqual({
      'feed:1': 'starred',
      'folder:2': 'unread',
    })
    expect(parseScopeFilters('not json')).toEqual({})
  })

  it('keeps the unread filter when there are unread items', async () => {
    const apiObj = makeApi(async (query) => {
      const unread = query.includes('unread=true')
      return { items: unread ? [item(1)] : [], total: 1, page: 1, limit: 50 } as ItemsResponse
    })
    const p = createItemsProvider({ apiObj, getSelection: () => selection, onItemsChanged })
    await p.load()
    expect(p.state.filter).toBe('unread')
    expect(p.state.items.length).toBe(1)
  })

  it('does not fall back to all when the user explicitly picks unread', async () => {
    const apiObj = makeApi(async (query) =>
      query.includes('unread=true')
        ? ({ items: [], total: 0, page: 1, limit: 50 } as ItemsResponse)
        : ({ items: [item(1)], total: 1, page: 1, limit: 50 } as ItemsResponse),
    )
    const p = createItemsProvider({ apiObj, getSelection: () => selection, onItemsChanged })
    await p.setFilter('unread')
    expect(p.state.filter).toBe('unread')
    expect(p.state.items.length).toBe(0)
  })

  it('toggles filters', async () => {
    const p = makeProvider()
    await p.setFilter('unread')
    expect(calls[0]).toContain('unread=true')
    expect(p.state.filter).toBe('unread')
    await p.setSearch('cats')
    expect(calls[1]).toContain('search=cats')
  })

  it('marks an item read optimistically and notifies', async () => {
    const p = makeProvider()
    await p.load()
    expect(p.state.items[0].is_read).toBe(false)
    await p.markRead(p.state.items[0].id)
    expect(p.state.items[0].is_read).toBe(true)
    expect(onItemsChanged).toHaveBeenCalled()
  })

  it('marks all read for the current selection', async () => {
    selection.folderId = 4
    const p = makeProvider()
    await p.load()
    await p.markAllRead()
    expect(p.state.items.every((i) => i.is_read)).toBe(true)
  })

  it('toggleRead flips read state via the api and notifies', async () => {
    const actions: string[] = []
    const apiObj = makeApi(async (query) => ({ items: [item(1)], total: 1, page: 1, limit: 50 }))
    apiObj.setItemState = async (_id, action) => {
      actions.push(action)
    }
    const p = createItemsProvider({ apiObj, getSelection: () => selection, onItemsChanged })
    await p.load()
    await p.toggleRead(p.state.items[0].id)
    expect(p.state.items[0].is_read).toBe(true)
    expect(actions).toEqual(['read'])
    expect(onItemsChanged).toHaveBeenCalled()
    await p.toggleRead(p.state.items[0].id)
    expect(p.state.items[0].is_read).toBe(false)
    expect(actions).toEqual(['read', 'unread'])
  })

  it('toggleStar flips star state via the api', async () => {
    const actions: string[] = []
    const apiObj = makeApi(async () => ({ items: [item(1)], total: 1, page: 1, limit: 50 }))
    apiObj.setItemState = async (_id, action) => {
      actions.push(action)
    }
    const p = createItemsProvider({ apiObj, getSelection: () => selection, onItemsChanged })
    await p.load()
    await p.toggleStar(p.state.items[0].id)
    expect(p.state.items[0].is_starred).toBe(true)
    expect(actions).toEqual(['star'])
    await p.toggleStar(p.state.items[0].id)
    expect(p.state.items[0].is_starred).toBe(false)
    expect(actions).toEqual(['star', 'unstar'])
  })
})
