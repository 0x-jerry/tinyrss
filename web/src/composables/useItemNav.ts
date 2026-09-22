import type { ItemsProvider } from '../providers/items'
import type { SelectionProvider } from '../providers/selection'

// Shared step navigation over the currently loaded list, backing both the j/k
// keyboard shortcuts (FeedLayout) and the reader toolbar's prev/next buttons.
export function useItemNav(items: ItemsProvider, selection: SelectionProvider) {
  async function move(step: number) {
    const list = items.state.items
    if (!list.length) return
    const idx = list.findIndex((i) => i.id === selection.state.itemId)
    const next = Math.min(Math.max(idx === -1 ? 0 : idx + step, 0), list.length - 1)
    await items.openItem(list[next].id)
  }

  return { move }
}
