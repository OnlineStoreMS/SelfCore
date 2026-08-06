import client, { unwrap } from './client'

export interface SelfShipmentItem {
  id?: number
  selfOrderItemId: number
  qty: number
}

export interface SelfShipment {
  id: number
  selfOrderId: number
  shipmentNo: string
  status: string
  carrierCode?: string
  carrierName?: string
  trackingNo?: string
  shippedAt?: string
  expectedArrivalDate?: string
  deliveredAt?: string
  callbackOk: boolean
  stockDeducted: boolean
  receiverName?: string
  receiverPhone?: string
  receiverAddress?: string
  remark?: string
  items: SelfShipmentItem[]
  createdAt: string
}

export interface SelfAttachment {
  id: number
  selfOrderId: number
  shipmentId?: number
  fileType: string
  fileName: string
  fileUrl: string
  uploadedBy?: number
  remark?: string
  createdAt: string
}

export const SELF_ATTACHMENT_TYPE_MAP: Record<string, string> = {
  shipment_photo: '物流发货照片',
  contract: '合同',
  other: '其他',
}

export async function fetchSelfShipments(selfOrderId: number) {
  return unwrap<SelfShipment[]>(await client.get(`/self-orders/${selfOrderId}/shipments`))
}

export async function createSelfShipment(
  selfOrderId: number,
  data: Partial<SelfShipment> & { items?: SelfShipmentItem[]; callback?: boolean },
) {
  return unwrap<SelfShipment>(await client.post(`/self-orders/${selfOrderId}/shipments`, data))
}

export async function syncSelfShipmentsFromOrders(selfOrderId: number, refSoId?: number) {
  return unwrap<{ created: number; updated: number; skipped: number; errors?: string[] }>(
    await client.post(`/self-orders/${selfOrderId}/shipments/sync-from-orders`, refSoId ? { refSoId } : {}),
  )
}

export async function updateSelfShipmentStatus(selfOrderId: number, shipmentId: number, status: string) {
  return unwrap<SelfShipment>(
    await client.patch(`/self-orders/${selfOrderId}/shipments/${shipmentId}/status`, { status }),
  )
}

export async function deleteSelfShipment(selfOrderId: number, shipmentId: number) {
  return unwrap(await client.delete(`/self-orders/${selfOrderId}/shipments/${shipmentId}`))
}

export async function fetchSelfAttachments(selfOrderId: number) {
  return unwrap<SelfAttachment[]>(await client.get(`/self-orders/${selfOrderId}/attachments`))
}

export async function createSelfAttachment(selfOrderId: number, data: {
  fileType: string
  fileName: string
  fileUrl: string
  shipmentId?: number
  remark?: string
}) {
  return unwrap<SelfAttachment>(await client.post(`/self-orders/${selfOrderId}/attachments`, data))
}

export async function deleteSelfAttachment(selfOrderId: number, attachmentId: number) {
  return unwrap(await client.delete(`/self-orders/${selfOrderId}/attachments/${attachmentId}`))
}

export { uploadFile } from './tracking'
