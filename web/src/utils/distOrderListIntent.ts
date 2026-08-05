import type { Router } from 'vue-router'

/** 分销订单列表进入意图（不写进 URL，避免地址栏堆参数） */
export type DistOrderListIntent = {
  fulfillmentType?: string
  status?: string
  /** 多状态，如部分发货+已发货 */
  statuses?: string[]
  /** 收款状态，如 unpaid,partial */
  payStatuses?: string[]
  /** 排除状态，如 draft,cancelled */
  excludeStatuses?: string[]
  refSoId?: number
  /** 工作台「今日」：按业务日 COALESCE(ordered_at, created_at) 筛今天 */
  today?: boolean
  /** 分销业务日范围 YYYY-MM-DD（优先于 today） */
  orderedDateStart?: string
  orderedDateEnd?: string
}

const KEY = 'selfcore.distOrderListIntent'

type IntentListener = () => void
let intentListener: IntentListener | null = null

/** 列表页注册：同路径再次带意图进入时回调 */
export function onDistOrderListIntent(listener: IntentListener) {
  intentListener = listener
  return () => {
    if (intentListener === listener) intentListener = null
  }
}

export function setDistOrderListIntent(intent: DistOrderListIntent) {
  try {
    sessionStorage.setItem(KEY, JSON.stringify(intent))
  } catch {
    // ignore
  }
}

/** 读取并清除意图 */
export function takeDistOrderListIntent(): DistOrderListIntent | null {
  try {
    const raw = sessionStorage.getItem(KEY)
    if (!raw) return null
    sessionStorage.removeItem(KEY)
    return JSON.parse(raw) as DistOrderListIntent
  } catch {
    return null
  }
}

export function goDistOrders(
  router: Router,
  intent: DistOrderListIntent = {},
) {
  const ft = intent.fulfillmentType
  setDistOrderListIntent(intent)
  let path = '/dist-orders'
  if (ft === 'dropship') path = '/dist-orders/dropship'
  else if (ft === 'wholesale') path = '/dist-orders/wholesale'

  const samePath = router.currentRoute.value.path === path
  void router.push(path).then(() => {
    // 同路径 push 不会触发路由 watch，需显式通知列表重新应用意图
    if (samePath) intentListener?.()
  })
}
