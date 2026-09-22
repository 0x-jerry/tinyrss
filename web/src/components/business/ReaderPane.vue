<script setup lang="ts">
import { computed, ref } from 'vue'
import DOMPurify from 'dompurify'
import { computedAsync } from '@vueuse/core'
import { useStore } from '../../store'
import { useViewNav } from '../../composables/useViewNav'
import { useApiToast } from '../../api/useApiToast'
import { api } from '../../api/endpoints'
import { renderKind } from '../../helpers'
import { useItemNav } from '../../composables/useItemNav'
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
const { move } = useItemNav(items, nav)

const detail = computed(() => items.state.selectedItem)

const listItem = computed(() =>
  nav.state.itemId == null ? null : items.state.items.find((i) => i.id === nav.state.itemId) ?? null,
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
  const d = detail.value
  if (!d) return null
  return feedsTree.state.feeds.find((f) => f.id === d.feed_id) ?? null
})

// Clicking the feed title selects that feed (its article list becomes the scope).
function onFeedTitleClick() {
  const id = feed.value?.id
  if (id != null) emit('selectFeed', id)
}
const kind = computed(() => renderKind(feed.value?.render_mode ?? 0, detail.value?.url ?? null))

// Feed content is ONLY ever rendered DOMPurify-sanitized; everything else is Vue-escaped.
const safeHtml = computed(() => {
  const d = detail.value
  if (!d) return ''
  return DOMPurify.sanitize(d.content || d.summary || '')
})

const publishedLabel = computed(() => {
  const iso = detail.value?.published_at
  if (!iso) return ''
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return ''
  return d.toLocaleDateString(undefined, { year: 'numeric', month: 'short', day: 'numeric' })
})

// Server-mode HTML is a derived value: it re-evaluates (cancelling any stale
// request) whenever the article or its render kind changes.
const serverLoading = ref(false)
const serverError = ref<string | null>(null)
const serverHtml = computedAsync(
  async (onCancel) => {
    const url = kind.value === 'server' ? detail.value?.url : null
    serverError.value = null
    if (!url) return ''
    let cancelled = false
    onCancel(() => {
      cancelled = true
    })
    try {
      const html = await api.renderUrl(url)
      return cancelled ? '' : html
    } catch (e) {
      if (cancelled) return ''
      serverError.value = e instanceof Error ? e.message : String(e)
      return ''
    }
  },
  '',
  { evaluating: serverLoading },
)

// Server HTML was previously isolated inside a sandboxed iframe; rendered inline
// it must be DOMPurify-sanitized too (which also strips the backend's doc wrapper).
const safeServerHtml = computed(() => (serverHtml.value ? DOMPurify.sanitize(serverHtml.value) : ''))

const contentHtml = computed(() => (kind.value === 'server' ? safeServerHtml.value : safeHtml.value))

const loading = computed(() => kind.value === 'server' && serverLoading.value)

const contentMsg = computed(() => {
  if (kind.value === 'server') {
    if (serverError.value) return serverError.value
    if (loading.value) return ''
    return 'No content for this article.'
  }
  return 'No content for this article.'
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
  if (!detail.value || !feed.value) return
  try {
    await feedsTree.setRenderMode(feed.value.id, mode)
  } catch (err) {
    toast.fromError(err)
  }
}

function openUrl() {
  const url = detail.value?.url
  if (!url) return
  window.open(url, '_blank', 'noopener,noreferrer')
}
</script>

<template>
  <section class="reader">
    <template v-if="detail">
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
          <Button variant="ghost" size="sm" :title="listItem?.is_read ? 'Mark unread' : 'Mark read'" @click="toggleRead">
            <span aria-hidden="true" class="i-lucide-check text-[16px]" /> {{ listItem?.is_read ? 'Unread' : 'Read' }}
          </Button>
          <Button variant="ghost" size="sm" :title="listItem?.is_starred ? 'Unstar' : 'Star'" @click="toggleStar">
            <span aria-hidden="true" class="i-lucide-star text-[16px]" :class="{ starred: listItem?.is_starred }" />
          </Button>
          <select
            class="reader__mode"
            :value="feed?.render_mode ?? 0"
            :disabled="!detail || !feed"
            title="Render mode"
            @change="onModeChange"
          >
            <option :value="0">Content</option>
            <option :value="2">Server</option>
          </select>
          <Button v-if="detail.url" variant="ghost" size="sm" title="Open in new window" @click="openUrl">
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
        :title="detail.title"
        :feed-title="detail.feed_title"
        :author="detail.author"
        :published-label="publishedLabel"
        :loading="loading"
        @feed-title-click="onFeedTitleClick"
      />
      <div v-else-if="contentMsg" class="reader__msg">{{ contentMsg }}</div>
    </template>
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
