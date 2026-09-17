import { describe, it, expect } from 'vitest'
import { renderKind, RenderMode } from '../src/renderMode'

describe('renderKind', () => {
  it('maps modes to render kinds', () => {
    expect(renderKind(RenderMode.Content, 'https://x/1')).toBe('content')
    expect(renderKind(RenderMode.Iframe, 'https://x/1')).toBe('iframe')
    expect(renderKind(RenderMode.Server, 'https://x/1')).toBe('server')
  })

  it('falls back to content when there is no url', () => {
    expect(renderKind(RenderMode.Iframe, null)).toBe('content')
    expect(renderKind(RenderMode.Server, null)).toBe('content')
    expect(renderKind(RenderMode.Content, null)).toBe('content')
  })
})
