import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { request, requestText, configureAuth } from '../src/api/client'

describe('api client', () => {
  const fakeFetch = vi.fn()
  let unauthorizedCalls = 0

  beforeEach(() => {
    unauthorizedCalls = 0
    fakeFetch.mockReset()
    vi.stubGlobal('fetch', fakeFetch)
    configureAuth({
      getToken: () => 'tok-123',
      onUnauthorized: () => {
        unauthorizedCalls++
      },
    })
  })

  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('attaches the Bearer token and returns parsed JSON', async () => {
    fakeFetch.mockResolvedValueOnce(
      new Response(JSON.stringify({ id: 1, name: 'x' }), { status: 200, headers: { 'Content-Type': 'application/json' } }),
    )
    const res = await request('GET', '/api/feeds')
    expect(res).toEqual({ id: 1, name: 'x' })
    const [, init] = fakeFetch.mock.calls[0] as unknown as [unknown, { headers: Record<string, string>; body: string }]
    expect(init.headers['Authorization']).toBe('Bearer tok-123')
  })

  it('sends a JSON body for POST', async () => {
    fakeFetch.mockResolvedValueOnce(new Response('{"id":2}', { status: 200, headers: { 'Content-Type': 'application/json' } }))
    await request('POST', '/api/feeds', { feed_url: 'http://x/rss' })
    const [, init] = fakeFetch.mock.calls[0] as unknown as [unknown, { body: string }]
    expect(JSON.parse(init.body)).toEqual({ feed_url: 'http://x/rss' })
  })

  it('throws ApiError with the backend error message on non-2xx', async () => {
    fakeFetch.mockResolvedValueOnce(new Response(JSON.stringify({ error: 'must be a feed URL' }), { status: 422 }))
    await expect(request('POST', '/api/feeds', { feed_url: 'x' })).rejects.toMatchObject({
      status: 422,
      message: 'must be a feed URL',
    })
  })

  it('returns undefined for 204', async () => {
    fakeFetch.mockResolvedValueOnce(new Response(null, { status: 204 }))
    await expect(request('DELETE', '/api/feeds/5')).resolves.toBeUndefined()
  })

  it('clears auth and throws on 401', async () => {
    fakeFetch.mockResolvedValueOnce(new Response(JSON.stringify({ error: 'unauthorized' }), { status: 401 }))
    await expect(request('GET', '/api/items')).rejects.toMatchObject({ status: 401, message: 'unauthorized' })
    expect(unauthorizedCalls).toBe(1)
  })

  it('requestText returns the raw body as text and attaches the token', async () => {
    const opml = '<?xml version="1.0"?><opml version="2.0"><body/></opml>'
    fakeFetch.mockResolvedValueOnce(new Response(opml, { status: 200 }))
    await expect(requestText('GET', '/api/opml/export')).resolves.toBe(opml)
    const [, init] = fakeFetch.mock.calls[0] as unknown as [unknown, { headers: Record<string, string> }]
    expect(init.headers['Authorization']).toBe('Bearer tok-123')
  })

  it('requestText throws ApiError on non-2xx', async () => {
    fakeFetch.mockResolvedValueOnce(new Response(JSON.stringify({ error: 'boom' }), { status: 500 }))
    await expect(requestText('GET', '/api/opml/export')).rejects.toMatchObject({ status: 500, message: 'boom' })
  })
})
