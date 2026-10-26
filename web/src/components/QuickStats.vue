<template>
  <AccordionCard section="summary" padding="p-3">
    <template #title>
      <span class="flex items-center gap-1.5">
        <svg xmlns="http://www.w3.org/2000/svg" width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><line x1="18" y1="20" x2="18" y2="10"/><line x1="12" y1="20" x2="12" y2="4"/><line x1="6" y1="20" x2="6" y2="14"/></svg>
        24h Summary
      </span>
    </template>
    <div v-if="!summary" class="card-empty">
      <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75" stroke-linecap="round" stroke-linejoin="round"><line x1="18" y1="20" x2="18" y2="10"/><line x1="12" y1="20" x2="12" y2="4"/><line x1="6" y1="20" x2="6" y2="14"/></svg>
      <span>No data yet</span>
    </div>
    <div v-else class="flex flex-col gap-1.5">
      <div class="card-row">
        <span class="card-row-label">Uptime</span>
        <span class="card-row-value font-semibold text-emerald-400">{{ uptimeText }}</span>
      </div>
      <div class="card-row">
        <span class="card-row-label" title="Consecutive multi-cycle failure events">Outages</span>
        <span class="card-row-value">{{ summary.outages_24h ?? 0 }}</span>
      </div>
      <div v-if="summary.failed_checks > 0" class="card-row">
        <span class="card-row-label text-muted">Failed Checks</span>
        <span class="card-row-value text-muted">{{ summary.failed_checks }}</span>
      </div>
      <div class="card-divider" />
      <div class="card-row">
        <span class="card-row-label">Peak Download</span>
        <span class="card-row-value">{{ peakDl }}</span>
      </div>
      <div class="card-row">
        <span class="card-row-label">Min Latency</span>
        <span class="card-row-value">{{ minLat }} ms</span>
      </div>
      <div class="card-row">
        <span class="card-row-label">Max Latency</span>
        <span class="card-row-value">{{ maxLat }} ms</span>
      </div>
      <div class="card-row">
        <span class="card-row-label">Avg Jitter</span>
        <span class="card-row-value">{{ summary.jitter_avg }} ms</span>
      </div>
      <div class="card-row">
        <span class="card-row-label">Packet Loss</span>
        <span class="card-row-value" :class="summary.packet_loss > 0 ? 'text-amber-400' : ''">{{ summary.packet_loss }}%</span>
      </div>
    </div>
  </AccordionCard>
</template>

<script setup>
import { computed } from 'vue'
import AccordionCard from './AccordionCard.vue'

const props = defineProps({ summary: Object, history: Array })

const uptimeText = computed(() => {
  const u = props.summary?.uptime_percent !== undefined ? props.summary.uptime_percent : props.summary?.uptime_24h
  if (u === undefined || u === null || isNaN(u) || u < 0) return 'No uptime data'
  if (u === 0) return '0% uptime'
  return `${u}%`
})

const peakDl = computed(() => {
  if (props.summary?.download_peak > 0) return props.summary.download_peak + ' Mbps'
  const vals = props.history?.map(h => h.download).filter(v => v > 0) ?? []
  return vals.length ? Math.max(...vals) + ' Mbps' : '—'
})
const minLat = computed(() => (props.summary?.latency_min > 0 ? props.summary.latency_min : props.history?.length ? Math.min(...props.history.map(h => h.latency).filter(l => l > 0)) : 0) || '—')
const maxLat = computed(() => props.summary?.latency_max || '—')
</script>
