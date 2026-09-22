<script setup lang="ts">
import { RouterView, useRouter } from 'vue-router'
import { useStore } from './store'
import { toastStore } from './api/useApiToast'

const router = useRouter()

const { auth } = useStore()

// Bounce to /login whenever a logout happens (user-initiated or a 401 from the
// api client calls auth.logout). The router guard handles initial nav.
auth.setLogoutHandler(() => {
  if (router.currentRoute.value.path !== '/login') router.replace('/login')
})
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
