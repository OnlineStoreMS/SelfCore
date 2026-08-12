import type { Router } from 'vue-router'

/** 自营订单列表进入意图（不写进 URL） */
export type SelfOrderListIntent = {
  status?: string
  statuses?: string[]
  /** 付款状态，如 unpaid,partial */
  payStatuses?: string[]
  /** 发货状态：wait_ship | partial_shipped | shipped */
  shipStatus?: string
  excludeStatuses?: string[]
  /** 工作台「今日」（按创建自营单时间 created_at） */
  today?: boolean
  /** 创建日范围 YYYY-MM-DD（映射到 createdAtStart/End） */
  orderedDateStart?: string
  orderedDateEnd?: string
}

const KEY = 'selfcore.selfOrderListIntent'

type IntentListener = () => void
let intentListener: IntentListener | null = null
let pendingIntent: SelfOrderListIntent | null = null

export function onSelfOrderListIntent(listener: IntentListener) {
  intentListener = listener
  return () => {
    if (intentListener === listener) intentListener = null
  }
}

export function setSelfOrderListIntent(intent: SelfOrderListIntent) {
  pendingIntent = intent
  try {
    sessionStorage.setItem(KEY, JSON.stringify(intent))
  } catch {
    // ignore
  }
}

export function takeSelfOrderListIntent(): SelfOrderListIntent | null {
  const mem = pendingIntent
  pendingIntent = null
  if (mem) {
    try {
      sessionStorage.removeItem(KEY)
    } catch {
      // ignore
    }
    return mem
  }
  try {
    const raw = sessionStorage.getItem(KEY)
    if (!raw) return null
    sessionStorage.removeItem(KEY)
    return JSON.parse(raw) as SelfOrderListIntent
  } catch {
    return null
  }
}

export function goSelfOrders(router: Router, intent: SelfOrderListIntent = {}) {
  setSelfOrderListIntent(intent)
  const samePath = router.currentRoute.value.path === '/self-orders'
  void router.push('/self-orders').then(() => {
    if (samePath) intentListener?.()
  })
}
