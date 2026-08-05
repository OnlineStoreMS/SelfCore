import client, { unwrap, type PageData } from './client'

export interface DistributorCategory {
  id: number
  name: string
  parentId?: number
  sort?: number
  status: number
  remark?: string
}

export interface Distributor {
  id: number
  categoryId?: number
  categoryName?: string
  code: string
  name: string
  shortName?: string
  status: number
  buyerName?: string
  cutOffTime?: string
  arrivalDays?: number
  paymentDays?: number
  settlementCycle?: string
  settlementCustomDays?: number
  settlementMergeTime?: string
  autoCreateDropshipPO?: boolean
  /** 同步批发价来源：fen_fa_remark | alloc_remark | seller_remark | printer_remark */
  syncPurchasePriceFrom?: string
  contactName?: string
  address?: string
  officePhone?: string
  mobile?: string
  phone?: string
  wangwangId?: string
  qq?: string
  email?: string
  website?: string
  remark?: string
  defaultPaymentTerms?: string
  bankName?: string
  bankAccount?: string
  accountName?: string
  createdAt?: string
}

export const SETTLEMENT_CYCLE_MAP: Record<string, string> = {
  '': '不启用',
  day: '按天',
  week: '按周',
  month: '按月',
  custom: '自定义天数',
}

export interface DistributorAddress {
  id: number
  distributorId: number
  addressType?: string
  label: string
  contactName?: string
  phone?: string
  province?: string
  city?: string
  district?: string
  address?: string
  isDefault: boolean
  status: number
}

export interface DistributorPaymentAccount {
  id: number
  distributorId: number
  label: string
  accountType: string
  bankName?: string
  bankAccount?: string
  accountName?: string
  isDefault: boolean
  status: number
  remark?: string
}

export interface DistributorPaymentQR {
  id: number
  distributorId: number
  label: string
  payType: string
  imageUrl: string
  accountName?: string
  isDefault: boolean
  status: number
  remark?: string
}

export const ACCOUNT_TYPE_MAP: Record<string, string> = {
  bank: '银行账户',
  alipay: '支付宝',
  wechat: '微信',
  other: '其他',
}

export const PAY_TYPE_MAP: Record<string, string> = {
  wechat: '微信',
  alipay: '支付宝',
  other: '其他',
}

export interface SkuPrice {
  id: number
  skuId: number
  distributorId: number
  distributorName?: string
  distributorCode?: string
  distributorSkuCode?: string
  wholesalePrice: number
  currency: string
  minOrderQty: number
  leadTimeDays: number
  shipFromAddressId?: number
  shipFromLabel?: string
  shipFromCity?: string
  supportsDropship: boolean
  supportsWholesale: boolean
  isPrimary: boolean
  priority: number
  status: number
  remark?: string
}

export async function fetchDistributorCategories() {
  return unwrap<DistributorCategory[]>(await client.get('/distributor-categories'))
}

export async function createDistributorCategory(data: Partial<DistributorCategory>) {
  return unwrap<DistributorCategory>(await client.post('/distributor-categories', data))
}

export async function updateDistributorCategory(id: number, data: Partial<DistributorCategory>) {
  return unwrap<DistributorCategory>(await client.put(`/distributor-categories/${id}`, data))
}

export async function deleteDistributorCategory(id: number) {
  return unwrap(await client.delete(`/distributor-categories/${id}`))
}

export async function fetchDistributors(params?: {
  keyword?: string
  categoryId?: number
  page?: number
  pageSize?: number
}) {
  const { keyword, categoryId, page = 1, pageSize = 20 } = params ?? {}
  const res = await client.get('/distributors', {
    params: { keyword, categoryId: categoryId || undefined, page, pageSize },
  })
  return unwrap<PageData<Distributor>>(res)
}

export async function fetchDistributor(id: number) {
  return unwrap<Distributor>(await client.get(`/distributors/${id}`))
}

export async function createDistributor(data: Partial<Distributor>) {
  return unwrap<Distributor>(await client.post('/distributors', data))
}

export async function updateDistributor(id: number, data: Partial<Distributor>) {
  return unwrap<Distributor>(await client.put(`/distributors/${id}`, data))
}

export async function deleteDistributor(id: number) {
  return unwrap(await client.delete(`/distributors/${id}`))
}

export async function fetchDistributorAddresses(distributorId: number, addressType?: string) {
  return unwrap<DistributorAddress[]>(
    await client.get(`/distributors/${distributorId}/addresses`, {
      params: { type: addressType || undefined },
    }),
  )
}

export async function createDistributorAddress(distributorId: number, data: Partial<DistributorAddress>) {
  return unwrap<DistributorAddress>(await client.post(`/distributors/${distributorId}/addresses`, data))
}

export async function updateDistributorAddress(distributorId: number, addressId: number, data: Partial<DistributorAddress>) {
  return unwrap<DistributorAddress>(await client.put(`/distributors/${distributorId}/addresses/${addressId}`, data))
}

export async function deleteDistributorAddress(distributorId: number, addressId: number) {
  return unwrap(await client.delete(`/distributors/${distributorId}/addresses/${addressId}`))
}

export async function fetchDistributorPaymentAccounts(distributorId: number) {
  return unwrap<DistributorPaymentAccount[]>(await client.get(`/distributors/${distributorId}/payment-accounts`))
}

export async function createDistributorPaymentAccount(distributorId: number, data: Partial<DistributorPaymentAccount>) {
  return unwrap<DistributorPaymentAccount>(await client.post(`/distributors/${distributorId}/payment-accounts`, data))
}

export async function updateDistributorPaymentAccount(distributorId: number, accountId: number, data: Partial<DistributorPaymentAccount>) {
  return unwrap<DistributorPaymentAccount>(await client.put(`/distributors/${distributorId}/payment-accounts/${accountId}`, data))
}

export async function deleteDistributorPaymentAccount(distributorId: number, accountId: number) {
  return unwrap(await client.delete(`/distributors/${distributorId}/payment-accounts/${accountId}`))
}

export async function fetchDistributorPaymentQRs(distributorId: number) {
  return unwrap<DistributorPaymentQR[]>(await client.get(`/distributors/${distributorId}/payment-qrs`))
}

export async function createDistributorPaymentQR(distributorId: number, data: Partial<DistributorPaymentQR>) {
  return unwrap<DistributorPaymentQR>(await client.post(`/distributors/${distributorId}/payment-qrs`, data))
}

export async function updateDistributorPaymentQR(distributorId: number, qrId: number, data: Partial<DistributorPaymentQR>) {
  return unwrap<DistributorPaymentQR>(await client.put(`/distributors/${distributorId}/payment-qrs/${qrId}`, data))
}

export async function deleteDistributorPaymentQR(distributorId: number, qrId: number) {
  return unwrap(await client.delete(`/distributors/${distributorId}/payment-qrs/${qrId}`))
}

export async function fetchSkuPrices(params: { skuId?: number; distributorId?: number; page?: number; pageSize?: number }) {
  const res = await client.get('/sku-prices', { params })
  return unwrap<PageData<SkuPrice>>(res)
}

export async function createSkuPrice(data: Partial<SkuPrice>) {
  return unwrap<SkuPrice>(await client.post('/sku-prices', data))
}

export async function updateSkuPrice(id: number, data: Partial<SkuPrice>) {
  return unwrap<SkuPrice>(await client.put(`/sku-prices/${id}`, data))
}

export async function deleteSkuPrice(id: number) {
  return unwrap(await client.delete(`/sku-prices/${id}`))
}

/** Display mobile with fallback to legacy phone field */
export function distributorMobile(row: Distributor): string {
  return row.mobile || row.phone || ''
}
