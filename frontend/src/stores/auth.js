import { defineStore, getActivePinia } from 'pinia'
import { ref, computed } from 'vue'
import * as authApi from '@/api/auth'

function resetAllStores() {
  const pinia = getActivePinia()
  if (!pinia) return
  Object.values(pinia.state.value).forEach((state) => {
    // 逐字段清空（conversations、messages、friends 等列表型数据）
    Object.keys(state).forEach((k) => {
      const v = state[k]
      if (Array.isArray(v)) state[k] = []
      else if (v !== null && typeof v === 'object' && !Array.isArray(v)) state[k] = {}
    })
  })
}

export const useAuthStore = defineStore('auth', () => {
  const token = ref(localStorage.getItem('token') || '')
  const user  = ref(null)

  const isLoggedIn = computed(() => !!token.value)

  async function fetchMe() {
    if (!token.value) return
    try {
      const res = await authApi.getMe()
      user.value = res.data
    } catch {}
  }

  async function loginAction(payload) {
    const res = await authApi.login(payload)
    resetAllStores()
    token.value = res.data.token
    user.value  = res.data.user
    localStorage.setItem('token', res.data.token)
  }

  async function registerAction(data) {
    const res = await authApi.register(data)
    resetAllStores()
    token.value = res.data.token
    user.value  = res.data.user
    localStorage.setItem('token', res.data.token)
  }

  async function logoutAction() {
    try { await authApi.logout() } catch {}
    localStorage.removeItem('token')
    resetAllStores()
    token.value = ''
    user.value  = null
  }

  function updateUser(data) {
    user.value = { ...user.value, ...data }
  }

  return { token, user, isLoggedIn, fetchMe, loginAction, registerAction, logoutAction, updateUser }
})
