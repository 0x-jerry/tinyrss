<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import { useVirtualList, useIntersectionObserver } from '@vueuse/core'
import { injectItems, type Filter } from '../../providers/items'
import { injectFeedsTree } from '../../providers/feedsTree'
import { injectSelection } from '../../providers/selection'
import type { Item } from '../../types/models'
import Button from '../shared/Button.vue'
import EmptyState from '../shared/EmptyState.vue'
import { useApiToast } from '../../api/useApiToast'
import { useLoading } from '../../composables/useLoading'
import { nearestScrollTop } from '../../utils/scrollIntoViewNearest'

const items = injectItems()
const feeds = injectFeedsTree()
const selection = injectSelection()
const toast = useApiToast()

export interface ArticleListProps {
  /** Whether this pane is the active full-screen view (mobile screen == list). */
  active?: boolean
}

export interface ArticleListEmits {
  openFeeds: []
  openReader: []
}

const props = defineProps<ArticleListProps>()
const emit = defineEmits<ArticleListEmits>()

const ITEM_HEIGHT = 64
const filterOptions: { value: Filter; label: string; icon: string }[] = [
  { value: 'all', label: 'All', icon: 'i-lucide-list' },
  { value: 'unread', label: 'Unread', icon: 'i-lucide-mail-open' },
  { value: 'starred', label: 'Starred', icon: 'i-lucide-star' },
]

const search = ref('')
const loadTrigger = ref<HTMLElement | null>(null)

const source = computed<Item[]>(() => items.state.items)
const { list: rows, containerProps, wrapperProps } = useVirtualList(source, { itemHeight: ITEM_HEIGHT })

// When the list pane becomes the active mobile screen again (returning from the
// reader/feeds), keep the currently selected article in view. The active row may
// be off-screen after paging in the reader. Rows are fixed-height, so visibility
// is computed numerically without needing the virtual list to have rendered it;
// only scroll when the row is out of view, aligning to the near edge.
async function scrollToActive() {
  await nextTick()
  const id = selection.state.itemId
  if (id == null) return
  const idx = source.value.findIndex((i) => i.id === id)
  if (idx === -1) return
  const viewport = containerProps.ref.value
  if (!viewport) return
  const target = nearestScrollTop(idx * ITEM_HEIGHT, ITEM_HEIGHT, viewport.scrollTop, viewport.clientHeight)
  if (target !== viewport.scrollTop) viewport.scrollTop = target
}
// Scroll when the list pane becomes the active mobile screen (returning from the
// reader/feeds) or when the selected article changes while the list is shown
// (reader nav, j/k). Gating on props.active skips mobile reader-side item changes,
// where the list pane is hidden.
watch([() => props.active, () => selection.state.itemId], () => {
  if (props.active) scrollToActive()
})

useIntersectionObserver(loadTrigger, async ([entry]) => {
  if (entry.isIntersecting) await items.nextPage()
})

async function select(item: Item) {
  await items.openItem(item.id)
  emit('openReader')
}

async function setFilter(f: Filter) {
  try {
    await items.setFilter(f)
  } catch (e) {
    toast.fromError(e)
  }
}

async function runSearch() {
  try {
    await items.setSearch(search.value.trim())
  } catch (e) {
    toast.fromError(e)
  }
}

async function toggleStar(item: Item) {
  try {
    await items.toggleStar(item.id)
  } catch (e) {
    toast.fromError(e)
  }
}

const markAllRead = useLoading(async () => {
  try {
    await items.markAllRead()
    toast.success('Marked all read')
  } catch (e) {
    toast.fromError(e)
  }
})

// Refresh button is shown for a specific feed or the "all articles" scope.
// Folder scope has no dedicated refresh, so it is excluded.
const showRefresh = computed(
  () => selection.state.feedId != null || (selection.state.feedId == null && selection.state.folderId == null),
)
const refreshTitle = computed(() => (selection.state.feedId != null ? 'Refresh feed' : 'Refresh all articles'))

const refresh = useLoading(async () => {
  const feedId = selection.state.feedId
  if (feedId != null) {
    try {
      await feeds.refreshFeed(feedId)
      await items.load()
      toast.success('Feed refreshed')
    } catch (e) {
      toast.fromError(e)
    }
    return
  }
  // All articles scope: refresh every feed. Guard against re-entry while a
  // background refresh-all is already running.
  if (feeds.state.refresh.running) return
  try {
    await feeds.refreshAll()
    await items.load()
    const { failed } = feeds.state.refresh
    toast.success(failed > 0 ? `Feeds refreshed (${failed} failed)` : 'Feeds refreshed')
  } catch (e) {
    toast.fromError(e)
  }
})

// Sync the button's busy state with the global refresh-all progress so the
// spinner reflects a refresh from any source (this button, FeedTree, or a
// resumed server job) — but only in the all-articles scope. When a specific
// feed is selected, the button only reflects its own local refresh and stays
// independent of any running refresh-all.
const refreshBusy = computed(() => refresh.isLoading || (selection.state.feedId == null && feeds.state.refresh.running))

const hasMore = computed(() => items.state.page * items.state.limit < items.state.total)

// Compact publish date: "Sep 21" for the current year, "Dec 3, 2024" otherwise.
function dateLabel(iso: string): string {
  if (!iso) return ''
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return ''
  const sameYear = d.getFullYear() === new Date().getFullYear()
  return d.toLocaleDateString(undefined, sameYear ? { month: 'short', day: 'numeric' } : { year: 'numeric', month: 'short', day: 'numeric' })
}
</script>

<template>
  <section class="artlist">
    <header class="artlist__toolbar">
      <div class="toolbar__top">
        <Button
          variant="ghost"
          size="sm"
          title="Menu"
          class="artlist__menu"
          @click="emit('openFeeds')"
        >
          <span aria-hidden="true" class="i-lucide-menu text-[16px]" />
        </Button>
        <div class="tabs">
          <button
            v-for="opt in filterOptions"
            :key="opt.value"
            class="tab"
            :class="{ active: items.state.filter === opt.value }"
            @click="setFilter(opt.value)"
          >
            <span aria-hidden="true" :class="opt.icon" />
            {{ opt.label }}
          </button>
        </div>
        <Button
          v-if="showRefresh"
          variant="ghost"
          size="sm"
          :disabled="refreshBusy"
          :title="refreshTitle"
          class="toolbar__refresh"
          @click="refresh"
        >
          <span aria-hidden="true" class="i-lucide-refresh-cw text-[16px]" :class="{ spin: refreshBusy }" />
        </Button>
      </div>
      <div class="toolbar__actions">
        <form @submit.prevent="runSearch" class="search">
          <span aria-hidden="true" class="i-lucide-search text-[14px] search__icon" />
          <input v-model="search" class="search__input" placeholder="Search…" aria-label="Search" />
        </form>
        <Button variant="ghost" size="sm" title="Mark all read" :loading="markAllRead.isLoading" @click="markAllRead">
          <span aria-hidden="true" class="i-lucide-check text-[16px]" />
        </Button>
      </div>
    </header>

    <div v-if="!items.state.items.length && !items.state.loading" class="artlist__empty">
      <EmptyState message="No articles here yet. Add a feed or adjust filters." />
    </div>

    <div v-else v-bind="containerProps" class="artlist__viewport">
      <div v-bind="wrapperProps">
        <div
          v-for="{ data } in rows"
          :key="data.id"
          class="row"
          :class="{ active: selection.state.itemId === data.id, read: data.is_read }"
          :style="{ height: ITEM_HEIGHT + 'px' }"
          @click="select(data)"
        >
          <span class="row__star" :class="{ starred: data.is_starred }" @click.stop="toggleStar(data)">
            <span aria-hidden="true" class="i-lucide-star text-[14px]" />
          </span>
          <div class="row__body">
            <div class="row__title">{{ data.is_read ? '' : '● ' }}{{ data.title }}</div>
            <div class="row__meta">
              <span class="row__meta-text">
                {{ data.feed_title }}<template v-if="data.author"> · {{ data.author }}</template>
              </span>
              <span class="row__date"><time :datetime="data.published_at">{{ dateLabel(data.published_at) }}</time></span>
            </div>
          </div>
        </div>
        <div v-if="hasMore" ref="loadTrigger" class="loadmore">Loading more…</div>
        <div v-else-if="items.state.items.length && !items.state.loading" class="loadmore">No more articles</div>
      </div>
    </div>
  </section>
</template>

<style scoped>
.artlist {
  display: flex;
  flex-direction: column;
  width: 360px;
  min-width: 300px;
  height: 100%;
  border-right: 1px solid var(--border);
}
.artlist__toolbar {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 8px 10px;
  border-bottom: 1px solid var(--border-subtle);
}
.toolbar__top {
  height: 26px;
  display: flex;
  align-items: center;
  gap: 8px;
}
.artlist__menu {
  display: none;
}
@media (max-width: 768px) {
  .artlist__menu {
    display: inline-flex;
  }
  .artlist {
    width: 100%;
    min-width: 0;
    flex: 1 1 auto;
    border-right: 0;
  }
}
.tabs {
  display: flex;
  gap: 4px;
  flex: 1 1 auto;
  min-width: 0;
}
.tab {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 4px 8px;
  border: 1px solid transparent;
  border-radius: 6px;
  background: none;
  color: var(--text-muted);
  cursor: pointer;
  font-size: 12px;
}
.tab.active {
  background: var(--surface-active);
  color: var(--accent-soft-text);
}
.toolbar__refresh {
  flex: 0 0 auto;
}
.toolbar__actions {
  display: flex;
  gap: 6px;
  align-items: center;
}
.search {
  display: flex;
  flex: 1;
  align-items: center;
  gap: 4px;
  border: 1px solid var(--border-strong);
  border-radius: 6px;
  padding: 3px 6px;
}
.search__icon {
  color: var(--text-faint);
}
.search__input {
  flex: 1;
  min-width: 0;
  border: 0;
  outline: none;
  font: inherit;
  font-size: 12px;
}
.artlist__viewport {
  flex: 1;
  overflow-y: auto;
}
.row {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  border-bottom: 1px solid var(--border-subtle);
  cursor: pointer;
  box-sizing: border-box;
}
.row.read {
  opacity: 0.65;
}
.row:hover {
  background: var(--bg-hover);
}
.row.active {
  background: var(--surface-active);
}
.row__star {
  display: inline-flex;
  color: var(--text-faint);
}
.row__star.starred {
  color: var(--star);
}
.row__body {
  flex: 1;
  min-width: 0;
}
.row__title {
  font-size: 13px;
  font-weight: 500;
  color: var(--text);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.row__meta {
  margin-top: 2px;
  display: flex;
  align-items: baseline;
  gap: 2px;
  font-size: 11.5px;
  color: var(--text-faint);
}
.row__meta-text {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.row__date {
  flex: none;
  white-space: nowrap;
}
.artlist__empty {
  flex: 1;
}
.loadmore {
  padding: 12px;
  text-align: center;
  font-size: 12px;
  color: var(--text-faint);
}
.spin {
  animation: refresh-spin 1s linear infinite;
}
@keyframes refresh-spin {
  to {
    transform: rotate(360deg);
  }
}
</style>
