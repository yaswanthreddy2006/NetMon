<template>
  <div class="min-h-screen px-3 sm:px-6 lg:px-8 py-2 max-w-[1720px] mx-auto flex flex-col gap-4">

    <!-- 1. TOP DYNAMIC FLOATING NAVIGATION BAR (Reference Inspiration) -->
    <DynamicNavBar
      :active-section="activeSection"
      :status="statusBadge.class"
      :status-label="statusBadge.label"
      :target-count="targetCount"
      :device-count="deviceCount"
      :live-throughput="liveThroughputBadge"
      :alert-count="alertCount"
      :is-refreshing="isLoading"
      :is-speed-testing="isSpeedTesting"
      :is-dark="isDark"
      @toggle-theme="$emit('toggle-theme')"
      @open-settings="settingsOpen = true"
      @refresh-now="loadData(true)"
      @run-speed-test="runManualSpeedTest"
      @select-section="onSelectSection"
    >
      <template #duration-picker>
        <DurationPicker v-model="currentMinutes" @change="onDurationChange" />
      </template>
    </DynamicNavBar>

    <!-- 2. SECTION: OVERVIEW -->
    <section id="sec-overview" class="scroll-mt-20">
      <OverviewSection
        :summary="summary"
        :latest-measurement="latestMeasurement"
        :live-traffic="liveTraffic"
        :speed-test="latestSpeedTest"
        :hotspot="hotspot"
        :diagnosis="diagnosis"
        :targets="targets"
        :history="history"
        :is-online="isOnline"
      />
    </section>

    <!-- 3. SECTION: NETWORK -->
    <section id="sec-network" class="scroll-mt-20">
      <NetworkSection
        :targets="targets"
        :summary="summary"
        :live-traffic="liveTraffic"
        :hotspot="hotspot"
        :latest-measurement="latestMeasurement"
        :dns="dns"
        :history="history"
        :is-online="isOnline"
        :now="now"
        :connection-info="connectionInfo"
        :local-ip="localIP"
        :gateway-ip="gatewayIP"
        :network-id="networkID"
      />
    </section>

    <!-- 4. SECTION: DEVICES -->
    <section id="sec-devices" class="scroll-mt-20">
      <ConnectedDevicesSection
        :hotspot="hotspot"
        :live-traffic="liveTraffic"
        :targets="targets"
        :latest-measurement="latestMeasurement"
        :connection-info="connectionInfo"
        :local-ip="localIP"
        :network-id="networkID"
      />
    </section>

    <!-- 5. SECTION: TRAFFIC -->
    <section id="sec-traffic" class="scroll-mt-20">
      <TrafficSection
        :live-traffic="liveTraffic"
        :app-traffic="appTraffic"
        :speed-test="latestSpeedTest"
        :summary="summary"
        :is-speed-testing="isSpeedTesting"
        :now="now"
        @run-speed-test="runManualSpeedTest"
      />
    </section>

    <!-- 6. SECTION: SECURITY -->
    <section id="sec-security" class="scroll-mt-20">
      <SecuritySection
        :summary="summary"
        :latest-measurement="latestMeasurement"
        :live-traffic="liveTraffic"
        :diagnosis="diagnosis"
        :targets="targets"
        :history="history"
        :now="now"
      />
    </section>

    <!-- 7. SECTION: ANALYTICS -->
    <section id="sec-analytics" class="scroll-mt-20">
      <AnalyticsSection
        :summary="summary"
        :history="history"
        :speed-tests="speedTests"
        :latest-speed-test="latestSpeedTest"
        :diagnosis="diagnosis"
        :chart-key="chartKey"
        :is-dark="isDark"
        :current-minutes="currentMinutes"
        :is-speed-testing="isSpeedTesting"
      />
    </section>

    <!-- SETTINGS MODAL -->
    <SettingsModal
      :open="settingsOpen"
      @close="settingsOpen = false"
      @saved="loadData(true)"
    />

  </div>
</template>

<script setup>
import { ref, computed, watch, onMounted, onUnmounted } from 'vue'
import DynamicNavBar           from './DynamicNavBar.vue'
import DurationPicker          from './DurationPicker.vue'
import SettingsModal           from './SettingsModal.vue'
import OverviewSection         from './OverviewSection.vue'
import NetworkSection          from './NetworkSection.vue'
import ConnectedDevicesSection from './ConnectedDevicesSection.vue'
import TrafficSection          from './TrafficSection.vue'
import SecuritySection         from './SecuritySection.vue'
import AnalyticsSection        from './AnalyticsSection.vue'
import { useToasts }           from '../composables/useToasts.js'
import { computeAnomalyMetrics } from '../utils/anomaly.js'

const props = defineProps({ isDark: Boolean })
defineEmits(['toggle-theme'])

const REFRESH_MS = 15000

const { show: showToast } = useToasts()

// ── State ───────────────────────────────────────────────────────────────
const apiData        = ref(null)
const apiConfig      = ref(null)
const currentMinutes = ref(60)
const isLoading      = ref(false)
const isSpeedTesting = ref(false)
const lastUpdateTime = ref(null)
const prevStatus     = ref(null)
const chartKey       = ref(0)
const settingsOpen   = ref(false)
const now            = ref(Date.now())
const activeSection  = ref('overview')

// Recreate charts when theme changes
watch(() => props.isDark, () => { chartKey.value++ })

// ── Ticker ──────────────────────────────────────────────────────────────
const ticker = setInterval(() => {
  now.value = Date.now()
}, 1000)

// ── Scroll Spy (IntersectionObserver) ───────────────────────────────────
let observer = null

function initScrollSpy() {
  const sections = ['overview', 'network', 'devices', 'traffic', 'security', 'analytics']
  const options = {
    root: null,
    rootMargin: '-20% 0px -70% 0px',
    threshold: 0
  }

  observer = new IntersectionObserver((entries) => {
    entries.forEach(entry => {
      if (entry.isIntersecting) {
        const id = entry.target.id.replace('sec-', '')
        activeSection.value = id
      }
    })
  }, options)

  sections.forEach(s => {
    const el = document.getElementById(`sec-${s}`)
    if (el) observer.observe(el)
  })
}

function onSelectSection(id) {
  activeSection.value = id
}

// ── Data loading ────────────────────────────────────────────────────────
let activeRequestId = 0

async function loadData(forceRecreate = false) {
  const reqId = ++activeRequestId
  isLoading.value = true
  try {
    const [d, cfg] = await Promise.all([
      fetch(`/api/data?minutes=${currentMinutes.value}`).then(r => r.json()),
      fetch('/api/config').then(r => r.json()),
    ])
    if (reqId !== activeRequestId) return

    if (cfg) {
      const allowedPing = new Set(cfg.ping_targets.map(h => h.toLowerCase()))
      d.targets = (d.targets || []).filter(t => t.host.toLowerCase() === 'gateway' || allowedPing.has(t.host.toLowerCase()))
      const allowedDns = new Set(cfg.dns_targets.map(h => h.toLowerCase()))
      d.dns = (d.dns || []).filter(x => allowedDns.has(x.host.toLowerCase()))
    }

    apiData.value        = d
    apiConfig.value      = cfg
    lastUpdateTime.value = Date.now()
    checkThresholds(statusBadge.value)
    if (forceRecreate) chartKey.value++
  } catch (_) { /* maintain previous state on network error */ }
  if (reqId === activeRequestId) {
    isLoading.value = false
  }
}

async function runManualSpeedTest() {
  if (isSpeedTesting.value) return
  isSpeedTesting.value = true
  showToast('st-start', 'Bandwidth Test Initiated', 'Running download & upload throughput benchmark…', 'info', 4000)
  try {
    const res = await fetch('/api/speedtest/run', { method: 'POST' })
    const data = await res.json()
    if (!res.ok) {
      showToast('st-fail', 'Speed Test Failed', data.error || 'Test failed', 'error', 6000)
    } else {
      const st = data.speed_test
      showToast('st-done', 'Speed Test Complete', `↓ ${st.download_mbps} Mbps · ↑ ${st.upload_mbps} Mbps`, 'ok', 5000)
    }
    await loadData(true)
  } catch (err) {
    showToast('st-err', 'Speed Test Error', err.message || 'Request failed', 'error', 6000)
  } finally {
    isSpeedTesting.value = false
  }
}

function checkThresholds(currentStatusBadge) {
  const curr = currentStatusBadge?.class || 'online'
  if (prevStatus.value !== null && prevStatus.value !== curr) {
    if (curr === 'offline') {
      showToast('st-offline', 'Connection Offline', 'Internet reachability lost or backend service paused', 'error', 8000)
    } else if (curr === 'online') {
      showToast('st-ok', 'Connection Online', 'Internet connectivity verified active', 'ok', 5000)
    }
  }
  prevStatus.value = curr
}

// ── Polling & Lifecycle ──────────────────────────────────────────────────
let pollTimer = null, resizeTimer = null

onMounted(() => {
  loadData()
  pollTimer = setInterval(loadData, REFRESH_MS)
  window.addEventListener('resize', onResize)
  setTimeout(initScrollSpy, 300)
})

onUnmounted(() => {
  clearInterval(pollTimer)
  clearInterval(ticker)
  window.removeEventListener('resize', onResize)
  if (observer) observer.disconnect()
})

function onResize() {
  clearTimeout(resizeTimer)
  resizeTimer = setTimeout(() => loadData(true), 250)
}

function onDurationChange(minutes) {
  currentMinutes.value = minutes
  loadData(true)
}

// ── Computed Data ─────────────────────────────────────────────────────────
const summary           = computed(() => apiData.value?.summary)
const history           = computed(() => apiData.value?.history || [])
const targets           = computed(() => apiData.value?.targets || [])
const dns               = computed(() => apiData.value?.dns || [])
const diagnosis         = computed(() => apiData.value?.diagnosis || null)
const speedTests        = computed(() => apiData.value?.speed_tests || [])
const liveTraffic       = computed(() => apiData.value?.live_traffic || null)
const appTraffic        = computed(() => apiData.value?.application_traffic || null)
const hotspot           = computed(() => apiData.value?.hotspot || null)
const latestMeasurement = computed(() => apiData.value?.latest_measurement || (history.value.length ? history.value[history.value.length - 1] : null))
const latestSpeedTest   = computed(() => apiData.value?.speed_test || (speedTests.value.length ? speedTests.value[speedTests.value.length - 1] : null))
const targetCount       = computed(() => apiConfig.value ? apiConfig.value.ping_targets.length : targets.value.length)
const connectionInfo    = computed(() => apiData.value?.connection_info || null)
const localIP           = computed(() => apiData.value?.local_ip || '')
const gatewayIP         = computed(() => apiData.value?.gateway_ip || '')
const networkID         = computed(() => apiData.value?.network_id || '')

const isOnline = computed(() => {
  return statusBadge.value.class === 'online'
})

const statusBadge = computed(() => {
  if (!apiData.value || !lastUpdateTime.value) {
    return { class: 'offline', label: 'Connecting…' }
  }
  const timeSinceLastUpdate = now.value - lastUpdateTime.value
  const staleThreshold = Math.max(REFRESH_MS * 3, 45000)
  if (timeSinceLastUpdate > staleThreshold) {
    return { class: 'offline', label: 'Offline' }
  }

  const tList = targets.value || []
  const remoteTargets = tList.filter(t => t.host.toLowerCase() !== 'gateway')

  if (remoteTargets.length > 0) {
    const hasActiveRemoteProbe = remoteTargets.some(t => t.status === 'up' && t.loss < 100)
    if (!hasActiveRemoteProbe) {
      return { class: 'offline', label: 'Offline' }
    }
    return { class: 'online', label: 'Online' }
  }

  const lm = latestMeasurement.value
  if (lm && lm.loss >= 100) {
    return { class: 'offline', label: 'Offline' }
  }

  return { class: 'online', label: 'Online' }
})

const deviceCount = computed(() => {
  if (hotspot.value?.active) {
    return (hotspot.value.client_count || hotspot.value.clients?.length || 0) + 1
  }
  return 1
})

const liveThroughputBadge = computed(() => {
  const rx = liveTraffic.value?.rx_mbps || 0
  const tx = liveTraffic.value?.tx_mbps || 0
  if (rx + tx > 0.05) {
    return `↓ ${rx.toFixed(1)}M · ↑ ${tx.toFixed(1)}M`
  }
  return null
})

const alertCount = computed(() => {
  const s = summary.value
  const lm = latestMeasurement.value
  let count = 0
  const loss = s?.packet_loss ?? lm?.loss ?? 0
  if (loss > 1.0) count++
  const lat = s?.latency_avg ?? lm?.latency ?? 0
  if (lat > 80.0) count++
  const anomaly = computeAnomalyMetrics({
    summary: s,
    latestMeasurement: lm,
    liveTraffic: liveTraffic.value,
    diagnosis: diagnosis.value,
    history: history.value,
  })
  if (anomaly.status === 'SUSPICIOUS') count++
  return count
})
</script>

<style scoped>
.scroll-mt-20 {
  scroll-margin-top: 5rem;
}
</style>
