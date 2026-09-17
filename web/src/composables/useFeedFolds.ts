import { useLocalStorage } from '@vueuse/core'

const FOLD_KEY = 'tinyrss.folderFold'
const UNCAT_KEY = 'tinyrss.uncategorizedFold'

// localStorage is a trust boundary; drop any malformed entries on read.
export function parseFoldState(raw: string): Record<number, boolean> {
  try {
    const v = JSON.parse(raw)
    if (typeof v !== 'object' || v === null) return {}
    const out: Record<number, boolean> = {}
    for (const [k, val] of Object.entries(v)) {
      const id = Number(k)
      if (Number.isInteger(id) && typeof val === 'boolean') out[id] = val
    }
    return out
  } catch {
    return {}
  }
}

const foldSerializer = {
  read: (raw: string) => parseFoldState(raw),
  write: (v: Record<number, boolean>) => JSON.stringify(v),
}

export function useFeedFolds() {
  // Folded by default: a folder with no saved entry is treated as collapsed.
  const collapsed = useLocalStorage<Record<number, boolean>>(FOLD_KEY, {}, { serializer: foldSerializer })
  const uncategorizedCollapsed = useLocalStorage<boolean>(UNCAT_KEY, true)

  function isCollapsed(id: number): boolean {
    return collapsed.value[id] ?? true
  }

  function toggleFolder(id: number) {
    collapsed.value[id] = !isCollapsed(id)
  }

  function toggleUncategorized() {
    uncategorizedCollapsed.value = !uncategorizedCollapsed.value
  }

  return { uncategorizedCollapsed, isCollapsed, toggleFolder, toggleUncategorized }
}
