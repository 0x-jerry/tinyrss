import { describe, expect, it } from 'vitest'
import { nearestScrollTop } from '../src/utils/scrollIntoViewNearest'

const SIZE = 64
const VIEW = 300

describe('nearestScrollTop', () => {
  it('returns the current scrollTop when the span is already fully visible', () => {
    // span [100, 164) sits inside [scrollTop=100, scrollTop+300=400)
    expect(nearestScrollTop(100, SIZE, 100, VIEW)).toBe(100)
    expect(nearestScrollTop(120, SIZE, 100, VIEW)).toBe(100)
  })

  it('aligns to the top edge when the span is above the viewport', () => {
    expect(nearestScrollTop(50, SIZE, 300, VIEW)).toBe(50)
  })

  it('aligns to the bottom edge when the span is below the viewport', () => {
    expect(nearestScrollTop(400, SIZE, 100, VIEW)).toBe(164)
  })

  it('aligns to the bottom edge when the span hangs partially past the bottom', () => {
    expect(nearestScrollTop(350, SIZE, 100, VIEW)).toBe(114)
  })

  it('treats a zero-height viewport as fully below (hidden pane)', () => {
    expect(nearestScrollTop(200, SIZE, 0, 0)).toBe(264)
  })
})
