// Mirror of the backend JSON contract. Field names must stay exact.

export interface Folder {
  id: number
  name: string
  sort_order: number
}

export interface Feed {
  id: number
  title: string
  feed_url: string
  site_url: string
  description: string
  render_mode: number // 0=content, 1=iframe site view
  folder_id: number | null
  etag: string
  last_modified: string
  last_fetched_at: string // "" when never fetched
  fetch_error: string
  unread: number
  created_at: string
  updated_at: string
}

// Item as returned by the list endpoint (no summary/content).
export interface Item {
  id: number
  feed_id: number
  feed_title: string
  title: string
  url: string
  author: string
  published_at: string
  is_read: boolean
  is_starred: boolean
  created_at: string
}

// Item as returned by the detail endpoint: list fields plus summary/content.
export interface ItemDetail extends Item {
  summary: string
  content: string
}

export interface ItemsResponse {
  items: Item[]
  total: number
  page: number
  limit: number
}

export interface FeedRefreshResult {
  new_items: number
  error: string
}

export interface ReadAllResult {
  count: number
}

export interface RefreshResult {
  running: boolean
  total: number
  done: number
  failed: number
  new_items: number
  current_feed_id: number
  current_feed_title: string
}

export interface Stats {
  feeds: number
  items: number
  unread: number
}

export interface Health {
  ok: boolean
}

export interface OpmlImportResult {
  added: number
}
