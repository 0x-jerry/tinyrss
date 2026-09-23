<script setup lang="ts">
import { computed, ref } from 'vue'
import DOMPurify from 'dompurify'
import { computedAsync } from '@vueuse/core'
import { useStore } from '../../store'
import { useViewNav } from '../../composables/useViewNav'
import { useApiToast } from '../../api/useApiToast'
import { api } from '../../api/endpoints'
import { ApiError } from '../../api/client'
import { renderKind } from '../../helpers'
import type { ItemDetail } from '../../types/models'
import Button from '../shared/Button.vue'
import EmptyState from '../shared/EmptyState.vue'
import ReaderContent from './ReaderContent.vue'

export interface ReaderPaneEmits {
  openList: []
  selectFeed: [feedId: number]
}

const emit = defineEmits<ReaderPaneEmits>()

const { items, feeds: feedsTree } = useStore()
const nav = useViewNav()
const toast = useApiToast()

async function move(step: number) {
  const list = items.state.items
  if (!list.length) return
  const idx = list.findIndex((i) => i.id === nav.state.itemId)
  const next = Math.min(Math.max(idx === -1 ? 0 : idx + step, 0), list.length - 1)
  await items.openItem(list[next].id)
}

const currentListItem = computed(() =>
  nav.state.itemId == null ? null : items.state.items.find((i) => i.id === nav.state.itemId) ?? null,
)

interface ReaderData {
  detail: ItemDetail | null
  html: string
}

const EMPTY_READER: ReaderData = { detail: null, html: '' }

const loading = ref(false)
const error = ref<string | null>(null)

// Article content is derived from the selected id in a single request: the detail
// always, plus the server-rendered body when the feed uses server mode. It
// re-evaluates (cancelling any stale request) when the article or its feed's
// render mode changes, and `evaluating` drives the reader's loading overlay. A
// stale (404) id is dropped so it isn't re-fetched on every load.
const fetched = computedAsync<ReaderData>(
  async (onCancel) => {
    const itemId = nav.state.itemId
    // Snapshot feed render modes before awaiting: reactive reads after an await
    // are not tracked, so this keeps a feed's mode change re-running the fetch.
    const renderModes = new Map(feedsTree.state.feeds.map((f) => [f.id, f.render_mode]))
    error.value = null
    if (itemId == null) return EMPTY_READER
    let cancelled = false
    onCancel(() => {
      cancelled = true
    })
    try {
      const detail = await api.getItem(itemId)
      if (cancelled) return EMPTY_READER
      let html = ''
      if (renderKind(renderModes.get(detail.feed_id) ?? 0, detail.url) === 'server') {
        try {
          html = await api.renderItem(itemId)
        } catch (e) {
          if (!cancelled) {
            error.value = e instanceof Error ? e.message : String(e)
            toast.fromError(e)
          }
        }
      }
      return { detail, html: cancelled ? '' : html }
    } catch (e) {
      if (cancelled) return EMPTY_READER
      error.value = e instanceof Error ? e.message : String(e)
      toast.fromError(e)
      if (e instanceof ApiError && e.status === 404) nav.selectItem(null)
      return EMPTY_READER
    }
  },
  EMPTY_READER,
  { evaluating: loading },
)

// Previous/next article in the loaded list, for the mobile bottom nav bar.
const neighbors = computed(() => {
  const list = items.state.items
  const idx = list.findIndex((i) => i.id === nav.state.itemId)
  if (idx === -1) return { prev: null, next: null }
  return {
    prev: idx > 0 ? list[idx - 1] : null,
    next: idx < list.length - 1 ? list[idx + 1] : null,
  }
})

const feed = computed(() => {
  const d = currentListItem.value
  if (!d) return null
  return feedsTree.state.feeds.find((f) => f.id === d.feed_id) ?? null
})

// Clicking the feed title selects that feed (its article list becomes the scope).
function onFeedTitleClick() {
  const id = feed.value?.id
  if (id != null) emit('selectFeed', id)
}
const kind = computed(() => renderKind(feed.value?.render_mode ?? 0, currentListItem.value?.url ?? null))

const publishedLabel = computed(() => {
  const iso = currentListItem.value?.published_at
  if (!iso) return ''
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return ''
  return d.toLocaleDateString(undefined, { year: 'numeric', month: 'short', day: 'numeric' })
})

// Feed content is ONLY ever rendered DOMPurify-sanitized; everything else is Vue-escaped.
const contentHtml = computed(() => {
  const { detail, html } = fetched.value
  const raw = kind.value === 'server' ? html : detail?.content || detail?.summary || ''
  return raw ? DOMPurify.sanitize(raw) : ''
})

const contentMsg = computed(() => {
  if (error.value) return error.value
  return loading.value ? '' : 'No content for this article.'
})

async function toggleRead() {
  const id = nav.state.itemId
  if (id == null) return
  try {
    await items.toggleRead(id)
  } catch (e) {
    toast.fromError(e)
  }
}

async function toggleStar() {
  const id = nav.state.itemId
  if (id == null) return
  try {
    await items.toggleStar(id)
  } catch (e) {
    toast.fromError(e)
  }
}

async function onModeChange(e: Event) {
  const mode = Number((e.target as HTMLSelectElement).value)
  if (!currentListItem.value || !feed.value) return
  try {
    await feedsTree.setRenderMode(feed.value.id, mode)
  } catch (err) {
    toast.fromError(err)
  }
}

function openUrl() {
  const url = currentListItem.value?.url
  if (!url) return
  window.open(url, '_blank', 'noopener,noreferrer')
}
</script>

<template>
  <section class="reader">
    <header class="reader__head">
      <div class="reader__actions">
        <Button variant="ghost" size="sm" title="Back to list" class="reader__close" @click="emit('openList')">
          <span aria-hidden="true" class="i-lucide-chevron-left text-[16px]" />
        </Button>
        <div class="reader__head-nav">
          <Button variant="ghost" size="sm" title="Previous" @click="move(-1)">
            <span aria-hidden="true" class="i-lucide-chevron-left text-[16px]" />
          </Button>
          <Button variant="ghost" size="sm" title="Next" @click="move(1)">
            <span aria-hidden="true" class="i-lucide-chevron-right text-[16px]" />
          </Button>
        </div>
        <Button variant="ghost" size="sm" :title="currentListItem?.is_read ? 'Mark unread' : 'Mark read'" @click="toggleRead">
          <span aria-hidden="true" class="i-lucide-check text-[16px]" /> {{ currentListItem?.is_read ? 'Unread' : 'Read' }}
        </Button>
        <Button variant="ghost" size="sm" :title="currentListItem?.is_starred ? 'Unstar' : 'Star'" @click="toggleStar">
          <span aria-hidden="true" class="i-lucide-star text-[16px]" :class="{ starred: currentListItem?.is_starred }" />
        </Button>
        <select
          class="reader__mode"
          :value="feed?.render_mode ?? 0"
          :disabled="!currentListItem || !feed"
          title="Render mode"
          @change="onModeChange"
        >
          <option :value="0">Content</option>
          <option :value="2">Server</option>
        </select>
        <Button v-if="currentListItem?.url" variant="ghost" size="sm" title="Open in new window" @click="openUrl">
          <span aria-hidden="true" class="i-lucide-external-link text-[16px]" />
        </Button>
      </div>
    </header>
    <div class="reader__nav" role="navigation" aria-label="Article navigation">
      <button
        type="button"
        class="reader__nav-btn reader__nav-btn--prev"
        :disabled="!neighbors.prev"
        :title="neighbors.prev?.title ?? 'Previous article'"
        @click="move(-1)"
      >
        <span aria-hidden="true" class="i-lucide-chevron-left" />
        <span class="reader__nav-label">{{ neighbors.prev?.title ?? 'Previous' }}</span>
      </button>
      <button
        type="button"
        class="reader__nav-btn reader__nav-btn--next"
        :disabled="!neighbors.next"
        :title="neighbors.next?.title ?? 'Next article'"
        @click="move(1)"
      >
        <span class="reader__nav-label">{{ neighbors.next?.title ?? 'Next' }}</span>
        <span aria-hidden="true" class="i-lucide-chevron-right" />
      </button>
    </div>
    <ReaderContent
      v-if="contentHtml || loading"
      :html="contentHtml"
      :title="currentListItem?.title"
      :feed-title="currentListItem?.feed_title"
      :author="currentListItem?.author"
      :published-label="publishedLabel"
      :loading="loading"
      @feed-title-click="onFeedTitleClick"
    />
    <div v-else-if="contentMsg" class="reader__msg">{{ contentMsg }}</div>
    <EmptyState v-else message="Select an article to read it." icon="i-lucide-filter" />
  </section>
</template>

<style scoped>
.reader {
  flex: 1;
  min-width: 0;
  height: 100%;
  display: flex;
  flex-direction: column;
  background: var(--bg);
}
.reader__head {
  display: flex;
  justify-content: flex-end;
  padding: 10px 14px;
  border-bottom: 1px solid var(--border-subtle);
}
.reader__actions {
  display: flex;
  gap: 6px;
}
.reader__actions .starred {
  color: var(--star);
}
.reader__mode {
  height: 26px;
  padding: 0 6px;
  border: 1px solid var(--border-strong);
  border-radius: 6px;
  background: var(--surface);
  font-family: inherit;
  font-size: 12px;
  color: var(--text-secondary);
}
.reader__msg {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 100%;
  padding: 24px;
  font-size: 14px;
  color: var(--text-faint);
}
.reader__head-nav {
  display: inline-flex;
  gap: 6px;
}
.reader__close, .reader__nav {
  display: none;
}
@media (max-width: 768px) {
  .reader__close {
    display: inline-flex;
  }
  .reader__head-nav {
    display: none;
  }
  .reader__head {
    justify-content: flex-start;
  }
  .reader__nav {
    display: flex;
    position: fixed;
    left: 0;
    right: 0;
    bottom: 0;
    z-index: 10;
    background: var(--bg);
    border-top: 1px solid var(--border-subtle);
    padding-bottom: env(safe-area-inset-bottom, 0px);
  }
  .reader__nav-btn {
    flex: 1 1 0;
    min-width: 0;
    display: inline-flex;
    align-items: center;
    gap: 6px;
    padding: 12px 16px;
    border: 0;
    background: transparent;
    font: inherit;
    font-size: 13px;
    color: var(--text-muted);
    cursor: pointer;
    overflow: hidden;
  }
  .reader__nav-btn--next {
    border-left: 1px solid var(--border);
  }
  .reader__nav-btn:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }
  .reader__nav-label {
    flex: 1;
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
}
</style>
