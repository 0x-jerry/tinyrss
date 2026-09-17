<script setup lang="ts">
import { computed } from 'vue'
import DOMPurify from 'dompurify'
import { injectItems } from '../../providers/items'
import { injectSelection } from '../../providers/selection'
import { useApiToast } from '../../api/useApiToast'
import Button from '../shared/Button.vue'
import EmptyState from '../shared/EmptyState.vue'

const items = injectItems()
const selection = injectSelection()
const toast = useApiToast()

const detail = computed(() => items.state.selectedItem)

const listItem = computed(() =>
  selection.state.itemId == null ? null : items.state.items.find((i) => i.id === selection.state.itemId) ?? null,
)

// Feeds render ONLY through DOMPurify before v-html; everything else is Vue-escaped.
const safeHtml = computed(() => {
  const d = detail.value
  if (!d) return ''
  return DOMPurify.sanitize(d.content || d.summary || '')
})

async function toggleRead() {
  const id = selection.state.itemId
  if (id == null) return
  try {
    if (listItem.value?.is_read) await items.markUnread(id)
    else await items.markRead(id)
  } catch (e) {
    toast.fromError(e)
  }
}

async function toggleStar() {
  const id = selection.state.itemId
  if (id == null) return
  try {
    if (listItem.value?.is_starred) await items.unstar(id)
    else await items.star(id)
  } catch (e) {
    toast.fromError(e)
  }
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
          <Button variant="ghost" size="sm" :title="listItem?.is_read ? 'Mark unread' : 'Mark read'" @click="toggleRead">
            <span aria-hidden="true" class="i-lucide-check text-[16px]" /> {{ listItem?.is_read ? 'Unread' : 'Read' }}
          </Button>
          <Button variant="ghost" size="sm" :title="listItem?.is_starred ? 'Unstar' : 'Star'" @click="toggleStar">
            <span aria-hidden="true" class="i-lucide-star text-[16px]" />
          </Button>
          <Button v-if="detail.url" variant="ghost" size="sm" title="Open original">
            <a class="reader__link" :href="detail.url" target="_blank" rel="noopener noreferrer">
              <span aria-hidden="true" class="i-lucide-external-link text-[16px]" /> Open
            </a>
          </Button>
        </div>
      </header>
      <div class="reader__scroll">
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
.reader__link {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  color: inherit;
  text-decoration: none;
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
</style>
