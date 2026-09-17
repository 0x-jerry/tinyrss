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
})
