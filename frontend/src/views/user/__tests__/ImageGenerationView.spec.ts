import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import { ref } from 'vue'
vi.mock('vue-i18n', async (importOriginal) => ({
  ...await importOriginal<typeof import('vue-i18n')>(),
  useI18n: () => ({ t: (key: string) => key, locale: ref('en') })
}))

const api = vi.hoisted(() => ({ list: vi.fn(), generate: vi.fn(), listModels: vi.fn() }))
vi.mock('@/api', () => ({ keysAPI: { list: api.list }, imageGenerationAPI: api }))
vi.mock('@/stores/app', () => ({ useAppStore: () => ({ showError: vi.fn(), showSuccess: vi.fn() }) }))
import ImageGenerationView from '../ImageGenerationView.vue'

function render() {
  return mount(ImageGenerationView, { global: {
    stubs: { AppLayout: { template: '<div><slot /></div>' }, RouterLink: true, Icon: true, HelpTooltip: true }
  } })
}

describe('ImageGenerationView', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    localStorage.clear()
    api.list.mockResolvedValue({ items: [{ id: 1, name: 'Image key', key: 'sk-test', status: 'active', group: {
      name: 'Images', platform: 'gemini', allow_image_generation: true, image_price_1k: null
    } }], pages: 1 })
    api.listModels.mockResolvedValue(['gemini-3-pro-image-preview'])
    api.generate.mockResolvedValue({ images: [{ url: 'data:image/png;base64,aW1hZ2U=', mimeType: 'image/png' }] })
  })

  it('uses the discovered preview model ID and displays a generated image', async () => {
    const wrapper = render()
    await flushPromises()
    expect(wrapper.findAll('option').some(option => option.attributes('value') === 'gemini-3-pro-image-preview')).toBe(true)
    await wrapper.get('textarea').setValue('Draw a quiet mountain lake')
    const generate = wrapper.findAll('button').find(button => button.text() === 'imageGeneration.actions.generate')!
    await generate.trigger('click')
    await flushPromises()
    expect(api.generate).toHaveBeenCalledWith(expect.objectContaining({ model: 'gemini-3-pro-image-preview', platform: 'gemini', apiKey: 'sk-test' }))
    expect(wrapper.find('img[src="data:image/png;base64,aW1hZ2U="]').exists()).toBe(true)
    wrapper.unmount()
  })

  it('does not offer preset models when discovery returns no image models', async () => {
    api.listModels.mockResolvedValue(['gemini-text-model'])
    const wrapper = render()
    await flushPromises()
    expect(wrapper.findAll('button').find(button => button.text() === 'imageGeneration.actions.generate')!.attributes('disabled')).toBeDefined()
    wrapper.unmount()
  })
})
