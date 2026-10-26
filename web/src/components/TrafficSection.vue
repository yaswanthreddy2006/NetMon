<template>
  <div class="card p-4 sm:p-5 flex flex-col gap-4">
    <!-- Traffic Header -->
    <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-2 pb-2 border-b border-[var(--border)]">
      <div class="flex items-center gap-2">
        <div class="w-8 h-8 rounded-lg bg-emerald-500/10 border border-emerald-500/20 flex items-center justify-center text-emerald-400">
          <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
            <line x1="12" y1="19" x2="12" y2="5"/>
            <polyline points="5 12 12 5 19 12"/>
            <line x1="19" y1="12" x2="19" y2="19"/>
            <polyline points="15 15 19 19 23 15"/>
          </svg>
        </div>
        <div>
          <h2 class="text-xs sm:text-sm font-black tracking-wide uppercase text-[var(--fg)] flex items-center gap-2">
            Network Traffic &amp; Bandwidth Dynamics
          </h2>
          <p class="text-[11px] text-[var(--muted)]">
            Continuous interface throughput telemetry, socket activity, and causal rate analytics
          </p>
        </div>
      </div>

      <div class="flex items-center gap-2">
        <span class="text-[11px] px-2.5 py-1 rounded-full bg-[var(--bg-secondary)] border border-[var(--border)] font-mono text-[var(--muted)] flex items-center gap-1.5">
          <span class="w-2 h-2 rounded-full bg-emerald-400 animate-pulse" />
          {{ interfaceName }} · {{ isTrafficStale ? 'Paused' : 'Streaming Real-Time' }}
        </span>
      </div>
    </div>

    <!-- 4 Primary Traffic Cards with Clear DOWNLOAD and UPLOAD Separation -->
    <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-3.5">
      
      <!-- 1. DOWNLOAD (GREEN ACCENT) -->
      <div class="p-4 rounded-xl border border-emerald-500/30 bg-emerald-950/10 flex flex-col justify-between relative overflow-hidden transition-all hover:border-emerald-500/50">
        <div>
          <div class="flex items-center justify-between">
            <span class="text-xs font-black uppercase text-emerald-400 tracking-wider flex items-center gap-1.5">
              <svg xmlns="http://www.w3.org/2000/svg" width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><line x1="12" y1="5" x2="12" y2="19"/><polyline points="19 12 12 19 5 12"/></svg>
              DOWNLOAD
            </span>
            <span class="text-[9px] font-mono font-bold px-2 py-0.5 rounded bg-emerald-500/15 text-emerald-400 border border-emerald-500/25">
              INBOUND (Rx)
            </span>
          </div>
          <div class="my-2.5 flex items-baseline gap-1">
            <span class="text-2xl sm:text-3xl font-black font-mono text-emerald-400 transition-all duration-300">
              {{ formatSpeed(liveRx) }}
            </span>
          </div>
          <div class="text-[11px] font-mono text-[var(--muted)]">
            Byte counter: {{ formatBytes(liveTraffic?.rx_bytes_sec) }}/s
          </div>
        </div>

        <!-- Why is it high / low explanation -->
        <div class="mt-3 pt-2.5 border-t border-[var(--border)] text-[10px] text-[var(--subtle)] leading-relaxed">
          <span class="font-bold text-emerald-400 block mb-0.5">
            {{ isDownloadHigh ? 'Why is download high?' : 'Why is download low?' }}
          </span>
          <span>{{ downloadCausalText }}</span>
        </div>
      </div>

      <!-- 2. UPLOAD (PURPLE ACCENT) -->
      <div class="p-4 rounded-xl border border-purple-500/30 bg-purple-950/10 flex flex-col justify-between relative overflow-hidden transition-all hover:border-purple-500/50">
        <div>
          <div class="flex items-center justify-between">
            <span class="text-xs font-black uppercase text-purple-400 tracking-wider flex items-center gap-1.5">
              <svg xmlns="http://www.w3.org/2000/svg" width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><line x1="12" y1="19" x2="12" y2="5"/><polyline points="5 12 12 5 19 12"/></svg>
              UPLOAD
            </span>
            <span class="text-[9px] font-mono font-bold px-2 py-0.5 rounded bg-purple-500/15 text-purple-400 border border-purple-500/25">
              OUTBOUND (Tx)
            </span>
          </div>
          <div class="my-2.5 flex items-baseline gap-1">
            <span class="text-2xl sm:text-3xl font-black font-mono text-purple-400 transition-all duration-300">
              {{ formatSpeed(liveTx) }}
            </span>
          </div>
          <div class="text-[11px] font-mono text-[var(--muted)]">
            Byte counter: {{ formatBytes(liveTraffic?.tx_bytes_sec) }}/s
          </div>
        </div>

        <!-- Why is it high / low explanation -->
        <div class="mt-3 pt-2.5 border-t border-[var(--border)] text-[10px] text-[var(--subtle)] leading-relaxed">
          <span class="font-bold text-purple-400 block mb-0.5">
            {{ isUploadHigh ? 'Why is upload high?' : 'Why is upload low?' }}
          </span>
          <span>{{ uploadCausalText }}</span>
        </div>
      </div>

      <!-- 3. TOTAL TRAFFIC -->
      <div class="p-4 rounded-xl border border-[var(--border)] bg-[var(--bg-secondary)] flex flex-col justify-between transition-all hover:border-cyan-500/40">
        <div>
          <div class="flex items-center justify-between">
            <span class="text-xs font-black uppercase text-cyan-400 tracking-wider">
              TOTAL TRAFFIC
            </span>
            <span class="text-[9px] font-mono text-[var(--muted)]">Combined I/O</span>
          </div>
          <div class="my-2.5 flex items-baseline gap-1">
            <span class="text-2xl sm:text-3xl font-black font-mono text-[var(--fg)]">
              {{ formatSpeed(totalMbps) }}
            </span>
          </div>
          <div class="text-[11px] font-mono text-[var(--muted)]">
            Aggregated: {{ formatBytes((liveTraffic?.rx_bytes_sec || 0) + (liveTraffic?.tx_bytes_sec || 0)) }}/s
          </div>
        </div>

        <div class="mt-3 pt-2.5 border-t border-[var(--border)] text-[10px] text-[var(--subtle)] leading-relaxed">
          <span class="font-bold text-cyan-400 block mb-0.5">Traffic Dynamics:</span>
          <span>{{ totalTrafficCausalText }}</span>
        </div>
      </div>

      <!-- 4. CURRENT BANDWIDTH BENCHMARK -->
      <div class="p-4 rounded-xl border border-[var(--border)] bg-[var(--bg-secondary)] flex flex-col justify-between transition-all hover:border-amber-500/40">
        <div>
          <div class="flex items-center justify-between">
            <span class="text-xs font-black uppercase text-amber-400 tracking-wider">
              CURRENT BANDWIDTH
            </span>
            <span class="text-[9px] font-mono text-[var(--muted)]">{{ speedTestAgoText }}</span>
          </div>
          <div class="my-2.5 flex items-baseline gap-2">
            <span class="text-xl font-bold font-mono text-emerald-400">
              ↓ {{ latestSpeedTest?.download_mbps ? latestSpeedTest.download_mbps : '—' }}M
            </span>
            <span class="text-xl font-bold font-mono text-purple-400">
              ↑ {{ latestSpeedTest?.upload_mbps ? latestSpeedTest.upload_mbps : '—' }}M
            </span>
          </div>
          <div class="text-[11px] font-mono text-[var(--muted)] truncate">
            Baseline capacity reference
          </div>
        </div>

        <div class="mt-3 pt-2.5 border-t border-[var(--border)] flex items-center justify-between">
          <span class="text-[10px] text-[var(--muted)] font-mono">Benchmark</span>
          <button
            type="button"
            class="text-[10px] font-bold px-2 py-0.5 rounded bg-emerald-500/10 text-emerald-400 hover:bg-emerald-500/20 border border-emerald-500/20 transition-all flex items-center gap-1"
            :disabled="isSpeedTesting"
            @click="$emit('run-speed-test')"
          >
            <span v-if="isSpeedTesting" class="animate-spin text-[8px]">●</span>
            {{ isSpeedTesting ? 'Testing…' : 'Run Speed Test' }}
          </button>
        </div>
      </div>

    </div>

    <!-- Application Bandwidth Breakdown (Process level) -->
    <div class="p-4 rounded-xl border border-[var(--border)] bg-[var(--bg-secondary)] flex flex-col gap-3">
      <div class="flex items-center justify-between">
        <span class="text-xs font-black uppercase tracking-wider text-[var(--fg)] flex items-center gap-2">
          <svg xmlns="http://www.w3.org/2000/svg" width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="text-cyan-400">
            <rect width="7" height="7" x="3" y="3" rx="1"/>
            <rect width="7" height="7" x="14" y="3" rx="1"/>
            <rect width="7" height="7" x="14" y="14" rx="1"/>
            <rect width="7" height="7" x="3" y="14" rx="1"/>
          </svg>
          Active Process Network Sockets
        </span>
        <span class="text-[10px] font-mono text-[var(--muted)]">
          {{ appList.length }} Monitored Application{{ appList.length === 1 ? '' : 's' }}
        </span>
      </div>

      <div v-if="appList.length > 0" class="flex flex-col gap-2 max-h-[240px] overflow-y-auto pr-1">
        <div
          v-for="app in appList"
          :key="app.pid ? `${app.pid}-${app.process}` : app.name"
          class="p-2.5 rounded-lg bg-[var(--card)] border border-[var(--border)] flex flex-col gap-1.5 transition-all hover:border-[var(--border-hover)]"
        >
          <div class="grid grid-cols-[1fr_80px_80px_85px] items-center gap-2 text-xs">
            <div class="flex items-center gap-2 min-w-0">
              <span class="w-2 h-2 rounded-full flex-shrink-0 bg-cyan-400" />
              <div class="flex flex-col min-w-0">
                <span class="font-bold text-[11px] truncate text-[var(--fg)]">
                  {{ app.name }}
                </span>
                <span class="text-[9px] text-[var(--muted)] font-mono truncate">
                  {{ app.pid ? `PID ${app.pid} · ${app.process}` : app.process }}
                </span>
              </div>
            </div>

            <!-- Download -->
            <span class="text-right font-mono text-xs font-bold text-emerald-400">
              ↓ {{ formatSpeed(app.rx_mbps) }}
            </span>

            <!-- Upload -->
            <span class="text-right font-mono text-xs font-bold text-purple-400">
              ↑ {{ formatSpeed(app.tx_mbps) }}
            </span>

            <!-- Total -->
            <span class="text-right font-mono text-xs font-black text-[var(--fg)]">
              {{ formatSpeed(app.total_mbps) }}
            </span>
          </div>

          <!-- Progress Bar of relative consumption -->
          <div class="w-full h-1 rounded-full bg-[var(--border)] overflow-hidden">
            <div
              class="h-full rounded-full bg-gradient-to-r from-emerald-400 via-cyan-400 to-purple-400 transition-all duration-500"
              :style="{ width: Math.max(2, Math.min(100, (app.total_mbps / Math.max(0.1, totalMbps)) * 100)) + '%' }"
            />
          </div>
        </div>
      </div>

      <div v-else class="py-5 text-center text-xs text-[var(--muted)] flex flex-col items-center gap-1.5">
        <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="opacity-40">
          <circle cx="12" cy="12" r="10"/>
          <line x1="12" y1="8" x2="12" y2="12"/>
          <line x1="12" y1="16" x2="12.01" y2="16"/>
        </svg>
        <span>No background application socket transfers detected on this interface</span>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { formatTimeAgo, isStale } from '../utils/time.js'

const props = defineProps({
  liveTraffic: { type: Object, default: () => null },
  appTraffic: { type: Object, default: () => null },
  speedTest: { type: Object, default: () => null },
  summary: { type: Object, default: () => null },
  isSpeedTesting: { type: Boolean, default: false },
  now: { type: Number, default: () => Date.now() },
})

defineEmits(['run-speed-test'])

const liveRx = computed(() => props.liveTraffic?.rx_mbps || 0)
const liveTx = computed(() => props.liveTraffic?.tx_mbps || 0)
const totalMbps = computed(() => liveRx.value + liveTx.value)

const isDownloadHigh = computed(() => liveRx.value >= 8.0)
const isUploadHigh   = computed(() => liveTx.value >= 3.0)

const interfaceName = computed(() => {
  return props.liveTraffic?.interface || 'Active Network Adapter'
})

const isTrafficStale = computed(() => {
  if (!props.liveTraffic?.sampled_at) return true
  return isStale(props.liveTraffic.sampled_at, 6000, props.now)
})

const latestSpeedTest = computed(() => {
  return props.speedTest
})

const speedTestAgoText = computed(() => {
  if (!props.speedTest?.time && !props.speedTest?.ts) return 'No test recorded'
  return formatTimeAgo(props.speedTest.time || props.speedTest.ts, props.now)
})

const downloadCausalText = computed(() => {
  if (isDownloadHigh.value) {
    const top = appList.value[0]
    if (top && top.rx_mbps > 1.0) {
      return `Elevated: ${top.name} is consuming ${top.rx_mbps.toFixed(1)} Mbps.`
    }
    return 'Elevated: Media streaming, large downloads, or software update packages detected.'
  }
  return 'Light: No heavy file downloads detected; background connections idle.'
})

const uploadCausalText = computed(() => {
  if (isUploadHigh.value) {
    const top = appList.value.find(a => a.tx_mbps > 0.5)
    if (top) {
      return `Elevated: Outbound socket activity detected from ${top.name} (${top.tx_mbps.toFixed(1)} Mbps).`
    }
    return 'Elevated: Cloud synchronization or outbound multimedia transfer active.'
  }
  return 'Light: Minimal outbound transmission; background upload queues are clear.'
})

const totalTrafficCausalText = computed(() => {
  if (totalMbps.value > 15.0) {
    return 'Network interface is actively transferring high-bandwidth payloads.'
  }
  return 'Interface traffic is running comfortably below peak link capacity.'
})

const appList = computed(() => {
  const apps = props.appTraffic?.applications || []
  return [...apps].sort((a, b) => (b.total_mbps || 0) - (a.total_mbps || 0)).slice(0, 10)
})

function formatSpeed(mbps) {
  if (!mbps || mbps <= 0) return '0.00 Mbps'
  if (mbps >= 1000) return (mbps / 1000).toFixed(2) + ' Gbps'
  if (mbps >= 1) return mbps.toFixed(2) + ' Mbps'
  return (mbps * 1000).toFixed(0) + ' Kbps'
}

function formatBytes(bytesSec) {
  if (!bytesSec || bytesSec <= 0) return '0 B'
  if (bytesSec >= 1048576) return (bytesSec / 1048576).toFixed(1) + ' MB'
  if (bytesSec >= 1024) return (bytesSec / 1024).toFixed(1) + ' KB'
  return Math.round(bytesSec) + ' B'
}
</script>
