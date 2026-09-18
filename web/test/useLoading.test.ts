import { describe, it, expect } from 'vitest'
import { useLoading } from '../src/composables/useLoading'

describe('useLoading', () => {
  it('reflects pending executions and resets after', async () => {
    let release!: () => void
    const gate = new Promise<void>((resolve) => {
      release = resolve
    })
    const run = useLoading(async () => {
      await gate
      return 1
    })

    expect(run.isLoading).toBe(false)
    const pending = run()
    expect(run.isLoading).toBe(true)
    release()
    await expect(pending).resolves.toBe(1)
    expect(run.isLoading).toBe(false)
  })

  it('stays pending until every concurrent call settles', async () => {
    let release!: () => void
    const gate = new Promise<void>((resolve) => {
      release = resolve
    })
    const run = useLoading(async () => {
      await gate
    })

    const first = run()
    const second = run()
    expect(run.isLoading).toBe(true)
    release()
    await Promise.all([first, second])
    expect(run.isLoading).toBe(false)
  })

  it('clears pending when the action rejects', async () => {
    const run = useLoading(async () => {
      throw new Error('boom')
    })

    await expect(run()).rejects.toThrow('boom')
    expect(run.isLoading).toBe(false)
  })
})
