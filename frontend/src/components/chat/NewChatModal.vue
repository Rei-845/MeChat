<template>
  <Teleport to="body">
    <div class="fixed inset-0 z-50 flex items-center justify-center p-4"
         style="background:rgba(0,0,0,0.6);backdrop-filter:blur(4px)"
         @click.self="$emit('close')">
      <div class="w-full max-w-sm glass-strong rounded-2xl p-6 animate-scale-in">
        <div class="flex items-center justify-between mb-4">
          <h3 class="font-semibold text-ink">新建对话</h3>
          <button @click="$emit('close')" class="text-ink/40 hover:text-ink/70 transition-colors">
            <X :size="18" />
          </button>
        </div>

        <!-- Mode tabs -->
        <div class="flex gap-1 p-1 rounded-lg mb-4" style="background:rgb(var(--ink) / 0.04)">
          <button v-for="t in tabs" :key="t.key" @click="mode = t.key"
                  class="flex-1 py-1.5 text-xs font-semibold rounded-md transition-all flex items-center justify-center gap-1.5"
                  :style="mode === t.key
                    ? 'background:linear-gradient(135deg,#3390EC,#2980DE);color:#fff;color:#fff'
                    : 'color:rgb(var(--ink) / 0.4)'">
            <component :is="t.icon" :size="13" /> {{ t.label }}
          </button>
        </div>

        <!-- ── 单聊 ── -->
        <template v-if="mode === 'private'">
          <div class="relative mb-3">
            <Search :size="14" class="absolute left-3 top-1/2 -translate-y-1/2 text-ink/30" />
            <input v-model="query" @input="doSearch" placeholder="搜索用户…" type="search"
                   class="mc-input pl-9 py-2.5 text-sm" autofocus />
          </div>

          <div class="max-h-72 overflow-y-auto">
            <div v-if="loading" class="text-center py-6">
              <Loader2 :size="20" class="animate-spin text-ink/30 mx-auto" />
            </div>

            <!-- Search results -->
            <template v-else-if="query">
              <button v-for="u in results" :key="u.id"
                      @click="startChat(u.id)"
                      class="w-full flex items-center gap-3 px-3 py-2.5 rounded-xl hover:bg-ink/5 transition-all text-left">
                <Avatar :name="u.nickname" :url="u.avatar_url" :size="36" />
                <div>
                  <p class="text-sm font-medium text-ink/90">{{ u.nickname }}</p>
                  <p class="text-xs text-ink/40">{{ u.bio || '暂无简介' }}</p>
                </div>
              </button>
              <p v-if="!results.length" class="text-center py-6 text-sm text-ink/30">未找到用户</p>
            </template>

            <!-- Recommended users (when no query) -->
            <template v-else>
              <p class="text-xs font-semibold text-ink/30 px-3 pb-2">推荐用户</p>
              <div v-if="loadingRec" class="text-center py-6">
                <Loader2 :size="16" class="animate-spin text-ink/30 mx-auto" />
              </div>
              <button v-for="u in recommended" :key="u.id"
                      @click="startChat(u.id)"
                      class="w-full flex items-center gap-3 px-3 py-2.5 rounded-xl hover:bg-ink/5 transition-all text-left">
                <Avatar :name="u.nickname" :url="u.avatar_url" :size="36" />
                <div class="flex-1 min-w-0">
                  <p class="text-sm font-medium text-ink/90">{{ u.nickname }}</p>
                  <p class="text-xs text-ink/40 truncate">{{ u.bio || '暂无简介' }}</p>
                </div>
              </button>
              <p v-if="!loadingRec && !recommended.length" class="text-center py-4 text-xs text-ink/25">暂无推荐</p>
            </template>
          </div>
        </template>

        <!-- ── 群聊 ── -->
        <template v-else>
          <input v-model="groupName" placeholder="群聊名称…" type="text" maxlength="30"
                 class="mc-input py-2.5 text-sm mb-3" />

          <p class="text-xs text-ink/40 mb-2">从好友中选择成员（已选 {{ selected.size }}）</p>
          <div class="space-y-1 max-h-56 overflow-y-auto mb-4">
            <button v-for="f in friendStore.friends" :key="f.user_id"
                    @click="toggle(f.user_id)"
                    class="w-full flex items-center gap-3 px-3 py-2 rounded-xl transition-all text-left"
                    :style="selected.has(f.user_id) ? 'background:rgba(51,144,236,0.12)' : ''"
                    :class="!selected.has(f.user_id) && 'hover:bg-ink/5'">
              <Avatar :name="f.nickname" :url="f.avatar_url" :size="34" />
              <span class="flex-1 text-sm text-ink/85">{{ f.nickname }}</span>
              <span class="w-5 h-5 rounded-full flex items-center justify-center shrink-0"
                    :style="selected.has(f.user_id)
                      ? 'background:linear-gradient(135deg,#3390EC,#2980DE);color:#fff'
                      : 'border:1.5px solid rgb(var(--ink) / 0.2)'">
                <Check v-if="selected.has(f.user_id)" :size="12" class="text-ink" />
              </span>
            </button>
            <p v-if="!friendStore.friends.length" class="text-center py-6 text-sm text-ink/30">
              还没有好友，先去通讯录添加
            </p>
          </div>

          <button @click="makeGroup" :disabled="!canCreate || creating"
                  class="btn-primary w-full py-2.5 text-sm"
                  :class="(!canCreate || creating) && 'opacity-40 cursor-not-allowed'">
            <Loader2 v-if="creating" :size="14" class="inline animate-spin mr-1" />
            创建群聊{{ selected.size ? `（${selected.size}人）` : '' }}
          </button>
        </template>
      </div>
    </div>
  </Teleport>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { X, Search, Loader2, User, Users, Check } from 'lucide-vue-next'
import { searchUsers, recommendUsers } from '@/api/auth'
import { useChatStore }   from '@/stores/chat'
import { useFriendStore } from '@/stores/friend'
import { useToast }       from '@/composables/useToast'
import { useRouter }      from 'vue-router'
import Avatar             from '@/components/ui/Avatar.vue'

const emit        = defineEmits(['close'])
const chat        = useChatStore()
const friendStore = useFriendStore()
const toast       = useToast()
const router      = useRouter()

const tabs = [
  { key: 'private', label: '单聊', icon: User },
  { key: 'group',   label: '群聊', icon: Users },
]
const mode = ref('private')

// 单聊
const query       = ref('')
const results     = ref([])
const loading     = ref(false)
const recommended = ref([])
const loadingRec  = ref(false)
let   timer       = null

function doSearch() {
  clearTimeout(timer)
  if (!query.value.trim()) { results.value = []; return }
  timer = setTimeout(async () => {
    loading.value = true
    try {
      const res = await searchUsers(query.value)
      results.value = res.data.list || []
    } finally {
      loading.value = false
    }
  }, 300)
}

async function startChat(userId) {
  await chat.openOrCreatePrivateChat(userId)
  router.push('/chat')
  emit('close')
}

// 群聊
const groupName = ref('')
const selected  = ref(new Set())
const creating  = ref(false)

const canCreate = computed(() => groupName.value.trim().length >= 2 && selected.value.size >= 1)

function toggle(uid) {
  if (selected.value.has(uid)) selected.value.delete(uid)
  else selected.value.add(uid)
  selected.value = new Set(selected.value) // 触发响应式
}

async function makeGroup() {
  if (!canCreate.value || creating.value) return
  creating.value = true
  try {
    await chat.createGroup(groupName.value.trim(), [...selected.value])
    router.push('/chat')
    toast.success('群聊创建成功')
    emit('close')
  } catch (e) {
    toast.error(typeof e === 'string' ? e : '创建群聊失败')
  } finally {
    creating.value = false
  }
}

onMounted(async () => {
  if (!friendStore.friends.length) friendStore.loadFriends()
  loadingRec.value = true
  try {
    const res = await recommendUsers()
    recommended.value = res.data.list || []
  } catch {
    recommended.value = []
  } finally {
    loadingRec.value = false
  }
})
</script>
