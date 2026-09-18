<script setup lang="ts">
import { computed, ref } from 'vue'
import { injectFeedsTree, buildTree } from '../../providers/feedsTree'
import { injectSelection } from '../../providers/selection'
import { injectAuth } from '../../providers/auth'
import { useApiToast } from '../../api/useApiToast'
import { useFeedFolds } from '../../composables/useFeedFolds'
import type { Feed } from '../../types/models'
import Button from '../shared/Button.vue'
import Badge from '../shared/Badge.vue'
import ConfirmDialog from '../shared/ConfirmDialog.vue'
import SettingsModal from './SettingsModal.vue'
import AddFeedModal from './AddFeedModal.vue'
import EditFeedModal from './EditFeedModal.vue'
import RenameFolderModal from './RenameFolderModal.vue'

export interface FeedTreeEmits {
  openList: []
  close: []
}

const emit = defineEmits<FeedTreeEmits>()

const feeds = injectFeedsTree()
const selection = injectSelection()
const auth = injectAuth()
const toast = useApiToast()
const { uncategorizedCollapsed, isCollapsed, toggleFolder, toggleUncategorized } = useFeedFolds()

const newFolderName = ref('')
const confirmOpen = ref(false)
const pendingDelete = ref<{ kind: 'feed' | 'folder'; id: number; name: string } | null>(null)
const draggingFeedId = ref<number | null>(null)
const dropTarget = ref<{ folderId: number | null } | null>(null)
const settingsOpen = ref(false)
const addFeedOpen = ref(false)
const editOpen = ref(false)
const feedToEdit = ref<Feed | null>(null)
const renameOpen = ref(false)
const folderToRename = ref<{ id: number; name: string } | null>(null)
const search = ref('')

const filteredTree = computed(() => {
  const q = search.value.trim().toLowerCase()
  if (!q) return feeds.state.tree
  return buildTree(
    feeds.state.feeds.filter((f) => f.title.toLowerCase().includes(q)),
    feeds.state.folders,
  )
})

// While searching, expand every folder so matches inside collapsed ones are visible.
const expanded = computed(() => search.value.trim().length > 0)

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

function openEdit(feed: Feed) {
  feedToEdit.value = feed
  editOpen.value = true
}

function openRenameFolder(folder: { id: number; name: string }) {
  folderToRename.value = folder
  renameOpen.value = true
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
  emit('openList')
}

function selectFeed(id: number) {
  selection.selectFeed(id)
  emit('openList')
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

function folderIcon(id: number): string {
  return isCollapsed(id) ? 'i-lucide-folder' : 'i-lucide-folder-open'
}

function folderTitle(id: number): string {
  return isCollapsed(id) ? 'Expand folder' : 'Collapse folder'
}

function uncategorizedFolderIcon(): string {
  return uncategorizedCollapsed.value ? 'i-lucide-folder' : 'i-lucide-folder-open'
}

function uncategorizedFolderTitle(): string {
  return uncategorizedCollapsed.value ? 'Expand' : 'Collapse'
}

async function refreshAll() {
  if (feeds.state.refresh.running) return
  try {
    await feeds.refreshAll()
    const { failed } = feeds.state.refresh
    toast.success(failed > 0 ? `Feeds refreshed (${failed} failed)` : 'Feeds refreshed')
  } catch (e) {
    toast.fromError(e)
  }
}

const refreshPercent = computed(() => {
  const { total, done } = feeds.state.refresh
  return total > 0 ? Math.round((done / total) * 100) : 0
})
</script>

<template>
  <aside class="feeds">
    <header class="feeds__header">
      <Button variant="ghost" size="sm" title="Close menu" class="feeds__close" @click="emit('close')">
        <span aria-hidden="true" class="i-lucide-chevron-left text-[16px]" />
      </Button>
      <span class="brand"><span aria-hidden="true" class="i-lucide-rss text-[16px]" /> TinyRSS</span>
      <div class="feeds__actions">
        <Button variant="ghost" size="sm" :disabled="feeds.state.refresh.running" title="Refresh all feeds" @click="refreshAll">
          <span aria-hidden="true" class="i-lucide-refresh-cw text-[16px]" :class="{ spin: feeds.state.refresh.running }" />
        </Button>
      </div>
    </header>

    <div v-if="feeds.state.refresh.running" class="refresh-bar" role="progressbar"
      aria-valuemin="0" aria-valuemax="100" :aria-valuenow="refreshPercent">
      <div class="refresh-bar__track">
        <div class="refresh-bar__fill" :style="{ width: refreshPercent + '%' }" />
      </div>
      <span class="refresh-bar__label">
        Refreshing {{ feeds.state.refresh.done }}/{{ feeds.state.refresh.total
        }}<template v-if="feeds.state.refresh.currentTitle"> · {{ feeds.state.refresh.currentTitle }}</template>
      </span>
    </div>

    <div class="add">
      <input v-model="search" class="add__input" placeholder="Search feeds by name" aria-label="Search feeds" />
      <Button size="sm" title="Add feed" @click="addFeedOpen = true"><span aria-hidden="true" class="i-lucide-plus text-[16px]" /></Button>
    </div>

    <nav class="tree">
      <div class="row row--inbox" :class="{ active: selection.state.folderId === null && selection.state.feedId === null }" @click="selectFolder(null)">
        <span aria-hidden="true" class="i-lucide-rss text-[16px]" />
        <span class="row__label">All articles</span>
        <Badge :count="filteredTree.totalUnread" />
      </div>

      <section v-for="folder in filteredTree.folderNodes" :key="folder.id" class="folder">
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
          <button v-if="folder.feeds.length" class="row__act" title="Rename folder" @click.stop="openRenameFolder({ id: folder.id, name: folder.name })">
            <span aria-hidden="true" class="i-lucide-pencil text-[13px]" />
          </button>
          <button class="row__act row__act--danger" title="Delete folder" @click.stop="confirmDelete('folder', folder.id, folder.name)">
            <span aria-hidden="true" class="i-lucide-trash text-[13px]" />
          </button>
        </div>
        <div v-if="expanded || !isCollapsed(folder.id)" class="folder__feeds">
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
            <span v-if="feed.fetch_error" class="feed__err" :title="feed.fetch_error" aria-label="Fetch error" />
            <button class="row__act" title="Edit feed" @click.stop="openEdit(feed)"><span aria-hidden="true" class="i-lucide-pencil text-[13px]" /></button>
            <button class="row__act row__act--danger" title="Delete feed" @click.stop="confirmDelete('feed', feed.id, feed.title)"><span aria-hidden="true" class="i-lucide-trash text-[13px]" /></button>
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
          <Badge :count="filteredTree.uncategorizedUnread" />
        </div>
        <div v-if="expanded || !uncategorizedCollapsed" class="folder__feeds">
          <div
            v-for="feed in filteredTree.uncategorized"
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
            <span v-if="feed.fetch_error" class="feed__err" :title="feed.fetch_error" aria-label="Fetch error" />
            <button class="row__act" title="Edit feed" @click.stop="openEdit(feed)"><span aria-hidden="true" class="i-lucide-pencil text-[13px]" /></button>
            <button class="row__act row__act--danger" title="Delete feed" @click.stop="confirmDelete('feed', feed.id, feed.title)"><span aria-hidden="true" class="i-lucide-trash text-[13px]" /></button>
          </div>
        </div>
      </section>
    </nav>

    <footer class="feeds__footer">
      <form class="add" @submit.prevent="addFolder">
        <input v-model="newFolderName" class="add__input" placeholder="New folder name" aria-label="New folder name" />
        <Button size="sm" type="submit" title="Add folder"><span aria-hidden="true" class="i-lucide-plus text-[16px]" /></Button>
      </form>
      <div class="footer__bar">
        <Button variant="ghost" size="sm" class="logout" @click="auth.logout()">
          <span aria-hidden="true" class="i-lucide-log-out text-[16px]" /> Log out
        </Button>
        <Button variant="ghost" size="sm" title="Settings" @click="settingsOpen = true">
          <span aria-hidden="true" class="i-lucide-settings text-[16px]" />
        </Button>
      </div>
    </footer>

    <SettingsModal v-model="settingsOpen" />
    <AddFeedModal v-model="addFeedOpen" />
    <EditFeedModal :feed="feedToEdit" v-model="editOpen" />
    <RenameFolderModal :folder="folderToRename" v-model="renameOpen" />
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
  border-right: 1px solid var(--border);
  background: var(--bg-subtle);
}
.feeds__close {
  display: none;
}
@media (max-width: 768px) {
  .feeds__close {
    display: inline-flex;
  }
  .feeds {
    width: 100%;
    min-width: 0;
    flex: 1 1 auto;
    border-right: 0;
  }
}
.feeds__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px;
  border-bottom: 1px solid var(--border-subtle);
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
  color: var(--accent);
}
.add {
  display: flex;
  gap: 6px;
  padding: 8px 12px;
}
.refresh-bar {
  padding: 6px 12px;
  border-bottom: 1px solid var(--border-subtle);
}
.refresh-bar__track {
  height: 4px;
  border-radius: 2px;
  background: var(--accent-soft);
  overflow: hidden;
}
.refresh-bar__fill {
  height: 100%;
  background: var(--accent);
  transition: width 0.2s ease;
}
.refresh-bar__label {
  display: block;
  margin-top: 5px;
  font-size: 12px;
  color: var(--text-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.add__input {
  flex: 1;
  min-width: 0;
  padding: 6px 8px;
  border: 1px solid var(--border-strong);
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
  color: var(--text-secondary);
  cursor: pointer;
  font-size: 13px;
}
.row:hover {
  background: var(--bg-hover);
}
.row.active {
  background: var(--surface-active);
  color: var(--accent-soft-text);
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
  color: var(--text-faint);
  cursor: pointer;
  padding: 3px;
  border-radius: 5px;
  transition: background-color 0.12s ease, color 0.12s ease;
}
.row__act:hover {
  background: var(--bg-hover);
  color: var(--text);
}
.row__act--danger:hover {
  background: var(--danger-bg);
  color: var(--danger);
}
.row__act:active {
  transform: translateY(0.5px);
}
.row__act:focus-visible {
  outline: 2px solid var(--accent);
  outline-offset: -1px;
}
.feed__err {
  flex: none;
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--danger);
}
.row:hover .row__act {
  display: inline-flex;
}
.row.dragging {
  opacity: 0.4;
}
.row.drop-target {
  outline: 1px dashed var(--accent);
  outline-offset: -1px;
  background: var(--accent-soft);
}
.feeds__footer {
  display: flex;
  flex-direction: column;
  gap: 4px;
  border-top: 1px solid var(--border-subtle);
}
.footer__bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 12px 12px;
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
