<template>
  <table class="target-table">
    <thead v-if="checks?.length">
      <tr><th>Host / Resolver</th><th>Status / Time</th></tr>
    </thead>
    <tbody>
      <tr v-if="!checks?.length">
        <td colspan="2" style="border-bottom:none">
          <div class="card-empty">
            <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><line x1="2" y1="12" x2="22" y2="12"/><path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1 4-10z"/></svg>
            <span>No lookups yet</span>
          </div>
        </td>
      </tr>
      <tr v-else v-for="d in checks" :key="d.host">
        <td>
          <div class="flex items-center gap-1.5">
            <span class="inline-dot" :class="isSuccess(d) ? 'up' : 'down'" />
            <span class="font-semibold text-xs text-[var(--fg)]">{{ d.host }}</span>
          </div>
          <div class="text-[10px] text-muted flex items-center gap-1.5 pt-0.5">
            <span class="font-mono">{{ d.server_ip || d.resolver || 'system' }}</span>
            <span v-if="d.last_checked" class="text-[9px]">· {{ formatAgo(d.last_checked) }}</span>
          </div>
        </td>
        <td class="text-right">
          <div class="flex items-center justify-end gap-1.5">
            <span
              v-if="d.status && d.status !== 'SUCCESS'"
              class="text-[9px] uppercase px-1 py-0.2 rounded font-bold"
              :class="d.status === 'NXDOMAIN' ? 'bg-amber-500/10 text-amber-500' : 'bg-red-500/10 text-red-500'"
              :title="d.error || d.rcode_name"
            >
              {{ d.rcode_name || d.status }}
            </span>
            <span class="font-semibold text-xs text-[var(--fg)]">
              {{ d.time_ms > 0 ? d.time_ms + ' ms' : (d.status === 'TIMEOUT' ? 'Timeout' : 'Failed') }}
            </span>
          </div>
          <div v-if="d.error && d.status !== 'SUCCESS'" class="text-[9px] text-red-400 truncate max-w-[120px] ml-auto" :title="d.error">
            {{ d.error }}
          </div>
        </td>
      </tr>
    </tbody>
  </table>
</template>

<script setup>
import { formatTimeAgo } from '../utils/time.js'

const props = defineProps({
  checks: Array,
  now: { type: Number, default: () => Date.now() },
})

function isSuccess(d) {
  return !d.status || d.status === 'SUCCESS' || (d.time_ms > 0 && d.status !== 'FAILED' && d.status !== 'TIMEOUT')
}

function formatAgo(ts) {
  return formatTimeAgo(ts, props.now)
}
</script>
