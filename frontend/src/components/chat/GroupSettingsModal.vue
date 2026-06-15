<template>
  <Teleport to="body">
    <div class="fixed inset-0 z-50 flex items-center justify-center p-4"
         style="background:rgba(0,0,0,0.6);backdrop-filter:blur(4px)"
         @click.self="$emit('close')">
      <div class="w-full max-w-sm glass-strong rounded-2xl overflow-hidden animate-scale-in">
        <!-- Header -->
        <div class="flex items-center justify-between px-5 py-4"
             style="border-bottom:1px solid rgb(var(--ink) / 0.06)">
          <h3 class="font-semibold text-ink">群聊设置</h3>
          <button @click="$emit('close')" class="text-ink/40 hover:text-ink/70 transition-colors">
            <X :size="18" />
          </button>
        </div>

        <div class="px-5 py-4 space-y-5 max-h-[70vh] overflow-y-auto">
          <!-- Group avatar -->
          <div class="flex flex-col items-center gap-3">
            <div class="relative group cursor-pointer" @click="isOwner && avatarInput?.click()">
              <div class="w-20 h-20 rounded-2xl overflow-hidden"
                   style="border:2px solid rgba(51,144,236,0.3)">
                <img v-if="avatarUrl" :src="avatarUrl" class="w-full h-full object-cover" />
                <div v-else class="w-full h-full flex items-center justify-center text-2xl font-bold"
                     style="background:linear-gradient(135deg,#3390EC,#2980DE);color:#fff">
                  {{ conv.group_info?.name?.[0]?.toUpperCase() }}
                </div>
              </div>
              <div v-if="isOwner"
                   class="absolute inset-0 rounded-2xl flex items-center justify-center opacity-0 group-hover:opacity-100 transition-opacity"
                   style="background:rgba(0,0,0,0.5)">
                <Camera v-if="!uploadingAvatar" :size="20" class="text-ink" />
                <Loader2 v-else :size="20" class="text-ink animate-spin" />
              </div>
            </div>
            <input ref="avatarInput" type="file" accept="image/*" class="hidden" @change="uploadAvatar" />
            <p class="text-sm font-semibold text-ink">{{ conv.group_info?.name }}</p>
            <p class="text-xs text-ink/40">{{ members.length }} 名成员</p>
          </div>

          <!-- Members list -->
          <div>
            <p class="text-xs font-semibold text-ink/40 uppercase tracking-wider mb-2">成员</p>
            <div class="space-y-1">
              <div v-if="loading" class="text-center py-4">
                <Loader2 :size="16" class="animate-spin text-ink/30 mx-auto" />
              </div>
              <div v-for="m in members" :key="m.user_id"
                   class="flex items-center gap-3 px-3 py-2 rounded-xl"
                   :class="isOwner && m.user_id !== myId && 'hover:bg-ink/5 cursor-pointer group/member'"
                   @click="isOwner && m.user_id !== myId && kickMember(m)">
                <div class="relative shrink-0 cursor-pointer" title="查看资料"
                     @click.stop="openUserProfile(m.user_id)">
                  <Avatar :name="m.nickname" :url="m.avatar_url" :size="34" />
                  <span v-if="m.is_online"
                        class="absolute bottom-0 right-0 w-2.5 h-2.5 rounded-full border-2"
                        style="background:#10B981;border-color:rgb(var(--surface))" />
                </div>
                <div class="flex-1 min-w-0">
                  <p class="text-sm text-ink/85 truncate">{{ m.nickname }}</p>
                  <p class="text-[11px]" :class="m.is_online ? 'text-emerald-400' : 'text-ink/25'">
                    {{ m.is_online ? '在线' : '离线' }}
                  </p>
                </div>
                <span v-if="m.role === 1"
                      class="text-[10px] font-semibold px-1.5 py-0.5 rounded"
                      style="background:rgba(245,158,11,0.15);color:#FBBF24">群主</span>
                <Trash2 v-else-if="isOwner && m.user_id !== myId"
                        :size="14" class="text-danger/40 opacity-0 group-hover/member:opacity-100 transition-opacity shrink-0" />
              </div>
            </div>
          </div>

          <!-- Invite (owner only) -->
          <div v-if="isOwner">
            <p class="text-xs font-semibold text-ink/40 uppercase tracking-wider mb-2">邀请好友</p>
            <div class="space-y-1">
              <div v-for="f in invitableFriends" :key="f.user_id"
                   class="flex items-center gap-3 px-3 py-2 rounded-xl hover:bg-ink/5 cursor-pointer"
                   @click="inviteMember(f.user_id)">
                <Avatar :name="f.nickname" :url="f.avatar_url" :size="32" />
                <span class="flex-1 text-sm text-ink/70">{{ f.nickname }}</span>
                <UserPlus :size="14" class="text-primary-light shrink-0" />
              </div>
              <p v-if="!invitableFriends.length" class="text-xs text-ink/30 px-3 py-2">所有好友都在群里了</p>
            </div>
          </div>

          <!-- Leave group -->
          <button @click="leaveGroup"
                  class="w-full py-2.5 rounded-xl text-sm font-medium transition-all text-danger/70 hover:text-danger hover:bg-danger/10">
            {{ isOwner ? '解散群聊' : '退出群聊' }}
          </button>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { X, Camera, Loader2, Trash2, UserPlus } from 'lucide-vue-next'
import * as chatApi   from '@/api/chat'
import { useAuthStore } from '@/stores/auth'
import { useFriendStore } from '@/stores/friend'
import { useChatStore } from '@/stores/chat'
import { useToast }   from '@/composables/useToast'
import { useRouter }  from 'vue-router'
import { useUserProfile } from '@/composables/useUserProfile'
import Avatar         from '@/components/ui/Avatar.vue'

const props = defineProps({ conv: { type: Object, required: true } })
const emit  = defineEmits(['close', 'updated'])

const auth        = useAuthStore()
const friendStore = useFriendStore()
const chatStore   = useChatStore()
const toast       = useToast()
const router      = useRouter()
const { openUserProfile } = useUserProfile()

const myId          = computed(() => auth.user?.id)
const members       = ref([])
const loading       = ref(false)
const avatarUrl     = ref(props.conv.group_info?.avatar_url || '')
const avatarInput   = ref(null)
const uploadingAvatar = ref(false)

const isOwner = computed(() => members.value.find(m => m.user_id === myId.value)?.role === 1)

const memberIds = computed(() => new Set(members.value.map(m => m.user_id)))
const invitableFriends = computed(() => friendStore.friends.filter(f => !memberIds.value.has(f.user_id)))

async function loadMembers() {
  loading.value = true
  try {
    const res = await chatApi.getGroupMembers(props.conv.group_id)
    members.value = res.data.list || []
  } finally {
    loading.value = false
  }
}

async function uploadAvatar(e) {
  const file = e.target.files?.[0]
  e.target.value = ''
  if (!file) return
  uploadingAvatar.value = true
  try {
    const res = await chatApi.uploadGroupAvatar(props.conv.group_id, file)
    avatarUrl.value = res.data.url
    toast.success('群头像已更新')
    emit('updated')
  } catch (err) {
    toast.error(typeof err === 'string' ? err : '上传失败')
  } finally {
    uploadingAvatar.value = false
  }
}

async function kickMember(m) {
  if (!confirm(`确定要踢出 ${m.nickname}？`)) return
  try {
    await chatApi.removeGroupMember(props.conv.id, m.user_id)
    members.value = members.value.filter(x => x.user_id !== m.user_id)
    toast.success(`已移除 ${m.nickname}`)
  } catch (err) {
    toast.error(typeof err === 'string' ? err : '操作失败')
  }
}

async function inviteMember(uid) {
  try {
    await chatApi.addGroupMembers(props.conv.id, [uid])
    await loadMembers()
    toast.success('邀请成功')
  } catch (err) {
    toast.error(typeof err === 'string' ? err : '邀请失败')
  }
}

async function leaveGroup() {
  const action = isOwner.value ? '解散群聊' : '退出群聊'
  if (!confirm(`确定要${action}吗？`)) return
  try {
    if (isOwner.value) {
      await chatApi.disbandGroup(props.conv.id)
    } else {
      await chatApi.leaveGroup(props.conv.id)
    }
    emit('close')
    emit('updated')
    chatStore.removeConversation(props.conv.id)
    router.push('/chat')
    toast.success(action + '成功')
  } catch (err) {
    toast.error(typeof err === 'string' ? err : '操作失败')
  }
}

onMounted(async () => {
  await loadMembers()
  if (!friendStore.friends.length) friendStore.loadFriends()
})
</script>
