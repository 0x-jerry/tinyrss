<script setup lang="ts">
import { ref } from 'vue'
import { injectFeedsTree } from '../../providers/feedsTree'
import { useApiToast } from '../../api/useApiToast'
import { useLoading } from '../../composables/useLoading'
import Modal from '../shared/Modal.vue'
import Button from '../shared/Button.vue'

const feeds = injectFeedsTree()
const toast = useApiToast()

const open = defineModel<boolean>({ default: false })

const url = ref('')
const folderId = ref<number | null>(null)

const addFeed = useLoading(async () => {
  const feedUrl = url.value.trim()
  if (!feedUrl) return
  try {
    await feeds.addFeed(feedUrl, folderId.value)
    toast.success('Feed added')
    open.value = false
    url.value = ''
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
        <span class="field__label">URL</span>
        <input v-model="url" class="field__input" type="url" placeholder="https://example.com/feed.xml" aria-label="Feed URL" />
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
}
.actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  margin-top: 4px;
}
</style>
