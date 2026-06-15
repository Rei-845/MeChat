<template>
  <div class="flex h-full">
    <!-- Conversation list:
         Desktop: fixed 280px column
         Mobile:  full screen, hidden when a conv is open -->
    <aside class="flex flex-col shrink-0 transition-all"
           :class="mobileShowList ? 'flex' : 'hidden md:flex'"
           style="border-right:1px solid rgb(var(--ink) / 0.06);background:rgb(var(--ink) / 0.015)"
           :style="{ width: isMobile ? '100%' : '280px' }">
      <!-- Header -->
      <div class="flex items-center justify-between px-4 py-4 shrink-0">
        <h2 class="text-2xl font-extrabold text-primary">MeChat</h2>
        <div class="flex items-center gap-1">
          <button @click="refreshChat" :disabled="refreshing"
                  class="w-8 h-8 rounded-lg flex items-center justify-center
                         text-ink/40 hover:text-ink hover:bg-ink/8 transition-all"
                  :class="refreshing && 'opacity-40'" title="刷新">
            <RefreshCw :size="15" :class="refreshing && 'animate-spin'" />
          </button>
          <button @click="showNewChat = true"
                  class="w-8 h-8 rounded-lg flex items-center justify-center
                         text-ink/40 hover:text-ink hover:bg-ink/8 transition-all"
                  title="发起聊天">
            <SquarePen :size="16" />
          </button>
        </div>
      </div>

      <!-- Search -->
      <div class="px-3 pb-3">
        <div class="relative">
          <Search :size="14" class="absolute left-3 top-1/2 -translate-y-1/2 text-ink/30" />
          <input v-model="search" placeholder="搜索联系人…" type="search"
                 class="mc-input pl-9 py-2 text-xs" />
        </div>
      </div>

      <!-- Conversation list -->
      <div ref="convListScroll" class="flex-1 overflow-y-auto px-2 space-y-0.5">
        <!-- 下拉刷新指示器 -->
        <div class="flex justify-center items-end overflow-hidden"
             :style="{ height: ptr.pullY.value + 'px' }">
          <RefreshCw :size="16" class="mb-1 text-ink/40"
                     :class="ptr.refreshing.value && 'animate-spin'"
                     :style="ptr.indicatorStyle.value" />
        </div>
        <button
          v-for="conv in filteredConvs"
          :key="conv.id"
          class="w-full flex items-center gap-3 px-3 py-3 rounded-xl text-left
                 transition-all group relative"
          :style="chatStore.currentConvId === conv.id
            ? 'background:rgba(51,144,236,0.12);border:1px solid rgba(51,144,236,0.2)'
            : 'border:1px solid transparent'"
          :class="chatStore.currentConvId !== conv.id && 'hover:bg-ink/[0.04]'"
          @click="selectConvAndShow(conv.id)"
        >
          <!-- Avatar -->
          <div class="relative shrink-0">
            <div class="w-11 h-11 rounded-full overflow-hidden shrink-0">
              <img v-if="convAvatar(conv)" :src="convAvatar(conv)" class="w-full h-full object-cover" />
              <div v-else class="w-full h-full flex items-center justify-center text-sm font-bold"
                   :style="`background:${conv.type === 2 ? 'linear-gradient(135deg,#10B981,#059669)' : 'linear-gradient(135deg,#3390EC,#2980DE)'}`">
                {{ convName(conv)?.[0]?.toUpperCase() || '?' }}
              </div>
            </div>
            <!-- Online dot (private only) -->
            <span v-if="conv.type === 1 && conv.target_user?.is_online"
                  class="absolute bottom-0 right-0 w-3 h-3 rounded-full border-2"
                  style="background:#10B981;border-color:rgb(var(--surface))" />
            <!-- Group badge -->
            <span v-else-if="conv.type === 2"
                  class="absolute -bottom-0.5 -right-0.5 w-4 h-4 rounded-full flex items-center justify-center"
                  style="background:linear-gradient(135deg,#10B981,#059669);color:#fff;border:1.5px solid rgb(var(--surface))">
              <Users :size="8" class="text-ink" />
            </span>
          </div>

          <!-- Info -->
          <div class="flex-1 min-w-0">
            <div class="flex items-center justify-between">
              <span class="flex items-center gap-1 min-w-0">
                <span class="text-sm font-semibold text-ink/90 truncate">{{ convName(conv) }}</span>
                <VipBadge v-if="conv.target_user?.is_vip" dense icon-only />
              </span>
              <span class="text-[10px] text-ink/30 shrink-0 ml-2">
                {{ conv.last_msg ? formatTime(conv.last_msg.created_at) : '' }}
              </span>
            </div>
            <p class="text-xs text-ink/40 truncate mt-0.5">
              <span v-if="conv.last_msg?.is_recalled" class="italic">消息已撤回</span>
              <span v-else-if="conv.last_msg?.msg_type === 2">📷 {{ conv.last_msg?.content?.text || '图片' }}</span>
              <span v-else>{{ conv.last_msg?.content?.text || '' }}</span>
            </p>
          </div>

          <!-- Unread badge -->
          <span v-if="conv.unread_count > 0"
                class="shrink-0 min-w-[18px] h-[18px] px-1 rounded-full flex items-center justify-center
                       text-[10px] font-bold text-ink"
                style="background:#EF4444">
            {{ conv.unread_count > 99 ? '99+' : conv.unread_count }}
          </span>
        </button>

        <div v-if="!chatStore.conversations.length"
             class="text-center py-12 text-ink/30 text-sm">
          <MessageSquare :size="40" class="mx-auto mb-3 opacity-30" />
          <p>还没有会话</p>
          <p class="text-xs mt-1">添加好友后开始聊天</p>
        </div>
      </div>
    </aside>

    <!-- Message Panel:
         Desktop: flex-1 beside the list
         Mobile:  full screen, shown only when a conv is selected -->
    <main class="flex-1 flex flex-col overflow-hidden"
          :class="!mobileShowList ? 'flex' : 'hidden md:flex'">
      <template v-if="chatStore.currentConvId">
        <!-- Panel Header -->
        <div class="flex items-center justify-between px-4 md:px-6 py-4 shrink-0"
             style="border-bottom:1px solid rgb(var(--ink) / 0.06)">
          <div class="flex items-center gap-3">
            <!-- Mobile back button -->
            <button class="md:hidden w-8 h-8 rounded-lg flex items-center justify-center
                           text-ink/50 hover:text-ink/80 transition-colors mr-1"
                    @click="backToList">
              <ArrowLeft :size="18" />
            </button>
            <div class="w-9 h-9 rounded-full overflow-hidden cursor-pointer hover:opacity-90 transition-opacity"
                 @click="onHeaderAvatarClick">
              <img v-if="convAvatar(chatStore.currentConv)"
                   :src="convAvatar(chatStore.currentConv)" class="w-full h-full object-cover" />
              <div v-else class="w-full h-full flex items-center justify-center text-sm font-bold"
                   style="background:linear-gradient(135deg,#3390EC,#2980DE);color:#fff">
                {{ convName(chatStore.currentConv)?.[0]?.toUpperCase() }}
              </div>
            </div>
            <div>
              <h3 class="flex items-center gap-1.5 text-sm font-semibold text-ink">
                {{ convName(chatStore.currentConv) }}
                <span v-if="chatStore.currentConv?.type === 2" class="text-ink/40 font-normal">
                  ({{ chatStore.currentConv?.group_info?.members ?? 0 }})
                </span>
                <VipBadge v-if="chatStore.currentConv?.target_user?.is_vip" dense />
              </h3>
              <p v-if="chatStore.currentConv?.type === 1" class="text-xs"
                 :class="chatStore.currentConv?.target_user?.is_online ? 'text-accent' : 'text-ink/30'">
                {{ chatStore.currentConv?.target_user?.is_online ? '在线' : '离线' }}
              </p>
            </div>
          </div>
          <div class="flex items-center gap-2">
            <button v-if="isVIP" @click="summarizeChat"
                    class="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-medium transition-all"
                    style="background:rgba(51,144,236,0.1);border:1px solid rgba(51,144,236,0.25);color:#3390EC"
                    :class="summarizing && 'opacity-50 cursor-wait'">
              <Sparkles :size="12" />
              {{ summarizing ? 'AI 总结中…' : 'AI 总结' }}
            </button>
            <button v-if="chatStore.currentConv?.type === 2"
                    @click="showGroupSettings = true"
                    class="w-8 h-8 rounded-lg flex items-center justify-center text-ink/30 hover:text-ink/70 hover:bg-ink/5 transition-all">
              <Settings :size="16" />
            </button>
            <div v-else class="relative">
              <button @click="showPrivateMenu = !showPrivateMenu; deletingFriend = false"
                      class="w-8 h-8 rounded-lg flex items-center justify-center text-ink/30 hover:text-ink/70 hover:bg-ink/5 transition-all">
                <MoreHorizontal :size="16" />
              </button>
              <div v-if="showPrivateMenu"
                   v-click-outside-chat="() => { showPrivateMenu = false; deletingFriend = false }"
                   class="absolute right-0 top-9 glass-strong rounded-xl py-1 z-10 min-w-36 animate-scale-in">
                <template v-if="!deletingFriend">
                  <button @click="() => { openUserProfile(chatStore.currentConv?.target_user?.user_id); showPrivateMenu = false }"
                          class="w-full px-4 py-2 text-sm text-ink/70 hover:bg-ink/[0.06] transition-all text-left">
                    查看资料
                  </button>
                  <button @click="deletingFriend = true"
                          class="w-full px-4 py-2 text-sm text-red-400 hover:bg-red-500/10 transition-all text-left">
                    删除好友
                  </button>
                </template>
                <template v-else>
                  <p class="px-4 pt-2 pb-1 text-xs text-ink/40">确定删除该好友？</p>
                  <button @click="doDeleteFriend"
                          class="w-full px-4 py-2 text-sm text-red-400 hover:bg-red-500/10 transition-all text-left font-medium">
                    确认删除
                  </button>
                  <button @click="deletingFriend = false"
                          class="w-full px-4 py-2 text-sm text-ink/50 hover:bg-ink/[0.06] transition-all text-left">
                    取消
                  </button>
                </template>
              </div>
            </div>
          </div>
        </div>

        <!-- Messages -->
        <div ref="msgContainer" class="flex-1 overflow-y-auto px-3 md:px-6 py-4 flex flex-col-reverse gap-1">
          <template v-for="msg in chatStore.currentMessages" :key="msg.msg_id">
            <MessageItem :msg="msg" :is-self="msg.sender_id === authStore.user?.id"
                         @resend="resendMsg" @recall="recallMsg" />
          </template>

          <!-- sentinel: DOM 末位 = flex-col-reverse 视觉顶部，滚到这里触发加载历史 -->
          <div ref="topSentinel" style="overflow-anchor:none"
               class="flex justify-center items-center py-2 shrink-0">
            <Loader2 v-if="chatStore.loadingMessages"
                     :size="16" class="animate-spin text-ink/30" />
            <span v-else-if="chatStore.messageHasMore[chatStore.currentConvId] === false
                             && chatStore.currentMessages.length"
                  class="text-[11px] text-ink/20">— 已到最早的消息 —</span>
          </div>
        </div>

        <!-- AI Summary result -->
        <div v-if="summaryResult"
             class="mx-6 mb-2 p-3 rounded-xl text-xs text-ink/70 animate-slide-up"
             style="background:rgba(51,144,236,0.08);border:1px solid rgba(51,144,236,0.2)">
          <div class="flex items-center gap-2 mb-2">
            <Sparkles :size="12" class="text-primary-light" />
            <span class="text-primary-light font-semibold">AI 会话总结</span>
            <button @click="summaryResult = ''" class="ml-auto text-ink/30 hover:text-ink/60">
              <X :size="12" />
            </button>
          </div>
          <p class="leading-relaxed">{{ summaryResult }}</p>
        </div>

        <!-- Input -->
        <MessageInput @send="sendMsg" />
      </template>

      <!-- Empty state -->
      <div v-else class="flex-1 flex flex-col items-center justify-center text-center px-8">
        <div class="w-20 h-20 rounded-2xl flex items-center justify-center mb-6"
             style="background:rgba(51,144,236,0.1);border:1px solid rgba(51,144,236,0.2)">
          <MessageSquare :size="36" class="text-primary/60" />
        </div>
        <h3 class="text-lg font-semibold text-ink/60 mb-2">选择一个会话开始聊天</h3>
        <p class="text-sm text-ink/30">或者搜索好友发起新对话</p>
      </div>
    </main>

    <!-- New Chat Modal -->
    <NewChatModal v-if="showNewChat" @close="showNewChat = false" />

    <!-- Group Settings Modal -->
    <GroupSettingsModal
      v-if="showGroupSettings && chatStore.currentConv"
      :conv="chatStore.currentConv"
      @close="showGroupSettings = false"
      @updated="chatStore.loadConversations()" />
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted, watch, nextTick } from 'vue'
import { MessageSquare, Search, SquarePen, MoreHorizontal, Sparkles, Loader2, X, Settings, ArrowLeft, Users, RefreshCw } from 'lucide-vue-next'
import { useChatStore }   from '@/stores/chat'
import { useAuthStore }   from '@/stores/auth'
import { useWsStore }     from '@/stores/ws'
import { useToast }       from '@/composables/useToast'
import { useFriendStore } from '@/stores/friend'
import { usePullToRefresh } from '@/composables/usePullToRefresh'
import * as aiApi       from '@/api/ai'
import * as chatApi     from '@/api/chat'
import MessageItem      from '@/components/chat/MessageItem.vue'
import MessageInput     from '@/components/chat/MessageInput.vue'
import NewChatModal       from '@/components/chat/NewChatModal.vue'
import GroupSettingsModal from '@/components/chat/GroupSettingsModal.vue'
import VipBadge           from '@/components/ui/VipBadge.vue'
import { useUserProfile } from '@/composables/useUserProfile'
import { formatChatTime as formatTime } from '@/utils/time'

const { openUserProfile } = useUserProfile()

const chatStore    = useChatStore()
const authStore    = useAuthStore()
const wsStore      = useWsStore()
const toast        = useToast()
const isVIP        = computed(() => authStore.user?.vip_level > 0)   // AI 总结仅 VIP
const friendStore  = useFriendStore()

const search            = ref('')
const showNewChat       = ref(false)
const showGroupSettings = ref(false)
const showPrivateMenu   = ref(false)
const deletingFriend    = ref(false)
const msgContainer      = ref(null)
const topSentinel       = ref(null)
const summarizing       = ref(false)
const summaryResult     = ref('')
const refreshing        = ref(false)
const convListScroll    = ref(null)
const ptr = usePullToRefresh(() => convListScroll.value, refreshChat)

let historyObserver = null

async function loadMoreHistory() {
  const convId = chatStore.currentConvId
  if (!convId || chatStore.loadingMessages || chatStore.messageHasMore[convId] === false) return
  const msgs = chatStore.messages[convId]
  if (!msgs?.length) return
  await chatStore.loadMessages(convId, msgs[msgs.length - 1].msg_id)
  // 加载完后若 sentinel 仍在视口内则继续加载（内容不足一屏时）
  await nextTick()
  setupHistoryObserver()
}

function setupHistoryObserver() {
  historyObserver?.disconnect()
  historyObserver = null
  if (!msgContainer.value || !topSentinel.value) return
  historyObserver = new IntersectionObserver(
    ([entry]) => { if (entry.isIntersecting) loadMoreHistory() },
    { root: msgContainer.value }
  )
  historyObserver.observe(topSentinel.value)
}

watch(() => chatStore.currentConvId, async () => {
  await nextTick()
  setupHistoryObserver()
})

async function doDeleteFriend() {
  const userId = chatStore.currentConv?.target_user?.user_id
  if (!userId) return
  try {
    await friendStore.removeFriend(userId)
    showPrivateMenu.value = false
    deletingFriend.value  = false
    toast.success('已删除好友')
    chatStore.removeConversation(chatStore.currentConvId)
  } catch {
    toast.error('删除失败，请重试')
  }
}

function onHeaderAvatarClick() {
  const conv = chatStore.currentConv
  if (!conv) return
  if (conv.type === 1 && conv.target_user) {
    openUserProfile(conv.target_user.user_id)
  } else if (conv.type === 2) {
    showGroupSettings.value = true
  }
}

const vClickOutsideChat = {
  mounted(el, binding) {
    el._fn = (e) => { if (!el.contains(e.target)) binding.value(e) }
    document.addEventListener('click', el._fn, true)
  },
  unmounted(el) { document.removeEventListener('click', el._fn, true) },
}

async function refreshChat() {
  if (refreshing.value) return
  refreshing.value = true
  try {
    await chatStore.loadConversations()
    // 当前会话已打开时，重新拉取最新消息
    if (chatStore.currentConvId) {
      await chatStore.loadMessages(chatStore.currentConvId)
    }
  } catch {
    toast.error('刷新失败')
  } finally {
    refreshing.value = false
  }
}

// Mobile panel state: true = show list, false = show chat
const isMobile       = ref(window.innerWidth < 768)
const mobileShowList = ref(true) // mobile always starts on conversation list

function onResize() {
  isMobile.value = window.innerWidth < 768
  if (!isMobile.value) mobileShowList.value = true // desktop: always show both
}

function selectConvAndShow(id) {
  chatStore.selectConv(id)
  if (isMobile.value) mobileShowList.value = false
}

function backToList() {
  mobileShowList.value = true
}

onMounted(() => {
  window.addEventListener('resize', onResize)
  ptr.attach()
  wsStore.on('ai_result', onAIResult)
  nextTick().then(setupHistoryObserver)
})
onUnmounted(() => {
  window.removeEventListener('resize', onResize)
  ptr.detach()
  historyObserver?.disconnect()
})

const filteredConvs = computed(() => {
  if (!search.value) return chatStore.conversations
  const q = search.value.toLowerCase()
  return chatStore.conversations.filter(c =>
    convName(c)?.toLowerCase().includes(q)
  )
})

function convName(conv) {
  if (!conv) return ''
  if (conv.type === 1) return conv.target_user?.nickname || '未知用户'
  return conv.group_info?.name || '群聊'
}

function convAvatar(conv) {
  if (!conv) return ''
  if (conv.type === 1) return conv.target_user?.avatar_url || ''
  return conv.group_info?.avatar_url || ''
}


function doSend(convId, msgType, content) {
  const sent = wsStore.sendMessage(convId, msgType, content)
  chatStore.pushMessage({
    msg_id:          String(Date.now()),
    conversation_id: convId,
    sender_id:       authStore.user?.id,
    sender_nickname: authStore.user?.nickname,
    sender_avatar:   authStore.user?.avatar_url,
    msg_type:        msgType,
    content,
    is_recalled:     false,
    created_at:      new Date().toISOString(),
    _pending:        sent,
    _failed:         !sent,
  })
}

function sendMsg({ text, _type, url }) {
  if (!chatStore.currentConvId) return
  const msgType = _type || 1
  const content = msgType === 2 ? { url, text } : { text }
  doSend(chatStore.currentConvId, msgType, content)
}

function resendMsg(msg) {
  chatStore.removeMessage(chatStore.currentConvId, msg.msg_id)
  doSend(chatStore.currentConvId, msg.msg_type, msg.content)
}

async function recallMsg(msg) {
  const convId = chatStore.currentConvId
  try {
    await chatApi.recallMessage(msg.msg_id)
    // 乐观更新：后端也会通过 msg_recall WS 事件广播给所有成员（含自己），此处先就地置为已撤回
    chatStore.recallMessage(convId, msg.msg_id)
  } catch (e) {
    toast.error(typeof e === 'string' ? e : '撤回失败')
  }
}

async function summarizeChat() {
  if (summarizing.value || !chatStore.currentConvId) return
  summarizing.value = true
  summaryResult.value = ''
  try {
    const res = await aiApi.summarize(chatStore.currentConvId, 50)
    if (res.data?.async) {
      // 异步：总结任务已入队，结果稍后通过 WebSocket 的 ai_result 事件返回，保持 loading 等待
      return
    }
    // 同步降级（MQ 不可用）：直接拿到结果
    summaryResult.value = res.data.result
    summarizing.value = false
  } catch (e) {
    toast.error(typeof e === 'string' ? e : 'AI 总结失败')
    summarizing.value = false
  }
}

// 异步 AI 总结结果通过 WS ai_result 事件回传
function onAIResult(data) {
  if (data.kind && data.kind !== 'summarize') return
  summarizing.value = false
  if (data.error) {
    toast.error(data.error || 'AI 总结失败')
    return
  }
  // 用户可能已切换会话；仅展示属于当前会话的结果
  if (data.conversation_id && String(data.conversation_id) !== String(chatStore.currentConvId)) return
  summaryResult.value = data.result
}

</script>
