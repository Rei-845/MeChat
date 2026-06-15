<template>
  <div class="shrink-0 px-3 md:px-6 py-3 md:py-4" style="border-top:1px solid rgb(var(--ink) / 0.06)">
    <!-- AI draft suggestion (streaming) -->
    <div v-if="aiDraft || loadingAI"
         class="mb-2 flex items-start gap-2 p-3 rounded-xl text-xs animate-slide-up"
         style="background:rgba(51,144,236,0.08);border:1px solid rgba(51,144,236,0.2)">
      <Sparkles :size="12" class="text-primary-light mt-0.5 shrink-0" />
      <div class="flex-1 min-w-0">
        <p class="text-ink/60 mb-1.5">AI 建议回复：</p>
        <p class="text-ink/80 whitespace-pre-wrap break-words">{{ aiDraft }}</p>
        <span v-if="loadingAI" class="inline-block w-1 h-3 bg-primary-light/70 animate-pulse ml-0.5" />
        <div v-if="aiDraft && !loadingAI" class="flex gap-2 mt-2">
          <button @click="acceptDraft"
                  class="text-xs px-3 py-1 rounded-lg text-accent font-medium transition-colors hover:bg-accent/10">
            使用
          </button>
          <button @click="clearDraft"
                  class="text-xs px-3 py-1 rounded-lg text-ink/30 transition-colors hover:text-ink/60">
            忽略
          </button>
        </div>
      </div>
    </div>

    <!-- AI keyword input (shown when no text in textarea) -->
    <div v-if="showAIInput" class="mb-2 flex gap-2 animate-slide-up">
      <input v-model="aiKeyword" ref="aiInputRef" placeholder="描述你想说的内容，AI 来写…"
             type="text" class="mc-input flex-1 py-2 text-xs"
             @keydown.enter.prevent="runDraft"
             @keydown.escape="showAIInput = false" />
      <button @click="runDraft" :disabled="!aiKeyword.trim() || loadingAI"
              class="px-3 py-2 rounded-lg text-xs font-medium transition-all whitespace-nowrap"
              style="background:rgba(51,144,236,0.15);color:#3390EC;border:1px solid rgba(51,144,236,0.3)"
              :class="(!aiKeyword.trim() || loadingAI) && 'opacity-50 cursor-not-allowed'">
        <Loader2 v-if="loadingAI" :size="13" class="inline animate-spin" />
        <span v-else>生成</span>
      </button>
      <button @click="showAIInput = false"
              class="w-8 rounded-lg text-ink/30 hover:text-ink/60 transition-all">
        <X :size="14" class="mx-auto" />
      </button>
    </div>

    <!-- Image preview before send -->
    <div v-if="pendingImage" class="mb-2 flex items-center gap-2 p-2 rounded-xl"
         style="background:rgb(var(--ink) / 0.04);border:1px solid rgb(var(--ink) / 0.08)">
      <img :src="pendingImage.preview" class="w-14 h-14 rounded-lg object-cover shrink-0" />
      <div class="flex-1 min-w-0">
        <p class="text-xs text-ink/60 truncate">{{ pendingImage.name }}</p>
        <p v-if="uploadingImage" class="text-xs text-primary-light mt-0.5">上传中…</p>
      </div>
      <button @click="cancelImage" class="text-ink/30 hover:text-ink/60">
        <X :size="14" />
      </button>
    </div>

    <div class="flex items-end gap-2">
      <!-- Image upload button -->
      <label class="w-9 h-9 rounded-xl flex items-center justify-center shrink-0 cursor-pointer
                    text-ink/30 hover:text-ink/60 hover:bg-ink/5 transition-all"
             :class="uploadingImage && 'opacity-40 pointer-events-none'">
        <ImageIcon :size="18" />
        <input type="file" accept="image/*" class="hidden" @change="pickImage" :disabled="uploadingImage" />
      </label>

      <!-- Textarea -->
      <div class="flex-1 relative">
        <textarea
          ref="inputRef"
          v-model="text"
          :placeholder="pendingImage ? '添加文字说明（可选）…' : '发送消息… (Shift+Enter 换行)'"
          rows="1"
          class="mc-input resize-none py-3 pr-10 max-h-32 overflow-y-auto leading-relaxed"
          style="min-height:48px"
          @keydown.enter.exact.prevent="sendMsg"
          @keydown.shift.enter.prevent="insertNewline"
          @input="autoResize"
        />
        <!-- AI draft button inside input -->
        <div v-if="isVIP" class="absolute right-3 bottom-3">
          <button @click="toggleAI"
                  :class="[(showAIInput || loadingAI) ? 'text-primary-light bg-primary/10' : 'text-ink/30 hover:text-primary-light hover:bg-primary/10',
                           'w-7 h-7 rounded-lg flex items-center justify-center transition-all']"
                  title="AI 帮写">
            <Sparkles v-if="!loadingAI" :size="14" />
            <Loader2  v-else :size="14" class="animate-spin" />
          </button>
        </div>
      </div>

      <!-- Send button -->
      <button @click="sendMsg" :disabled="!canSend"
              class="w-11 h-11 rounded-xl flex items-center justify-center shrink-0 transition-all"
              :style="canSend
                ? 'background:linear-gradient(135deg,#3390EC,#2980DE);color:#fff;box-shadow:0 4px 12px rgba(51,144,236,0.3)'
                : 'background:rgb(var(--ink) / 0.05);cursor:not-allowed'">
        <SendHorizonal :size="18" :class="canSend ? 'text-ink' : 'text-ink/20'" />
      </button>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, nextTick } from 'vue'
import { Sparkles, SendHorizonal, Loader2, ImageIcon, X } from 'lucide-vue-next'
import { uploadImage } from '@/api/feed'
import { streamDraftMessage } from '@/api/ai'
import { useAuthStore } from '@/stores/auth'

const emit = defineEmits(['send'])

const auth           = useAuthStore()
const isVIP          = computed(() => auth.user?.vip_level > 0)   // AI 帮写仅 VIP
const text           = ref('')
const aiDraft        = ref('')
const aiKeyword      = ref('')
const loadingAI      = ref(false)
const showAIInput    = ref(false)
const inputRef       = ref(null)
const aiInputRef     = ref(null)
const pendingImage   = ref(null)
const uploadingImage = ref(false)

const canSend = computed(() => text.value.trim() || pendingImage.value?.url)

async function pickImage(e) {
  const file = e.target.files?.[0]
  e.target.value = ''
  if (!file) return
  pendingImage.value = { file, preview: URL.createObjectURL(file), name: file.name, url: null }
  uploadingImage.value = true
  try {
    const res = await uploadImage(file)
    pendingImage.value.url = res.data.url
  } catch {
    pendingImage.value = null
  } finally {
    uploadingImage.value = false
  }
}

function cancelImage() {
  if (pendingImage.value?.preview) URL.revokeObjectURL(pendingImage.value.preview)
  pendingImage.value = null
}

function sendMsg() {
  if (!canSend.value || uploadingImage.value) return
  if (pendingImage.value?.url) {
    emit('send', { _type: 2, url: pendingImage.value.url, text: text.value.trim() })
    cancelImage()
  } else {
    emit('send', { text: text.value.trim() })
  }
  text.value = ''
  aiDraft.value = ''
  nextTick(() => autoResize())
}

// 切换 AI 帮写：有输入时直接拿当前内容生成，否则展开关键词框
function toggleAI() {
  if (loadingAI.value) return
  if (text.value.trim().length > 3) {
    // 有文本：直接以当前内容为描述生成
    aiKeyword.value = text.value.trim()
    runDraft()
  } else {
    showAIInput.value = !showAIInput.value
    if (showAIInput.value) {
      aiDraft.value = ''
      nextTick(() => aiInputRef.value?.focus())
    }
  }
}

async function runDraft() {
  const kw = (aiKeyword.value || text.value).trim()
  if (!kw || loadingAI.value) return
  loadingAI.value = true
  aiDraft.value = ''
  showAIInput.value = false
  try {
    await streamDraftMessage(kw, '', {
      onEvent(ev) {
        if (ev.type === 'token') aiDraft.value += ev.text
      },
    })
  } catch {
    aiDraft.value = aiDraft.value || '（生成失败，请重试）'
  } finally {
    loadingAI.value = false
    aiKeyword.value = ''
  }
}

function acceptDraft() {
  text.value = aiDraft.value
  aiDraft.value = ''
  nextTick(() => { autoResize(); inputRef.value?.focus() })
}

function clearDraft() {
  aiDraft.value = ''
  aiKeyword.value = ''
}

function autoResize() {
  const el = inputRef.value
  if (!el) return
  el.style.height = 'auto'
  el.style.height = Math.min(el.scrollHeight, 128) + 'px'
}

function insertNewline() {
  const el = inputRef.value
  if (!el) return
  const start = el.selectionStart
  const end   = el.selectionEnd
  el.value = el.value.slice(0, start) + '\n' + el.value.slice(end)
  el.selectionStart = el.selectionEnd = start + 1
  el.dispatchEvent(new Event('input'))  // 同步 v-model + autoResize
}

defineExpose({ setAIDraft: (v) => { aiDraft.value = v } })
</script>
