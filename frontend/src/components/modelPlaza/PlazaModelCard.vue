<template>
  <article class="group relative overflow-hidden rounded-2xl border border-gray-200/80 bg-white p-5 shadow-sm transition hover:-translate-y-0.5 hover:border-primary-300 hover:shadow-lg dark:border-dark-700 dark:bg-dark-800/80 dark:hover:border-primary-500/60">
    <div class="flex items-start justify-between gap-3">
      <div class="min-w-0">
        <div class="flex flex-wrap items-center gap-2">
          <PlatformIcon :platform="model.platform as GroupPlatform" size="md" :class="platformIconClass(model.platform)" />
          <h3 class="truncate text-base font-semibold text-gray-950 dark:text-white">{{ model.name }}</h3>
          <span class="rounded-md px-2 py-0.5 text-[10px] font-semibold" :class="platformBadgeLightClass(model.platform)">{{ platformLabel(model.platform) }}</span>
        </div>
        <p class="mt-1 text-xs text-gray-400 dark:text-dark-400">{{ billingLabel }}</p>
      </div>
      <span v-if="discountPercent != null" class="shrink-0 rounded-full border px-2.5 py-1 text-xs font-bold" :class="discountBadgeClass">{{ t('modelPlaza.table.discountBadge', { percent: discountPercent }) }}</span>
    </div>

    <div class="mt-4 space-y-1.5">
      <PlazaPriceMetric :label="t('modelPlaza.table.input')" :paid="paidInput" :official="officialInput" :unit="unit" primary />
      <PlazaPriceMetric :label="t('modelPlaza.table.output')" :paid="paidOutput" :official="officialOutput" :unit="unit" />
      <PlazaPriceMetric :label="t('modelPlaza.table.cacheWrite')" :paid="paidCacheWrite" :official="officialCacheWrite" :unit="unit" />
      <PlazaPriceMetric :label="t('modelPlaza.table.cacheRead')" :paid="paidCacheRead" :official="officialCacheRead" :unit="unit" />
    </div>

    <div class="mt-4 flex items-center justify-between border-t border-gray-100 pt-3 text-xs dark:border-dark-700">
      <span class="font-mono tabular-nums text-gray-500 dark:text-dark-400">{{ t('modelPlaza.table.output') }} {{ paidOutput }}</span>
      <span class="font-semibold text-primary-600 dark:text-primary-300">{{ effectiveRate }}x</span>
    </div>
  </article>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { formatScaled } from '@/utils/pricing'
import { platformBadgeLightClass, platformIconClass, platformLabel } from '@/utils/platformColors'
import { BILLING_MODE_IMAGE, BILLING_MODE_TOKEN, type BillingMode } from '@/constants/channel'
import type { PlazaModel } from '@/api/modelPlaza'
import PlazaPriceMetric from './PlazaPriceMetric.vue'
import PlatformIcon from '@/components/common/PlatformIcon.vue'
import type { GroupPlatform } from '@/types'


const props = defineProps<{ model: PlazaModel; rateMultiplier: number; userRateMultiplier?: number | null; imageRateIndependent?: boolean; imageRateMultiplier?: number | null }>()
const { t } = useI18n()
const rate = computed(() => props.userRateMultiplier ?? props.rateMultiplier)
const pricing = computed(() => props.model.pricing)
const mode = computed(() => (pricing.value?.billing_mode || BILLING_MODE_TOKEN) as BillingMode)
const unit = computed(() => mode.value === BILLING_MODE_TOKEN ? '$ / 1M tokens' : mode.value === BILLING_MODE_IMAGE ? '/ image' : '/ request')
const effectiveRate = computed(() => mode.value === BILLING_MODE_IMAGE && props.imageRateIndependent ? (props.imageRateMultiplier ?? 1) : rate.value)
const paid = (v: number | null | undefined) => v == null ? '-' : formatScaled(v * effectiveRate.value, mode.value === BILLING_MODE_TOKEN ? 1_000_000 : 1, 2)
const formatOfficial = (v: number | null | undefined) => v == null ? '-' : formatScaled(v, 1_000_000, 2)
const paidInput = computed(() => paid(pricing.value?.input_price))
const paidOutput = computed(() => paid(pricing.value?.output_price))
const paidCacheWrite = computed(() => paid(pricing.value?.cache_write_price))
const paidCacheRead = computed(() => paid(pricing.value?.cache_read_price))
const officialInput = computed(() => formatOfficial(props.model.official_pricing?.input_price))
const officialOutput = computed(() => formatOfficial(props.model.official_pricing?.output_price))
const officialCacheWrite = computed(() => formatOfficial(props.model.official_pricing?.cache_write_price))
const officialCacheRead = computed(() => formatOfficial(props.model.official_pricing?.cache_read_price))
const billingLabel = computed(() => mode.value === BILLING_MODE_TOKEN ? 'Token billing' : mode.value === BILLING_MODE_IMAGE ? 'Image billing' : 'Per-request billing')
const discountPercent = computed(() => {
  const official = props.model.official_pricing?.input_price
  const actual = pricing.value?.input_price
  if (mode.value !== BILLING_MODE_TOKEN || official == null || actual == null || official <= 0) return null
  const discount = 1 - (actual * effectiveRate.value) / official
  return discount > 0.001 ? Math.round(discount * 100) : null
})
// 折扣深浅用色阶表达:浅灰 → emerald → 高幅橙,避免所有折扣长得一样。
const DISCOUNT_TIERS = [
  [20, 'border-gray-200 bg-gray-50 text-gray-600 dark:border-dark-600 dark:bg-dark-700/60 dark:text-dark-300'],
  [40, 'border-emerald-200 bg-emerald-50 text-emerald-700 dark:border-emerald-800 dark:bg-emerald-900/25 dark:text-emerald-300'],
] as const
const discountBadgeClass = computed(() => {
  const percent = discountPercent.value ?? 0
  return DISCOUNT_TIERS.find(([max]) => percent < max)?.[1]
    ?? 'border-transparent bg-gradient-to-r from-amber-400 to-orange-500 text-white shadow-sm'
})
</script>

