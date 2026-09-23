import { describe, it, expect, vi, beforeEach } from 'vitest'
import { buildItemQuery, createItemsStore, scopeKeyOf, parseScopeFilters } from '../src/store/items'
import { api } from '../src/api/endpoints'
import type { ItemsResponse } from '../src/types/models'

vi.mock('../src/api/endpoints', () => ({
  api: {
    listItems: vi.fn(),
    getItem: vi.fn(),
    setItemState: vi.fn(),
    readAll: vi.fn(),
  },
}))

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

const pageResponse = (query: string): ItemsResponse => {
  const page = Number(/page=(\d+)/.exec(query)?.[1] ?? 1)
  return { items: [item(page)], total: 150, page, limit: 50 }
}

describe('buildItemQuery', () => {
  it('adds feed_id when a feed is selected and omits it otherwise', () => {
    expect(buildItemQuery({ feedId: 5 }, 'all', '', 1, 50)).toContain('feed_id=5')
    expect(buildItemQuery({ feedId: null }, 'all', '', 1, 50)).not.toContain('feed_id')
  })

  it('renders filter, search, pagination and limits', () => {
    const q = buildItemQuery({ feedId: null }, 'unread', 'hello', 2, 50)
    expect(q).toContain('unread=true')
    expect(q).toContain('search=hello')
    expect(q).toContain('page=2')
    expect(q).toContain('limit=50')
  })
})

describe('items provider', () => {
  const selection = { feedId: null }
  let calls: string[]
  let onItemsChanged: ReturnType<typeof vi.fn>
  let adjustUnread: ReturnType<typeof vi.fn>

  beforeEach(() => {
    calls = []
    onItemsChanged = vi.fn()
    adjustUnread = vi.fn()
    vi.mocked(api.listItems).mockReset().mockImplementation(async (query: string) => {
      calls.push(query)
      return pageResponse(query)
    })
    vi.mocked(api.getItem).mockReset()
    vi.mocked(api.setItemState).mockReset()
    vi.mocked(api.readAll).mockReset()
  })

  function provider() {
    return createItemsStore({ getSelection: () => selection, onItemsChanged, adjustUnread })
  }

  it('loads the first page', async () => {
    const p = provider()
    await p.load()
    expect(p.state.items.length).toBe(1)
    expect(p.state.page).toBe(1)
    expect(p.state.total).toBe(150)
    expect(calls[0]).toContain('page=1')
    expect(calls[0]).not.toContain('feed_id')
  })

  it('appends the next page and advances pagination', async () => {
    const p = provider()
    await p.load()
    await p.nextPage()
    expect(p.state.page).toBe(2)
    expect(p.state.items.length).toBe(2)
  })

  it('selects a feed and rebuilds the query on reload', async () => {
    selection.feedId = 7
    const p = provider()
    await p.load()
    expect(calls[0]).toContain('feed_id=7')
  })

  it('defaults to unread and falls back to all when nothing is unread', async () => {
    const queries: string[] = []
    vi.mocked(api.listItems).mockImplementation(async (query: string) => {
      queries.push(query)
      return query.includes('unread=true')
        ? ({ items: [], total: 0, page: 1, limit: 50 } as ItemsResponse)
        : ({ items: [item(1)], total: 1, page: 1, limit: 50 } as ItemsResponse)
    })
    const p = createItemsStore({ getSelection: () => selection, onItemsChanged })
    expect(p.state.filter).toBe('unread')
    await p.load()
    expect(p.state.filter).toBe('all')
    expect(p.state.items.length).toBe(1)
    expect(queries[0]).toContain('unread=true')
    expect(queries[1]).not.toContain('unread=true')
  })

  it('honors a filter remembered per feed scope', async () => {
    const sel = { feedId: null as number | null }
    const queries: string[] = []
    vi.mocked(api.listItems).mockImplementation(async (query: string) => {
      queries.push(query)
      return { items: [item(1)], total: 1, page: 1, limit: 50 } as ItemsResponse
    })
    const p = createItemsStore({ getSelection: () => sel, onItemsChanged })
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

  it('keys the scope by feed, then all articles', () => {
    expect(scopeKeyOf({ feedId: 7 })).toBe('feed:7')
    expect(scopeKeyOf({ feedId: null })).toBe('all')
  })

  it('parses scope filters, dropping malformed entries', () => {
    expect(parseScopeFilters('{"feed:1":"starred","folder:2":"unread","feed:3":"nope","all":true}')).toEqual({
      'feed:1': 'starred',
      'folder:2': 'unread',
    })
    expect(parseScopeFilters('not json')).toEqual({})
  })

  it('keeps the unread filter when there are unread items', async () => {
    vi.mocked(api.listItems).mockImplementation(async (query: string) => {
      const unread = query.includes('unread=true')
      return { items: unread ? [item(1)] : [], total: 1, page: 1, limit: 50 } as ItemsResponse
    })
    const p = createItemsStore({ getSelection: () => selection, onItemsChanged })
    await p.load()
    expect(p.state.filter).toBe('unread')
    expect(p.state.items.length).toBe(1)
  })

  it('does not fall back to all when the user explicitly picks unread', async () => {
    vi.mocked(api.listItems).mockImplementation(async (query: string) =>
      query.includes('unread=true')
        ? ({ items: [], total: 0, page: 1, limit: 50 } as ItemsResponse)
        : ({ items: [item(1)], total: 1, page: 1, limit: 50 } as ItemsResponse),
    )
    const p = createItemsStore({ getSelection: () => selection, onItemsChanged })
    await p.setFilter('unread')
    expect(p.state.filter).toBe('unread')
    expect(p.state.items.length).toBe(0)
  })

  it('toggles filters', async () => {
    const p = provider()
    await p.setFilter('unread')
    expect(calls[0]).toContain('unread=true')
    expect(p.state.filter).toBe('unread')
    await p.setSearch('cats')
    expect(calls[1]).toContain('search=cats')
  })

  it('marks an item read optimistically and decrements the feed badge', async () => {
    const p = provider()
    await p.load()
    expect(p.state.items[0].is_read).toBe(false)
    await p.markRead(p.state.items[0].id)
    expect(p.state.items[0].is_read).toBe(true)
    expect(adjustUnread).toHaveBeenCalledWith(1, -1)
  })

  it('marks all read for the current selection', async () => {
    selection.feedId = 4
    const p = provider()
    await p.load()
    await p.markAllRead()
    expect(p.state.items.every((i) => i.is_read)).toBe(true)
  })

  it('toggleRead flips read state via the api and adjusts the feed badge', async () => {
    const actions: string[] = []
    vi.mocked(api.setItemState).mockImplementation(async (_id: number, action: string) => {
      actions.push(action)
    })
    const p = createItemsStore({ getSelection: () => selection, onItemsChanged, adjustUnread })
    await p.load()
    await p.toggleRead(p.state.items[0].id)
    expect(p.state.items[0].is_read).toBe(true)
    expect(actions).toEqual(['read'])
    expect(adjustUnread).toHaveBeenLastCalledWith(1, -1)
    await p.toggleRead(p.state.items[0].id)
    expect(p.state.items[0].is_read).toBe(false)
    expect(actions).toEqual(['read', 'unread'])
    expect(adjustUnread).toHaveBeenLastCalledWith(1, 1)
  })

  it('openItem marks read optimistically and decrements the feed badge', async () => {
    const p = createItemsStore({ getSelection: () => selection, onItemsChanged, adjustUnread })
    await p.load()
    await p.openItem(p.state.items[0].id)
    expect(p.state.items[0].is_read).toBe(true)
    expect(adjustUnread).toHaveBeenCalledWith(1, -1)
  })

  it('openItem selects synchronously', async () => {
    const selectItem = vi.fn()
    const p = createItemsStore({ getSelection: () => selection, selectItem })
    await p.load()
    await p.openItem(p.state.items[0].id)
    expect(selectItem).toHaveBeenCalledWith(1)
  })

  it('openItem skips unread updates when the item is already read', async () => {
    const actions: string[] = []
    vi.mocked(api.setItemState).mockImplementation(async (_id: number, action: string) => {
      actions.push(action)
    })
    vi.mocked(api.listItems).mockResolvedValue({
      items: [{ ...item(1), is_read: true }],
      total: 1,
      page: 1,
      limit: 50,
    } as ItemsResponse)
    const p = createItemsStore({ getSelection: () => selection, onItemsChanged, adjustUnread })
    await p.load()
    await p.openItem(p.state.items[0].id)
    expect(adjustUnread).not.toHaveBeenCalled()
    expect(actions).toEqual([])
  })

  it('toggleStar flips star state via the api', async () => {
    const actions: string[] = []
    vi.mocked(api.setItemState).mockImplementation(async (_id: number, action: string) => {
      actions.push(action)
    })
    const p = createItemsStore({ getSelection: () => selection, onItemsChanged })
    await p.load()
    await p.toggleStar(p.state.items[0].id)
    expect(p.state.items[0].is_starred).toBe(true)
    expect(actions).toEqual(['star'])
    await p.toggleStar(p.state.items[0].id)
    expect(p.state.items[0].is_starred).toBe(false)
    expect(actions).toEqual(['star', 'unstar'])
  })
})
