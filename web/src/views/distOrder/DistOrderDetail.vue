<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { ArrowLeft, Edit } from '@element-plus/icons-vue'
import {
  fetchDistOrder,
  submitDistOrder,
  markDistOrderPaid,
  completeDistOrder,
  cancelDistOrder,
  deleteDistOrder,
  updateDistOrderItemPrices,
  detachSalesOrder,
  DIST_STATUS_MAP,
  PAY_STATUS_MAP,
  type DistOrder,
} from '../../api/distOrder'
import { resolveProductSkus, type ProductSkuSearchItem } from '../../api/productSku'
import { fetchShipments, type Shipment } from '../../api/tracking'
import { decryptOrders, fetchOrder, type OrderBrief } from '../../api/order'
import { copyToClipboard } from '../../utils/clipboard'
import {
  buildMultiOrderCopyText,
  canDecryptOrder,
  isMaskedReceiver,
} from '../../utils/orderCopy'
import PoShipmentTab from './PoShipmentTab.vue'
import ReceiptTab from './ReceiptTab.vue'
import PoAttachmentTab from './PoAttachmentTab.vue'
import { orderCoreOrderUrl } from '../../utils/runtimeConfig'

const route = useRoute()
const router = useRouter()
const poId = computed(() => Number(route.params.id))
const activeTab = ref('overview')

const loading = ref(false)
const acting = ref(false)
const savingPrice = ref(false)
const decrypting = ref(false)
const copying = ref(false)
const po = ref<DistOrder | null>(null)
const skuMap = ref<Map<number, ProductSkuSearchItem>>(new Map())
const shipments = ref<Shipment[]>([])

/** 草稿或已确认且未收款时可改分销订单价（代发单常自动提交后补价） */
const canEditUnitPrice = computed(() => {
  if (!po.value) return false
  if (po.value.payStatus === 'paid' || po.value.payStatus === 'partial') return false
  return po.value.status === 'draft' || po.value.status === 'confirmed'
})

async function onUnitPriceChange(row: { id?: number; unitPrice: number; cancelled?: boolean }) {
  if (!po.value || !row.id || row.cancelled || !canEditUnitPrice.value) return
  const price = Number(row.unitPrice)
  if (Number.isNaN(price) || price < 0) {
    ElMessage.warning('分销订单价不能为负数')
    await loadData()
    return
  }
  savingPrice.value = true
  try {
    po.value = await updateDistOrderItemPrices(poId.value, [{ itemId: row.id, unitPrice: price }])
  } catch (e) {
    ElMessage.error((e as Error).message || '保存单价失败')
    await loadData()
  } finally {
    savingPrice.value = false
  }
}

function openOrderCore(soId?: number) {
  const id = soId && soId > 0 ? soId : po.value?.refSoId
  if (!id) return
  window.open(orderCoreOrderUrl(id), '_blank', 'noopener,noreferrer')
}

function splitRefOrders(trace?: string) {
  if (!trace?.trim()) return [] as string[]
  return trace
    .split(/[,，;\s]+/)
    .map((s) => s.trim())
    .filter(Boolean)
}

/** 关联订单：优先从明细汇总（含已撤回），便于整单撤销后仍能看到历史关联 */
const refOrders = computed(() => {
  const items = po.value?.items || []
  const map = new Map<string, { cancelled: boolean; soId: number }>()
  for (const it of items) {
    const no = (it.refOrderNo || '').trim()
    if (!no) continue
    const prev = map.get(no)
    if (!prev) {
      map.set(no, { cancelled: !!it.cancelled, soId: Number(it.refSoId || 0) })
    } else {
      // 任一明细未撤回，则该销售单视为仍关联
      prev.cancelled = prev.cancelled && !!it.cancelled
      if (!prev.soId && it.refSoId) prev.soId = Number(it.refSoId)
    }
  }
  if (map.size > 0) {
    return [...map.entries()].map(([no, v]) => ({ no, cancelled: v.cancelled, soId: v.soId }))
  }
  const allCancelled = po.value?.status === 'cancelled'
  return splitRefOrders(po.value?.refTraceId).map((no) => ({
    no,
    cancelled: allCancelled,
    soId: 0,
  }))
})

const activeRefOrders = computed(() => refOrders.value.filter((r) => !r.cancelled))
const allRefWithdrawn = computed(
  () => refOrders.value.length > 0 && activeRefOrders.value.length === 0,
)

const canDetachSales = computed(
  () => po.value?.fulfillmentType === 'dropship' && po.value.status !== 'cancelled',
)

async function handleDetachSales(r: { no: string; soId: number }) {
  if (!po.value || !canDetachSales.value || !r.no) return
  const paidLike =
    po.value.payStatus === 'paid' ||
    po.value.payStatus === 'partial' ||
    ['paid', 'partial_shipped', 'shipped', 'partial_received', 'completed'].includes(po.value.status)
  const tip = paidLike
    ? `确定从本代发单解绑销售单 ${r.no}？\n单据已收款/履约，仅划线解绑并回写订单中心为待分配，不冲销收款记录。`
    : `确定从本代发单解绑销售单 ${r.no}？将同步清空订单中心分配。`
  try {
    await ElMessageBox.confirm(tip, '解绑销售单', { type: 'warning' })
  } catch {
    return
  }
  acting.value = true
  try {
    const res = await detachSalesOrder({
      distNo: po.value.distNo,
      orderNo: r.no,
      soId: r.soId || undefined,
      reason: '供应链手工解绑',
    })
    po.value = res.distOrder
    if (po.value?.items) {
      skuMap.value = await resolveProductSkus(po.value.items.map((it) => it.skuId))
    }
    if (res.unlinkWarning) {
      ElMessage.warning(`已解绑代发明细，但回写订单中心失败：${res.unlinkWarning}`)
    } else {
      ElMessage.success('已解绑，订单中心已恢复待分配')
    }
  } catch (e) {
    ElMessage.error((e as Error).message || '解绑失败')
  } finally {
    acting.value = false
  }
}

const trackable = computed(() => po.value && po.value.status !== 'draft' && po.value.status !== 'cancelled')

const logisticsByItem = computed(() => {
  const map = new Map<number, string[]>()
  for (const sh of shipments.value) {
    const tracking = [sh.carrierName, sh.trackingNo].filter(Boolean).join(' ')
    if (!tracking) continue
    for (const it of sh.items || []) {
      const arr = map.get(it.distOrderItemId) || []
      if (!arr.includes(tracking)) arr.push(tracking)
      map.set(it.distOrderItemId, arr)
    }
  }
  return map
})

async function loadData() {
  loading.value = true
  try {
    po.value = await fetchDistOrder(poId.value)
    skuMap.value = await resolveProductSkus((po.value.items || []).map((it) => it.skuId))
    if (po.value.status !== 'draft' && po.value.status !== 'cancelled') {
      shipments.value = await fetchShipments(poId.value)
    } else {
      shipments.value = []
    }
  } catch (e) {
    ElMessage.error((e as Error).message || '加载失败')
  } finally {
    loading.value = false
  }
}

onMounted(loadData)

function statusLabel(s: string) {
  return DIST_STATUS_MAP[s]?.label || s
}

function statusType(s: string) {
  return DIST_STATUS_MAP[s]?.type || 'info'
}

function lineSkuCode(row: { skuId: number; skuCode?: string }) {
  const code = row.skuCode?.trim()
  if (code) return code
  return skuMap.value.get(row.skuId)?.skuCode?.trim() || '—'
}

function itemRowClass({ row }: { row: { cancelled?: boolean } }) {
  return row.cancelled ? 'po-item-cancelled' : ''
}

async function doAction(label: string, fn: () => Promise<unknown>) {
  try {
    await ElMessageBox.confirm(`确定${label}？`, '确认')
  } catch {
    return
  }
  acting.value = true
  try {
    po.value = (await fn()) as DistOrder
    if (po.value?.items) {
      skuMap.value = await resolveProductSkus(po.value.items.map((it) => it.skuId))
    }
    if (po.value && po.value.status !== 'draft' && po.value.status !== 'cancelled') {
      shipments.value = await fetchShipments(poId.value)
    } else {
      shipments.value = []
    }
    ElMessage.success('操作成功')
  } catch (e) {
    ElMessage.error((e as Error).message || '操作失败')
  } finally {
    acting.value = false
  }
}

async function handleDelete() {
  await doAction('删除此分销订单（物流/收款等一并删除）', async () => {
    await deleteDistOrder(poId.value)
    router.push('/dist-orders')
    return null
  })
}

function collectRefSoIds(order: DistOrder): number[] {
  const ids = new Set<number>()
  if (order.refSoId && order.refSoId > 0) ids.add(order.refSoId)
  for (const it of order.items || []) {
    if (it.refSoId && it.refSoId > 0) ids.add(it.refSoId)
  }
  return [...ids]
}

async function loadLinkedOrders(): Promise<OrderBrief[]> {
  if (!po.value) throw new Error('分销订单未加载')
  const ids = collectRefSoIds(po.value)
  if (!ids.length) throw new Error('该代发单未关联销售订单')
  const orders: OrderBrief[] = []
  for (const id of ids) {
    orders.push(await fetchOrder(id))
  }
  return orders
}

async function handleDecrypt() {
  if (po.value?.fulfillmentType !== 'dropship') return
  decrypting.value = true
  try {
    const orders = await loadLinkedOrders()
    const ecommerce = orders.filter((o) => canDecryptOrder(o))
    if (!ecommerce.length) {
      ElMessage.warning('关联销售单中无可解密的电商订单')
      return
    }
    const data = await decryptOrders(ecommerce.map((o) => o.id))
    ElMessage.success(
      data.success > 1 ? `已依次解密 ${data.success} 笔电商订单` : '解密成功',
    )
  } catch (e) {
    ElMessage.error((e as Error).message || '解密失败')
  } finally {
    decrypting.value = false
  }
}

async function handleCopy() {
  if (po.value?.fulfillmentType !== 'dropship') return
  copying.value = true
  try {
    let orders = await loadLinkedOrders()
    const needDecrypt = orders.filter((o) => canDecryptOrder(o) && isMaskedReceiver(o))
    if (needDecrypt.length) {
      const data = await decryptOrders(needDecrypt.map((o) => o.id))
      const byId = new Map((data.items || []).map((o) => [o.id, o]))
      orders = orders.map((o) => byId.get(o.id) || o)
    }
    const text = buildMultiOrderCopyText(orders)
    if (!text.trim()) {
      ElMessage.warning('暂无收件信息可复制')
      return
    }
    const ok = await copyToClipboard(text)
    if (ok) {
      ElMessage.success(orders.length > 1 ? `已复制 ${orders.length} 笔（已标序号）` : '已复制')
    } else {
      ElMessage.error('复制失败')
    }
  } catch (e) {
    ElMessage.error((e as Error).message || '复制失败')
  } finally {
    copying.value = false
  }
}
</script>

<template>
  <div v-loading="loading" class="po-detail">
    <div class="top-bar">
      <el-button :icon="ArrowLeft" text @click="router.push('/dist-orders')">返回列表</el-button>
      <div v-if="po" class="actions">
        <template v-if="po.fulfillmentType === 'dropship'">
          <el-button type="warning" plain :loading="decrypting" @click="handleDecrypt">解密</el-button>
          <el-button :loading="copying" @click="handleCopy">复制</el-button>
        </template>
        <el-button
          v-if="po.status === 'draft'"
          :icon="Edit"
          @click="router.push(`/dist-orders/${po.id}/edit`)"
        >
          编辑
        </el-button>
        <el-button
          v-if="po.status !== 'completed'"
          type="danger"
          plain
          :loading="acting"
          @click="handleDelete"
        >
          删除
        </el-button>
        <el-button v-if="po.status === 'draft'" type="primary" :loading="acting" @click="doAction('提交下单', () => submitDistOrder(poId))">
          提交下单
        </el-button>
        <el-button v-if="po.status === 'confirmed'" type="warning" :loading="acting" @click="doAction('标记已收款', () => markDistOrderPaid(poId))">
          快捷标记已收款
        </el-button>
        <el-button
          v-if="['paid', 'partial_shipped', 'shipped', 'partial_received'].includes(po.status)"
          type="success"
          :loading="acting"
          @click="doAction('完成采购', () => completeDistOrder(poId))"
        >
          完成
        </el-button>
        <el-button
          v-if="po.status === 'draft' || po.status === 'confirmed'"
          :loading="acting"
          @click="doAction('取消分销订单', () => cancelDistOrder(poId))"
        >
          取消
        </el-button>
      </div>
    </div>

    <el-card v-if="po">
      <template #header>
        <div class="header-row">
          <span>{{ po.distNo }}</span>
          <el-tag :type="statusType(po.status)">{{ statusLabel(po.status) }}</el-tag>
        </div>
      </template>

      <el-tabs v-model="activeTab">
        <el-tab-pane label="基本信息" name="overview">
          <el-descriptions :column="2" border>
            <el-descriptions-item label="分销商">{{ po.distributorName }}（{{ po.distributorCode }}）</el-descriptions-item>
            <el-descriptions-item label="收款状态">{{ PAY_STATUS_MAP[po.payStatus] || po.payStatus }}</el-descriptions-item>
            <el-descriptions-item label="订单类型">{{ po.fulfillmentType === 'dropship' ? '代发' : '批发' }}</el-descriptions-item>
            <el-descriptions-item label="关联订单" :span="2">
              <template v-if="refOrders.length">
                <div class="ref-orders">
                  <div
                    v-for="(r, idx) in refOrders"
                    :key="idx"
                    class="ref-order-line"
                    :class="{ 'line-cancelled': r.cancelled }"
                  >
                    <span
                      class="ref-order-no"
                      :class="{ linkable: !r.cancelled && (!!r.soId || !!po.refSoId) }"
                      @click="!r.cancelled && (r.soId || po.refSoId) ? openOrderCore(r.soId || po.refSoId) : undefined"
                    >
                      {{ r.no }}
                    </span>
                    <el-tag v-if="r.cancelled" type="info" size="small" class="cancel-tag">已撤回</el-tag>
                    <el-button
                      v-else-if="canDetachSales"
                      type="danger"
                      link
                      size="small"
                      :loading="acting"
                      class="detach-btn"
                      @click.stop="handleDetachSales(r)"
                    >
                      解绑
                    </el-button>
                  </div>
                </div>
                <div v-if="refOrders.length > 1" class="ref-count">
                  <template v-if="allRefWithdrawn">共 {{ refOrders.length }} 单（均已撤回）</template>
                  <template v-else-if="activeRefOrders.length < refOrders.length">
                    有效 {{ activeRefOrders.length }} / 共 {{ refOrders.length }} 单
                  </template>
                  <template v-else>共 {{ refOrders.length }} 单</template>
                </div>
              </template>
              <el-link v-else-if="po.refSoId" type="primary" @click="openOrderCore(po.refSoId)">
                #{{ po.refSoId }}
              </el-link>
              <span v-else>—</span>
            </el-descriptions-item>
            <el-descriptions-item label="订单总金额">
              ¥{{ Number(po.saleAmount || 0).toFixed(2) }} {{ po.currency }}
            </el-descriptions-item>
            <el-descriptions-item label="采购总额">¥{{ po.totalAmount.toFixed(2) }} {{ po.currency }}</el-descriptions-item>
            <el-descriptions-item label="预计到货">{{ po.expectedArrivalDate || '—' }}</el-descriptions-item>
            <el-descriptions-item label="采购员">{{ po.buyerName || '—' }}</el-descriptions-item>
            <el-descriptions-item label="下单时间">{{ po.orderedAt || '—' }}</el-descriptions-item>
            <el-descriptions-item label="完成时间">{{ po.completedAt || '—' }}</el-descriptions-item>
            <el-descriptions-item label="备注" :span="2">{{ po.remark || '—' }}</el-descriptions-item>
          </el-descriptions>

          <h4 class="section-title">采购明细</h4>
          <el-table :data="po.items" border stripe :row-class-name="itemRowClass">
            <el-table-column label="图片" width="72" align="center">
              <template #default="{ row }">
                <el-image
                  v-if="row.picUrl"
                  :src="row.picUrl"
                  :preview-src-list="[row.picUrl]"
                  fit="cover"
                  style="width: 40px; height: 40px; border-radius: 4px"
                  preview-teleported
                />
                <span v-else class="muted">—</span>
              </template>
            </el-table-column>
            <el-table-column label="销售单" width="140" show-overflow-tooltip>
              <template #default="{ row }">
                <span :class="{ 'line-cancelled': row.cancelled }">{{ row.refOrderNo || '—' }}</span>
                <el-tag v-if="row.cancelled" type="info" size="small" class="cancel-tag">已撤回</el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="productName" label="商品" min-width="180" show-overflow-tooltip>
              <template #default="{ row }">
                <span :class="{ 'line-cancelled': row.cancelled }">
                  {{ row.productName || skuMap.get(row.skuId)?.productName || '—' }}
                </span>
              </template>
            </el-table-column>
            <el-table-column label="规格" width="130" show-overflow-tooltip>
              <template #default="{ row }">
                <span :class="{ 'line-cancelled': row.cancelled }">
                  {{ row.skuSpecs || skuMap.get(row.skuId)?.specLabel || '—' }}
                </span>
              </template>
            </el-table-column>
            <el-table-column label="商家编码" width="140">
              <template #default="{ row }">
                <span :class="{ 'line-cancelled': row.cancelled }">{{ lineSkuCode(row) }}</span>
              </template>
            </el-table-column>
            <el-table-column prop="distributorSkuCode" label="对方货号" width="120">
              <template #default="{ row }">
                <span :class="{ 'line-cancelled': row.cancelled }">{{ row.distributorSkuCode || '—' }}</span>
              </template>
            </el-table-column>
            <el-table-column prop="qty" label="数量" width="80" align="center">
              <template #default="{ row }">
                <span :class="{ 'line-cancelled': row.cancelled }">{{ row.qty }}</span>
              </template>
            </el-table-column>
            <el-table-column label="实付金额" width="110" align="right">
              <template #default="{ row }">
                <span v-if="row.saleAmount > 0" :class="{ 'line-cancelled': row.cancelled }">¥{{ Number(row.saleAmount).toFixed(2) }}</span>
                <span v-else class="muted">—</span>
              </template>
            </el-table-column>
            <el-table-column label="分销订单价" width="120" align="right" class-name="unit-price-col">
              <template #default="{ row }">
                <el-input
                  v-if="canEditUnitPrice && !row.cancelled"
                  v-model="row.unitPrice"
                  size="small"
                  class="unit-price-input"
                  :disabled="savingPrice"
                  @change="onUnitPriceChange(row)"
                />
                <span v-else :class="{ 'line-cancelled': row.cancelled }">¥{{ Number(row.unitPrice || 0).toFixed(2) }}</span>
              </template>
            </el-table-column>
            <el-table-column label="采购小计" width="110" align="right">
              <template #default="{ row }">
                <span :class="{ 'line-cancelled': row.cancelled }">¥{{ row.lineAmount.toFixed(2) }}</span>
              </template>
            </el-table-column>
            <el-table-column label="物流" min-width="160" show-overflow-tooltip>
              <template #default="{ row }">
                <template v-if="row.id && logisticsByItem.get(row.id)?.length">
                  <div v-for="t in logisticsByItem.get(row.id)" :key="t">{{ t }}</div>
                </template>
                <span v-else class="muted">未关联</span>
              </template>
            </el-table-column>
            <el-table-column prop="remark" label="商品备注" min-width="160" show-overflow-tooltip />
          </el-table>
        </el-tab-pane>

        <el-tab-pane label="物流" name="shipments" :disabled="!trackable">
          <PoShipmentTab v-if="trackable && po" :po-id="poId" :po="po" :readonly="false" @refresh="loadData" />
          <el-empty v-else description="提交下单后可维护物流" />
        </el-tab-pane>

        <el-tab-pane label="收款" name="payments" :disabled="!trackable">
          <ReceiptTab v-if="trackable && po" :po-id="poId" :po="po" :readonly="false" @refresh="loadData" />
          <el-empty v-else description="提交下单后可记录收款" />
        </el-tab-pane>

        <el-tab-pane label="附件" name="attachments" :disabled="!trackable">
          <PoAttachmentTab v-if="trackable" :po-id="poId" :readonly="false" />
          <el-empty v-else description="提交下单后可上传附件" />
        </el-tab-pane>
      </el-tabs>
    </el-card>
  </div>
</template>

<style scoped>
.po-detail {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.top-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: 8px;
}
.actions {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}
.header-row {
  display: flex;
  align-items: center;
  gap: 12px;
  font-weight: 600;
}
.section-title {
  margin: 20px 0 12px;
  font-size: 15px;
}
.muted {
  color: #c0c4cc;
}
.line-cancelled {
  color: #a8abb2;
  text-decoration: line-through;
}
.cancel-tag {
  margin-left: 6px;
}
:deep(.unit-price-col .cell) {
  overflow: visible;
}
.unit-price-input {
  width: 100%;
}
.unit-price-input :deep(.el-input__inner) {
  text-align: right;
}
:deep(.po-item-cancelled) {
  background-color: #f5f7fa !important;
  color: #a8abb2;
}
:deep(.po-item-cancelled td) {
  background-color: #f5f7fa !important;
}
.ref-orders {
  display: flex;
  flex-direction: column;
  gap: 2px;
  max-height: 16em;
  overflow-y: auto;
  line-height: 1.4;
}
.ref-order-line {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 4px 8px;
  font-size: 13px;
  color: #606266;
  word-break: break-all;
}
.ref-order-no.linkable {
  cursor: pointer;
  color: var(--el-color-primary);
}
.detach-btn {
  flex-shrink: 0;
}
.ref-count {
  margin-top: 6px;
  font-size: 12px;
  color: #909399;
}
</style>
