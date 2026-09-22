import { reactive, readonly } from 'vue'

export type ThemeMode = 'system' | 'light' | 'dark'
export type ResolvedTheme = 'light' | 'dark'

const STORAGE = 'tinyrss.theme'
const DARK_QUERY = '(prefers-color-scheme: dark)'

function readStored(): ThemeMode {
  try {
    return sanitizeTheme(localStorage.getItem(STORAGE))
  } catch {
    return 'system'
  }
}

function storeStored(value: ThemeMode) {
  try {
    localStorage.setItem(STORAGE, value)
  } catch {
    /* storage unavailable — keep in-memory only */
  }
}

function applyResolved(theme: ResolvedTheme) {
  if (typeof document !== 'undefined') document.documentElement.setAttribute('data-theme', theme)
}

function resolve(mode: ThemeMode): ResolvedTheme {
  if (mode !== 'system') return mode
  try {
    if (typeof window !== 'undefined' && window.matchMedia(DARK_QUERY).matches) return 'dark'
  } catch {
    /* matchMedia unavailable — fall through to light */
  }
  return 'light'
}

// localStorage is a trust boundary: coerce malformed/legacy values to 'system'.
export function sanitizeTheme(v: unknown): ThemeMode {
  return v === 'light' || v === 'dark' || v === 'system' ? v : 'system'
}

export interface ThemeState {
  mode: ThemeMode
  resolved: ResolvedTheme
}

export interface ThemeStore {
  state: Readonly<ThemeState>
  setMode: (mode: ThemeMode) => void
}

// Call from the store composition so the theme is applied before the first render.
export function createThemeStore(): ThemeStore {
  const initialMode = readStored()
  const state = reactive<ThemeState>({
    mode: initialMode,
    resolved: resolve(initialMode),
  })
  applyResolved(state.resolved)

  const mql = typeof window !== 'undefined' ? window.matchMedia(DARK_QUERY) : null
  if (mql) {
    mql.addEventListener('change', () => {
      if (state.mode === 'system') {
        state.resolved = resolve('system')
        applyResolved(state.resolved)
      }
    })
  }

  return {
    state: readonly(state),
    setMode: (mode) => {
      const safe = sanitizeTheme(mode)
      state.mode = safe
      state.resolved = resolve(safe)
      storeStored(safe)
      applyResolved(state.resolved)
    },
  }
}
