import { describe, it, expect, vi } from 'vitest'
import { createFeedsTreeProvider } from '../src/providers/feedsTree'
import type { FeedsTreeApi } from '../src/api/endpoints'
import type { Feed, Folder } from '../src/types/models'

function makeApi(overrides: Partial<FeedsTreeApi> = {}): FeedsTreeApi {
  return {
    listFeeds: vi.fn().mockResolvedValue([] as Feed[]),
    listFolders: vi.fn().mockResolvedValue([] as Folder[]),
    createFeed: vi.fn(),
    updateFeed: vi.fn(),
    deleteFeed: vi.fn(),
    refreshFeed: vi.fn(),
    setFeedRenderMode: vi.fn(),
    refreshAllFeeds: vi.fn(),
    readAll: vi.fn(),
    createFolder: vi.fn(),
    updateFolder: vi.fn(),
    deleteFolder: vi.fn(),
    importOpml: vi.fn(),
    exportOpml: vi.fn(),
    ...overrides,
  }
}

describe('feedsTree provider import/export', () => {
  it('imports an OPML file and reloads the tree', async () => {
    const apiObj = makeApi({ importOpml: vi.fn().mockResolvedValue({ added: 3 }) })
    const p = createFeedsTreeProvider(apiObj)

    const added = await p.importOpmlForm(new FormData())

    expect(added).toBe(3)
    expect(apiObj.importOpml).toHaveBeenCalledWith(expect.any(FormData))
    // reload() pulls feeds+folders after import.
    expect(apiObj.listFeeds).toHaveBeenCalled()
    expect(apiObj.listFolders).toHaveBeenCalled()
  })

  it('returns the exported OPML text', async () => {
    const opml = '<?xml version="1.0"?><opml version="2.0"><body/></opml>'
    const apiObj = makeApi({ exportOpml: vi.fn().mockResolvedValue(opml) })
    const p = createFeedsTreeProvider(apiObj)

    await expect(p.exportOpmlText()).resolves.toBe(opml)
  })

  it('sets the render mode on a feed and reflects it in state', async () => {
    const feed = { id: 7, render_mode: 0 } as Feed
    const apiObj = makeApi({
      listFeeds: vi.fn().mockResolvedValue([feed]),
      setFeedRenderMode: vi.fn().mockResolvedValue({ ...feed, render_mode: 1 }),
    })
    const p = createFeedsTreeProvider(apiObj)
    await p.reload()

    await p.setRenderMode(7, 1)

    expect(apiObj.setFeedRenderMode).toHaveBeenCalledWith(7, 1)
    expect(p.state.feeds.find((f) => f.id === 7)?.render_mode).toBe(1)
  })

  it('persists server render mode', async () => {
    const feed = { id: 8, render_mode: 0 } as Feed
    const apiObj = makeApi({
      listFeeds: vi.fn().mockResolvedValue([feed]),
      setFeedRenderMode: vi.fn().mockResolvedValue({ ...feed, render_mode: 2 }),
    })
    const p = createFeedsTreeProvider(apiObj)
    await p.reload()

    await p.setRenderMode(8, 2)

    expect(p.state.feeds.find((f) => f.id === 8)?.render_mode).toBe(2)
  })
})
