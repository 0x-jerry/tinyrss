import { inject, reactive, readonly, provide } from 'vue'
import { useLocalStorage } from '@vueuse/core'
import { selectionKey } from './keys'

export interface SelectionState {
  folderId: number | null
  feedId: number | null
  itemId: number | null
}

export interface SelectionProvider {
  state: Readonly<SelectionState>
  selectFolder: (id: number | null) => void
  selectFeed: (id: number | null) => void
  selectItem: (id: number | null) => void
  clear: () => void
  /** Subscribe to feed/folder scope changes (item-only changes do not fire). */
  onScopeChange: (fn: () => void) => void
}

export const PERSIST_VIEW_KEY = 'tinyrss.view'

const EMPTY: SelectionState = { folderId: null, feedId: null, itemId: null }

// localStorage is a trust boundary: coerce malformed/legacy values to safe defaults.
export function sanitizeSelection(v: Partial<SelectionState> | null | undefined): SelectionState {
  const num = (n: unknown) => (typeof n === 'number' && Number.isFinite(n) ? n : null)
  return {
    folderId: num(v?.folderId),
    feedId: num(v?.feedId),
    itemId: num(v?.itemId),
  }
}

const persisted = useLocalStorage<SelectionState>(PERSIST_VIEW_KEY, EMPTY)

export function createSelectionProvider(initial?: SelectionState): SelectionProvider {
  const seed = sanitizeSelection(initial ?? persisted.value)
  const raw = reactive<SelectionState>({ ...EMPTY, ...seed })
  let onScope: (() => void) | null = null

  function commit(next: Partial<SelectionState>, scopeChanged: boolean) {
    Object.assign(raw, next)
    persisted.value = { ...raw }
    if (scopeChanged) onScope?.()
  }

  return {
    state: readonly(raw),
    onScopeChange: (fn) => {
      onScope = fn
    },
    selectFolder: (id) => commit({ folderId: id, feedId: null, itemId: null }, true),
    selectFeed: (id) => commit({ feedId: id, folderId: null, itemId: null }, true),
    selectItem: (id) => commit({ itemId: id }, false),
    clear: () => commit({ folderId: null, feedId: null, itemId: null }, true),
  }
}

export function provideSelection(initial?: SelectionState): SelectionProvider {
  const p = createSelectionProvider(initial)
  provide(selectionKey, p)
  return p
}

export function injectSelection(): SelectionProvider {
  const p = inject<SelectionProvider>(selectionKey)
  if (!p) throw new Error('selection provider not provided')
  return p
}
