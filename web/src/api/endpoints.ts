import { request, requestText } from './client'
import type {
  Feed,
  FeedRefreshResult,
  Folder,
  Health,
  ItemDetail,
  ItemsResponse,
  OpmlImportResult,
  ReadAllResult,
  RefreshResult,
  Stats,
} from '../types/models'

export type ItemAction = 'read' | 'unread' | 'star' | 'unstar'

function readAllQuery(feedId: number | null, folderId: number | null): string {
  if (feedId != null) return `?feed_id=${feedId}`
  if (folderId != null) return `?folder_id=${folderId}`
  return ''
}

export const api = {
  // Feeds
  listFeeds: () => request<Feed[]>('GET', '/api/feeds'),
  createFeed: (feedUrl: string, folderId?: number | null) =>
    request<Feed>('POST', '/api/feeds', { feed_url: feedUrl, folder_id: folderId ?? null }),
  getFeed: (id: number) => request<Feed>('GET', `/api/feeds/${id}`),
  updateFeed: (
    id: number,
    patch: { title: string; feed_url?: string; site_url?: string; description?: string; folder_id?: number | null },
  ) => request<Feed>('PUT', `/api/feeds/${id}`, patch),
  deleteFeed: (id: number) => request<void>('DELETE', `/api/feeds/${id}`),
  refreshFeed: (id: number) => request<FeedRefreshResult>('POST', `/api/feeds/${id}/refresh`),
  setFeedRenderMode: (id: number, mode: number) =>
    request<Feed>('POST', `/api/feeds/${id}/render-mode`, { render_mode: mode }),

  // Folders
  listFolders: () => request<Folder[]>('GET', '/api/folders'),
  createFolder: (name: string) => request<Folder>('POST', '/api/folders', { name }),
  updateFolder: (id: number, name: string) => request<Folder>('PUT', `/api/folders/${id}`, { name }),
  deleteFolder: (id: number) => request<void>('DELETE', `/api/folders/${id}`),

  // Items
  listItems: (query: string) => request<ItemsResponse>('GET', `/api/items?${query}`),
  getItem: (id: number) => request<ItemDetail>('GET', `/api/items/${id}`),
  setItemState: (id: number, action: ItemAction) => request<void>('POST', `/api/items/${id}/${action}`),
  readAll: (feedId: number | null, folderId: number | null) =>
    request<ReadAllResult>('POST', `/api/items/read-all${readAllQuery(feedId, folderId)}`),

  // Render
  renderUrl: (url: string) => requestText('GET', `/api/render?url=${encodeURIComponent(url)}`),

  // System
  refreshAllFeeds: () => request<RefreshResult>('POST', '/api/refresh'),
  refreshProgress: () => request<RefreshResult>('GET', '/api/refresh/progress'),
  health: () => request<Health>('GET', '/api/health'),
  stats: () => request<Stats>('GET', '/api/stats'),
  importOpml: (form: FormData) => request<OpmlImportResult>('POST', '/api/opml/import', form),
  exportOpml: () => requestText('GET', '/api/opml/export'),
}
