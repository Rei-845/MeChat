import { defineStore } from 'pinia'
import { ref } from 'vue'
import { useChatStore } from './chat'

export const useWsStore = defineStore('ws', () => {
  const connected = ref(false)
  const ws        = ref(null)
  let   pingTimer = null
  let   seq       = 1

  const handlers = {}

  function on(type, fn) { handlers[type] = fn }

  function connect(token) {
    if (ws.value) { disconnect() }
    const url = `${location.protocol === 'https:' ? 'wss' : 'ws'}://${location.host}/ws?token=${token}`
    const sock = new WebSocket(url)

    sock.onopen = () => {
      connected.value = true
      pingTimer = setInterval(() => send('ping', {}), 30000)
    }

    sock.onmessage = (e) => {
      try {
        const env = JSON.parse(e.data)
        dispatch(env)
      } catch {}
    }

    sock.onclose = () => {
      connected.value = false
      ws.value = null
      clearInterval(pingTimer)
      // 自动重连（5s 后）
      setTimeout(() => connect(token), 5000)
    }

    sock.onerror = () => sock.close()
    ws.value = sock
  }

  function disconnect() {
    clearInterval(pingTimer)
    ws.value?.close()
    ws.value = null
    connected.value = false
  }

  function send(type, data) {
    if (ws.value?.readyState === WebSocket.OPEN) {
      ws.value.send(JSON.stringify({ type, seq: seq++, data }))
      return true
    }
    return false
  }

  function sendMessage(convId, msgType, content) {
    return send('send_msg', { conversation_id: convId, msg_type: msgType, content })
  }

  function dispatch(env) {
    const chatStore = useChatStore()
    switch (env.type) {
      case 'new_msg':
        chatStore.pushMessage(env.data)
        if (handlers.new_msg) handlers.new_msg(env.data)
        break
      case 'msg_ack':
        chatStore.ackMessage(env.data.conversation_id, env.data.msg_id, env.data.created_at)
        break
      case 'msg_recall':
        chatStore.recallMessage(env.data.conversation_id, env.data.msg_id)
        break
      case 'ai_result':
        if (handlers.ai_result) handlers.ai_result(env.data)
        break
      case 'group_dissolve':
      case 'conversation_remove':
        chatStore.removeConversation(env.data.conversation_id)
        if (handlers[env.type]) handlers[env.type](env.data)
        break
      case 'friend_req':
      case 'friend_accept':
        if (handlers[env.type]) handlers[env.type](env.data)
        break
    }
  }

  return { connected, connect, disconnect, send, sendMessage, on }
})
