import { describe, it, expect } from 'vitest'
import { parseView, toQuery, mergeQuery } from '../src/composables/useViewNav'

describe('parseView', () => {
  it('defaults to an empty selection and the list view', () => {
    expect(parseView({})).toEqual({ folderId: null, feedId: null, itemId: null, view: 'list' })
  })

  it('decodes scope, item and view', () => {
    expect(parseView({ feed: '3', item: '45', view: 'reader' })).toEqual({
      folderId: null,
      feedId: 3,
      itemId: 45,
      view: 'reader',
    })
  })

  it('parses folder scope', () => {
    expect(parseView({ folder: '2', view: 'feeds' })).toEqual({
      folderId: 2,
      feedId: null,
      itemId: null,
      view: 'feeds',
    })
  })

  it('coerces malformed ids to null', () => {
    expect(parseView({ feed: 'abc', item: '', folder: '1.5' })).toEqual({
      folderId: null,
      feedId: null,
      itemId: null,
      view: 'list',
    })
  })

  it('falls back to the list view for an unknown view value', () => {
    expect(parseView({ view: 'bogus' }).view).toBe('list')
  })
})

describe('toQuery', () => {
  it('omits null ids but always keeps the view', () => {
    expect(toQuery({ folderId: null, feedId: 3, itemId: null, view: 'list' })).toEqual({
      feed: '3',
      view: 'list',
    })
  })

  it('keeps a non-default view too', () => {
    expect(toQuery({ folderId: null, feedId: null, itemId: 5, view: 'reader' })).toEqual({
      item: '5',
      view: 'reader',
    })
  })

  it('round-trips through parseView', () => {
    const state = { folderId: null, feedId: 7, itemId: 9, view: 'reader' }
    expect(parseView(toQuery(state))).toEqual(state)
  })
})

describe('mergeQuery', () => {
  it('carries forward unmanaged params and applies the view state', () => {
    expect(
      mergeQuery({ foo: '1', add_feed: 'http://x/rss' }, { folderId: null, feedId: 3, itemId: 9, view: 'reader' }),
    ).toEqual({
      foo: '1',
      add_feed: 'http://x/rss',
      feed: '3',
      item: '9',
      view: 'reader',
    })
  })

  it('overwrites managed view params instead of inheriting them', () => {
    expect(
      mergeQuery(
        { feed: '1', folder: '2', item: '3', view: 'list' },
        { folderId: null, feedId: 7, itemId: null, view: 'feeds' },
      ),
    ).toEqual({ feed: '7', view: 'feeds' })
  })

  it('drops null ids but always keeps the view', () => {
    expect(mergeQuery({}, { folderId: null, feedId: null, itemId: null, view: 'list' })).toEqual({ view: 'list' })
  })
})
