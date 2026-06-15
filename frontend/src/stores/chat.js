import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import * as chatApi from '@/api/chat'

export const useChatStore = defineStore('chat', () => {
  const conversations   = ref([])
  const currentConvId   = ref(null)
  const messages        = ref({})   // convId -> []
  const loadingMessages = ref(false)
  const messageHasMore  = ref({})   // convId -> bool，false 表示已到顶部无更多历史

  const currentConv = computed(() =>
    conversations.value.find(c => c.id === currentConvId.value)
  )

  const currentMessages = computed(() =>
    messages.value[currentConvId.value] || []
  )

  async function loadConversations() {
    const res = await chatApi.getConversations()
    conversations.value = res.data.list || []
  }

  async function selectConv(convId) {
    currentConvId.value = convId
    if (!messages.value[convId]) {
      await loadMessages(convId)
    }
    chatApi.markRead(convId, 0).catch(() => {})
    // 清除未读
    const conv = conversations.value.find(c => c.id === convId)
    if (conv) conv.unread_count = 0
  }

  async function loadMessages(convId, beforeId) {
    loadingMessages.value = true
    try {
      const params = { limit: 30 }
      if (beforeId) params.before_id = beforeId
      const res = await chatApi.getMessages(convId, params)
      const list = res.data.list || []
      if (beforeId) {
        messages.value[convId] = [...(messages.value[convId] || []), ...list]
      } else {
        messages.value[convId] = list
      }
      messageHasMore.value[convId] = res.data.has_more || false
      return res.data.has_more
    } finally {
      loadingMessages.value = false
    }
  }

  function pushMessage(msg) {
    const convId = msg.conversation_id
    // 只有该会话的消息已经从 API 加载过（数组存在），才往里追加。
    // 若数组不存在说明用户还没打开过这个会话，跳过追加——
    // 等用户点进去时 selectConv 会正常调用 loadMessages 拉取完整历史。
    if (messages.value[convId]) {
      if (!messages.value[convId].find(m => m.msg_id === msg.msg_id)) {
        messages.value[convId].unshift(msg)
      }
    }
    // 更新会话最后消息
    const conv = conversations.value.find(c => c.id === convId)
    if (conv) {
      conv.last_msg = msg
      conv.updated_at = msg.created_at
      // msg.sync 为上线补推/离线追投的历史消息：未读已在服务端发送时计入，
      // 且已通过会话列表接口（读 Redis 未读数）反映，这里不能再本地 +1，否则重复计数。
      if (convId !== currentConvId.value && !msg.sync) {
        conv.unread_count = (conv.unread_count || 0) + 1
      }
      // 把该会话移到顶部
      conversations.value = [
        conv,
        ...conversations.value.filter(c => c.id !== convId)
      ]
    }
  }

  function ackMessage(convId, msgId, createdAt) {
    // 找该会话里最早一条 _pending（按插入顺序）
    const list = messages.value[convId] || []
    // 消息以 unshift 方式插入，最新在 index 0，所以从末尾找最早的 pending
    for (let i = list.length - 1; i >= 0; i--) {
      if (list[i]._pending) {
        list[i].msg_id     = msgId
        list[i].created_at = createdAt
        list[i]._pending   = false
        break
      }
    }
  }

  function recallMessage(convId, msgId) {
    const msg = (messages.value[convId] || []).find(m => m.msg_id === msgId)
    if (msg) msg.is_recalled = true
  }

  function failMessage(convId, msgId) {
    const msg = (messages.value[convId] || []).find(m => m.msg_id === msgId)
    if (msg) { msg._pending = false; msg._failed = true }
  }

  function removeMessage(convId, msgId) {
    if (messages.value[convId]) {
      messages.value[convId] = messages.value[convId].filter(m => m.msg_id !== msgId)
    }
  }

  async function openOrCreatePrivateChat(userId) {
    const res = await chatApi.createPrivateConv(userId)
    const conv = res.data
    if (!conversations.value.find(c => c.id === conv.id)) {
      conversations.value.unshift(conv)
    }
    await selectConv(conv.id)
    return conv.id
  }

  async function createGroup(name, memberIds) {
    const res = await chatApi.createGroup({ name, member_ids: memberIds })
    const conv = res.data
    if (!conversations.value.find(c => c.id === conv.id)) {
      conversations.value.unshift(conv)
    }
    await selectConv(conv.id)
    return conv.id
  }

  // 收到群解散通知时移除会话
  function removeConversation(convId) {
    conversations.value = conversations.value.filter(c => c.id !== convId)
    if (currentConvId.value === convId) currentConvId.value = null
  }

  const totalUnread = computed(() =>
    conversations.value.reduce((sum, c) => sum + (c.unread_count || 0), 0)
  )

  return {
    conversations, currentConvId, messages, loadingMessages, messageHasMore,
    currentConv, currentMessages, totalUnread,
    loadConversations, selectConv, loadMessages,
    pushMessage, ackMessage, recallMessage, failMessage, removeMessage,
    openOrCreatePrivateChat, createGroup, removeConversation,
  }
})
