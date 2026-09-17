import { inject, reactive, readonly, provide } from 'vue'
import { authKey } from './keys'
import { configureAuth, ApiError } from '../api/client'
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

export interface AuthProvider {
  state: Readonly<AuthState>
  login: (token: string) => void
  logout: () => void
  /** True when the app may proceed: auth disabled server-side, or a token is set. */
  probe: () => Promise<boolean>
}

// Module-scoped singleton so the router guard can read auth state before any component renders.
const state = reactive<AuthState>({ token: readStored(), isAuthenticated: readStored() !== '' })

const provider: AuthProvider = {
  state: readonly(state),
  login: (token) => {
    state.token = token
    state.isAuthenticated = true
    storeStored(token)
  },
  logout: () => {
    state.token = ''
    state.isAuthenticated = false
    storeStored('')
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

export function provideAuth(): AuthProvider {
  configureAuth({
    getToken: () => state.token || null,
    onUnauthorized: () => provider.logout(),
  })
  provide(authKey, provider)
  return provider
}

export function injectAuth(): AuthProvider {
  const p = inject<AuthProvider>(authKey)
  if (!p) throw new Error('auth provider not provided')
  return p
}

export function isAuthenticated(): boolean {
  return state.isAuthenticated
}

export function getAuthState(): Readonly<AuthState> {
  return state
}
