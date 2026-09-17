<script setup lang="ts">
import { reactive, ref } from 'vue'
import { injectFeedsTree } from '../../providers/feedsTree'
import { injectSelection } from '../../providers/selection'
import { injectAuth } from '../../providers/auth'
import { useApiToast } from '../../api/useApiToast'
import type { Feed } from '../../types/models'
import Button from '../shared/Button.vue'
import Badge from '../shared/Badge.vue'
import ConfirmDialog from '../shared/ConfirmDialog.vue'

const feeds = injectFeedsTree()
const selection = injectSelection()
const auth = injectAuth()
const toast = useApiToast()

const newUrl = ref('')
const newFolderName = ref('')
const confirmOpen = ref(false)
const pendingDelete = ref<{ kind: 'feed' | 'folder'; id: number; name: string } | null>(null)
const refreshing = ref(false)
const draggingFeedId = ref<number | null>(null)
const dropTarget = ref<{ folderId: number | null } | null>(null)
const collapsed = reactive<Record<number, boolean>>({})
const uncategorizedCollapsed = ref(false)

async function addFeed() {
  const url = newUrl.value.trim()
  if (!url) return
  try {
    await feeds.addFeed(url)
    newUrl.value = ''
    toast.success('Feed added')
  } catch (e) {
    toast.fromError(e)
  }
}

async function addFolder() {
  const name = newFolderName.value.trim()
  if (!name) return
  try {
    await feeds.addFolder(name)
    newFolderName.value = ''
  } catch (e) {
    toast.fromError(e)
  }
}

function renameFeed(feed: Feed) {
  const title = window.prompt('Rename feed', feed.title)
  if (title && title.trim() && title.trim() !== feed.title) {
    feeds.renameFeed(feed.id, title.trim()).catch((e) => toast.fromError(e))
  }
}

function renameFolder(id: number, name: string) {
  const newName = window.prompt('Rename folder', name)
  if (newName && newName.trim() && newName.trim() !== name) {
    feeds.renameFolder(id, newName.trim()).catch((e) => toast.fromError(e))
  }
}

function confirmDelete(kind: 'feed' | 'folder', id: number, name: string) {
  pendingDelete.value = { kind, id, name }
  confirmOpen.value = true
}

async function doDelete() {
  const target = pendingDelete.value
  if (!target) return
  try {
    if (target.kind === 'feed') {
      if (selection.state.feedId === target.id) selection.selectFeed(null)
      await feeds.deleteFeed(target.id)
    } else {
      if (selection.state.folderId === target.id) selection.selectFolder(null)
      await feeds.deleteFolder(target.id)
    }
    toast.success(target.kind === 'feed' ? 'Feed deleted' : 'Folder deleted')
  } catch (e) {
    toast.fromError(e)
  } finally {
    pendingDelete.value = null
    confirmOpen.value = false
  }
}

function selectFolder(id: number | null) {
  selection.selectFolder(id)
}

function selectFeed(id: number) {
  selection.selectFeed(id)
}

function onDragStart(feed: Feed) {
  draggingFeedId.value = feed.id
}

function onDragEnd() {
  draggingFeedId.value = null
  dropTarget.value = null
}

function onDragOver(target: number | null) {
  dropTarget.value = { folderId: target }
}

function onDrop(target: number | null) {
  dropTarget.value = null
  const id = draggingFeedId.value
  draggingFeedId.value = null
  if (id == null) return
  const feed = feeds.state.feeds.find((f) => f.id === id)
  if (!feed || feed.folder_id === target) return
  feeds.moveFeed(id, target).catch((e) => toast.fromError(e))
}

function toggleFolder(id: number) {
  collapsed[id] = !collapsed[id]
}

function toggleUncategorized() {
  uncategorizedCollapsed.value = !uncategorizedCollapsed.value
}

function folderIcon(id: number): string {
  return collapsed[id] ? 'i-lucide-folder' : 'i-lucide-folder-open'
}

function folderTitle(id: number): string {
  return collapsed[id] ? 'Expand folder' : 'Collapse folder'
}

function uncategorizedFolderIcon(): string {
  return uncategorizedCollapsed.value ? 'i-lucide-folder' : 'i-lucide-folder-open'
}

function uncategorizedFolderTitle(): string {
  return uncategorizedCollapsed.value ? 'Expand' : 'Collapse'
}

async function refreshAll() {
  if (refreshing.value) return
  refreshing.value = true
  try {
    await feeds.refreshAll()
    toast.success('Feeds refreshed')
  } catch (e) {
    toast.fromError(e)
  } finally {
    refreshing.value = false
  }
}

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
  <aside class="feeds">
    <header class="feeds__header">
      <span class="brand"><span aria-hidden="true" class="i-lucide-rss text-[16px]" /> tinyrss</span>
      <div class="feeds__actions">
        <Button variant="ghost" size="sm" :disabled="refreshing" title="Refresh all feeds" @click="refreshAll">
          <span aria-hidden="true" class="i-lucide-refresh-cw text-[16px]" :class="{ spin: refreshing }" />
        </Button>
        <Button variant="ghost" size="sm" title="Import OPML" @click="openImport">
          <span aria-hidden="true" class="i-lucide-upload text-[16px]" />
        </Button>
        <Button variant="ghost" size="sm" title="Export OPML" @click="exportOpml">
          <span aria-hidden="true" class="i-lucide-download text-[16px]" />
        </Button>
      </div>
    </header>

    <input
      ref="fileInput"
      class="import-input"
      type="file"
      accept=".opml,.xml,application/xml,text/xml"
      :disabled="importing"
      @change="onImportFile"
    />

    <form class="add" @submit.prevent="addFeed">
      <input v-model="newUrl" class="add__input" placeholder="Paste feed URL" aria-label="Feed URL" />
      <Button size="sm" type="submit" title="Add feed"><span aria-hidden="true" class="i-lucide-plus text-[16px]" /></Button>
    </form>

    <nav class="tree">
      <div class="row row--inbox" :class="{ active: selection.state.folderId === null && selection.state.feedId === null }" @click="selectFolder(null)">
        <span aria-hidden="true" class="i-lucide-rss text-[16px]" />
        <span class="row__label">All articles</span>
        <Badge :count="feeds.state.tree.totalUnread" />
      </div>

      <section v-for="folder in feeds.state.tree.folderNodes" :key="folder.id" class="folder">
        <div
          class="row"
          :class="{ 'drop-target': dropTarget?.folderId === folder.id }"
          :title="folderTitle(folder.id)"
          @click="toggleFolder(folder.id)"
          @dragover.prevent="onDragOver(folder.id)"
          @drop="onDrop(folder.id)"
        >
          <span
            aria-hidden="true"
            class="text-[16px]"
            :class="folderIcon(folder.id)"
          />
          <span class="row__label">{{ folder.name }}</span>
          <Badge :count="folder.unread" />
          <button v-if="folder.feeds.length" class="row__act" title="Rename folder" @click.stop="renameFolder(folder.id, folder.name)">
            <span aria-hidden="true" class="i-lucide-pencil text-[13px]" />
          </button>
          <button class="row__act" title="Delete folder" @click.stop="confirmDelete('folder', folder.id, folder.name)">
            <span aria-hidden="true" class="i-lucide-trash text-[13px]" />
          </button>
        </div>
        <div v-if="!collapsed[folder.id]" class="folder__feeds">
          <div
            v-for="feed in folder.feeds"
            :key="feed.id"
            class="row row--feed"
            :class="{ active: selection.state.feedId === feed.id, dragging: draggingFeedId === feed.id }"
            draggable="true"
            @click="selectFeed(feed.id)"
            @dragstart="onDragStart(feed)"
            @dragend="onDragEnd"
          >
            <span aria-hidden="true" class="i-lucide-rss text-[14px]" />
            <span class="row__label row__label--clip">{{ feed.title }}</span>
            <Badge :count="feed.unread" />
            <button class="row__act" title="Rename feed" @click.stop="renameFeed(feed)"><span aria-hidden="true" class="i-lucide-pencil text-[13px]" /></button>
            <button class="row__act" title="Delete feed" @click.stop="confirmDelete('feed', feed.id, feed.title)"><span aria-hidden="true" class="i-lucide-trash text-[13px]" /></button>
          </div>
        </div>
      </section>

      <section class="folder">
        <div
          class="row row--inbox"
          :class="{ 'drop-target': dropTarget !== null && dropTarget.folderId === null }"
          :title="uncategorizedFolderTitle()"
          @click="toggleUncategorized()"
          @dragover.prevent="onDragOver(null)"
          @drop="onDrop(null)"
        >
          <span
            aria-hidden="true"
            class="text-[16px]"
            :class="uncategorizedFolderIcon()"
          />
          <span class="row__label">Uncategorized</span>
          <Badge :count="feeds.state.tree.uncategorizedUnread" />
        </div>
        <div v-if="!uncategorizedCollapsed" class="folder__feeds">
          <div
            v-for="feed in feeds.state.tree.uncategorized"
            :key="feed.id"
            class="row row--feed"
            :class="{ active: selection.state.feedId === feed.id, dragging: draggingFeedId === feed.id }"
            draggable="true"
            @click="selectFeed(feed.id)"
            @dragstart="onDragStart(feed)"
            @dragend="onDragEnd"
          >
            <span aria-hidden="true" class="i-lucide-rss text-[14px]" />
            <span class="row__label row__label--clip">{{ feed.title }}</span>
            <Badge :count="feed.unread" />
            <button class="row__act" title="Rename feed" @click.stop="renameFeed(feed)"><span aria-hidden="true" class="i-lucide-pencil text-[13px]" /></button>
            <button class="row__act" title="Delete feed" @click.stop="confirmDelete('feed', feed.id, feed.title)"><span aria-hidden="true" class="i-lucide-trash text-[13px]" /></button>
          </div>
        </div>
      </section>
    </nav>

    <footer class="feeds__footer">
      <form class="add" @submit.prevent="addFolder">
        <input v-model="newFolderName" class="add__input" placeholder="New folder name" aria-label="New folder name" />
        <Button size="sm" type="submit" title="Add folder"><span aria-hidden="true" class="i-lucide-plus text-[16px]" /></Button>
      </form>
      <Button variant="ghost" size="sm" class="logout" @click="auth.logout()">
        <span aria-hidden="true" class="i-lucide-log-out text-[16px]" /> Log out
      </Button>
    </footer>

    <ConfirmDialog
      v-model="confirmOpen"
      title="Delete"
      :message="`Delete ${pendingDelete?.kind ?? ''} “${pendingDelete?.name ?? ''}”?`"
      confirm-text="Delete"
      @confirm="doDelete"
    />
  </aside>
</template>

<style scoped>
.feeds {
  display: flex;
  flex-direction: column;
  width: 280px;
  min-width: 280px;
  height: 100%;
  border-right: 1px solid #e3e7ee;
  background: #fafbfc;
}
.feeds__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px;
  border-bottom: 1px solid #eef0f4;
}
.feeds__actions {
  display: flex;
  align-items: center;
  gap: 8px;
}
.brand {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-weight: 700;
  color: #2f6fed;
}
.add {
  display: flex;
  gap: 6px;
  padding: 8px 12px;
}
.add__input {
  flex: 1;
  min-width: 0;
  padding: 6px 8px;
  border: 1px solid #d8dde6;
  border-radius: 6px;
  font: inherit;
  font-size: 13px;
}
.tree {
  flex: 1;
  overflow-y: auto;
  padding: 6px 8px 12px;
}
.row {
  display: flex;
  align-items: center;
  gap: 6px;
  height: 30px;
  padding: 0 7px;
  border-radius: 6px;
  color: #3c4450;
  cursor: pointer;
  font-size: 13px;
}
.row:hover {
  background: #eef1f6;
}
.row.active {
  background: #e3ecfd;
  color: #1f55c4;
}
.row--feed {
  padding-left: 22px;
  font-size: 12.5px;
}
.row__label {
  flex: 1;
  min-width: 0;
}
.row__label--clip {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.row__act {
  display: none;
  border: 0;
  background: none;
  color: #7b8491;
  cursor: pointer;
  padding: 2px;
}
.row:hover .row__act {
  display: inline-flex;
}
.row.dragging {
  opacity: 0.4;
}
.row.drop-target {
  outline: 1px dashed #2f6fed;
  outline-offset: -1px;
  background: #eef4ff;
}
.feeds__footer {
  display: flex;
  flex-direction: column;
  gap: 4px;
  border-top: 1px solid #eef0f4;
}
.logout {
  align-self: flex-start;
  margin-left: 12px;
  margin-bottom: 12px;
}
.import-input {
  display: none;
}
.spin {
  animation: refresh-spin 1s linear infinite;
}
@keyframes refresh-spin {
  to {
    transform: rotate(360deg);
  }
}
</style>
