<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { onKeyStroke, useIntervalFn, useMediaQuery } from '@vueuse/core'
import { injectFeedsTree } from '../providers/feedsTree'
import { injectSelection } from '../providers/selection'
import { injectItems } from '../providers/items'
import { useApiToast } from '../api/useApiToast'
import { useItemNav } from '../composables/useItemNav'
import { useViewNav, type ViewState } from '../composables/useViewNav'
import FeedTree from '../components/business/FeedTree.vue'
import ArticleList from '../components/business/ArticleList.vue'
import ReaderPane from '../components/business/ReaderPane.vue'

const feeds = injectFeedsTree()
const selection = injectSelection()
const items = injectItems()
const toast = useApiToast()
const { move } = useItemNav(items, selection)
const nav = useViewNav(selection)
const isMobile = useMediaQuery('(max-width: 768px)')

// Which pane is full-screen on mobile comes from the URL, so the browser
// back/forward buttons move between screens. Desktop shows all three panes.
const screen = computed(() => nav.screen.value)

onMounted(async () => {
  try {
    await feeds.reload()
    await items.load()
    // Reopen the article into the reader: on desktop the reader is always
    // visible; on mobile only when the URL asks for it (a deep link). A restored
    // session on mobile stays on the list, matching the old behavior, without
    // fetching and marking the last article read. If the item no longer exists
    // (a shared link to a deleted/stale article) clear it so the dead id isn't
    // kept in the selection, localStorage, or URL and re-fetched on every load.
    const itemId = selection.state.itemId
    if (itemId != null && (!isMobile.value || nav.screen.value === 'reader')) {
      await items.openItem(itemId).catch(() => selection.selectItem(null))
    }
    // Resume a refresh-all that was already running when this page loaded, so
    // its progress bar shows and the tree resyncs when it finishes.
    await feeds.resumeRefresh().catch(() => {})
  } catch (e) {
    toast.fromError(e)
  }
})

// Scope changes and mobile screen transitions push a history entry (back works),
// so the selected feed/folder and the open screen both become shareable links.
async function openScopeList() {
  const feedId = selection.state.feedId
  const folderId = selection.state.folderId
  const scope: Partial<ViewState> = { feedId, folderId, itemId: null }
  if (isMobile.value && (feedId != null || folderId != null)) {
    // Selecting a scope on mobile moves through the feeds screen first: replace
    // the current feeds entry with the chosen feed/folder so it survives as the
    // previous entry (back from the list returns to feeds), then push the list.
    // The replace must settle before the push — an overlapping push would cancel
    // it and collapse the two steps into one.
    await nav.replace({ ...scope, view: 'feeds' })
    nav.push({ ...scope, view: 'list' })
    return
  }
  const patch: Partial<ViewState> = { ...scope }
  if (isMobile.value) patch.view = 'list'
  nav.push(patch)
}

function openFeeds() {
  nav.push({ view: 'feeds' })
}

function openReader() {
  // Desktop's reader is always visible, so opening an article only replaces the
  // item param (back returns to the previous scope, not through every article).
  // On mobile it's a screen transition (push) so back returns to the list.
  const patch: Partial<ViewState> = { itemId: selection.state.itemId }
  if (isMobile.value) {
    patch.view = 'reader'
    nav.push(patch)
  } else {
    nav.replace(patch)
  }
}

function closeReader() {
  nav.back({ view: 'list', itemId: selection.state.itemId })
}

function closeFeeds() {
  nav.back({ view: 'list' })
}

function toggleRead() {
  const id = selection.state.itemId
  if (id == null) return
  items.toggleRead(id).catch(() => {})
}

onKeyStroke('j', () => move(1))
onKeyStroke('k', () => move(-1))
onKeyStroke('m', toggleRead)
useIntervalFn(
  () => {
    feeds.reload().catch(() => {})
    items.load().catch(() => {})
  },
  60_000,
  { immediate: false },
)
</script>

<template>
  <main class="layout">
    <FeedTree
      class="pane"
      :class="{ 'pane--active': screen === 'feeds' }"
      @open-list="openScopeList"
      @close="closeFeeds"
    />
    <ArticleList
      class="pane"
      :class="{ 'pane--active': screen === 'list' }"
      @open-feeds="openFeeds"
      @open-reader="openReader"
    />
    <ReaderPane
      class="pane"
      :class="{ 'pane--active': screen === 'reader' }"
      @open-list="closeReader"
    />
  </main>
</template>

<style scoped>
.layout {
  display: flex;
  height: 100%;
  overflow: hidden;
}

@media (max-width: 768px) {
  :deep(.pane) { display: none; }
  :deep(.pane--active) { display: flex; flex-direction: column; }
}
</style>
