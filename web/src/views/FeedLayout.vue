<script setup lang="ts">
import { onMounted, watch } from 'vue'
import { injectFeedsTree } from '../providers/feedsTree'
import { injectSelection } from '../providers/selection'
import { injectItems } from '../providers/items'
import { useAutoRefresh } from '../composables/useAutoRefresh'
import { useKeyboard } from '../composables/useKeyboard'
import { useApiToast } from '../api/useApiToast'
import FeedTree from '../components/business/FeedTree.vue'
import ArticleList from '../components/business/ArticleList.vue'
import ReaderPane from '../components/business/ReaderPane.vue'

const feeds = injectFeedsTree()
const selection = injectSelection()
const items = injectItems()
const toast = useApiToast()

onMounted(async () => {
  try {
    await feeds.reload()
    await items.load()
  } catch (e) {
    toast.fromError(e)
  }
})

// Load the list whenever the feed/folder selection changes.
watch(
  () => [selection.state.feedId, selection.state.folderId],
  () => {
    items.load().catch(() => {})
  },
)

// Opening a selected item loads its detail and marks it read (optimistic).
watch(
  () => selection.state.itemId,
  (id) => {
    if (id != null) items.openItem(id).catch(() => {})
  },
)

function move(step: number) {
  const list = items.state.items
  if (!list.length) return
  const idx = list.findIndex((i) => i.id === selection.state.itemId)
  const next = Math.min(Math.max(idx === -1 ? 0 : idx + step, 0), list.length - 1)
  selection.selectItem(list[next].id)
}

function toggleRead() {
  const id = selection.state.itemId
  if (id == null) return
  const item = items.state.items.find((i) => i.id === id)
  if (item?.is_read) items.markUnread(id).catch(() => {})
  else items.markRead(id).catch(() => {})
}

useKeyboard({ next: () => move(1), prev: () => move(-1), toggleRead })
useAutoRefresh(feeds, items)
</script>

<template>
  <main class="layout">
    <FeedTree />
    <ArticleList />
    <ReaderPane />
  </main>
</template>

<style scoped>
.layout {
  display: flex;
  height: 100%;
  overflow: hidden;
}
</style>
