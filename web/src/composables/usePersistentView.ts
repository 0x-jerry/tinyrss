import { useLocalStorage } from '@vueuse/core'
import type { SelectionState } from '../providers/selection'

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

export function usePersistentView() {
  return useLocalStorage<SelectionState>(PERSIST_VIEW_KEY, EMPTY, { mergeDefaults: true })
}
