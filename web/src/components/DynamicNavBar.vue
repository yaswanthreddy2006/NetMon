<template>
  <nav class="sticky top-2 z-50 w-full mb-4 px-2 sm:px-0">
    <div class="nav-island flex items-center justify-between gap-2 p-1.5 sm:p-2 rounded-2xl bg-[var(--card)]/80 backdrop-blur-xl border border-[var(--border)] shadow-[0_8px_32px_rgba(0,0,0,0.25)] transition-all">
      
      <!-- Brand & Status Pill -->
      <div class="flex items-center gap-2 pl-2 flex-shrink-0">
        <div class="w-8 h-8 rounded-xl bg-gradient-to-tr from-cyan-500/20 to-blue-600/30 border border-cyan-500/30 flex items-center justify-center text-cyan-400 shadow-[0_0_12px_rgba(6,182,212,0.25)]">
          <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
            <path d="M5 12.55a11 11 0 0 1 14.08 0"/>
            <path d="M1.42 9a16 16 0 0 1 21.16 0"/>
            <path d="M8.53 16.11a6 6 0 0 1 6.95 0"/>
            <circle cx="12" cy="20" r="1" fill="currentColor" stroke="none"/>
          </svg>
        </div>
        <div class="hidden xl:flex flex-col leading-none">
          <span class="text-xs font-black tracking-tight text-[var(--fg)]">
            NET<span class="text-cyan-400">MON</span>
          </span>
          <span class="text-[8px] font-mono text-[var(--muted)] uppercase tracking-wider">
            NOC DASHBOARD
          </span>
        </div>
        <!-- Online/Offline Ping Dot -->
        <span class="status-badge text-[10px] py-0.5 px-2 flex-shrink-0" :class="status">
          <span class="status-dot" />
          {{ statusLabel || (status === 'online' ? 'Online' : 'Offline') }}
        </span>
      </div>

      <!-- Center Floating Navigation Items (Reference Image Inspiration) -->
      <div class="flex items-center gap-1 sm:gap-1.5 overflow-x-auto no-scrollbar py-0.5 px-1 max-w-full">
        <button
          v-for="item in navItems"
          :key="item.id"
          type="button"
          class="nav-pill flex items-center gap-1.5 px-2.5 sm:px-3.5 py-1.5 rounded-xl text-xs font-bold transition-all whitespace-nowrap cursor-pointer select-none"
          :class="activeSection === item.id ? 'active-pill' : 'inactive-pill'"
          @click="scrollTo(item.id)"
        >
          <span class="nav-icon" v-html="item.icon" />
          <span>{{ item.label }}</span>
          <span
            v-if="item.badge"
            class="text-[9px] font-mono px-1.5 py-0.2 rounded-full font-bold ml-0.5"
            :class="item.badgeClass"
          >
            {{ item.badge }}
          </span>
        </button>
      </div>

      <!-- Quick Action Controls Toolbar -->
      <div class="flex items-center gap-1.5 pr-1 flex-shrink-0">
        <!-- Scan / Refresh Button -->
        <button
          type="button"
          class="btn-tool flex items-center gap-1 px-2.5 py-1.5 rounded-xl text-xs font-bold transition-all text-cyan-400 bg-cyan-500/10 hover:bg-cyan-500/20 border border-cyan-500/25"
          :disabled="isRefreshing"
          @click="$emit('refresh-now')"
          title="Scan network and refresh metrics"
        >
          <svg
            xmlns="http://www.w3.org/2000/svg"
            width="13"
            height="13"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2.5"
            stroke-linecap="round"
            stroke-linejoin="round"
            :class="{ 'animate-spin': isRefreshing }"
          >
            <path d="M21 12a9 9 0 1 1-6.219-8.56"/>
          </svg>
          <span class="hidden md:inline">{{ isRefreshing ? 'Scanning…' : 'Scan' }}</span>
        </button>

        <!-- Speed Test Button -->
        <button
          type="button"
          class="btn-tool flex items-center gap-1 px-2.5 py-1.5 rounded-xl text-xs font-bold transition-all text-emerald-400 bg-emerald-500/10 hover:bg-emerald-500/20 border border-emerald-500/25"
          :disabled="isSpeedTesting"
          @click="$emit('run-speed-test')"
          title="Trigger manual bandwidth test"
        >
          <svg
            v-if="!isSpeedTesting"
            xmlns="http://www.w3.org/2000/svg"
            width="13"
            height="13"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2.5"
            stroke-linecap="round"
            stroke-linejoin="round"
          >
            <polygon points="13 2 3 14 12 14 11 22 21 10 12 10 13 2"/>
          </svg>
          <span v-else class="animate-spin text-[8px]">●</span>
          <span class="hidden md:inline">{{ isSpeedTesting ? 'Testing…' : 'Speed' }}</span>
        </button>

        <slot name="duration-picker" />

        <!-- Theme Toggle -->
        <button
          type="button"
          class="w-8 h-8 rounded-xl bg-[var(--card)] hover:bg-[var(--card-hover)] border border-[var(--border)] flex items-center justify-center text-[var(--subtle)] hover:text-[var(--fg)] transition-all"
          :title="isDark ? 'Switch to light mode' : 'Switch to dark mode'"
          @click="$emit('toggle-theme')"
        >
          <svg v-if="!isDark" xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z"/></svg>
          <svg v-else xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="5"/><line x1="12" y1="1" x2="12" y2="3"/><line x1="12" y1="21" x2="12" y2="23"/><line x1="4.22" y1="4.22" x2="5.64" y2="5.64"/><line x1="18.36" y1="18.36" x2="19.78" y2="19.78"/><line x1="1" y1="12" x2="3" y2="12"/><line x1="21" y1="12" x2="23" y2="12"/><line x1="4.22" y1="19.78" x2="5.64" y2="18.36"/><line x1="18.36" y1="5.64" x2="19.78" y2="4.22"/></svg>
        </button>

        <!-- Settings Modal Trigger -->
        <button
          type="button"
          class="w-8 h-8 rounded-xl bg-[var(--card)] hover:bg-[var(--card-hover)] border border-[var(--border)] flex items-center justify-center text-[var(--subtle)] hover:text-[var(--fg)] transition-all"
          title="Settings &amp; Configuration"
          @click="$emit('open-settings')"
        >
          <svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1-2.83 2.83l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-4 0v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83-2.83l.06-.06A1.65 1.65 0 0 0 4.68 15a1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1 0-4h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 2.83-2.83l.06.06A1.65 1.65 0 0 0 9 4.68a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 4 0v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 2.83l-.06.06A1.65 1.65 0 0 0 19.4 9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 0 4h-.09a1.65 1.65 0 0 0-1.51 1z"/></svg>
        </button>
      </div>

    </div>
  </nav>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  activeSection:  { type: String, default: 'overview' },
  status:         { type: String, default: 'online' },
  statusLabel:    { type: String, default: '' },
  targetCount:    { type: Number, default: 0 },
  deviceCount:    { type: Number, default: 1 },
  liveThroughput: { type: String, default: '' },
  alertCount:     { type: Number, default: 0 },
  isRefreshing:   { type: Boolean, default: false },
  isSpeedTesting: { type: Boolean, default: false },
  isDark:         { type: Boolean, default: false },
})

const emit = defineEmits(['toggle-theme', 'open-settings', 'refresh-now', 'run-speed-test', 'select-section'])

const SVG = (d) => `<svg xmlns="http://www.w3.org/2000/svg" width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.25" stroke-linecap="round" stroke-linejoin="round">${d}</svg>`

const navItems = computed(() => [
  {
    id: 'overview',
    label: 'Overview',
    icon: SVG('<rect width="7" height="7" x="3" y="3" rx="1"/><rect width="7" height="7" x="14" y="3" rx="1"/><rect width="7" height="7" x="14" y="14" rx="1"/><rect width="7" height="7" x="3" y="14" rx="1"/>'),
    badge: null,
  },
  {
    id: 'network',
    label: 'Network',
    icon: SVG('<circle cx="12" cy="12" r="10"/><line x1="2" y1="12" x2="22" y2="12"/><path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z"/>'),
    badge: `${props.targetCount} Probes`,
    badgeClass: 'bg-cyan-500/15 text-cyan-400 border border-cyan-500/30'
  },
  {
    id: 'devices',
    label: 'Devices',
    icon: SVG('<rect width="16" height="12" x="4" y="4" rx="2"/><line x1="12" x2="12" y1="16" y2="20"/><line x1="8" x2="16" y1="20" y2="20"/>'),
    badge: `${props.deviceCount}`,
    badgeClass: 'bg-blue-500/15 text-blue-400 border border-blue-500/30'
  },
  {
    id: 'traffic',
    label: 'Traffic',
    icon: SVG('<line x1="12" y1="19" x2="12" y2="5"/><polyline points="5 12 12 5 19 12"/><line x1="19" y1="12" x2="19" y2="19"/><polyline points="15 15 19 19 23 15"/>'),
    badge: props.liveThroughput || null,
    badgeClass: 'bg-emerald-500/15 text-emerald-400 border border-emerald-500/30 font-mono'
  },
  {
    id: 'security',
    label: 'Security',
    icon: SVG('<path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/>'),
    badge: props.alertCount > 0 ? `${props.alertCount} Alert${props.alertCount > 1 ? 's' : ''}` : 'Optimal',
    badgeClass: props.alertCount > 0 ? 'bg-amber-500/20 text-amber-400 border border-amber-500/40' : 'bg-purple-500/15 text-purple-400 border border-purple-500/30'
  },
  {
    id: 'analytics',
    label: 'Analytics',
    icon: SVG('<polyline points="22 12 18 12 15 21 9 3 6 12 2 12"/>'),
    badge: null,
  }
])

function scrollTo(id) {
  emit('select-section', id)
  const el = document.getElementById(`sec-${id}`)
  if (el) {
    const yOffset = -75
    const y = el.getBoundingClientRect().top + window.pageYOffset + yOffset
    window.scrollTo({ top: y, behavior: 'smooth' })
  }
}
</script>

<style scoped>
.nav-island {
  background: var(--card);
  backdrop-filter: blur(16px);
  -webkit-backdrop-filter: blur(16px);
}

.nav-pill {
  transition: all 0.25s cubic-bezier(0.16, 1, 0.3, 1);
}

.inactive-pill {
  color: var(--muted);
  background: transparent;
  border: 1px solid transparent;
}
.inactive-pill:hover {
  color: var(--fg);
  background: var(--bg-secondary);
  border-color: var(--border);
  transform: translateY(-1px);
}

.active-pill {
  color: #38bdf8;
  background: linear-gradient(135deg, rgba(6, 182, 212, 0.15), rgba(59, 130, 246, 0.22));
  border: 1px solid rgba(56, 189, 248, 0.45);
  box-shadow: 0 0 16px rgba(56, 189, 248, 0.22);
  transform: translateY(-1px) scale(1.02);
}

.no-scrollbar::-webkit-scrollbar {
  display: none;
}
.no-scrollbar {
  -ms-overflow-style: none;
  scrollbar-width: none;
}
</style>
