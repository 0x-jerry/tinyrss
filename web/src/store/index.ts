import { createGlobalState } from '@vueuse/core'
import { authStore } from './auth'
import { createThemeStore } from './theme'
import { createFeedsStore } from './feeds'
import { createItemsStore } from './items'
import { useViewNav } from '../composables/useViewNav'
import { configureAuth } from '../api/client'

// One shared, lazily-initialized world state: every useStore() call returns the
// same store instance, so components read/write a single source of truth. The
// selection lives on the URL via useViewNav (shared), its deps feed the item
// store. useViewNav needs an active router, so it is bound here inside the
// createGlobalState factory, which first runs during a component's setup.
export const useStore = createGlobalState(() => {
  const theme = createThemeStore()
  const nav = useViewNav()
  const feeds = createFeedsStore()

  const items = createItemsStore({
    getSelection: () => ({ feedId: nav.state.feedId, folderId: nav.state.folderId }),
    selectItem: nav.selectItem,
    onItemsChanged: () => feeds.reload(),
    adjustUnread: (feedId, delta) => feeds.adjustUnread(feedId, delta),
  })
  // Reload the list whenever the feed/folder scope changes. No observer: the
  // nav composable invokes this from its scope-changing mutators.
  nav.onScopeChange(() => items.load())

  // auth is a self-hosted singleton (the router guard reads it before render),
  // referenced here so headers stay uniform across the store.
  return { auth: authStore, theme, feeds, items }
})

export type Store = ReturnType<typeof useStore>

// The api client reads the token off the store and clears auth on 401.
configureAuth({
  getToken: () => authStore.state.token || null,
  onUnauthorized: () => authStore.logout(),
})
