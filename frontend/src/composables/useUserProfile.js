import { ref } from 'vue'

// 全局单例：当前正在查看的用户 ID（null = 关闭）
const viewingUserId = ref(null)

export function useUserProfile() {
  function openUserProfile(userId) {
    if (!userId) return
    viewingUserId.value = userId
  }
  function closeUserProfile() {
    viewingUserId.value = null
  }
  return { viewingUserId, openUserProfile, closeUserProfile }
}
