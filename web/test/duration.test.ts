import { describe, it, expect } from 'vitest'
import { secondsToDuration, durationToSeconds, type DurationUnit } from '../src/helpers'

describe('secondsToDuration', () => {
  it('picks the largest whole unit that divides evenly', () => {
    expect(secondsToDuration(2592000)).toEqual({ value: 30, unit: 'days' })
    expect(secondsToDuration(7200)).toEqual({ value: 2, unit: 'hours' })
    expect(secondsToDuration(900)).toEqual({ value: 15, unit: 'minutes' })
    expect(secondsToDuration(30)).toEqual({ value: 30, unit: 'seconds' })
  })

  it('keeps non-even remainders in seconds', () => {
    expect(secondsToDuration(100)).toEqual({ value: 100, unit: 'seconds' })
  })

  it('handles zero as disabled', () => {
    expect(secondsToDuration(0)).toEqual({ value: 0, unit: 'days' })
  })
})

describe('duration round-trip', () => {
  it('converts back to the same seconds for every unit', () => {
    const cases: [number, DurationUnit][] = [
      [5, 'days'],
      [3, 'hours'],
      [15, 'minutes'],
      [45, 'seconds'],
    ]
    for (const [value, unit] of cases) {
      const seconds = durationToSeconds(value, unit)
      const { value: v, unit: u } = secondsToDuration(seconds)
      expect(durationToSeconds(v, u)).toBe(seconds)
    }
  })
})
