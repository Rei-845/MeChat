<template>
  <Teleport to="body">
    <Transition name="addfriend">
      <div v-if="addFriendTarget" class="fixed inset-0 z-[160] flex items-center justify-center p-4"
           style="background:rgba(0,0,0,0.6);backdrop-filter:blur(6px)"
           @click.self="closeAddFriend">
        <div class="w-full max-w-sm glass-strong rounded-2xl p-5 animate-scale-in">
          <!-- Header -->
          <div class="flex items-center justify-between mb-3">
            <h3 class="text-base font-bold text-ink">添加好友</h3>
            <button @click="closeAddFriend" class="text-ink/40 hover:text-ink transition-colors">
              <X :size="18" />
            </button>
          </div>

          <p class="text-sm text-ink/50 mb-3">
            向 <span class="text-ink/80 font-medium">{{ addFriendTarget.nickname || '对方' }}</span>
            发送好友申请，附上一句招呼语：
          </p>

          <!-- Greeting input + AI draft -->
          <div class="relative">
            <textarea v-model="message" rows="3" maxlength="100"
                      placeholder="你好，我想加你为好友～"
                      class="mc-input resize-none py-3 pr-10 leading-relaxed w-full"
                      @keydown.enter.exact.prevent="send" />
            <button v-if="isVIP" @click="aiDraft" :disabled="loadingAI" title="AI 帮写招呼语"
                    class="absolute right-2 bottom-2 w-7 h-7 rounded-lg flex items-center justify-center
                           text-ink/30 hover:text-primary-light hover:bg-primary/10 transition-all"
                    :class="loadingAI && 'text-primary-light'">
              <Loader2 v-if="loadingAI" :size="14" class="animate-spin" />
              <Sparkles v-else :size="14" />
            </button>
          </div>
          <div class="flex justify-end mt-1">
            <span class="text-[11px] text-ink/30">{{ message.length }}/100</span>
          </div>

          <!-- Actions -->
          <div class="flex gap-2 mt-4">
            <button @click="closeAddFriend"
                    class="flex-1 py-2 rounded-xl text-sm text-ink/60 hover:bg-ink/5 transition-all"
                    style="border:1px solid rgb(var(--ink) / 0.08)">
              取消
            </button>
            <button @click="send" :disabled="sending"
                    class="flex-1 py-2 rounded-xl text-sm font-medium text-ink transition-all"
                    style="background:linear-gradient(135deg,#3390EC,#2980DE);color:#fff"
                    :class="sending && 'opacity-60 cursor-wait'">
              <Loader2 v-if="sending" :size="14" class="inline animate-spin" />
              <span v-else>发送申请</span>
            </button>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup>
import { ref, computed, watch } from 'vue'
import { X, Sparkles, Loader2 } from 'lucide-vue-next'
import { useAddFriend } from '@/composables/useAddFriend'
import { useFriendStore } from '@/stores/friend'
import { useToast } from '@/composables/useToast'
import { streamDraftMessage } from '@/api/ai'
import { useAuthStore } from '@/stores/auth'

const { addFriendTarget, closeAddFriend, fireAddFriendSuccess } = useAddFriend()
const friendStore = useFriendStore()
const toast = useToast()
const auth = useAuthStore()
const isVIP = computed(() => auth.user?.vip_level > 0)   // AI 帮写招呼语仅 VIP

const message   = ref('')
const sending   = ref(false)
const loadingAI = ref(false)

// 每次打开弹层时重置输入与状态
watch(addFriendTarget, (t) => {
  if (t) { message.value = ''; sending.value = false; loadingAI.value = false }
})

// 复用通用「帮写」AI 流式接口，给它一个生成招呼语的指令；结果可直接编辑
async function aiDraft() {
  if (loadingAI.value) return
  loadingAI.value = true
  message.value = ''
  const nick = addFriendTarget.value?.nickname || '对方'
  try {
    await streamDraftMessage(`给「${nick}」写一句简短、友好、自然的加好友招呼语，30 字以内，不要解释`, '', {
      onEvent(ev) { if (ev.type === 'token') message.value += ev.text },
    })
    message.value = message.value.slice(0, 100)
  } catch {
    if (!message.value) toast.error('AI 生成失败，请手动填写')
  } finally {
    loadingAI.value = false
  }
}

async function send() {
  if (sending.value) return
  const t = addFriendTarget.value
  if (!t) return
  sending.value = true
  try {
    await friendStore.sendRequest(t.id, message.value.trim())
    toast.success(`已向 ${t.nickname || '对方'} 发送好友申请`)
    fireAddFriendSuccess()
    closeAddFriend()
  } catch (e) {
    toast.error(typeof e === 'string' ? e : '发送失败')
  } finally {
    sending.value = false
  }
}
</script>

<style scoped>
.addfriend-enter-active, .addfriend-leave-active { transition: opacity 0.2s ease; }
.addfriend-enter-from, .addfriend-leave-to { opacity: 0; }
</style>
