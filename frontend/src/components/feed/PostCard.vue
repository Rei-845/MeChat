<template>
  <!-- ── Compact grid card (小红书 style) ── -->
  <article v-if="compact"
           class="glass rounded-2xl overflow-hidden cursor-pointer transition-all hover:scale-[1.01] hover:brightness-110 active:scale-[0.98]"
           style="border:1px solid rgb(var(--ink) / 0.07)"
           @click="onCardClick">
    <!-- Cover: real image (natural height) OR patterned text cover -->
    <div v-if="coverImage" class="w-full overflow-hidden">
      <img :src="coverImage" class="w-full object-cover"
           style="min-height:140px;max-height:320px" loading="lazy" />
    </div>
    <div v-else class="w-full relative flex items-center justify-center px-6"
         :style="genCoverStyle" style="min-height:220px">
      <div class="absolute inset-0" :style="patternOverlay" />
      <p class="relative text-center font-black text-white leading-tight line-clamp-4 z-10"
         style="font-size:26px;line-height:1.35;text-shadow:0 1px 12px rgba(0,0,0,0.25)">{{ post.title }}</p>
    </div>

    <div class="p-3">
      <!-- Title (only when there's a cover image; text-cover already shows it) -->
      <h3 v-if="coverImage" class="text-[14px] font-semibold text-ink/90 leading-snug line-clamp-2 mb-2">
        {{ post.title }}
      </h3>
      <p v-else-if="post.content" class="text-[12px] text-ink/50 leading-relaxed line-clamp-2 mb-2">
        {{ post.content }}
      </p>

      <!-- Footer: avatar + name + date + comment + like -->
      <div class="flex items-center justify-between gap-1 mt-1">
        <div class="flex items-center gap-1.5 min-w-0 cursor-pointer"
             @click.stop="openUserProfile(post.user.id)">
          <Avatar :name="post.user.nickname" :url="post.user.avatar_url" :size="24" />
          <span class="text-[12px] text-ink/55 truncate max-w-[5rem]">{{ post.user.nickname }}</span>
          <LevelBadge v-if="post.user.level" :level="post.user.level" :tier="post.user.tier" dense />
          <VipBadge v-if="post.user.vip" dense icon-only />
        </div>
        <div class="flex items-center gap-2 shrink-0">
          <span class="text-[11px] text-ink/25 hidden sm:inline">{{ formatPostDate(post.created_at) }}</span>
          <span class="flex items-center gap-0.5 text-ink/35">
            <MessageCircle :size="13" />
            <span class="text-[12px]">{{ post.comment_count || 0 }}</span>
          </span>
          <button @click.stop="$emit('like', post)"
                  class="flex items-center gap-0.5 transition-all"
                  :class="post.is_liked ? 'text-red-400' : 'text-ink/35 hover:text-red-300'">
            <Heart :size="13" :fill="post.is_liked ? 'currentColor' : 'none'" />
            <span class="text-[12px]">{{ post.like_count || 0 }}</span>
          </button>
        </div>
      </div>
      <!-- Date on small screens (below footer row) -->
      <p class="text-[10px] text-ink/25 mt-1 sm:hidden">{{ formatPostDate(post.created_at) }}</p>
    </div>
  </article>

  <!-- ── Full card (detail / single-column) ── -->
  <article v-else
           class="glass rounded-2xl overflow-hidden animate-fade-in transition-all relative"
           :class="clickable && 'cursor-pointer hover:bg-ink/[0.05]'"
           style="border:1px solid rgb(var(--ink) / 0.07)"
           @click="onCardClick">
    <ImageViewer :src="viewImg" @close="viewImg = ''" />

    <div class="p-5 relative">
      <!-- Header -->
      <div class="flex items-start justify-between mb-3">
        <div class="flex items-center gap-3 min-w-0">
          <!-- Avatar -->
          <Avatar :name="post.user.nickname" :url="post.user.avatar_url" :size="44"
                  class="cursor-pointer hover:opacity-90 transition-opacity"
                  @click.stop="openUserProfile(post.user.id)" />
          <div class="min-w-0">
            <div class="flex items-center gap-1.5">
              <span class="text-sm font-semibold text-ink/90 truncate">{{ post.user.nickname }}</span>
              <LevelBadge v-if="post.user.level" :level="post.user.level" :tier="post.user.tier" dense />
              <VipBadge v-if="post.user.vip" dense />
              <span v-if="post.is_friend && !isOwn"
                    class="text-[10px] px-1.5 py-0.5 rounded-md font-medium shrink-0"
                    style="background:rgba(51,144,236,0.15);color:#3390EC">好友</span>
            </div>
            <div class="flex items-center gap-1.5 text-xs text-ink/30 mt-0.5">
              <span>{{ formatTime(post.created_at) }}</span>
              <span class="text-ink/15">·</span>
              <span class="flex items-center gap-0.5">
                <MapPin :size="10" /> {{ post.ip || '未知地区' }}
              </span>
            </div>
          </div>
        </div>

        <!-- Right actions -->
        <div class="flex items-center gap-2 shrink-0">
          <template v-if="!isOwn && !post.is_friend">
            <button v-if="!requested" @click.stop="addFriend"
                    class="flex items-center gap-1 px-2.5 py-1 rounded-lg text-xs font-medium transition-all"
                    style="background:rgba(51,144,236,0.15);color:#3390EC;border:1px solid rgba(51,144,236,0.25)">
              <UserPlus :size="12" /> 加好友
            </button>
            <span v-else class="flex items-center gap-1 px-2.5 py-1 rounded-lg text-xs font-medium"
                  style="background:rgb(var(--ink) / 0.05);color:rgb(var(--ink) / 0.3)">
              <Check :size="12" /> 已申请
            </span>
          </template>
          <div class="relative" v-if="isOwn">
            <button @click.stop="showMenu = !showMenu"
                    class="w-8 h-8 rounded-lg flex items-center justify-center
                           text-ink/30 hover:text-ink/60 hover:bg-ink/5 transition-all">
              <MoreHorizontal :size="16" />
            </button>
            <div v-if="showMenu" v-click-outside="() => showMenu = false"
                 class="absolute right-0 top-9 glass-strong rounded-xl py-1 z-10 min-w-28 animate-scale-in">
              <button @click.stop="() => { $emit('delete', post); showMenu = false }"
                      class="w-full px-4 py-2 text-sm text-danger hover:bg-danger/10 transition-all text-left">
                删除
              </button>
            </div>
          </div>
        </div>
      </div>

      <!-- Title -->
      <h2 class="text-lg font-bold text-ink leading-snug mb-2 break-words">{{ post.title }}</h2>

      <!-- Content (Markdown) -->
      <div v-if="post.content"
           class="md-body text-[15px] text-ink/85 leading-relaxed mb-3 break-words"
           v-html="renderMarkdown(post.content)"
           @click.stop="handleMdClick" />

      <!-- Images -->
      <div v-if="post.images?.length"
           class="grid gap-1.5 mb-3 rounded-xl overflow-hidden"
           :class="post.images.length === 1 ? 'grid-cols-1' : post.images.length === 2 ? 'grid-cols-2' : 'grid-cols-3'">
        <img v-for="(img, i) in post.images" :key="i" :src="img"
             class="w-full object-cover rounded-lg cursor-zoom-in hover:opacity-90 transition-opacity"
             :class="post.images.length === 1 ? 'max-h-80' : 'aspect-square'"
             loading="lazy" @click.stop="viewImg = img" />
      </div>

      <!-- AI tag -->
      <div v-if="post.ai_assisted" class="mb-3">
        <span class="inline-flex items-center gap-1 text-xs text-primary/60">
          <Sparkles :size="11" /> AI 辅助创作
        </span>
      </div>

      <!-- Action bar: 分享 | 评论 | 点赞 -->
      <div class="grid grid-cols-3 pt-2" style="border-top:1px solid rgb(var(--ink) / 0.06)">
        <button @click.stop="share"
                class="flex items-center justify-start gap-1.5 px-2 py-2 rounded-lg text-sm
                       text-ink/40 hover:bg-ink/5 hover:text-ink/70 transition-all">
          <Share2 :size="15" />
          <span class="text-xs">分享</span>
        </button>

        <button @click.stop="$emit('comment', post.post_id)"
                class="flex items-center justify-center gap-1.5 px-2 py-2 rounded-lg text-sm
                       text-ink/40 hover:bg-ink/5 hover:text-ink/70 transition-all">
          <MessageCircle :size="15" />
          <span class="text-xs">{{ post.comment_count || '评论' }}</span>
        </button>

        <button @click.stop="$emit('like', post)"
                class="flex items-center justify-end gap-1.5 px-2 py-2 rounded-lg text-sm transition-all"
                :class="post.is_liked
                  ? 'text-red-400 hover:bg-red-500/10'
                  : 'text-ink/40 hover:bg-ink/5 hover:text-ink/60'">
          <Heart :size="15" :fill="post.is_liked ? 'currentColor' : 'none'" />
          <span class="text-xs">{{ post.like_count || '赞' }}</span>
        </button>
      </div>
    </div>
  </article>
</template>

<script setup>
import { ref, computed } from 'vue'
import { useRouter } from 'vue-router'
import { Heart, MessageCircle, MoreHorizontal, Sparkles, UserPlus, Check, Share2, MapPin } from 'lucide-vue-next'
import { useAuthStore }   from '@/stores/auth'
import { useToast }       from '@/composables/useToast'
import { useUserProfile } from '@/composables/useUserProfile'
import { useAddFriend }   from '@/composables/useAddFriend'
import { copyText }       from '@/utils/clipboard'
import { renderMarkdown } from '@/utils/markdown'
import { formatRelative as formatTime, formatPostDate } from '@/utils/time'
import ImageViewer from '@/components/ui/ImageViewer.vue'
import VipBadge    from '@/components/ui/VipBadge.vue'
import LevelBadge  from '@/components/ui/LevelBadge.vue'
import Avatar      from '@/components/ui/Avatar.vue'

const props = defineProps({
  post:      { type: Object, required: true },
  clickable: { type: Boolean, default: true },
  compact:   { type: Boolean, default: false },
})
defineEmits(['like', 'comment', 'delete'])

const router      = useRouter()
const authStore   = useAuthStore()
const toast       = useToast()
const { openAddFriend } = useAddFriend()
const { openUserProfile } = useUserProfile()

function handleMdClick(e) {
  const a = e.target.closest('a')
  if (!a) return
  const href = a.getAttribute('href') || ''
  if (href.startsWith('/')) {
    e.preventDefault()
    e.stopPropagation()
    router.push(href)
  }
}

const coverImage = computed(() => props.post.images?.[0] || '')

// 无图帖子封面（Telegram 风）：纯色底（取自 Telegram 头像配色）+ 淡白 doodle 密铺花纹，
// 白字可读，与聊天区花纹底同一套视觉语言。不用渐变。
const COVER_COLORS = ['#5CAFFA', '#7BC862', '#E17076', '#F0883E', '#A695E7', '#6EC9CB', '#EE7AAE']
const genCoverStyle = computed(() => {
  const id = Number(props.post.post_id) || 0
  return `background-color:${COVER_COLORS[id % COVER_COLORS.length]}`
})

// Telegram 涂鸦花纹（圆圈 + 加号），淡白描边密铺，与 .app-bg 同源
const TG_DOODLE = encodeURIComponent(
  `<svg xmlns='http://www.w3.org/2000/svg' width='120' height='120' viewBox='0 0 120 120'>` +
  `<g fill='none' stroke='#ffffff' stroke-opacity='0.18' stroke-width='2'>` +
  `<circle cx='20' cy='24' r='9'/>` +
  `<path d='M70 14v18M61 23h18'/>` +
  `<circle cx='100' cy='44' r='5'/>` +
  `<path d='M28 70v16M20 78h16'/>` +
  `<circle cx='90' cy='88' r='11'/>` +
  `<path d='M106 104v14M99 111h14'/>` +
  `<circle cx='52' cy='106' r='5'/>` +
  `</g></svg>`
)
const patternOverlay = {
  backgroundImage: `url("data:image/svg+xml,${TG_DOODLE}")`,
  backgroundSize: '120px 120px',
  backgroundRepeat: 'repeat',
}

function onCardClick() {
  if (props.compact || props.clickable) router.push(`/post/${props.post.post_id}`)
}
const showMenu    = ref(false)
const isOwn       = computed(() => props.post.user.id === authStore.user?.id)
const viewImg     = ref('')
const requested   = ref(false)

function addFriend() {
  // 打开带招呼语输入的弹层；发送成功后标记为「已申请」
  openAddFriend(props.post.user, () => { requested.value = true })
}

async function share() {
  const url = `${window.location.origin}/post/${props.post.post_id}`
  const ok = await copyText(url)
  if (ok) toast.success('帖子链接已复制到剪贴板')
  else    toast.error('复制失败：' + url)
}

const vClickOutside = {
  mounted(el, binding) {
    el._outside = (e) => { if (!el.contains(e.target)) binding.value(e) }
    document.addEventListener('click', el._outside, true)
  },
  unmounted(el) { document.removeEventListener('click', el._outside, true) },
}
</script>
