<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import type { SeedanceVideoPrices } from '@/types'

const props = defineProps<{ modelValue: SeedanceVideoPrices }>()
const emit = defineEmits<{ 'update:modelValue': [value: SeedanceVideoPrices] }>()
const { t } = useI18n()
const key = (name: string) => `admin.groups.seedancePricing.${name}`
const modelName = ref('')
const error = ref('')
const baseResolutions = ['480p', '720p', '1080p']
const newResolution = ref('')
const customResolutions = ref<string[]>([])
const resolutions = computed(() => [...new Set([...baseResolutions, ...customResolutions.value, ...Object.values(props.modelValue).flatMap(modes => Object.values(modes).flatMap(tiers => Object.keys(tiers)))])].sort((a,b) => baseResolutions.indexOf(a) < 0 ? (baseResolutions.indexOf(b) < 0 ? a.localeCompare(b) : 1) : (baseResolutions.indexOf(b) < 0 ? -1 : baseResolutions.indexOf(a)-baseResolutions.indexOf(b))))
const modes = ['token', 'video', 'per_request']
const units: Record<string, string> = { token: '$/MToken', video: '$/s', per_request: '$/request' }
const rows = computed(() => Object.entries(props.modelValue).map(([model, prices]) => ({ model, mode: Object.keys(prices)[0] ?? 'token', prices })))
function add() {
 const model = modelName.value.trim()
 if (!model) { error.value = t(key('required')); return }
 if (props.modelValue[model]) { error.value = t(key('duplicate')); return }
 const tiers = Object.fromEntries(resolutions.value.map(r => [r, { none: null, input: null }]))
 // Null drafts are required inputs and cannot be submitted by the enclosing form.
 emit('update:modelValue', { ...props.modelValue, [model]: { token: tiers } } as unknown as SeedanceVideoPrices)
 modelName.value = ''
 error.value = ''
}
function addResolution() {
 const resolution = newResolution.value.trim()
 if (!resolution || !/^[a-zA-Z0-9][a-zA-Z0-9._-]{0,31}$/.test(resolution)) { error.value = t(key('resolutionInvalid')); return }
 if (resolutions.value.includes(resolution)) { error.value = t(key('resolutionDuplicate')); return }
 customResolutions.value.push(resolution)
 const copy: any = JSON.parse(JSON.stringify(props.modelValue))
 for (const modes of Object.values(copy) as any[]) for (const tiers of Object.values(modes) as any[]) tiers[resolution] = { none: null, input: null }
 emit('update:modelValue', copy)
 newResolution.value = ''; error.value = ''
}
function removeResolution(resolution: string) {
 if (baseResolutions.includes(resolution)) return
 customResolutions.value = customResolutions.value.filter(item => item !== resolution)
 const copy: any = JSON.parse(JSON.stringify(props.modelValue))
 for (const modes of Object.values(copy) as any[]) for (const tiers of Object.values(modes) as any[]) delete tiers[resolution]
 emit('update:modelValue', copy)
}
function remove(model: string) {
 const copy = { ...props.modelValue }
 delete copy[model]
 emit('update:modelValue', copy)
}
function changeMode(model: string, event: Event) {
 const mode = (event.target as HTMLSelectElement).value
 const current = props.modelValue[model]
 // Unit changes require explicit new prices to avoid reusing a token price as a per-request price.
 if (mode === Object.keys(current)[0]) return
 const tiers = Object.fromEntries(resolutions.value.map(r => [r, { none: null, input: null }]))
 emit('update:modelValue', { ...props.modelValue, [model]: { [mode]: tiers } } as unknown as SeedanceVideoPrices)
}
function price(model: string, mode: string, resolution: string, material: string, event: Event) {
 const raw = (event.target as HTMLInputElement).value
 const copy: any = JSON.parse(JSON.stringify(props.modelValue))
 copy[model][mode][resolution] ??= { none: null, input: null }
 copy[model][mode][resolution][material] = raw === '' ? null : Number(raw)
 emit('update:modelValue', copy)
}
</script>

<template>
 <div class="space-y-3" data-testid="seedance-pricing-editor">
  <p class="font-medium text-gray-700 dark:text-gray-300">{{ t(key('title')) }}</p>
  <p class="text-xs text-gray-500">{{ t(key('hint')) }}</p>
  <div class="flex items-end gap-2">
   <label class="min-w-0 flex-1"><span class="input-label">{{ t(key('model')) }}</span><input v-model="modelName" class="input" placeholder="seedance-2.0" @keydown.enter.prevent="add" /></label>
   <button type="button" class="btn btn-secondary" @click="add">{{ t(key('add')) }}</button>
  </div>
  <p v-if="error" class="text-sm text-red-500" role="alert">{{ error }}</p>
  <p v-if="!rows.length" class="text-xs text-gray-500">{{ t(key('empty')) }}</p>
  <div v-for="row in rows" :key="row.model" class="space-y-3 rounded-lg border border-gray-200 p-3 dark:border-dark-700">
   <div class="flex flex-wrap items-center gap-3">
    <span class="min-w-0 flex-1 break-all font-mono text-sm">{{ row.model }}</span>
    <label class="flex items-center gap-2 text-sm">{{ t(key('mode')) }}
     <select class="input w-auto" :value="row.mode" @change="changeMode(row.model, $event)"><option v-for="mode in modes" :key="mode" :value="mode">{{ t(key(mode)) }} ({{ units[mode] }})</option></select>
    </label>
    <button type="button" class="text-sm text-red-500" @click="remove(row.model)">{{ t(key('remove')) }}</button>
   </div>
   <div class="grid grid-cols-[auto_1fr_1fr] items-center gap-2 text-sm">
    <span>{{ t(key('resolution')) }}</span><span>{{ t(key('none')) }} ({{ units[row.mode] }})</span><span>{{ t(key('input')) }} ({{ units[row.mode] }})</span>
    <template v-for="resolution in resolutions" :key="resolution">
     <span class="flex items-center gap-1">{{ resolution }}<button v-if="!baseResolutions.includes(resolution)" type="button" class="text-red-500" :aria-label="`${resolution} ${t(key('removeResolution'))}`" @click="removeResolution(resolution)">×</button></span>
     <input v-for="material in ['none', 'input']" :key="material" :value="row.prices[row.mode]?.[resolution]?.[material]" :aria-label="`${row.model} ${resolution} ${t(key(material))} ${units[row.mode]}`" class="input" type="number" min="0" step="any" @input="price(row.model, row.mode, resolution, material, $event)" />
    </template>
   </div>
  </div>
  <div class="flex items-end gap-2">
   <label class="min-w-0 flex-1"><span class="input-label">{{ t(key('customResolution')) }}</span><input v-model="newResolution" class="input" placeholder="1440p" @keydown.enter.prevent="addResolution" /></label>
   <button type="button" class="btn btn-secondary" @click="addResolution">{{ t(key('addResolution')) }}</button>
  </div>
 </div>
</template>
