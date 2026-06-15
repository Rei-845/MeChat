import { ref } from 'vue'

// 全局单例：默认浅色（Telegram Light），可切换到夜间。
const isDark = ref(localStorage.getItem('theme') === 'dark')

function apply() {
  const el = document.documentElement
  if (isDark.value) {
    el.classList.add('dark')
    localStorage.setItem('theme', 'dark')
  } else {
    el.classList.remove('dark')
    localStorage.setItem('theme', 'light')
  }
}

// 启动时立即应用一次
apply()

export function useTheme() {
  function toggleTheme() {
    isDark.value = !isDark.value
    apply()
  }
  return { isDark, toggleTheme }
}
