import { describe, it, expect, vi } from 'vitest'
import { createFeedsTreeProvider } from '../src/providers/feedsTree'
import { api } from '../src/api/endpoints'
import type { Feed, Folder } from '../src/types/models'

vi.mock('../src/api/endpoints', () => ({
  api: {
    listFeeds: vi.fn(),
    listFolders: vi.fn(),
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
  },
}))

function withFeedsTree(feeds: Feed[], folders: Folder[] = []) {
  vi.mocked(api.listFeeds).mockResolvedValue(feeds)
  vi.mocked(api.listFolders).mockResolvedValue(folders)
  return createFeedsTreeProvider()
}

describe('feedsTree provider import/export', () => {
  it('imports an OPML file and reloads the tree', async () => {
    vi.mocked(api.importOpml).mockResolvedValue({ added: 3 })
    withFeedsTree([] as Feed[])

    const provider = createFeedsTreeProvider()
    const added = await provider.importOpmlForm(new FormData())

    expect(added).toBe(3)
    expect(api.importOpml).toHaveBeenCalledWith(expect.any(FormData))
    // reload() pulls feeds+folders after import.
    expect(api.listFeeds).toHaveBeenCalled()
    expect(api.listFolders).toHaveBeenCalled()
  })

  it('returns the exported OPML text', async () => {
    const opml = '<?xml version="1.0"?><opml version="2.0"><body/></opml>'
    vi.mocked(api.exportOpml).mockResolvedValue(opml)

    const provider = createFeedsTreeProvider()

    await expect(provider.exportOpmlText()).resolves.toBe(opml)
  })

  it('sets the render mode on a feed and reflects it in state', async () => {
    const feed = { id: 7, render_mode: 0 } as Feed
    vi.mocked(api.setFeedRenderMode).mockResolvedValue({ ...feed, render_mode: 1 })
    const provider = withFeedsTree([feed])
    await provider.reload()

    await provider.setRenderMode(7, 1)

    expect(api.setFeedRenderMode).toHaveBeenCalledWith(7, 1)
    expect(provider.state.feeds.find((f) => f.id === 7)?.render_mode).toBe(1)
  })

  it('persists server render mode', async () => {
    const feed = { id: 8, render_mode: 0 } as Feed
    vi.mocked(api.setFeedRenderMode).mockResolvedValue({ ...feed, render_mode: 2 })
    const provider = withFeedsTree([feed])
    await provider.reload()

    await provider.setRenderMode(8, 2)

    expect(provider.state.feeds.find((f) => f.id === 8)?.render_mode).toBe(2)
  })
})
