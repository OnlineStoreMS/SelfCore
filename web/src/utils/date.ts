function pad(n: number) {
  return String(n).padStart(2, '0')
}

export function formatDateTimeLocal(d: Date) {
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`
}

/** 写入 URL：用 T 分隔，避免空格变成 + */
export function toQueryDateTime(s: string) {
  return s.trim().replace(' ', 'T')
}

/** 从 URL 读回：兼容空格、+、T */
export function fromQueryDateTime(s: string) {
  return s.trim().replace(/\+/g, ' ').replace('T', ' ')
}

/** 今天 00:00:00 ~ 23:59:59（与订单中心一致） */
export function todayDateTimeRange(): [string, string] {
  const start = new Date()
  start.setHours(0, 0, 0, 0)
  const end = new Date()
  end.setHours(23, 59, 59, 0)
  return [formatDateTimeLocal(start), formatDateTimeLocal(end)]
}

/** 近 7 天（含今天） */
export function last7DaysDateTimeRange(): [string, string] {
  const end = new Date()
  end.setHours(23, 59, 59, 0)
  const start = new Date()
  start.setDate(start.getDate() - 6)
  start.setHours(0, 0, 0, 0)
  return [formatDateTimeLocal(start), formatDateTimeLocal(end)]
}

/** 查询参数中的日期/时间范围规范化 */
export function rangeFromQueryDates(startRaw?: unknown, endRaw?: unknown): [string, string] | null {
  if (typeof startRaw !== 'string' || typeof endRaw !== 'string' || !startRaw || !endRaw) return null
  let start = fromQueryDateTime(startRaw)
  let end = fromQueryDateTime(endRaw)
  if (start.length <= 10) start = `${start} 00:00:00`
  if (end.length <= 10) end = `${end} 23:59:59`
  return [start, end]
}

/** 与订单中心订单列表一致的快捷选项 */
/** 自定义选日期时默认起止时刻（el-date-picker datetimerange 的 default-time） */
export const dateRangeDefaultTime: [Date, Date] = [
  new Date(2000, 0, 1, 0, 0, 0),
  new Date(2000, 0, 1, 23, 59, 59),
]

export const dateShortcuts = [
  {
    text: '今天',
    value: () => {
      const start = new Date()
      start.setHours(0, 0, 0, 0)
      const end = new Date()
      end.setHours(23, 59, 59, 0)
      return [start, end]
    },
  },
  {
    text: '近7天',
    value: () => {
      const end = new Date()
      end.setHours(23, 59, 59, 0)
      const start = new Date()
      start.setDate(start.getDate() - 6)
      start.setHours(0, 0, 0, 0)
      return [start, end]
    },
  },
  {
    text: '近30天',
    value: () => {
      const end = new Date()
      end.setHours(23, 59, 59, 0)
      const start = new Date()
      start.setDate(start.getDate() - 29)
      start.setHours(0, 0, 0, 0)
      return [start, end]
    },
  },
]
