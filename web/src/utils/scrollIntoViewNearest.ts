// Return the minimal scrollTop that brings the span [offset, offset+size] fully
// into a viewport of height viewportSize, aligning to the near edge. Returns
// the current scrollTop unchanged when the span is already fully visible.
export function nearestScrollTop(offset: number, size: number, scrollTop: number, viewportSize: number): number {
  if (offset < scrollTop) return offset // above → align top edge
  const bottom = offset + size
  if (bottom > scrollTop + viewportSize) return bottom - viewportSize // below → align bottom edge
  return scrollTop // already visible → no-op
}
