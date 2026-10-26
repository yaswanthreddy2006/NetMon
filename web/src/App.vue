<template>
  <NetworkBackground :is-dark="isDark" />
  <Dashboard :is-dark="isDark" @toggle-theme="toggleTheme" />
  <ToastContainer :toasts="toasts" @dismiss="dismissToast" />
</template>

<script setup>
import { ref } from 'vue'
import Dashboard         from './components/Dashboard.vue'
import NetworkBackground from './components/NetworkBackground.vue'
import ToastContainer    from './components/ToastContainer.vue'
import { useToasts }     from './composables/useToasts.js'

const { toasts, dismiss: dismissToast } = useToasts()

// Default to dark mode for cybersecurity theme unless user explicitly saved 'light'
const isDark = ref(localStorage.getItem('theme') !== 'light')

function applyTheme(dark) {
  document.body.classList.toggle('dark-mode', dark)
}
applyTheme(isDark.value)

function toggleTheme() {
  isDark.value = !isDark.value
  applyTheme(isDark.value)
  localStorage.setItem('theme', isDark.value ? 'dark' : 'light')
}
</script>
