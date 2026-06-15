<template>
  <div class="rounded-xl overflow-hidden w-full max-w-md"
       style="background:rgb(var(--ink) / 0.04);border:1px solid rgba(51,144,236,0.25)">
    <!-- Header -->
    <div class="flex items-center gap-2 px-3.5 py-2.5"
         style="background:rgba(51,144,236,0.1);border-bottom:1px solid rgba(51,144,236,0.18)">
      <component :is="actionIcon" :size="15" class="text-primary-light shrink-0" />
      <span class="text-[13px] font-semibold text-ink/90">{{ action.label }}</span>
      <span class="ml-auto text-[11px] px-2 py-0.5 rounded-md shrink-0"
            :style="statusStyle">{{ statusText }}</span>
    </div>

    <!-- Preview -->
    <div v-if="action.preview" class="px-3.5 py-3">
      <p class="text-[13px] text-ink/75 leading-relaxed whitespace-pre-wrap break-words">{{ action.preview }}</p>
    </div>

    <!-- Result (after confirm) -->
    <div v-if="action.status === 'confirmed' && action.result"
         class="px-3.5 pb-3 text-[12px] text-emerald-400/90 flex items-center gap-1.5">
      <Check :size="13" /> {{ action.result }}
    </div>

    <!-- Actions -->
    <div v-if="action.status === 'pending'" class="flex gap-2 px-3.5 pb-3 pt-1">
      <button @click="$emit('confirm', action)" :disabled="loading"
              class="flex-1 py-2 rounded-lg text-[13px] font-semibold text-ink transition-all flex items-center justify-center gap-1.5"
              style="background:linear-gradient(135deg,#3390EC,#2980DE);color:#fff"
              :class="loading && 'opacity-60 cursor-not-allowed'">
        <Loader2 v-if="loading" :size="14" class="animate-spin" />
        <Check v-else :size="14" /> 确认执行
      </button>
      <button @click="$emit('cancel', action)" :disabled="loading"
              class="px-4 py-2 rounded-lg text-[13px] font-medium text-ink/60 transition-all hover:bg-ink/5"
              style="border:1px solid rgb(var(--ink) / 0.12)">
        取消
      </button>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { Check, Loader2, Send, FileText, UserPlus, Wrench } from 'lucide-vue-next'

const props = defineProps({
  // { tool, label, preview, args, status: 'pending'|'confirmed'|'cancelled', result }
  action:  { type: Object, required: true },
  loading: { type: Boolean, default: false },
})
defineEmits(['confirm', 'cancel'])

const ICONS = { send_message: Send, create_post: FileText, send_friend_request: UserPlus }
const actionIcon = computed(() => ICONS[props.action.tool] || Wrench)

const statusText = computed(() => ({
  pending:   '待确认',
  confirmed: '已执行',
  cancelled: '已取消',
}[props.action.status] || ''))

const statusStyle = computed(() => {
  switch (props.action.status) {
    case 'confirmed': return 'background:rgba(16,185,129,0.15);color:#34d399'
    case 'cancelled': return 'background:rgb(var(--ink) / 0.08);color:rgb(var(--ink) / 0.4)'
    default:          return 'background:rgba(245,158,11,0.15);color:#fbbf24'
  }
})
</script>
