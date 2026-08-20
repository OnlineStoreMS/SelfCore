/** 自营订单列表筛选记忆（进详情再返回时恢复；工作台意图 / 手动重置会覆盖） */

export type SelfOrderListFilterSnapshot = {
  status: string
  statusesFilter: string
  payStatusesFilter: string
  excludeStatusesFilter: string
  shipStatus: string
  payStatus: string
  keyword: string
  createdRange: [string, string] | null
  orderedRange: [string, string] | null
  page: number
  pageSize: number
}

const KEY = 'selfcore.selfOrderListFilters'

export function saveSelfOrderListFilters(snap: SelfOrderListFilterSnapshot) {
  try {
    sessionStorage.setItem(KEY, JSON.stringify(snap))
  } catch {
    // ignore
  }
}

export function loadSelfOrderListFilters(): SelfOrderListFilterSnapshot | null {
  try {
    const raw = sessionStorage.getItem(KEY)
    if (!raw) return null
    const obj = JSON.parse(raw) as SelfOrderListFilterSnapshot
    if (!obj || typeof obj !== 'object') return null
    return obj
  } catch {
    return null
  }
}

export function clearSelfOrderListFilters() {
  try {
    sessionStorage.removeItem(KEY)
  } catch {
    // ignore
  }
}
