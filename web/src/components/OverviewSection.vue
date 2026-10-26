<template>
  <div class="card p-4 sm:p-5 flex flex-col gap-4">
    <!-- Section Title & Overview Header -->
    <div class="flex items-center justify-between pb-2 border-b border-[var(--border)]">
      <div class="flex items-center gap-2">
        <div class="w-8 h-8 rounded-lg bg-cyan-500/10 border border-cyan-500/20 flex items-center justify-center text-cyan-400">
          <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <rect width="7" height="7" x="3" y="3" rx="1"/>
            <rect width="7" height="7" x="14" y="3" rx="1"/>
            <rect width="7" height="7" x="14" y="14" rx="1"/>
            <rect width="7" height="7" x="3" y="14" rx="1"/>
          </svg>
        </div>
        <div>
          <h2 class="text-xs sm:text-sm font-black tracking-wide uppercase text-[var(--fg)] flex items-center gap-2">
            Overview &amp; Executive Telemetry
          </h2>
          <p class="text-[11px] text-[var(--muted)]">
            High-level live operational status, bandwidth throughput, and security posture
          </p>
        </div>
      </div>
      <div class="text-[10px] font-mono text-[var(--muted)] px-2 py-0.5 rounded bg-[var(--bg-secondary)] border border-[var(--border)]">
        Live Observability
      </div>
    </div>

    <!-- 6 Primary Modern Overview Cards -->
    <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3.5">
      
      <!-- 1. NETWORK STATUS -->
      <div class="p-4 rounded-xl border border-[var(--border)] bg-[var(--bg-secondary)] flex flex-col justify-between transition-all hover:border-emerald-500/40">
        <div>
          <div class="flex items-center justify-between">
            <span class="text-[10px] uppercase font-bold text-[var(--muted)] tracking-wider flex items-center gap-1.5">
              <svg xmlns="http://www.w3.org/2000/svg" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="text-emerald-400"><circle cx="12" cy="12" r="10"/><line x1="2" y1="12" x2="22" y2="12"/><path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z"/></svg>
              Network Status
            </span>
            <span class="w-2.5 h-2.5 rounded-full animate-pulse" :class="isOnline ? 'bg-emerald-400' : 'bg-rose-500'" />
          </div>
          <div class="my-2 flex items-baseline gap-2">
            <span class="text-xl sm:text-2xl font-black tracking-tight" :class="isOnline ? 'text-emerald-400' : 'text-rose-400'">
              {{ isOnline ? 'Online & Stable' : 'Connection Offline' }}
            </span>
          </div>
        </div>
        <!-- Dynamic Explanation based strictly on existing metrics -->
        <div class="p-2 rounded-lg bg-[var(--card)] border border-[var(--border)] text-[10px] text-[var(--subtle)] leading-relaxed mt-1">
          <span class="font-bold text-[var(--fg)]">Status Context: </span>
          <span>{{ networkStatusExplanation }}</span>
        </div>
      </div>

      <!-- 2. CONNECTED DEVICES -->
      <div class="p-4 rounded-xl border border-[var(--border)] bg-[var(--bg-secondary)] flex flex-col justify-between transition-all hover:border-blue-500/40">
        <div>
          <div class="flex items-center justify-between">
            <span class="text-[10px] uppercase font-bold text-blue-400 tracking-wider flex items-center gap-1.5">
              <svg xmlns="http://www.w3.org/2000/svg" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect width="16" height="12" x="4" y="4" rx="2"/><line x1="12" x2="12" y1="16" y2="20"/><line x1="8" x2="16" y1="20" y2="20"/></svg>
              Connected Devices
            </span>
            <span class="text-[9px] font-mono font-bold px-1.5 py-0.2 rounded" :class="hotspot?.active ? 'bg-emerald-500/15 text-emerald-400' : 'bg-[var(--bg)] text-[var(--muted)]'">
              {{ hotspot?.active ? 'Hotspot ON' : 'Host Only' }}
            </span>
          </div>
          <div class="my-2 flex items-baseline gap-2">
            <span class="text-xl sm:text-2xl font-black font-mono text-blue-400">
              {{ deviceCount }}
            </span>
            <span class="text-xs font-semibold text-[var(--fg)]">
              Device{{ deviceCount === 1 ? '' : 's' }} Active
            </span>
          </div>
        </div>
        <div class="p-2 rounded-lg bg-[var(--card)] border border-[var(--border)] text-[10px] text-[var(--subtle)] leading-relaxed mt-1">
          <span class="font-bold text-[var(--fg)]">Device Context: </span>
          <span>{{ deviceExplanation }}</span>
        </div>
      </div>

      <!-- 3. DOWNLOAD SPEED -->
      <div class="p-4 rounded-xl border border-[var(--border)] bg-[var(--bg-secondary)] flex flex-col justify-between transition-all hover:border-emerald-500/40">
        <div>
          <div class="flex items-center justify-between">
            <span class="text-[10px] uppercase font-bold text-emerald-400 tracking-wider flex items-center gap-1.5">
              <svg xmlns="http://www.w3.org/2000/svg" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><line x1="12" y1="5" x2="12" y2="19"/><polyline points="19 12 12 19 5 12"/></svg>
              Download Speed
            </span>
            <span class="text-[10px] font-mono font-bold" :class="downloadTrend.color">
              {{ downloadTrend.text }}
            </span>
          </div>
          <div class="my-2 flex items-baseline gap-1">
            <span class="text-xl sm:text-2xl font-black font-mono text-emerald-400">
              {{ formatSpeed(currentDownload) }}
            </span>
          </div>
        </div>
        <div class="p-2 rounded-lg bg-[var(--card)] border border-[var(--border)] text-[10px] text-[var(--subtle)] leading-relaxed mt-1">
          <span class="font-bold text-[var(--fg)]">Why is it {{ isDownloadHigh ? 'high' : 'low' }}? </span>
          <span>{{ downloadExplanation }}</span>
        </div>
      </div>

      <!-- 4. UPLOAD SPEED -->
      <div class="p-4 rounded-xl border border-[var(--border)] bg-[var(--bg-secondary)] flex flex-col justify-between transition-all hover:border-purple-500/40">
        <div>
          <div class="flex items-center justify-between">
            <span class="text-[10px] uppercase font-bold text-purple-400 tracking-wider flex items-center gap-1.5">
              <svg xmlns="http://www.w3.org/2000/svg" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><line x1="12" y1="19" x2="12" y2="5"/><polyline points="5 12 12 5 19 12"/></svg>
              Upload Speed
            </span>
            <span class="text-[10px] font-mono font-bold" :class="uploadTrend.color">
              {{ uploadTrend.trend }}
            </span>
          </div>
          <div class="my-2 flex items-baseline gap-1">
            <span class="text-xl sm:text-2xl font-black font-mono text-purple-400">
              {{ formatSpeed(currentUpload) }}
            </span>
          </div>
        </div>
        <div class="p-2 rounded-lg bg-[var(--card)] border border-[var(--border)] text-[10px] text-[var(--subtle)] leading-relaxed mt-1">
          <span class="font-bold text-[var(--fg)]">Why is it {{ isUploadHigh ? 'high' : 'low' }}? </span>
          <span>{{ uploadExplanation }}</span>
        </div>
      </div>

      <!-- 5. NETWORK USAGE (TOTAL BANDWIDTH) -->
      <div class="p-4 rounded-xl border border-[var(--border)] bg-[var(--bg-secondary)] flex flex-col justify-between transition-all hover:border-cyan-500/40">
        <div>
          <div class="flex items-center justify-between">
            <span class="text-[10px] uppercase font-bold text-cyan-400 tracking-wider flex items-center gap-1.5">
              <svg xmlns="http://www.w3.org/2000/svg" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M22 12h-4l-3 9L9 3l-3 9H2"/></svg>
              Network Usage
            </span>
            <span class="text-[9px] font-mono font-bold px-1.5 py-0.2 rounded" :class="isUsageHigh ? 'bg-amber-500/15 text-amber-400' : 'bg-emerald-500/15 text-emerald-400'">
              {{ isUsageHigh ? 'HEAVY I/O' : 'NORMAL LOAD' }}
            </span>
          </div>
          <div class="my-2 flex items-baseline gap-2">
            <span class="text-xl sm:text-2xl font-black font-mono text-cyan-400">
              {{ formatSpeed(totalMbps) }}
            </span>
            <span class="text-[10px] font-mono text-[var(--muted)]">Combined Rx/Tx</span>
          </div>
        </div>
        <div class="p-2 rounded-lg bg-[var(--card)] border border-[var(--border)] text-[10px] text-[var(--subtle)] leading-relaxed mt-1">
          <span class="font-bold text-[var(--fg)]">Usage Analysis: </span>
          <span>{{ networkUsageExplanation }}</span>
        </div>
      </div>

      <!-- 6. SECURITY STATUS -->
      <div class="p-4 rounded-xl border border-[var(--border)] bg-[var(--bg-secondary)] flex flex-col justify-between transition-all hover:border-purple-500/40">
        <div>
          <div class="flex items-center justify-between">
            <span class="text-[10px] uppercase font-bold text-purple-400 tracking-wider flex items-center gap-1.5">
              <svg xmlns="http://www.w3.org/2000/svg" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/></svg>
              Security &amp; ML Status
            </span>
            <span class="text-[9px] font-mono font-bold px-1.5 py-0.2 rounded" :class="securityBadgeClass">
              {{ anomalyData.confidence }}% CONF
            </span>
          </div>
          <div class="my-2 flex items-baseline gap-2">
            <span class="text-xl sm:text-2xl font-black tracking-tight" :class="securityTextClass">
              {{ anomalyData.status === 'NORMAL' ? 'Normal Posture' : (anomalyData.status === 'ELEVATED' ? 'Elevated Strain' : 'Suspicious Anomaly') }}
            </span>
            <span class="text-xs font-mono text-[var(--muted)]">({{ anomalyData.anomalyScore.toFixed(2) }})</span>
          </div>
        </div>
        <div class="p-2 rounded-lg bg-[var(--card)] border border-[var(--border)] text-[10px] text-[var(--subtle)] leading-relaxed mt-1">
          <span class="font-bold text-[var(--fg)]">Security Context: </span>
          <span>{{ securityExplanation }}</span>
        </div>
      </div>

    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { computeAnomalyMetrics } from '../utils/anomaly.js'

const props = defineProps({
  summary: { type: Object, default: () => null },
  latestMeasurement: { type: Object, default: () => null },
  liveTraffic: { type: Object, default: () => null },
  speedTest: { type: Object, default: () => null },
  hotspot: { type: Object, default: () => null },
  diagnosis: { type: Object, default: () => null },
  targets: { type: Array, default: () => [] },
  history: { type: Array, default: () => [] },
  isOnline: { type: Boolean, default: true },
})

const anomalyData = computed(() => {
  return computeAnomalyMetrics({
    summary: props.summary,
    latestMeasurement: props.latestMeasurement,
    liveTraffic: props.liveTraffic,
    diagnosis: props.diagnosis,
    history: props.history,
  })
})

const liveRx = computed(() => props.liveTraffic?.rx_mbps || 0)
const liveTx = computed(() => props.liveTraffic?.tx_mbps || 0)
const totalMbps = computed(() => liveRx.value + liveTx.value)

const currentDownload = computed(() => {
  if (liveRx.value > 0) return liveRx.value
  if (props.speedTest?.download_mbps > 0) return props.speedTest.download_mbps
  return props.summary?.download_avg || 0
})

const currentUpload = computed(() => {
  if (liveTx.value > 0) return liveTx.value
  if (props.speedTest?.upload_mbps > 0) return props.speedTest.upload_mbps
  return props.summary?.upload_avg || 0
})

const isDownloadHigh = computed(() => currentDownload.value >= 8.0)
const isUploadHigh   = computed(() => currentUpload.value >= 3.0)
const isUsageHigh    = computed(() => totalMbps.value >= 12.0)

// Dynamic explanations generated strictly from real measurements
const networkStatusExplanation = computed(() => {
  if (!props.isOnline) {
    return 'Internet reachability is lost. Probes to external targets are experiencing 100% packet loss.'
  }
  const gw = props.targets.find(t => t.host.toLowerCase() === 'gateway')
  if (gw && gw.latency > 35) {
    return `Local gateway ping is slow (${gw.latency.toFixed(1)} ms), which may cause mild response delays.`
  }
  const loss = props.summary?.packet_loss ?? 0
  if (loss > 0) {
    return `Connection is active but experiencing ${loss.toFixed(1)}% packet drop.`
  }
  return 'Gateway router and external cloud endpoints are responding within optimal latency limits.'
})

const deviceCount = computed(() => {
  if (props.hotspot?.active) {
    return (props.hotspot.client_count || props.hotspot.clients?.length || 0) + 1
  }
  return 1
})

const deviceExplanation = computed(() => {
  if (props.hotspot?.active) {
    const clients = props.hotspot.client_count || props.hotspot.clients?.length || 0
    return `${clients} wireless client${clients === 1 ? '' : 's'} connected via Windows Mobile Hotspot.`
  }
  return 'Hotspot is currently turned off. Monitoring the host system adapter.'
})

const downloadTrend = computed(() => {
  const avg = props.summary?.download_avg || 0
  const curr = currentDownload.value
  if (avg <= 0 || curr <= 0) return { text: 'Stable', color: 'text-emerald-400' }
  const pct = Math.round(((curr - avg) / avg) * 100)
  if (Math.abs(pct) < 5) return { text: 'Stable', color: 'text-emerald-400' }
  if (pct > 0) return { text: `↑ ${pct}% vs avg`, color: 'text-emerald-400' }
  return { text: `↓ ${Math.abs(pct)}% vs avg`, color: 'text-amber-400' }
})

const uploadTrend = computed(() => {
  const avg = props.summary?.upload_avg || 0
  const curr = currentUpload.value
  if (avg <= 0 || curr <= 0) return { trend: 'Stable', color: 'text-purple-400' }
  const pct = Math.round(((curr - avg) / avg) * 100)
  if (Math.abs(pct) < 5) return { trend: 'Stable', color: 'text-purple-400' }
  if (pct > 0) return { trend: `↑ ${pct}% vs avg`, color: 'text-purple-400' }
  return { trend: `↓ ${Math.abs(pct)}% vs avg`, color: 'text-amber-400' }
})

const downloadExplanation = computed(() => {
  if (isDownloadHigh.value) {
    return 'Download usage is high because active socket transfer or streaming traffic increased significantly.'
  }
  return 'Download usage is low because background application bandwidth consumption is below normal load.'
})

const uploadExplanation = computed(() => {
  if (isUploadHigh.value) {
    return 'Upload usage is elevated because outbound transfers or cloud sync services are transferring data.'
  }
  return 'Upload activity is low because outbound buffers and applications are currently in standby.'
})

const networkUsageExplanation = computed(() => {
  if (isUsageHigh.value) {
    return 'Network usage is high because download/upload traffic increased significantly.'
  }
  return 'Network usage is low because current bandwidth consumption is below the normal range.'
})

const securityExplanation = computed(() => {
  if (anomalyData.value.status === 'SUSPICIOUS') {
    return `ML flagged unusual pattern (score ${anomalyData.value.anomalyScore.toFixed(2)}): ${anomalyData.value.reasons[0]?.text || 'Deviation detected.'}`
  }
  if (anomalyData.value.status === 'ELEVATED') {
    return 'Minor telemetry variance detected; packet transmission is stable.'
  }
  return 'Traffic pattern and ICMP probe behaviors conform to clean baseline expectations.'
})

const securityBadgeClass = computed(() => {
  if (anomalyData.value.status === 'SUSPICIOUS') return 'bg-rose-500/15 text-rose-400 border border-rose-500/30'
  if (anomalyData.value.status === 'ELEVATED') return 'bg-amber-500/15 text-amber-400 border border-amber-500/30'
  return 'bg-purple-500/15 text-purple-400 border border-purple-500/30'
})

const securityTextClass = computed(() => {
  if (anomalyData.value.status === 'SUSPICIOUS') return 'text-rose-400'
  if (anomalyData.value.status === 'ELEVATED') return 'text-amber-400'
  return 'text-emerald-400'
})

function formatSpeed(mbps) {
  if (!mbps || mbps <= 0) return '0.00 Mbps'
  if (mbps >= 1000) return (mbps / 1000).toFixed(2) + ' Gbps'
  return mbps.toFixed(2) + ' Mbps'
}
</script>
