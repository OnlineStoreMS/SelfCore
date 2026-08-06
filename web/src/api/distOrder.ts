import client, { unwrap, type PageData } from './client'

export interface DistOrderItem {
  id?: number
  skuId: number
  offerId?: number
  productName?: string
  skuCode?: string
  skuSpecs?: string
  picUrl?: string
  distributorSkuCode?: string
  qty: number
  saleUnitPrice?: number
  saleAmount?: number
  unitPrice: number
  lineAmount?: number
  receivedQty?: number
  refSoId?: number
  refOrderNo?: string
  cancelled?: boolean
  remark?: string
}

export interface DistOrder {
  id: number
  distNo: string
  distributorId: number
  distributorName?: string
  distributorCode?: string
  status: string
  totalAmount: number
  saleAmount?: number
  currency: string
  expectedArrivalDate?: string
  warehouseId?: number
  fulfillmentType: string
  refSoId?: number
  refTraceId?: string
  buyerId?: number
  buyerName?: string
  payStatus: string
  remark?: string
  orderedAt?: string
  completedAt?: string
  createdAt?: string
  items: DistOrderItem[]
}

export interface DistOrderListItem {
  id: number
  distNo: string
  distributorId: number
  distributorName?: string
  status: string
  payStatus: string
  fulfillmentType: string
  totalAmount: number
  currency: string
  itemCount: number
  skuSpecs?: string
  refSoId?: number
  refTraceId?: string
  orderedAt?: string
  createdAt: string
}

export interface DistOrderInput {
  distributorId?: number
  fulfillmentType?: string
  currency?: string
  expectedArrivalDate?: string
  warehouseId?: number
  refSoId?: number
  refTraceId?: string
  orderedAt?: string
  saleAmount?: number
  remark?: string
  items: Array<{
    skuId?: number
    offerId?: number
    productName?: string
    skuCode?: string
    skuSpecs?: string
    picUrl?: string
    distributorSkuCode?: string
    qty: number
    saleUnitPrice?: number
    saleAmount?: number
    unitPrice?: number
    remark?: string
  }>
}

export const DIST_STATUS_MAP: Record<string, { label: string; type: '' | 'success' | 'warning' | 'info' | 'danger' }> = {
  draft: { label: '草稿', type: 'info' },
  confirmed: { label: '已确认', type: '' },
  paid: { label: '已收款', type: 'warning' },
  partial_shipped: { label: '部分发货', type: '' },
  shipped: { label: '已发货', type: 'warning' },
  partial_received: { label: '部分到货', type: '' },
  completed: { label: '已完成', type: 'success' },
  cancelled: { label: '已取消', type: 'danger' },
}

export const PAY_STATUS_MAP: Record<string, string> = {
  unpaid: '未收款',
  partial: '部分收款',
  paid: '已收款',
}

export const FULFILLMENT_TYPE_MAP: Record<string, string> = {
  wholesale: '批发',
  dropship: '分销直发',
}

export async function fetchDistOrders(params: {
  status?: string
  /** 多状态，逗号分隔，优先于 status */
  statuses?: string
  /** 收款状态，逗号分隔 unpaid|partial|paid */
  payStatus?: string
  /** 排除状态，逗号分隔 */
  excludeStatuses?: string
  fulfillmentType?: string
  distributorId?: number
  refSoId?: number
  refTraceId?: string
  keyword?: string
  createdAtStart?: string
  createdAtEnd?: string
  orderedAtStart?: string
  orderedAtEnd?: string
  sortBy?: string
  sortOrder?: string
  page?: number
  pageSize?: number
} = {}) {
  const res = await client.get('/dist-orders', { params })
  return unwrap<PageData<DistOrderListItem>>(res)
}

export async function fetchDistOrder(id: number) {
  return unwrap<DistOrder>(await client.get(`/dist-orders/${id}`))
}

export async function createDistOrder(data: DistOrderInput) {
  return unwrap<DistOrder>(await client.post('/dist-orders', data))
}

export async function updateDistOrder(id: number, data: DistOrderInput) {
  return unwrap<DistOrder>(await client.put(`/dist-orders/${id}`, data))
}

export async function deleteDistOrder(id: number) {
  return unwrap(await client.delete(`/dist-orders/${id}`))
}

export async function updateDistOrderItemPrices(
  id: number,
  items: { itemId: number; unitPrice: number }[],
) {
  return unwrap<DistOrder>(await client.put(`/dist-orders/${id}/item-prices`, { items }))
}

export async function submitDistOrder(id: number) {
  return unwrap<DistOrder>(await client.post(`/dist-orders/${id}/submit`))
}

export async function markDistOrderPaid(id: number) {
  return unwrap<DistOrder>(await client.post(`/dist-orders/${id}/mark-paid`))
}

export async function completeDistOrder(id: number) {
  return unwrap<DistOrder>(await client.post(`/dist-orders/${id}/complete`))
}

export async function cancelDistOrder(id: number) {
  return unwrap<DistOrder>(await client.post(`/dist-orders/${id}/cancel`))
}

export async function detachSalesOrder(data: {
  distNo: string
  orderNo?: string
  soId?: number
  reason?: string
}) {
  return unwrap<{ distOrder: DistOrder; unlinkWarning?: string }>(
    await client.post('/dist-orders/detach-sales-order', data),
  )
}

export async function mergeDistOrders(data: { sourceDistOrderIds: number[]; targetDistOrderId?: number }) {
  return unwrap<DistOrder & { mergedFromDistNos?: string[]; relinked?: number }>(
    await client.post('/dist-orders/merge', data),
  )
}
