import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import * as friendApi from '@/api/friend'

export const useFriendStore = defineStore('friend', () => {
  const friends  = ref([])
  const requests = ref([])

  const pendingCount = computed(() => requests.value.length)

  async function loadFriends() {
    const res = await friendApi.getFriends()
    friends.value = res.data.list || []
  }

  async function loadRequests() {
    const res = await friendApi.getFriendRequests()
    requests.value = res.data.list || []
  }

  async function sendRequest(userId, message = '') {
    await friendApi.sendFriendRequest(userId, message)
  }

  async function handleRequest(id, action) {
    await friendApi.handleRequest(id, action)
    requests.value = requests.value.filter(r => r.id !== id)
    if (action === 'accept') await loadFriends()
  }

  async function removeFriend(userId) {
    await friendApi.deleteFriend(userId)
    friends.value = friends.value.filter(f => f.user_id !== userId)
  }

  function isFriend(userId) {
    return friends.value.some(f => f.user_id === userId)
  }

  return {
    friends, requests, pendingCount,
    loadFriends, loadRequests, sendRequest, handleRequest, removeFriend, isFriend,
  }
})
