<script setup lang="ts">
import { onMounted, reactive } from 'vue'
import { onKeyStroke, useIntervalFn, useMediaQuery } from '@vueuse/core'
import { injectFeedsTree } from '../providers/feedsTree'
import { injectSelection } from '../providers/selection'
import { injectItems } from '../providers/items'
import { useApiToast } from '../api/useApiToast'
import { useItemNav } from '../composables/useItemNav'
import FeedTree from '../components/business/FeedTree.vue'
import ArticleList from '../components/business/ArticleList.vue'
import ReaderPane from '../components/business/ReaderPane.vue'

const feeds = injectFeedsTree()
const selection = injectSelection()
const items = injectItems()
const toast = useApiToast()
const { move } = useItemNav(items, selection)

type MobileScreen = 'feeds' | 'list' | 'reader'

const nav = reactive<{ screen: MobileScreen }>({ screen: 'list' })
const isMobile = useMediaQuery('(max-width: 768px)')
function goList() { nav.screen = 'list' }
function goFeeds() { nav.screen = 'feeds' }
function goReader() { nav.screen = 'reader' }

onMounted(async () => {
  try {
    if (isMobile.value) selection.clear()
    await feeds.reload()
    await items.load()
    // Reopen the article remembered from the last session. openItem is the
    // explicit select+open action, so it also handles the item highlighting.
    if (!isMobile.value && selection.state.itemId != null) await items.openItem(selection.state.itemId).catch(() => {})
    // Resume a refresh-all that was already running when this page loaded, so
    // its progress bar shows and the tree resyncs when it finishes.
    await feeds.resumeRefresh().catch(() => {})
  } catch (e) {
    toast.fromError(e)
  }
})

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
      :class="{ 'pane--active': nav.screen === 'feeds' }"
      @open-list="goList"
      @close="goList"
    />
    <ArticleList
      class="pane"
      :class="{ 'pane--active': nav.screen === 'list' }"
      @open-feeds="goFeeds"
      @open-reader="goReader"
    />
    <ReaderPane
      class="pane"
      :class="{ 'pane--active': nav.screen === 'reader' }"
      @open-list="goList"
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
