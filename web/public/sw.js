// Caches the static app shell so an installed TinyRSS opens offline. API
// responses are never cached: they are token-authed and change constantly.
const CACHE = 'tinyrss-shell-v3'
const SHELL = ['/', '/index.html', '/favicon.svg', '/manifest.webmanifest', '/icon.svg']

async function cacheAndReturn(key, response) {
  const cache = await caches.open(CACHE)
  await cache.put(key, response.clone())
  return response
}

self.addEventListener('install', (event) => {
  event.waitUntil(
    caches
      .open(CACHE)
      // Individually, so one unreachable shell file cannot fail the install.
      .then((cache) => Promise.all(SHELL.map((path) => cache.add(path).catch(() => {}))))
      .then(() => self.skipWaiting()),
  )
})

self.addEventListener('activate', (event) => {
  event.waitUntil(
    caches
      .keys()
      .then((names) => Promise.all(names.filter((name) => name !== CACHE).map((name) => caches.delete(name))))
      .then(() => self.clients.claim()),
  )
})

self.addEventListener('fetch', (event) => {
  const { request } = event
  if (request.method !== 'GET') return
  const url = new URL(request.url)
  if (url.origin !== self.location.origin) return
  if (url.pathname.startsWith('/api/') || url.pathname === '/sw.js') return

  if (request.mode === 'navigate') {
    event.respondWith(
      fetch(request)
        // Keyed on the shell, not the route: every history route resolves to index.html.
        .then((response) => (response.ok ? cacheAndReturn('/index.html', response) : response))
        .catch(() => caches.match('/index.html').then((cached) => cached ?? Response.error())),
    )
    return
  }

  if (url.pathname.startsWith('/assets/')) {
    event.respondWith(
      caches.match(request).then(
        (cached) =>
          cached ??
          fetch(request).then((response) => (response.ok ? cacheAndReturn(request, response) : response)),
      ),
    )
    return
  }

  event.respondWith(
    fetch(request)
      .then((response) => (response.ok ? cacheAndReturn(request, response) : response))
      .catch(() => caches.match(request).then((cached) => cached ?? Response.error())),
  )
})
