<template>
  <div class="card p-4 sm:p-5 flex flex-col gap-4">
    <!-- Header -->
    <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-2 pb-2 border-b border-[var(--border)]">
      <div class="flex items-center gap-2">
        <div class="w-8 h-8 rounded-lg bg-blue-500/10 border border-blue-500/20 flex items-center justify-center text-blue-400">
          <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <rect width="16" height="12" x="4" y="4" rx="2"/>
            <line x1="12" x2="12" y1="16" y2="20"/>
            <line x1="8" x2="16" y1="20" y2="20"/>
          </svg>
        </div>
        <div>
          <h2 class="text-xs sm:text-sm font-black tracking-wide uppercase text-[var(--fg)] flex items-center gap-2">
            Connected Devices &amp; Topology Nodes
            <span class="text-[10px] font-mono px-2 py-0.5 rounded-full bg-blue-500/10 text-blue-400 border border-blue-500/20 font-bold">
              {{ totalNodesCount }} Active Nodes
            </span>
          </h2>
          <p class="text-[11px] text-[var(--muted)]">
            Verified network devices, Wi-Fi hotspot clients, and gateway endpoints
          </p>
        </div>
      </div>

      <!-- Hotspot Switch/Status Tag -->
      <div class="flex items-center gap-2">
        <span
          class="inline-flex items-center gap-1.5 px-3 py-1 rounded-full text-[11px] font-bold tracking-wide"
          :class="hotspot?.active ? 'bg-emerald-500/15 text-emerald-400 border border-emerald-500/30' : 'bg-[var(--bg-secondary)] text-[var(--muted)] border border-[var(--border)]'"
        >
          <span
            class="w-2 h-2 rounded-full"
            :class="hotspot?.active ? 'bg-emerald-400 animate-pulse' : 'bg-slate-500'"
          />
          {{ hotspot?.active ? 'Hotspot Active' : 'Hotspot Inactive' }}
        </span>
      </div>
    </div>

    <!-- Hotspot Client Devices (If active) -->
    <div v-if="hotspot?.active && hotspotClients.length > 0" class="flex flex-col gap-2.5">
      <div class="text-[10px] font-bold uppercase tracking-wider text-[var(--muted)] flex items-center gap-1.5">
        <span>Connected Wireless Clients ({{ hotspotClients.length }})</span>
      </div>
      
      <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-3">
        <div
          v-for="client in hotspotClients"
          :key="client.ip || client.mac"
          class="p-3.5 rounded-xl border border-[var(--border)] bg-[var(--bg-secondary)] flex flex-col gap-2.5 transition-all hover:border-blue-500/40"
        >
          <div class="flex items-center justify-between">
            <div class="flex items-center gap-2 min-w-0">
              <span class="text-xl flex-shrink-0">{{ client.device_icon || '📱' }}</span>
              <div class="min-w-0">
                <div class="text-xs font-bold text-[var(--fg)] truncate">
                  {{ client.hostname || 'Client Device' }}
                </div>
                <div class="text-[10px] font-mono text-[var(--muted)] truncate">
                  {{ client.ip }}
                </div>
              </div>
            </div>
            <!-- Status indicator: ONLINE, OFFLINE, ACTIVE, IDLE, SUSPICIOUS -->
            <span
              class="text-[9px] font-mono px-2 py-0.5 rounded-full font-bold uppercase tracking-wider"
              :class="getClientStatusClass(client)"
            >
              {{ getClientStatus(client) }}
            </span>
          </div>

          <!-- Usage Metrics -->
          <div class="grid grid-cols-3 gap-2 pt-1 border-t border-[var(--border)] text-[10px]">
            <div>
              <span class="text-[var(--muted)] block text-[9px]">↓ Download</span>
              <span class="font-mono font-bold text-emerald-400">{{ formatMbps(client.rx_mbps) }}</span>
            </div>
            <div>
              <span class="text-[var(--muted)] block text-[9px]">↑ Upload</span>
              <span class="font-mono font-bold text-purple-400">{{ formatMbps(client.tx_mbps) }}</span>
            </div>
            <div>
              <span class="text-[var(--muted)] block text-[9px]">Total Usage</span>
              <span class="font-mono font-bold text-[var(--fg)]">{{ formatBytes(client.total_rx_bytes + client.total_tx_bytes) }}</span>
            </div>
          </div>

          <!-- MAC & Activity Details -->
          <div class="flex items-center justify-between text-[9px] text-[var(--muted)] pt-1 border-t border-[var(--border)]">
            <span class="font-mono truncate" :title="client.mac">{{ client.mac || 'MAC Protected' }}</span>
            <span class="font-semibold" :class="client.is_active ? 'text-emerald-400' : 'text-slate-400'">
              {{ client.is_active ? '● ACTIVE STREAM' : '○ IDLE' }}
            </span>
          </div>
        </div>
      </div>
    </div>

    <!-- Hotspot Inactive / Platform Status Notice -->
    <div v-else-if="!hotspot?.active" class="p-4 rounded-xl border border-dashed border-[var(--border)] bg-[var(--bg-secondary)] flex flex-col sm:flex-row items-start sm:items-center justify-between gap-3">
      <div class="flex items-center gap-3">
        <div class="w-9 h-9 rounded-full bg-[var(--card)] border border-[var(--border)] flex items-center justify-center text-[var(--muted)] flex-shrink-0">
          <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <line x1="1" y1="1" x2="23" y2="23"/>
            <path d="M16.72 11.06A10.94 10.94 0 0 1 19 12.55"/>
            <path d="M5 12.55a10.94 10.94 0 0 1 5.17-2.39"/>
            <circle cx="12" cy="20" r="1"/>
          </svg>
        </div>
        <div>
          <div class="text-xs font-bold text-[var(--fg)]">
            Hotspot Inactive · 0 Connected Devices
          </div>
          <div class="text-[11px] text-[var(--muted)] mt-0.5">
            Turn on Windows Mobile Hotspot in Windows Network Settings to monitor connected client phones, laptops, and smart TVs in real time.
          </div>
        </div>
      </div>
      <span class="text-[10px] font-mono px-3 py-1 rounded-lg bg-[var(--card)] text-[var(--muted)] border border-[var(--border)] whitespace-nowrap font-medium">
        Adapter Ready
      </span>
    </div>

    <!-- Verified Infrastructure Network Devices Table -->
    <div class="flex flex-col gap-2 pt-1">
      <div class="text-[10px] font-bold uppercase tracking-wider text-[var(--muted)] flex items-center justify-between">
        <span>Verified Network Infrastructure Nodes ({{ localNodes.length }})</span>
        <span class="text-[9px] font-mono text-[var(--muted)]">Hardware / Socket Layer</span>
      </div>

      <div class="overflow-x-auto">
        <table class="w-full text-left text-xs border-collapse">
          <thead>
            <tr class="text-[10px] uppercase font-bold text-[var(--muted)] border-b border-[var(--border)]">
              <th class="py-2.5 px-3">Device / Endpoint</th>
              <th class="py-2.5 px-3">IP Address</th>
              <th class="py-2.5 px-3">Role / Interface</th>
              <th class="py-2.5 px-3 text-right">Throughput / Latency</th>
              <th class="py-2.5 px-3 text-right">Status</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-[var(--border)]">
            <tr v-for="node in localNodes" :key="node.ip + node.name" class="hover:bg-[var(--bg-secondary)] transition-colors">
              <td class="py-2.5 px-3 font-medium text-[var(--fg)] flex items-center gap-2.5">
                <span class="text-lg">{{ node.icon }}</span>
                <div>
                  <div class="font-bold text-xs">{{ node.name }}</div>
                  <div class="text-[10px] text-[var(--muted)]">{{ node.type }}</div>
                </div>
              </td>
              <td class="py-2.5 px-3 font-mono text-xs text-[var(--subtle)]">
                {{ node.ip }}
              </td>
              <td class="py-2.5 px-3 text-[11px] text-[var(--muted)]">
                {{ node.role }}
              </td>
              <td class="py-2.5 px-3 font-mono text-xs text-right font-semibold" :style="{ color: node.speedColor || 'inherit' }">
                {{ node.metric }}
              </td>
              <td class="py-2.5 px-3 text-right">
                <span
                  class="text-[9px] font-mono px-2 py-0.5 rounded-full font-bold uppercase tracking-wider"
                  :class="node.statusClass"
                >
                  {{ node.status }}
                </span>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  hotspot: { type: Object, default: () => null },
  liveTraffic: { type: Object, default: () => null },
  targets: { type: Array, default: () => [] },
  latestMeasurement: { type: Object, default: () => null },
  connectionInfo: { type: Object, default: () => null },
  localIp: { type: String, default: '' },
  networkId: { type: String, default: '' },
})

const hotspotClients = computed(() => {
  return props.hotspot?.clients || []
})

const totalNodesCount = computed(() => {
  return hotspotClients.value.length + localNodes.value.length
})

const localNodes = computed(() => {
  const nodes = []

  // 1. Host Machine
  const iface = props.connectionInfo?.adapter || props.liveTraffic?.interface || 'Wi-Fi Adapter'
  const rx = props.liveTraffic?.rx_mbps || 0
  const tx = props.liveTraffic?.tx_mbps || 0
  const hostIP = props.localIp || props.connectionInfo?.ip || '127.0.0.1'
  const ssid = props.networkId || props.connectionInfo?.ssid || 'Active Wi-Fi'
  nodes.push({
    icon: '💻',
    name: 'Host Monitoring Machine (This Device)',
    type: `${iface} · ${ssid}`,
    ip: hostIP,
    role: props.connectionInfo?.link_rate ? `Wi-Fi PHY ${props.connectionInfo.link_rate} Mbps` : 'Primary Host Adapter',
    metric: `↓ ${rx.toFixed(2)}M · ↑ ${tx.toFixed(2)}M`,
    speedColor: '#0ea5e9',
    status: 'ACTIVE',
    statusClass: 'bg-emerald-500/15 text-emerald-400 border border-emerald-500/30'
  })

  // 2. Gateway Router
  const gw = props.targets.find(t => t.host.toLowerCase() === 'gateway')
  if (gw) {
    const isDown = gw.status === 'down' || gw.loss >= 100
    const isSlow = gw.latency > 35
    nodes.push({
      icon: '📡',
      name: 'Default Gateway Router',
      type: 'Local Subnet Gateway',
      ip: gw.ip || props.connectionInfo?.gateway || '10.10.0.1',
      role: 'Default Route & DHCP',
      metric: gw.latency > 0 ? `${gw.latency.toFixed(1)} ms` : '—',
      speedColor: isSlow ? '#f59e0b' : '#10b981',
      status: isDown ? 'OFFLINE' : (isSlow ? 'DEGRADED' : 'ONLINE'),
      statusClass: isDown ? 'bg-rose-500/15 text-rose-400 border border-rose-500/30' : (isSlow ? 'bg-amber-500/15 text-amber-400 border border-amber-500/30' : 'bg-emerald-500/15 text-emerald-400 border border-emerald-500/30')
    })
  }

  // 3. Monitored Remote Targets
  props.targets.filter(t => t.host.toLowerCase() !== 'gateway').slice(0, 3).forEach(t => {
    const isDown = t.status === 'down' || t.loss >= 100
    nodes.push({
      icon: '🌐',
      name: `${t.host} Probe`,
      type: 'External Edge Probe',
      ip: t.ip || 'Resolving…',
      role: 'ICMP Latency Target',
      metric: t.latency > 0 ? `${t.latency.toFixed(1)} ms` : 'Down',
      speedColor: '#38bdf8',
      status: isDown ? 'OFFLINE' : 'ONLINE',
      statusClass: isDown ? 'bg-rose-500/15 text-rose-400 border border-rose-500/30' : 'bg-cyan-500/15 text-cyan-400 border border-cyan-500/30'
    })
  })

  return nodes
})

function getClientStatus(client) {
  if (client.rx_mbps > 25 || client.tx_mbps > 15) return 'SUSPICIOUS'
  if (client.is_active || client.rx_mbps > 0.05 || client.tx_mbps > 0.05) return 'ACTIVE'
  return 'IDLE'
}

function getClientStatusClass(client) {
  const status = getClientStatus(client)
  if (status === 'SUSPICIOUS') {
    return 'bg-amber-500/15 text-amber-400 border border-amber-500/30'
  }
  if (status === 'ACTIVE') {
    return 'bg-emerald-500/15 text-emerald-400 border border-emerald-500/30'
  }
  return 'bg-[var(--card)] text-slate-400 border border-[var(--border)]'
}

function formatMbps(v) {
  if (!v || v <= 0) return '0.00 M'
  return `${v.toFixed(2)} M`
}

function formatBytes(bytes) {
  if (!bytes || bytes <= 0) return '0 B'
  if (bytes >= 1048576 * 1024) return (bytes / (1048576 * 1024)).toFixed(2) + ' GB'
  if (bytes >= 1048576) return (bytes / 1048576).toFixed(1) + ' MB'
  if (bytes >= 1024) return (bytes / 1024).toFixed(0) + ' KB'
  return bytes + ' B'
}
</script>
