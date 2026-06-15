<template>
  <div class="rounded-full overflow-hidden shrink-0"
       :style="`width:${size}px;height:${size}px`">
    <img v-if="url" :src="url" :alt="name" class="w-full h-full object-cover" />
    <div v-else class="w-full h-full flex items-center justify-center font-bold text-ink"
         :style="`background:linear-gradient(135deg,#3390EC,#2980DE);color:#fff;font-size:${size * 0.4}px`">
      {{ initial }}
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  name: { type: String, default: '' },
  url:  { type: String, default: '' },
  size: { type: [Number, String], default: 40 },
})

const initial = computed(() => (props.name?.[0] || '?').toUpperCase())
</script>
