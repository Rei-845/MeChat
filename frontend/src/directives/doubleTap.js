// v-double-tap="handler" —— 仅触摸设备（移动端）：检测 300ms 内、位置相近的两次轻触，
// 触发 handler({ x, y })（x/y 为第二次轻触的屏幕坐标，供红心动画定位）。
//
// 会忽略落在 button / a / input / textarea / [data-no-dbltap] 上的轻触，
// 避免与点击、导航、点赞按钮本身冲突。桌面端（鼠标）不触发，不影响既有交互。
export const vDoubleTap = {
  mounted(el, binding) {
    el._dt = { last: 0, x: 0, y: 0 }
    el._dtHandler = (e) => {
      // 交互子元素上的轻触不参与双击判定（防止与按钮/链接冲突）
      if (e.target.closest('button, a, input, textarea, [data-no-dbltap]')) {
        el._dt.last = 0
        return
      }
      const t = e.changedTouches && e.changedTouches[0]
      if (!t) return
      const now = Date.now()
      const x = t.clientX
      const y = t.clientY
      const s = el._dt
      if (now - s.last < 300 && Math.abs(x - s.x) < 30 && Math.abs(y - s.y) < 30) {
        if (typeof binding.value === 'function') binding.value({ x, y })
        s.last = 0 // 复位，避免三连击重复触发
      } else {
        s.last = now
        s.x = x
        s.y = y
      }
    }
    el.addEventListener('touchend', el._dtHandler, { passive: true })
  },
  unmounted(el) {
    if (el._dtHandler) el.removeEventListener('touchend', el._dtHandler)
    delete el._dtHandler
    delete el._dt
  },
}
