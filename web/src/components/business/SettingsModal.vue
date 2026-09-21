<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { injectFeedsTree } from '../../providers/feedsTree'
import { injectTheme, type ThemeMode } from '../../providers/theme'
import { useApiToast } from '../../api/useApiToast'
import { useLoading } from '../../composables/useLoading'
import { api } from '../../api/endpoints'
import { ApiError } from '../../api/client'
import { durationToSeconds, secondsToDuration, type DurationUnit } from '../../helpers'
import type { FetchLog } from '../../types/models'
import Modal from '../shared/Modal.vue'
import Button from '../shared/Button.vue'
import DurationInput from '../shared/DurationInput.vue'

const feeds = injectFeedsTree()
const theme = injectTheme()
const toast = useApiToast()

const themeOptions: { mode: ThemeMode; label: string }[] = [
  { mode: 'system', label: 'System' },
  { mode: 'light', label: 'Light' },
  { mode: 'dark', label: 'Dark' },
]

function onThemeChange(event: Event) {
  theme.setMode((event.target as HTMLSelectElement).value as ThemeMode)
}

const open = defineModel<boolean>({ default: false })

const fileInput = ref<HTMLInputElement | null>(null)

const logs = ref<FetchLog[]>([])
const logsLoading = ref(false)
const logsError = ref('')
const showFailedOnly = ref(true)

const filteredLogs = computed(() =>
  showFailedOnly.value ? logs.value.filter((log) => !log.success) : logs.value,
)

const cleanupValue = ref(30)
const cleanupUnit = ref<DurationUnit>('days')
const renderValue = ref(30)
const renderUnit = ref<DurationUnit>('days')
const refreshValue = ref(15)
const refreshUnit = ref<DurationUnit>('minutes')
const gapValue = ref(10)
const gapUnit = ref<DurationUnit>('minutes')
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
    const cleanup = secondsToDuration(s.fetch_log_cleanup_seconds)
    cleanupValue.value = cleanup.value
    cleanupUnit.value = cleanup.unit
    const render = secondsToDuration(s.render_cache_cleanup_seconds)
    renderValue.value = render.value
    renderUnit.value = render.unit
    const refresh = secondsToDuration(Math.max(1, s.refresh_interval_seconds))
    refreshValue.value = refresh.value
    refreshUnit.value = refresh.unit
    const gap = secondsToDuration(Math.max(1, s.min_refresh_gap_seconds))
    gapValue.value = gap.value
    gapUnit.value = gap.unit
  } catch (e) {
    toast.fromError(e)
  } finally {
    settingsLoading.value = false
  }
}

async function saveCleanupDays() {
  const value = Math.max(0, Math.floor(Number(cleanupValue.value) || 0))
  cleanupValue.value = value
  const seconds = value === 0 ? 0 : durationToSeconds(value, cleanupUnit.value)
  try {
    await api.updateSettings({ fetch_log_cleanup_seconds: seconds })
    toast.success(
      value > 0
        ? `Auto-clean set to ${value} ${cleanupUnit.value}`
        : 'Auto-clean off',
    )
  } catch (e) {
    toast.fromError(e)
  }
}

async function saveRenderCleanupDays() {
  const value = Math.max(0, Math.floor(Number(renderValue.value) || 0))
  renderValue.value = value
  const seconds = value === 0 ? 0 : durationToSeconds(value, renderUnit.value)
  try {
    await api.updateSettings({ render_cache_cleanup_seconds: seconds })
    toast.success(
      value > 0
        ? `Render cache retention set to ${value} ${renderUnit.value}`
        : 'Render cache never cleaned',
    )
  } catch (e) {
    toast.fromError(e)
  }
}

async function saveRefreshInterval() {
  const value = Math.max(1, Math.floor(Number(refreshValue.value) || 0))
  refreshValue.value = value
  const seconds = durationToSeconds(value, refreshUnit.value)
  try {
    await api.updateSettings({ refresh_interval_seconds: seconds })
    toast.success(`Feeds auto-refresh every ${value} ${refreshUnit.value}`)
  } catch (e) {
    toast.fromError(e)
  }
}

async function saveRefreshGap() {
  const value = Math.max(1, Math.floor(Number(gapValue.value) || 0))
  gapValue.value = value
  const seconds = durationToSeconds(value, gapUnit.value)
  try {
    await api.updateSettings({ min_refresh_gap_seconds: seconds })
    toast.success(`Minimum gap between manual refreshes: ${value} ${gapUnit.value}`)
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

const importFile = useLoading(async (file: File) => {
  const form = new FormData()
  form.append('file', file)
  try {
    const added = await feeds.importOpmlForm(form)
    toast.success(`Added ${added} feed${added === 1 ? '' : 's'}`)
  } catch (e) {
    toast.fromError(e)
  }
})

const exportOpml = useLoading(async () => {
  try {
    const text = await feeds.exportOpmlText()
    downloadText('tinyrss-subscriptions.opml.xml', text, 'application/xml')
  } catch (e) {
    toast.fromError(e)
  }
})

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
          <section class="section" aria-labelledby="preferences-title">
        <div class="section__heading">
          <div>
            <h4 id="preferences-title" class="section__title">Preferences</h4>
            <p class="section__description">Tune how TinyRSS looks and manage your subscription data.</p>
          </div>
          <span aria-hidden="true" class="section__icon i-lucide-tune" />
        </div>
        <label class="theme-select">
          <span class="theme-select__label">Theme</span>
          <select
            class="theme-select__control"
            :value="theme.state.mode"
            @change="onThemeChange"
          >
            <option v-for="opt in themeOptions" :key="opt.mode" :value="opt.mode">{{ opt.label }}</option>
          </select>
        </label>
        <div class="preferences-divider"></div>
        <input
          ref="fileInput"
          class="import-input"
          type="file"
          accept=".opml,.xml,application/xml,text/xml"
          :disabled="importFile.isLoading"
          @change="onImportFile"
        />
        <div class="action-grid">
          <div class="action-card">
            <span aria-hidden="true" class="action-card__icon i-lucide-upload" />
            <div class="action-card__content">
              <strong>Import an OPML file</strong>
            </div>
            <Button class="action-card__button" variant="ghost" :loading="importFile.isLoading" @click="openImport">
              Import
            </Button>
          </div>
          <div class="action-card">
            <span aria-hidden="true" class="action-card__icon i-lucide-download" />
            <div class="action-card__content">
              <strong>Export your subscriptions</strong>
            </div>
            <Button class="action-card__button" variant="ghost" :loading="exportOpml.isLoading" @click="exportOpml">Export</Button>
          </div>
        </div>
      </section>

      <section class="section" aria-labelledby="maintenance-title">
        <div class="section__heading">
          <div>
            <h4 id="maintenance-title" class="section__title">Maintenance</h4>
            <p class="section__description">Control the cleanup and refresh routines TinyRSS runs automatically.</p>
          </div>
          <span aria-hidden="true" class="section__icon i-lucide-wrench" />
        </div>
        <div class="retention-list">
          <label class="retention">
            <span class="retention__label">Fetch logs older than</span>
            <span class="retention__control">
              <DurationInput
                v-model:value="cleanupValue"
                v-model:unit="cleanupUnit"
                :min="0"
                :disabled="settingsLoading"
                aria-label="Auto-clean fetch logs older than"
                @change="saveCleanupDays"
              />
            </span>
            <span class="retention__hint">Set to 0 to keep logs indefinitely.</span>
          </label>
          <label class="retention">
            <span class="retention__label">Render cache older than</span>
            <span class="retention__control">
              <DurationInput
                v-model:value="renderValue"
                v-model:unit="renderUnit"
                :min="0"
                :disabled="settingsLoading"
                aria-label="Auto-clean render cache older than"
                @change="saveRenderCleanupDays"
              />
            </span>
            <span class="retention__hint">Set to 0 to never clean cached articles.</span>
          </label>
          <label class="retention">
            <span class="retention__label">Refresh feeds every</span>
            <span class="retention__control">
              <DurationInput
                v-model:value="refreshValue"
                v-model:unit="refreshUnit"
                :min="1"
                :disabled="settingsLoading"
                aria-label="Auto-refresh interval"
                @change="saveRefreshInterval"
              />
            </span>
            <span class="retention__hint">Changes take effect on the next cycle.</span>
          </label>
          <label class="retention">
            <span class="retention__label">Minimum gap between manual refreshes</span>
            <span class="retention__control">
              <DurationInput
                v-model:value="gapValue"
                v-model:unit="gapUnit"
                :min="1"
                :disabled="settingsLoading"
                aria-label="Minimum gap between manual refreshes"
                @change="saveRefreshGap"
              />
            </span>
            <span class="retention__hint">Feeds refreshed within this window are skipped on a manual refresh-all.</span>
          </label>
        </div>
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
  display: flex;
  min-width: 0;
  height: 542px;
}
.intro {
  display: flex;
  align-items: flex-start;
  gap: 9px;
  padding: 10px 12px;
  border: 1px solid var(--accent-soft);
  border-radius: 8px;
  background: var(--accent-soft);
  color: var(--text-muted);
}
.intro__icon {
  flex: none;
  margin-top: 1px;
  color: var(--accent);
  font-size: 16px;
}
.intro__copy {
  margin: 0;
  font-size: 12px;
  line-height: 1.45;
}
.section {
  padding: 13px;
  border: 1px solid var(--border);
  border-radius: 9px;
  background: var(--surface);
}
.section--activity {
  display: flex;
  flex: 1;
  width: 100%;
  min-height: 0;
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
  color: var(--text);
  font-size: 13px;
  font-weight: 650;
}
.section__description {
  margin: 3px 0 0;
  color: var(--text-faint);
  font-size: 11.5px;
  line-height: 1.35;
}
.section__icon {
  flex: none;
  color: var(--text-faint);
  font-size: 17px;
}
.section__tools {
  display: flex;
  flex: none;
  align-items: center;
  gap: 10px;
}
.theme-select {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  color: var(--text-secondary);
  font-size: 12px;
}
.theme-select__control {
  width: 140px;
  padding: 6px 8px;
  border: 1px solid var(--border-strong);
  border-radius: 6px;
  background: var(--surface);
  color: var(--text);
  font: inherit;
  font-size: 12px;
}
.theme-select__control:focus {
  border-color: var(--accent);
  outline: 2px solid var(--focus-ring);
  outline-offset: 1px;
}
.preferences-divider {
  margin-top: 12px;
  border-top: 1px solid var(--border-subtle);
  padding-top: 12px;
}
.filter-btn {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  padding: 4px 8px;
  border: 1px solid var(--border-strong);
  border-radius: 6px;
  background: var(--surface);
  color: var(--text-muted);
  font: inherit;
  font-size: 11.5px;
  line-height: 1;
  cursor: pointer;
}
.filter-btn:hover {
  background: var(--bg-hover);
}
.filter-btn--active {
  border-color: var(--accent);
  background: var(--accent-soft);
  color: var(--accent-soft-text);
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
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
  padding: 5px 8px;
  border: 1px solid var(--border-subtle);
  border-radius: 7px;
  background: var(--bg-subtle);
}
.action-card__icon {
  flex: none;
  color: var(--accent);
  font-size: 14px;
}
.action-card__content {
  display: flex;
  min-width: 0;
  flex: 1;
  flex-direction: column;
  gap: 1px;
}
.action-card__content strong {
  overflow: hidden;
  color: var(--text-secondary);
  font-size: 12px;
  font-weight: 600;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.action-card__button {
  flex: none;
}
.retention-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.retention-list .retention + .retention {
  padding-top: 10px;
  border-top: 1px solid var(--border-subtle);
}
.retention {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  align-items: center;
  gap: 7px 10px;
  color: var(--text-secondary);
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
.retention__hint {
  grid-column: 1 / -1;
  color: var(--text-faint);
  font-size: 11px;
}
.state {
  display: flex;
  align-items: center;
  gap: 7px;
  padding: 9px 2px 5px;
  color: var(--text-faint);
  font-size: 12px;
}
.state__icon {
  flex: none;
  font-size: 14px;
}
.state--error {
  color: var(--danger);
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
  border-bottom: 1px solid var(--border-subtle);
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
  background: var(--danger);
}
.log__dot--ok {
  background: var(--success);
}
.log__dot--err {
  background: var(--danger);
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
  color: var(--text-secondary);
  font-weight: 600;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.log__time {
  flex: none;
  color: var(--text-faint);
  font-size: 10.5px;
}
.log__error {
  margin-top: 3px;
  color: var(--danger);
  font-size: 11.5px;
  line-height: 1.35;
  overflow-wrap: anywhere;
}
@media (max-width: 390px) {
  .action-grid {
    grid-template-columns: 1fr;
  }
}
@media (max-width: 768px) {
  .layout { flex-direction: column; }
  .layout__left { width: 100%; }
  .layout__right { width: 100%; }
  .section--activity { height: 300px; }
}
</style>
