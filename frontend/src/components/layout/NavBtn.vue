<template>
  <RouterLink :to="to" custom v-slot="{ isActive, navigate }">
    <button
      @click="navigate"
      class="relative w-12 h-12 rounded-xl flex items-center justify-center transition-all group"
      :style="isActive
        ? 'background:rgba(51,144,236,0.18);box-shadow:inset 0 0 0 1px rgba(51,144,236,0.35)'
        : ''"
      :title="label"
    >
      <component :is="icon" :size="20"
        :class="isActive ? 'text-primary-light' : 'text-ink/40 group-hover:text-ink/70'"
        class="transition-colors" />

      <!-- Tooltip -->
      <span class="absolute left-14 px-2 py-1 rounded text-xs font-medium whitespace-nowrap text-ink
                   opacity-0 group-hover:opacity-100 pointer-events-none transition-opacity shadow-glass"
            style="background:rgb(var(--surface));border:1px solid rgb(var(--border))">
        {{ label }}
      </span>

      <!-- Badge -->
      <span v-if="badge && badge > 0"
            class="absolute -top-1 -right-1 min-w-[16px] h-4 px-1 rounded-full
                   flex items-center justify-center text-[10px] font-bold text-white"
            style="background:#EF4444">
        {{ badge > 99 ? '99+' : badge }}
      </span>
    </button>
  </RouterLink>
</template>

<script setup>
import { RouterLink } from 'vue-router'
defineProps({ to: String, icon: Object, label: String, badge: Number })
</script>
