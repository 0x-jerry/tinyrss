<script setup lang="ts">
import { ref, watch } from 'vue'
import { injectFeedsTree } from '../../providers/feedsTree'
import { useApiToast } from '../../api/useApiToast'
import { api } from '../../api/endpoints'
import { ApiError } from '../../api/client'
import type { FetchLog } from '../../types/models'
import Modal from '../shared/Modal.vue'
import Button from '../shared/Button.vue'

const feeds = injectFeedsTree()
const toast = useApiToast()

const open = defineModel<boolean>({ default: false })

const fileInput = ref<HTMLInputElement | null>(null)
const importing = ref(false)

const logs = ref<FetchLog[]>([])
const logsLoading = ref(false)
const logsError = ref('')

const cleanupDays = ref(30)
const settingsLoading = ref(false)

watch(open, (isOpen) => {
  if (!isOpen) return
  loadSettings()
  loadLogs()
})

async function loadSettings() {
  settingsLoading.value = true
  try {
    const s = await api.getSettings()
    cleanupDays.value = s.fetch_log_cleanup_days
  } catch (e) {
    toast.fromError(e)
  } finally {
    settingsLoading.value = false
  }
}

async function saveCleanupDays() {
  const days = Math.max(0, Math.floor(Number(cleanupDays.value) || 0))
  cleanupDays.value = days
  try {
    await api.updateSettings({ fetch_log_cleanup_days: days })
    toast.success(days > 0 ? `Auto-clean set to ${days} day${days === 1 ? '' : 's'}` : 'Auto-clean off')
  } catch (e) {
    toast.fromError(e)
  }
}

async function loadLogs() {
  logsLoading.value = true
  logsError.value = ''
  try {
    logs.value = await api.fetchLogs(100)
  } catch (e) {
    logsError.value = e instanceof ApiError ? e.message : 'Could not load fetch logs'
  } finally {
    logsLoading.value = false
  }
}

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

    <section class="panel">
      <h4 class="panel__title">Fetch logs</h4>
      <label class="panel__field">
        <span class="panel__label">Auto-clean fetch logs older than</span>
        <input
          v-model.number="cleanupDays"
          class="panel__input"
          type="number"
          min="0"
          step="1"
          :disabled="settingsLoading"
          aria-label="Auto-clean fetch logs older than (days)"
          @change="saveCleanupDays"
        />
        <span class="panel__hint">days (0 = off)</span>
      </label>

      <div v-if="logsLoading" class="panel__muted">Loading…</div>
      <div v-else-if="logsError" class="panel__muted panel__muted--error">{{ logsError }}</div>
      <div v-else-if="logs.length === 0" class="panel__muted">No fetch activity yet.</div>
      <ul v-else class="logs">
        <li v-for="log in logs" :key="log.id" class="log">
          <span class="log__dot" :class="{ 'log__dot--ok': log.success, 'log__dot--err': !log.success }" />
          <div class="log__body">
            <div class="log__head">
              <span class="log__feed">{{ log.feed_title }}</span>
              <span class="log__time">{{ log.fetched_at }}</span>
            </div>
            <div v-if="!log.success && log.error" class="log__error">{{ log.error }}</div>
          </div>
        </li>
      </ul>
    </section>
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
.panel {
  margin-top: 16px;
  border-top: 1px solid #eef0f4;
  padding-top: 12px;
}
.panel__title {
  margin: 0 0 8px;
  font-size: 13px;
  color: #3c4450;
}
.panel__field {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-bottom: 10px;
  font-size: 12.5px;
  color: #3c4450;
}
.panel__label {
  white-space: nowrap;
}
.panel__input {
  width: 56px;
  padding: 4px 6px;
  border: 1px solid #d8dde6;
  border-radius: 6px;
  font: inherit;
  font-size: 12.5px;
}
.panel__hint {
  color: #7b8491;
  white-space: nowrap;
}
.panel__muted {
  font-size: 12.5px;
  color: #7b8491;
}
.panel__muted--error {
  color: #c0392b;
}
.logs {
  max-height: 320px;
  overflow-y: auto;
  list-style: none;
  margin: 0;
  padding: 0 2px 0 0;
}
.log {
  display: flex;
  gap: 8px;
  padding: 6px 0;
  border-bottom: 1px solid #eef0f4;
}
.log:last-child {
  border-bottom: 0;
}
.log__dot {
  flex: none;
  width: 8px;
  height: 8px;
  margin-top: 4px;
  border-radius: 50%;
  background: #e5484d;
}
.log__dot--ok {
  background: #2ea043;
}
.log__dot--err {
  background: #e5484d;
}
.log__body {
  min-width: 0;
}
.log__head {
  display: flex;
  align-items: baseline;
  gap: 8px;
  font-size: 12.5px;
}
.log__feed {
  font-weight: 600;
  color: #3c4450;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.log__time {
  flex: none;
  color: #7b8491;
  font-size: 11.5px;
}
.log__error {
  margin-top: 2px;
  font-size: 12px;
  color: #c0392b;
  word-break: break-word;
}
</style>
