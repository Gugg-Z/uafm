import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import { defineComponent, ref } from 'vue'
import { createI18n } from 'vue-i18n'
import SeedancePricingEditor from '../SeedancePricingEditor.vue'
import type { SeedanceVideoPrices } from '@/types'

function editor(initial: SeedanceVideoPrices = {}) {
 const host = defineComponent({ components: { SeedancePricingEditor }, setup() { const prices = ref(initial); return { prices } }, template: '<form><SeedancePricingEditor v-model="prices" /></form>' })
 return mount(host, { global: { plugins: [createI18n({ legacy: false, locale: 'en', missingWarn: false, fallbackWarn: false, messages: { en: {} } })] } })
}
describe('Seedance pricing editor', () => {
 it('adds one model with six required prices and resets prices when units change', async () => {
  const host = editor()
  await host.find('input').setValue('seedance-2.0')
  await host.find('button').trigger('click')
  const fields = host.findAll('input[type="number"]')
  expect(fields).toHaveLength(6)
  for (let i = 0; i < fields.length; i++) await fields[i].setValue(String(i))
  expect(host.vm.prices['seedance-2.0'].token['480p']).toEqual({ none: 0, input: 1 })
  expect(host.vm.prices['seedance-2.0'].token['1080p']).toEqual({ none: 4, input: 5 })
  expect(host.text()).toContain('$/MToken')
  await host.find('select').setValue('per_request')
  expect(Object.keys(host.vm.prices['seedance-2.0'])).toEqual(['per_request'])
  expect(host.text()).toContain('$/request')
  expect(host.findAll('input[type="number"]').every(f => (f.element as HTMLInputElement).value === '')).toBe(true)
  expect(host.findAll('input[type="number"]').every(f => (f.element as HTMLInputElement).checkValidity())).toBe(true)
  await host.find('select').setValue('video')
  expect(Object.keys(host.vm.prices['seedance-2.0'])).toEqual(['video'])
  expect(host.text()).toContain('$/s')
  host.unmount()
 })
 it('round-trips stored prices and prevents duplicate model replacement', async () => {
  const initial = { custom: { video: { '480p': { none: 0, input: 1 }, '720p': { none: 2, input: 3 }, '1080p': { none: 4, input: 5 } } } }
  const host = editor(initial)
  expect(host.find('select').element.value).toBe('video')
  expect(host.findAll('input[type="number"]').map(f => (f.element as HTMLInputElement).value)).toEqual(['0','1','2','3','4','5'])
  await host.find('input').setValue(' custom ')
  await host.find('button').trigger('click')
  expect(Object.keys(host.vm.prices)).toEqual(['custom'])
  expect(host.vm.prices).toEqual(initial)
  await host.findAll('button')[1].trigger('click')
  expect(host.vm.prices).toEqual({})
  host.unmount()
 })
 it('adds an arbitrary resolution and includes it for each configured model', async () => {
  const host = editor({ first: { token: { '480p': { none: 1, input: 2 }, '720p': { none: 3, input: 4 }, '1080p': { none: 5, input: 6 } } } })
  await host.find('input[placeholder="1440p"]').setValue('1440p')
  await host.findAll('button').at(-1)!.trigger('click')
  expect(host.text()).toContain('1440p')
  const prices = host.findAll('input[type="number"]')
  expect(prices).toHaveLength(8)
  await prices[6].setValue('7')
  await prices[7].setValue('8')
  expect(host.vm.prices.first.token['1440p']).toEqual({ none: 7, input: 8 })
  host.unmount()
 })

})
