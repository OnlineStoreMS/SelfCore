import client, { unwrap, type PageData } from './client'

export interface ProductSkuSearchItem {
  productId: number
  productName: string
  materialCode?: string
  productSn?: string
  productPic?: string
  brandName?: string
  categoryName?: string
  skuId: number
  skuCode: string
  specs: Record<string, string>
  specLabel: string
  price: number
  stock: number
  pic?: string
}

export interface ProductBrief {
  id: number
  name: string
  materialCode?: string
  productSn?: string
  pic?: string
  brandName?: string
  categoryName?: string
  price?: number
  stock?: number
  skuCount?: number
}

export interface ProductSkuItem {
  id: number
  skuCode: string
  specs: Record<string, string>
  price: number
  stock: number
  pic?: string
}

export interface ProductSkusPayload {
  id: number
  name: string
  materialCode?: string
  pic?: string
  skuCount: number
  skus: ProductSkuItem[]
}

export function formatSkuOptionLabel(item: ProductSkuSearchItem): string {
  const code = item.skuCode?.trim() || '未编码'
  const spec = item.specLabel?.trim() || '-'
  const name = item.productName?.trim() || ''
  const parts: string[] = [code, spec]
  if (name) parts.push(name)
  return parts.join(' · ')
}

/** 列表/下拉展示用的商家编码（不暴露内部 id） */
export function skuCodeLabel(item?: Pick<ProductSkuSearchItem, 'skuCode'> | null, fallback = '—'): string {
  const code = item?.skuCode?.trim()
  return code || fallback
}

export function formatSkuSpecLabel(specs?: Record<string, string>): string {
  if (!specs) return ''
  return Object.entries(specs)
    .filter(([, v]) => String(v || '').trim())
    .map(([k, v]) => `${k}: ${v}`)
    .join(' / ')
}

export function skuDisplayPic(item?: Pick<ProductSkuSearchItem, 'pic' | 'productPic'> | null): string {
  return item?.pic || item?.productPic || ''
}

const skuResolveCache = new Map<number, ProductSkuSearchItem>()

export async function searchProductSkus(params: {
  keyword: string
  page?: number
  pageSize?: number
}) {
  const res = await client.get('/product-skus/search', { params })
  const page = unwrap<PageData<ProductSkuSearchItem>>(res)
  for (const item of page.list) {
    skuResolveCache.set(item.skuId, item)
  }
  return page
}

export async function searchProducts(params?: {
  keyword?: string
  page?: number
  pageSize?: number
}) {
  const res = await client.get('/products/search', { params })
  return unwrap<PageData<ProductBrief>>(res)
}

export async function fetchProductSkus(productId: number) {
  return unwrap<ProductSkusPayload>(await client.get(`/products/${productId}/skus`))
}

/** 批量解析 SKU 详情（图片、规格等），带内存缓存 */
export async function resolveProductSkus(ids: number[]): Promise<Map<number, ProductSkuSearchItem>> {
  const unique = [...new Set(ids.filter((id) => id > 0))]
  const missing = unique.filter((id) => !skuResolveCache.has(id))
  await Promise.all(
    missing.map(async (id) => {
      try {
        const data = await searchProductSkus({ keyword: String(id), page: 1, pageSize: 10 })
        const hit = data.list.find((item) => item.skuId === id)
        if (hit) {
          skuResolveCache.set(id, hit)
        }
      } catch {
        // ignore single resolve failure
      }
    }),
  )
  const result = new Map<number, ProductSkuSearchItem>()
  for (const id of unique) {
    const hit = skuResolveCache.get(id)
    if (hit) result.set(id, hit)
  }
  return result
}

export function cacheSkuSearchItem(item: ProductSkuSearchItem) {
  skuResolveCache.set(item.skuId, item)
}
