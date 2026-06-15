<template>
  <div class="h-full overflow-y-auto">
    <div class="max-w-xl mx-auto px-6 py-8">
      <!-- Header -->
      <div class="mb-6">
        <h2 class="text-2xl font-extrabold text-primary">MeChatFriends</h2>
      </div>

      <!-- Search -->
      <div class="relative mb-2">
        <Search :size="16" class="absolute left-4 top-1/2 -translate-y-1/2 text-ink/30" />
        <input v-model="keyword" @input="onSearch" type="text"
               placeholder="搜索MeChatID或昵称"
               class="mc-input pl-11" />
        <Loader2 v-if="searching" :size="16"
                 class="absolute right-4 top-1/2 -translate-y-1/2 text-ink/30 animate-spin" />
      </div>

      <!-- Search results -->
      <div v-if="keyword && results.length" class="glass rounded-2xl p-2 mb-6 animate-slide-up">
        <div v-for="u in results" :key="u.id"
             class="flex items-center gap-3 px-3 py-2.5 rounded-xl hover:bg-ink/5 transition-all">
          <Avatar :name="u.nickname" :url="u.avatar_url" :size="40" class="cursor-pointer" @click="openUserProfile(u.id)" />
          <div class="flex-1 min-w-0">
            <p class="text-sm font-medium text-ink/90 truncate">{{ u.nickname }}</p>
            <p class="text-xs text-ink/40 truncate">
              <span class="text-ink/55">ID {{ u.id }}</span>
              <span v-if="u.bio"> · {{ u.bio }}</span>
            </p>
          </div>
          <button v-if="friendStore.isFriend(u.id)" disabled
                  class="px-3 py-1.5 rounded-lg text-xs font-medium text-ink/30"
                  style="background:rgb(var(--ink) / 0.04)">已是好友</button>
          <button v-else-if="sentIds.has(u.id)" disabled
                  class="px-3 py-1.5 rounded-lg text-xs font-medium text-accent"
                  style="background:rgba(16,185,129,0.12)">已申请</button>
          <button v-else @click="addFriend(u)"
                  class="flex items-center gap-1 px-3 py-1.5 rounded-lg text-xs font-semibold transition-all"
                  style="background:rgba(51,144,236,0.15);color:#3390EC;border:1px solid rgba(51,144,236,0.3)">
            <UserPlus :size="12" /> 加好友
          </button>
        </div>
      </div>
      <p v-else-if="keyword && !searching && searched" class="text-center text-sm text-ink/30 py-6 mb-6">
        没有找到相关用户
      </p>

      <!-- Friend requests -->
      <div v-if="friendStore.requests.length" class="mb-6">
        <div class="flex items-center gap-2 mb-3">
          <h3 class="text-sm font-semibold text-ink/70">新的朋友</h3>
          <span class="px-1.5 py-0.5 rounded-full text-[10px] font-bold text-ink"
                style="background:#EF4444">{{ friendStore.requests.length }}</span>
        </div>
        <div class="glass rounded-2xl p-2 space-y-1">
          <div v-for="r in friendStore.requests" :key="r.id"
               class="flex items-center gap-3 px-3 py-2.5 rounded-xl">
            <Avatar :name="r.nickname" :url="r.avatar_url" :size="40" class="cursor-pointer" @click="openUserProfile(r.from_user_id)" />
            <div class="flex-1 min-w-0">
              <p class="text-sm font-medium text-ink/90 truncate">{{ r.nickname }}</p>
              <p class="text-xs text-ink/40 truncate">{{ r.message || '请求添加你为好友' }}</p>
            </div>
            <div class="flex items-center gap-2">
              <button @click="respond(r, 'accept')"
                      class="flex items-center gap-1 px-3 py-1.5 rounded-lg text-xs font-semibold text-ink transition-all"
                      style="background:linear-gradient(135deg,#3390EC,#2980DE);color:#fff">
                <Check :size="12" /> 接受
              </button>
              <button @click="respond(r, 'reject')"
                      class="w-7 h-7 rounded-lg flex items-center justify-center text-ink/40 hover:text-ink/70 transition-all"
                      style="background:rgb(var(--ink) / 0.05)">
                <X :size="14" />
              </button>
            </div>
          </div>
        </div>
      </div>

      <!-- Friends list -->
      <div>
        <h3 class="text-sm font-semibold text-ink/70 mb-3">
          我的好友 <span class="text-ink/30">({{ friendStore.friends.length }})</span>
        </h3>

        <div v-if="loadingFriends" class="space-y-2">
          <div v-for="i in 3" :key="i" class="glass rounded-xl h-16 animate-pulse" />
        </div>

        <div v-else-if="friendStore.friends.length" class="glass rounded-2xl p-2 space-y-1">
          <div v-for="f in friendStore.friends" :key="f.user_id"
               class="group flex items-center gap-3 px-3 py-2.5 rounded-xl hover:bg-ink/5 transition-all">
            <div class="relative shrink-0">
              <Avatar :name="f.nickname" :url="f.avatar_url" :size="44" class="cursor-pointer" @click="openUserProfile(f.user_id)" />
              <span class="absolute bottom-0 right-0 w-3 h-3 rounded-full border-2"
                    :style="`border-color:rgb(var(--surface-2));background:${f.is_online ? '#10B981' : '#6B7280'}`" />
            </div>
            <div class="flex-1 min-w-0">
              <div class="flex items-center gap-1.5">
                <p class="text-sm font-medium text-ink/90 truncate">{{ f.nickname }}</p>
                <LevelBadge v-if="f.level" :level="f.level" :tier="f.tier" dense />
              </div>
              <p class="text-xs" :class="f.is_online ? 'text-accent' : 'text-ink/40'">
                {{ f.is_online ? '在线' : '离线' }}
              </p>
            </div>
            <div class="flex items-center gap-1">
              <button @click="message(f)"
                      class="flex items-center gap-1 px-3 py-1.5 rounded-lg text-xs font-medium transition-all"
                      style="background:rgba(51,144,236,0.12);color:#3390EC">
                <MessageSquare :size="12" /> 发消息
              </button>
              <button @click="confirmRemove(f)"
                      class="w-7 h-7 rounded-lg flex items-center justify-center text-ink/30 hover:text-danger transition-all"
                      style="background:rgb(var(--ink) / 0.05)">
                <UserMinus :size="13" />
              </button>
            </div>
          </div>
        </div>

        <div v-else class="text-center py-8">
          <Users :size="40" class="mx-auto text-ink/20 mb-3" />
          <p class="text-ink/40 text-sm">还没有好友</p>
          <p class="text-ink/25 text-xs mt-1">用上方搜索框找到并添加好友吧</p>
        </div>
      </div>

      <!-- Recommended users -->
      <div v-if="recommended.length" class="mt-6">
        <h3 class="text-sm font-semibold text-ink/70 mb-3 flex items-center gap-2">
          <Sparkles :size="14" class="text-primary-light" /> 推荐认识
        </h3>
        <div class="glass rounded-2xl p-2 space-y-1">
          <div v-for="u in recommended" :key="u.id"
               class="flex items-center gap-3 px-3 py-2.5 rounded-xl hover:bg-ink/5 transition-all">
            <Avatar :name="u.nickname" :url="u.avatar_url" :size="42" class="cursor-pointer" @click="openUserProfile(u.id)" />
            <div class="flex-1 min-w-0">
              <p class="text-sm font-medium text-ink/90 truncate">{{ u.nickname }}</p>
              <p class="text-xs text-ink/40 truncate">
                <span class="text-ink/55">ID {{ u.id }}</span>
                <span v-if="u.bio"> · {{ u.bio }}</span>
              </p>
            </div>
            <button @click="addRecommended(u)"
                    :disabled="pendingIds.has(u.id)"
                    class="flex items-center gap-1 px-3 py-1.5 rounded-lg text-xs font-medium transition-all shrink-0"
                    style="background:rgba(51,144,236,0.12);color:#3390EC;border:1px solid rgba(51,144,236,0.2)"
                    :class="pendingIds.has(u.id) && 'opacity-50 cursor-not-allowed'">
              <UserPlus :size="12" />
              {{ pendingIds.has(u.id) ? '已发送' : '加好友' }}
            </button>
          </div>
        </div>
      </div>
    </div>

    <!-- Remove confirm -->
    <Teleport to="body">
      <div v-if="removeTarget"
           class="fixed inset-0 z-50 flex items-center justify-center p-4"
           style="background:rgba(0,0,0,0.6);backdrop-filter:blur(4px)"
           @click.self="removeTarget = null">
        <div class="glass-strong rounded-2xl p-6 max-w-sm w-full animate-scale-in text-center">
          <UserMinus :size="32" class="text-danger/60 mx-auto mb-4" />
          <h3 class="font-semibold text-ink mb-2">删除好友？</h3>
          <p class="text-sm text-ink/40 mb-6">将与「{{ removeTarget.nickname }}」解除好友关系</p>
          <div class="flex gap-3">
            <button @click="removeTarget = null" class="btn-ghost flex-1 py-2.5">取消</button>
            <button @click="doRemove"
                    class="flex-1 py-2.5 rounded-md text-sm font-semibold text-ink transition-all"
                    style="background:rgba(239,68,68,0.2);border:1px solid rgba(239,68,68,0.3)">
              删除
            </button>
          </div>
        </div>
      </div>
    </Teleport>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { Search, UserPlus, Check, X, MessageSquare, UserMinus, Users, Loader2, Sparkles } from 'lucide-vue-next'
import LevelBadge from '@/components/ui/LevelBadge.vue'
import { useFriendStore }           from '@/stores/friend'
import { useChatStore }             from '@/stores/chat'
import { useAuthStore }             from '@/stores/auth'
import { useToast }                 from '@/composables/useToast'
import { searchUsers, recommendUsers } from '@/api/auth'
import { useUserProfile }           from '@/composables/useUserProfile'
import { useAddFriend }             from '@/composables/useAddFriend'
import Avatar                       from '@/components/ui/Avatar.vue'

const { openUserProfile } = useUserProfile()
const { openAddFriend }   = useAddFriend()

const router      = useRouter()
const friendStore = useFriendStore()
const chatStore   = useChatStore()
const auth        = useAuthStore()
const toast       = useToast()

const keyword   = ref('')
const results   = ref([])
const searching = ref(false)
const searched  = ref(false)
const sentIds   = ref(new Set())
const loadingFriends = ref(false)
const removeTarget   = ref(null)
const recommended    = ref([])
const pendingIds     = ref(new Set())

let searchTimer = null
function onSearch() {
  clearTimeout(searchTimer)
  const q = keyword.value.trim()
  if (!q) { results.value = []; searched.value = false; return }
  searchTimer = setTimeout(async () => {
    searching.value = true
    try {
      const res = await searchUsers(q)
      results.value = (res.data.list || []).filter(u => u.id !== auth.user?.id)
      searched.value = true
    } catch {
      results.value = []
    } finally {
      searching.value = false
    }
  }, 300)
}

function addFriend(u) {
  // 打开带招呼语输入的弹层；发送成功后由回调把 u 标记为「已申请」
  openAddFriend(u, () => {
    sentIds.value.add(u.id)
    sentIds.value = new Set(sentIds.value) // 重新赋值以触发 Set 的响应式更新
  })
}

async function respond(r, action) {
  try {
    await friendStore.handleRequest(r.id, action)
    toast.success(action === 'accept' ? `已添加 ${r.nickname}` : '已拒绝')
  } catch (e) {
    toast.error(typeof e === 'string' ? e : '操作失败')
  }
}

async function message(f) {
  try {
    await chatStore.openOrCreatePrivateChat(f.user_id)
    router.push('/chat')
  } catch (e) {
    toast.error(typeof e === 'string' ? e : '无法发起会话')
  }
}

function confirmRemove(f) { removeTarget.value = f }

async function doRemove() {
  const f = removeTarget.value
  removeTarget.value = null
  try {
    await friendStore.removeFriend(f.user_id)
    toast.success('已删除好友')
  } catch (e) {
    toast.error(typeof e === 'string' ? e : '删除失败')
  }
}

function addRecommended(u) {
  openAddFriend(u, () => {
    pendingIds.value.add(u.id)
    pendingIds.value = new Set(pendingIds.value)
  })
}

onMounted(async () => {
  loadingFriends.value = true
  try {
    await Promise.all([friendStore.loadFriends(), friendStore.loadRequests()])
  } finally {
    loadingFriends.value = false
  }
  try {
    const res = await recommendUsers()
    recommended.value = (res.data.list || []).filter(u => u.id !== auth.user?.id)
  } catch {}
})
</script>
