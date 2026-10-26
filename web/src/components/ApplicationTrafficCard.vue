<template>
  <AccordionCard section="app_traffic" padding="p-3">
    <template #title>
      <span class="flex items-center gap-1.5">
        <svg xmlns="http://www.w3.org/2000/svg" width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><rect width="7" height="7" x="3" y="3" rx="1"/><rect width="7" height="7" x="14" y="3" rx="1"/><rect width="7" height="7" x="14" y="14" rx="1"/><rect width="7" height="7" x="3" y="14" rx="1"/></svg>
        Application Traffic
      </span>
    </template>

    <div v-if="!hasActiveApps" class="card-empty py-4 text-center">
      <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75" stroke-linecap="round" stroke-linejoin="round" class="mx-auto mb-1 opacity-60"><circle cx="12" cy="12" r="10"/><line x1="12" y1="8" x2="12" y2="12"/><line x1="12" y1="16" x2="12.01" y2="16"/></svg>
      <span>No active application traffic</span>
    </div>

    <div v-else class="flex flex-col gap-2">
      <!-- Table Header -->
      <div class="grid grid-cols-[1fr_56px_56px_58px] text-[10px] uppercase font-semibold text-muted tracking-wider pb-1 border-b border-[var(--border)]">
        <span>Application</span>
        <span class="text-right text-emerald-400">↓ Down</span>
        <span class="text-right text-purple-400">↑ Up</span>
        <span class="text-right">Total</span>
      </div>

      <!-- App Rows -->
      <div class="flex flex-col gap-1.5 max-h-[220px] overflow-y-auto pr-0.5">
        <div
          v-for="app in appList"
          :key="app.pid ? `${app.pid}-${app.process}` : app.name"
          class="flex flex-col gap-1 p-1.5 rounded bg-[var(--bg-secondary)] hover:bg-[var(--bg-hover)] transition-colors text-xs"
        >
          <div class="grid grid-cols-[1fr_56px_56px_58px] items-center">
            <!-- App Name & PID -->
            <div class="flex items-center gap-1.5 min-w-0 pr-1">
              <span class="w-2 h-2 rounded-full flex-shrink-0" :class="getAppDotClass(app.process)" />
              <div class="flex flex-col min-w-0 leading-tight">
                <span class="font-medium truncate text-[var(--text)]" :title="app.path || app.name">
                  {{ app.name }}
                </span>
                <span class="text-[9px] text-muted truncate">
                  {{ app.pid ? `PID ${app.pid} · ${app.process}` : app.process }}
                </span>
              </div>
            </div>

            <!-- Down -->
            <span class="text-right font-mono text-[11px] text-emerald-400">
              {{ formatMbps(app.rx_mbps) }}
            </span>

            <!-- Up -->
            <span class="text-right font-mono text-[11px] text-purple-400">
              {{ formatMbps(app.tx_mbps) }}
            </span>

            <!-- Total -->
            <span class="text-right font-mono font-medium text-[11px] text-[var(--text)]">
              {{ formatMbps(app.total_mbps) }}
            </span>
          </div>

          <!-- Mini Percentage Bar -->
          <div v-if="app.percent > 0" class="w-full bg-[var(--border)] h-1 rounded-full overflow-hidden">
            <div
              class="h-full rounded-full transition-all duration-300"
              :class="getAppDotClass(app.process)"
              :style="{ width: `${Math.min(100, Math.max(3, app.percent))}%` }"
            />
          </div>
        </div>
      </div>

      <div class="card-divider my-0.5" />

      <!-- Footer / Summary -->
      <div class="grid grid-cols-[1fr_56px_56px_58px] text-[11px] font-medium pt-0.5">
        <span class="text-muted">Total App Traffic</span>
        <span class="text-right font-mono text-emerald-400">{{ formatMbps(totalRx) }}</span>
        <span class="text-right font-mono text-purple-400">{{ formatMbps(totalTx) }}</span>
        <span class="text-right font-mono text-[var(--text)]">{{ formatMbps(totalAll) }}</span>
      </div>

      <div class="flex items-center justify-between text-[10px] text-muted mt-0.5">
        <span :class="isAppTrafficStale ? 'text-amber-500 font-medium' : ''">
          {{ isAppTrafficStale ? '⚠️ ' + sampledAgoText : sampledAgoText }}
        </span>
        <span class="truncate text-[9px] max-w-[140px] text-muted" title="Some traffic may be unattributed because Windows/system services operate outside normal user-process attribution.">
          Per-Process Accounting
        </span>
      </div>
    </div>
  </AccordionCard>
</template>

<script setup>
import { computed } from 'vue'
import AccordionCard from './AccordionCard.vue'
import { formatTimeAgo, isStale } from '../utils/time.js'

const props = defineProps({
  traffic: Object,
  now: { type: Number, default: () => Date.now() },
})

const appList = computed(() => {
  return props.traffic?.applications || []
})

const hasActiveApps = computed(() => {
  return appList.value.length > 0 && (totalAll.value > 0.001 || appList.value.some(a => a.total_mbps > 0 || a.rx_bytes > 0 || a.tx_bytes > 0))
})

const totalRx = computed(() => props.traffic?.total_rx_mbps || 0)
const totalTx = computed(() => props.traffic?.total_tx_mbps || 0)
const totalAll = computed(() => props.traffic?.total_mbps || (totalRx.value + totalTx.value))

const isAppTrafficStale = computed(() => {
  if (!props.traffic?.timestamp) return true
  return isStale(props.traffic.timestamp, 6000, props.now)
})

const sampledAgoText = computed(() => {
  if (!props.traffic?.timestamp) return 'No sample'
  const ago = formatTimeAgo(props.traffic.timestamp, props.now)
  return isAppTrafficStale.value ? `Paused (${ago})` : `Sampled ${ago}`
})

function formatMbps(mbps) {
  if (mbps === undefined || mbps === null) return '0.00'
  if (mbps >= 100) return mbps.toFixed(0)
  if (mbps >= 10) return mbps.toFixed(1)
  return mbps.toFixed(2)
}

function getAppDotClass(proc) {
  const p = (proc || '').toLowerCase()
  if (p.includes('chrome') || p.includes('brave') || p.includes('edge') || p.includes('firefox')) {
    return 'bg-blue-500'
  }
  if (p.includes('code') || p.includes('antigravity') || p.includes('gopls') || p.includes('node') || p.includes('python')) {
    return 'bg-emerald-500'
  }
  if (p.includes('discord') || p.includes('slack') || p.includes('teams') || p.includes('telegram')) {
    return 'bg-purple-500'
  }
  if (p.includes('spotify') || p.includes('steam')) {
    return 'bg-pink-500'
  }
  if (p.includes('netmon')) {
    return 'bg-cyan-500'
  }
  return 'bg-amber-500'
}
</script>
