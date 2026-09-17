<script setup lang="ts">
import { computed, ref } from 'vue'
import { useVirtualList, useIntersectionObserver } from '@vueuse/core'
import { injectItems, type Filter } from '../../providers/items'
import { injectSelection } from '../../providers/selection'
import type { Item } from '../../types/models'
import Button from '../shared/Button.vue'
import EmptyState from '../shared/EmptyState.vue'
import { useApiToast } from '../../api/useApiToast'

const items = injectItems()
const selection = injectSelection()
const toast = useApiToast()

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

useIntersectionObserver(loadTrigger, ([entry]) => {
  if (entry.isIntersecting) items.nextPage().catch(() => {})
})

function select(item: Item) {
  items.openItem(item.id).catch(() => {})
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

function toggleStar(item: Item) {
  items.toggleStar(item.id).catch((e) => toast.fromError(e))
}

async function markAllRead() {
  try {
    await items.markAllRead()
    toast.success('Marked all read')
  } catch (e) {
    toast.fromError(e)
  }
}

const hasMore = computed(() => items.state.page * items.state.limit < items.state.total)
</script>

<template>
  <section class="artlist">
    <header class="artlist__toolbar">
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
      <div class="toolbar__right">
        <form @submit.prevent="runSearch" class="search">
          <span aria-hidden="true" class="i-lucide-search text-[14px] search__icon" />
          <input v-model="search" class="search__input" placeholder="Search…" aria-label="Search" />
        </form>
        <Button variant="ghost" size="sm" title="Mark all read" @click="markAllRead">
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
              <span>{{ data.feed_title }}</span>
              <span v-if="data.author"> · {{ data.author }}</span>
            </div>
          </div>
        </div>
        <div v-if="hasMore" ref="loadTrigger" class="loadmore">Loading more…</div>
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
  border-right: 1px solid #e3e7ee;
}
.artlist__toolbar {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
  padding: 8px 10px;
  border-bottom: 1px solid #eef0f4;
}
.tabs {
  display: flex;
  gap: 4px;
  flex: 1 1 100%;
}
.tab {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 4px 8px;
  border: 1px solid transparent;
  border-radius: 6px;
  background: none;
  color: #5b6472;
  cursor: pointer;
  font-size: 12px;
}
.tab.active {
  background: #e3ecfd;
  color: #1f55c4;
}
.toolbar__right {
  display: flex;
  gap: 6px;
  align-items: center;
  flex: 1 1 100%;
}
.search {
  display: flex;
  flex: 1;
  align-items: center;
  gap: 4px;
  border: 1px solid #d8dde6;
  border-radius: 6px;
  padding: 3px 6px;
}
.search__icon {
  color: #9aa2b0;
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
  border-bottom: 1px solid #f2f4f8;
  cursor: pointer;
  box-sizing: border-box;
}
.row.read {
  opacity: 0.65;
}
.row:hover {
  background: #f6f8fb;
}
.row.active {
  background: #e3ecfd;
}
.row__star {
  display: inline-flex;
  color: #c3cad6;
}
.row__star.starred {
  color: #f5a623;
}
.row__body {
  flex: 1;
  min-width: 0;
}
.row__title {
  font-size: 13px;
  font-weight: 500;
  color: #22282f;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.row__meta {
  margin-top: 2px;
  font-size: 11.5px;
  color: #9aa2b0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.artlist__empty {
  flex: 1;
}
.loadmore {
  padding: 12px;
  text-align: center;
  font-size: 12px;
  color: #9aa2b0;
}
</style>
