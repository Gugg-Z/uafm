import { describe, expect, it, vi, beforeEach } from 'vitest'
import { installVisitorTracking, normalizeAttributionCode, resolveAttribution, sanitizeTrackedURL, shouldTrackPath } from '../visitorTracking'
import type { Router } from 'vue-router'

describe('visitorTracking', () => {
  beforeEach(() => {
    localStorage.clear()
    vi.useRealTimers()
    vi.restoreAllMocks()
    vi.unstubAllGlobals()
    sessionStorage.clear()
  })

  it('normalizes safe channel codes and rejects arbitrary values', () => {
    expect(normalizeAttributionCode(' Discord_Main ')).toBe('discord_main')
    expect(normalizeAttributionCode('https://example.com')).toBe('')
  })

  it('persists first-party attribution across navigation', () => {
    expect(resolveAttribution('?ref=reddit')).toBe('reddit')
    expect(resolveAttribution('')).toBe('reddit')
  })

  it('does not collect administrator or setup page views', () => {
    expect(shouldTrackPath('/home')).toBe(true)
    expect(shouldTrackPath('/login')).toBe(true)
    expect(shouldTrackPath('/admin/dashboard')).toBe(false)
    expect(shouldTrackPath('/setup')).toBe(false)
    expect(shouldTrackPath('/auth/callback')).toBe(false)
  })

  it('expires channel attribution after 30 days', () => {
    vi.useFakeTimers()
    resolveAttribution('?ref=reddit')
    vi.advanceTimersByTime(31 * 24 * 60 * 60 * 1000)
    expect(resolveAttribution('')).toBe('direct')
    vi.useRealTimers()
  })

  it('leaves an absent referrer empty', () => {
    expect(sanitizeTrackedURL('')).toBe('')
  })

  it('sends authenticated page visits, deduplicates navigation and skips admin pages', () => {
    const fetchMock = vi.fn().mockResolvedValue({ ok: true })
    vi.stubGlobal('fetch', fetchMock)
    localStorage.setItem('auth_token', 'test-token')
    let onNavigate: (to: {path: string; fullPath: string}) => void = () => {}
    const remove = vi.fn()
    const route = {path: '/home', fullPath: '/home?ref=reddit&token=secret'}
    const router = {currentRoute: {value: route}, afterEach: (callback: typeof onNavigate) => {onNavigate=callback;return remove}} as unknown as Router
    expect(installVisitorTracking(router)).toBe(remove)
    onNavigate(route)
    onNavigate({path:'/admin/dashboard',fullPath:'/admin/dashboard'})
    expect(fetchMock).toHaveBeenCalledTimes(1)
    const [url, options] = fetchMock.mock.calls[0]
    expect(url).toBe('/api/v1/analytics/visit')
    expect(options.headers.Authorization).toBe('Bearer test-token')
    const payload = JSON.parse(options.body)
    expect(payload.channel_code).toBe('reddit')
    expect(payload.landing_url).not.toContain('secret')
    expect(payload.visitor_id).toBeTruthy()
    expect(payload.session_id).toBeTruthy()
  })

  it('does not interrupt navigation when browser storage is blocked', () => {
    vi.spyOn(Storage.prototype, 'getItem').mockImplementation(() => {throw new Error('blocked')})
    const router = {currentRoute: {value:{path:'/home',fullPath:'/home'}}, afterEach: vi.fn()} as unknown as Router
    expect(() => installVisitorTracking(router)).not.toThrow()
  })

  it('drops sensitive query parameters while preserving attribution fields', () => {
    const sanitized = new URL(sanitizeTrackedURL('https://example.com/payment/result?resume_token=secret&ref=reddit&utm_campaign=launch'))
    expect(sanitized.searchParams.get('resume_token')).toBeNull()
    expect(sanitized.searchParams.get('ref')).toBe('reddit')
    expect(sanitized.searchParams.get('utm_campaign')).toBe('launch')
  })
})
