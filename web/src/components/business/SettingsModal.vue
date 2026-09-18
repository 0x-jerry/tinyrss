<script setup lang="ts">
import { ref } from 'vue'
import { injectFeedsTree } from '../../providers/feedsTree'
import { useApiToast } from '../../api/useApiToast'
import Modal from '../shared/Modal.vue'
import Button from '../shared/Button.vue'

const feeds = injectFeedsTree()
const toast = useApiToast()

const open = defineModel<boolean>({ default: false })

const fileInput = ref<HTMLInputElement | null>(null)
const importing = ref(false)

function openImport() {
  fileInput.value?.click()
}

async function onImportFile(event: Event) {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  if (!file) return
  importFile(file).finally(() => {
    input.value = ''
  })
}

async function importFile(file: File) {
  if (importing.value) return
  importing.value = true
  const form = new FormData()
  form.append('file', file)
  try {
    const added = await feeds.importOpmlForm(form)
    toast.success(`Added ${added} feed${added === 1 ? '' : 's'}`)
  } catch (e) {
    toast.fromError(e)
  } finally {
    importing.value = false
  }
}

function exportOpml() {
  feeds
    .exportOpmlText()
    .then((text) => downloadText('tinyrss-subscriptions.opml.xml', text, 'application/xml'))
    .catch((e) => toast.fromError(e))
}

function downloadText(filename: string, text: string, mime: string) {
  const url = URL.createObjectURL(new Blob([text], { type: mime }))
  const a = document.createElement('a')
  a.href = url
  a.download = filename
  a.click()
  URL.revokeObjectURL(url)
}
</script>

<template>
  <Modal v-model="open" title="Settings">
    <input
      ref="fileInput"
      class="import-input"
      type="file"
      accept=".opml,.xml,application/xml,text/xml"
      :disabled="importing"
      @change="onImportFile"
    />
    <div class="actions">
      <Button variant="ghost" :disabled="importing" @click="openImport">
        <span aria-hidden="true" class="i-lucide-upload text-[16px]" /> Import OPML…
      </Button>
      <Button variant="ghost" @click="exportOpml">
        <span aria-hidden="true" class="i-lucide-download text-[16px]" /> Export OPML
      </Button>
    </div>
  </Modal>
</template>

<style scoped>
.actions {
  display: flex;
  flex-direction: column;
  align-items: stretch;
  gap: 8px;
}
.import-input {
  display: none;
}
</style>
