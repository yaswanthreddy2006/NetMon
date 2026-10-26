<template>
  <div class="card p-4 sm:p-5 flex flex-col gap-4">
    <!-- Header -->
    <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-2 pb-2 border-b border-[var(--border)]">
      <div class="flex items-center gap-2">
        <div class="w-8 h-8 rounded-lg bg-cyan-500/10 border border-cyan-500/20 flex items-center justify-center text-cyan-400">
          <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <circle cx="12" cy="12" r="10"/>
            <line x1="2" y1="12" x2="22" y2="12"/>
            <path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z"/>
          </svg>
        </div>
        <div>
          <h2 class="text-xs sm:text-sm font-black tracking-wide uppercase text-[var(--fg)] flex items-center gap-2">
            Network Architecture &amp; Physical Infrastructure
          </h2>
          <p class="text-[11px] text-[var(--muted)]">
            End-to-end topological path, gateway routing, and physical link specifications
          </p>
        </div>
      </div>

      <div class="flex items-center gap-2">
        <span class="text-[11px] font-mono px-2.5 py-0.5 rounded bg-[var(--bg-secondary)] border border-[var(--border)] text-[var(--muted)]">
          {{ targets.length }} Active Probes
        </span>
      </div>
    </div>

    <!-- Topology Visualization Flow -->
    <NetworkVisualization
      :targets="targets"
      :summary="summary"
      :live-traffic="liveTraffic"
      :hotspot="hotspot"
      :is-online="isOnline"
      :latest-measurement="latestMeasurement"
      :connection-info="connectionInfo"
      :local-ip="localIp"
      :gateway-ip="gatewayIp"
      :network-id="networkId"
    />

    <!-- Infrastructure Telemetry Grid (Tables & Specs) -->
    <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-3.5 pt-2 border-t border-[var(--border)]">
      
      <!-- TARGET PROBES TABLE -->
      <AccordionCard section="connectivity_net" padding="p-3.5">
        <template #title>
          <span class="flex items-center gap-2 text-xs font-bold uppercase tracking-wider text-[var(--fg)]">
            <svg xmlns="http://www.w3.org/2000/svg" width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" class="text-cyan-400"><circle cx="12" cy="12" r="10"/><circle cx="12" cy="6"/><circle cx="12" cy="2"/></svg>
            Target Probes Telemetry
          </span>
        </template>
        <ConnectivityTable :targets="targets" :now="now" />
      </AccordionCard>

      <!-- DNS RESOLVERS TABLE -->
      <AccordionCard section="dns_net" padding="p-3.5">
        <template #title>
          <span class="flex items-center gap-2 text-xs font-bold uppercase tracking-wider text-[var(--fg)]">
            <svg xmlns="http://www.w3.org/2000/svg" width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" class="text-purple-400"><circle cx="12" cy="12" r="10"/><line x1="2" y1="12" x2="22" y2="12"/><path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z"/></svg>
            DNS Resolvers
          </span>
        </template>
        <DnsTable :checks="dns" :now="now" />
      </AccordionCard>

      <!-- PHYSICAL CONNECTION / WI-FI / ETHERNET SPECS -->
      <ConnectionCard
        :history="history"
        :now="now"
        :connection-info="connectionInfo"
        :network-id="networkId"
        :local-ip="localIp"
      />

    </div>
  </div>
</template>

<script setup>
import NetworkVisualization from './NetworkVisualization.vue'
import AccordionCard        from './AccordionCard.vue'
import ConnectivityTable    from './ConnectivityTable.vue'
import DnsTable             from './DnsTable.vue'
import ConnectionCard       from './ConnectionCard.vue'

defineProps({
  targets:           { type: Array, default: () => [] },
  summary:           { type: Object, default: () => null },
  liveTraffic:       { type: Object, default: () => null },
  hotspot:           { type: Object, default: () => null },
  latestMeasurement: { type: Object, default: () => null },
  dns:               { type: Array, default: () => [] },
  history:           { type: Array, default: () => [] },
  isOnline:          { type: Boolean, default: true },
  now:               { type: Number, default: () => Date.now() },
  connectionInfo:    { type: Object, default: () => null },
  localIp:           { type: String, default: '' },
  gatewayIp:         { type: String, default: '' },
  networkId:         { type: String, default: '' },
})
</script>
