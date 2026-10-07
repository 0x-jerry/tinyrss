import { reactive, readonly } from 'vue'

// Chromium-only event, absent from the DOM lib's type definitions.
interface BeforeInstallPromptEvent extends Event {
  prompt: () => Promise<void>
  userChoice: Promise<{ outcome: 'accepted' | 'dismissed' }>
}

export type InstallOutcome = 'accepted' | 'dismissed' | 'unavailable'

export interface InstallState {
  canPrompt: boolean
  isInstalled: boolean
  isIOS: boolean
}

export interface InstallStore {
  state: Readonly<InstallState>
  init: () => void
  promptInstall: () => Promise<InstallOutcome>
}

const STANDALONE = '(display-mode: standalone)'

export function isStandaloneDisplay(displayModeMatches: boolean, navStandalone: boolean | undefined): boolean {
  return displayModeMatches || navStandalone === true
}

// iPadOS reports a desktop Safari UA, so touch points are the giveaway.
export function isIOSDevice(userAgent: string, maxTouchPoints: number): boolean {
  return /iPad|iPhone|iPod/.test(userAgent) || (/Mac/.test(userAgent) && maxTouchPoints > 1)
}

type NavigatorWithStandalone = Navigator & { standalone?: boolean }

const state = reactive<InstallState>({
  canPrompt: false,
  isInstalled: false,
  isIOS: false,
})

let deferredPrompt: BeforeInstallPromptEvent | null = null
let started = false

// Captured at app start: beforeinstallprompt can fire long before the settings
// dialog that offers the button is mounted.
export const installStore: InstallStore = {
  state: readonly(state),
  init: () => {
    if (started || typeof window === 'undefined') return
    started = true

    const nav = window.navigator as NavigatorWithStandalone
    const mql = window.matchMedia?.(STANDALONE)
    state.isIOS = isIOSDevice(nav.userAgent, nav.maxTouchPoints)
    state.isInstalled = isStandaloneDisplay(mql?.matches ?? false, nav.standalone)
    mql?.addEventListener('change', () => {
      state.isInstalled = isStandaloneDisplay(mql.matches, nav.standalone)
    })

    window.addEventListener('beforeinstallprompt', (event) => {
      event.preventDefault()
      deferredPrompt = event as BeforeInstallPromptEvent
      state.canPrompt = true
    })
    window.addEventListener('appinstalled', () => {
      deferredPrompt = null
      state.canPrompt = false
      state.isInstalled = true
    })
  },
  promptInstall: async () => {
    const event = deferredPrompt
    if (!event) return 'unavailable'
    deferredPrompt = null
    state.canPrompt = false
    await event.prompt()
    return (await event.userChoice).outcome
  },
}
