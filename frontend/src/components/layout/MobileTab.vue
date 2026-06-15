<template>
  <RouterLink :to="to" custom v-slot="{ isActive, navigate }">
    <button @click="navigate" :title="label"
            class="relative flex items-center justify-center px-4 py-2.5 transition-all"
            :class="isActive ? 'opacity-100' : 'opacity-50'">
      <component :is="icon" :size="24"
                 :class="isActive ? 'text-primary-light' : 'text-ink/70'"
                 class="transition-colors" />
      <!-- Badge -->
      <span v-if="badge && badge > 0"
            class="absolute top-0.5 right-2 min-w-[16px] h-4 px-1 rounded-full
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
