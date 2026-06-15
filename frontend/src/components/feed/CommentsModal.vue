<template>
  <Teleport to="body">
    <div class="fixed inset-0 z-50 flex items-end sm:items-center justify-center"
         style="background:rgba(0,0,0,0.6);backdrop-filter:blur(4px)"
         @click.self="$emit('close')">
      <div class="w-full max-w-lg glass-strong sm:rounded-2xl rounded-t-2xl animate-slide-up max-h-[80vh] flex flex-col">
        <div class="flex items-center gap-3 px-5 py-4 shrink-0"
             style="border-bottom:1px solid rgb(var(--ink) / 0.06)">
          <h3 class="font-semibold text-ink">评论</h3>
          <button @click="toggleSort"
                  class="flex items-center gap-1.5 px-2.5 py-1 rounded-lg text-[11px] font-medium transition-all"
                  style="background:rgb(var(--ink) / 0.04);border:1px solid rgb(var(--ink) / 0.08);color:rgb(var(--ink) / 0.6)"
                  :title="sortMode === 'hot' ? '当前按热度，点击切换最新' : '当前按最新，点击切换热度'">
            <Flame v-if="sortMode === 'hot'" :size="12" class="text-orange-400" />
            <Clock v-else :size="12" class="text-primary-light" />
            {{ sortMode === 'hot' ? '热度' : '最新' }}
          </button>
          <button @click="$emit('close')" class="ml-auto text-ink/40 hover:text-ink/70 transition-colors">
            <X :size="18" />
          </button>
        </div>

        <div class="flex-1 overflow-y-auto px-5 py-4 space-y-4">
          <div v-if="loading" class="text-center py-8">
            <Loader2 :size="20" class="animate-spin text-ink/30 mx-auto" />
          </div>
          <div v-for="c in comments" :key="c.id" class="flex gap-3">
            <div class="w-8 h-8 rounded-full shrink-0 overflow-hidden cursor-pointer hover:opacity-90 transition-opacity"
                 @click="openUserProfile(c.user.id)">
              <img v-if="c.user.avatar_url" :src="c.user.avatar_url" class="w-full h-full object-cover" />
              <div v-else class="w-full h-full flex items-center justify-center text-xs font-bold"
                   style="background:linear-gradient(135deg,#3390EC,#2980DE);color:#fff">
                {{ c.user.nickname[0].toUpperCase() }}
              </div>
            </div>
            <div class="flex-1 min-w-0">
              <span class="text-xs font-semibold text-ink/70 mr-1">{{ c.user.nickname }}</span>
              <LevelBadge v-if="c.user.level" :level="c.user.level" :tier="c.user.tier" dense class="mr-1" />
              <span class="text-xs text-ink/40">{{ formatFromNow(c.created_at) }}</span>
              <p class="text-sm text-ink/80 mt-0.5 leading-relaxed break-words">{{ c.content }}</p>
            </div>
            <!-- Like -->
            <button @click="toggleLike(c)"
                    class="flex flex-col items-center gap-0.5 shrink-0 px-1 transition-all"
                    :class="c.is_liked ? 'text-red-400' : 'text-ink/30 hover:text-ink/60'">
              <Heart :size="14" :fill="c.is_liked ? 'currentColor' : 'none'" />
              <span class="text-[10px]">{{ c.like_count || 0 }}</span>
            </button>
          </div>

          <!-- 加载更多 -->
          <div v-if="hasMore" class="text-center pt-1 pb-2">
            <button @click="loadMore" :disabled="loadingMore"
                    class="text-xs text-ink/40 hover:text-ink/70 transition-colors flex items-center gap-1.5 mx-auto">
              <Loader2 v-if="loadingMore" :size="12" class="animate-spin" />
              <ChevronDown v-else :size="12" />
              {{ loadingMore ? '加载中…' : '加载更多评论' }}
            </button>
          </div>

          <p v-if="!loading && !comments.length" class="text-center text-sm text-ink/30 py-8">还没有评论，来抢沙发！</p>
        </div>

        <div class="px-5 py-4 flex gap-3 shrink-0" style="border-top:1px solid rgb(var(--ink) / 0.06)">
          <input v-model="newComment" placeholder="说点什么…" type="text"
                 class="mc-input flex-1 py-2.5 text-sm"
                 @keydown.enter="submit" />
          <button @click="submit" :disabled="!newComment.trim() || submitting"
                  class="btn-primary px-4 py-2.5 text-sm">
            <Loader2 v-if="submitting" :size="14" class="inline animate-spin" />
            <SendHorizonal v-else :size="14" />
          </button>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { X, Loader2, SendHorizonal, Heart, Flame, Clock, ChevronDown } from 'lucide-vue-next'
import LevelBadge   from '@/components/ui/LevelBadge.vue'
import { useAuthStore } from '@/stores/auth'
import { useXPNotify }  from '@/composables/useXPNotify'
import * as feedApi from '@/api/feed'
import { useUserProfile } from '@/composables/useUserProfile'
import { formatFromNow } from '@/utils/time'

const { openUserProfile } = useUserProfile()
const auth = useAuthStore()
const { showXP } = useXPNotify()

const props = defineProps({ postId: { type: Number, required: true } })
const emit  = defineEmits(['close'])

const comments    = ref([])
const loading     = ref(false)
const loadingMore = ref(false)
const hasMore     = ref(false)
const page        = ref(1)
const newComment  = ref('')
const submitting  = ref(false)
const sortMode    = ref('hot')  // 'hot' | 'time'

async function loadComments() {
  loading.value = true
  page.value = 1
  try {
    const res = await feedApi.getComments(props.postId, 1, sortMode.value)
    comments.value = res.data.list || []
    hasMore.value  = res.data.has_more || false
  } finally {
    loading.value = false
  }
}

async function loadMore() {
  if (loadingMore.value) return
  loadingMore.value = true
  try {
    const res = await feedApi.getComments(props.postId, page.value + 1, sortMode.value)
    const list = res.data.list || []
    comments.value.push(...list)
    hasMore.value = res.data.has_more || false
    if (list.length) page.value++
  } finally {
    loadingMore.value = false
  }
}

function toggleSort() {
  sortMode.value = sortMode.value === 'hot' ? 'time' : 'hot'
  loadComments()
}

async function submit() {
  if (!newComment.value.trim() || submitting.value) return
  submitting.value = true
  try {
    const res = await feedApi.createComment(props.postId, newComment.value.trim(), 0)
    comments.value.unshift({
      ...res.data.comment,
      like_count: 0,
      is_liked: false,
      user: { id: auth.user.id, nickname: auth.user.nickname, avatar_url: auth.user.avatar_url }
    })
    if (res.data.xp_gained > 0) showXP(res.data.xp_gained)
    newComment.value = ''
  } finally {
    submitting.value = false
  }
}

async function toggleLike(c) {
  const wasLiked = c.is_liked
  c.is_liked = !wasLiked
  c.like_count = Math.max(0, (c.like_count || 0) + (wasLiked ? -1 : 1))
  try {
    if (wasLiked) await feedApi.unlikeComment(props.postId, c.id)
    else          await feedApi.likeComment(props.postId, c.id)
  } catch {
    c.is_liked = wasLiked
    c.like_count = Math.max(0, (c.like_count || 0) + (wasLiked ? 1 : -1))
  }
}

onMounted(loadComments)
</script>
