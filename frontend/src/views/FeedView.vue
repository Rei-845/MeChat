<template>
  <div class="flex flex-col h-full overflow-hidden">
    <!-- Header (fixed, above the scroll area) -->
    <div class="shrink-0 px-4 md:px-6 pt-5 pb-3"
         style="border-bottom:1px solid rgb(var(--ink) / 0.05)">
      <div class="max-w-[1600px] mx-auto">
        <div class="flex items-center gap-3">
          <h2 class="text-2xl font-extrabold text-primary mr-1">MeChatPosts</h2>

          <div class="ml-auto flex items-center gap-1">
            <!-- Sort toggle（图标）：火苗=热度 / 时钟=最新 -->
            <button v-if="showSort" @click="toggleSort"
                    class="w-8 h-8 rounded-lg flex items-center justify-center text-ink/40
                           hover:text-ink/70 hover:bg-ink/5 transition-all"
                    :title="sortMode === 'hot' ? '当前按热度排序，点击切换为最新' : '当前按最新排序，点击切换为热度'">
              <Flame v-if="sortMode === 'hot'" :size="15" class="text-orange-400" />
              <Clock v-else :size="15" class="text-primary-light" />
            </button>
            <!-- Refresh -->
            <button @click="refresh" :disabled="loading"
                    class="w-8 h-8 rounded-lg flex items-center justify-center text-ink/40
                           hover:text-ink/70 hover:bg-ink/5 transition-all"
                    :class="loading && 'opacity-40'">
              <RefreshCw :size="14" :class="loading && 'animate-spin'" />
            </button>
          </div>
        </div>

        <!-- Search row（居中） -->
        <div class="mt-3 max-w-2xl mx-auto">
          <div class="relative">
            <Search :size="14" class="absolute left-3 top-1/2 -translate-y-1/2 text-ink/30" />
            <input v-model="searchQuery" @input="onSearchInput" type="search"
                   placeholder="搜索帖子标题…"
                   class="w-full pl-9 pr-9 py-2 text-sm rounded-xl bg-transparent outline-none text-ink/90 placeholder:text-ink/30"
                   style="background:rgb(var(--ink) / 0.04);border:1px solid rgb(var(--ink) / 0.08)" />
            <button v-if="searchQuery" @click="clearSearch"
                    class="absolute right-2.5 top-1/2 -translate-y-1/2 text-ink/30 hover:text-ink/60">
              <X :size="14" />
            </button>
          </div>
        </div>
      </div>
    </div>

    <!-- Scrollable masonry grid -->
    <div ref="feedScroll" class="flex-1 overflow-y-auto" @scroll="onFeedScroll">
      <!-- 下拉刷新指示器 -->
      <div class="flex justify-center items-end overflow-hidden"
           :style="{ height: ptr.pullY.value + 'px' }">
        <RefreshCw :size="18" class="mb-1 text-ink/50"
                   :class="ptr.refreshing.value && 'animate-spin'"
                   :style="ptr.indicatorStyle.value" />
      </div>

      <div class="max-w-[1600px] mx-auto px-3 md:px-5 py-4">

        <!-- Skeleton loader -->
        <div v-if="loading && !posts.length"
             class="columns-2 md:columns-3 xl:columns-4 gap-3">
          <div v-for="i in 8" :key="i" class="break-inside-avoid mb-3">
            <div class="glass rounded-2xl overflow-hidden animate-pulse">
              <div class="bg-ink/5" :style="`height:${120 + (i % 3) * 60}px`" />
              <div class="p-3 space-y-2">
                <div class="h-2.5 bg-ink/5 rounded w-full" />
                <div class="h-2.5 bg-ink/5 rounded w-3/4" />
                <div class="flex items-center gap-1.5 mt-3">
                  <div class="w-5 h-5 rounded-full bg-ink/5" />
                  <div class="h-2 bg-ink/5 rounded w-16" />
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- 2-column masonry grid -->
        <div v-else-if="posts.length"
             class="columns-2 md:columns-3 xl:columns-4 gap-3">
          <div v-for="post in posts" :key="post.post_id" class="break-inside-avoid mb-3">
            <PostCard
              :post="post"
              compact
              @like="toggleLike"
              @delete="deletePost"
            />
          </div>
        </div>

        <!-- Empty state -->
        <div v-else-if="!loading" class="text-center py-20">
          <Newspaper :size="48" class="mx-auto text-ink/20 mb-4" />
          <template v-if="isSearching">
            <p class="text-ink/40 text-sm">没有找到标题包含「{{ searchQuery.trim() }}」的帖子</p>
            <p class="text-ink/25 text-xs mt-1">换个关键词试试</p>
          </template>
          <template v-else>
            <p class="text-ink/40 text-sm">还没有动态</p>
            <p class="text-ink/25 text-xs mt-1">关注更多好友，发现精彩内容</p>
          </template>
        </div>

        <!-- 无限滚动：底部加载指示 -->
        <div v-if="hasMore && posts.length" class="text-center py-6">
          <Loader2 v-if="loading" :size="18" class="animate-spin text-ink/30 mx-auto" />
        </div>
        <!-- 底部哨兵（IntersectionObserver 锚点，始终保留在 DOM） -->
        <div ref="loadMoreSentinel" class="h-1" />
      </div>
    </div>

    <!-- 回到顶部 -->
    <Transition enter-active-class="transition-all duration-200" enter-from-class="opacity-0 scale-75"
                leave-active-class="transition-all duration-200" leave-to-class="opacity-0 scale-75">
      <button v-if="showBackTop" @click="scrollToTop"
              class="fixed bottom-36 md:bottom-24 right-5 md:right-7 z-30 w-10 h-10 rounded-full
                     flex items-center justify-center transition-all hover:scale-110 active:scale-95"
              style="background:rgb(var(--ink) / 0.08);border:1px solid rgb(var(--ink) / 0.12);backdrop-filter:blur(8px)"
              title="回到顶部">
        <ArrowUp :size="18" class="text-ink/60" />
      </button>
    </Transition>

    <!-- FAB -->
    <button @click="showCreate = true"
            class="fixed bottom-20 md:bottom-7 right-5 md:right-7 z-30 w-14 h-14 rounded-full flex items-center justify-center
                   transition-all hover:scale-105 active:scale-95"
            style="background:linear-gradient(135deg,#3390EC,#2980DE);color:#fff;box-shadow:0 8px 24px rgba(51,144,236,0.45)"
            title="发布动态">
      <Plus :size="26" class="text-white" />
    </button>

    <CreatePostModal v-if="showCreate" @close="showCreate = false" @created="onPostCreated" />
    <CommentsModal   v-if="commentPostId" :post-id="commentPostId" @close="commentPostId = null" />
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted, nextTick } from 'vue'
import { Plus, Loader2, Newspaper, RefreshCw, Search, X, Flame, Clock, ArrowUp } from 'lucide-vue-next'
import { usePullToRefresh } from '@/composables/usePullToRefresh'
import { useToast }     from '@/composables/useToast'
import * as feedApi     from '@/api/feed'
import PostCard         from '@/components/feed/PostCard.vue'
import CreatePostModal  from '@/components/feed/CreatePostModal.vue'
import CommentsModal    from '@/components/feed/CommentsModal.vue'

const toast        = useToast()

const posts        = ref([])
const loading      = ref(false)
const hasMore      = ref(false)
const page         = ref(1)
const feedScroll        = ref(null)
const showBackTop       = ref(false)
const loadMoreSentinel  = ref(null)
const ptr = usePullToRefresh(() => feedScroll.value, refresh)
let scrollObserver = null

function onFeedScroll(e) {
  showBackTop.value = e.target.scrollTop > 400
}

function setupObserver() {
  if (scrollObserver) scrollObserver.disconnect()
  if (!loadMoreSentinel.value || !feedScroll.value) return
  // root 必须是滚动容器本身，否则 rootMargin 对祖先 overflow 裁剪无效
  scrollObserver = new IntersectionObserver((entries) => {
    if (entries[0].isIntersecting && !loading.value && hasMore.value) {
      loadMore()
    }
  }, { root: feedScroll.value, rootMargin: '0px 0px 300px 0px' })
  scrollObserver.observe(loadMoreSentinel.value)
}
function scrollToTop() {
  feedScroll.value?.scrollTo({ top: 0, behavior: 'smooth' })
}
const showCreate   = ref(false)
const commentPostId = ref(null)
const searchQuery  = ref('')
const sortMode     = ref('hot')        // 'hot' | 'time'
let   searchTimer  = null

// 首页只展示推荐 feed；排序按钮始终可用（「我的帖子」改由个人主页进入）
const isSearching = computed(() => searchQuery.value.trim().length > 0)
const showSort    = computed(() => true)

async function loadPosts() {
  loading.value = true
  try {
    let res
    if (isSearching.value) {
      res = await feedApi.searchPosts(searchQuery.value.trim(), sortMode.value, page.value, 20)
    } else {
      res = await feedApi.getFeed(page.value, 20, sortMode.value)
    }
    if (page.value === 1) posts.value = res.data.list || []
    else posts.value.push(...(res.data.list || []))
    hasMore.value = res.data.has_more
  } finally {
    loading.value = false
  }
}

function onSearchInput() {
  clearTimeout(searchTimer)
  searchTimer = setTimeout(() => { page.value = 1; posts.value = []; loadPosts() }, 350)
}

function clearSearch() {
  searchQuery.value = ''
  page.value = 1
  posts.value = []
  loadPosts()
}

function toggleSort() {
  sortMode.value = sortMode.value === 'hot' ? 'time' : 'hot'
  page.value = 1
  posts.value = []
  loadPosts()
}

async function loadMore() {
  page.value++
  await loadPosts()
}

async function refresh() {
  page.value = 1
  posts.value = []
  await loadPosts()
}

async function toggleLike(post) {
  try {
    if (post.is_liked) {
      await feedApi.unlikePost(post.post_id)
      post.is_liked = false
      post.like_count = Math.max(0, post.like_count - 1)
    } else {
      await feedApi.likePost(post.post_id)
      post.is_liked = true
      post.like_count++
    }
  } catch {}
}

async function deletePost(post) {
  try {
    await feedApi.deletePost(post.post_id)
    posts.value = posts.value.filter(p => p.post_id !== post.post_id)
    toast.success('已删除')
  } catch {}
}

function onPostCreated(newPost) {
  posts.value.unshift(newPost)
  showCreate.value = false
}

onMounted(() => { loadPosts(); ptr.attach(); nextTick(() => setupObserver()) })
onUnmounted(() => { ptr.detach(); if (scrollObserver) scrollObserver.disconnect() })
</script>
