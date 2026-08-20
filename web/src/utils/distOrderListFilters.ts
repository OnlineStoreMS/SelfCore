/** 分销订单列表筛选记忆（按履约类型分 Tab；进详情再返回时恢复） */

export type DistOrderListFilterSnapshot = {
  status: string
  statusesFilter: string
  payStatusFilter: string
  excludeStatusesFilter: string
  distributorId?: number
  keyword: string
  refSoId?: number
  createdRange: [string, string] | null
  orderedRange: [string, string] | null
  page: number
  pageSize: number
  sortBy: string
  sortOrder: 'asc' | 'desc'
}

function storageKey(fulfillmentType: string) {
  const ft = fulfillmentType || 'all'
  return `selfcore.distOrderListFilters.${ft}`
}

export function saveDistOrderListFilters(fulfillmentType: string, snap: DistOrderListFilterSnapshot) {
  try {
    sessionStorage.setItem(storageKey(fulfillmentType), JSON.stringify(snap))
  } catch {
    // ignore
  }
}

export function loadDistOrderListFilters(fulfillmentType: string): DistOrderListFilterSnapshot | null {
  try {
    const raw = sessionStorage.getItem(storageKey(fulfillmentType))
    if (!raw) return null
    const obj = JSON.parse(raw) as DistOrderListFilterSnapshot
    if (!obj || typeof obj !== 'object') return null
    return obj
  } catch {
    return null
  }
}

export function clearDistOrderListFilters(fulfillmentType: string) {
  try {
    sessionStorage.removeItem(storageKey(fulfillmentType))
  } catch {
    // ignore
  }
}
