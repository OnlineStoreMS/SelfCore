import client, { unwrap } from './client'

export interface ShipmentItem {
  id?: number
  distOrderItemId: number
  skuId?: number
  qty: number
}

export interface Shipment {
  id: number
  distOrderId: number
  shipmentNo: string
  status: string
  carrierCode?: string
  carrierName?: string
  trackingNo?: string
  shippedAt?: string
  expectedArrivalDate?: string
  deliveredAt?: string
  receiverName?: string
  receiverPhone?: string
  receiverAddress?: string
  remark?: string
  items: ShipmentItem[]
  createdAt: string
}

export interface Receipt {
  id: number
  distOrderId: number
  payAmount: number
  payMethod?: string
  payAccount?: string
  payeeAccount?: string
  payeeName?: string
  payStatus: string
  paidAt?: string
  remark?: string
  createdAt: string
}

export interface Attachment {
  id: number
  distOrderId: number
  paymentId?: number
  shipmentId?: number
  fileType: string
  fileName: string
  fileUrl: string
  uploadedBy?: number
  remark?: string
  createdAt: string
}

export const SHIPMENT_STATUS_MAP: Record<string, string> = {
  pending: '待发货',
  shipped: '已发货',
  in_transit: '运输中',
  delivered: '已签收',
  exception: '异常',
}

export const ATTACHMENT_TYPE_MAP: Record<string, string> = {
  dist_sales_order: '分销商销售单',
  payment_screenshot: '收款截图',
  shipment_photo: '物流发货照片',
  contract: '合同',
  other: '其他',
}

export async function fetchShipments(distOrderId: number) {
  return unwrap<Shipment[]>(await client.get(`/dist-orders/${distOrderId}/shipments`))
}

export async function createShipment(distOrderId: number, data: Partial<Shipment> & { items?: ShipmentItem[] }) {
  return unwrap<Shipment>(await client.post(`/dist-orders/${distOrderId}/shipments`, data))
}

export async function syncShipmentsFromOrders(distOrderId: number, refSoId?: number) {
  return unwrap<{ created: number; updated: number; skipped: number; errors?: string[] }>(
    await client.post(`/dist-orders/${distOrderId}/shipments/sync-from-orders`, refSoId ? { refSoId } : {}),
  )
}

export async function updateShipmentStatus(distOrderId: number, shipmentId: number, status: string) {
  return unwrap<Shipment>(await client.patch(`/dist-orders/${distOrderId}/shipments/${shipmentId}/status`, { status }))
}

export async function deleteShipment(distOrderId: number, shipmentId: number) {
  return unwrap(await client.delete(`/dist-orders/${distOrderId}/shipments/${shipmentId}`))
}

export async function fetchReceipts(distOrderId: number) {
  return unwrap<Receipt[]>(await client.get(`/dist-orders/${distOrderId}/receipts`))
}

export async function createReceipt(distOrderId: number, data: Partial<Receipt>) {
  return unwrap<Receipt>(await client.post(`/dist-orders/${distOrderId}/receipts`, data))
}

export async function deleteReceipt(distOrderId: number, receiptId: number) {
  return unwrap(await client.delete(`/dist-orders/${distOrderId}/receipts/${receiptId}`))
}

export async function fetchAttachments(distOrderId: number) {
  return unwrap<Attachment[]>(await client.get(`/dist-orders/${distOrderId}/attachments`))
}

export async function createAttachment(distOrderId: number, data: {
  fileType: string
  fileName: string
  fileUrl: string
  paymentId?: number
  shipmentId?: number
  remark?: string
}) {
  return unwrap<Attachment>(await client.post(`/dist-orders/${distOrderId}/attachments`, data))
}

export async function deleteAttachment(distOrderId: number, attachmentId: number) {
  return unwrap(await client.delete(`/dist-orders/${distOrderId}/attachments/${attachmentId}`))
}

export async function uploadFile(file: File) {
  const form = new FormData()
  form.append('file', file)
  const res = await client.post('/upload', form, {
    headers: { 'Content-Type': 'multipart/form-data' },
  })
  return unwrap<{ url: string; fileName: string }>(res)
}
