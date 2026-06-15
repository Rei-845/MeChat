import { ref } from 'vue'

// 模块级单例，全局共享同一队列
const items = ref([])
let nextId = 0

export function useXPNotify() {
  function showXP(amount) {
    if (!amount || amount <= 0) return
    const id = nextId++
    items.value.push({ id, amount })
    setTimeout(() => {
      items.value = items.value.filter(x => x.id !== id)
    }, 1400)
  }
  return { items, showXP }
}
