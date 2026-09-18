import { inject, reactive, readonly, provide, type DeepReadonly } from 'vue'
import { feedsTreeKey } from './keys'
import { api } from '../api/endpoints'
import type { Feed, Folder, RefreshResult } from '../types/models'

export interface FolderNode {
  id: number
  name: string
  sortOrder: number
  feeds: Feed[]
  unread: number
}

export interface TreeShape {
  folderNodes: FolderNode[]
  uncategorized: Feed[]
  uncategorizedUnread: number
  totalFeeds: number
  totalUnread: number
}

export function buildTree(feeds: readonly Feed[], folders: readonly Folder[]): TreeShape {
  const sorted = [...feeds].sort((a, b) => a.title.localeCompare(b.title))
  const folderNodes = folders
    .map((f) => {
      const folderFeeds = sorted.filter((x) => x.folder_id === f.id)
      return {
        id: f.id,
        name: f.name,
        sortOrder: f.sort_order,
        feeds: folderFeeds,
        unread: folderFeeds.reduce((s, x) => s + x.unread, 0),
      }
    })
    .sort((a, b) => a.sortOrder - b.sortOrder)
  const uncategorized = sorted.filter((x) => x.folder_id === null)
  return {
    folderNodes,
    uncategorized,
    uncategorizedUnread: uncategorized.reduce((s, x) => s + x.unread, 0),
    totalFeeds: sorted.length,
    totalUnread: feeds.reduce((s, x) => s + x.unread, 0),
  }
}

export interface UpdateFeedPatch {
  title: string
  feed_url?: string
  site_url?: string
  description?: string
  folder_id?: number | null
}

export interface RefreshState {
  running: boolean
  total: number
  done: number
  failed: number
  newItems: number
  currentTitle: string
}

export interface FeedsTreeState {
  feeds: Feed[]
  folders: Folder[]
  tree: TreeShape
  loading: boolean
  refresh: RefreshState
}

export interface FeedsTreeProvider {
  state: DeepReadonly<FeedsTreeState>
  reload: () => Promise<void>
  addFeed: (feedUrl: string, folderId?: number | null) => Promise<Feed>
  updateFeed: (id: number, patch: UpdateFeedPatch) => Promise<void>
  deleteFeed: (id: number) => Promise<void>
  moveFeed: (id: number, folderId: number | null) => Promise<void>
  addFolder: (name: string) => Promise<void>
  renameFolder: (id: number, name: string) => Promise<void>
  deleteFolder: (id: number) => Promise<void>
  refreshAll: () => Promise<void>
  refreshFeed: (id: number) => Promise<void>
  resumeRefresh: () => Promise<void>
  setRenderMode: (id: number, mode: number) => Promise<void>
  markAllRead: () => Promise<void>
  importOpmlForm: (form: FormData) => Promise<number>
  exportOpmlText: () => Promise<string>
}

export function createFeedsTreeProvider(): FeedsTreeProvider {
  const raw = reactive<FeedsTreeState>({
    feeds: [],
    folders: [],
    tree: {
      folderNodes: [],
      uncategorized: [],
      uncategorizedUnread: 0,
      totalFeeds: 0,
      totalUnread: 0,
    },
    loading: false,
    refresh: { running: false, total: 0, done: 0, failed: 0, newItems: 0, currentTitle: '' },
  })

  function apply(feeds: Feed[], folders: Folder[]) {
    raw.feeds = feeds
    raw.folders = folders
    raw.tree = buildTree(feeds, folders)
  }

  function applyRefresh(r: RefreshResult) {
    raw.refresh = {
      running: r.running,
      total: r.total,
      done: r.done,
      failed: r.failed,
      newItems: r.new_items,
      currentTitle: r.current_feed_title,
    }
  }

  // Poll the background refresh-all job until it stops, applying progress to
  // the tree state as it goes. Shared by refreshAll and resumeRefresh.
  async function pollRefresh() {
    for (;;) {
      const job = await api.refreshProgress()
      applyRefresh(job)
      if (!job.running) break
      await sleep(500)
    }
  }

  async function reload() {
    raw.loading = true
    try {
      const [feeds, folders] = await Promise.all([api.listFeeds(), api.listFolders()])
      apply(feeds, folders)
    } finally {
      raw.loading = false
    }
  }

  return {
    state: readonly(raw),
    reload,
    async addFeed(feedUrl, folderId = null) {
      const feed = await api.createFeed(feedUrl, folderId)
      await reload()
      return feed
    },
    updateFeed: async (id, patch) => {
      await api.updateFeed(id, patch)
      await reload()
    },
    deleteFeed: async (id) => {
      await api.deleteFeed(id)
      await reload()
    },
    moveFeed: async (id, folderId) => {
      const feed = raw.feeds.find((f) => f.id === id)
      await api.updateFeed(id, { title: feed?.title ?? '', folder_id: folderId })
      await reload()
    },
    addFolder: async (name) => {
      await api.createFolder(name)
      await reload()
    },
    renameFolder: async (id, name) => {
      await api.updateFolder(id, name)
      await reload()
    },
    deleteFolder: async (id) => {
      await api.deleteFolder(id)
      await reload()
    },
    async refreshAll() {
      await api.refreshAllFeeds()
      await pollRefresh()
      await reload()
    },
    refreshFeed: async (id) => {
      await api.refreshFeed(id)
      await reload()
    },
    // Pick up a refresh-all still running on the server (e.g. after a page
    // reload): keep driving the progress bar, then resync the tree when done.
    async resumeRefresh() {
      const job = await api.refreshProgress()
      if (!job.running) return
      await pollRefresh()
      await reload()
    },
    setRenderMode: async (id, mode) => {
      const updated = await api.setFeedRenderMode(id, mode)
      const feed = raw.feeds.find((f) => f.id === id)
      if (feed) feed.render_mode = updated.render_mode
    },
    markAllRead: async () => {
      await api.readAll(null, null)
      await reload()
    },
    importOpmlForm: async (form) => {
      const res = await api.importOpml(form)
      await reload()
      return res.added
    },
    exportOpmlText: () => api.exportOpml(),
  }
}

export function provideFeedsTree(): FeedsTreeProvider {
  const p = createFeedsTreeProvider()
  provide(feedsTreeKey, p)
  return p
}

function sleep(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms))
}

export function injectFeedsTree(): FeedsTreeProvider {
  const p = inject<FeedsTreeProvider>(feedsTreeKey)
  if (!p) throw new Error('feedsTree provider not provided')
  return p
}
