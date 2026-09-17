import { request } from './client'
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

export interface ItemsApi {
  listItems(query: string): Promise<ItemsResponse>
  getItem(id: number): Promise<ItemDetail>
  setItemState(id: number, action: ItemAction): Promise<void>
  readAll(feedId: number | null, folderId: number | null): Promise<ReadAllResult>
}

export interface FeedsTreeApi {
  listFeeds(): Promise<Feed[]>
  listFolders(): Promise<Folder[]>
  createFeed(feedUrl: string): Promise<Feed>
  updateFeed(id: number, title: string, folderId: number | null): Promise<Feed>
  deleteFeed(id: number): Promise<void>
  refreshFeed(id: number): Promise<FeedRefreshResult>
  refreshAllFeeds(): Promise<RefreshResult>
  readAll(feedId: number | null, folderId: number | null): Promise<ReadAllResult>
  createFolder(name: string): Promise<Folder>
  updateFolder(id: number, name: string): Promise<Folder>
  deleteFolder(id: number): Promise<void>
}

function readAllQuery(feedId: number | null, folderId: number | null): string {
  if (feedId != null) return `?feed_id=${feedId}`
  if (folderId != null) return `?folder_id=${folderId}`
  return ''
}

export const api: ItemsApi & FeedsTreeApi & {
  getFeed: (id: number) => Promise<Feed>
  importOpml: (form: FormData) => Promise<OpmlImportResult>
  exportOpml: () => Promise<string>
  health: () => Promise<Health>
  stats: () => Promise<Stats>
} = {
  // Feeds
  listFeeds: () => request<Feed[]>('GET', '/api/feeds'),
  createFeed: (feedUrl) => request<Feed>('POST', '/api/feeds', { feed_url: feedUrl }),
  getFeed: (id) => request<Feed>('GET', `/api/feeds/${id}`),
  updateFeed: (id, title, folderId) => request<Feed>('PUT', `/api/feeds/${id}`, { title, folder_id: folderId }),
  deleteFeed: (id) => request<void>('DELETE', `/api/feeds/${id}`),
  refreshFeed: (id) => request<FeedRefreshResult>('POST', `/api/feeds/${id}/refresh`),

  // Folders
  listFolders: () => request<Folder[]>('GET', '/api/folders'),
  createFolder: (name) => request<Folder>('POST', '/api/folders', { name }),
  updateFolder: (id, name) => request<Folder>('PUT', `/api/folders/${id}`, { name }),
  deleteFolder: (id) => request<void>('DELETE', `/api/folders/${id}`),

  // Items
  listItems: (query) => request<ItemsResponse>('GET', `/api/items?${query}`),
  getItem: (id) => request<ItemDetail>('GET', `/api/items/${id}`),
  setItemState: (id, action) => request<void>('POST', `/api/items/${id}/${action}`),
  readAll: (feedId, folderId) => request<ReadAllResult>('POST', `/api/items/read-all${readAllQuery(feedId, folderId)}`),

  // System
  refreshAllFeeds: () => request<RefreshResult>('POST', '/api/refresh'),
  health: () => request<Health>('GET', '/api/health'),
  stats: () => request<Stats>('GET', '/api/stats'),
  importOpml: (form) => request<OpmlImportResult>('POST', '/api/opml/import', form),
  exportOpml: () => request<string>('GET', '/api/opml/export'),
}
