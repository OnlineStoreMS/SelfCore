import type { OrderBrief, OrderItemBrief, OrderAddressBrief } from '../api/order'

function pad(n: number) {
  return String(n).padStart(2, '0')
}

/** 订单解密复制用：2026 07/11 12:42（与订单中心一致） */
export function formatOrderCopyDateTime(d: Date): string {
  return `${d.getFullYear()} ${pad(d.getMonth() + 1)}/${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`
}

export function formatAddress(addr?: OrderAddressBrief | null) {
  if (!addr) return '-'
  if (addr.fullText?.trim()) return addr.fullText.trim()
  const parts = [addr.name, addr.phone, addr.province, addr.city, addr.district, addr.address].filter(
    (s) => s?.trim(),
  )
  return parts.join(' ') || '-'
}

/** 是否仍为脱敏地址（含 *） */
export function isMaskedReceiver(order: Pick<OrderBrief, 'buyerName' | 'buyerPhone' | 'address'>) {
  const text = [order.buyerName, order.buyerPhone, formatAddress(order.address)].join(' ')
  return /[*＊]/.test(text)
}

export function formatOrderCopyGoodsLines(items?: OrderItemBrief[]) {
  return (items || [])
    .map((it) => {
      const spec = (it.skuSpecs || it.productName || it.skuCode || '').trim()
      if (!spec) return ''
      const num = it.quantity && it.quantity > 0 ? it.quantity : 1
      return `${spec} x${num}`
    })
    .filter(Boolean)
    .join('\n')
}

/** 与订单中心一致：时间 + 空行 + 收件信息 + --- + 规格行 */
export function buildOrderCopyText(order: OrderBrief, now = new Date()) {
  const body = buildOrderCopyBody(order)
  if (!body) return ''
  return [formatOrderCopyDateTime(now), '', body].join('\n')
}

/** 单笔内容（不含顶部时间）：收件信息 + --- + 规格行 */
export function buildOrderCopyBody(order: OrderBrief) {
  const address = formatAddress(order.address)
  const goodsBlock = formatOrderCopyGoodsLines(order.items)
  const lines = [address === '-' ? '' : address, '---']
  if (goodsBlock) lines.push(goodsBlock)
  return lines.join('\n')
}

/**
 * 多销售单复制：顶部一个时间，各单用【1】【2】标注。
 * 单条时不带序号，与订单中心单笔格式一致。
 */
export function buildMultiOrderCopyText(orders: OrderBrief[], now = new Date()) {
  const bodies = orders.map((o) => buildOrderCopyBody(o)).filter((t) => t.trim())
  if (!bodies.length) return ''
  if (bodies.length === 1) {
    return [formatOrderCopyDateTime(now), '', bodies[0]].join('\n')
  }
  const numbered = bodies.map((body, i) => `【${i + 1}】\n${body}`).join('\n\n')
  return [formatOrderCopyDateTime(now), '', numbered].join('\n')
}

export function canDecryptOrder(order: Pick<OrderBrief, 'sourceChannel' | 'platformSysTid'>) {
  return order.sourceChannel === 'kdzs' && !!order.platformSysTid?.trim()
}
