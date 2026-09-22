import { it, expect, vi } from 'vitest'
import { createApp } from 'vue'

// Browser stubs with a recording history and a matchMedia that always matches,
// so useViewNav takes the mobile branch.
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
    matchMedia: (q: string) => ({
      matches: true, // mobile viewport
      media: q,
      onchange: null,
      addEventListener() {},
      removeEventListener() {},
      addListener() {},
      removeListener() {},
      dispatchEvent: () => false,
    }),
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

it('openScope: the feeds replace lands before the list push, so back keeps the feed', async () => {
  ;(getAuthState() as { isAuthenticated: boolean }).isAuthenticated = true
  const app = createApp({ render: () => null })
  app.use(router)
  await router.isReady()
  await tick()

  let nav!: ReturnType<typeof useViewNav>
  app.runWithContext(() => {
    nav = useViewNav()
  })
  await tick()

  // Mobile: open the feeds screen (as the menu button does).
  await nav.push({ view: 'feeds' })
  await tick()
  expect(router.currentRoute.value.fullPath).toBe('/?view=feeds')

  rec.mutations.length = 0
  // Tap feed 9: openScope awaits replace({feed:9, view:'feeds'}) before pushing
  // {feed:9, view:'list'} — the feeds entry keeps feed 9, so a browser-back from
  // the list returns to feeds with feed 9 selected (not the previous scope).
  nav.selectFeed(9)
  await tick()
  await tick()
  await tick()

  // eslint-disable-next-line no-console
  console.log('[mobile] mutations after selectFeed:', rec.mutations)
  // eslint-disable-next-line no-console
  console.log('[mobile] history.state:', JSON.stringify(rec.history.state))

  // The feeds entry must carry feed 9 and land before the list push, so a
  // browser-back from the list returns to feeds with feed 9 selected (not the
  // previous scope). Vue-router may emit the same-URL replaceState more than
  // once, so assert ordering + the resulting history, not the exact array.
  const feedsReplace = 'replace http://localhost/?feed=9&view=feeds'
  const listPush = 'push http://localhost/?feed=9&view=list'
  expect(rec.mutations.indexOf(feedsReplace)).toBeGreaterThanOrEqual(0)
  expect(rec.mutations.indexOf(listPush)).toBeGreaterThan(rec.mutations.lastIndexOf(feedsReplace))
  expect(router.currentRoute.value.fullPath).toBe('/?feed=9&view=list')
  // The feeds entry carries feed 9, so back returns to feeds with it selected.
  expect((rec.history.state as { back: string }).back).toBe('/?feed=9&view=feeds')
})
