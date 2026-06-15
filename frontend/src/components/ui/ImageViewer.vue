<template>
  <Teleport to="body">
    <div v-if="src" class="fixed inset-0 z-[200] flex items-center justify-center"
         style="background:rgba(0,0,0,0.85);backdrop-filter:blur(8px)"
         @click="close" @keydown.esc="close" tabindex="-1">
      <img :src="src" class="max-w-[90vw] max-h-[90vh] object-contain rounded-xl shadow-2xl"
           style="animation:scale-in 0.15s ease"
           @click.stop />
      <button @click="close"
              class="absolute top-4 right-4 w-10 h-10 rounded-full flex items-center justify-center
                     text-white/70 hover:text-white transition-colors"
              style="background:rgba(255,255,255,0.12)">
        <X :size="20" />
      </button>
    </div>
  </Teleport>
</template>

<script setup>
import { X } from 'lucide-vue-next'
const props = defineProps({ src: { type: String, default: '' } })
const emit  = defineEmits(['close'])
function close() { emit('close') }
</script>
