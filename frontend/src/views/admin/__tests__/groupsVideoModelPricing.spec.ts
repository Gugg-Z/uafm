import { describe, expect, it } from 'vitest'

import {
  createVideoModelPricesForm,
  addVideoModelPriceRow,
  serializeVideoModelPrices,
  videoModelPriceFamilyRows
} from '../groupsVideoModelPricing'

describe('Grok video model pricing form', () => {
  it('provides editable rows for both canonical Grok video families', () => {
    const form = createVideoModelPricesForm()

    expect(videoModelPriceFamilyRows(form).map(({ key }) => key)).toEqual([
      'grok-imagine-video',
      'grok-imagine-video-1.5'
    ])
    expect(form['grok-imagine-video']['480p']).toBeNull()
    expect(form['grok-imagine-video-1.5']['1080p']).toBeNull()
  })

  it('serializes only finite non-negative prices and preserves future families', () => {
    const form = createVideoModelPricesForm({
      'custom-video': { '1080p': 0.4 }
    })
    form['grok-imagine-video']['480p'] = 0.05
    form['grok-imagine-video']['720p'] = ''
    form['grok-imagine-video-1.5']['1080p'] = -1

    expect(serializeVideoModelPrices(form)).toEqual({
      'grok-imagine-video': { '480p': 0.05 },
      'custom-video': { '1080p': 0.4 }
    })
  })

  it('round-trips unknown model families so editing does not discard them', () => {
    const form = createVideoModelPricesForm({
      'custom-video': { '480p': 0.2 }
    })

    expect(videoModelPriceFamilyRows(form).map(({ key }) => key)).toContain(
      'custom-video'
    )
    expect(serializeVideoModelPrices(form)).toMatchObject({
      'custom-video': { '480p': 0.2 }
    })
  })
})

it('adds a custom model without replacing configured resolution prices', () => {
  const form = createVideoModelPricesForm({ 'custom-video': { '720p': 0 } })
  expect(addVideoModelPriceRow(form, ' Custom-Video ')).toBe(true)
  expect(form['custom-video']['720p']).toBe(0)
  expect(addVideoModelPriceRow(form, 'provider-video')).toBe(true)
  form['provider-video']['1080p'] = 0.25
  expect(serializeVideoModelPrices(form)['provider-video']).toEqual({ '1080p': 0.25 })
  expect(addVideoModelPriceRow(form, '  ')).toBe(false)
})
