<template>
  <div class="flex flex-col h-full">
    <!-- Header -->
    <div class="flex items-center justify-between px-6 py-4 shrink-0"
         style="border-bottom:1px solid rgb(var(--ink) / 0.06)">
      <button @click="showInfo = true" class="flex items-center gap-3 transition-all hover:opacity-80"
              title="查看 AI 助手能做什么">
        <div class="w-9 h-9 rounded-xl flex items-center justify-center"
             style="background:linear-gradient(135deg,rgba(51,144,236,0.25),rgba(94,181,247,0.25));border:1px solid rgba(51,144,236,0.3)">
          <Sparkles :size="18" class="text-primary-light" />
        </div>
        <div class="text-left">
          <h2 class="text-2xl font-extrabold text-primary leading-tight flex items-center gap-1.5">
            MeChatAgent <Info :size="15" class="text-ink/30" />
          </h2>
        </div>
      </button>

      <!-- Clear context -->
      <button v-if="messages.length" @click="clearContext" :disabled="loading"
              class="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-medium transition-all
                     text-ink/45 hover:text-ink/80 hover:bg-ink/5"
              :class="loading && 'opacity-40 pointer-events-none'"
              title="清空对话并删除所有聊天记录">
        <Trash2 :size="14" /> 清空
      </button>
    </div>

    <!-- Messages -->
    <div ref="scrollEl" class="flex-1 overflow-y-auto px-4 sm:px-6 py-6">
      <div class="max-w-2xl mx-auto">
        <!-- Empty state -->
        <div v-if="!messages.length" class="flex flex-col items-center justify-center text-center pt-10">
          <div class="w-16 h-16 rounded-2xl flex items-center justify-center mb-5"
               style="background:linear-gradient(135deg,rgba(51,144,236,0.2),rgba(94,181,247,0.2));border:1px solid rgba(51,144,236,0.3);box-shadow:0 0 40px rgba(51,144,236,0.15)">
            <Sparkles :size="30" class="text-primary-light" />
          </div>
          <h3 class="text-lg font-bold text-ink">有什么可以帮你的？</h3>
          <p class="text-sm text-ink/40 mt-2 mb-7">
            {{ quota.vip_user ? '智能助手已解锁，可帮你处理 MeChat 内的操作' : '日常问答不限次数，试试这些' }}
          </p>

          <!-- 示例提示词按 VIP 区分：非 VIP 只给纯对话类，避免点了 Agent 能力却没反应、一脸懵 -->
          <div class="grid sm:grid-cols-2 gap-3 w-full">
            <button v-for="s in activeSuggestions" :key="s"
                    @click="useSuggestion(s)"
                    class="text-left px-4 py-3 rounded-xl text-sm text-ink/70 transition-all hover:bg-ink/[0.06]"
                    style="background:rgb(var(--ink) / 0.03);border:1px solid rgb(var(--ink) / 0.07)">
              {{ s }}
            </button>
          </div>

          <!-- VIP：已解锁提示 -->
          <div v-if="quota.vip_user"
               class="mt-6 w-full flex items-start gap-2.5 px-4 py-3 rounded-xl text-left"
               style="background:rgba(245,158,11,0.08);border:1px solid rgba(245,158,11,0.2)">
            <Zap :size="15" class="shrink-0 mt-0.5 text-yellow-400" />
            <div class="text-[12px] leading-relaxed">
              <span class="text-yellow-400 font-semibold">智能助手已解锁（VIP）</span>
              <span class="text-ink/55"> · 可帮你查看会话/好友、发消息、发帖、加好友、总结聊天记录（操作前会请你确认），并保存最近 30 轮对话记忆。</span>
            </div>
          </div>

          <!-- 非 VIP：明确说明「当前只是普通对话」+ 醒目开通入口 -->
          <div v-else class="mt-6 w-full rounded-2xl overflow-hidden text-left"
               style="border:1px solid rgba(245,158,11,0.35)">
            <div class="px-4 py-3.5" style="background:rgba(245,158,11,0.08)">
              <div class="flex items-center gap-2 mb-1.5">
                <Crown :size="17" class="text-yellow-400" />
                <span class="text-sm font-bold text-ink">当前为普通对话模式</span>
              </div>
              <p class="text-[12px] text-ink/55 leading-relaxed">
                你可以<b class="text-ink/75">不限次数</b>地自由问答、润色、写作。但让 AI 真正
                <b class="text-ink/75">查未读 / 发消息 / 发帖 / 加好友 / 总结聊天记录</b>
                等智能助手（Agent）能力为 <b class="text-ink/75">VIP 专属</b>（对话记忆也从 10 轮提升到 30 轮）。
              </p>
            </div>
            <button @click="$router.push('/vip')"
                    class="w-full flex items-center justify-center gap-1.5 py-3 text-sm font-bold text-yellow-900 transition-all hover:brightness-105 active:brightness-95"
                    style="background:linear-gradient(135deg,#F59E0B,#FBBF24)">
              <Crown :size="15" /> 开通 VIP 解锁智能助手
            </button>
          </div>
        </div>

        <!-- Conversation -->
        <div v-else class="space-y-5">
          <div v-for="(m, i) in messages" :key="i"
               class="flex gap-3" :class="m.role === 'user' ? 'flex-row-reverse' : ''">
            <!-- Avatar -->
            <div v-if="m.role === 'user'"
                 class="w-8 h-8 rounded-full overflow-hidden shrink-0 cursor-pointer hover:opacity-90 transition-opacity"
                 @click="openUserProfile(auth.user?.id)">
              <img v-if="auth.user?.avatar_url" :src="auth.user.avatar_url" class="w-full h-full object-cover" />
              <div v-else class="w-full h-full flex items-center justify-center text-sm font-bold"
                   style="background:rgb(var(--ink) / 0.12)">
                {{ auth.user?.nickname?.[0]?.toUpperCase() || '我' }}
              </div>
            </div>
            <button v-else
                    @click="showInfo = true"
                    class="w-8 h-8 rounded-lg shrink-0 flex items-center justify-center transition-all hover:scale-110 active:scale-95"
                    style="background:linear-gradient(135deg,#3390EC,#2980DE);color:#fff"
                    title="查看 AI 助手能做什么">
              <Sparkles :size="15" class="text-ink" />
            </button>

            <!-- Content column: tool chips + bubble + confirm cards -->
            <div class="flex flex-col gap-2 max-w-[82%] min-w-0"
                 :class="m.role === 'user' ? 'items-end' : 'items-start'">
              <!-- Tool call chips (assistant) -->
              <ToolCallChip v-for="(t, ti) in m.tools" :key="'t'+ti" :tool="t" />

              <!-- Bubble -->
              <div v-if="m.content"
                   class="px-4 py-2.5 text-sm leading-relaxed break-words"
                   :class="m.role === 'user' ? 'msg-bubble-self whitespace-pre-wrap' : 'msg-bubble-other'">
                <template v-if="m.role === 'user'">{{ m.content }}</template>
                <div v-else class="md-body" v-html="renderMarkdown(m.content)" @click="handleMdClick($event)" />
                <button v-if="m.role === 'assistant'"
                        @click="copyText(m.content)"
                        class="block mt-2 text-[11px] text-ink/30 hover:text-primary-light transition-colors">
                  <Copy :size="10" class="inline mr-1" />复制
                </button>
              </div>

              <!-- Inline typing dots: 仅当该 assistant 消息尚未产生任何内容时显示（避免双头像） -->
              <div v-if="m.role === 'assistant' && loading && i === messages.length - 1
                         && !m.content && !m.tools?.length && !m.actions?.length"
                   class="msg-bubble-other px-4 py-3.5">
                <div class="flex gap-1">
                  <span class="typing-dot" />
                  <span class="typing-dot" style="animation-delay:0.15s" />
                  <span class="typing-dot" style="animation-delay:0.3s" />
                </div>
              </div>

              <!-- Confirm cards (assistant write actions) -->
              <ConfirmCard v-for="(a, ai) in m.actions" :key="'a'+ai"
                           :action="a" :loading="a._loading"
                           @confirm="onConfirm(a)" @cancel="onCancel(a)" />
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Input -->
    <div class="shrink-0 px-4 sm:px-6 pb-5 pt-2">
      <div class="max-w-2xl mx-auto">
        <div class="flex items-end gap-2 p-2 rounded-2xl"
             style="background:rgb(var(--ink) / 0.05);border:1px solid rgb(var(--ink) / 0.1)">
          <textarea
            ref="inputEl"
            v-model="draft"
            @keydown.enter.exact.prevent="send"
            @keydown.shift.enter.prevent="insertNewline"
            @input="autoGrow"
            rows="1"
            placeholder="有问题，尽管问"
            class="flex-1 bg-transparent resize-none outline-none text-sm text-ink/90 placeholder:text-ink/30 px-2 py-1.5 max-h-40"
          />
          <button @click="send" :disabled="!draft.trim() || loading"
                  class="w-9 h-9 rounded-xl flex items-center justify-center shrink-0 transition-all"
                  :style="draft.trim() && !loading
                    ? 'background:linear-gradient(135deg,#3390EC,#2980DE);color:#fff;box-shadow:0 2px 10px rgba(51,144,236,0.3)'
                    : 'background:rgb(var(--ink) / 0.06);cursor:not-allowed'">
            <Loader2 v-if="loading" :size="16" class="text-ink animate-spin" />
            <ArrowUp v-else :size="16" :class="draft.trim() ? 'text-ink' : 'text-ink/30'" />
          </button>
        </div>
        <p class="text-center text-[11px] text-ink/20 mt-2">
          AI 生成内容仅供参考
        </p>
      </div>
    </div>

    <!-- AI 助手能力说明弹窗 -->
    <Teleport to="body">
      <div v-if="showInfo" class="fixed inset-0 z-50 flex items-center justify-center p-4"
           style="background:rgba(0,0,0,0.6);backdrop-filter:blur(4px)"
           @click.self="showInfo = false">
        <div class="w-full max-w-md glass-strong rounded-2xl p-6 animate-scale-in max-h-[85vh] overflow-y-auto">
          <div class="flex items-center justify-between mb-1">
            <div class="flex items-center gap-2.5">
              <div class="w-9 h-9 rounded-xl flex items-center justify-center"
                   style="background:linear-gradient(135deg,#3390EC,#2980DE);color:#fff">
                <Sparkles :size="18" class="text-ink" />
              </div>
              <h3 class="font-bold text-ink">AI 助手能做什么</h3>
            </div>
            <button @click="showInfo = false" class="text-ink/40 hover:text-ink/70 transition-colors">
              <X :size="18" />
            </button>
          </div>

          <p class="text-[12px] text-ink/45 mb-4 leading-relaxed">
            日常问答对所有用户<b class="text-ink/70">不限次数免费</b>使用。以下操作能力为 VIP 专属，涉及发送/发布的操作会先弹出确认框，确认后才执行。
          </p>

          <div class="space-y-1.5">
            <div v-for="cap in capabilities" :key="cap.title"
                 class="flex items-start gap-3 px-3 py-2.5 rounded-xl"
                 style="background:rgb(var(--ink) / 0.03);border:1px solid rgb(var(--ink) / 0.06)">
              <component :is="cap.icon" :size="16" class="shrink-0 mt-0.5 text-primary-light" />
              <div class="min-w-0">
                <p class="text-[13px] font-medium text-ink/85 flex items-center gap-1.5">
                  {{ cap.title }}
                  <span v-if="cap.write" class="text-[10px] px-1.5 py-0.5 rounded"
                        style="background:rgba(245,158,11,0.15);color:#fbbf24">需确认</span>
                </p>
                <p class="text-[12px] text-ink/45 leading-snug">{{ cap.desc }}</p>
              </div>
            </div>
          </div>

          <div v-if="!quota.vip_user"
               class="mt-4 flex items-center gap-2 px-3 py-2.5 rounded-xl text-[12px]"
               style="background:rgba(245,158,11,0.08);border:1px solid rgba(245,158,11,0.2)">
            <Zap :size="14" class="shrink-0 text-yellow-400" />
            <span class="text-ink/55 flex-1">以上操作能力 + 30 轮对话记忆为 VIP 专属，当前为普通对话模式（10 轮记忆）</span>
            <button @click="showInfo = false; $router.push('/vip')"
                    class="shrink-0 px-2.5 py-1 rounded-lg text-[12px] font-semibold text-yellow-900"
                    style="background:linear-gradient(135deg,#F59E0B,#FBBF24)">去开通</button>
          </div>
        </div>
      </div>
    </Teleport>
  </div>
</template>

<script setup>
import { ref, reactive, computed, nextTick, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { Sparkles, Zap, Copy, Loader2, ArrowUp, Info, X, Trash2, Crown,
         MessageSquare, Users, Compass, Search, FileText, Send, UserPlus, ScrollText } from 'lucide-vue-next'
import * as aiApi    from '@/api/ai'
import { useToast }  from '@/composables/useToast'
import { useAuthStore } from '@/stores/auth'
import { useUserProfile } from '@/composables/useUserProfile'
import { copyText as copyToClipboard } from '@/utils/clipboard'
import { renderMarkdown } from '@/utils/markdown'
import ToolCallChip from '@/components/ai/ToolCallChip.vue'
import ConfirmCard  from '@/components/ai/ConfirmCard.vue'

const toast  = useToast()
const auth   = useAuthStore()
const router = useRouter()
const { openUserProfile } = useUserProfile()

const messages  = ref([])           // [{ role:'user'|'assistant', content }]
const draft     = ref('')
const loading   = ref(false)
const showInfo  = ref(false)
const quota     = reactive({ vip_user: false })   // 仅区分 VIP（AI 调用已不限次数）

// AI 助手能力清单（与后端 agent 工具对应）
const capabilities = [
  { icon: MessageSquare, title: '查看会话与未读', desc: '查询你的私聊/群聊列表和未读消息数' },
  { icon: Users,         title: '查看好友',       desc: '查询好友列表及在线状态' },
  { icon: Compass,       title: '浏览动态',       desc: '获取动态广场的推荐帖子摘要' },
  { icon: FileText,      title: '查看帖子详情',   desc: '读取某篇帖子的完整正文并总结' },
  { icon: Search,        title: '搜索帖子/用户',  desc: '按标题搜索帖子、按昵称查找用户' },
  { icon: ScrollText,    title: '总结聊天记录',   desc: '提炼某个会话最近聊了什么' },
  { icon: Send,          title: '发送消息',       desc: '代你给好友或会话发消息', write: true },
  { icon: FileText,      title: '发布帖子',       desc: '代你在动态广场发帖', write: true },
  { icon: UserPlus,      title: '添加好友',       desc: '代你向某人发送好友申请', write: true },
]

// 历史由后端持久化 打开时加载 切走/刷新都不丢
let pollTimer = null
// 后端消息转本地结构 带 pending_action 的恢复确认框
function mapMsg(m) {
  const msg = { role: m.role, content: m.content, status: m.status, tools: [], actions: [] }
  if (m.pending_action) {
    try {
      const a = JSON.parse(m.pending_action)
      msg.actions = [{ tool: a.tool, label: a.label, preview: a.preview || '', args: a.args || {}, status: 'pending', result: '', _loading: false }]
    } catch {}
  }
  return msg
}

async function loadHistory() {
  try {
    const res = await aiApi.getHistory()
    messages.value = (res.data?.messages || []).map(mapMsg)
    pollPending()
  } catch { messages.value = [] }
}

// 末条仍在生成则轮询 直到后台落库完成 修复刷新时答案接不上
function pollPending() {
  const last = messages.value[messages.value.length - 1]
  if (!last || last.role !== 'assistant' || last.status !== 'pending') return
  clearTimeout(pollTimer)
  pollTimer = setTimeout(async () => {
    try {
      const res = await aiApi.getHistory()
      messages.value = (res.data?.messages || []).map(mapMsg)
    } catch {}
    pollPending()
  }, 2000)
}

// 清空对话 删后端记录
async function clearContext() {
  if (loading.value || !messages.value.length) return
  try { await aiApi.clearHistory() } catch {}
  messages.value = []
  toast.success('已清空对话')
}

const scrollEl = ref(null)
const inputEl  = ref(null)

// 示例提示词分两套：VIP 给 Agent 能力类；普通用户只给纯对话类，
// 避免非 VIP 点了「查未读/发帖」却没反应、被无 Agent 能力的 AI 整懵。
const agentSuggestions = [
  '我有哪些未读消息？',
  '帮我发一条帖子，主题是周末爬山',
  '总结一下我和某人最近的聊天',
  '帮我给好友发条消息说晚点到',
]
const chatSuggestions = [
  '把这句话润色得更礼貌：我现在没空',
  '帮我写一段周末爬山的朋友圈文案',
  '用一句话解释什么是区块链',
  '推荐三个适合周末放松的活动',
]
const activeSuggestions = computed(() => quota.vip_user ? agentSuggestions : chatSuggestions)

function scrollToBottom() {
  nextTick(() => {
    if (scrollEl.value) scrollEl.value.scrollTop = scrollEl.value.scrollHeight
  })
}

function autoGrow() {
  const el = inputEl.value
  if (!el) return
  el.style.height = 'auto'
  el.style.height = Math.min(el.scrollHeight, 160) + 'px'
}

function insertNewline() {
  const el = inputEl.value
  if (!el) return
  const start = el.selectionStart
  const end   = el.selectionEnd
  el.value = el.value.slice(0, start) + '\n' + el.value.slice(end)
  el.selectionStart = el.selectionEnd = start + 1
  el.dispatchEvent(new Event('input'))  // 同步 v-model + autoGrow
}

function useSuggestion(s) {
  draft.value = s
  nextTick(autoGrow)
  send()
}

async function send() {
  const text = draft.value.trim()
  if (!text || loading.value) return

  messages.value.push({ role: 'user', content: text })
  draft.value = ''
  nextTick(() => { autoGrow(); scrollToBottom() })

  // 只发当前这条 历史与记忆窗口由后端按 VIP 决定
  const assistant = reactive({ role: 'assistant', content: '', status: 'pending', tools: [], actions: [] })
  messages.value.push(assistant)

  // 超出记忆窗口的旧对话不再显示 与后端记忆长度一致 一轮两条
  const win = (quota.vip_user ? 30 : 10) * 2
  if (messages.value.length > win) messages.value = messages.value.slice(-win)

  loading.value = true

  try {
    await aiApi.streamChat(text, { onEvent: (ev) => handleEvent(assistant, ev) })
    assistant.status = 'done'
  } catch (e) {
    assistant.content += (assistant.content ? '\n\n' : '') + '⚠️ ' + (e?.message || '出错了，请稍后再试')
    assistant.status = 'error'
  } finally {
    loading.value = false
    await loadQuota()
    scrollToBottom()
  }
}

// 处理 SSE 事件，实时更新当前 assistant 消息
function handleEvent(assistant, ev) {
  switch (ev.type) {
    case 'token':
      assistant.content += ev.text || ''
      scrollToBottom()
      break
    case 'tool_start':
      assistant.tools.push({ name: ev.name, label: ev.label, status: 'running', summary: '' })
      scrollToBottom()
      break
    case 'tool_done': {
      const t = [...assistant.tools].reverse().find(x => x.name === ev.name && x.status === 'running')
      if (t) { t.status = 'done'; t.summary = ev.summary || '' }
      else assistant.tools.push({ name: ev.name, label: ev.label, status: 'done', summary: ev.summary || '' })
      break
    }
    case 'confirm':
      assistant.actions.push({
        tool: ev.tool, label: ev.label, preview: ev.preview || '',
        args: ev.args || {}, status: 'pending', result: '', _loading: false,
      })
      scrollToBottom()
      break
    case 'reset':
      // 后端检测到“假装完成写操作”，清空已输出文本，准备重答
      assistant.content = ''
      break
    case 'error':
      assistant.content += (assistant.content ? '\n\n' : '') + '⚠️ ' + (ev.message || '出错了')
      break
  }
}

// 用户确认执行某个写操作
async function onConfirm(action) {
  if (action._loading || action.status !== 'pending') return
  action._loading = true
  try {
    const res = await aiApi.confirmAction(action.tool, action.args)
    action.status = 'confirmed'
    action.result = res.data?.result || '已执行'
    toast.success('操作已执行')
  } catch (e) {
    toast.error(typeof e === 'string' ? e : '执行失败')
  } finally {
    action._loading = false
  }
}

async function onCancel(action) {
  if (action.status !== 'pending') return
  action.status = 'cancelled'
  try { await aiApi.cancelAction() } catch {}
}

async function copyText(text) {
  const ok = await copyToClipboard(text)
  if (ok) toast.success('已复制')
  else    toast.error('复制失败，请手动复制')
}

// 拦截 markdown 气泡内的链接点击：内部路由用 router.push（避免整页刷新）
function handleMdClick(e) {
  const a = e.target.closest('a')
  if (!a) return
  const href = a.getAttribute('href') || ''
  if (href.startsWith('/')) {
    e.preventDefault()
    e.stopPropagation()
    router.push(href)
  }
  // 外部链接保留原生 target="_blank" 行为，不处理
}

async function loadQuota() {
  try {
    const res = await aiApi.getQuota()
    Object.assign(quota, res.data)
  } catch {}
}

onMounted(() => {
  loadHistory()
  loadQuota()
  scrollToBottom()
})
</script>

<style scoped>
.typing-dot {
  width: 6px;
  height: 6px;
  border-radius: 9999px;
  background: rgb(var(--ink) / 0.5);
  display: inline-block;
  animation: typing 1.2s infinite ease-in-out;
}
@keyframes typing {
  0%, 60%, 100% { transform: translateY(0); opacity: 0.4; }
  30% { transform: translateY(-4px); opacity: 1; }
}
@media (prefers-reduced-motion: reduce) {
  .typing-dot { animation: none; }
}

/* md-body 全局样式已移至 style.css */
</style>
