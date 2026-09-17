<script setup lang="ts">
import { onMounted } from 'vue'
import { onKeyStroke, useIntervalFn } from '@vueuse/core'
import { injectFeedsTree } from '../providers/feedsTree'
import { injectSelection } from '../providers/selection'
import { injectItems } from '../providers/items'
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
    // Reopen the article remembered from the last session. openItem is the
    // explicit select+open action, so it also handles the item highlighting.
    if (selection.state.itemId != null) await items.openItem(selection.state.itemId).catch(() => {})
  } catch (e) {
    toast.fromError(e)
  }
})

function move(step: number) {
  const list = items.state.items
  if (!list.length) return
  const idx = list.findIndex((i) => i.id === selection.state.itemId)
  const next = Math.min(Math.max(idx === -1 ? 0 : idx + step, 0), list.length - 1)
  items.openItem(list[next].id).catch(() => {})
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
