import { inject, reactive, readonly, provide } from 'vue'
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
}

const EMPTY: SelectionState = { folderId: null, feedId: null, itemId: null }

export function createSelectionProvider(initial: SelectionState = EMPTY): SelectionProvider {
  const raw = reactive<SelectionState>({ ...EMPTY, ...initial })

  return {
    state: readonly(raw),
    selectFolder: (id) => {
      raw.folderId = id
      raw.feedId = null
      raw.itemId = null
    },
    selectFeed: (id) => {
      raw.feedId = id
      raw.folderId = null
      raw.itemId = null
    },
    selectItem: (id) => {
      raw.itemId = id
    },
    clear: () => {
      raw.folderId = null
      raw.feedId = null
      raw.itemId = null
    },
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
