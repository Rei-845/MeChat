/**
 * 下拉刷新（Pull-to-Refresh）
 *
 * 用法：
 *   const ptr = usePullToRefresh(() => scrollEl.value, onRefresh)
 *   onMounted(() => ptr.attach())
 *   onUnmounted(() => ptr.detach())
 *
 * 模板里在滚动容器顶部放指示器：
 *   <div class="ptr-indicator" :style="ptr.indicatorStyle.value">
 *     <RefreshCw :class="ptr.refreshing.value && 'animate-spin'" />
 *   </div>
 */
import { ref, computed } from 'vue'

const THRESHOLD = 60   // 触发刷新所需的最小下拉距离（px）
const DAMPING   = 0.45 // 拖拽阻尼，让拉伸感觉更自然

export function usePullToRefresh(getEl, onRefresh) {
  const pullY      = ref(0)     // 当前指示器高度
  const refreshing = ref(false)

  let startY    = 0
  let isPulling = false

  // 指示器旋转角度随拉伸线性增加，达到阈值时转满一圈
  const indicatorStyle = computed(() => ({
    height:     `${pullY.value}px`,
    opacity:    pullY.value / THRESHOLD,
    transform:  `rotate(${(pullY.value / THRESHOLD) * 360}deg)`,
    transition: isPulling ? 'none' : 'height 0.25s ease, opacity 0.25s ease',
  }))

  function handleTouchStart(e) {
    const el = getEl()
    if (!el || el.scrollTop > 2 || refreshing.value) return
    startY    = e.touches[0].clientY
    isPulling = false
  }

  function handleTouchMove(e) {
    const el = getEl()
    if (!el || refreshing.value || !startY) return
    if (el.scrollTop > 2) { startY = 0; return }

    const dy = e.touches[0].clientY - startY
    if (dy <= 0) { pullY.value = 0; return }

    e.preventDefault()   // 阻止原生滚动，确保事件 listener 以 passive:false 注册
    isPulling    = true
    pullY.value  = Math.min(Math.floor(dy * DAMPING), THRESHOLD + 10)
  }

  async function handleTouchEnd() {
    if (!isPulling) return
    isPulling = false

    if (pullY.value >= THRESHOLD * 0.85 && !refreshing.value) {
      refreshing.value = true
      pullY.value      = Math.floor(THRESHOLD * 0.65)  // 刷新中保持悬停
      const MIN_MS = 800  // 指示器至少展示 800ms，太快弹回不丝滑
      const [,] = await Promise.all([onRefresh(), new Promise(r => setTimeout(r, MIN_MS))])
      refreshing.value = false
      pullY.value      = 0
    } else {
      pullY.value = 0
    }
    startY = 0
  }

  function attach() {
    const el = getEl()
    if (!el) return
    el.addEventListener('touchstart', handleTouchStart, { passive: true })
    el.addEventListener('touchmove',  handleTouchMove,  { passive: false })
    el.addEventListener('touchend',   handleTouchEnd,   { passive: true })
  }

  function detach() {
    const el = getEl()
    if (!el) return
    el.removeEventListener('touchstart', handleTouchStart)
    el.removeEventListener('touchmove',  handleTouchMove)
    el.removeEventListener('touchend',   handleTouchEnd)
  }

  return { pullY, refreshing, indicatorStyle, attach, detach }
}
