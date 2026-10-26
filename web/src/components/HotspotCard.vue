<template>
  <AccordionCard section="hotspot_monitor" padding="p-3">
    <template #title>
      <div class="flex items-center justify-between w-full pr-2">
        <span class="flex items-center gap-1.5 font-semibold text-xs text-[var(--fg)]">
          <svg xmlns="http://www.w3.org/2000/svg" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.25" stroke-linecap="round" stroke-linejoin="round" class="text-amber-500">
            <path d="M5 12.55a11 11 0 0 1 14.08 0"/>
            <path d="M1.42 9a16 16 0 0 1 21.16 0"/>
            <path d="M8.53 16.11a6 6 0 0 1 6.95 0"/>
            <circle cx="12" cy="20" r="1"/>
          </svg>
          HOTSPOT MONITOR
        </span>

        <!-- Hotspot status badge in header -->
        <span
          class="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-[10px] font-semibold tracking-wide"
          :class="isHotspotActive ? 'bg-emerald-500/10 text-emerald-500 border border-emerald-500/20' : 'bg-muted/10 text-muted border border-[var(--border)]'"
        >
          <span
            class="w-1.5 h-1.5 rounded-full"
            :class="isHotspotActive ? 'bg-emerald-500 animate-pulse' : 'bg-muted'"
          />
          {{ isHotspotActive ? 'Hotspot Active' : 'Hotspot Inactive' }}
        </span>
      </div>
    </template>

    <!-- INACTIVE STATE -->
    <div v-if="!isHotspotActive" class="card-empty py-4 text-center">
      <div class="w-8 h-8 rounded-full bg-[var(--bg-secondary)] flex items-center justify-center mx-auto mb-2 text-muted">
        <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <line x1="1" y1="1" x2="23" y2="23"/>
          <path d="M16.72 11.06A10.94 10.94 0 0 1 19 12.55"/>
          <path d="M5 12.55a10.94 10.94 0 0 1 5.17-2.39"/>
          <path d="M10.71 5.05A16 16 0 0 1 22.58 9"/>
          <path d="M1.42 9a15.91 15.91 0 0 1 4.7-2.88"/>
          <path d="M8.53 16.11a6 6 0 0 1 6.95 0"/>
          <circle cx="12" cy="20" r="1"/>
        </svg>
      </div>
      <div class="flex items-center justify-center gap-1.5 font-medium text-xs text-[var(--fg)]">
        <span class="text-muted text-sm">○</span> Hotspot Inactive
      </div>
      <p class="text-[10px] text-muted mt-1 max-w-[220px] mx-auto leading-relaxed">
        Turn on your hotspot to monitor connected devices and bandwidth.
      </p>
    </div>

    <!-- ACTIVE STATE -->
    <div v-else class="flex flex-col gap-2.5">

      <!-- Total Hotspot Bandwidth Hero Card -->
      <div class="p-2.5 rounded-lg bg-[var(--bg-secondary)] border border-[var(--border)] flex flex-col gap-2">
        <div class="flex items-center justify-between">
          <div class="flex items-center gap-1.5">
            <span class="text-[10px] uppercase font-bold tracking-wider text-muted">Current Usage</span>
            <span v-if="hotspot?.interface_name" class="text-[9px] px-1.5 py-0.2 rounded bg-[var(--card)] text-muted border border-[var(--border)] truncate max-w-[100px]" :title="hotspot.interface_name">
              {{ formatIfaceName(hotspot.interface_name) }}
            </span>
          </div>
          <span class="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-[10px] font-medium bg-blue-500/10 text-blue-400 border border-blue-500/20">
            {{ clientCount }} {{ clientCount === 1 ? 'Device Connected' : 'Devices Connected' }}
          </span>
        </div>

        <div class="grid grid-cols-2 gap-2 pt-0.5">
          <!-- Download Rate (PC -> Client) -->
          <div class="flex flex-col">
            <span class="text-[10px] text-emerald-400 font-medium flex items-center gap-0.5">
              <svg xmlns="http://www.w3.org/2000/svg" width="9" height="9" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><line x1="12" y1="5" x2="12" y2="19"/><polyline points="19 12 12 19 5 12"/></svg>
              ↓ Download
            </span>
            <span class="font-mono text-sm font-bold text-[var(--fg)]">
              {{ formatSpeed(hotspot?.total_rx_mbps) }}
            </span>
          </div>

          <!-- Upload Rate (Client -> PC) -->
          <div class="flex flex-col">
            <span class="text-[10px] text-purple-400 font-medium flex items-center gap-0.5">
              <svg xmlns="http://www.w3.org/2000/svg" width="9" height="9" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><line x1="12" y1="19" x2="12" y2="5"/><polyline points="5 12 12 5 19 12"/></svg>
              ↑ Upload
            </span>
            <span class="font-mono text-sm font-bold text-[var(--fg)]">
              {{ formatSpeed(hotspot?.total_tx_mbps) }}
            </span>
          </div>
        </div>
      </div>

      <!-- NO CLIENTS CONNECTED YET -->
      <div v-if="clientCount === 0" class="card-empty py-3 text-center">
        <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" class="mx-auto mb-1 opacity-60"><path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M23 21v-2a4 4 0 0 0-3-3.87"/><path d="M16 3.13a4 4 0 0 1 0 7.75"/></svg>
        <span class="text-xs text-muted">0 Devices Connected · Waiting for devices…</span>
      </div>

      <!-- CONNECTED DEVICES LIST -->
      <div v-else class="flex flex-col gap-2">
        <div class="flex items-center justify-between text-[10px] font-semibold text-muted uppercase tracking-wider px-0.5 border-b border-[var(--border)] pb-1">
          <span>CONNECTED DEVICES</span>
          <span>Live Usage</span>
        </div>

        <div class="flex flex-col gap-2 max-h-[260px] overflow-y-auto pr-0.5">
          <div
            v-for="device in clientList"
            :key="device.ip"
            class="flex flex-col gap-1.5 p-2 rounded-lg bg-[var(--card)] hover:bg-[var(--bg-hover)] border border-[var(--border)] transition-all duration-200"
          >
            <!-- Device Row Top -->
            <div class="flex items-center justify-between">
              <div class="flex items-center gap-2 min-w-0 pr-1">
                <!-- Device Emoji Icon -->
                <span class="text-base flex-shrink-0" :title="device.device_type">
                  {{ device.device_icon || '📱' }}
                </span>

                <!-- Hostname & IP/MAC -->
                <div class="flex flex-col min-w-0 leading-tight">
                  <div class="flex items-center gap-1.5">
                    <span class="font-semibold text-xs truncate text-[var(--fg)]" :title="device.hostname">
                      {{ device.hostname || 'Unknown Device' }}
                    </span>
                    <!-- Activity Dot -->
                    <span
                      class="inline-flex items-center gap-0.5 text-[9px] px-1 py-0.2 rounded font-medium"
                      :class="device.is_active ? 'bg-emerald-500/10 text-emerald-500' : 'bg-muted/10 text-muted'"
                    >
                      <span class="w-1.5 h-1.5 rounded-full" :class="device.is_active ? 'bg-emerald-500' : 'bg-muted'" />
                      {{ device.is_active ? 'Active' : 'Idle' }}
                    </span>
                  </div>
                  <span class="text-[9px] font-mono text-muted truncate">
                    {{ device.ip }} <span v-if="device.mac">· {{ device.mac }}</span>
                  </span>
                </div>
              </div>

              <!-- Rates -->
              <div class="flex flex-col items-end text-right flex-shrink-0">
                <template v-if="device.rx_mbps > 0 || device.tx_mbps > 0 || (hotspot && hotspot.total_mbps > 0)">
                  <span class="font-mono text-xs font-bold text-emerald-400">
                    ↓ {{ formatSpeed(device.rx_mbps) }}
                  </span>
                  <span class="font-mono text-[10px] text-purple-400">
                    ↑ {{ formatSpeed(device.tx_mbps) }}
                  </span>
                </template>
                <template v-else>
                  <span class="text-[9px] text-muted italic">
                    Traffic idle
                  </span>
                </template>
              </div>
            </div>

            <!-- Usage Proportion Bar -->
            <div class="w-full bg-[var(--bg-secondary)] h-1 rounded-full overflow-hidden">
              <div
                class="h-full rounded-full bg-gradient-to-r from-emerald-400 to-blue-500 transition-all duration-500"
                :style="{ width: `${device.percent > 0 ? Math.max(4, device.percent) : (device.is_active ? 8 : 2)}%` }"
              />
            </div>

            <!-- Bottom Sub-info (Total Data Transferred) -->
            <div v-if="device.total_rx_bytes > 0 || device.total_tx_bytes > 0" class="flex items-center justify-between text-[9px] text-muted pt-0.5">
              <span>Total Data</span>
              <span>
                ↓ {{ formatBytes(device.total_rx_bytes) }} · ↑ {{ formatBytes(device.total_tx_bytes) }}
              </span>
            </div>
          </div>
        </div>
      </div>

      <!-- Card Footer -->
      <div class="card-divider my-0.5" />
      <div class="flex items-center justify-between text-[10px] text-muted">
        <span :class="isStaleStatus ? 'text-amber-500 font-medium' : ''">
          {{ isStaleStatus ? '⚠️ ' + sampledAgoText : sampledAgoText }}
        </span>
        <span class="text-[9px] text-muted">
          Subnet: {{ hotspot?.subnet || '192.168.137.0/24' }}
        </span>
      </div>

    </div>
  </AccordionCard>
</template>

<script setup>
import { computed } from 'vue'
import AccordionCard from './AccordionCard.vue'
import { formatTimeAgo, isStale } from '../utils/time.js'

const props = defineProps({
  hotspot: Object,
  now: { type: Number, default: () => Date.now() },
})

const isHotspotActive = computed(() => {
  return Boolean(props.hotspot && props.hotspot.active)
})

const clientCount = computed(() => {
  return props.hotspot?.client_count || 0
})

const clientList = computed(() => {
  return props.hotspot?.clients || []
})

const isStaleStatus = computed(() => {
  if (!props.hotspot?.sampled_at) return true
  return isStale(props.hotspot.sampled_at, 6000, props.now)
})

const sampledAgoText = computed(() => {
  if (!props.hotspot?.sampled_at) return 'No sample'
  const ago = formatTimeAgo(props.hotspot.sampled_at, props.now)
  return isStaleStatus.value ? `Paused (${ago})` : `Sampled ${ago}`
})

function formatIfaceName(name) {
  if (!name) return 'Hotspot Adapter'
  return name.replace('Local Area Connection* ', 'Hotspot *')
}

function formatSpeed(mbps) {
  if (mbps === undefined || mbps === null) return '0.00 Mbps'
  if (mbps >= 1000) return (mbps / 1000).toFixed(2) + ' Gbps'
  if (mbps >= 10) return mbps.toFixed(1) + ' Mbps'
  if (mbps >= 1) return mbps.toFixed(2) + ' Mbps'
  if (mbps > 0) return (mbps * 1000).toFixed(0) + ' Kbps'
  return '0.00 Mbps'
}

function formatBytes(bytes) {
  if (!bytes || bytes <= 0) return '0 B'
  if (bytes >= 1073741824) return (bytes / 1073741824).toFixed(1) + ' GB'
  if (bytes >= 1048576) return (bytes / 1048576).toFixed(1) + ' MB'
  if (bytes >= 1024) return (bytes / 1024).toFixed(0) + ' KB'
  return bytes.toFixed(0) + ' B'
}
</script>
