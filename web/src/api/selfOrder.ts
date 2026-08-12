import client, { unwrap, type PageData } from './client'

export interface SelfOrderListItem {
  id: number
  soNo: string
  status: string
  warehouseId: number
  refSoId: number
  refTraceId: string
  saleAmount: number
  costAmount: number
  payStatus?: string
  paidAt?: string
  sourceChannel?: string
  platform?: string
  shopName?: string
  buyerRemark?: string
  sellerRemark?: string
  fenFaRemark?: string
  printerRemark?: string
  skuSpecs?: string
  itemCount: number
  stockDeducted: boolean
  stockError: string
  orderedAt?: string
  shippedAt?: string
  createdAt: string
}

export interface SelfOrderItem {
  id: number
  pimSkuId: number
  skuCode: string
  productName: string
  skuSpecs: string
  picUrl: string
  qty: number
  saleUnitPrice: number
  saleAmount: number
  invSkuId: number
  invSkuCode: string
  costUnitPrice: number
  costAmount: number
  refSoId: number
  refOrderNo: string
  remark: string
}

export interface SelfOrderDetail {
  id: number
  soNo: string
  status: string
  warehouseId: number
  refSoId: number
  refTraceId: string
  saleAmount: number
  costAmount: number
  payStatus?: string
  paidAt?: string
  buyerName: string
  buyerPhone: string
  address: string
  remark: string
  sourceChannel?: string
  platform?: string
  shopName?: string
  buyerRemark?: string
  sellerRemark?: string
  fenFaRemark?: string
  printerRemark?: string
  stockDeducted: boolean
  stockError: string
  orderedAt?: string
  shippedAt?: string
  completedAt?: string
  createdAt: string
  updatedAt: string
  items: SelfOrderItem[]
}

export interface SelfShipment {
  id: number
  selfOrderId?: number
  shipmentNo: string
  status: string
  carrierCode?: string
  carrierName: string
  trackingNo: string
  shippedAt?: string
  expectedArrivalDate?: string
  deliveredAt?: string
  callbackOk: boolean
  stockDeducted?: boolean
  receiverName: string
  receiverPhone: string
  receiverAddress: string
  remark: string
  items?: { id?: number; selfOrderItemId: number; qty: number }[]
  createdAt?: string
}

export interface BindInvSkuInput {
  invSkuId: number
  invSkuCode?: string
  costUnitPrice?: number
}

export interface SelfShipInput {
  expressCompany: string
  expressNo: string
  remark?: string
  callback?: boolean
}

export interface WarehouseSku {
  id: number
  skuCode: string
  name: string
  pickName?: string
  lastPurchasePrice: number
}

export interface Warehouse {
  id: number
  code: string
  name: string
  status: number
  isDefault: number
}

/** 单据状态展示：已下单 / 已完成 / 已取消（内部 paid/发货中等均归为已下单） */
export const SELF_ORDER_STATUS_MAP: Record<
  string,
  { label: string; type: '' | 'success' | 'warning' | 'info' | 'danger' }
> = {
  draft: { label: '已下单', type: 'warning' },
  ordered: { label: '已下单', type: 'warning' },
  confirmed: { label: '已下单', type: 'warning' },
  paid: { label: '已下单', type: 'warning' },
  partial_shipped: { label: '已下单', type: 'warning' },
  shipped: { label: '已下单', type: 'warning' },
  completed: { label: '已完成', type: 'success' },
  cancelled: { label: '已取消', type: 'info' },
}

/** 单据状态筛选项 */
export const SELF_ORDER_DOC_STATUS_OPTIONS = [
  { value: 'ordered', label: '已下单' },
  { value: 'completed', label: '已完成' },
  { value: 'cancelled', label: '已取消' },
] as const

/** 发货状态：待发货 / 部分发货 / 已发货 */
export const SELF_SHIP_STATUS_MAP: Record<
  string,
  { label: string; type: '' | 'success' | 'warning' | 'info' | 'danger' }
> = {
  wait_ship: { label: '待发货', type: 'warning' },
  partial_shipped: { label: '部分发货', type: 'warning' },
  shipped: { label: '已发货', type: 'success' },
}

/** 付款状态筛选项 */
export const SELF_PAY_STATUS_OPTIONS = [
  { value: 'unpaid', label: '未付款' },
  { value: 'partial', label: '部分付款' },
  { value: 'paid', label: '已付清' },
] as const

export function deriveSelfDocStatus(status?: string): string {
  const s = (status || '').trim()
  if (s === 'completed') return 'completed'
  if (s === 'cancelled') return 'cancelled'
  if (!s) return ''
  return 'ordered'
}

export function deriveSelfShipStatus(status?: string): string {
  switch ((status || '').trim()) {
    case 'partial_shipped':
      return 'partial_shipped'
    case 'shipped':
    case 'completed':
      return 'shipped'
    case 'draft':
    case 'ordered':
    case 'paid':
    case 'confirmed':
      return 'wait_ship'
    default:
      return ''
  }
}

export async function listSelfOrders(params: {
  status?: string
  statuses?: string
  excludeStatuses?: string
  payStatus?: string
  shipStatus?: string
  refSoId?: number
  keyword?: string
  createdAtStart?: string
  createdAtEnd?: string
  orderedAtStart?: string
  orderedAtEnd?: string
  shippedAtStart?: string
  shippedAtEnd?: string
  page?: number
  pageSize?: number
} = {}) {
  const res = await client.get('/self-orders', { params })
  return unwrap<PageData<SelfOrderListItem>>(res)
}

export async function getSelfOrder(id: number) {
  return unwrap<SelfOrderDetail>(await client.get(`/self-orders/${id}`))
}

export async function shipSelfOrder(id: number, data: SelfShipInput) {
  return unwrap<SelfOrderDetail>(await client.post(`/self-orders/${id}/ship`, data))
}

export async function retryCallback(id: number, shipmentId?: number) {
  return unwrap<SelfOrderDetail>(
    await client.post(`/self-orders/${id}/retry-callback`, {}, {
      params: shipmentId ? { shipmentId } : undefined,
    }),
  )
}

export async function retryStock(id: number) {
  return unwrap<SelfOrderDetail>(await client.post(`/self-orders/${id}/retry-stock`))
}

export async function submitSelfOrder(id: number) {
  return unwrap<SelfOrderDetail>(await client.post(`/self-orders/${id}/submit`))
}

export async function markSelfOrderPaid(id: number) {
  return unwrap<SelfOrderDetail>(await client.post(`/self-orders/${id}/mark-paid`))
}

export async function completeSelfOrder(id: number) {
  return unwrap<SelfOrderDetail>(await client.post(`/self-orders/${id}/complete`))
}

export async function cancelSelfOrder(id: number) {
  return unwrap<SelfOrderDetail>(await client.post(`/self-orders/${id}/cancel`))
}

export async function deleteSelfOrder(id: number) {
  return unwrap(await client.delete(`/self-orders/${id}`))
}

export async function bindInvSku(itemId: number, data: BindInvSkuInput) {
  return unwrap<SelfOrderDetail>(await client.put(`/self-order-items/${itemId}/inv-sku`, data))
}

export async function updateItemCost(itemId: number, costUnitPrice: number) {
  return unwrap<SelfOrderDetail>(await client.put(`/self-order-items/${itemId}/cost`, { costUnitPrice }))
}

export async function listSelfShipments(selfOrderId: number) {
  return unwrap<SelfShipment[]>(await client.get(`/self-orders/${selfOrderId}/shipments`))
}

export async function searchWarehouseSkus(params: {
  keyword?: string
  page?: number
  pageSize?: number
} = {}) {
  const res = await client.get('/warehouse-skus/search', { params })
  return unwrap<PageData<WarehouseSku>>(res)
}

export async function listWarehouses(params: {
  keyword?: string
  page?: number
  pageSize?: number
} = {}) {
  const res = await client.get('/warehouses', { params })
  return unwrap<PageData<Warehouse>>(res)
}
