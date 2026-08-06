import client, { unwrap, type PageData } from './client'

export interface OrderItemBrief {
  id: number
  skuId?: number
  skuCode?: string
  productName?: string
  skuSpecs?: string
  picUrl?: string
  quantity: number
  price?: number
  totalAmount?: number
}

export interface OrderAddressBrief {
  name?: string
  phone?: string
  province?: string
  city?: string
  district?: string
  address?: string
  fullText?: string
}

export interface OrderBrief {
  id: number
  orderNo: string
  sourceChannel?: string
  platform?: string
  platformOrderId?: string
  platformSysTid?: string
  shopName?: string
  buyerName?: string
  buyerNick?: string
  buyerPhone?: string
  status?: string
  shipStatus?: string
  allocType?: string
  totalAmount?: number
  payAmount?: number
  remark?: string
  sellerRemark?: string
  fenFaRemark?: string
  printerRemark?: string
  allocRemark?: string
  platformStatus?: string
  platformStatusText?: string
  ecommerceStatus?: string
  ecommerceStatusText?: string
  afterSaleStatus?: string
  afterSaleStatusText?: string
  agentType?: number
  payTime?: string
  orderedAt?: string
  shippedAt?: string
  createdAt?: string
  address?: OrderAddressBrief
  items?: OrderItemBrief[]
}

const sourceLabels: Record<string, string> = {
  kdzs: '电商',
  wx_mall: '小程序',
  store: '门店',
  xianyu: '闲鱼',
  manual: '手工订单',
}

const statusLabels: Record<string, string> = {
  pending_payment: '待付款',
  pending_alloc: '待分配',
  pending_ship: '待分配',
  allocated: '已分配',
  purchasing: '采购中',
  shipped: '已发货',
  partial_ship: '部分发货',
  completed: '已完成',
  closed: '已关闭',
}

const shipStatusLabels: Record<string, string> = {
  wait_ship: '待发货',
  shipped: '已发货',
}

export function labelSource(v?: string) {
  return (v && sourceLabels[v]) || v || '-'
}

export function labelStatus(v?: string) {
  return (v && statusLabels[v]) || v || '-'
}

export function labelShipStatus(v?: string) {
  return (v && shipStatusLabels[v]) || v || '-'
}

export function formatPlatformShop(row: Pick<OrderBrief, 'platform' | 'shopName'>) {
  const p = (row.platform || '').trim()
  const shop = (row.shopName || '').trim()
  if (p && shop) return `${p} / ${shop}`
  if (p) return p
  if (shop) return shop
  return '-'
}

export function formatDateTime(v?: string | null) {
  if (!v) return '-'
  const normalized = String(v).trim().replace(' ', 'T')
  const d = new Date(normalized)
  if (Number.isNaN(d.getTime())) {
    if (/^\d{4}-\d{2}-\d{2}/.test(String(v))) {
      return String(v).replace('T', ' ').replace(/\.\d+/, '').replace(/([+-]\d{2}:\d{2}|Z)$/, '').trim()
    }
    return String(v)
  }
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`
}

export function formatOrderReceiverAddress(addr?: OrderAddressBrief | null) {
  if (!addr) return ''
  if (addr.fullText?.trim()) return addr.fullText.trim()
  return [addr.province, addr.city, addr.district, addr.address].filter((s) => s?.trim()).join('')
}

export function formatAddress(addr?: OrderAddressBrief | null) {
  if (!addr) return '-'
  if (addr.fullText?.trim()) return addr.fullText.trim()
  const parts = [addr.name, addr.phone, addr.province, addr.city, addr.district, addr.address].filter((s) => s?.trim())
  return parts.join(' ') || '-'
}

export function formatRemarkLines(row: Pick<OrderBrief, 'remark' | 'sellerRemark' | 'fenFaRemark' | 'printerRemark' | 'allocRemark'>) {
  const lines: string[] = []
  const push = (label: string, v?: string) => {
    const t = (v || '').trim()
    if (t) lines.push(`${label}：${t}`)
  }
  push('买家', row.remark)
  push('卖家', row.sellerRemark)
  push('分发', row.fenFaRemark)
  push('打印', row.printerRemark)
  push('分配', row.allocRemark)
  return lines
}

export async function searchOrders(params: {
  keyword?: string
  page?: number
  pageSize?: number
  sourceChannel?: string
  status?: string
  shipStatus?: string
  allocType?: string
  platform?: string
  salesChannel?: string
  orderedAtStart?: string
  orderedAtEnd?: string
  shippedAtStart?: string
  shippedAtEnd?: string
  payTimeStart?: string
  payTimeEnd?: string
}) {
  const res = await client.get('/orders/search', { params })
  return unwrap<PageData<OrderBrief>>(res)
}

export async function fetchOrder(id: number) {
  return unwrap<OrderBrief>(await client.get(`/orders/${id}`))
}

export async function decryptOrders(orderIds: number[]) {
  return unwrap<{ items: OrderBrief[]; success: number }>(
    await client.post('/orders/decrypt', { orderIds }),
  )
}

export async function shipOrder(
  id: number,
  body: { expressCompany: string; expressNo: string; remark?: string; callback?: boolean },
) {
  return unwrap<OrderBrief>(await client.post(`/orders/${id}/ship`, body, { timeout: 180000 }))
}
