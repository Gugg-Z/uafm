<template>
  <div class="flex items-baseline justify-between gap-3 rounded-lg px-3 py-1.5" :class="primary ? 'bg-primary-50/70 dark:bg-primary-900/15' : ''">
    <p class="min-w-0 truncate text-[11px] font-medium" :class="primary ? 'text-primary-700 dark:text-primary-300' : 'text-gray-500 dark:text-dark-400'">
      {{ label }}<span class="ml-1.5 font-normal text-gray-400 dark:text-dark-500">{{ unit }}</span>
    </p>
    <p class="shrink-0 text-right leading-5">
      <span class="tabular-nums text-[11px] font-medium text-gray-400 dark:text-dark-500">官方 {{ official }}</span>
      <span class="mx-1 text-gray-300 dark:text-dark-600">→</span>
      <span class="font-mono text-sm font-bold tabular-nums text-gray-950 dark:text-white">{{ paid }}</span>
    </p>
  </div>
</template>

<script setup lang="ts">
defineProps<{ label: string; paid: string; official: string; unit: string; primary?: boolean }>()
</script>
