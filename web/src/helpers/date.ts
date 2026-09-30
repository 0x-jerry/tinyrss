// Format an ISO timestamp with the user's locale, e.g. "Jan 5, 2026, 03:04 PM".
// Falls back to the raw value when it is empty or unparseable.
export function formatDateTime(iso: string): string {
  if (!iso) return ''
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return iso
  return d.toLocaleString(undefined, {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  })
}
