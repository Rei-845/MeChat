import { ref } from 'vue'

// 全局单例：任意位置调用 openAddFriend(user, onSuccess) 即可弹出「填写招呼语」浮层。
// 模式与 useUserProfile 一致——状态定义在模块顶层，AddFriendModal 挂载于 App.vue 一次。
// onSuccess 在好友申请发送成功后触发，供调用方更新各自的「已申请」本地态。

const target = ref(null) // { id, nickname } 或 null
let successCb = null

export function useAddFriend() {
  // user 可来自不同结构：有的字段是 id，有的是 user_id
  function openAddFriend(user, onSuccess) {
    if (!user) return
    target.value = { id: user.id ?? user.user_id, nickname: user.nickname || '' }
    successCb = typeof onSuccess === 'function' ? onSuccess : null
  }

  function closeAddFriend() {
    target.value = null
    successCb = null
  }

  // 由 AddFriendModal 在发送成功后调用
  function fireAddFriendSuccess() {
    if (successCb) successCb()
  }

  return { addFriendTarget: target, openAddFriend, closeAddFriend, fireAddFriendSuccess }
}
