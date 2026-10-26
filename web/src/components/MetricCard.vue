<template>
  <div class="card metric-card flash-trigger" :data-color="color" :class="{ 'cursor-pointer': hasAction }" @click="hasAction && $emit('action')">
    <div class="flex items-center justify-between">
      <div class="metric-label">{{ label }}</div>
      <div class="flex items-center gap-1.5">
        <button
          v-if="actionLabel"
          type="button"
          class="text-[10px] font-semibold px-2 py-0.5 rounded bg-emerald-500/10 hover:bg-emerald-500/20 text-emerald-600 dark:text-emerald-400 border border-emerald-500/20 transition-all flex items-center gap-1"
          :disabled="actionLoading"
          :class="{ 'opacity-60 cursor-not-allowed': actionLoading }"
          @click.stop="$emit('action')"
        >
          <span v-if="actionLoading" class="animate-spin text-[8px]">●</span>
          {{ actionLoading ? 'Testing…' : actionLabel }}
        </button>
        <div class="metric-icon" :class="color" v-html="icon" />
      </div>
    </div>
    <div class="metric-value">{{ value }}</div>
    <div class="metric-sub">{{ sub }}</div>
    <span v-if="spark" v-html="spark" />
  </div>
</template>

<script setup>
defineProps({
  label:         String,
  value:         String,
  sub:           String,
  color:         String,
  icon:          String,
  spark:         String,
  actionLabel:   String,
  actionLoading: Boolean,
  hasAction:     Boolean,
})

defineEmits(['action'])
</script>
