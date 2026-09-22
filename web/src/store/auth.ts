import { reactive, readonly } from 'vue'
import { ApiError } from '../api/client'
import { api } from '../api/endpoints'

const STORAGE = 'tinyrss.token'

function readStored(): string {
  try {
    return localStorage.getItem(STORAGE) ?? ''
  } catch {
    return ''
  }
}

function storeStored(value: string) {
  try {
    if (value) localStorage.setItem(STORAGE, value)
    else localStorage.removeItem(STORAGE)
  } catch {
    /* storage unavailable — keep in-memory only */
  }
}

export interface AuthState {
  token: string
  isAuthenticated: boolean
}

export interface AuthStore {
  state: Readonly<AuthState>
  login: (token: string) => void
  logout: () => void
  /** True when the app may proceed: auth disabled server-side, or a token is set. */
  probe: () => Promise<boolean>
  /** Set the logout redirect (App wires it to /login); also fired on 401. */
  setLogoutHandler: (fn: () => void) => void
}

// Module-scoped singleton so the router guard can read auth state before any
// component renders, independent of the rest of the store.
const state = reactive<AuthState>({ token: readStored(), isAuthenticated: readStored() !== '' })

let onLogout: (() => void) | null = null

export const authStore: AuthStore = {
  state: readonly(state),
  setLogoutHandler: (fn) => {
    onLogout = fn
  },
  login: (token) => {
    state.token = token
    state.isAuthenticated = true
    storeStored(token)
  },
  logout: () => {
    state.token = ''
    state.isAuthenticated = false
    storeStored('')
    onLogout?.()
  },
  probe: async () => {
    if (state.token) {
      state.isAuthenticated = true
      return true
    }
    // No token: a 2xx from a guarded endpoint means auth is disabled.
    try {
      await api.stats()
      state.isAuthenticated = true
      return true
    } catch (e) {
      if (e instanceof ApiError && e.status === 401) return false
      throw e
    }
  },
}

export function isAuthenticated(): boolean {
  return state.isAuthenticated
}

// Returns the raw reactive state (not the readonly wrapper) so the router guard
// and tests can read and toggle it directly.
export function getAuthState(): Readonly<AuthState> {
  return state
}
