<template>
  <div class="card p-4 sm:p-5 flex flex-col gap-4 relative overflow-hidden border border-purple-500/25 bg-gradient-to-br from-purple-950/15 via-[var(--card)] to-[var(--bg-secondary)]">
    <!-- Neural Glow Accents -->
    <div class="absolute -top-20 -right-20 w-44 h-44 bg-purple-500/10 rounded-full blur-3xl pointer-events-none" />
    <div class="absolute -bottom-20 -left-20 w-44 h-44 bg-cyan-500/10 rounded-full blur-3xl pointer-events-none" />

    <!-- Security Header -->
    <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-2 pb-3 border-b border-[var(--border)] relative z-10">
      <div class="flex items-center gap-2.5">
        <div class="w-9 h-9 rounded-xl bg-purple-500/15 border border-purple-500/30 flex items-center justify-center text-purple-400 shadow-[0_0_12px_rgba(168,85,247,0.25)]">
          <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.25" stroke-linecap="round" stroke-linejoin="round">
            <path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/>
            <path d="m9 12 2 2 4-4"/>
          </svg>
        </div>
        <div>
          <div class="flex items-center gap-2">
            <h2 class="text-xs sm:text-sm font-black tracking-wide uppercase text-[var(--fg)] flex items-center gap-1.5">
              Security, ML Telemetry &amp; Anomaly Detection
            </h2>
            <span class="text-[9px] font-mono px-2 py-0.5 rounded bg-purple-500/15 text-purple-300 border border-purple-500/25 font-bold uppercase">
              Security Operations Center
            </span>
          </div>
          <p class="text-[11px] text-[var(--muted)]">
            Continuous threat posture, behavioral anomaly vectors, and incident alerts
          </p>
        </div>
      </div>

      <!-- Security Status Badge -->
      <div class="flex items-center gap-2">
        <span
          class="inline-flex items-center gap-1.5 px-3 py-1 rounded-full text-xs font-black tracking-wider uppercase shadow-sm"
          :class="predictionBadgeClass"
        >
          <span class="w-2 h-2 rounded-full animate-pulse" :class="predictionDotClass" />
          {{ metrics.status }}
        </span>
      </div>
    </div>

    <!-- Core Security Status Cards -->
    <div class="grid grid-cols-2 lg:grid-cols-4 gap-3 relative z-10">
      
      <!-- 1. SECURITY STATUS -->
      <div class="p-3.5 rounded-xl border border-[var(--border)] bg-[var(--card)] flex flex-col justify-between">
        <span class="text-[10px] uppercase font-bold text-[var(--muted)] tracking-wider">
          Security Status
        </span>
        <div class="my-1.5 flex items-baseline gap-1.5">
          <span class="text-lg sm:text-xl font-black" :class="predictionTextClass">
            {{ metrics.status === 'NORMAL' ? 'Clean Baseline' : (metrics.status === 'ELEVATED' ? 'Elevated Strain' : 'Suspicious Anomaly') }}
          </span>
        </div>
        <div class="text-[10px] font-mono text-[var(--muted)] truncate">
          {{ faultDomainLabel }}
        </div>
      </div>

      <!-- 2. ANOMALY SCORE -->
      <div class="p-3.5 rounded-xl border border-[var(--border)] bg-[var(--card)] flex flex-col justify-between">
        <div class="flex items-center justify-between">
          <span class="text-[10px] uppercase font-bold text-[var(--muted)] tracking-wider">Anomaly Score</span>
          <span class="text-[9px] font-mono font-bold" :class="scoreSeverityClass">
            {{ metrics.anomalyScore > 0.6 ? 'HIGH' : (metrics.anomalyScore > 0.3 ? 'MODERATE' : 'OPTIMAL') }}
          </span>
        </div>
        <div class="my-1 flex items-baseline gap-2">
          <span class="text-2xl font-black font-mono" :class="scoreSeverityClass">
            {{ metrics.anomalyScore.toFixed(2) }}
          </span>
          <span class="text-[11px] text-[var(--muted)] font-mono">/ 1.00</span>
        </div>
        <div class="w-full h-1.5 rounded-full bg-[var(--border)] overflow-hidden">
          <div
            class="h-full rounded-full transition-all duration-700"
            :style="{ width: Math.max(5, metrics.anomalyScore * 100) + '%' }"
            :class="scoreBarClass"
          />
        </div>
      </div>

      <!-- 3. MODEL CONFIDENCE -->
      <div class="p-3.5 rounded-xl border border-[var(--border)] bg-[var(--card)] flex flex-col justify-between">
        <span class="text-[10px] uppercase font-bold text-[var(--muted)] tracking-wider">
          Model Confidence
        </span>
        <div class="my-1.5 flex items-baseline gap-2">
          <span class="text-2xl font-black font-mono text-cyan-400">
            {{ metrics.confidence }}%
          </span>
          <span class="text-[10px] text-[var(--muted)]">Certainty</span>
        </div>
        <div class="text-[10px] text-[var(--muted)] font-mono truncate">
          {{ targets.length || 2 }} verified probes
        </div>
      </div>

      <!-- 4. ACTIVE ALERTS COUNT -->
      <div class="p-3.5 rounded-xl border border-[var(--border)] bg-[var(--card)] flex flex-col justify-between">
        <span class="text-[10px] uppercase font-bold text-[var(--muted)] tracking-wider">
          Incident Events
        </span>
        <div class="my-1.5 flex items-baseline gap-2">
          <span class="text-2xl font-black font-mono" :class="activeAlerts.length > 0 ? 'text-amber-400' : 'text-emerald-400'">
            {{ activeAlerts.length }}
          </span>
          <span class="text-[10px] text-[var(--muted)]">Active</span>
        </div>
        <div class="text-[10px] font-semibold" :class="activeAlerts.length > 0 ? 'text-amber-400' : 'text-emerald-400'">
          {{ activeAlerts.length > 0 ? 'Action Recommended' : 'All Clear' }}
        </div>
      </div>

    </div>

    <!-- Active Alerts with "WHY THIS WAS FLAGGED" -->
    <div class="flex flex-col gap-2.5 relative z-10">
      <div class="text-[10px] font-bold uppercase tracking-wider text-[var(--muted)] flex items-center justify-between">
        <span>Security Alerts &amp; Explanations</span>
        <span class="text-[9px] font-mono text-purple-400">Contextual Telemetry Rules</span>
      </div>

      <div v-if="activeAlerts.length > 0" class="flex flex-col gap-2.5">
        <div
          v-for="alert in activeAlerts"
          :key="alert.id"
          class="p-4 rounded-xl border flex flex-col gap-2.5 transition-all"
          :class="getAlertContainerClass(alert.severity)"
        >
          <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-1.5">
            <div class="flex items-center gap-2 min-w-0">
              <span
                class="text-[9px] font-mono px-2 py-0.5 rounded font-black uppercase tracking-wider flex-shrink-0"
                :class="getAlertBadgeClass(alert.severity)"
              >
                {{ alert.severity }}
              </span>
              <span class="font-bold text-xs text-[var(--fg)] truncate">
                {{ alert.title }}
              </span>
            </div>

            <div class="flex items-center gap-3 text-[10px] text-[var(--muted)] font-mono">
              <span>Metric: <strong class="text-[var(--fg)]">{{ alert.metric }}</strong></span>
              <span>Value: <strong class="text-[var(--fg)]">{{ alert.value }}</strong></span>
              <span>{{ alert.time }}</span>
            </div>
          </div>

          <!-- WHY THIS WAS FLAGGED (Requirement 5) -->
          <div class="p-3 rounded-lg bg-[var(--card)] border border-[var(--border)] flex flex-col gap-1">
            <div class="text-[10px] font-bold uppercase tracking-wider text-purple-400 flex items-center gap-1">
              <span>WHY THIS WAS FLAGGED:</span>
            </div>
            <p class="text-xs text-[var(--fg)] font-medium leading-relaxed">
              {{ alert.whyFlagged }}
            </p>
            <div class="text-[10px] text-[var(--muted)] mt-1 pt-1 border-t border-[var(--border)] flex items-center gap-1">
              <span class="font-semibold text-[var(--subtle)]">Recommended action:</span>
              <span>{{ alert.action }}</span>
            </div>
          </div>
        </div>
      </div>

      <!-- No Alerts Clean State -->
      <div v-else class="p-5 rounded-xl border border-[var(--border)] bg-[var(--card)] flex items-center gap-3">
        <div class="w-8 h-8 rounded-full bg-emerald-500/10 border border-emerald-500/20 flex items-center justify-center text-emerald-400 flex-shrink-0">
          <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
            <path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"/>
            <polyline points="22 4 12 14.01 9 11.01"/>
          </svg>
        </div>
        <div class="min-w-0">
          <div class="text-xs font-bold text-[var(--fg)]">Clean Threat &amp; Anomaly Profile</div>
          <div class="text-[11px] text-[var(--muted)]">
            Traffic is operating strictly within the normal observed pattern. No anomalous packet drop or route congestion detected.
          </div>
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
  diagnosis: { type: Object, default: () => null },
  targets: { type: Array, default: () => [] },
  history: { type: Array, default: () => [] },
  now: { type: Number, default: () => Date.now() },
})

const metrics = computed(() => {
  return computeAnomalyMetrics({
    summary: props.summary,
    latestMeasurement: props.latestMeasurement,
    liveTraffic: props.liveTraffic,
    diagnosis: props.diagnosis,
    history: props.history,
  })
})

const predictionBadgeClass = computed(() => {
  if (metrics.value.status === 'SUSPICIOUS') {
    return 'bg-rose-500/20 text-rose-300 border border-rose-500/40'
  }
  if (metrics.value.status === 'ELEVATED') {
    return 'bg-amber-500/20 text-amber-300 border border-amber-500/40'
  }
  return 'bg-emerald-500/20 text-emerald-300 border border-emerald-500/40'
})

const predictionDotClass = computed(() => {
  if (metrics.value.status === 'SUSPICIOUS') return 'bg-rose-400'
  if (metrics.value.status === 'ELEVATED') return 'bg-amber-400'
  return 'bg-emerald-400'
})

const predictionTextClass = computed(() => {
  if (metrics.value.status === 'SUSPICIOUS') return 'text-rose-400'
  if (metrics.value.status === 'ELEVATED') return 'text-amber-400'
  return 'text-emerald-400'
})

const scoreSeverityClass = computed(() => {
  const s = metrics.value.anomalyScore
  if (s >= 0.70) return 'text-rose-400'
  if (s >= 0.40) return 'text-amber-400'
  return 'text-emerald-400'
})

const scoreBarClass = computed(() => {
  const s = metrics.value.anomalyScore
  if (s >= 0.70) return 'bg-rose-500'
  if (s >= 0.40) return 'bg-amber-500'
  return 'bg-emerald-500'
})

const faultDomainLabel = computed(() => {
  const cause = props.diagnosis?.likely_cause || 'none'
  switch (cause) {
    case 'local_network': return 'Local Gateway / Wi-Fi Strain'
    case 'isp_upstream':  return 'ISP Upstream Congestion'
    case 'dns':           return 'DNS Resolver Delay'
    case 'disconnection': return 'Network Disconnect'
    case 'traffic_saturation': return 'Traffic Saturation'
    default: return 'Optimal End-to-End Route'
  }
})

// Active Alerts with explicit "WHY THIS WAS FLAGGED"
const activeAlerts = computed(() => {
  const alerts = []
  const nowStr = new Date(props.now).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' })
  const s = props.summary
  const lm = props.latestMeasurement
  const tr = props.liveTraffic

  // 1. Packet Loss
  const loss = s?.packet_loss ?? lm?.loss ?? 0
  if (loss > 1.0) {
    alerts.push({
      id: 'loss',
      severity: loss > 5.0 ? 'CRITICAL' : 'WARNING',
      title: 'Packet Loss Threshold Exceeded',
      metric: 'Packet Loss',
      value: `${loss.toFixed(1)}%`,
      time: nowStr,
      whyFlagged: `Packet drop reached ${loss.toFixed(1)}%, which is higher than the normal observed zero-loss baseline.`,
      action: 'Check Wi-Fi signal interference, switch wireless channel, or test wired Ethernet link.'
    })
  }

  // 2. High Latency
  const lat = s?.latency_avg ?? lm?.latency ?? 0
  if (lat > 80.0) {
    alerts.push({
      id: 'latency',
      severity: 'WARNING',
      title: 'Elevated Round-Trip Latency',
      metric: 'Latency RTT',
      value: `${lat.toFixed(0)} ms`,
      whyFlagged: `Latency (${lat.toFixed(0)} ms) exceeds the normal threshold; packet round-trip time is significantly delayed.`,
      action: 'Inspect concurrent heavy downloads or test connection directly to the router gateway.'
    })
  }

  // 3. Traffic Saturation / High Upload
  const tx = tr?.tx_mbps ?? 0
  if (tx > 20.0) {
    alerts.push({
      id: 'high-upload',
      severity: 'WARNING',
      title: 'High Outbound Upload Traffic',
      metric: 'Upload Rate',
      value: `${tx.toFixed(1)} Mbps`,
      whyFlagged: 'Upload activity increased significantly compared with the current baseline, utilizing high outbound capacity.',
      action: 'Check cloud synchronization software (OneDrive/Dropbox) or background video streaming.'
    })
  }

  // 4. ML Anomaly Detection Flag
  if (metrics.value.status === 'SUSPICIOUS') {
    alerts.push({
      id: 'ml-suspicious',
      severity: 'SUSPICIOUS',
      title: 'ML Anomaly Signature Detected',
      metric: 'Anomaly Score',
      value: metrics.value.anomalyScore.toFixed(2),
      time: nowStr,
      whyFlagged: 'Traffic is higher than the normal observed pattern and multi-probe latency variance spiked.',
      action: 'Review active connected devices and high-bandwidth socket applications.'
    })
  }

  return alerts
})

function getAlertContainerClass(sev) {
  switch (sev) {
    case 'CRITICAL':
      return 'border-rose-500/30 bg-rose-950/20'
    case 'SUSPICIOUS':
      return 'border-amber-500/30 bg-amber-950/20'
    case 'WARNING':
      return 'border-amber-500/20 bg-amber-950/10'
    default:
      return 'border-cyan-500/20 bg-cyan-950/10'
  }
}

function getAlertBadgeClass(sev) {
  switch (sev) {
    case 'CRITICAL':
      return 'bg-rose-500/20 text-rose-300 border border-rose-500/40'
    case 'SUSPICIOUS':
      return 'bg-amber-500/20 text-amber-300 border border-amber-500/40'
    case 'WARNING':
      return 'bg-amber-500/20 text-amber-300 border border-amber-500/40'
    default:
      return 'bg-cyan-500/20 text-cyan-300 border border-cyan-500/40'
  }
}
</script>
