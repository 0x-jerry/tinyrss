<script setup lang="ts">
import { computed, ref } from 'vue'
import DOMPurify from 'dompurify'
import { computedAsync } from '@vueuse/core'
import { injectItems } from '../../providers/items'
import { injectSelection } from '../../providers/selection'
import { injectFeedsTree } from '../../providers/feedsTree'
import { useApiToast } from '../../api/useApiToast'
import { api } from '../../api/endpoints'
import { renderKind } from '../../renderMode'
import { useItemNav } from '../../composables/useItemNav'
import Button from '../shared/Button.vue'
import EmptyState from '../shared/EmptyState.vue'

export interface ReaderPaneEmits {
  openList: []
}

const emit = defineEmits<ReaderPaneEmits>()

const items = injectItems()
const selection = injectSelection()
const feedsTree = injectFeedsTree()
const toast = useApiToast()
const { move } = useItemNav(items, selection)

const detail = computed(() => items.state.selectedItem)

const listItem = computed(() =>
  selection.state.itemId == null ? null : items.state.items.find((i) => i.id === selection.state.itemId) ?? null,
)

const feed = computed(() => {
  const d = detail.value
  if (!d) return null
  return feedsTree.state.feeds.find((f) => f.id === d.feed_id) ?? null
})
const kind = computed(() => renderKind(feed.value?.render_mode ?? 0, detail.value?.url ?? null))

// Feeds render ONLY through DOMPurify before v-html; everything else is Vue-escaped.
const safeHtml = computed(() => {
  const d = detail.value
  if (!d) return ''
  return DOMPurify.sanitize(d.content || d.summary || '')
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

function toggleRead() {
  const id = selection.state.itemId
  if (id == null) return
  items.toggleRead(id).catch((e) => toast.fromError(e))
}

function toggleStar() {
  const id = selection.state.itemId
  if (id == null) return
  items.toggleStar(id).catch((e) => toast.fromError(e))
}

function onModeChange(e: Event) {
  const mode = Number((e.target as HTMLSelectElement).value)
  if (!detail.value || !feed.value) return
  feedsTree.setRenderMode(feed.value.id, mode).catch((err) => toast.fromError(err))
}

function openUrl() {
  const url = detail.value?.url
  if (!url) return
  window.open(url, '_blank', 'noopener,noreferrer')
}

function formatDate(iso: string): string {
  if (!iso) return ''
  const d = new Date(iso)
  return Number.isNaN(d.getTime()) ? '' : d.toLocaleString()
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
            <option :value="1">Iframe</option>
            <option :value="2">Server</option>
          </select>
          <Button v-if="detail.url" variant="ghost" size="sm" title="Open in new window" @click="openUrl">
            <span aria-hidden="true" class="i-lucide-external-link text-[16px]" />
          </Button>
        </div>
      </header>
      <div class="reader__nav" role="navigation" aria-label="Article navigation">
        <Button variant="ghost" size="sm" title="Previous article" @click="move(-1)">
          <span aria-hidden="true" class="i-lucide-chevron-left" /> Previous
        </Button>
        <Button variant="ghost" size="sm" title="Next article" @click="move(1)">
          Next <span aria-hidden="true" class="i-lucide-chevron-right" />
        </Button>
      </div>
      <div v-if="kind === 'server'" class="reader__frame-wrap">
        <iframe v-if="serverHtml" class="reader__frame" :srcdoc="serverHtml" sandbox="allow-scripts" title="Article" />
        <div v-else class="reader__frame-msg">{{ serverError || 'Loading…' }}</div>
      </div>
      <div v-else-if="kind === 'iframe'" class="reader__frame-wrap">
        <iframe class="reader__frame" :src="detail.url" :title="detail.title" />
      </div>
      <div v-else class="reader__scroll">
        <h1 class="reader__title">
          <a v-if="detail.url" :href="detail.url" target="_blank" rel="noopener noreferrer">{{ detail.title }}</a>
          <template v-else>{{ detail.title }}</template>
        </h1>
        <div class="reader__meta">
          <span v-if="detail.feed_title">{{ detail.feed_title }}</span>
          <span v-if="detail.author"> · {{ detail.author }}</span>
          <span v-if="detail.published_at"> · {{ formatDate(detail.published_at) }}</span>
        </div>
        <article v-if="safeHtml" class="reader__content" v-html="safeHtml"></article>
        <p v-else class="reader__summary">{{ detail.summary }}</p>
      </div>
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
  background: #fff;
}
.reader__head {
  display: flex;
  justify-content: flex-end;
  padding: 10px 14px;
  border-bottom: 1px solid #eef0f4;
}
.reader__actions {
  display: flex;
  gap: 6px;
}
.reader__actions .starred {
  color: #f5a623;
}
.reader__mode {
  height: 26px;
  padding: 0 6px;
  border: 1px solid #d8dde6;
  border-radius: 6px;
  background: #fff;
  font-family: inherit;
  font-size: 12px;
  color: #3c4450;
}
.reader__frame-wrap {
  flex: 1;
  min-height: 0;
}
.reader__frame {
  width: 100%;
  height: 100%;
  border: 0;
}
.reader__frame-msg {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 100%;
  padding: 24px;
  font-size: 14px;
  color: #8a93a3;
}
.reader__scroll {
  flex: 1;
  overflow-y: auto;
  padding: 20px 28px 48px;
}
.reader__title {
  margin: 0 0 8px;
  font-size: 22px;
  line-height: 1.25;
  color: #1c222a;
}
.reader__title a {
  color: inherit;
  text-decoration: none;
}
.reader__title a:hover {
  text-decoration: underline;
}
.reader__meta {
  margin-bottom: 18px;
  font-size: 13px;
  color: #8a93a3;
}
.reader__content {
  font-size: 15px;
  line-height: 1.65;
  color: #242b33;
  overflow-wrap: break-word;
}
.reader__content :deep(img) {
  max-width: 100%;
  height: auto;
}
.reader__summary {
  color: #5b6472;
  font-size: 14px;
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
    justify-content: space-between;
    gap: 8px;
    position: fixed;
    left: 0;
    right: 0;
    bottom: 0;
    z-index: 10;
    padding: 8px 12px calc(8px + env(safe-area-inset-bottom, 0px));
    background: #fff;
    border-top: 1px solid #eef0f4;
  }
  .reader__scroll {
    padding-bottom: 72px;
  }
}
</style>
