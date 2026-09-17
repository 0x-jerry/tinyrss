import { describe, it, expect, vi, beforeEach } from 'vitest'
import { buildItemQuery, createItemsProvider } from '../src/providers/items'
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
})
