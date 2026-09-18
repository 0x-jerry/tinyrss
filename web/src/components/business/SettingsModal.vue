<script setup lang="ts">
import { computed, ref, watch } from 'vue'
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
const showFailedOnly = ref(true)

const filteredLogs = computed(() =>
  showFailedOnly.value ? logs.value.filter((log) => !log.success) : logs.value,
)

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
  <Modal v-model="open" title="Settings" wide>
    <div class="settings">
      <div class="intro">
        <span aria-hidden="true" class="intro__icon i-lucide-sliders-horizontal" />
        <p class="intro__copy">Manage your subscriptions and keep fetch activity under control.</p>
      </div>

      <div class="layout">
        <div class="layout__left">
          <section class="section" aria-labelledby="subscriptions-title">
        <div class="section__heading">
          <div>
            <h4 id="subscriptions-title" class="section__title">Subscriptions</h4>
            <p class="section__description">Move your feeds in or out of TinyRSS.</p>
          </div>
          <span aria-hidden="true" class="section__icon i-lucide-rss" />
        </div>
        <input
          ref="fileInput"
          class="import-input"
          type="file"
          accept=".opml,.xml,application/xml,text/xml"
          :disabled="importing"
          @change="onImportFile"
        />
        <div class="action-grid">
          <div class="action-card">
            <span aria-hidden="true" class="action-card__icon i-lucide-upload" />
            <div class="action-card__content">
              <strong>Import an OPML file</strong>
              <span>Bring subscriptions from another reader.</span>
            </div>
            <Button class="action-card__button" variant="ghost" :disabled="importing" @click="openImport">
              Import
            </Button>
          </div>
          <div class="action-card">
            <span aria-hidden="true" class="action-card__icon i-lucide-download" />
            <div class="action-card__content">
              <strong>Export your subscriptions</strong>
              <span>Download a portable OPML backup.</span>
            </div>
            <Button class="action-card__button" variant="ghost" @click="exportOpml">Export</Button>
          </div>
        </div>
      </section>

      <section class="section" aria-labelledby="retention-title">
        <div class="section__heading">
          <div>
            <h4 id="retention-title" class="section__title">Fetch log retention</h4>
            <p class="section__description">Choose how long completed fetch attempts are kept.</p>
          </div>
          <span aria-hidden="true" class="section__icon i-lucide-clock" />
        </div>
        <label class="retention">
          <span class="retention__label">Auto-clean logs older than</span>
          <span class="retention__control">
            <input
              v-model.number="cleanupDays"
              class="retention__input"
              type="number"
              min="0"
              step="1"
              :disabled="settingsLoading"
              aria-label="Auto-clean fetch logs older than (days)"
              @change="saveCleanupDays"
            />
            <span class="retention__unit">days</span>
          </span>
          <span class="retention__hint">Set to 0 to keep logs indefinitely.</span>
        </label>
      </section>

        </div>

        <div class="layout__right">
          <section class="section section--activity" aria-labelledby="activity-title">
        <div class="section__heading">
          <div>
            <h4 id="activity-title" class="section__title">Recent fetch activity</h4>
            <p class="section__description">The latest attempts across all subscriptions.</p>
          </div>
          <div class="section__tools">
            <button
              class="filter-btn"
              :class="{ 'filter-btn--active': showFailedOnly }"
              type="button"
              :aria-pressed="showFailedOnly"
              @click="showFailedOnly = !showFailedOnly"
            >
              <span aria-hidden="true" class="i-lucide-filter text-[13px]" />
              Failed only
            </button>
            <span aria-hidden="true" class="section__icon i-lucide-activity" />
          </div>
        </div>

        <div v-if="logsLoading" class="state" aria-live="polite">
          <span aria-hidden="true" class="state__icon i-lucide-loader-circle" />
          Loading activity…
        </div>
        <div v-else-if="logsError" class="state state--error" role="alert">
          <span aria-hidden="true" class="state__icon i-lucide-circle-alert" />
          {{ logsError }}
        </div>
        <div v-else-if="filteredLogs.length === 0" class="state">
          <span aria-hidden="true" class="state__icon i-lucide-inbox" />
          {{ showFailedOnly ? 'No failed fetches.' : 'No fetch activity yet.' }}
        </div>
        <ul v-else class="logs">
          <li v-for="log in filteredLogs" :key="log.id" class="log">
            <span
              class="log__dot"
              :class="{ 'log__dot--ok': log.success, 'log__dot--err': !log.success }"
              :title="log.success ? 'Fetch succeeded' : 'Fetch failed'"
            />
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
        </div>
      </div>
    </div>
  </Modal>
</template>

<style scoped>
.settings {
  display: flex;
  flex-direction: column;
  gap: 12px;
  min-width: 0;
}
.layout {
  display: flex;
  gap: 12px;
}

.layout__left {
  width: 400px;
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: 12px;
}
.layout__right {
  width: 0;
  flex: 1;
}
.intro {
  display: flex;
  align-items: flex-start;
  gap: 9px;
  padding: 10px 12px;
  border: 1px solid #dbe7ff;
  border-radius: 8px;
  background: #f5f8ff;
  color: #4c6080;
}
.intro__icon {
  flex: none;
  margin-top: 1px;
  color: #2f6fed;
  font-size: 16px;
}
.intro__copy {
  margin: 0;
  font-size: 12px;
  line-height: 1.45;
}
.section {
  padding: 13px;
  border: 1px solid #e3e7ee;
  border-radius: 9px;
  background: #fff;
}
.section--activity {
  display: flex;
  height: 330px;
  flex-direction: column;
  padding-bottom: 8px;
}
.section__heading {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 11px;
}
.section__title {
  margin: 0;
  color: #2c3440;
  font-size: 13px;
  font-weight: 650;
}
.section__description {
  margin: 3px 0 0;
  color: #7b8491;
  font-size: 11.5px;
  line-height: 1.35;
}
.section__icon {
  flex: none;
  color: #9aa4b2;
  font-size: 17px;
}
.section__tools {
  display: flex;
  flex: none;
  align-items: center;
  gap: 10px;
}
.filter-btn {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  padding: 4px 8px;
  border: 1px solid #d8dde6;
  border-radius: 6px;
  background: #fff;
  color: #5b6472;
  font: inherit;
  font-size: 11.5px;
  line-height: 1;
  cursor: pointer;
}
.filter-btn:hover {
  background: #f2f4f8;
}
.filter-btn--active {
  border-color: #2f6fed;
  background: #eef4ff;
  color: #1f55c4;
}
.import-input {
  display: none;
}
.action-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 8px;
}
.action-card {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr);
  grid-template-rows: minmax(0, 1fr) auto;
  align-items: stretch;
  gap: 7px 8px;
  min-width: 0;
  padding: 10px;
  border: 1px solid #eef0f4;
  border-radius: 7px;
  background: #fafbfc;
}
.action-card__icon {
  grid-row: 1 / 3;
  align-self: start;
  margin-top: 2px;
  color: #2f6fed;
  font-size: 15px;
}
.action-card__content {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: 2px;
  align-self: start;
}
.action-card__content strong {
  color: #3c4450;
  font-size: 12px;
  font-weight: 600;
}
.action-card__content span {
  color: #7b8491;
  font-size: 11px;
  line-height: 1.35;
}
.action-card__button {
  grid-column: 1 / -1;
  justify-content: center;
  width: 100%;
}
.retention {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  align-items: center;
  gap: 7px 10px;
  color: #3c4450;
  font-size: 12px;
}
.retention__label {
  min-width: 0;
}
.retention__control {
  display: inline-flex;
  align-items: center;
  gap: 5px;
}
.retention__input {
  width: 58px;
  padding: 6px 7px;
  border: 1px solid #cfd6e1;
  border-radius: 6px;
  background: #fff;
  color: #1c222a;
  font: inherit;
  font-size: 12px;
  text-align: right;
}
.retention__input:focus {
  border-color: #2f6fed;
  outline: 2px solid rgba(47, 111, 237, 0.16);
  outline-offset: 1px;
}
.retention__input:disabled {
  background: #f2f4f8;
  cursor: wait;
  opacity: 0.65;
}
.retention__unit {
  color: #5b6472;
}
.retention__hint {
  grid-column: 1 / -1;
  color: #7b8491;
  font-size: 11px;
}
.state {
  display: flex;
  align-items: center;
  gap: 7px;
  padding: 9px 2px 5px;
  color: #7b8491;
  font-size: 12px;
}
.state__icon {
  flex: none;
  font-size: 14px;
}
.state--error {
  color: #c0392b;
}
.logs {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  list-style: none;
  margin: 0;
  padding: 0 2px 0 0;
}
.log {
  display: flex;
  gap: 9px;
  padding: 8px 0;
  border-bottom: 1px solid #eef0f4;
}
.log:first-child {
  padding-top: 2px;
}
.log:last-child {
  border-bottom: 0;
  padding-bottom: 2px;
}
.log__dot {
  flex: none;
  width: 7px;
  height: 7px;
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
  flex: 1;
}
.log__head {
  display: flex;
  align-items: baseline;
  gap: 8px;
  min-width: 0;
  font-size: 12px;
}
.log__feed {
  min-width: 0;
  overflow: hidden;
  color: #3c4450;
  font-weight: 600;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.log__time {
  flex: none;
  color: #7b8491;
  font-size: 10.5px;
}
.log__error {
  margin-top: 3px;
  color: #c0392b;
  font-size: 11.5px;
  line-height: 1.35;
  overflow-wrap: anywhere;
}
@media (max-width: 390px) {
  .action-grid {
    grid-template-columns: 1fr;
  }
}
@media (max-width: 720px) {
  .layout {
    grid-template-columns: 1fr;
  }
  .section--activity {
    height: 260px;
  }
}
</style>
