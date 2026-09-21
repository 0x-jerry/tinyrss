import { readonly, ref, watch } from 'vue'
import { useRoute, useRouter, type LocationQuery } from 'vue-router'
import type { SelectionProvider } from '../providers/selection'

export type MobileScreen = 'feeds' | 'list' | 'reader'

export interface ViewState {
  folderId: number | null
  feedId: number | null
  itemId: number | null
  view: MobileScreen
}

const VIEWS: readonly string[] = ['feeds', 'list', 'reader']
const DEFAULT_VIEW: MobileScreen = 'list'

function num(v: unknown): number | null {
  if (typeof v === 'string' && v.trim() !== '') {
    const n = Number(v)
    return Number.isInteger(n) ? n : null
  }
  return null
}

// Decode the home-route query into sanitized view state. Malformed ids and an
// unknown/absent view fall back to safe defaults, so a hand-edited or stale
// link can never crash the layout.
export function parseView(raw: Record<string, unknown> = {}): ViewState {
  const view: MobileScreen =
    typeof raw.view === 'string' && VIEWS.includes(raw.view) ? (raw.view as MobileScreen) : DEFAULT_VIEW
  return { folderId: num(raw.folder), feedId: num(raw.feed), itemId: num(raw.item), view }
}

export function toQuery(v: ViewState): LocationQuery {
  const q: LocationQuery = {}
  if (v.feedId != null) q.feed = String(v.feedId)
  if (v.folderId != null) q.folder = String(v.folderId)
  if (v.itemId != null) q.item = String(v.itemId)
  // Always write view, including the default, so the current mobile screen is
  // explicit in the URL (and hence in history / shared links).
  q.view = v.view
  return q
}

// Binds the home route to the selection provider so the browser URL is the
// source of truth for the current scope/article, giving working back/forward
// navigation and deep-linkable/shareable views. Scope and mobile-screen changes
// push a history entry (back returns to the previous scope/screen); item-only
// changes replace the current entry so paging through articles doesn't flood
// history.
export function useViewNav(selection: SelectionProvider) {
  const route = useRoute()
  const router = useRouter()

  // First load with an empty URL means "restore the persisted view" from
  // localStorage, so don't wipe the seed. Only steer the selection when the URL
  // actually carries a scope or item (a deep link).
  let isFirstLoad = true

  // ?add_feed is a transient subscribe-entry parameter handled by FeedTree when
  // it mounts (it opens the dialog pre-filled, then clears the param). While it
  // is present the URL-nav layer stands aside: it must not rewrite the URL (and
  // drop add_feed) nor apply selection from a URL that has no scope yet. The two
  // watchers stay idempotent — A only mutates selection to match the URL, and B
  // always writes back the exact current selection — so they converge to a fixed
  // point rather than looping.
  const hasAddFeed = () => route.query.add_feed != null

  // Set while a handler-initiated navigation (push/replace/back) is in flight.
  // vue-router cancels an overlapping second navigation: after a push, the item
  // watcher's replace would cancel that push and turn it into an in-place
  // replace, silently dropping the history entry (so back lands on the wrong
  // screen). Re-armed once the navigation settles.
  let navPending = false
  router.afterEach(() => {
    navPending = false
  })

  // The mobile screen is read synchronously here (a ref) rather than from the
  // URL, which only updates after a router navigation completes. Reading the
  // route would race an in-flight push and drop the screen when an item change
  // is written right after opening the reader.
  const screen = ref<MobileScreen>(parseView(route.query).view)

  function buildQuery(patch: Partial<ViewState> = {}): LocationQuery {
    return toQuery({
      feedId: selection.state.feedId,
      folderId: selection.state.folderId,
      itemId: selection.state.itemId,
      view: screen.value,
      ...patch,
    })
  }

  // URL -> selection, and keep the local screen in step with the URL's view
  // (external navigation: back/forward, deep links).
  watch(
    () => route.query,
    (q) => {
      if (hasAddFeed()) return
      const v = parseView(q)
      screen.value = v.view
      if (isFirstLoad) {
        isFirstLoad = false
        if (v.feedId == null && v.folderId == null && v.itemId == null) return
      }
      const cur = selection.state
      if (v.feedId !== cur.feedId || v.folderId !== cur.folderId) {
        if (v.feedId != null) selection.selectFeed(v.feedId)
        else if (v.folderId != null) selection.selectFolder(v.folderId)
        else selection.clear()
      }
      if (v.itemId !== cur.itemId) selection.selectItem(v.itemId)
    },
    { immediate: true },
  )

  // Selection -> URL for item changes (j/k, reader prev/next) and for selection
  // restored from localStorage on first load. We don't push here, so paging
  // through articles replaces the current entry instead of growing history.
  watch(
    () => selection.state.itemId,
    (id) => {
      if (hasAddFeed() || navPending) return
      router.replace({ query: buildQuery({ itemId: id }) })
    },
    { immediate: true },
  )

  function push(patch: Partial<ViewState>) {
    if (patch.view != null) screen.value = patch.view
    navPending = true
    return router.push({ query: buildQuery(patch) })
  }
  function replace(patch: Partial<ViewState>) {
    if (patch.view != null) screen.value = patch.view
    navPending = true
    return router.replace({ query: buildQuery(patch) })
  }
  // Backing out of a pane pops browser history so "back" from the returned pane
  // goes to the previous scope rather than forward into the closed pane. When
  // there's no in-app entry to pop (a shared deep link), fall back to a replace.
  // Reads vue-router's history-entry shape (state.back); works while pushes are
  // used for scope/screen changes, which is the only supported entry path.
  function back(fallback: Partial<ViewState>) {
    if (fallback.view != null) screen.value = fallback.view
    if (typeof window !== 'undefined' && window.history.state?.back != null) {
      navPending = true
      router.back()
    } else {
      replace(fallback)
    }
  }

  return { screen: readonly(screen), push, replace, back }
}
