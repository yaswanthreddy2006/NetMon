<template>
  <AccordionCard section="live_traffic" padding="p-3">
    <template #title>
      <span class="flex items-center gap-1.5">
        <svg xmlns="http://www.w3.org/2000/svg" width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><path d="M13 2L3 14h9l-1 8 10-12h-9l1-8z"/></svg>
        Live Interface Traffic
      </span>
    </template>

    <div v-if="!traffic || (!traffic.interface && traffic.rx_mbps === 0 && traffic.tx_mbps === 0)" class="card-empty">
      <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><line x1="12" y1="8" x2="12" y2="12"/><line x1="12" y1="16" x2="12.01" y2="16"/></svg>
      <span>Sampling interface...</span>
    </div>

    <div v-else class="flex flex-col gap-1.5">
      <div class="card-row">
        <span class="card-row-label">Adapter</span>
        <span class="card-row-value font-medium text-blue-400">{{ traffic.interface || 'Default Interface' }}</span>
      </div>
      <div class="card-row">
        <span class="card-row-label flex items-center gap-1 text-emerald-400">
          <svg xmlns="http://www.w3.org/2000/svg" width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><line x1="12" y1="5" x2="12" y2="19"/><polyline points="19 12 12 19 5 12"/></svg>
          Live Inbound (Rx)
        </span>
        <span class="card-row-value">{{ formatSpeed(traffic.rx_mbps) }}</span>
      </div>
      <div class="card-row">
        <span class="card-row-label flex items-center gap-1 text-purple-400">
          <svg xmlns="http://www.w3.org/2000/svg" width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><line x1="12" y1="19" x2="12" y2="5"/><polyline points="5 12 12 5 19 12"/></svg>
          Live Outbound (Tx)
        </span>
        <span class="card-row-value">{{ formatSpeed(traffic.tx_mbps) }}</span>
      </div>
      <div class="card-divider" />
      <div class="flex items-center justify-between text-[10px] text-muted">
        <span :class="isTrafficStale ? 'text-amber-500 font-medium' : ''">
          {{ isTrafficStale ? '⚠️ ' + sampledAgoText : sampledAgoText }}
        </span>
        <span>{{ traffic.rx_bytes_sec > 0 ? formatBytes(traffic.rx_bytes_sec) + '/s' : '0 B/s' }}</span>
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

const isTrafficStale = computed(() => {
  if (!props.traffic?.sampled_at) return true
  return isStale(props.traffic.sampled_at, 6000, props.now)
})

const sampledAgoText = computed(() => {
  if (!props.traffic?.sampled_at) return 'No sample'
  const ago = formatTimeAgo(props.traffic.sampled_at, props.now)
  return isTrafficStale.value ? `Paused (${ago})` : `Sampled ${ago}`
})

function formatSpeed(mbps) {
  if (mbps === undefined || mbps === null) return '—'
  if (mbps >= 1000) return (mbps / 1000).toFixed(2) + ' Gbps'
  if (mbps >= 1) return mbps.toFixed(2) + ' Mbps'
  if (mbps > 0) return (mbps * 1000).toFixed(0) + ' Kbps'
  return '0.00 Mbps'
}

function formatBytes(bytesSec) {
  if (!bytesSec || bytesSec <= 0) return '0 B'
  if (bytesSec >= 1048576) return (bytesSec / 1048576).toFixed(1) + ' MB'
  if (bytesSec >= 1024) return (bytesSec / 1024).toFixed(1) + ' KB'
  return bytesSec.toFixed(0) + ' B'
}
</script>
