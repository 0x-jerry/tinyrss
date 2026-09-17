import { describe, it, expect } from 'vitest'
import { createSelectionProvider } from '../src/providers/selection'

describe('selection provider', () => {
  it('starts with no selection', () => {
    const p = createSelectionProvider()
    expect(p.state).toEqual({ folderId: null, feedId: null, itemId: null })
  })

  it('accepts an initial selection to restore', () => {
    const p = createSelectionProvider({ feedId: 3, itemId: 9 })
    expect(p.state.feedId).toBe(3)
    expect(p.state.itemId).toBe(9)
    expect(p.state.folderId).toBeNull()
  })

  it('selecting a feed clears any folder', () => {
    const p = createSelectionProvider()
    p.selectFolder(2)
    p.selectFeed(3)
    expect(p.state.feedId).toBe(3)
    expect(p.state.folderId).toBeNull()
    expect(p.state.itemId).toBeNull()
  })

  it('selecting a folder clears any feed', () => {
    const p = createSelectionProvider()
    p.selectFeed(3)
    p.selectFolder(2)
    expect(p.state.folderId).toBe(2)
    expect(p.state.feedId).toBeNull()
  })

  it('selecting an item only changes the item', () => {
    const p = createSelectionProvider()
    p.selectFeed(3)
    p.selectItem(9)
    expect(p.state.itemId).toBe(9)
    expect(p.state.feedId).toBe(3)
    expect(p.state.folderId).toBeNull()
  })

  it('selecting a feed resets the item', () => {
    const p = createSelectionProvider()
    p.selectItem(9)
    p.selectFeed(1)
    expect(p.state.itemId).toBeNull()
  })
})
