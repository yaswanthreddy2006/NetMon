<template>
  <table class="target-table">
    <thead v-if="targets?.length">
      <tr><th>Host / Method</th><th>Latency (min/avg/max)</th><th>Loss (lost/sent)</th></tr>
    </thead>
    <tbody>
      <tr v-if="!targets?.length">
        <td colspan="3" style="border-bottom:none">
          <div class="card-empty">
            <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><circle cx="12" cy="12" r="6"/><circle cx="12" cy="12" r="2"/></svg>
            <span>No targets yet</span>
          </div>
        </td>
      </tr>
      <tr v-else v-for="t in targets" :key="t.host">
        <td>
          <div class="host-cell flex items-center gap-1.5">
            <span class="inline-dot" :class="t.status === 'up' ? 'up' : t.status === 'degraded' ? 'degraded' : 'down'" />
            <span class="font-medium">{{ t.host === 'gateway' ? 'Gateway' : t.host }}</span>
            <span v-if="t.host === 'gateway'" class="text-[9px] uppercase tracking-wider px-1.5 py-0.5 rounded bg-blue-500/10 text-blue-500 font-bold ml-0.5">GW</span>
            <span
              v-if="t.method"
              class="text-[9px] uppercase tracking-wider px-1 py-0.2 rounded font-mono font-medium"
              :class="t.method === 'icmp' ? 'bg-muted/10 text-muted' : 'bg-amber-500/10 text-amber-500 border border-amber-500/20'"
              :title="t.method === 'icmp' ? 'Native ICMP Echo' : 'TCP Connect Fallback'"
            >
              {{ t.method === 'icmp' ? 'ICMP' : 'TCP' }}
            </span>
          </div>
          <div class="ip-cell flex items-center gap-1.5 text-[10px] text-muted">
            <span class="font-mono">{{ t.ip || 'Resolving…' }}</span>
            <span v-if="t.jitter > 0" class="text-[9px]">· Jitter: {{ t.jitter }}ms</span>
            <span v-if="t.last_checked" class="text-[9px]">· {{ formatAgo(t.last_checked) }}</span>
          </div>
        </td>
        <td>
          <div class="font-semibold text-xs text-[var(--fg)]">
            {{ t.latency > 0 ? t.latency + ' ms' : (t.status === 'down' ? 'Down' : '—') }}
          </div>
          <div v-if="t.min_latency > 0 || t.max_latency > 0" class="text-[9px] text-muted font-mono">
            {{ t.min_latency }} / {{ t.avg_latency || t.latency }} / {{ t.max_latency }} ms
          </div>
        </td>
        <td>
          <div class="font-semibold text-xs" :class="t.loss > 0 ? 'text-amber-500' : 'text-emerald-500'">
            {{ t.loss }}%
          </div>
          <div v-if="t.probes_sent > 0" class="text-[9px] text-muted font-mono">
            {{ t.probes_lost }}/{{ t.probes_sent }} lost
          </div>
        </td>
      </tr>
    </tbody>
  </table>
</template>

<script setup>
import { formatTimeAgo } from '../utils/time.js'

const props = defineProps({
  targets: Array,
  now: { type: Number, default: () => Date.now() },
})

function formatAgo(ts) {
  return formatTimeAgo(ts, props.now)
}
</script>
