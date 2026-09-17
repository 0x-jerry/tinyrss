import { inject, reactive, readonly, provide, type DeepReadonly } from 'vue'
import { feedsTreeKey } from './keys'
import { api, type FeedsTreeApi } from '../api/endpoints'
import type { Feed, Folder } from '../types/models'

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

export function buildTree(feeds: Feed[], folders: Folder[]): TreeShape {
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

export interface FeedsTreeState {
  feeds: Feed[]
  folders: Folder[]
  tree: TreeShape
  loading: boolean
}

export interface FeedsTreeProvider {
  state: DeepReadonly<FeedsTreeState>
  reload: () => Promise<void>
  addFeed: (feedUrl: string) => Promise<Feed>
  renameFeed: (id: number, title: string) => Promise<void>
  deleteFeed: (id: number) => Promise<void>
  moveFeed: (id: number, folderId: number | null) => Promise<void>
  addFolder: (name: string) => Promise<void>
  renameFolder: (id: number, name: string) => Promise<void>
  deleteFolder: (id: number) => Promise<void>
  refreshAll: () => Promise<void>
  refreshFeed: (id: number) => Promise<void>
  markAllRead: () => Promise<void>
}

export function createFeedsTreeProvider(apiObj: FeedsTreeApi = api): FeedsTreeProvider {
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
  })

  function apply(feeds: Feed[], folders: Folder[]) {
    raw.feeds = feeds
    raw.folders = folders
    raw.tree = buildTree(feeds, folders)
  }

  async function reload() {
    raw.loading = true
    try {
      const [feeds, folders] = await Promise.all([apiObj.listFeeds(), apiObj.listFolders()])
      apply(feeds, folders)
    } finally {
      raw.loading = false
    }
  }

  return {
    state: readonly(raw),
    reload,
    async addFeed(feedUrl) {
      const feed = await apiObj.createFeed(feedUrl)
      await reload()
      return feed
    },
    renameFeed: async (id, title) => {
      const feed = raw.feeds.find((f) => f.id === id)
      await apiObj.updateFeed(id, title, feed?.folder_id ?? null)
      await reload()
    },
    deleteFeed: async (id) => {
      await apiObj.deleteFeed(id)
      await reload()
    },
    moveFeed: async (id, folderId) => {
      const feed = raw.feeds.find((f) => f.id === id)
      await apiObj.updateFeed(id, feed?.title ?? '', folderId)
      await reload()
    },
    addFolder: async (name) => {
      await apiObj.createFolder(name)
      await reload()
    },
    renameFolder: async (id, name) => {
      await apiObj.updateFolder(id, name)
      await reload()
    },
    deleteFolder: async (id) => {
      await apiObj.deleteFolder(id)
      await reload()
    },
    refreshAll: async () => {
      await apiObj.refreshAllFeeds()
      await reload()
    },
    refreshFeed: async (id) => {
      await apiObj.refreshFeed(id)
      await reload()
    },
    markAllRead: async () => {
      await apiObj.readAll(null, null)
      await reload()
    },
  }
}

export function provideFeedsTree(): FeedsTreeProvider {
  const p = createFeedsTreeProvider()
  provide(feedsTreeKey, p)
  return p
}

export function injectFeedsTree(): FeedsTreeProvider {
  const p = inject<FeedsTreeProvider>(feedsTreeKey)
  if (!p) throw new Error('feedsTree provider not provided')
  return p
}
