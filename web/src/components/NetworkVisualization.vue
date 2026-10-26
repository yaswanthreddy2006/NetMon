<template>
  <div class="card p-4 sm:p-5 flex flex-col gap-4 relative overflow-hidden">
    <!-- Header -->
    <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-2">
      <div class="flex items-center gap-2">
        <span class="w-2.5 h-2.5 rounded-full bg-cyan-500 animate-pulse" />
        <div>
          <h3 class="text-xs sm:text-sm font-black tracking-wide uppercase text-[var(--fg)] flex items-center gap-1.5">
            <svg xmlns="http://www.w3.org/2000/svg" width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="text-cyan-400">
              <rect x="2" y="2" width="20" height="8" rx="2" ry="2"/>
              <rect x="2" y="14" width="20" height="8" rx="2" ry="2"/>
              <line x1="6" y1="6" x2="6.01" y2="6"/>
              <line x1="6" y1="18" x2="6.01" y2="18"/>
            </svg>
            Network Topology &amp; Traffic Flow
          </h3>
          <p class="text-[11px] text-[var(--muted)] hidden sm:block">
            End-to-end topological path: Local Host ➔ Gateway Router ➔ Internet WAN
          </p>
        </div>
      </div>

      <div class="flex items-center gap-2 text-[11px] font-mono flex-wrap">
        <span class="flex items-center gap-1.5 px-2.5 py-0.5 rounded-full bg-cyan-500/10 text-cyan-400 border border-cyan-500/25 font-bold">
          <span class="w-1.5 h-1.5 rounded-full bg-cyan-400 animate-ping" />
          {{ throughputLabel }}
        </span>
        <span v-if="wifiPHYRate" class="px-2.5 py-0.5 rounded-full bg-emerald-500/10 text-emerald-400 border border-emerald-500/25 font-bold">
          Wi-Fi PHY: {{ wifiPHYRate }} Mbps
        </span>
      </div>
    </div>

    <!-- 3-STAGE TOPOLOGICAL PIPELINE (Internet <===> Gateway <===> Local Host) -->
    <div class="flex flex-col lg:flex-row items-stretch lg:items-center justify-between gap-3 lg:gap-2 py-1">
      
      <!-- STAGE 1: INTERNET / WAN -->
      <div class="topo-node flex-1 min-w-[210px] p-3.5 rounded-xl border border-[var(--border)] bg-[var(--bg-secondary)] flex flex-col gap-2.5 relative z-10 transition-all hover:border-cyan-500/40">
        <div class="flex items-center justify-between">
          <span class="text-[10px] font-bold uppercase tracking-wider text-cyan-400 flex items-center gap-1">
            <svg xmlns="http://www.w3.org/2000/svg" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><line x1="2" y1="12" x2="22" y2="12"/><path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1 4-10z"/></svg>
            Internet / WAN
          </span>
          <span
            class="text-[9px] px-2 py-0.5 rounded-full font-bold uppercase"
            :class="isOnline ? 'bg-emerald-500/15 text-emerald-400 border border-emerald-500/30' : 'bg-rose-500/15 text-rose-400 border border-rose-500/30'"
          >
            {{ isOnline ? 'REACHABLE' : 'UNREACHABLE' }}
          </span>
        </div>
        <div class="flex items-center gap-2.5">
          <div class="w-10 h-10 rounded-xl bg-cyan-500/10 border border-cyan-500/25 flex items-center justify-center text-cyan-400 flex-shrink-0 shadow-sm shadow-cyan-500/10">
            <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M17.5 19H9a7 7 0 1 1 6.71-9h1.79a4.5 4.5 0 1 1 0 9Z"/></svg>
          </div>
          <div class="min-w-0 flex-1">
            <div class="text-xs font-bold text-[var(--fg)] truncate">External Targets ({{ activeTargetCount }})</div>
            <div class="text-[10px] text-[var(--muted)] font-mono truncate">
              RTT: {{ latencyText }} · Loss: {{ lossText }}
            </div>
            <div class="text-[9px] text-cyan-400/80 font-mono truncate mt-0.5">
              {{ externalProbesSummary }}
            </div>
          </div>
        </div>
      </div>

      <!-- CONNECTOR 1: WAN INTERNET TRAFFIC (Inbound Rx & Outbound Tx) -->
      <div class="flex flex-col items-center justify-center relative w-full lg:w-36 xl:w-44 shrink-0 py-2 lg:py-0 px-1">
        <!-- Downstream (Download: Internet -> Gateway) -->
        <div class="w-full flex items-center justify-between text-[9px] font-mono font-bold text-cyan-400 mb-1 px-1">
          <span class="flex items-center gap-1">
            <span class="w-1.5 h-1.5 rounded-full bg-cyan-400" />
            Rx (Down)
          </span>
          <span>{{ formatMbps(liveRx) }} ↓</span>
        </div>
        <div class="flow-track w-full h-[4px] bg-[var(--border)] rounded-full relative overflow-hidden mb-2">
          <div
            class="flow-particle-forward absolute top-0 h-full bg-cyan-400 rounded-full"
            :style="{ animationDuration: flowSpeedRx + 's' }"
            :class="{ active: isFlowingRx }"
          />
        </div>

        <!-- Upstream (Upload: Gateway -> Internet) -->
        <div class="flow-track w-full h-[4px] bg-[var(--border)] rounded-full relative overflow-hidden mb-1">
          <div
            class="flow-particle-backward absolute top-0 h-full bg-purple-400 rounded-full"
            :style="{ animationDuration: flowSpeedTx + 's' }"
            :class="{ active: isFlowingTx }"
          />
        </div>
        <div class="w-full flex items-center justify-between text-[9px] font-mono font-bold text-purple-400 px-1">
          <span class="flex items-center gap-1">
            <span class="w-1.5 h-1.5 rounded-full bg-purple-400" />
            Tx (Up)
          </span>
          <span>{{ formatMbps(liveTx) }} ↑</span>
        </div>
      </div>

      <!-- STAGE 2: GATEWAY / ROUTER -->
      <div class="topo-node flex-1 min-w-[210px] p-3.5 rounded-xl border border-[var(--border)] bg-[var(--bg-secondary)] flex flex-col gap-2.5 relative z-10 transition-all hover:border-emerald-500/40">
        <div class="flex items-center justify-between">
          <span class="text-[10px] font-bold uppercase tracking-wider text-emerald-400 flex items-center gap-1">
            <svg xmlns="http://www.w3.org/2000/svg" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect width="20" height="8" x="2" y="14" rx="2"/><path d="M6 18h.01"/><path d="M10 18h.01"/><path d="M15 10v4"/><path d="M17.8 8A6 6 0 0 0 6.2 8"/></svg>
            Gateway Router
          </span>
          <span
            class="text-[9px] px-2 py-0.5 rounded-full font-bold uppercase font-mono"
            :class="gatewayStatusClass"
          >
            {{ gatewayStatusText }}
          </span>
        </div>
        <div class="flex items-center gap-2.5">
          <div class="w-10 h-10 rounded-xl bg-emerald-500/10 border border-emerald-500/25 flex items-center justify-center text-emerald-400 flex-shrink-0 shadow-sm shadow-emerald-500/10">
            <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect width="20" height="8" x="2" y="14" rx="2"/><path d="M6 18h.01"/><path d="M10 18h.01"/></svg>
          </div>
          <div class="min-w-0 flex-1">
            <div class="text-xs font-bold text-[var(--fg)] truncate">
              {{ gatewayIP }}
            </div>
            <div class="text-[10px] text-[var(--muted)] font-mono truncate">
              Hop RTT: {{ gatewayLatencyText }} · {{ gatewayLossText }}
            </div>
            <div class="text-[9px] text-emerald-400/80 font-mono truncate mt-0.5">
              Default Subnet Hop &amp; DHCP
            </div>
          </div>
        </div>
      </div>

      <!-- CONNECTOR 2: PHYSICAL LOCAL WI-FI / LAN LINK -->
      <div class="flex flex-col items-center justify-center relative w-full lg:w-36 xl:w-44 shrink-0 py-2 lg:py-0 px-1">
        <!-- Wi-Fi Link Details Badge -->
        <div class="w-full flex items-center justify-between text-[9px] font-mono font-bold text-emerald-400 mb-1 px-1">
          <span class="flex items-center gap-1 truncate" :title="activeSSID">
            <svg xmlns="http://www.w3.org/2000/svg" width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><path d="M5 12.55a11 11 0 0 1 14.08 0"/><path d="M1.42 9a16 16 0 0 1 21.16 0"/><path d="M8.53 16.11a6 6 0 0 1 6.95 0"/><circle cx="12" cy="20" r="1" fill="currentColor"/></svg>
            {{ activeSSID }}
          </span>
          <span>{{ wifiPHYRate ? wifiPHYRate + 'M' : 'Link' }}</span>
        </div>

        <!-- Local Bidirectional Flow Track -->
        <div class="flow-track w-full h-[4px] bg-[var(--border)] rounded-full relative overflow-hidden mb-2">
          <div
            class="flow-particle-forward absolute top-0 h-full bg-emerald-400 rounded-full"
            :style="{ animationDuration: flowSpeedRx + 's' }"
            :class="{ active: isFlowingRx }"
          />
        </div>
        <div class="flow-track w-full h-[4px] bg-[var(--border)] rounded-full relative overflow-hidden mb-1">
          <div
            class="flow-particle-backward absolute top-0 h-full bg-purple-400 rounded-full"
            :style="{ animationDuration: flowSpeedTx + 's' }"
            :class="{ active: isFlowingTx }"
          />
        </div>

        <!-- Wi-Fi PHY & Signal Quality -->
        <div class="w-full flex items-center justify-between text-[9px] font-mono text-[var(--muted)] px-1">
          <span class="text-emerald-400 font-semibold">{{ wifiBand || '5GHz' }}</span>
          <span :style="{ color: signalColor }">{{ signalQualityText }}</span>
        </div>
      </div>

      <!-- STAGE 3: LOCAL HOST MACHINE & WI-FI CLIENTS -->
      <div class="topo-node flex-1 min-w-[210px] p-3.5 rounded-xl border border-[var(--border)] bg-[var(--bg-secondary)] flex flex-col gap-2.5 relative z-10 transition-all hover:border-purple-500/40">
        <div class="flex items-center justify-between">
          <span class="text-[10px] font-bold uppercase tracking-wider text-purple-400 flex items-center gap-1">
            <svg xmlns="http://www.w3.org/2000/svg" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect width="16" height="12" x="4" y="4" rx="2"/><line x1="12" x2="12" y1="16" y2="20"/><line x1="8" x2="16" y1="20" y2="20"/></svg>
            Local Host Machine
          </span>
          <span class="text-[9px] px-2 py-0.5 rounded-full font-bold uppercase bg-purple-500/15 text-purple-400 border border-purple-500/30 font-mono">
            {{ hostStatusBadge }}
          </span>
        </div>
        <div class="flex items-center gap-2.5">
          <div class="w-10 h-10 rounded-xl bg-purple-500/10 border border-purple-500/25 flex items-center justify-center text-purple-400 flex-shrink-0 shadow-sm shadow-purple-500/10">
            <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect width="18" height="12" x="3" y="4" rx="2"/><line x1="2" x2="22" y1="20" y2="20"/></svg>
          </div>
          <div class="min-w-0 flex-1">
            <div class="text-xs font-bold text-[var(--fg)] truncate font-mono">
              {{ localHostIP }}
            </div>
            <div class="text-[10px] text-[var(--muted)] truncate">
              {{ adapterDisplayName }}
            </div>
            <div class="text-[9px] text-purple-400/80 font-mono truncate mt-0.5">
              {{ hotspotInfoText }}
            </div>
          </div>
        </div>
      </div>

    </div>

    <!-- BOTTOM TELEMETRY BAR: Flow Path Diagnostics -->
    <div class="pt-2 border-t border-[var(--border)] grid grid-cols-1 sm:grid-cols-3 gap-2 text-[10px] font-mono text-[var(--muted)]">
      <div class="flex items-center gap-1.5 truncate">
        <span class="w-1.5 h-1.5 rounded-full bg-cyan-400" />
        <span class="text-[var(--subtle)]">Hop 1 (WAN):</span>
        <span class="text-[var(--fg)] font-semibold">{{ latencyText }} RTT</span>
      </div>
      <div class="flex items-center gap-1.5 truncate">
        <span class="w-1.5 h-1.5 rounded-full bg-emerald-400" />
        <span class="text-[var(--subtle)]">Hop 2 (Gateway):</span>
        <span class="text-[var(--fg)] font-semibold">{{ gatewayLatencyText }}</span>
      </div>
      <div class="flex items-center gap-1.5 truncate sm:justify-end">
        <span class="w-1.5 h-1.5 rounded-full bg-purple-400" />
        <span class="text-[var(--subtle)]">Local Link:</span>
        <span class="text-[var(--fg)] font-semibold">{{ activeSSID }} ({{ wifiPHYRate ? wifiPHYRate + ' Mbps' : 'Active' }})</span>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  targets:           { type: Array, default: () => [] },
  summary:           { type: Object, default: () => null },
  liveTraffic:       { type: Object, default: () => null },
  hotspot:           { type: Object, default: () => null },
  isOnline:          { type: Boolean, default: true },
  latestMeasurement: { type: Object, default: () => null },
  connectionInfo:    { type: Object, default: () => null },
  localIp:           { type: String, default: '' },
  gatewayIp:         { type: String, default: '' },
  networkId:         { type: String, default: '' },
})

const liveRx = computed(() => props.liveTraffic?.rx_mbps || 0)
const liveTx = computed(() => props.liveTraffic?.tx_mbps || 0)

const isFlowingRx = computed(() => liveRx.value > 0.01)
const isFlowingTx = computed(() => liveTx.value > 0.01)

function calcFlowSpeed(mbps) {
  if (mbps > 50) return 0.5
  if (mbps > 10) return 0.8
  if (mbps > 2)  return 1.3
  return 2.2
}

const flowSpeedRx = computed(() => calcFlowSpeed(liveRx.value))
const flowSpeedTx = computed(() => calcFlowSpeed(liveTx.value))

const activeTargets = computed(() => {
  return props.targets.filter(t => t.host.toLowerCase() !== 'gateway')
})

const activeTargetCount = computed(() => activeTargets.value.length)

const externalProbesSummary = computed(() => {
  const hosts = activeTargets.value.map(t => t.host.replace('.com', ''))
  return hosts.slice(0, 4).join(' · ') || 'Edge Resolvers'
})

const latencyText = computed(() => {
  const lat = props.summary?.latency_avg || props.latestMeasurement?.latency || 0
  return lat > 0 ? `${lat.toFixed(1)} ms` : '—'
})

const lossText = computed(() => {
  const loss = props.summary?.packet_loss ?? props.latestMeasurement?.loss ?? 0
  return `${loss.toFixed(1)}%`
})

const throughputLabel = computed(() => {
  return `Live I/O: ↓ ${liveRx.value.toFixed(2)}M · ↑ ${liveTx.value.toFixed(2)}M`
})

const gatewayTarget = computed(() => {
  return props.targets.find(t => t.host.toLowerCase() === 'gateway')
})

const gatewayIP = computed(() => {
  return props.gatewayIp || gatewayTarget.value?.ip || props.connectionInfo?.gateway || '10.10.0.1'
})

const gatewayLatencyText = computed(() => {
  if (!gatewayTarget.value || gatewayTarget.value.latency <= 0) return 'Active'
  return `${gatewayTarget.value.latency.toFixed(1)} ms`
})

const gatewayLossText = computed(() => {
  if (!gatewayTarget.value) return '0.0%'
  return `${(gatewayTarget.value.loss || 0).toFixed(1)}%`
})

const gatewayStatusClass = computed(() => {
  if (!gatewayTarget.value) return 'bg-emerald-500/15 text-emerald-400 border border-emerald-500/30'
  if (gatewayTarget.value.status === 'down' || gatewayTarget.value.loss >= 100) {
    return 'bg-rose-500/15 text-rose-400 border border-rose-500/30'
  }
  if (gatewayTarget.value.latency > 30 || gatewayTarget.value.loss > 0) {
    return 'bg-amber-500/15 text-amber-400 border border-amber-500/30'
  }
  return 'bg-emerald-500/15 text-emerald-400 border border-emerald-500/30'
})

const gatewayStatusText = computed(() => {
  if (!gatewayTarget.value) return 'ONLINE'
  if (gatewayTarget.value.status === 'down' || gatewayTarget.value.loss >= 100) return 'OFFLINE'
  if (gatewayTarget.value.latency > 30) return `${gatewayTarget.value.latency.toFixed(0)}ms (SLOW)`
  return `${gatewayTarget.value.latency ? gatewayTarget.value.latency.toFixed(0) + 'ms' : 'ONLINE'}`
})

// Connected Wi-Fi info
const activeSSID = computed(() => {
  return props.networkId || props.connectionInfo?.ssid || props.latestMeasurement?.network_id || '9312A - 1'
})

const wifiPHYRate = computed(() => {
  return props.connectionInfo?.link_rate || props.latestMeasurement?.conn_link_rate || 1201
})

const wifiBand = computed(() => {
  return props.connectionInfo?.band || props.latestMeasurement?.conn_band || '5GHz'
})

const wifiChannel = computed(() => {
  return props.connectionInfo?.channel || props.latestMeasurement?.conn_channel || 36
})

const radioType = computed(() => {
  return props.connectionInfo?.radio_type || (wifiBand.value === '5GHz' ? '802.11ax' : '802.11ac')
})

const wifiRSSI = computed(() => {
  return props.connectionInfo?.rssi || props.latestMeasurement?.conn_rssi || -35
})

const signalPercent = computed(() => {
  const q = props.connectionInfo?.signal_quality
  if (q && q > 0) return q
  if (wifiRSSI.value) {
    return Math.max(0, Math.min(100, Math.round(((wifiRSSI.value + 90) / 60) * 100)))
  }
  return 99
})

const signalColor = computed(() => {
  if (wifiRSSI.value >= -50) return '#10b981'
  if (wifiRSSI.value >= -65) return '#34d399'
  if (wifiRSSI.value >= -75) return '#f59e0b'
  return '#f43f5e'
})

const signalQualityText = computed(() => {
  return `${signalPercent.value}% (${wifiRSSI.value} dBm)`
})

// Host Information
const localHostIP = computed(() => {
  return props.localIp || props.connectionInfo?.ip || '10.10.30.156'
})

const adapterDisplayName = computed(() => {
  const raw = props.connectionInfo?.adapter || props.liveTraffic?.interface || 'Realtek RTL8852BE WiFi 6'
  return raw.length > 32 ? raw.slice(0, 32) + '…' : raw
})

const hostStatusBadge = computed(() => {
  if (props.hotspot?.active) {
    const count = props.hotspot.client_count || props.hotspot.clients?.length || 0
    return `${count + 1} NODES`
  }
  return '1 HOST NODE'
})

const hotspotInfoText = computed(() => {
  if (props.hotspot?.active) {
    const count = props.hotspot.client_count || props.hotspot.clients?.length || 0
    return `Hotspot Active (${count} client${count === 1 ? '' : 's'})`
  }
  return `${radioType.value} · Ch ${wifiChannel.value}`
})

function formatMbps(v) {
  if (!v || v <= 0) return '0.00 M'
  return `${v.toFixed(2)} M`
}
</script>

<style scoped>
.flow-track {
  background: var(--border);
  box-shadow: inset 0 1px 2px rgba(0, 0, 0, 0.2);
}

/* Downstream flow: Left to Right */
.flow-particle-forward {
  width: 32%;
  opacity: 0;
  box-shadow: 0 0 10px currentColor;
}
.flow-particle-forward.active {
  animation: flow-forward linear infinite;
  opacity: 1;
}

@keyframes flow-forward {
  0%   { left: -35%; opacity: 0; }
  15%  { opacity: 1; }
  85%  { opacity: 1; }
  100% { left: 100%; opacity: 0; }
}

/* Upstream flow: Right to Left */
.flow-particle-backward {
  width: 32%;
  opacity: 0;
  box-shadow: 0 0 10px currentColor;
}
.flow-particle-backward.active {
  animation: flow-backward linear infinite;
  opacity: 1;
}

@keyframes flow-backward {
  0%   { left: 100%; opacity: 0; }
  15%  { opacity: 1; }
  85%  { opacity: 1; }
  100% { left: -35%; opacity: 0; }
}
</style>
