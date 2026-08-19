import type { SelfOrderItem } from '../api/selfOrder'

export type SelfItemTreeRow = {
  key: string
  item: SelfOrderItem
  isSplitChild: boolean
  isSplitParent: boolean
}

function isSplitChildItem(it: SelfOrderItem) {
  return !!(it.splitKind || (it.parentSelfOrderItemId && it.parentSelfOrderItemId > 0))
}

/** 详情/物流：根行 + └ 拆分子行（对齐供应链中心） */
export function buildSelfItemTreeRows(items: SelfOrderItem[] | undefined): SelfItemTreeRow[] {
  if (!items?.length) return []
  const childrenByParent = new Map<number, SelfOrderItem[]>()
  const fullChildren: SelfOrderItem[] = []
  const roots: SelfOrderItem[] = []
  for (const it of items) {
    if (it.splitKind === 'full') {
      fullChildren.push(it)
      continue
    }
    if (it.splitKind === 'partial' && it.parentSelfOrderItemId) {
      const list = childrenByParent.get(it.parentSelfOrderItemId) || []
      list.push(it)
      childrenByParent.set(it.parentSelfOrderItemId, list)
      continue
    }
    roots.push(it)
  }
  const out: SelfItemTreeRow[] = []
  for (const root of roots) {
    const kids = childrenByParent.get(root.id || 0) || []
    out.push({
      key: `root-${root.id}`,
      item: root,
      isSplitChild: false,
      isSplitParent: kids.length > 0,
    })
    for (const ch of kids) {
      out.push({
        key: `child-${ch.id}`,
        item: ch,
        isSplitChild: true,
        isSplitParent: false,
      })
    }
  }
  // 整单拆分：无父行，直接以拆分子行展示（与供应链一致）
  for (const ch of fullChildren) {
    out.push({
      key: `full-${ch.id}`,
      item: ch,
      isSplitChild: true,
      isSplitParent: false,
    })
  }
  return out
}

export function selfItemTreeTitle(node: SelfItemTreeRow): string {
  const it = node.item
  if (node.isSplitChild) {
    return (it.skuSpecs || it.productName || it.skuCode || '规格').trim() || '规格'
  }
  return (it.skuSpecs || it.productName || it.skuCode || '商品').trim() || '商品'
}

export function isSelfItemShippable(it: SelfOrderItem, all: SelfOrderItem[]): boolean {
  if (isSplitChildItem(it)) return true
  const hasFull = all.some((x) => x.splitKind === 'full')
  if (hasFull) return false
  return !all.some((x) => x.splitKind === 'partial' && x.parentSelfOrderItemId === it.id)
}
