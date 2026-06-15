<template>
  <div class="flex items-center gap-2 px-2.5 py-1.5 rounded-lg text-[12px] w-fit max-w-full"
       style="background:rgba(51,144,236,0.08);border:1px solid rgba(51,144,236,0.18)">
    <Loader2 v-if="tool.status === 'running'" :size="13" class="shrink-0 animate-spin text-primary-light" />
    <Check   v-else :size="13" class="shrink-0 text-emerald-400" />
    <Wrench :size="12" class="shrink-0 text-ink/35" />
    <span class="text-ink/70 font-medium shrink-0">{{ tool.label }}</span>
    <span v-if="tool.summary" class="text-ink/35 truncate">· {{ tool.summary }}</span>
  </div>
</template>

<script setup>
import { Loader2, Check, Wrench } from 'lucide-vue-next'

defineProps({
  // { name, label, status: 'running' | 'done', summary }
  tool: { type: Object, required: true },
})
</script>
