<script setup lang="ts">
import { RouterView, useRouter } from 'vue-router'
import { provideAuth, provideSelection, provideFeedsTree, provideItems, provideTheme } from './providers'
import { toastStore } from './api/useApiToast'

const router = useRouter()

provideTheme()

// Bounce to /login whenever a logout happens (user-initiated or a 401 from the
// api client calls provider.logout). The router guard handles initial nav.
provideAuth({
  onLogout: () => {
    if (router.currentRoute.value.path !== '/login') router.replace('/login')
  },
})

// Selection persists itself via localStorage; items reloads on scope changes
// through the selection provider's onScopeChange callback — no observers.
const selection = provideSelection()
const feedsTree = provideFeedsTree()
provideItems({ selection, feedsTree })
</script>

<template>
  <div class="app">
    <RouterView />
    <div class="toasts" aria-live="polite">
      <div v-for="t in toastStore.list" :key="t.id" class="toast" :class="`toast--${t.kind}`">
        <span v-if="t.kind === 'error'" aria-hidden="true" class="i-lucide-x text-[13px]" />
        {{ t.message }}
      </div>
    </div>
  </div>
</template>

<style>
.app {
  height: 100%;
}
.toasts {
  position: fixed;
  top: 12px;
  right: 12px;
  z-index: 200;
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.toast {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  max-width: 320px;
  padding: 9px 12px;
  border-radius: 8px;
  font-size: 13px;
  color: var(--on-accent);
  box-shadow: 0 6px 20px rgba(0, 0, 0, 0.18);
}
.toast--error {
  background: var(--danger);
}
.toast--success {
  background: var(--success);
}
</style>
