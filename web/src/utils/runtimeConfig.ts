declare global {
  interface Window {
    __RUNTIME_CONFIG__?: {
      portalUrl?: string
      orderCoreUrl?: string
    }
  }
}

function trimUrl(v?: string | null): string {
  return (v || '').trim().replace(/\/$/, '')
}

function isLocalHost(hostname: string): boolean {
  return hostname === 'localhost' || hostname === '127.0.0.1' || hostname === '::1'
}

/** 与当前访问主机同机的 UserCore 应用中心 */
function portalFromLocation(): string {
  if (typeof window === 'undefined' || !window.location?.hostname) return ''
  const { protocol, hostname } = window.location
  if (!hostname || isLocalHost(hostname)) return ''
  return `${protocol}//${hostname}:5174`
}

function deriveHostBase(port: number, fallbackHost = 'localhost'): string {
  const portal = getPortalUrl()
  try {
    const u = new URL(portal)
    return `${u.protocol}//${u.hostname}:${port}`
  } catch {
    return `http://${fallbackHost}:${port}`
  }
}

/**
 * 门户地址优先级：
 * 1) 本机访问 → 固定 http://localhost:5174
 * 2) runtime-config.js（部署注入）
 * 3) 当前访问主机推导
 * 4) 构建期 VITE_PORTAL_URL（仅非 localhost）
 * 5) http://localhost:5174
 */
export function getPortalUrl(): string {
  const host = typeof window !== 'undefined' ? window.location.hostname : ''
  if (isLocalHost(host)) {
    return 'http://localhost:5174'
  }

  const fromRuntime = trimUrl(window.__RUNTIME_CONFIG__?.portalUrl)
  if (fromRuntime) return fromRuntime

  const fromHost = portalFromLocation()
  if (fromHost) return fromHost

  const fromEnv = trimUrl(import.meta.env.VITE_PORTAL_URL)
  if (fromEnv && !/^https?:\/\/(localhost|127\.0\.0\.1)(:|\/|$)/i.test(fromEnv)) {
    return fromEnv
  }

  return 'http://localhost:5174'
}

/** 订单中心 Web 根地址 */
export function getOrderCoreUrl(): string {
  return (
    trimUrl(window.__RUNTIME_CONFIG__?.orderCoreUrl) ||
    trimUrl(import.meta.env.VITE_ORDERCORE_URL) ||
    deriveHostBase(5182)
  )
}

/** 跳转订单中心订单详情 */
export function orderCoreOrderUrl(orderId: number | string): string {
  const base = getOrderCoreUrl().replace(/\/$/, '')
  return `${base}/orders/${orderId}`
}
