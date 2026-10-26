<template>
  <div class="flex flex-col items-center">
    <div v-if="!summary" class="card-empty">
      <svg xmlns="http://www.w3.org/2000/svg" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75" stroke-linecap="round" stroke-linejoin="round"><path d="M22 12h-4l-3 9L9 3l-3 9H2"/></svg>
      <span>No data yet</span>
    </div>

    <template v-else>
      <div class="health-ring cursor-pointer" @click="showBreakdown = !showBreakdown" :title="'Click to ' + (showBreakdown ? 'hide' : 'view') + ' score breakdown'">
        <svg viewBox="0 0 100 100" width="100" height="100">
          <circle class="ring-bg" cx="50" cy="50" r="42" />
          <circle class="ring-fg" cx="50" cy="50" r="42"
            stroke-dasharray="263.89"
            :stroke-dashoffset="dashOffset"
            :style="{ stroke: evalResult.rating.color }"
          />
        </svg>
        <div class="health-score-text">
          <span class="health-score-num" :style="{ color: evalResult.rating.color }">{{ evalResult.score }}</span>
          <span class="health-score-label">{{ evalResult.rating.label }}</span>
        </div>
      </div>

      <!-- Diagnostic Insight -->
      <div class="w-full mt-2.5 px-2.5 py-2 rounded bg-[var(--card-bg)] border border-[var(--border)] text-center">
        <div class="flex items-center justify-between text-[9px] text-muted uppercase tracking-wider font-semibold">
          <span>Fault Diagnosis</span>
          <span v-if="diagnosis?.confidence" class="text-blue-400 font-mono">{{ Math.round(diagnosis.confidence * 100) }}% conf</span>
        </div>
        <div class="text-[11px] font-semibold leading-snug mt-1" :style="{ color: diagColor }">
          {{ diagSummary }}
        </div>
        <div v-if="diagReason" class="text-[10px] text-muted leading-tight mt-0.5 text-left bg-black/20 p-1.5 rounded mt-1.5">
          {{ diagReason }}
        </div>
      </div>

      <!-- Component Badges Grid -->
      <div class="grid grid-cols-3 gap-1.5 w-full mt-2.5">
        <div v-if="evalResult.components.latency" class="flex flex-col items-center p-1 rounded bg-[var(--card-bg)] border border-[var(--border)]">
          <span class="health-metric-value" :style="{ color: evalResult.components.latency.rating.color }">{{ evalResult.components.latency.rating.label }}</span>
          <span class="health-metric-label">Latency</span>
        </div>
        <div v-if="evalResult.components.loss" class="flex flex-col items-center p-1 rounded bg-[var(--card-bg)] border border-[var(--border)]">
          <span class="health-metric-value" :style="{ color: evalResult.components.loss.rating.color }">{{ evalResult.components.loss.rating.label }}</span>
          <span class="health-metric-label">Loss</span>
        </div>
        <div v-if="evalResult.components.jitter" class="flex flex-col items-center p-1 rounded bg-[var(--card-bg)] border border-[var(--border)]">
          <span class="health-metric-value" :style="{ color: evalResult.components.jitter.rating.color }">{{ evalResult.components.jitter.rating.label }}</span>
          <span class="health-metric-label">Jitter</span>
        </div>
        <div v-if="evalResult.components.dns" class="flex flex-col items-center p-1 rounded bg-[var(--card-bg)] border border-[var(--border)]">
          <span class="health-metric-value" :style="{ color: evalResult.components.dns.rating.color }">{{ evalResult.components.dns.rating.label }}</span>
          <span class="health-metric-label">DNS</span>
        </div>
        <div v-if="evalResult.components.uptime" class="flex flex-col items-center p-1 rounded bg-[var(--card-bg)] border border-[var(--border)]">
          <span class="health-metric-value" :style="{ color: evalResult.components.uptime.rating.color }">{{ evalResult.components.uptime.rating.label }}</span>
          <span class="health-metric-label">Uptime</span>
        </div>
        <div v-if="evalResult.components.speed" class="flex flex-col items-center p-1 rounded bg-[var(--card-bg)] border border-[var(--border)]">
          <span class="health-metric-value" :style="{ color: evalResult.components.speed.rating.color }">{{ evalResult.components.speed.rating.label }}</span>
          <span class="health-metric-label">Speed</span>
        </div>
      </div>

      <!-- Expandable Detailed Breakdown -->
      <div v-if="showBreakdown" class="w-full mt-2 flex flex-col gap-1 text-[11px] p-2 rounded bg-[var(--card-bg)] border border-[var(--border)]">
        <div class="text-[10px] uppercase font-bold text-muted mb-0.5">Scoring Breakdown</div>
        <div v-for="(comp, key) in evalResult.components" :key="key" class="flex justify-between items-center py-0.5 border-b border-[var(--border)] last:border-0">
          <span class="text-muted">{{ comp.name }} ({{ comp.raw || '—' }})</span>
          <span class="font-semibold" :style="{ color: comp.rating.color }">{{ comp.score }}/100</span>
        </div>
      </div>
    </template>
  </div>
</template>

<script setup>
import { ref, computed } from 'vue'
import { evaluateHealth } from '../utils/health.js'

const props = defineProps({
  summary: Object,
  diagnosis: Object,
})

const showBreakdown = ref(false)
const CIRC = 2 * Math.PI * 42

const evalResult = computed(() => evaluateHealth(props.summary))
const dashOffset = computed(() => CIRC - (evalResult.value.score / 100) * CIRC)

const diagSummary = computed(() => {
  if (props.diagnosis?.summary) return props.diagnosis.summary
  return evalResult.value.mainIssue
})

const diagReason = computed(() => {
  if (props.diagnosis?.reason) return props.diagnosis.reason
  return evalResult.value.reasons.length ? evalResult.value.reasons[0] : ''
})

const diagColor = computed(() => {
  if (props.diagnosis?.status === 'offline') return 'var(--red)'
  if (props.diagnosis?.status === 'degraded') return 'var(--yellow)'
  return 'var(--green)'
})
</script>
