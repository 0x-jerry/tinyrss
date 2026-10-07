import { describe, it, expect } from 'vitest'
import { installStore, isIOSDevice, isStandaloneDisplay } from '../src/store/install'

describe('isStandaloneDisplay', () => {
  it('is true for a standalone display mode or iOS navigator.standalone', () => {
    expect(isStandaloneDisplay(true, undefined)).toBe(true)
    expect(isStandaloneDisplay(false, true)).toBe(true)
  })

  it('is false in a normal browser tab', () => {
    expect(isStandaloneDisplay(false, undefined)).toBe(false)
    expect(isStandaloneDisplay(false, false)).toBe(false)
  })
})

describe('isIOSDevice', () => {
  it('detects iPhone and iPad Safari', () => {
    expect(
      isIOSDevice('Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15 Safari/604.1', 5),
    ).toBe(true)
    expect(
      isIOSDevice('Mozilla/5.0 (iPad; CPU OS 16_0 like Mac OS X) AppleWebKit/605.1.15 Safari/604.1', 5),
    ).toBe(true)
  })

  it('treats a touch-capable Mac user agent as iPadOS', () => {
    expect(isIOSDevice('Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 Safari/604.1', 5)).toBe(
      true,
    )
  })

  it('ignores desktop and Android browsers', () => {
    expect(isIOSDevice('Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) Chrome/120.0.0.0 Safari/537.36', 0)).toBe(false)
    expect(isIOSDevice('Mozilla/5.0 (Linux; Android 14; Pixel 8) Chrome/120.0.0.0', 5)).toBe(false)
  })
})

describe('installStore', () => {
  it('stays inert without a browser window', async () => {
    installStore.init()
    expect(installStore.state.canPrompt).toBe(false)
    await expect(installStore.promptInstall()).resolves.toBe('unavailable')
  })
})
