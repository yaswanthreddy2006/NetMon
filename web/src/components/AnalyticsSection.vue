<template>
  <div class="card p-4 sm:p-5 flex flex-col gap-4">
    <!-- Header -->
    <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-2 pb-2 border-b border-[var(--border)]">
      <div class="flex items-center gap-2">
        <div class="w-8 h-8 rounded-lg bg-purple-500/10 border border-purple-500/20 flex items-center justify-center text-purple-400">
          <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <polyline points="22 12 18 12 15 21 9 3 6 12 2 12"/>
          </svg>
        </div>
        <div>
          <h2 class="text-xs sm:text-sm font-black tracking-wide uppercase text-[var(--fg)] flex items-center gap-2">
            Historical Telemetry &amp; Analytics
          </h2>
          <p class="text-[11px] text-[var(--muted)]">
            Longitudinal trends, bandwidth distribution, and connection health score audit
          </p>
        </div>
      </div>

      <div class="flex items-center gap-2">
        <span class="text-[11px] font-mono text-[var(--muted)] px-2 py-0.5 rounded bg-[var(--bg-secondary)] border border-[var(--border)]">
          Timeline Window: {{ currentMinutes >= 60 ? (currentMinutes / 60) + 'h' : currentMinutes + 'm' }}
        </span>
      </div>
    </div>

    <!-- 4 Core High-Definition ApexCharts -->
    <div class="grid grid-cols-1 lg:grid-cols-2 gap-3.5">
      <ChartPanel
        :key="'lat-' + chartKey"
        :config="chartConfigs.latency"
        :label="chartLabels.latency"
        title="Round-Trip Latency History (ms)"
        dot="var(--accent)"
        label-id="latencyChart"
        :icon="CHART_ICONS.latency"
      />
      <ChartPanel
        :key="'spd-' + chartKey"
        :config="chartConfigs.speed"
        :label="chartLabels.speed"
        title="Upload vs Download Throughput (Mbps)"
        dot="var(--green)"
        label-id="speedChart"
        :icon="CHART_ICONS.speed"
      />
      <ChartPanel
        :key="'los-' + chartKey"
        :config="chartConfigs.loss"
        :label="chartLabels.loss"
        title="Packet Loss (%) &amp; Jitter (ms)"
        dot="var(--red)"
        label-id="lossChart"
        :icon="CHART_ICONS.loss"
      />
      <ChartPanel
        :key="'dns-' + chartKey"
        :config="chartConfigs.dns"
        :label="chartLabels.dns"
        title="DNS Query Resolution Time (ms)"
        dot="var(--purple)"
        label-id="dnsChart"
        :icon="CHART_ICONS.dns"
      />
    </div>

    <!-- Summary Distribution & Health Score Audit -->
    <div class="grid grid-cols-1 lg:grid-cols-3 gap-3.5 pt-2 border-t border-[var(--border)]">
      <div class="lg:col-span-2">
        <AccordionCard section="health_audit" padding="p-3.5">
          <template #title>
            <span class="flex items-center gap-2 text-xs font-bold uppercase tracking-wider text-[var(--fg)]">
              <svg xmlns="http://www.w3.org/2000/svg" width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round" class="text-emerald-400"><path d="M22 12h-4l-3 9L9 3l-3 9H2"/></svg>
              Connection Health Engine &amp; Factor Breakdown
            </span>
          </template>
          <HealthRing :summary="summary" :diagnosis="diagnosis" />
        </AccordionCard>
      </div>

      <QuickStats :summary="summary" :history="history" />
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import ChartPanel    from './ChartPanel.vue'
import AccordionCard from './AccordionCard.vue'
import HealthRing    from './HealthRing.vue'
import QuickStats    from './QuickStats.vue'

const props = defineProps({
  summary:        { type: Object, default: () => null },
  history:        { type: Array, default: () => [] },
  speedTests:     { type: Array, default: () => [] },
  latestSpeedTest:{ type: Object, default: () => null },
  diagnosis:      { type: Object, default: () => null },
  chartKey:       { type: Number, default: 0 },
  isDark:         { type: Boolean, default: false },
  currentMinutes: { type: Number, default: 60 },
  isSpeedTesting: { type: Boolean, default: false },
})

const SVG = (d) => `<svg xmlns="http://www.w3.org/2000/svg" width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">${d}</svg>`
const CHART_ICONS = {
  latency: SVG('<circle cx="12" cy="12" r="10"/><polyline points="12 6 12 12 16 14"/>'),
  speed:   SVG('<polygon points="13 2 3 14 12 14 11 22 21 10 12 10 13 2"/>'),
  loss:    SVG('<polyline points="22 12 18 12 15 21 9 3 6 12 2 12"/>'),
  dns:     SVG('<rect x="2" y="2" width="20" height="8" rx="2" ry="2"/><rect x="2" y="14" width="20" height="8" rx="2" ry="2"/><line x1="6" y1="6" x2="6.01" y2="6"/><line x1="6" y1="18" x2="6.01" y2="18"/>'),
}

const chartConfigs = computed(() => {
  const h      = props.history
  const times  = h.map(x => x.time)
  const st     = props.speedTests
  const isSmall = window.innerWidth < 640
  const tick    = Math.min(times.length, isSmall ? 4 : 6)
  const ls      = { colors: 'var(--muted)', fontSize: isSmall ? '9px' : '10px', fontFamily: "'Inter', monospace" }
  const dark    = props.isDark

  const noData = {
    text: 'Awaiting telemetry samples…',
    align: 'center', verticalAlign: 'middle',
    style: { color: dark ? '#7487a3' : '#94a3b8', fontSize: '12px', fontFamily: "'Inter', sans-serif" },
  }

  const base = {
    chart: { type: 'area', background: 'transparent', fontFamily: "'Inter', sans-serif", toolbar: { show: false }, height: 210, animations: { speed: 450 }, zoom: { enabled: false } },
    stroke: { curve: 'smooth', width: 2 },
    markers: { size: times.length > 0 && times.length <= 5 ? 4 : 0, hover: { size: 5 } },
    grid: { borderColor: 'var(--border)', strokeDashArray: 4, xaxis: { lines: { show: false } }, yaxis: { lines: { show: true } }, padding: { top: -8, right: 8, bottom: 0, left: 14 } },
    xaxis: { categories: times, tickAmount: tick, labels: { style: ls, rotate: 0, hideOverlappingLabels: true, maxHeight: 30 }, axisBorder: { show: true, color: 'var(--border)' }, axisTicks: { show: false } },
    yaxis: { labels: { style: ls }, forceNiceScale: true, axisBorder: { show: true, color: 'var(--border)' }, axisTicks: { show: false } },
    legend: { position: 'top', fontSize: '10px', labels: { colors: 'var(--subtle)' }, markers: { width: 6, height: 6, radius: 2 }, itemMargin: { horizontal: 8 } },
    tooltip: { theme: dark ? 'dark' : 'light', style: { fontSize: '11px' }, x: { show: true } },
    dataLabels: { enabled: false },
    noData,
  }
  const fill = (o = 0.12) => ({ type: 'gradient', gradient: { shadeIntensity: 0, opacityFrom: o, opacityTo: 0.01, stops: [0, 95] } })

  let speedCategories = times
  let speedSeries = [
    { name: 'Download (Mbps)', data: h.map(x => x.download || null) },
    { name: 'Upload (Mbps)', data: h.map(x => x.upload || null) }
  ]
  if (st.length > 0) {
    speedCategories = st.map(x => {
      const raw = x.time || x.ts || ''
      const t = new Date(raw)
      return isNaN(t.getTime()) ? (raw.length > 16 ? raw.slice(11, 16) : raw) : t.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
    })
    speedSeries = [
      { name: 'Download (Mbps)', data: st.map(x => x.download_mbps) },
      { name: 'Upload (Mbps)', data: st.map(x => x.upload_mbps) }
    ]
  }

  return {
    latency: {
      ...base,
      series: [{ name: 'Latency (ms)', data: h.map(x => x.latency) }],
      colors: ['#0ea5e9'],
      fill: fill(),
      yaxis: { ...base.yaxis, min: 0 }
    },
    speed: {
      ...base,
      xaxis: { ...base.xaxis, categories: speedCategories },
      markers: { size: speedCategories.length > 0 ? 5 : 0, hover: { size: 6 } },
      series: speedSeries,
      colors: ['#10b981', '#a855f7'],
      fill: fill(0.08)
    },
    loss: {
      ...base,
      series: [
        { name: 'Loss (%)', data: h.map(x => x.loss) },
        { name: 'Jitter (ms)', data: h.map(x => x.jitter) }
      ],
      colors: ['#f43f5e', '#f59e0b'],
      fill: fill(0.1),
      yaxis: [
        { forceNiceScale: true, min: 0, axisBorder: { show: true, color: 'var(--border)' }, axisTicks: { show: false }, title: { text: 'Loss %', style: { fontSize: '10px', color: 'var(--muted)' } }, labels: { style: { colors: 'var(--muted)', fontSize: '10px' } } },
        { opposite: true, forceNiceScale: true, min: 0, axisBorder: { show: false }, axisTicks: { show: false }, title: { text: 'Jitter ms', style: { fontSize: '10px', color: 'var(--muted)' } }, labels: { style: { colors: 'var(--muted)', fontSize: '10px' } } },
      ],
    },
    dns: {
      ...base,
      series: [{ name: 'DNS (ms)', data: h.map(x => x.dns || null) }],
      colors: ['#a855f7'],
      fill: fill(),
      yaxis: { ...base.yaxis, min: 0 }
    },
  }
})

const chartLabels = computed(() => {
  if (!props.summary) return { latency: '—', speed: '—', loss: '—', dns: '—' }
  const s = props.summary
  const latestSt = props.latestSpeedTest

  let speedText = '—'
  if (props.isSpeedTesting) {
    speedText = 'Testing…'
  } else if (latestSt && latestSt.status !== 'failed' && (latestSt.download_mbps > 0 || latestSt.upload_mbps > 0)) {
    speedText = `↓ ${latestSt.download_mbps}  ↑ ${latestSt.upload_mbps} Mbps`
  } else if (s.download_avg > 0) {
    speedText = `24h Avg: ↓ ${s.download_avg}  ↑ ${s.upload_avg} Mbps`
  }

  return {
    latency: s.latency_avg > 0 ? `${s.latency_avg} ms` : '—',
    speed:   speedText,
    loss:    `${s.packet_loss}% · ${s.jitter_avg} ms`,
    dns:     s.dns_avg > 0 ? `${s.dns_avg} ms` : '—',
  }
})
</script>
