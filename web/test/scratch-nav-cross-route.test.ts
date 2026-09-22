import { it, expect, vi } from 'vitest'
import { createApp } from 'vue'

// Same browser stubs as router.test.ts, plus a record of history mutations so
// we can see exactly which navigations reach window.history.
const rec = vi.hoisted(() => {
  const mutations: string[] = []
  const location = {
    href: 'http://localhost/',
    protocol: 'http:',
    origin: 'http://localhost',
    host: 'localhost',
    hostname: 'localhost',
    port: '',
    pathname: '/',
    search: '',
    hash: '',
  }
  const history = {
    length: 1,
    state: null as unknown,
    pushState(s: unknown, _t: string, url?: string | URL | null) {
      mutations.push(`push ${url}`)
      this.state = s
    },
    replaceState(s: unknown, _t: string, url?: string | URL | null) {
      mutations.push(`replace ${url}`)
      this.state = s
    },
    go() {},
    back() {},
    forward() {},
  }
  const makeElement = () => ({
    style: {},
    nodeType: 1,
    childNodes: [],
    parentNode: null,
    nextSibling: null,
    setAttribute() {},
    getAttribute: () => null,
    appendChild() {},
    insertBefore() {},
    removeChild() {},
    addEventListener() {},
    removeEventListener() {},
  })
  const document = {
    title: '',
    baseURI: location.href,
    querySelector: () => null,
    createElement: () => makeElement(),
    createTextNode: () => ({ nodeType: 3, nodeValue: '' }),
    createComment: () => ({ nodeType: 8 }),
  }
  const window = {
    location,
    history,
    addEventListener() {},
    removeEventListener() {},
    document,
  }
  globalThis.window = window as unknown as Window & typeof globalThis
  globalThis.location = location as unknown as Location
  globalThis.document = window.document as unknown as Document
  globalThis.history = history as unknown as History
  return { mutations, history }
})

vi.mock('../src/views/LoginView.vue', () => ({ default: { name: 'LoginViewStub' } }))
vi.mock('../src/views/FeedLayout.vue', () => ({ default: { name: 'FeedLayoutStub' } }))
vi.mock('../src/views/StatsView.vue', () => ({ default: { name: 'StatsViewStub' } }))
vi.mock('../src/views/NotFoundView.vue', () => ({ default: { name: 'NotFoundViewStub' } }))

import { router } from '../src/router'
import { getAuthState } from '../src/store/auth'
import { useViewNav } from '../src/composables/useViewNav'

const tick = () => new Promise((r) => setTimeout(r, 0))

it('nav watchers stand aside on other routes: selection kept, no scope load, URL untouched', async () => {
  ;(getAuthState() as { isAuthenticated: boolean }).isAuthenticated = true
  console.log('cp1: app created')
  const app = createApp({ render: () => null })
  app.use(router)
  console.log('cp2: router installed')
  await router.isReady()
  console.log('cp3: router ready', router.currentRoute.value.fullPath)
  await router.replace('/?feed=3&item=7&view=list')
  console.log('cp4: replaced', router.currentRoute.value.fullPath)
  await tick()
  console.log('cp5: ticked')

  let nav!: ReturnType<typeof useViewNav>
  app.runWithContext(() => {
    nav = useViewNav()
  })
  await tick()
  console.log('cp6: nav created', JSON.stringify({ ...nav.state }))
  expect(nav.state.feedId).toBe(3)
  expect(nav.state.itemId).toBe(7)

  let scopeLoads = 0
  nav.onScopeChange(() => {
    scopeLoads++
  })
  rec.mutations.length = 0

  // User clicks "Statistics" in the feed tree while feed 3 / item 7 are selected.
  console.log('cp7: pushing /stats')
  await router.push('/stats')
  console.log('cp8: pushed /stats', router.currentRoute.value.fullPath)
  await tick()
  await tick()
  await tick()
  console.log('cp9: settled')

  // eslint-disable-next-line no-console
  console.log('[cross-route] mutations:', rec.mutations)
  // eslint-disable-next-line no-console
  console.log('[cross-route] fullPath:', router.currentRoute.value.fullPath)
  // eslint-disable-next-line no-console
  console.log('[cross-route] state:', JSON.stringify({ ...nav.state }), 'scopeLoads:', scopeLoads)

  // On a non-home route the nav sync must not clear the selection, fire the
  // scope callback (items.load), or rewrite that route's URL.
  expect(nav.state.feedId).toBe(3) // selection preserved
  expect(nav.state.itemId).toBe(7)
  expect(scopeLoads).toBe(0) // no items.load() while on /stats
  expect(router.currentRoute.value.fullPath).toBe('/stats') // URL left untouched
  expect(rec.mutations.filter((m) => m.includes('/stats?'))).toEqual([])
})
