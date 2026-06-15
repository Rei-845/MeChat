<template>
  <Teleport to="body">
    <div class="fixed inset-0 z-50 flex items-center justify-center p-4"
         style="background:rgba(0,0,0,0.6);backdrop-filter:blur(8px)"
         @click.self="$emit('close')">
      <div class="w-full max-w-lg glass-strong rounded-2xl animate-scale-in overflow-hidden">
        <!-- Header -->
        <div class="flex items-center justify-between px-6 py-4"
             style="border-bottom:1px solid rgb(var(--ink) / 0.06)">
          <h3 class="font-semibold text-ink">发布动态</h3>
          <button @click="$emit('close')" class="text-ink/40 hover:text-ink/70 transition-colors">
            <X :size="18" />
          </button>
        </div>

        <!-- Body -->
        <div class="px-6 py-5">
          <!-- Title -->
          <input
            v-model="title"
            placeholder="给帖子起个标题"
            maxlength="60"
            class="mc-input text-sm font-semibold mb-3" />

          <!-- Content -->
          <textarea
            v-model="content"
            placeholder="正文支持 Markdown 格式（不提供预览）"
            rows="4"
            class="mc-input resize-none text-sm leading-relaxed mb-4"
          />

          <!-- Image preview -->
          <div v-if="imageUrls.length" class="grid grid-cols-3 gap-2 mb-4">
            <div v-for="(url, i) in imageUrls" :key="i" class="relative rounded-xl overflow-hidden aspect-square">
              <img :src="url" class="w-full h-full object-cover" />
              <button @click="removeImage(i)"
                      class="absolute top-1 right-1 w-5 h-5 rounded-full bg-black/60 flex items-center justify-center">
                <X :size="10" class="text-ink" />
              </button>
            </div>
          </div>

          <!-- AI Draft (streaming) -->
          <div v-if="aiDraft || aiLoading"
               class="mb-4 p-3 rounded-xl text-sm animate-slide-up"
               style="background:rgba(51,144,236,0.08);border:1px solid rgba(51,144,236,0.2)">
            <div class="flex items-center gap-2 mb-2">
              <Sparkles :size="12" class="text-primary-light" :class="aiLoading && 'animate-pulse'" />
              <span class="text-xs font-semibold text-primary-light">{{ aiLoading ? 'AI 正在生成…' : 'AI 生成内容' }}</span>
              <button v-if="!aiLoading" @click="aiDraft = null" class="ml-auto text-ink/30 hover:text-ink/60 text-xs">忽略</button>
            </div>
            <p v-if="aiDraft?.title" class="text-ink/90 text-xs font-semibold mb-1">{{ aiDraft.title }}</p>
            <p class="text-ink/70 text-xs leading-relaxed whitespace-pre-wrap">{{ aiDraft?.content }}<span v-if="aiLoading" class="inline-block w-1 h-3 bg-primary-light/70 animate-pulse ml-0.5" /></p>
            <button v-if="!aiLoading && aiDraft?.content" @click="useAIDraft"
                    class="mt-2 text-xs text-accent font-medium hover:underline">使用此内容</button>
          </div>

          <!-- AI keyword input (shown only when no draft yet) -->
          <div v-if="showAI" class="flex gap-2 mb-4 animate-slide-up">
            <input v-model="aiKeywords" placeholder="输入主题关键词，例如：周末爬山…" type="text"
                   class="mc-input flex-1 py-2 text-sm" @keydown.enter="generateAI" />
            <button @click="generateAI" :disabled="aiLoading || !aiKeywords.trim()"
                    class="px-4 py-2 rounded-lg text-sm font-medium transition-all whitespace-nowrap"
                    style="background:rgba(51,144,236,0.15);color:#3390EC;border:1px solid rgba(51,144,236,0.3)">
              <Loader2 v-if="aiLoading" :size="14" class="inline animate-spin mr-1" />
              生成
            </button>
          </div>
        </div>

        <!-- Footer -->
        <div class="flex items-center justify-between px-6 py-4"
             style="border-top:1px solid rgb(var(--ink) / 0.06)">
          <div class="flex items-center gap-2">
            <!-- Image upload -->
            <label class="cursor-pointer w-8 h-8 rounded-lg flex items-center justify-center
                          text-ink/40 hover:text-ink/70 hover:bg-ink/5 transition-all"
                   :class="uploading && 'pointer-events-none opacity-50'">
              <Loader2 v-if="uploading" :size="16" class="animate-spin" />
              <ImageIcon v-else :size="16" />
              <input type="file" accept="image/*" multiple class="hidden" @change="uploadImages" :disabled="uploading" />
            </label>
            <!-- AI write button 仅 VIP -->
            <button v-if="isVIP" @click="handleAI" :disabled="aiLoading"
                    class="flex items-center gap-1.5 px-2.5 py-1.5 rounded-lg text-xs font-medium transition-all"
                    :class="(showAI || aiLoading)
                      ? 'text-primary-light bg-primary/10 border border-primary/30'
                      : 'text-ink/40 hover:text-ink/70 hover:bg-ink/5 border border-transparent'">
              <Loader2 v-if="aiLoading" :size="13" class="animate-spin" />
              <Sparkles v-else :size="13" />
              AI 帮写
            </button>
          </div>

          <div class="flex items-center gap-3">
            <span class="text-xs" :class="title.length > 55 ? 'text-danger' : 'text-ink/30'">
              {{ title.length }}/60
            </span>
            <button @click="submit" :disabled="!title.trim() || submitting"
                    class="btn-primary px-5 py-2 text-sm">
              <Loader2 v-if="submitting" :size="14" class="inline animate-spin mr-1" />
              发布
            </button>
          </div>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<script setup>
import { ref, computed } from 'vue'
import { X, ImageIcon, Sparkles, Loader2 } from 'lucide-vue-next'
import * as feedApi from '@/api/feed'
import { streamDraftPost } from '@/api/ai'
import { useToast }    from '@/composables/useToast'
import { useAuthStore } from '@/stores/auth'
import { useXPNotify } from '@/composables/useXPNotify'

const emit     = defineEmits(['close', 'created'])
const toast    = useToast()
const auth     = useAuthStore()
const isVIP    = computed(() => auth.user?.vip_level > 0)   // AI 帮写帖子仅 VIP
const { showXP } = useXPNotify()

const title      = ref('')
const content    = ref('')
const imageUrls  = ref([])
const uploading  = ref(false)
const showAI     = ref(false)
const aiKeywords = ref('')
const aiDraft    = ref(null)   // { title, content }
const aiLoading  = ref(false)
const submitting = ref(false)

async function uploadImages(e) {
  const files = Array.from(e.target.files || [])
  if (!files.length) return
  uploading.value = true
  e.target.value = ''
  let failed = 0
  for (const file of files.slice(0, 9 - imageUrls.value.length)) {
    try {
      const res = await feedApi.uploadImage(file)
      imageUrls.value.push(res.data.url)
    } catch {
      failed++
    }
  }
  uploading.value = false
  if (failed > 0) toast.error(`${failed} 张图片上传失败，请重试`)
}

function removeImage(i) { imageUrls.value.splice(i, 1) }

// 解析 AI 流式累积的"标题：xxx\n正文：xxx"格式
function parseAIDraft(text) {
  const t = (text || '').trim()
  const titleMatch = t.match(/标题[:：]\s*(.+)/)
  const bodyMatch  = t.match(/正文[:：]\s*([\s\S]+)/)
  if (titleMatch || bodyMatch) {
    return {
      title:   titleMatch ? titleMatch[1].trim() : '',
      content: bodyMatch ? bodyMatch[1].trim() : t,
    }
  }
  return { title: '', content: t }
}

// 流式调用 AI 帮写帖子
async function callDraftStream(topic) {
  let raw = ''
  aiDraft.value = { title: '', content: '' } // 先展示空框占位
  await streamDraftPost(topic, {
    onEvent(ev) {
      if (ev.type === 'token') {
        raw += ev.text
        aiDraft.value = parseAIDraft(raw) // 实时解析并更新预览
      }
    },
  })
  if (!raw.trim()) throw new Error('AI 无响应')
  aiDraft.value = parseAIDraft(raw)
}

// 点击"AI 帮写"：始终展开关键词输入框，由用户明确输入主题再生成
function handleAI() {
  if (aiLoading.value) return
  showAI.value = !showAI.value
  if (!showAI.value) aiKeywords.value = ''
}

async function generateAI() {
  if (!aiKeywords.value.trim() || aiLoading.value) return
  aiLoading.value = true
  showAI.value = false
  try {
    await callDraftStream(aiKeywords.value)
  } catch (e) {
    toast.error(typeof e === 'string' ? e : (e.message || 'AI 生成失败'))
    aiDraft.value = null
  } finally {
    aiLoading.value = false
  }
}

function useAIDraft() {
  if (!aiDraft.value) return
  if (aiDraft.value.title) title.value = aiDraft.value.title.slice(0, 60)
  if (aiDraft.value.content) content.value = aiDraft.value.content
  aiDraft.value = null
  showAI.value = false
  aiKeywords.value = ''
}

async function submit() {
  if (!title.value.trim() || submitting.value) return
  submitting.value = true
  try {
    const res = await feedApi.createPost({
      title:   title.value.trim(),
      content: content.value.trim(),
      images:  imageUrls.value,
    })
    emit('created', res.data)
    showXP(auth.user?.vip_level > 0 ? 12 : 6)
    toast.success('发布成功！')
  } catch (e) {
    toast.error(typeof e === 'string' ? e : '发布失败')
  } finally {
    submitting.value = false
  }
}
</script>
