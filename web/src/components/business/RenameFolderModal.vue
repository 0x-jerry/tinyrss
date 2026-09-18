<script setup lang="ts">
import { ref, watch } from 'vue'
import { injectFeedsTree } from '../../providers/feedsTree'
import { useApiToast } from '../../api/useApiToast'
import Modal from '../shared/Modal.vue'
import Button from '../shared/Button.vue'

export interface RenameFolderModalProps {
  folder: { id: number; name: string } | null
}

const props = defineProps<RenameFolderModalProps>()

const open = defineModel<boolean>({ default: false })

const feeds = injectFeedsTree()
const toast = useApiToast()

const name = ref('')

watch(
  () => props.folder,
  (folder) => {
    if (folder) name.value = folder.name
  },
  { immediate: true },
)

async function save() {
  const folder = props.folder
  if (!folder) return
  const trimmed = name.value.trim()
  if (!trimmed) return
  if (trimmed === folder.name) {
    open.value = false
    return
  }
  try {
    await feeds.renameFolder(folder.id, trimmed)
    toast.success('Folder renamed')
    open.value = false
  } catch (e) {
    toast.fromError(e)
  }
}
</script>

<template>
  <Modal v-model="open" title="Rename folder">
    <form class="rename" @submit.prevent="save">
      <input v-model="name" class="rename__input" type="text" aria-label="Folder name" />
      <div class="actions">
        <Button variant="ghost" type="button" @click="open = false">Cancel</Button>
        <Button variant="primary" type="submit">Save</Button>
      </div>
    </form>
  </Modal>
</template>

<style scoped>
.rename {
  display: flex;
  flex-direction: column;
  gap: 14px;
}
.rename__input {
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
}
</style>
