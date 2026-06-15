import api from './index'

export const summarize    = (convId, count = 50) => api.post('/ai/summarize', { conversation_id: convId, message_count: count })
export const getQuota     = () => api.get('/ai/quota')

// AI 对话历史（后端持久化）
export const getHistory   = () => api.get('/ai/history')
export const clearHistory = () => api.delete('/ai/history')

// 确认执行 Agent 写操作（发消息/发帖/加好友）
export const confirmAction = (tool, args) => api.post('/ai/action/confirm', { tool, args })
// 取消待确认写操作（清掉后端存的确认框）
export const cancelAction  = () => api.post('/ai/action/cancel')

// 通用 SSE 流式请求：POST body JSON → 逐事件回调
async function _streamPost(path, body, { onEvent, signal } = {}) {
  const token = localStorage.getItem('token')
  const res = await fetch(path, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
    },
    body: JSON.stringify(body),
    signal,
  })
  if (res.status === 401) { localStorage.removeItem('token'); window.location.href = '/auth'; return }
  if (!res.ok || !res.body) throw new Error('请求失败：' + res.status)
  const reader  = res.body.getReader()
  const decoder = new TextDecoder()
  let buffer = ''
  while (true) {
    const { value, done } = await reader.read()
    if (done) break
    buffer += decoder.decode(value, { stream: true })
    let idx
    while ((idx = buffer.indexOf('\n\n')) >= 0) {
      const raw = buffer.slice(0, idx).trim()
      buffer = buffer.slice(idx + 2)
      if (!raw.startsWith('data:')) continue
      const payload = raw.slice(5).trim()
      if (!payload) continue
      try { onEvent?.(JSON.parse(payload)) } catch { }
    }
  }
}

// streamDraftMessage 流式帮写聊天消息（全用户可用）
export const streamDraftMessage = (draft, context, opts) =>
  _streamPost('/api/v1/ai/draft-message/stream', { draft, context }, opts)

// streamDraftPost 流式帮写帖子（全用户可用）
export const streamDraftPost = (keywords, opts) =>
  _streamPost('/api/v1/ai/draft-post/stream', { keywords }, opts)

// streamChat 流式对话（SSE）。只发当前这条消息，历史与记忆窗口由后端按 VIP 决定。
// 逐事件回调 onEvent({ type, ... })，事件类型：token / tool_start / tool_done / confirm / reset / error / done
export const streamChat = (text, opts) =>
  _streamPost('/api/v1/ai/chat/stream', { text }, opts)
