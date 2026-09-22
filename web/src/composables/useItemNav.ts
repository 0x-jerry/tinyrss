import type { ItemsStore } from '../store/items'
import type { ViewNav } from './useViewNav'

// Shared step navigation over the currently loaded list, backing the reader
// toolbar's prev/next buttons.
export function useItemNav(items: ItemsStore, nav: ViewNav) {
  async function move(step: number) {
    const list = items.state.items
    if (!list.length) return
    const idx = list.findIndex((i) => i.id === nav.state.itemId)
    const next = Math.min(Math.max(idx === -1 ? 0 : idx + step, 0), list.length - 1)
    await items.openItem(list[next].id)
  }

  return { move }
}
