import client, { unwrap } from './client'
import type { DistOrderListItem } from './distOrder'

export interface DashboardWorkbench {
  selfOrderPO: number
  selfUnpaidPO: number
  selfDraftPO: number
  selfWaitShipPO: number
  distOrderPO: number
  dropshipPO: number
  wholesalePO: number
  draftPO: number
  unpaidPO: number
  distWaitShipPO: number
  orderedPO?: number
  confirmedPO: number
  inTransitPO: number
  partialReceivedPO: number
  activeOffers: number
  todayDropshipSaleAmount: number
  todayDropshipWholesaleAmount: number
  todayDropshipProfit: number
  todayDistSaleAmount: number
  todaySelfSaleAmount: number
  weekSelfSaleAmount: number
  monthSelfSaleAmount: number
  monthDistSaleAmount: number
}

export interface DashboardDistributorStats {
  total: number
  active: number
  offerCount: number
  orderedThisMonth: number
}

export interface DashboardPOStats {
  total: number
  draft: number
  inProgress: number
  completed: number
  cancelled: number
  todayCount: number
  weekCount: number
  monthCount: number
}

export interface DashboardCostStats {
  todayAmount: number
  weekAmount: number
  monthAmount: number
  unpaidAmount: number
  yearAmount: number
}

export interface DashboardDistributorRank {
  distributorId: number
  distributorName: string
  orderCount: number
  totalAmount: number
}

export interface DashboardStatusCount {
  status: string
  count: number
}

export interface DashboardStats {
  workbench: DashboardWorkbench
  distributor: DashboardDistributorStats
  purchaseOrder: DashboardPOStats
  cost: DashboardCostStats
  topDistributors: DashboardDistributorRank[]
  recentOrders: DistOrderListItem[]
  statusBreakdown: DashboardStatusCount[]
}

export async function fetchDashboardStats() {
  return unwrap<DashboardStats>(await client.get('/dashboard/stats'))
}

export interface DashboardTrendPoint {
  date: string
  selfOrderCount: number
  selfSaleAmount: number
  orderCount: number
  saleAmount: number
  wholesaleAmount: number
  profit: number
}

export interface DashboardTrend {
  startDate: string
  endDate: string
  orderCount: number
  saleAmount: number
  wholesaleAmount: number
  profit: number
  points: DashboardTrendPoint[]
}

export async function fetchDashboardTrend(params: { startDate?: string; endDate?: string } = {}) {
  return unwrap<DashboardTrend>(await client.get('/dashboard/trend', { params }))
}
