<template>
  <section class="space-y-3 rounded-lg border border-gray-200 bg-gray-50/70 p-4 dark:border-dark-600 dark:bg-dark-800/50">
    <div class="flex flex-wrap items-start justify-between gap-3">
      <div>
        <label class="input-label mb-1">{{ t('admin.users.form.adminMenuPermissions') }}</label>
        <p class="text-xs text-gray-500 dark:text-dark-400">
          {{ t('admin.users.form.adminMenuPermissionsHint') }}
        </p>
      </div>
      <div class="flex gap-2">
        <button type="button" class="btn btn-secondary px-3 py-1 text-xs" @click="selectAll">
          {{ t('common.selectAll') }}
        </button>
        <button type="button" class="btn btn-secondary px-3 py-1 text-xs" @click="clearAll">
          {{ t('common.clear') }}
        </button>
      </div>
    </div>

    <div class="grid grid-cols-1 gap-2 sm:grid-cols-2">
      <label
        v-for="option in adminOptions"
        :key="option.key"
        class="flex cursor-pointer items-center gap-2 rounded-md border border-gray-200 bg-white px-3 py-2 text-sm text-gray-700 transition-colors hover:border-primary-300 hover:bg-primary-50/50 dark:border-dark-600 dark:bg-dark-700 dark:text-dark-200 dark:hover:border-primary-500/50 dark:hover:bg-primary-900/20"
      >
        <input
          type="checkbox"
          class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500"
          :checked="selectedSet.has(option.key)"
          @change="toggle(option.key)"
        />
        <span>{{ option.label }}</span>
      </label>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { ADMIN_MENU_ITEMS, normalizeAdminMenuPermissions, type AdminMenuItem } from '@/utils/adminMenuPermissions'

const props = defineProps<{
  modelValue: string[]
}>()
const emit = defineEmits<{ 'update:modelValue': [value: string[]] }>()
const { t } = useI18n()

const adminLabelKeys: Record<AdminMenuItem, string> = {
  admin_dashboard: 'nav.dashboard',
 admin_visitor_analytics: 'nav.visitorAnalytics',
  admin_download_resources: 'nav.downloadResources',
  admin_ops: 'nav.ops',
  admin_requests: 'nav.requests',
  admin_users: 'nav.users',
  admin_groups: 'nav.groups',
  admin_channel_pricing: 'nav.channelPricing',
  admin_channel_monitor: 'nav.channelMonitor',
  admin_subscriptions: 'nav.subscriptions',
  admin_accounts: 'nav.accounts',
  admin_announcements: 'nav.announcements',
  admin_proxies: 'nav.proxies',
  admin_plugins: 'nav.plugins',
  admin_risk_control: 'nav.riskControl',
  admin_prompt_audit: 'nav.promptAudit',
  admin_audit_logs: 'nav.auditLogs',
  admin_redeem: 'nav.redeemCodes',
  admin_promo_codes: 'nav.promoCodes',
  admin_affiliate_usage: 'nav.affiliateUsage',
  admin_affiliate_applications: 'nav.affiliateApplications',
  admin_affiliate_invites: 'nav.affiliateInviteRecords',
  admin_affiliate_rebates: 'nav.affiliateRebateRecords',
  admin_affiliate_transfers: 'nav.affiliateTransferRecords',
  admin_order_dashboard: 'nav.paymentDashboard',
  admin_orders: 'nav.orderManagement',
  admin_order_plans: 'nav.paymentPlans',
  admin_usage: 'nav.usage',
  admin_settings: 'nav.settings',
}

const adminOptions = computed(() => ADMIN_MENU_ITEMS.map((key) => ({ key, label: t(adminLabelKeys[key]) })))
const selectedSet = computed(() => {
  const allowed = new Set<string>(ADMIN_MENU_ITEMS)
  return new Set(normalizeAdminMenuPermissions(props.modelValue).filter((key) => allowed.has(key)))
})

function toggle(key: AdminMenuItem) {
  const next = new Set(selectedSet.value)
  if (next.has(key)) next.delete(key)
  else next.add(key)
  emit('update:modelValue', Array.from(next))
}

function selectAll() {
  emit('update:modelValue', [...ADMIN_MENU_ITEMS])
}

function clearAll() {
  emit('update:modelValue', [])
}
</script>
