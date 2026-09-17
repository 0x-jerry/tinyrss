<script setup lang="ts">
import { watch } from 'vue'
import { RouterView, useRouter } from 'vue-router'
import { provideAuth, provideSelection, provideFeedsTree, provideItems } from './providers'
import { getAuthState } from './providers/auth'
import { toastStore } from './api/useApiToast'
import Icon from './components/shared/Icon.vue'

provideAuth()
const selection = provideSelection()
const feedsTree = provideFeedsTree()
provideItems({ selection, feedsTree })

const router = useRouter()

// Bounce to /login when a 401 (or explicit logout) makes us unauthenticated.
// The router guard handles the initial navigation.
watch(
  () => getAuthState().isAuthenticated,
  (v) => {
    if (!v && router.currentRoute.value.path !== '/login') router.replace('/login')
  },
)
</script>

<template>
  <div class="app">
    <RouterView />
    <div class="toasts" aria-live="polite">
      <div v-for="t in toastStore.list" :key="t.id" class="toast" :class="`toast--${t.kind}`">
        <Icon v-if="t.kind === 'error'" name="x" :size="13" />
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
  color: #fff;
  box-shadow: 0 6px 20px rgba(0, 0, 0, 0.18);
}
.toast--error {
  background: #d64545;
}
.toast--success {
  background: #2e9e5b;
}
</style>
