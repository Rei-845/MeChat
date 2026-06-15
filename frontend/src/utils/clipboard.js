// 复制文本到剪贴板，兼容非安全上下文（http/局域网 IP 访问时 navigator.clipboard 不可用）
export async function copyText(text) {
  // 优先用现代 API（需安全上下文：https 或 localhost）
  if (navigator.clipboard && window.isSecureContext) {
    try {
      await navigator.clipboard.writeText(text)
      return true
    } catch { /* 落到下面的兜底方案 */ }
  }
  // 兜底：临时 textarea + execCommand
  try {
    const ta = document.createElement('textarea')
    ta.value = text
    ta.style.position = 'fixed'
    ta.style.top = '-9999px'
    ta.style.opacity = '0'
    document.body.appendChild(ta)
    ta.focus()
    ta.select()
    const ok = document.execCommand('copy')
    document.body.removeChild(ta)
    return ok
  } catch {
    return false
  }
}
