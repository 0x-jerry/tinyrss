import { reactive, readonly, ref, type Ref, watch } from 'vue'
import { createSharedComposable, useMediaQuery } from '@vueuse/core'
import { useRoute, useRouter, type LocationQuery } from 'vue-router'

export type MobileScreen = 'feeds' | 'list' | 'reader'

export interface SelectionState {
  feedId: number | null
  itemId: number | null
}

export interface ViewState extends SelectionState {
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
  return { feedId: num(raw.feed), itemId: num(raw.item), view }
}

export function toQuery(v: ViewState): LocationQuery {
  const q: LocationQuery = {}
  if (v.feedId != null) q.feed = String(v.feedId)
  if (v.itemId != null) q.item = String(v.itemId)
  // Always write view, including the default, so the current mobile screen is
  // explicit in the URL (and hence in history / shared links).
  q.view = v.view
  return q
}

// The view fields this composable owns. Every other query param is carried
// forward untouched, so an in-progress URL update never drops unrelated state.
const MANAGED_PARAMS = new Set(['feed', 'item', 'view'])

export function mergeQuery(current: LocationQuery, patch: ViewState): LocationQuery {
  const carried: LocationQuery = {}
  for (const [k, v] of Object.entries(current)) {
    if (!MANAGED_PARAMS.has(k)) carried[k] = v
  }
  return { ...carried, ...toQuery(patch) }
}

export interface ViewNav {
  state: Readonly<SelectionState>
  screen: Readonly<Ref<MobileScreen>>
  selectFeed: (id: number | null) => void
  selectItem: (id: number | null) => void
  clear: () => void
  /** Subscribe to feed scope changes (item-only changes do not fire). */
  onScopeChange: (fn: () => void) => void
  push: (patch: Partial<ViewState>) => Promise<void>
  replace: (patch: Partial<ViewState>) => Promise<void>
  back: (fallback: Partial<ViewState>) => void
}

// The browser URL is the single source of truth for the current scope/article
// and the mobile screen, giving working back/forward navigation and deep-linkable,
// shareable views. Scope and mobile-screen changes push a history entry (back
// returns to the previous scope/screen); item-only changes replace the current
// entry so paging through articles doesn't flood history. Shared across the
// store and every pane so selection reads/writes converge on one instance.
export const useViewNav = createSharedComposable((): ViewNav => {
  const route = useRoute()
  const router = useRouter()

  const state = reactive<SelectionState>({ feedId: null, itemId: null })
  let onScope: (() => void) | null = null

  // ?add_feed is a transient subscribe-entry parameter handled by FeedTree when
  // it mounts (it opens the dialog pre-filled, then clears the param). While it
  // is present the URL-nav layer stands aside: it must not rewrite the URL (and
  // drop add_feed) nor apply selection from a URL that has no scope yet. Both
  // watcher A (URL -> state) and selectItem (state -> URL) honour this, so the
  // layer converges to a fixed point rather than looping.
  const hasAddFeed = () => route.query.add_feed != null
  // Selection and the mobile screen live only on the home route's URL. When the
  // user navigates to another route (/stats, …), the sync watchers must stand
  // aside: they must not clear the selection, fire the scope callback, or
  // rewrite that route's URL. State is re-synced on return to home.
  const isHome = () => route.path === '/'

  // Set while a handler-initiated navigation (push/replace/back) is in flight.
  // vue-router cancels an overlapping second navigation: after a push,
  // selectItem's replace would cancel that push and turn it into an in-place
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
  const isMobile = useMediaQuery('(max-width: 768px)')

  function buildQuery(patch: Partial<ViewState> = {}): LocationQuery {
    return mergeQuery(route.query, {
      feedId: state.feedId,
      itemId: state.itemId,
      view: screen.value,
      ...patch,
    })
  }

  // URL -> state, and keep the local screen in step with the URL's view
  // (external navigation: back/forward, deep links). Scope change is detected
  // against the current state so an already-applied scope (e.g. one just pushed
  // by a handler) doesn't reload the item list a second time.
  watch(
    () => route.query,
    (q) => {
      if (!isHome() || hasAddFeed()) return
      const v = parseView(q)
      screen.value = v.view
      const scopeChanged = v.feedId !== state.feedId
      state.feedId = v.feedId
      state.itemId = v.itemId
      if (scopeChanged) onScope?.()
    },
    { immediate: true },
  )

  function commit(next: Partial<SelectionState>, scopeChanged: boolean) {
    Object.assign(state, next)
    if (scopeChanged) onScope?.()
  }

  async function push(patch: Partial<ViewState>): Promise<void> {
    if (patch.view != null) screen.value = patch.view
    navPending = true
    await router.push({ query: buildQuery(patch) })
  }
  async function replace(patch: Partial<ViewState>): Promise<void> {
    if (patch.view != null) screen.value = patch.view
    navPending = true
    await router.replace({ query: buildQuery(patch) })
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

  // Selecting a scope is one gesture: commit the selection, then open the list.
  // On mobile, picking a *different* feed first swaps the current entry for the
  // feeds screen (back from the list returns to feeds); re-selecting the current
  // feed, e.g. its name in the reader, skips that detour. Desktop just pushes
  // the new scope — its list is always visible.
  async function openScope(previousFeedId: number | null) {
    const feedId = state.feedId
    const scope: Partial<ViewState> = { feedId }

    if (isMobile.value && previousFeedId !== feedId) {
      // Await before the list push below: two synchronous navigations would have
      // vue-router cancel the first, dropping feedId from the feeds entry so a
      // browser-back re-selected the previous scope instead of this feed.
      await replace({ ...scope, view: 'feeds' })
    }

    const patch: Partial<ViewState> = { ...scope }
    if (isMobile.value) patch.view = 'list'
    await push(patch)
  }

  function selectFeed(id: number | null) {
    const previousFeedId = state.feedId
    commit({ feedId: id }, true)
    openScope(previousFeedId)
  }
  function selectItem(id: number | null) {
    commit({ itemId: id }, false)
    // Item-only changes replace the current URL entry (paging through articles
    // doesn't grow history). Stand aside while a navigation is in flight so our
    // replace can't cancel it, and while add_feed is present.
    if (!isHome() || hasAddFeed() || navPending) return
    router.replace({ query: buildQuery({ itemId: id }) })
  }
  function clearView() {
    const previousFeedId = state.feedId
    commit({ feedId: null, itemId: null }, true)
    openScope(previousFeedId)
  }

  return {
    state: readonly(state),
    screen: readonly(screen),
    selectFeed,
    selectItem,
    clear: clearView,
    onScopeChange: (fn) => {
      onScope = fn
    },
    push,
    replace,
    back,
  }
})
