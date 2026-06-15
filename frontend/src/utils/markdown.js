// 轻量 Markdown 渲染器（无第三方依赖）。
// 先整体转义 HTML，再做块级/行内替换，因此输出对 v-html 是安全的（不会注入原始标签）。
// 支持：标题、粗体/斜体/删除线、行内代码、围栏代码块、有序/无序/任务列表、引用、
//       分割线、链接、图片、表格、换行。

// 代码块占位符前后缀（正文几乎不可能出现，且不含 & < > 不受 HTML 转义影响）
const MARK_L = '@@MDCODE'
const MARK_R = 'ENDMD@@'

function escapeHtml(s) {
  return s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;')
}

// inline 统一处理所有行内格式。
// 处理顺序：① 行内代码 → ② 图片 → ③ 链接（先占位，保护 URL 不被格式规则破坏）
//           → ④ 加粗/斜体/删除线 → ⑤ 还原占位符。
// 关键：URL 中可能含 _ ( * 等字符，必须先占位再格式化，否则斜体/加粗规则会破坏 URL。
function inline(text) {
  const saved = []
  // \x00N\x00 作占位符：escapeHtml 不处理 \x00，格式规则也不含 \x00，绝对安全
  const ph = (html) => { const id = saved.length; saved.push(html); return `\x00${id}\x00` }

  // ① 行内代码（先占位，代码内容不做任何格式化）
  text = text.replace(/`([^`]+)`/g, (_, code) => ph(`<code class="md-code">${code}</code>`))

  // ② 图片（先于链接，避免 ![ 的 ! 被链接规则遗留）
  text = text.replace(/!\[([^\]]*)\]\((https?:\/\/[^\s)]+)\)/g, (_, alt, src) =>
    ph(`<img src="${src}" alt="${alt}" class="md-img" loading="lazy" />`)
  )

  // ③ 链接 [text](url)（占位，URL 不再暴露给后续格式规则）。
  //    url 可不带协议：/ 开头视为站内路由，http(s):// 原样，其余裸域名补 https://，
  //    其它协议（javascript:/data: 等）一律拒绝，原样保留为文本以防 XSS。
  text = text.replace(/\[([^\]]+)\]\(([^\s)]+)\)/g, (whole, linkText, href) => {
    const safe = safeHref(href)
    if (!safe) return whole
    const html = safe.internal
      ? `<a href="${safe.href}" class="md-link md-internal">${linkText}</a>`
      : `<a href="${safe.href}" target="_blank" rel="noopener noreferrer" class="md-link">${linkText}</a>`
    return ph(html)
  })

  // ④ 裸 URL 自动链接（GFM）：http(s):// 或 www. 开头，行内代码/已成链接已被占位，不会重复处理
  text = text.replace(/(^|[\s(])((?:https?:\/\/|www\.)[^\s<)]+)/g, (m, pre, url) => {
    const tail = url.match(/[.,;:!?)）。，；：！？]+$/)  // 末尾标点不计入链接
    const punct = tail ? tail[0] : ''
    const clean = punct ? url.slice(0, -punct.length) : url
    const href = /^www\./i.test(clean) ? 'https://' + clean : clean
    return pre + ph(`<a href="${href}" target="_blank" rel="noopener noreferrer" class="md-link">${clean}</a>`) + punct
  })

  // ⑤ 其余行内格式（此时 URL / 代码块已被占位，不会被误处理）
  text = text.replace(/\*\*([^*]+)\*\*/g, '<strong>$1</strong>')
  text = text.replace(/__([^_]+)__/g, '<strong>$1</strong>')
  text = text.replace(/(^|[^*])\*([^*\n]+)\*/g, '$1<em>$2</em>')
  // 斜体 _ 要求非单词字符边界，防止匹配变量名、路径、占位符中的下划线
  text = text.replace(/(?<![A-Za-z0-9_])_([^_\n]+)_(?![A-Za-z0-9_])/g, '<em>$1</em>')
  text = text.replace(/~~([^~]+)~~/g, '<del>$1</del>')

  // ⑥ 还原占位符
  return text.replace(/\x00(\d+)\x00/g, (_, i) => saved[+i])
}

// safeHref 归一化并校验链接地址。返回 null 表示不安全/不支持，调用方应原样当文本处理。
function safeHref(href) {
  if (href.startsWith('/')) return { href, internal: true }              // 站内路由
  if (/^https?:\/\//i.test(href)) return { href, internal: false }       // 完整外链
  if (/^[a-z][a-z0-9+.-]*:/i.test(href)) return null                     // javascript:/data: 等危险协议，拒绝
  return { href: 'https://' + href, internal: false }                    // 裸域名，补 https://
}

// 表格分隔行：|---|:--:|---|
function isTableSep(line) {
  const t = line.trim()
  if (!t.includes('|')) return false
  return /^\|?\s*:?-{1,}:?\s*(\|\s*:?-{1,}:?\s*)+\|?$/.test(t)
}

function splitRow(line) {
  let t = line.trim()
  if (t.startsWith('|')) t = t.slice(1)
  if (t.endsWith('|')) t = t.slice(0, -1)
  return t.split('|').map(c => c.trim())
}

export function renderMarkdown(src) {
  if (!src) return ''
  let text = escapeHtml(String(src))

  // 抽取围栏代码块，先占位，最后还原（避免内部内容被块级规则误处理）
  const blocks = []
  text = text.replace(/```(\w*)\n?([\s\S]*?)```/g, (_, _lang, code) => {
    const i = blocks.length
    blocks.push('<pre class="md-pre"><code>' + code.replace(/\n$/, '') + '</code></pre>')
    return MARK_L + i + MARK_R
  })

  const lines = text.split('\n')
  const out = []
  let listType = null    // 'ul' | 'ol'
  let para = []

  const flushPara = () => {
    if (para.length) { out.push('<p>' + inline(para.join('<br>')) + '</p>'); para = [] }
  }
  const closeList = () => {
    if (listType) { out.push(`</${listType}>`); listType = null }
  }

  const blockRe = new RegExp('^' + MARK_L + '\\d+' + MARK_R + '$')

  for (let i = 0; i < lines.length; i++) {
    const line = lines[i].replace(/\s+$/, '')
    const trimmed = line.trim()
    let m

    // 表格：当前行含 | 且下一行是分隔行
    if (trimmed.includes('|') && i + 1 < lines.length && isTableSep(lines[i + 1])) {
      flushPara(); closeList()
      const header = splitRow(trimmed)
      i++ // 跳过分隔行
      const rows = []
      while (i + 1 < lines.length && lines[i + 1].includes('|') && lines[i + 1].trim() !== '') {
        i++
        rows.push(splitRow(lines[i]))
      }
      let html = '<table class="md-table"><thead><tr>'
      html += header.map(c => '<th>' + inline(c) + '</th>').join('')
      html += '</tr></thead><tbody>'
      for (const r of rows) {
        html += '<tr>' + header.map((_, ci) => '<td>' + inline(r[ci] || '') + '</td>').join('') + '</tr>'
      }
      html += '</tbody></table>'
      out.push(html)
      continue
    }

    if (blockRe.test(trimmed)) { flushPara(); closeList(); out.push(trimmed); continue }
    if (trimmed === '') { flushPara(); closeList(); continue }
    if ((m = trimmed.match(/^(#{1,6})\s+(.*)$/))) {
      flushPara(); closeList()
      const lv = m[1].length
      out.push(`<h${lv} class="md-h md-h${lv}">` + inline(m[2]) + `</h${lv}>`)
      continue
    }
    if (/^(---|\*\*\*|___)$/.test(trimmed)) { flushPara(); closeList(); out.push('<hr class="md-hr"/>'); continue }
    // 引用：原始 > 被 escapeHtml 转为 &gt;，所以匹配 &gt;
    if ((m = trimmed.match(/^&gt;\s?(.*)$/))) {
      flushPara(); closeList()
      out.push('<blockquote class="md-quote">' + inline(m[1]) + '</blockquote>')
      continue
    }
    if ((m = trimmed.match(/^\d+\.\s+(.*)$/))) {
      flushPara()
      if (listType !== 'ol') { closeList(); out.push('<ol class="md-ol">'); listType = 'ol' }
      out.push('<li>' + inline(m[1]) + '</li>')
      continue
    }
    // 任务列表 - [ ] / - [x]（必须在普通无序列表之前匹配）
    if ((m = trimmed.match(/^[-*+]\s+\[([ xX])\]\s+(.*)$/))) {
      flushPara()
      if (listType !== 'ul') { closeList(); out.push('<ul class="md-ul">'); listType = 'ul' }
      const checked = m[1].toLowerCase() === 'x'
      out.push(`<li class="md-task"><input type="checkbox" disabled${checked ? ' checked' : ''}> ${inline(m[2])}</li>`)
      continue
    }
    if ((m = trimmed.match(/^[-*+]\s+(.*)$/))) {
      flushPara()
      if (listType !== 'ul') { closeList(); out.push('<ul class="md-ul">'); listType = 'ul' }
      out.push('<li>' + inline(m[1]) + '</li>')
      continue
    }
    closeList()
    para.push(trimmed)
  }
  flushPara(); closeList()

  const restoreRe = new RegExp(MARK_L + '(\\d+)' + MARK_R, 'g')
  return out.join('\n').replace(restoreRe, (_, i) => blocks[+i])
}
