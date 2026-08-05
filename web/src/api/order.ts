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
  totalAmount?: number
  payAmount?: number
  remark?: string
  sellerRemark?: string
  orderedAt?: string
  createdAt?: string
  address?: OrderAddressBrief
  items?: OrderItemBrief[]
}

export function formatOrderReceiverAddress(addr?: OrderAddressBrief | null) {
  if (!addr) return ''
  if (addr.fullText?.trim()) return addr.fullText.trim()
  return [addr.province, addr.city, addr.district, addr.address].filter((s) => s?.trim()).join('')
}

export async function searchOrders(params: {
  keyword: string
  page?: number
  pageSize?: number
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
  // 回传快递助手可能较慢（查单+发货+核对），避免默认 30s 超时中断
  return unwrap<OrderBrief>(await client.post(`/orders/${id}/ship`, body, { timeout: 180000 }))
}
