<template>
  <div class="h-full overflow-y-auto">
    <div class="max-w-xl mx-auto px-6 py-8">
      <!-- Hero card：色带 + 重叠头像 + 资料 -->
      <div class="glass rounded-2xl overflow-hidden mb-4">
        <div class="h-20" :style="user?.vip_level > 0
              ? 'background:linear-gradient(120deg,#F0883E,#F6B24E)'
              : 'background:linear-gradient(120deg,#3390EC,#54A9F0)'" />
        <div class="px-5 pb-5 -mt-11">
          <div class="relative inline-block">
            <div class="rounded-full overflow-hidden ring-4"
                 style="width:88px;height:88px;--tw-ring-color:rgb(var(--surface))">
              <img v-if="user?.avatar_url" :src="user.avatar_url" class="w-full h-full object-cover" />
              <div v-else class="w-full h-full flex items-center justify-center text-3xl font-bold"
                   style="background:linear-gradient(135deg,#3390EC,#2980DE);color:#fff">
                {{ user?.nickname?.[0]?.toUpperCase() }}
              </div>
            </div>
            <label class="absolute bottom-0.5 right-0.5 w-8 h-8 rounded-full flex items-center justify-center cursor-pointer
                          ring-2 transition-all hover:scale-110"
                   style="background:linear-gradient(135deg,#3390EC,#2980DE);--tw-ring-color:rgb(var(--surface))">
              <Camera :size="14" class="text-white" />
              <input type="file" accept="image/*" class="hidden" @change="uploadAvatar" />
            </label>
          </div>

          <div class="flex items-center gap-2 mt-3">
            <h2 class="text-xl font-extrabold truncate"
                :style="user?.vip_level > 0
                  ? 'background:linear-gradient(135deg,#F0A500,#FBC54E);-webkit-background-clip:text;background-clip:text;-webkit-text-fill-color:transparent'
                  : 'color:#3390EC'">{{ user?.nickname }}</h2>
            <LevelBadge v-if="levelInfo" :level="levelInfo.level" :tier="levelInfo.tier" />
            <span v-if="user?.vip_level > 0"
                  class="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-semibold shrink-0"
                  style="background:linear-gradient(135deg,#F59E0B,#FBBF24);color:#78350F">
              <Crown :size="10" /> VIP
            </span>
          </div>

          <div class="flex items-center gap-2 mt-1.5 text-xs text-ink/40">
            <span>MeChatID: {{ user?.id }}</span>
            <button @click="copyId" class="hover:text-primary-light transition-colors" title="复制 MeChatID">
              <Copy :size="12" />
            </button>
            <span class="text-ink/20">·</span>
            <span class="truncate">{{ user?.email }}</span>
          </div>

          <p class="text-sm text-ink/65 mt-2 leading-relaxed">{{ user?.bio || '还没有简介，点「编辑资料」介绍一下自己吧' }}</p>
        </div>
      </div>

      <!-- 数据概览 -->
      <div class="grid grid-cols-3 gap-3 mb-4">
        <div class="glass rounded-2xl py-3.5 text-center">
          <p class="text-xl font-extrabold text-ink leading-none">{{ levelInfo?.level ?? '–' }}</p>
          <p class="text-[11px] text-ink/40 mt-1.5">等级</p>
        </div>
        <div class="glass rounded-2xl py-3.5 text-center">
          <p class="text-xl font-extrabold text-ink leading-none">{{ levelInfo?.experience ?? '–' }}</p>
          <p class="text-[11px] text-ink/40 mt-1.5">经验</p>
        </div>
        <div class="glass rounded-2xl py-3.5 text-center">
          <p class="text-xl font-extrabold text-ink leading-none">{{ friendCount }}</p>
          <p class="text-[11px] text-ink/40 mt-1.5">好友</p>
        </div>
      </div>

      <!-- 等级进度 + 每日签到 -->
      <div v-if="levelInfo" class="glass rounded-2xl p-4 mb-4">
        <div class="flex items-center justify-between mb-2">
          <span class="text-sm font-semibold" :class="tierTextClass">{{ tierLabel }} · Lv.{{ levelInfo.level }}</span>
          <span class="text-xs text-ink/45">
            <template v-if="levelInfo.next_level_xp > 0">还需 {{ levelInfo.next_level_xp }} XP 升级</template>
            <template v-else>已达最高等级 🎉</template>
          </span>
        </div>
        <div class="h-2.5 rounded-full overflow-hidden" style="background:rgb(var(--ink) / 0.08)">
          <div class="h-full rounded-full transition-all duration-700"
               :style="`width:${progressPct}%;background:${tierColor}`" />
        </div>

        <button v-if="checkinState !== 'hidden'"
                @click="doCheckin" :disabled="checkinState !== 'available'"
                class="w-full flex items-center justify-center gap-1.5 mt-3.5 py-2.5 rounded-xl text-sm font-semibold transition-all"
                :style="checkinState === 'done'
                  ? 'background:rgba(60,188,93,0.12);color:#34A853;border:1px solid rgba(60,188,93,0.3)'
                  : 'background:rgba(51,144,236,0.12);color:#3390EC;border:1px solid rgba(51,144,236,0.3)'">
          <Loader2 v-if="checkinState === 'loading'" :size="15" class="animate-spin" />
          <CalendarCheck v-else-if="checkinState === 'done'" :size="15" />
          <CalendarDays  v-else :size="15" />
          {{ checkinState === 'done' ? '今日已签到' : '每日签到 · 领取经验' }}
        </button>
      </div>

      <!-- Menu items -->
      <div class="glass rounded-2xl overflow-hidden mb-4">
        <!-- Edit profile -->
        <button @click="openEdit"
                class="w-full flex items-center gap-4 px-5 py-4 transition-all hover:bg-ink/5"
                style="border-bottom:1px solid rgb(var(--ink) / 0.06)">
          <div class="w-9 h-9 rounded-xl flex items-center justify-center shrink-0"
               style="background:rgba(51,144,236,0.15)">
            <Pencil :size="16" class="text-primary-light" />
          </div>
          <div class="flex-1 text-left">
            <p class="text-sm font-medium text-ink/80">编辑资料</p>
            <p class="text-xs text-ink/40">修改昵称和简介</p>
          </div>
          <ChevronRight :size="16" class="text-ink/30" />
        </button>

        <!-- VIP -->
        <RouterLink to="/vip"
                    class="flex items-center gap-4 px-5 py-4 transition-all hover:bg-ink/5"
                    style="border-bottom:1px solid rgb(var(--ink) / 0.06)">
          <div class="w-9 h-9 rounded-xl flex items-center justify-center shrink-0"
               style="background:rgba(245,158,11,0.15)">
            <Crown :size="16" class="text-yellow-400" />
          </div>
          <div class="flex-1">
            <p class="text-sm font-medium text-ink/80">VIP 会员</p>
            <p class="text-xs text-ink/40">{{ user?.vip_level > 0 ? '已开通' : '免费版' }}</p>
          </div>
          <ChevronRight :size="16" class="text-ink/30" />
        </RouterLink>

        <!-- My posts (打开本人资料浮层，内含帖子列表) -->
        <button @click="user && openUserProfile(user.id)"
                class="w-full flex items-center gap-4 px-5 py-4 transition-all hover:bg-ink/5 text-left"
                style="border-bottom:1px solid rgb(var(--ink) / 0.06)">
          <div class="w-9 h-9 rounded-xl flex items-center justify-center shrink-0"
               style="background:rgba(51,144,236,0.15)">
            <FileText :size="16" class="text-primary-light" />
          </div>
          <div class="flex-1">
            <p class="text-sm font-medium text-ink/80">我的帖子</p>
            <p class="text-xs text-ink/40">查看我发布的全部动态</p>
          </div>
          <ChevronRight :size="16" class="text-ink/30" />
        </button>

        <!-- Theme toggle -->
        <button @click="toggleTheme"
                class="w-full flex items-center gap-4 px-5 py-4 transition-all hover:bg-ink/5">
          <div class="w-9 h-9 rounded-xl flex items-center justify-center shrink-0"
               :style="isDark
                 ? 'background:rgba(51,144,236,0.15)'
                 : 'background:rgba(245,158,11,0.15)'">
            <Moon v-if="isDark" :size="16" class="text-primary-light" />
            <Sun  v-else        :size="16" class="text-yellow-400" />
          </div>
          <div class="flex-1 text-left">
            <p class="text-sm font-medium text-ink/80">{{ isDark ? '深色模式' : '浅色模式' }}</p>
            <p class="text-xs text-ink/40">点击切换外观</p>
          </div>
          <div class="w-9 h-5 rounded-full transition-all relative"
               :style="`background:${isDark ? 'rgba(51,144,236,0.4)' : 'rgba(245,158,11,0.5)'}`">
            <div class="absolute top-0.5 w-4 h-4 rounded-full bg-white shadow transition-all"
                 :style="{ left: isDark ? '2px' : '18px' }" />
          </div>
        </button>
      </div>

      <!-- Logout -->
      <button @click="confirmLogout"
              class="w-full flex items-center justify-center gap-2 py-3 rounded-xl text-sm font-medium
                     transition-all text-danger/70 hover:text-danger hover:bg-danger/10"
              style="border:1px solid rgba(239,68,68,0.15)">
        <LogOut :size="16" />
        退出登录
      </button>
    </div>

    <!-- Edit profile modal -->
    <Teleport to="body">
      <div v-if="showEdit"
           class="fixed inset-0 z-50 flex items-center justify-center p-4"
           style="background:rgba(0,0,0,0.6);backdrop-filter:blur(4px)"
           @click.self="showEdit = false">
        <div class="glass-strong rounded-2xl p-6 max-w-sm w-full animate-scale-in">
          <div class="flex items-center justify-between mb-5">
            <h3 class="font-semibold text-ink">编辑资料</h3>
            <button @click="showEdit = false" class="text-ink/40 hover:text-ink/70 transition-colors">
              <X :size="18" />
            </button>
          </div>
          <div class="space-y-4">
            <div>
              <label class="block text-xs font-medium text-ink/40 mb-2">昵称</label>
              <input v-model="form.nickname" type="text" class="mc-input" maxlength="20" />
            </div>
            <div>
              <label class="block text-xs font-medium text-ink/40 mb-2">简介</label>
              <textarea v-model="form.bio" rows="3" class="mc-input resize-none" maxlength="200"
                        placeholder="介绍一下自己…" />
              <p class="text-xs text-ink/25 mt-1 text-right">{{ form.bio.length }}/200</p>
            </div>
            <div class="flex gap-3 pt-1">
              <button @click="showEdit = false" class="btn-ghost flex-1 py-2.5">取消</button>
              <button @click="saveProfile" :disabled="saving" class="btn-primary flex-1 py-2.5 text-sm">
                <Loader2 v-if="saving" :size="14" class="inline animate-spin mr-1" />
                保存
              </button>
            </div>
          </div>
        </div>
      </div>
    </Teleport>

    <!-- Logout confirm -->
    <Teleport to="body">
      <div v-if="showLogoutConfirm"
           class="fixed inset-0 z-50 flex items-center justify-center p-4"
           style="background:rgba(0,0,0,0.6);backdrop-filter:blur(4px)"
           @click.self="showLogoutConfirm = false">
        <div class="glass-strong rounded-2xl p-6 max-w-sm w-full animate-scale-in text-center">
          <LogOut :size="32" class="text-danger/60 mx-auto mb-4" />
          <h3 class="font-semibold text-ink mb-2">确认退出？</h3>
          <p class="text-sm text-ink/40 mb-6">退出后需要重新登录</p>
          <div class="flex gap-3">
            <button @click="showLogoutConfirm = false" class="btn-ghost flex-1 py-2.5">取消</button>
            <button @click="doLogout"
                    class="flex-1 py-2.5 rounded-md text-sm font-semibold text-ink transition-all"
                    style="background:rgba(239,68,68,0.2);border:1px solid rgba(239,68,68,0.3)">
              退出
            </button>
          </div>
        </div>
      </div>
    </Teleport>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { RouterLink, useRouter } from 'vue-router'
import { Camera, Crown, ChevronRight, LogOut, Loader2, FileText, Pencil, Moon, Sun, X, CalendarDays, CalendarCheck, Copy } from 'lucide-vue-next'
import { useAuthStore }  from '@/stores/auth'
import { useWsStore }    from '@/stores/ws'
import { useFriendStore } from '@/stores/friend'
import { useToast }      from '@/composables/useToast'
import { useTheme }      from '@/composables/useTheme'
import { useXPNotify }   from '@/composables/useXPNotify'
import { useUserProfile } from '@/composables/useUserProfile'
import { copyText }      from '@/utils/clipboard'
import * as authApi  from '@/api/auth'
import * as levelApi from '@/api/level'
import LevelBadge from '@/components/ui/LevelBadge.vue'

const router = useRouter()
const auth   = useAuthStore()
const ws     = useWsStore()
const friendStore = useFriendStore()
const toast  = useToast()
const { isDark, toggleTheme } = useTheme()
const { showXP } = useXPNotify()
const { openUserProfile } = useUserProfile()

const user  = computed(() => auth.user)
const friendCount = computed(() => friendStore.friends.length)
const saving = ref(false)
const showLogoutConfirm = ref(false)
const showEdit = ref(false)
const levelInfo = ref(null)
const checkinState = ref('hidden')   // 'hidden' | 'available' | 'done' | 'loading'

const TIER_COLORS  = { gray: '#9CA3AF', blue: '#60A5FA', yellow: '#EAB308', orange: '#F97316' }
const TIER_LABELS  = { gray: '灰牌', blue: '蓝牌', yellow: '黄牌', orange: '橙牌' }
const TIER_CLASSES = { gray: 'text-gray-400', blue: 'text-blue-400', yellow: 'text-yellow-400', orange: 'text-orange-400' }

const tierColor = computed(() => TIER_COLORS[levelInfo.value?.tier] || '#9CA3AF')
const tierLabel = computed(() => TIER_LABELS[levelInfo.value?.tier] || '')
const tierTextClass = computed(() => TIER_CLASSES[levelInfo.value?.tier] || 'text-gray-400')
const progressPct = computed(() => {
  if (!levelInfo.value) return 0
  const { current_level_xp, next_level_xp, level } = levelInfo.value
  if (level >= 10) return 100
  const total = current_level_xp + next_level_xp
  return total > 0 ? Math.round((current_level_xp / total) * 100) : 0
})

const form = ref({ nickname: '', bio: '' })

async function copyId() {
  const ok = await copyText(String(user.value?.id ?? ''))
  if (ok) toast.success('MeChatID 已复制')
}

onMounted(async () => {
  friendStore.loadFriends()   // 拉好友数用于「数据概览」
  try {
    const res = await levelApi.getMyLevel()
    levelInfo.value = res.data
    checkinState.value = res.data.today_checkin ? 'done' : 'available'
  } catch {}
})

async function doCheckin() {
  if (checkinState.value !== 'available') return
  checkinState.value = 'loading'
  try {
    const res = await levelApi.checkin()
    checkinState.value = 'done'
    if (res.data.xp_gained > 0) { showXP(res.data.xp_gained); toast.success('签到成功！') }
    // 签到后刷新经验条
    try { levelInfo.value = (await levelApi.getMyLevel()).data } catch {}
  } catch { checkinState.value = 'available' }
}

function openEdit() {
  form.value.nickname = user.value?.nickname || ''
  form.value.bio      = user.value?.bio || ''
  showEdit.value = true
}

async function saveProfile() {
  saving.value = true
  try {
    const res = await authApi.updateMe(form.value)
    auth.updateUser(res.data)
    toast.success('资料已更新')
    showEdit.value = false
  } catch (e) {
    toast.error(typeof e === 'string' ? e : '保存失败')
  } finally {
    saving.value = false
  }
}

async function uploadAvatar(e) {
  const file = e.target.files?.[0]
  if (!file) return
  const fd = new FormData()
  fd.append('file', file)
  try {
    const res = await fetch('/api/v1/users/me/avatar', {
      method: 'PUT',
      headers: { Authorization: `Bearer ${auth.token}` },
      body: fd,
    })
    const data = await res.json()
    if (data.code === 0) {
      auth.updateUser({ avatar_url: data.data.url })
      toast.success('头像已更新')
    }
  } catch {
    toast.error('上传失败')
  }
}

function confirmLogout() { showLogoutConfirm.value = true }

async function doLogout() {
  await auth.logoutAction()
  ws.disconnect()
  router.push('/auth')
}
</script>
