<template>
  <div class="flex h-screen w-full overflow-hidden app-bg">
    <!-- ── Desktop sidebar (hidden on mobile) ── -->
    <nav class="hidden md:flex flex-col items-center py-5 gap-1 z-20 shrink-0"
         style="width:72px;background:rgb(var(--ink) / 0.02);border-right:1px solid rgb(var(--ink) / 0.05)">
      <!-- Logo (clickable → home) -->
      <button @click="$router.push('/feed')"
              class="mb-5 shrink-0 select-none transition-all hover:scale-110 active:scale-95"
              title="回到首页">
        <span style="font-family:'Plus Jakarta Sans',system-ui,sans-serif;font-size:30px;font-weight:900;color:#3390EC;letter-spacing:-2px;line-height:1">M</span>
      </button>

      <!-- Nav items -->
      <NavBtn to="/feed"     :icon="Compass"       label="MeChatPost" />
      <NavBtn to="/chat"     :icon="MessageSquare" label="MeChat"   :badge="totalUnread" />
      <NavBtn to="/ai"       :icon="Sparkles"      label="MeChatAgent" />
      <NavBtn to="/contacts" :icon="Users"         label="MeChatFriends"   :badge="pendingRequests" />

      <div class="flex-1" />

      <!-- Theme toggle -->
      <button @click="toggleTheme"
              class="w-10 h-10 rounded-xl flex items-center justify-center mb-1 transition-all hover:scale-105"
              style="background:rgb(var(--ink) / 0.05);border:1px solid rgb(var(--ink) / 0.08)"
              :title="isDark ? '切换到白天模式' : '切换到黑夜模式'">
        <Moon v-if="isDark" :size="18" class="text-ink/50" />
        <Sun  v-else        :size="18" class="text-warning" />
      </button>

      <!-- VIP button -->
      <button @click="$router.push('/vip')"
              class="w-10 h-10 rounded-xl flex items-center justify-center mb-1 transition-all hover:scale-105"
              :style="isVIP
                ? 'background:linear-gradient(135deg,#F59E0B,#FBBF24);box-shadow:0 0 15px rgba(245,158,11,0.3)'
                : 'background:rgb(var(--ink) / 0.05);border:1px solid rgb(var(--ink) / 0.08)'"
              title="VIP">
        <Crown :size="18" :class="isVIP ? 'text-yellow-900' : 'text-ink/40'" />
      </button>

      <!-- Avatar -->
      <button @click="$router.push('/profile')"
              class="w-10 h-10 rounded-full overflow-hidden border-2 transition-all hover:scale-105"
              :style="$route.path === '/profile' ? 'border-color:#3390EC' : 'border-color:rgb(var(--ink) / 0.1)'">
        <img v-if="user?.avatar_url" :src="user.avatar_url" :alt="user.nickname" class="w-full h-full object-cover" />
        <div v-else class="w-full h-full flex items-center justify-center text-sm font-bold"
             style="background:linear-gradient(135deg,#3390EC,#2980DE);color:#fff">
          {{ user?.nickname?.[0]?.toUpperCase() }}
        </div>
      </button>
    </nav>

    <!-- ── Main content ── -->
    <!-- pb-16 on mobile to avoid overlap with bottom nav -->
    <div class="flex-1 overflow-hidden md:pb-0 pb-16">
      <RouterView />
    </div>

    <!-- ── Mobile bottom tab bar ── -->
    <nav class="md:hidden fixed bottom-0 inset-x-0 z-50 flex items-center justify-around px-2 pb-safe"
         style="height:64px;
                background:rgb(var(--surface) / 0.92);
                backdrop-filter:blur(16px);
                border-top:1px solid rgb(var(--border))">
      <MobileTab to="/feed"     :icon="Compass"       label="首页" />
      <MobileTab to="/chat"     :icon="MessageSquare" label="消息"   :badge="totalUnread" />
      <MobileTab to="/ai"       :icon="Sparkles"      label="AI" />
      <MobileTab to="/contacts" :icon="Users"         label="好友"   :badge="pendingRequests" />

      <!-- Profile tab (avatar) -->
      <button @click="$router.push('/profile')" title="我的"
              class="flex items-center justify-center px-4 py-2.5 transition-all"
              :class="$route.path === '/profile' ? 'opacity-100' : 'opacity-50'">
        <div class="w-7 h-7 rounded-full overflow-hidden border-2"
             :style="$route.path === '/profile' ? 'border-color:#3390EC' : 'border-color:transparent'">
          <img v-if="user?.avatar_url" :src="user.avatar_url" class="w-full h-full object-cover" />
          <div v-else class="w-full h-full flex items-center justify-center text-[10px] font-bold"
               style="background:linear-gradient(135deg,#3390EC,#2980DE);color:#fff">
            {{ user?.nickname?.[0]?.toUpperCase() }}
          </div>
        </div>
      </button>
    </nav>
  </div>
</template>

<script setup>
import { computed, onMounted } from 'vue'
import { RouterView, useRoute } from 'vue-router'
import { MessageSquare, Users, Compass, Sparkles, Crown, Moon, Sun } from 'lucide-vue-next'
import { useAuthStore }   from '@/stores/auth'
import { useWsStore }     from '@/stores/ws'
import { useChatStore }   from '@/stores/chat'
import { useFriendStore } from '@/stores/friend'
import { useTheme }       from '@/composables/useTheme'
import NavBtn    from './NavBtn.vue'
import MobileTab from './MobileTab.vue'

const { isDark, toggleTheme } = useTheme()

const auth        = useAuthStore()
const wsStore     = useWsStore()
const chatStore   = useChatStore()
const friendStore = useFriendStore()
const route       = useRoute()

const user            = computed(() => auth.user)
const totalUnread     = computed(() => chatStore.totalUnread)
const pendingRequests = computed(() => friendStore.pendingCount)
const isVIP           = computed(() => auth.user?.vip_level > 0)

onMounted(async () => {
  if (auth.token && !wsStore.connected) {
    wsStore.connect(auth.token)
  }
  wsStore.on('friend_req',    () => friendStore.loadRequests())
  wsStore.on('friend_accept', () => friendStore.loadFriends())
  await Promise.all([
    chatStore.loadConversations(),
    friendStore.loadRequests(),
  ])
})
</script>
