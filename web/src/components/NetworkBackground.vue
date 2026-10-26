<template>
  <div class="fixed inset-0 -z-10 overflow-hidden pointer-events-none select-none bg-[var(--bg)]">
    <!-- Animated World Network Map Image Layer -->
    <div
      class="absolute w-[114%] h-[114%] -left-[7%] -top-[7%] bg-cover bg-center will-change-transform network-bg-animated"
      :style="{ backgroundImage: `url(${bgImage})` }"
      :class="isDark ? 'opacity-40' : 'opacity-15'"
    />

    <!-- Deep Cyber Vignette Gradient Overlay (ensures clean, simple, readable UI) -->
    <div
      class="absolute inset-0 transition-colors duration-500"
      :class="isDark ? 'bg-gradient-to-b from-[#070b14]/75 via-[#070b14]/60 to-[#070b14]/90' : 'bg-gradient-to-b from-[#f4f6fa]/85 via-[#f4f6fa]/70 to-[#f4f6fa]/92'"
    />

    <!-- Radial Focus Vignette (glow in center, dark around edges) -->
    <div
      class="absolute inset-0"
      :style="{
        background: isDark
          ? 'radial-gradient(ellipse at 50% 35%, rgba(6, 182, 212, 0.08) 0%, rgba(7, 11, 20, 0.4) 60%, rgba(7, 11, 20, 0.85) 100%)'
          : 'radial-gradient(ellipse at 50% 35%, rgba(2, 132, 199, 0.04) 0%, rgba(244, 246, 250, 0.3) 60%, rgba(244, 246, 250, 0.8) 100%)'
      }"
    />

    <!-- Subtle Pulsing Energy Beacons -->
    <div class="absolute top-[28%] left-[24%] w-32 h-32 rounded-full bg-cyan-400/10 blur-2xl animate-pulse" />
    <div class="absolute top-[32%] right-[28%] w-36 h-36 rounded-full bg-blue-500/10 blur-2xl animate-pulse" style="animation-delay: 2s;" />
    <div class="absolute bottom-[25%] left-[48%] w-40 h-40 rounded-full bg-purple-500/10 blur-3xl animate-pulse" style="animation-delay: 4s;" />
  </div>
</template>

<script setup>
import bgImage from '../assets/network-bg.jpg'

defineProps({
  isDark: { type: Boolean, default: true }
})
</script>

<style scoped>
/* Smooth, continuous, subtle cinematic floating movement */
.network-bg-animated {
  animation: networkFloat 40s ease-in-out infinite alternate;
}

@keyframes networkFloat {
  0% {
    transform: scale(1.04) translate3d(0, 0, 0);
  }
  33% {
    transform: scale(1.10) translate3d(-1.5%, 1%, 0);
  }
  66% {
    transform: scale(1.08) translate3d(1.5%, -1%, 0);
  }
  100% {
    transform: scale(1.14) translate3d(0.5%, 1.5%, 0);
  }
}
</style>
