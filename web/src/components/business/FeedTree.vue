<script setup lang="ts">
import { ref } from 'vue'
import { injectFeedsTree } from '../../providers/feedsTree'
import { injectSelection } from '../../providers/selection'
import { injectItems } from '../../providers/items'
import { injectAuth } from '../../providers/auth'
import { useApiToast } from '../../api/useApiToast'
import type { Feed } from '../../types/models'
import Button from '../shared/Button.vue'
import Badge from '../shared/Badge.vue'
import ConfirmDialog from '../shared/ConfirmDialog.vue'

const feeds = injectFeedsTree()
const selection = injectSelection()
const items = injectItems()
const auth = injectAuth()
const toast = useApiToast()

const newUrl = ref('')
const newFolderName = ref('')
const confirmOpen = ref(false)
const pendingDelete = ref<{ kind: 'feed' | 'folder'; id: number; name: string } | null>(null)

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
  items.load().catch(() => {})
}

function selectFeed(id: number) {
  selection.selectFeed(id)
  items.load().catch(() => {})
}

function moveFeed(feed: Feed, raw: string) {
  const folderId = raw === 'none' ? null : Number(raw)
  feeds.moveFeed(feed.id, Number.isFinite(folderId) ? folderId : null).catch((e) => toast.fromError(e))
}

async function refreshAll() {
  try {
    await feeds.refreshAll()
    toast.success('Feeds refreshed')
  } catch (e) {
    toast.fromError(e)
  }
}
</script>

<template>
  <aside class="feeds">
    <header class="feeds__header">
      <span class="brand"><span aria-hidden="true" class="i-lucide-rss text-[16px]" /> tinyrss</span>
      <Button variant="ghost" size="sm" title="Refresh all feeds" @click="refreshAll">
        <span aria-hidden="true" class="i-lucide-refresh-cw text-[16px]" />
      </Button>
    </header>

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
        <div class="row" :class="{ active: selection.state.folderId === folder.id }" @click="selectFolder(folder.id)">
          <span aria-hidden="true" class="i-lucide-folder text-[16px]" />
          <span class="row__label">{{ folder.name }}</span>
          <Badge :count="folder.unread" />
          <button v-if="folder.feeds.length" class="row__act" title="Rename folder" @click.stop="renameFolder(folder.id, folder.name)">
            <span aria-hidden="true" class="i-lucide-pencil text-[13px]" />
          </button>
          <button class="row__act" title="Delete folder" @click.stop="confirmDelete('folder', folder.id, folder.name)">
            <span aria-hidden="true" class="i-lucide-trash text-[13px]" />
          </button>
        </div>
        <div class="folder__feeds">
          <div v-for="feed in folder.feeds" :key="feed.id" class="row row--feed" :class="{ active: selection.state.feedId === feed.id }" @click="selectFeed(feed.id)">
            <span aria-hidden="true" class="i-lucide-rss text-[14px]" />
            <span class="row__label row__label--clip">{{ feed.title }}</span>
            <Badge :count="feed.unread" />
            <select class="row__move" title="Move feed" :value="String(feed.folder_id ?? 'none')" @click.stop @change="moveFeed(feed, ($event.target as HTMLSelectElement).value)">
              <option value="none">Uncategorized</option>
              <option v-for="f in feeds.state.folders" :key="f.id" :value="String(f.id)">{{ f.name }}</option>
            </select>
            <button class="row__act" title="Rename feed" @click.stop="renameFeed(feed)"><span aria-hidden="true" class="i-lucide-pencil text-[13px]" /></button>
            <button class="row__act" title="Delete feed" @click.stop="confirmDelete('feed', feed.id, feed.title)"><span aria-hidden="true" class="i-lucide-trash text-[13px]" /></button>
          </div>
        </div>
      </section>

      <section v-if="feeds.state.tree.uncategorized.length" class="folder">
        <div class="row row--inbox" :class="{ active: selection.state.feedId === null && selection.state.folderId === null }" @click="selectFolder(null)">
          <span aria-hidden="true" class="i-lucide-folder-open text-[16px]" />
          <span class="row__label">Uncategorized</span>
          <Badge :count="feeds.state.tree.uncategorizedUnread" />
        </div>
        <div class="folder__feeds">
          <div v-for="feed in feeds.state.tree.uncategorized" :key="feed.id" class="row row--feed" :class="{ active: selection.state.feedId === feed.id }" @click="selectFeed(feed.id)">
            <span aria-hidden="true" class="i-lucide-rss text-[14px]" />
            <span class="row__label row__label--clip">{{ feed.title }}</span>
            <Badge :count="feed.unread" />
            <select class="row__move" title="Move feed" :value="String(feed.folder_id ?? 'none')" @click.stop @change="moveFeed(feed, ($event.target as HTMLSelectElement).value)">
              <option value="none">Uncategorized</option>
              <option v-for="f in feeds.state.folders" :key="f.id" :value="String(f.id)">{{ f.name }}</option>
            </select>
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
  padding: 5px 7px;
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
.row__move {
  display: none;
  max-width: 90px;
  font-size: 11px;
  border: 1px solid #d8dde6;
  border-radius: 4px;
}
.row:hover .row__move {
  display: inline-flex;
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
</style>
