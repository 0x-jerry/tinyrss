import { describe, it, expect } from 'vitest'
import { buildContentDocument } from '../src/helpers'

describe('buildContentDocument', () => {
  const doc = buildContentDocument('<p>Hello</p>')

  it('is a self-contained html document', () => {
    expect(doc.startsWith('<!doctype html><html>')).toBe(true)
    expect(doc).toContain('<meta charset="utf-8">')
    expect(doc).toContain('<style>')
  })

  it('embeds the sanitized content inside the body', () => {
    expect(doc).toContain('<body><p>Hello</p></body>')
  })
})
