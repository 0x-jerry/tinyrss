import { describe, it, expect } from 'vitest'
import { sanitizeSelection } from '../src/composables/usePersistentView'

describe('sanitizeSelection', () => {
  it('returns safe defaults for empty input', () => {
    expect(sanitizeSelection(undefined)).toEqual({ folderId: null, feedId: null, itemId: null })
  })

  it('keeps valid values', () => {
    expect(sanitizeSelection({ folderId: 2, feedId: null, itemId: 7 })).toEqual({
      folderId: 2,
      feedId: null,
      itemId: 7,
    })
  })

  it('coerces invalid ids to null', () => {
    expect(sanitizeSelection({ folderId: 'x' as unknown as number, feedId: NaN, itemId: null })).toEqual({
      folderId: null,
      feedId: null,
      itemId: null,
    })
  })
})
