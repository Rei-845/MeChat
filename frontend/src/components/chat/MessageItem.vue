<template>
  <div class="flex gap-3 group" :class="isSelf ? 'flex-row-reverse' : 'flex-row'">
    <!-- Avatar: both self and other, click to view profile -->
    <Avatar :name="msg.sender_nickname" :url="msg.sender_avatar" :size="32"
            class="mt-1 cursor-pointer hover:opacity-90 transition-opacity"
            @click="msg.sender_id && openUserProfile(msg.sender_id)" />

    <div class="max-w-[65%] flex flex-col" :class="isSelf ? 'items-end' : 'items-start'">
      <!-- Sender name (group chat) -->
      <span v-if="!isSelf && msg.sender_nickname" class="flex items-center gap-1 text-xs text-ink/40 mb-1 px-1">
        {{ msg.sender_nickname }}
        <VipBadge v-if="msg.sender_vip" dense icon-only />
      </span>

      <!-- Recalled -->
      <div v-if="msg.is_recalled"
           class="text-xs text-ink/30 italic px-3 py-2 rounded-xl"
           style="background:rgb(var(--ink) / 0.04);border:1px dashed rgb(var(--ink) / 0.08)">
        消息已撤回
      </div>

      <!-- Image (+ optional caption text) -->
      <div v-else-if="msg.msg_type === 2" class="flex flex-col" :class="isSelf ? 'items-end' : 'items-start'">
        <div class="rounded-xl overflow-hidden cursor-zoom-in hover:opacity-90 transition-opacity"
             @click="viewImg = msg.content.url">
          <img :src="msg.content.url" alt="图片" class="max-w-48 max-h-48 object-cover" loading="lazy" />
        </div>
        <div v-if="msg.content?.text"
             class="mt-1 px-4 py-2.5 text-sm leading-relaxed break-words whitespace-pre-wrap"
             :class="isSelf ? 'msg-bubble-self' : 'msg-bubble-other'">
          {{ msg.content.text }}
        </div>
      </div>

      <!-- Text -->
      <div v-else
           class="px-4 py-2.5 text-sm leading-relaxed break-words whitespace-pre-wrap transition-opacity"
           :class="[isSelf ? 'msg-bubble-self' : 'msg-bubble-other',
                    msg._failed && 'opacity-50']">
        <span v-if="msg.content?.ai_generated" class="inline-flex items-center gap-1 text-xs opacity-60 mb-1 block">
          <Sparkles :size="10" /> AI 生成
        </span>
        {{ msg.content?.text }}
      </div>

      <!-- Meta -->
      <div class="flex items-center gap-1.5 mt-1 px-1"
           :class="isSelf ? 'flex-row-reverse' : 'flex-row'">
        <span class="text-[10px] text-ink/25">
          {{ formatTime(msg.created_at) }}
        </span>
        <span v-if="msg._pending && !msg._failed" class="text-[10px] text-ink/20">发送中…</span>
        <button v-if="msg._failed" @click="emit('resend', msg)"
                class="flex items-center gap-1 text-[10px] text-red-400 hover:text-red-300 transition-colors"
                title="发送失败，点击重试">
          <AlertCircle :size="12" />
          <span>发送失败，点击重试</span>
        </button>
        <!-- 撤回（仅自己 2 分钟内、未撤回、已发送成功的消息）。两步确认防误触 -->
        <template v-if="canRecall">
          <button v-if="!confirmingRecall" @click="startRecall"
                  class="flex items-center gap-0.5 text-[10px] text-ink/25 hover:text-ink/60 transition-colors"
                  title="撤回消息">
            <Undo2 :size="11" />
            <span>撤回</span>
          </button>
          <button v-else @click="emit('recall', msg)"
                  class="flex items-center gap-0.5 text-[10px] text-red-400 hover:text-red-300 transition-colors"
                  title="点击确认撤回">
            <span>确认撤回？</span>
          </button>
        </template>
      </div>
    </div>
  </div>

  <ImageViewer :src="viewImg" @close="viewImg = ''" />
</template>

<script setup>
import { ref, computed } from 'vue'
import { Sparkles, AlertCircle, Undo2 } from 'lucide-vue-next'
import { formatMsgTime as formatTime } from '@/utils/time'
import ImageViewer from '@/components/ui/ImageViewer.vue'
import VipBadge    from '@/components/ui/VipBadge.vue'
import Avatar      from '@/components/ui/Avatar.vue'
import { useUserProfile } from '@/composables/useUserProfile'

const { openUserProfile } = useUserProfile()

const props = defineProps({
  msg:    { type: Object, required: true },
  isSelf: { type: Boolean, default: false },
})
const emit = defineEmits(['resend', 'recall'])

const viewImg = ref('')
const confirmingRecall = ref(false)

// 撤回条件：自己发的、未撤回、已发送成功（非 pending/failed）、且在 2 分钟内。
// 时间窗口与后端 RecallMessage 的 2 分钟限制一致；超时后按钮自然隐藏，避免无谓的失败请求。
const RECALL_WINDOW_MS = 2 * 60 * 1000
const canRecall = computed(() => {
  if (!props.isSelf || props.msg.is_recalled || props.msg._pending || props.msg._failed) return false
  if (!props.msg.msg_id || String(props.msg.msg_id).startsWith('temp-')) return false
  const created = props.msg.created_at ? new Date(props.msg.created_at).getTime() : 0
  return created > 0 && Date.now() - created < RECALL_WINDOW_MS
})

function startRecall() {
  // 第一步：进入确认态，3 秒内未确认则自动复位，避免误触后一直停在“确认”
  confirmingRecall.value = true
  setTimeout(() => { confirmingRecall.value = false }, 3000)
}
</script>
