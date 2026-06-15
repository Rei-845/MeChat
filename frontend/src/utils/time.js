import dayjs from 'dayjs'
import relativeTime from 'dayjs/plugin/relativeTime'
import 'dayjs/locale/zh-cn'

// 相对时间插件 + 中文 locale 在此统一初始化（全局只配一次）
dayjs.extend(relativeTime)
dayjs.locale('zh-cn')

// 时间格式化集中到这里：各处显示口径不同，但逻辑只在这一个文件维护。

// dayjs 原生相对时间（几分钟前 / 几小时前 / 几天前），评论区用
export function formatFromNow(ts) {
  return ts ? dayjs(ts).fromNow() : ''
}

// 会话列表时间戳：今天显示时刻，昨天「昨天」，更早 MM/DD
export function formatChatTime(ts) {
  if (!ts) return ''
  const d = dayjs(ts), now = dayjs()
  if (d.isSame(now, 'day')) return d.format('HH:mm')
  if (d.isSame(now.subtract(1, 'day'), 'day')) return '昨天'
  return d.format('MM/DD')
}

// 消息气泡时间：今天只显示时刻，昨天「昨天 HH:mm」，更早带日期。按日历天比较避免时区误差。
export function formatMsgTime(ts) {
  if (!ts) return ''
  const d = dayjs(ts), now = dayjs()
  const day = d.format('YYYY-MM-DD')
  if (day === now.format('YYYY-MM-DD')) return d.format('HH:mm')
  if (day === now.subtract(1, 'day').format('YYYY-MM-DD')) return '昨天 ' + d.format('HH:mm')
  if (d.format('YYYY') === now.format('YYYY')) return d.format('M月D日 HH:mm')
  return d.format('YYYY/M/D HH:mm')
}

// 相对时间：刚刚 / N分钟前 / N小时前 / MM-DD（帖子、资料浮层）
export function formatRelative(ts) {
  const d = dayjs(ts)
  const diff = dayjs().diff(d, 'minute')
  if (diff < 1) return '刚刚'
  if (diff < 60) return `${diff}分钟前`
  if (diff < 1440) return `${Math.floor(diff / 60)}小时前`
  return d.format('MM-DD')
}

// 帖子卡片底部日期：今天 HH:mm，昨天，今年 M月D日，更早 YY/M/D
export function formatPostDate(ts) {
  if (!ts) return ''
  const d = dayjs(ts), now = dayjs()
  if (d.isSame(now, 'day')) return d.format('HH:mm')
  if (d.isSame(now.subtract(1, 'day'), 'day')) return '昨天'
  if (d.isSame(now, 'year')) return d.format('M月D日')
  return d.format('YY/M/D')
}

// 完整日期：YYYY年MM月DD日（VIP 到期等）
export function formatFullDate(ts) {
  return ts ? dayjs(ts).format('YYYY年MM月DD日') : ''
}
