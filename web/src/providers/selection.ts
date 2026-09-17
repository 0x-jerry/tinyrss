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

export function createSelectionProvider(): SelectionProvider {
  const raw = reactive<SelectionState>({ folderId: null, feedId: null, itemId: null })

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

export function provideSelection(): SelectionProvider {
  const p = createSelectionProvider()
  provide(selectionKey, p)
  return p
}

export function injectSelection(): SelectionProvider {
  const p = inject<SelectionProvider>(selectionKey)
  if (!p) throw new Error('selection provider not provided')
  return p
}
