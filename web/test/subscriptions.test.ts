import { describe, it, expect, vi } from 'vitest'
import { createFeedsTreeProvider } from '../src/providers/feedsTree'
import { api } from '../src/api/endpoints'
import type { Feed, Folder, RefreshResult } from '../src/types/models'

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
    refreshProgress: vi.fn(),
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

  it('adds a feed into the chosen group and reloads the tree', async () => {
    const feed = { id: 9, title: 'New', render_mode: 0 } as Feed
    vi.mocked(api.createFeed).mockResolvedValue(feed)
    withFeedsTree([] as Feed[])

    const provider = createFeedsTreeProvider()
    const created = await provider.addFeed('https://x.example/rss', 3)

    expect(created).toBe(feed)
    expect(api.createFeed).toHaveBeenCalledWith('https://x.example/rss', 3)
    expect(api.listFeeds).toHaveBeenCalled()
  })

  it('updates a feed and reloads the tree', async () => {
    const feed = { id: 10, title: 'Old', render_mode: 0 } as Feed
    vi.mocked(api.updateFeed).mockResolvedValue({ ...feed, title: 'New' })
    withFeedsTree([feed])

    const provider = createFeedsTreeProvider()
    const patch = {
      title: 'New',
      feed_url: 'https://x.example/rss2',
      site_url: 'https://x.example',
      description: 'desc',
      folder_id: 3,
    }
    await provider.updateFeed(10, patch)

    expect(api.updateFeed).toHaveBeenCalledWith(10, patch)
    expect(api.listFeeds).toHaveBeenCalled()
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

  it('polls refresh progress until idle then reloads', async () => {
    const running: RefreshResult = {
      running: true,
      total: 2,
      done: 1,
      failed: 0,
      new_items: 0,
      current_feed_id: 1,
      current_feed_title: 'A',
    }
    const idle: RefreshResult = {
      running: false,
      total: 2,
      done: 2,
      failed: 0,
      new_items: 3,
      current_feed_id: 0,
      current_feed_title: '',
    }
    vi.mocked(api.refreshAllFeeds).mockResolvedValue(running)
    vi.mocked(api.refreshProgress)
      .mockResolvedValueOnce(running)
      .mockResolvedValueOnce(idle)
    withFeedsTree([] as Feed[])

    const provider = createFeedsTreeProvider()
    await provider.refreshAll()

    expect(api.refreshAllFeeds).toHaveBeenCalled()
    expect(api.refreshProgress).toHaveBeenCalledTimes(2)
    // reload() pulls feeds+folders once the job is idle.
    expect(api.listFeeds).toHaveBeenCalled()
    expect(provider.state.refresh).toMatchObject({ running: false, total: 2, done: 2, failed: 0 })
  })
})
