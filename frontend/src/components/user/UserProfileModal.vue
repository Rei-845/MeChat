<template>
  <Teleport to="body">
    <Transition name="profile">
      <div v-if="viewingUserId" class="fixed inset-0 z-[150] flex items-center justify-center p-4"
           style="background:rgba(0,0,0,0.6);backdrop-filter:blur(6px)"
           @click.self="closeUserProfile">
        <div class="relative w-full max-w-md glass-strong rounded-2xl overflow-hidden flex flex-col animate-scale-in"
             style="max-height:82vh">
          <!-- Close -->
          <button @click="closeUserProfile"
                  class="absolute top-3 right-3 z-10 w-8 h-8 rounded-full flex items-center justify-center
                         text-ink/50 hover:text-ink transition-colors"
                  style="background:rgba(0,0,0,0.25)">
            <X :size="18" />
          </button>

          <!-- Header -->
          <div class="relative px-6 pt-7 pb-5 shrink-0"
               :style="user?.vip
                 ? 'background:linear-gradient(160deg,rgba(245,158,11,0.14),rgba(245,158,11,0.02) 60%,transparent)'
                 : 'background:linear-gradient(160deg,rgba(51,144,236,0.12),transparent 60%)'">
            <div class="flex items-center gap-4">
              <!-- Big avatar (gold ring for VIP) -->
              <div class="w-16 h-16 rounded-2xl shrink-0 p-[2px]"
                   :style="user?.vip ? 'background:linear-gradient(135deg,#F59E0B,#FBBF24)' : 'background:rgb(var(--ink) / 0.1)'">
                <div class="w-full h-full rounded-2xl overflow-hidden" style="background:rgb(var(--surface-2))">
                  <img v-if="user?.avatar_url" :src="user.avatar_url" class="w-full h-full object-cover" />
                  <div v-else class="w-full h-full flex items-center justify-center text-xl font-bold"
                       style="background:linear-gradient(135deg,#3390EC,#2980DE);color:#fff">
                    {{ user?.nickname?.[0]?.toUpperCase() || '?' }}
                  </div>
                </div>
              </div>
              <div class="min-w-0 flex-1">
                <div class="flex items-center gap-2 flex-wrap">
                  <h3 class="text-lg font-bold text-ink truncate">{{ user?.nickname || '加载中…' }}</h3>
                  <LevelBadge v-if="user?.level" :level="user.level" :tier="user.tier" />
                  <VipBadge v-if="user?.vip" />
                </div>
                <p v-if="user" class="text-[11px] text-ink/40 mt-0.5">MeChatID: {{ user.id }}</p>
                <p class="text-xs text-ink/50 mt-1 line-clamp-2">{{ user?.bio || '这个人很神秘，什么都没留下' }}</p>
              </div>
            </div>

            <!-- Actions -->
            <div v-if="user && !isSelf" class="flex gap-2 mt-4">
              <button v-if="isFriend" @click="message"
                      class="flex-1 flex items-center justify-center gap-1.5 py-2 rounded-xl text-sm font-medium transition-all"
                      style="background:linear-gradient(135deg,#3390EC,#2980DE);color:#fff;color:#fff">
                <MessageSquare :size="14" /> 发消息
              </button>
              <template v-else>
                <button v-if="!requested" @click="addFriend"
                        class="flex-1 flex items-center justify-center gap-1.5 py-2 rounded-xl text-sm font-medium transition-all"
                        style="background:rgba(51,144,236,0.15);color:#3390EC;border:1px solid rgba(51,144,236,0.3)">
                  <UserPlus :size="14" /> 加好友
                </button>
                <span v-else class="flex-1 flex items-center justify-center gap-1.5 py-2 rounded-xl text-sm font-medium"
                      style="background:rgb(var(--ink) / 0.05);color:rgb(var(--ink) / 0.4)">
                  <Check :size="14" /> 已申请
                </span>
              </template>
            </div>
          </div>

          <!-- Posts -->
          <div ref="postScroll" class="flex-1 overflow-y-auto px-4 pb-4">
            <div class="flex items-center gap-2 px-2 py-2 text-xs font-semibold text-ink/40">
              <span>帖子</span>
              <span class="text-ink/20">·</span>
              <span>{{ posts.length }}{{ hasMore ? '+' : '' }}</span>
            </div>

            <div v-if="loadingFirst" class="text-center py-8">
              <Loader2 :size="20" class="animate-spin text-ink/30 mx-auto" />
            </div>

            <template v-else>
              <div v-if="posts.length" class="space-y-2">
                <div v-for="p in posts" :key="p.post_id"
                     class="rounded-xl p-3 cursor-pointer transition-all hover:bg-ink/[0.04]"
                     style="background:rgb(var(--ink) / 0.025);border:1px solid rgb(var(--ink) / 0.06)"
                     @click="goPost(p.post_id)">
                  <p class="text-sm font-semibold text-ink/90 leading-snug line-clamp-2 break-words">{{ p.title }}</p>
                  <p v-if="p.content" class="text-xs text-ink/55 leading-relaxed line-clamp-2 whitespace-pre-wrap break-words mt-1">{{ p.content }}</p>
                  <div v-if="p.images?.length" class="flex gap-1.5 mt-2">
                    <img v-for="(img, i) in p.images.slice(0, 3)" :key="i" :src="img"
                         class="w-14 h-14 rounded-lg object-cover" loading="lazy" />
                  </div>
                  <div class="flex items-center gap-4 mt-2 text-[11px] text-ink/35">
                    <span class="flex items-center gap-1"><Heart :size="11" /> {{ p.like_count }}</span>
                    <span class="flex items-center gap-1"><MessageCircle :size="11" /> {{ p.comment_count }}</span>
                    <span class="ml-auto">{{ formatTime(p.created_at) }}</span>
                  </div>
                </div>
              </div>

              <div v-else class="text-center py-10 text-ink/30 text-sm">
                <Newspaper :size="36" class="mx-auto mb-2 opacity-40" />
                还没有发过帖子
              </div>

              <!-- Infinite scroll sentinel -->
              <div ref="sentinel" class="h-1" />

              <div v-if="loadingMore" class="flex justify-center py-3">
                <Loader2 :size="16" class="animate-spin text-ink/30" />
              </div>
            </template>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup>
import { ref, computed, watch, nextTick, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { X, MessageSquare, UserPlus, Check, Loader2, Heart, MessageCircle, Newspaper } from 'lucide-vue-next'
import { getUser }       from '@/api/auth'
import * as feedApi      from '@/api/feed'
import { useAuthStore }   from '@/stores/auth'
import { useFriendStore } from '@/stores/friend'
import { useChatStore }   from '@/stores/chat'
import { useToast }       from '@/composables/useToast'
import { useUserProfile } from '@/composables/useUserProfile'
import { useAddFriend }   from '@/composables/useAddFriend'
import VipBadge   from '@/components/ui/VipBadge.vue'
import LevelBadge from '@/components/ui/LevelBadge.vue'
import { formatRelative as formatTime } from '@/utils/time'

const router      = useRouter()
const auth        = useAuthStore()
const friendStore = useFriendStore()
const chatStore   = useChatStore()
const toast       = useToast()
const { viewingUserId, closeUserProfile } = useUserProfile()
const { openAddFriend } = useAddFriend()

const user         = ref(null)
const posts        = ref([])
const loadingFirst = ref(false)
const loadingMore  = ref(false)
const hasMore      = ref(false)
const page         = ref(1)
const requested    = ref(false)
const postScroll   = ref(null)
const sentinel     = ref(null)

const isSelf   = computed(() => user.value && user.value.id === auth.user?.id)
const isFriend = computed(() => user.value && friendStore.isFriend(user.value.id))

let scrollObserver = null

function setupObserver() {
  if (scrollObserver) scrollObserver.disconnect()
  if (!sentinel.value || !postScroll.value) return
  scrollObserver = new IntersectionObserver((entries) => {
    if (entries[0].isIntersecting && !loadingMore.value && hasMore.value) {
      loadMore()
    }
  }, { root: postScroll.value, rootMargin: '0px 0px 200px 0px' })
  scrollObserver.observe(sentinel.value)
}

async function loadMore() {
  if (loadingMore.value || !hasMore.value) return
  loadingMore.value = true
  try {
    page.value++
    const res = await feedApi.getUserPosts(viewingUserId.value, page.value)
    const data = res.data
    posts.value.push(...(data.list || []))
    hasMore.value = data.has_more ?? false
  } catch {
    page.value--
  } finally {
    loadingMore.value = false
  }
}

watch(viewingUserId, async (id) => {
  if (scrollObserver) { scrollObserver.disconnect(); scrollObserver = null }
  if (!id) return
  user.value = null
  posts.value = []
  page.value = 1
  hasMore.value = false
  requested.value = false
  loadingFirst.value = true
  try {
    const [uRes, pRes] = await Promise.allSettled([getUser(id), feedApi.getUserPosts(id, 1)])
    if (uRes.status === 'fulfilled') user.value = uRes.value.data
    if (pRes.status === 'fulfilled') {
      const data = pRes.value.data
      posts.value = data.list || []
      hasMore.value = data.has_more ?? false
    }
  } finally {
    loadingFirst.value = false
    nextTick(() => setupObserver())
  }
})

onUnmounted(() => { if (scrollObserver) scrollObserver.disconnect() })

function addFriend() {
  // 打开带招呼语输入的弹层；发送成功后标记为「已申请」
  openAddFriend(user.value, () => { requested.value = true })
}

async function message() {
  try {
    await chatStore.openOrCreatePrivateChat(user.value.id)
    closeUserProfile()
    router.push('/chat')
  } catch (e) {
    toast.error(typeof e === 'string' ? e : '无法发起会话')
  }
}

function goPost(id) {
  closeUserProfile()
  router.push(`/post/${id}`)
}

</script>

<style scoped>
.profile-enter-active, .profile-leave-active { transition: opacity 0.2s ease; }
.profile-enter-from, .profile-leave-to { opacity: 0; }
</style>
