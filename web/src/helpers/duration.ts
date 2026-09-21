export type DurationUnit = 'seconds' | 'minutes' | 'hours' | 'days'

const UNIT_SECONDS: Record<DurationUnit, number> = {
  seconds: 1,
  minutes: 60,
  hours: 3600,
  days: 86400,
}

// Pick the largest whole unit that divides evenly, so 900s renders as
// "15 minutes" rather than "900 seconds"; non-even remainders stay in seconds.
export function secondsToDuration(seconds: number): { value: number; unit: DurationUnit } {
  const order: DurationUnit[] = ['days', 'hours', 'minutes', 'seconds']
  for (const unit of order) {
    if (seconds % UNIT_SECONDS[unit] === 0) return { value: seconds / UNIT_SECONDS[unit], unit }
  }
  return { value: seconds, unit: 'seconds' }
}

export function durationToSeconds(value: number, unit: DurationUnit): number {
  return value * UNIT_SECONDS[unit]
}
