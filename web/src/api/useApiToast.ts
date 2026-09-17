import { reactive } from 'vue'
import { ApiError } from './client'

export interface Toast {
  id: number
  kind: 'error' | 'success'
  message: string
}

// Module-scoped so any component can push toasts; App.vue renders the list.
export const toastStore = reactive<{ list: Toast[] }>({ list: [] })

let nextId = 1

function push(kind: Toast['kind'], message: string) {
  const id = nextId++
  toastStore.list.push({ id, kind, message })
  setTimeout(() => {
    const i = toastStore.list.findIndex((t) => t.id === id)
    if (i >= 0) toastStore.list.splice(i, 1)
  }, 3500)
}

export function useApiToast() {
  return {
    error: (message: string) => push('error', message),
    success: (message: string) => push('success', message),
    fromError: (e: unknown) => push('error', e instanceof ApiError ? e.message : String(e ?? 'request failed')),
  }
}

export function useToastStore() {
  return toastStore
}
