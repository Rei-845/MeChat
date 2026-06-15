// heartPop(x, y) —— 在屏幕坐标处弹出一颗放大上浮的红心，作为移动端双击点赞的视觉反馈。
// 纯 DOM + Web Animations API，无需任何组件状态或全局样式。
const HEART_PATH =
  'M19 14c1.49-1.46 3-3.21 3-5.5A5.5 5.5 0 0 0 16.5 3c-1.76 0-3 .5-4.5 2-1.5-1.5-2.74-2-4.5-2A5.5 5.5 0 0 0 2 8.5c0 2.29 1.49 4.04 3 5.5l7 7Z'

export function heartPop(x, y) {
  const NS = 'http://www.w3.org/2000/svg'
  const svg = document.createElementNS(NS, 'svg')
  svg.setAttribute('viewBox', '0 0 24 24')
  svg.setAttribute('width', '72')
  svg.setAttribute('height', '72')
  svg.style.cssText =
    `position:fixed;left:${x}px;top:${y}px;z-index:9999;pointer-events:none;` +
    `color:#f43f5e;filter:drop-shadow(0 2px 10px rgba(244,63,94,0.55))`

  const path = document.createElementNS(NS, 'path')
  path.setAttribute('d', HEART_PATH)
  path.setAttribute('fill', 'currentColor')
  svg.appendChild(path)
  document.body.appendChild(svg)

  const anim = svg.animate(
    [
      { transform: 'translate(-50%,-50%) scale(0.3)', opacity: 0 },
      { transform: 'translate(-50%,-50%) scale(1.15)', opacity: 1, offset: 0.3 },
      { transform: 'translate(-50%,-90%) scale(1)', opacity: 0 },
    ],
    { duration: 700, easing: 'cubic-bezier(0.16,1,0.3,1)' },
  )
  anim.onfinish = () => svg.remove()
}
