import { describe, it, expect, vi, beforeEach } from 'vitest'

// vue-router's createWebHistory touches window/location at import time; the
// test env is node with no DOM, so stub the few globals it reads before the
// router module is imported.
vi.hoisted(() => {
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
    state: null,
    pushState() {},
    replaceState() {},
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
})

// Stub the view components so importing the router doesn't pull in DOM-bound
// component graphs; route matching only needs the route records' names.
vi.mock('../src/views/LoginView.vue', () => ({ default: { name: 'LoginViewStub' } }))
vi.mock('../src/views/FeedLayout.vue', () => ({ default: { name: 'FeedLayoutStub' } }))
vi.mock('../src/views/NotFoundView.vue', () => ({ default: { name: 'NotFoundViewStub' } }))

import { router } from '../src/router'
import { getAuthState } from '../src/providers/auth'

beforeEach(() => {
  // The router guard bounces unauthenticated users to /login; mark us
  // authenticated so the catch-all is reachable in the test.
  getAuthState().isAuthenticated = true
})

describe('router 404 catch-all', () => {
  it('matches unmatched paths to the not-found route', () => {
    expect(router.resolve('/no/such/page').name).toBe('not-found')
  })

  it('keeps the home and login routes intact', () => {
    expect(router.resolve('/').name).toBe('home')
    expect(router.resolve('/login').name).toBe('login')
  })
})
