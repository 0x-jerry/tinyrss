import { describe, it, expect } from 'vitest'
import { parseFoldState } from '../src/composables/useFeedFolds'

describe('parseFoldState', () => {
  it('returns empty for empty or malformed input', () => {
    expect(parseFoldState('')).toEqual({})
    expect(parseFoldState('not json')).toEqual({})
    expect(parseFoldState('42')).toEqual({})
    expect(parseFoldState('"str"')).toEqual({})
  })

  it('keeps boolean entries keyed by integer folder ids', () => {
    expect(parseFoldState('{"1":true,"7":false}')).toEqual({ 1: true, 7: false })
  })

  it('drops non-boolean values and non-integer keys', () => {
    expect(parseFoldState('{"1":"yes","2":1,"bad":false,"3":null}')).toEqual({})
  })
})
