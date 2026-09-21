<script setup lang="ts">
import { ref, watch } from 'vue'
import { injectFeedsTree } from '../../providers/feedsTree'
import { useApiToast } from '../../api/useApiToast'
import { api } from '../../api/endpoints'
import { useLoading } from '../../composables/useLoading'
import type { Feed } from '../../types/models'
import Modal from '../shared/Modal.vue'
import Button from '../shared/Button.vue'

export interface EditFeedModalProps {
  feed: Feed | null
}

const props = defineProps<EditFeedModalProps>()

const open = defineModel<boolean>({ default: false })

const feeds = injectFeedsTree()
const toast = useApiToast()

const title = ref('')
const feedUrl = ref('')
const siteUrl = ref('')
const description = ref('')
const folderId = ref<number | null>(null)

watch(
  () => props.feed,
  (feed) => {
    if (!feed) return
    title.value = feed.title
    feedUrl.value = feed.feed_url
    siteUrl.value = feed.site_url
    description.value = feed.description
    folderId.value = feed.folder_id
  },
  { immediate: true },
)

const save = useLoading(async () => {
  const feed = props.feed
  if (!feed || !title.value.trim() || !feedUrl.value.trim()) return
  try {
    await feeds.updateFeed(feed.id, {
      title: title.value.trim(),
      feed_url: feedUrl.value.trim(),
      site_url: siteUrl.value.trim(),
      description: description.value,
      folder_id: folderId.value,
    })
    toast.success('Feed updated')
    open.value = false
  } catch (e) {
    toast.fromError(e)
  }
})

// Detect fetches the feed to fill the metadata; it only changes the URL when
// discovery resolves a different one.
const detect = useLoading(async () => {
  const url = feedUrl.value.trim()
  if (!url) return
  try {
    const d = await api.discoverFeed(url)
    title.value = d.title
    if (d.feed_url) feedUrl.value = d.feed_url
    siteUrl.value = d.site_url
    description.value = d.description
    toast.success(`Detected feed ${d.title || d.feed_url}`)
  } catch (e) {
    toast.fromError(e)
  }
})
</script>

<template>
  <Modal v-model="open" title="Edit feed">
    <form class="edit" @submit.prevent="save">
      <label class="field">
        <span class="field__label">Name</span>
        <input v-model="title" class="field__input" type="text" aria-label="Feed name" />
      </label>
      <label class="field">
        <span class="field__label">Feed URL</span>
        <div class="field__row">
          <input v-model="feedUrl" class="field__input field__input--row" type="url" aria-label="Feed URL" />
          <Button type="button" size="sm" :loading="detect.isLoading" title="Detect feed metadata" @click="detect">
            <span aria-hidden="true" class="i-lucide-search text-[14px]" /> Detect
          </Button>
        </div>
      </label>
      <label class="field">
        <span class="field__label">Site URL</span>
        <input v-model="siteUrl" class="field__input" type="url" aria-label="Site URL" />
      </label>
      <label class="field">
        <span class="field__label">Description</span>
        <textarea v-model="description" class="field__input field__input--area" rows="3" aria-label="Description" />
      </label>
      <label class="field">
        <span class="field__label">Group</span>
        <select v-model="folderId" class="field__input" aria-label="Group">
          <option :value="null">Uncategorized</option>
          <option v-for="folder in feeds.state.folders" :key="folder.id" :value="folder.id">
            {{ folder.name }}
          </option>
        </select>
      </label>
      <div class="actions">
        <Button variant="ghost" type="button" @click="open = false">Cancel</Button>
        <Button variant="primary" type="submit" :loading="save.isLoading">Save</Button>
      </div>
    </form>
  </Modal>
</template>

<style scoped>
.edit {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.field {
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.field__label {
  font-size: 12px;
  color: var(--text-muted);
}
.field__input {
  padding: 6px 8px;
  border: 1px solid var(--border-strong);
  border-radius: 6px;
  font: inherit;
  font-size: 13px;
  resize: vertical;
}
.field__row {
  display: flex;
  gap: 6px;
}
.field__input--row {
  flex: 1;
}
.actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  margin-top: 4px;
}
</style>
