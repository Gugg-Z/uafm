import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
const mock = vi.hoisted(() => ({ get: vi.fn(), error: vi.fn() }))
vi.mock('vue-i18n', async (importOriginal) => ({ ...await importOriginal<typeof import('vue-i18n')>(), useI18n: () => ({ t: (key: string) => key }) }))
vi.mock('@/api/auth', () => ({ getSupportContactConfig: mock.get }))
vi.mock('@/components/layout/AppLayout.vue', () => ({ default: { template: '<div><slot /></div>' } }))
vi.mock('@/stores/app', () => ({ useAppStore: () => ({ publicSettingsLoaded: true, showError: mock.error }) }))
import SupportContactView from '../SupportContactView.vue'
function render() { return mount(SupportContactView, { global: { stubs: { AppLayout: { template: '<div><slot /></div>' }, Icon: true } } }) }
describe('SupportContactView', () => {
 beforeEach(() => { vi.clearAllMocks() })
 it('shows contacts with QR images and working hours', async () => {
  mock.get.mockResolvedValue({enabled:true,title:'Help',description:'Contact us',contacts:[{name:'Agent',work_hours:'09:00-18:00',qr_image:'/qr.png'}, {name:'No image',qr_image:''}]})
  const wrapper=render(); await flushPromises()
  expect(wrapper.text()).toContain('Agent'); expect(wrapper.text()).toContain('09:00-18:00')
  expect(wrapper.findAll('img')).toHaveLength(1); expect(wrapper.get('img').attributes('src')).toBe('/qr.png')
  wrapper.unmount()
 })
 it('hides contact cards when disabled', async () => {
  mock.get.mockResolvedValue({enabled:false,contacts:[{name:'Agent',qr_image:'/qr.png'}]})
  const wrapper=render(); await flushPromises()
  expect(wrapper.text()).toContain('supportContact.disabledTitle'); expect(wrapper.find('img').exists()).toBe(false)
  wrapper.unmount()
 })
 it('handles failed loading without an unhandled rejection', async () => {
  mock.get.mockRejectedValue(new Error('offline')); const wrapper=render(); await flushPromises()
  expect(mock.error).toHaveBeenCalledWith('supportContact.loadFailed'); wrapper.unmount()
 })
})
