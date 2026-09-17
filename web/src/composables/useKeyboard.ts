import { onKeyStroke } from '@vueuse/core'

export interface KeyboardHandlers {
  next: () => void
  prev: () => void
  toggleRead: () => void
}

/** j/k move up/down the item list, m toggles read/unread. */
export function useKeyboard(handlers: KeyboardHandlers) {
  onKeyStroke('j', () => handlers.next())
  onKeyStroke('k', () => handlers.prev())
  onKeyStroke('m', () => handlers.toggleRead())
}
