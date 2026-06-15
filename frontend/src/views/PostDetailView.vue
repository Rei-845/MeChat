<template>
  <div ref="pageScroll" class="h-full overflow-y-auto" @scroll="onPageScroll">
    <div class="max-w-xl mx-auto px-4 pt-6 pb-28">
      <!-- Back -->
      <button @click="goBack"
              class="flex items-center gap-1.5 mb-4 text-sm text-ink/50 hover:text-ink/80 transition-colors">
        <ArrowLeft :size="16" /> 返回
      </button>

      <!-- Loading -->
      <div v-if="loading" class="glass rounded-2xl p-5 animate-pulse">
        <div class="flex gap-3 mb-4">
          <div class="w-11 h-11 rounded-full bg-ink/5" />
          <div class="flex-1 space-y-2 py-1">
            <div class="h-3 bg-ink/5 rounded w-32" />
            <div class="h-2 bg-ink/5 rounded w-20" />
          </div>
        </div>
        <div class="h-3 bg-ink/5 rounded mb-2" />
        <div class="h-3 bg-ink/5 rounded w-4/5" />
      </div>

      <template v-else-if="post">
        <!-- Post -->
        <PostCard v-double-tap="dblLikePost" :post="post" :clickable="false" @like="toggleLike" @comment="focusInput" @delete="onDelete" />

        <!-- Comments -->
        <div class="mt-4">
          <div class="flex items-center px-1 mb-3">
            <h3 class="text-sm font-semibold text-ink/60">
              评论 <span class="text-ink/30">{{ post.comment_count || 0 }}</span>
            </h3>
            <button @click="toggleSort"
                    class="ml-auto flex items-center gap-1.5 px-2.5 py-1 rounded-lg text-[11px] font-medium transition-all"
                    style="background:rgb(var(--ink) / 0.04);border:1px solid rgb(var(--ink) / 0.08);color:rgb(var(--ink) / 0.6)"
                    :title="sortMode === 'hot' ? '当前按热度，点击切换最新' : '当前按最新，点击切换热度'">
              <Flame v-if="sortMode === 'hot'" :size="12" class="text-orange-400" />
              <Clock v-else :size="12" class="text-primary-light" />
              {{ sortMode === 'hot' ? '热度' : '最新' }}
            </button>
          </div>

          <div v-if="loadingComments" class="text-center py-8">
            <Loader2 :size="20" class="animate-spin text-ink/30 mx-auto" />
          </div>

          <div v-else-if="comments.length" class="space-y-3">
            <div v-for="c in comments" :key="c.id"
                 class="glass rounded-xl p-3"
                 style="border:1px solid rgb(var(--ink) / 0.05)">
              <!-- Root comment -->
              <div class="flex gap-3" v-double-tap="(p) => dblLikeComment(c, p)">
                <Avatar :name="c.user.nickname" :url="c.user.avatar_url" :size="32"
                        class="cursor-pointer hover:opacity-90 transition-opacity"
                        @click="openUserProfile(c.user.id)" />
                <div class="flex-1 min-w-0">
                  <div class="flex items-center gap-1.5 flex-wrap">
                    <span class="text-xs font-semibold text-ink/80">{{ c.user.nickname }}</span>
                    <LevelBadge v-if="c.user.level" :level="c.user.level" :tier="c.user.tier" dense />
                    <span class="text-xs text-ink/30">{{ formatFromNow(c.created_at) }}</span>
                  </div>
                  <p class="text-sm text-ink/80 mt-0.5 leading-relaxed break-words">{{ c.content }}</p>
                  <!-- Actions row -->
                  <div class="flex items-center gap-3 mt-1.5">
                    <button @click="startReply(c)"
                            class="text-xs text-ink/30 hover:text-primary-light transition-colors flex items-center gap-1">
                      <Reply :size="12" /> 回复
                    </button>
                  </div>
                </div>
                <!-- Like -->
                <button @click="toggleCommentLike(c)"
                        class="flex flex-col items-center gap-0.5 shrink-0 px-1 transition-all"
                        :class="c.is_liked ? 'text-red-400' : 'text-ink/30 hover:text-ink/60'">
                  <Heart :size="14" :fill="c.is_liked ? 'currentColor' : 'none'" />
                  <span class="text-[10px]">{{ c.like_count || 0 }}</span>
                </button>
              </div>

              <!-- Replies -->
              <div v-if="c.replies?.length" class="mt-2.5 ml-11 space-y-2.5 pl-3"
                   style="border-left:2px solid rgb(var(--ink) / 0.06)">
                <div v-for="r in c.replies" :key="r.id" class="flex gap-2.5" v-double-tap="(p) => dblLikeComment(r, p)">
                  <Avatar :name="r.user.nickname" :url="r.user.avatar_url" :size="24"
                          class="cursor-pointer hover:opacity-90 transition-opacity"
                          @click="openUserProfile(r.user.id)" />
                  <div class="flex-1 min-w-0">
                    <div class="flex items-center gap-1.5 flex-wrap">
                      <span class="text-xs font-semibold text-ink/70">{{ r.user.nickname }}</span>
                      <LevelBadge v-if="r.user.level" :level="r.user.level" :tier="r.user.tier" dense />
                      <span class="text-xs text-ink/25">{{ formatFromNow(r.created_at) }}</span>
                    </div>
                    <p class="text-[13px] text-ink/70 mt-0.5 leading-relaxed break-words">{{ r.content }}</p>
                  </div>
                  <!-- Reply like -->
                  <button @click="toggleCommentLike(r)"
                          class="flex flex-col items-center gap-0.5 shrink-0 transition-all"
                          :class="r.is_liked ? 'text-red-400' : 'text-ink/25 hover:text-ink/50'">
                    <Heart :size="12" :fill="r.is_liked ? 'currentColor' : 'none'" />
                    <span class="text-[10px]">{{ r.like_count || 0 }}</span>
                  </button>
                </div>

                <!-- 展开回复（首次） -->
                <button v-if="c.has_more_replies && !expandedReplies.has(c.id)"
                        @click="expandReplies(c)"
                        class="flex items-center gap-1 text-xs text-ink/40 hover:text-primary-light transition-colors pt-0.5">
                  <Loader2 v-if="loadingReplies.has(c.id)" :size="11" class="animate-spin" />
                  <ChevronDown v-else :size="11" />
                  {{ loadingReplies.has(c.id) ? '加载中…' : '展开回复' }}
                </button>
                <!-- 加载更多回复 -->
                <button v-else-if="replyHasMore[c.id]"
                        @click="loadMoreReplies(c)"
                        class="flex items-center gap-1 text-xs text-ink/40 hover:text-primary-light transition-colors pt-0.5">
                  <Loader2 v-if="loadingReplies.has(c.id)" :size="11" class="animate-spin" />
                  <ChevronDown v-else :size="11" />
                  {{ loadingReplies.has(c.id) ? '加载中…' : '加载更多回复' }}
                </button>
              </div>
            </div>
          </div>

          <!-- 无限滚动：底部加载指示 -->
          <div v-if="hasMoreComments || loadingMoreComments" class="text-center py-4">
            <Loader2 v-if="loadingMoreComments" :size="16" class="animate-spin text-ink/30 mx-auto" />
          </div>

          <p v-else-if="!loadingComments && !comments.length" class="text-center text-sm text-ink/30 py-10">还没有评论，来抢沙发！</p>
        </div>
      </template>

      <div v-else class="text-center py-20">
        <FileQuestion :size="48" class="mx-auto text-ink/20 mb-4" />
        <p class="text-ink/40 text-sm">帖子不存在或已被删除</p>
      </div>
    </div>
  </div>

  <!-- 回到顶部 -->
  <Transition enter-active-class="transition-all duration-200" enter-from-class="opacity-0 scale-75"
              leave-active-class="transition-all duration-200" leave-to-class="opacity-0 scale-75">
    <button v-if="showBackTop" @click="scrollToTop"
            class="fixed bottom-36 md:bottom-20 right-5 md:right-7 z-30 w-10 h-10 rounded-full
                   flex items-center justify-center transition-all hover:scale-110 active:scale-95"
            style="background:rgb(var(--ink) / 0.08);border:1px solid rgb(var(--ink) / 0.12);backdrop-filter:blur(8px)"
            title="回到顶部">
      <ArrowUp :size="18" class="text-ink/60" />
    </button>
  </Transition>

  <!-- 悬浮评论输入卡片：filter 不能加在 fixed 元素上（会破坏 fixed 定位），shadow 由 glass 自带 -->
  <div v-if="post"
       class="fixed bottom-20 md:bottom-6 left-4 right-4 md:left-[72px] md:right-0 z-40">
    <div class="max-w-xl mx-auto md:px-6">
      <div class="glass rounded-2xl p-3" style="border:none">
        <Transition enter-active-class="transition-all duration-150 ease-out"
                    enter-from-class="opacity-0 -translate-y-1"
                    leave-active-class="transition-all duration-100 ease-in"
                    leave-to-class="opacity-0 -translate-y-1">
          <div v-if="replyingTo" class="flex items-center gap-2 mb-2">
            <Reply :size="12" class="text-primary-light shrink-0" />
            <span class="text-xs text-ink/50">回复 <span class="text-primary-light font-medium">{{ replyingTo.user.nickname }}</span></span>
            <button @click="cancelReply" class="ml-auto text-ink/30 hover:text-ink/60 transition-colors">
              <X :size="13" />
            </button>
          </div>
        </Transition>
        <div class="flex gap-2 items-center">
          <input ref="inputEl" v-model="newComment"
                 :placeholder="replyingTo ? `回复 ${replyingTo.user.nickname}…` : '写下你的评论…'"
                 type="text"
                 class="flex-1 py-2 px-1 text-sm text-ink/90 placeholder-ink/30 bg-transparent outline-none border-0"
                 @keydown.enter="submit" />
          <button @click="submit" :disabled="!newComment.trim() || submitting"
                  class="btn-primary px-4 py-2.5 text-sm shrink-0 transition-opacity"
                  :class="(!newComment.trim() || submitting) && 'opacity-50'">
            <Loader2 v-if="submitting" :size="14" class="animate-spin" />
            <SendHorizonal v-else :size="14" />
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, watch, nextTick } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ArrowLeft, FileQuestion, Loader2, SendHorizonal, Heart, Reply, X, Flame, Clock, ChevronDown, ArrowUp } from 'lucide-vue-next'
import * as feedApi from '@/api/feed'
import PostCard      from '@/components/feed/PostCard.vue'
import LevelBadge   from '@/components/ui/LevelBadge.vue'
import Avatar       from '@/components/ui/Avatar.vue'
import { useAuthStore }   from '@/stores/auth'
import { useUserProfile } from '@/composables/useUserProfile'
import { useToast }    from '@/composables/useToast'
import { useXPNotify } from '@/composables/useXPNotify'
import { heartPop }    from '@/utils/heartPop'
import { formatFromNow } from '@/utils/time'

const route  = useRoute()
const router = useRouter()
const auth   = useAuthStore()
const toast  = useToast()
const { showXP } = useXPNotify()
const { openUserProfile } = useUserProfile()

const post                 = ref(null)
const loading              = ref(true)
const comments             = ref([])
const loadingComments      = ref(false)
const loadingMoreComments  = ref(false)
const hasMoreComments      = ref(false)
const commentsPage         = ref(1)
const newComment           = ref('')
const submitting           = ref(false)
const inputEl              = ref(null)
const replyingTo           = ref(null)
const sortMode             = ref('hot')
// 已展开回复的评论 ID 集合，以及正在加载中的集合
const expandedReplies      = ref(new Set())
const loadingReplies       = ref(new Set())
const replyHasMore         = ref({})   // commentId -> bool
const replyPages           = ref({})   // commentId -> 已加载到的页码

// 回到顶部 + 无限滚动评论
const pageScroll  = ref(null)
const showBackTop = ref(false)
function onPageScroll(e) {
  const el = e.target
  showBackTop.value = el.scrollTop > 400
  if (el.scrollHeight - el.scrollTop - el.clientHeight < 150 &&
      !loadingMoreComments.value && hasMoreComments.value && !loadingComments.value) {
    loadMoreComments()
  }
}
function scrollToTop() { pageScroll.value?.scrollTo({ top: 0, behavior: 'smooth' }) }

function toggleSort() {
  sortMode.value = sortMode.value === 'hot' ? 'time' : 'hot'
  loadComments()
}

function goBack() {
  if (window.history.length > 1) router.back()
  else router.push('/feed')
}

function focusInput() {
  nextTick(() => inputEl.value?.focus())
}

function startReply(c) {
  replyingTo.value = c
  nextTick(() => inputEl.value?.focus())
}

function cancelReply() {
  replyingTo.value = null
  newComment.value = ''
}

async function toggleLike(p) {
  try {
    if (p.is_liked) {
      await feedApi.unlikePost(p.post_id)
      p.is_liked = false; p.like_count = Math.max(0, p.like_count - 1)
    } else {
      await feedApi.likePost(p.post_id)
      p.is_liked = true; p.like_count++
    }
  } catch {}
}

async function onDelete(p) {
  try { await feedApi.deletePost(p.post_id); router.push('/feed') } catch {}
}

// 移动端双击点赞：仅「点赞」从不「取消」，并在轻触处弹出红心动画
function dblLikePost(pos) {
  if (!post.value) return
  if (!post.value.is_liked) toggleLike(post.value)
  if (pos) heartPop(pos.x, pos.y)
}
function dblLikeComment(c, pos) {
  if (!c.is_liked) toggleCommentLike(c)
  if (pos) heartPop(pos.x, pos.y)
}

async function loadComments() {
  loadingComments.value = true
  commentsPage.value = 1
  expandedReplies.value = new Set()
  try {
    const res = await feedApi.getComments(route.params.id, 1, sortMode.value)
    comments.value     = res.data.list || []
    hasMoreComments.value = res.data.has_more || false
  } finally {
    loadingComments.value = false
  }
}

async function loadMoreComments() {
  if (loadingMoreComments.value) return
  loadingMoreComments.value = true
  try {
    const res = await feedApi.getComments(route.params.id, commentsPage.value + 1, sortMode.value)
    const list = res.data.list || []
    comments.value.push(...list)
    hasMoreComments.value = res.data.has_more || false
    if (list.length) commentsPage.value++
  } finally {
    loadingMoreComments.value = false
  }
}

// 展开某条评论的回复（加载第一页）
async function expandReplies(comment) {
  if (loadingReplies.value.has(comment.id)) return
  loadingReplies.value = new Set([...loadingReplies.value, comment.id])
  try {
    const res = await feedApi.getReplies(route.params.id, comment.id, 1, 10)
    comment.replies = res.data.list || []
    replyHasMore.value[comment.id] = res.data.has_more || false
    replyPages.value[comment.id] = 1
    expandedReplies.value = new Set([...expandedReplies.value, comment.id])
  } catch {
    toast.error('加载回复失败，请重试')
  } finally {
    const s = new Set(loadingReplies.value)
    s.delete(comment.id)
    loadingReplies.value = s
  }
}

// 加载下一页回复
async function loadMoreReplies(comment) {
  if (loadingReplies.value.has(comment.id)) return
  loadingReplies.value = new Set([...loadingReplies.value, comment.id])
  try {
    const nextPage = (replyPages.value[comment.id] || 1) + 1
    const res = await feedApi.getReplies(route.params.id, comment.id, nextPage, 10)
    const list = res.data.list || []
    comment.replies = [...(comment.replies || []), ...list]
    replyHasMore.value[comment.id] = res.data.has_more || false
    if (list.length) replyPages.value[comment.id] = nextPage
  } catch {
    toast.error('加载回复失败，请重试')
  } finally {
    const s = new Set(loadingReplies.value)
    s.delete(comment.id)
    loadingReplies.value = s
  }
}

async function submit() {
  if (!newComment.value.trim() || submitting.value) return
  submitting.value = true
  const parentId = replyingTo.value?.id || 0
  const text = newComment.value.trim()
  try {
    const res = await feedApi.createComment(route.params.id, text, parentId)
    const newItem = {
      ...res.data.comment, like_count: 0, is_liked: false,
      user: { id: auth.user.id, nickname: auth.user.nickname, avatar_url: auth.user.avatar_url },
    }
    if (parentId === 0) {
      comments.value.unshift({ ...newItem, replies: [], has_more_replies: false })
      if (post.value) post.value.comment_count = (post.value.comment_count || 0) + 1
    } else {
      const parent = comments.value.find(c => c.id === parentId)
      if (parent) {
        if (!parent.replies) parent.replies = []
        parent.replies.push(newItem)
      }
      if (post.value) post.value.comment_count = (post.value.comment_count || 0) + 1
    }
    if (res.data.xp_gained > 0) showXP(res.data.xp_gained)
    newComment.value = ''
    replyingTo.value = null
  } finally {
    submitting.value = false
  }
}

async function toggleCommentLike(c) {
  const was = c.is_liked
  c.is_liked = !was
  c.like_count = Math.max(0, (c.like_count || 0) + (was ? -1 : 1))
  try {
    if (was) await feedApi.unlikeComment(route.params.id, c.id)
    else     await feedApi.likeComment(route.params.id, c.id)
  } catch {
    c.is_liked = was
    c.like_count = Math.max(0, (c.like_count || 0) + (was ? 1 : -1))
  }
}

async function loadPost(id) {
  loading.value = true
  post.value = null
  comments.value = []
  replyingTo.value = null
  newComment.value = ''
  expandedReplies.value = new Set()
  replyHasMore.value = {}
  replyPages.value = {}
  try {
    const res = await feedApi.getPost(id)
    post.value = res.data
  } catch {
    post.value = null
  } finally {
    loading.value = false
  }
  if (post.value) loadComments()
}

onMounted(() => loadPost(route.params.id))

watch(() => route.params.id, (newId, oldId) => {
  if (newId && newId !== oldId) loadPost(newId)
})
</script>
