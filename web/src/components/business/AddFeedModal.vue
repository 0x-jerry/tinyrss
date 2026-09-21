<script setup lang="ts">
import { ref, watch } from 'vue'
import { injectFeedsTree } from '../../providers/feedsTree'
import { useApiToast } from '../../api/useApiToast'
import { api } from '../../api/endpoints'
import { useLoading } from '../../composables/useLoading'
import Modal from '../shared/Modal.vue'
import Button from '../shared/Button.vue'

export interface AddFeedModalProps {
  // A ?add_feed= URL prefills the field; the user still clicks Detect to
  // autodiscover a homepage.
  presetUrl?: string
}

const props = defineProps<AddFeedModalProps>()

const feeds = injectFeedsTree()
const toast = useApiToast()

const open = defineModel<boolean>({ default: false })

const title = ref('')
const feedUrl = ref('')
const siteUrl = ref('')
const description = ref('')
const folderId = ref<number | null>(null)

watch(
  () => props.presetUrl,
  (v) => {
    if (v) feedUrl.value = v
  },
)

// Detect fetches the feed to fill the form; add/update never fetch on their own.
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

const addFeed = useLoading(async () => {
  const url = feedUrl.value.trim()
  if (!url) return
  try {
    await feeds.addFeed({
      feed_url: url,
      title: title.value.trim(),
      site_url: siteUrl.value.trim(),
      description: description.value,
      folder_id: folderId.value,
    })
    toast.success('Feed added')
    open.value = false
    title.value = ''
    feedUrl.value = ''
    siteUrl.value = ''
    description.value = ''
    folderId.value = null
  } catch (e) {
    toast.fromError(e)
  }
})
</script>

<template>
  <Modal v-model="open" title="Add feed">
    <form class="add" @submit.prevent="addFeed">
      <label class="field">
        <span class="field__label">Name</span>
        <input v-model="title" class="field__input" type="text" aria-label="Feed name" />
      </label>
      <label class="field">
        <span class="field__label">Feed URL</span>
        <div class="field__row">
          <input v-model="feedUrl" class="field__input field__input--row" type="url" placeholder="https://example.com (site or feed URL)" aria-label="Feed URL" />
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
        <Button variant="primary" type="submit" :loading="addFeed.isLoading">Add feed</Button>
      </div>
    </form>
  </Modal>
</template>

<style scoped>
.add {
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
