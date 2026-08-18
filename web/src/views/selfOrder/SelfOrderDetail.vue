<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { ArrowLeft } from '@element-plus/icons-vue'
import {
  getSelfOrder,
  listSelfShipments,
  retryCallback,
  retryStock,
  submitSelfOrder,
  markSelfOrderPaid,
  completeSelfOrder,
  cancelSelfOrder,
  bindInvSku,
  updateItemCost,
  searchWarehouseSkus,
  SELF_ORDER_STATUS_MAP,
  SELF_SHIP_STATUS_MAP,
  deriveSelfDocStatus,
  deriveSelfShipStatus,
  type SelfOrderDetail,
  type SelfOrderItem,
  type WarehouseSku,
} from '../../api/selfOrder'
import type { SelfShipment } from '../../api/selfOrderTracking'
import SelfShipmentTab from './SelfShipmentTab.vue'
import SelfPaymentTab from './SelfPaymentTab.vue'
import { SELF_PAY_STATUS_MAP } from '../../api/selfOrderTracking'
import { orderCoreOrderUrl } from '../../utils/runtimeConfig'
import { buildSelfItemTreeRows, selfItemTreeTitle } from '../../utils/selfItemTree'

const route = useRoute()
const router = useRouter()
const soId = computed(() => Number(route.params.id))
const activeTab = ref('overview')

const loading = ref(false)
const acting = ref(false)
const order = ref<SelfOrderDetail | null>(null)
const shipments = ref<SelfShipment[]>([])

const bindDialogVisible = ref(false)
const bindSearching = ref(false)
const bindSaving = ref(false)
const bindKeyword = ref('')
const bindResults = ref<WarehouseSku[]>([])
const bindTargetItem = ref<SelfOrderItem | null>(null)

function statusLabel(s: string) {
  const doc = deriveSelfDocStatus(s)
  return SELF_ORDER_STATUS_MAP[doc]?.label || SELF_ORDER_STATUS_MAP[s]?.label || s
}

function statusType(s: string) {
  const doc = deriveSelfDocStatus(s)
  return SELF_ORDER_STATUS_MAP[doc]?.type || SELF_ORDER_STATUS_MAP[s]?.type || 'info'
}

function shipStatusLabel(s: string) {
  const ship = deriveSelfShipStatus(s)
  return ship ? (SELF_SHIP_STATUS_MAP[ship]?.label || ship) : ''
}

function shipStatusType(s: string) {
  const ship = deriveSelfShipStatus(s)
  return ship ? (SELF_SHIP_STATUS_MAP[ship]?.type || 'info') : 'info'
}

function openOrderCore(id?: number) {
  const target = id && id > 0 ? id : order.value?.refSoId
  if (!target) return
  window.open(orderCoreOrderUrl(target), '_blank', 'noopener,noreferrer')
}

/** 商品明细行 → 关联物流单号（与 SupplyCore 一致） */
const logisticsByItem = computed(() => {
  const map = new Map<number, string[]>()
  for (const sh of shipments.value) {
    const tracking = [sh.carrierName, sh.trackingNo].filter(Boolean).join(' ')
    if (!tracking) continue
    for (const it of sh.items || []) {
      const arr = map.get(it.selfOrderItemId) || []
      if (!arr.includes(tracking)) arr.push(tracking)
      map.set(it.selfOrderItemId, arr)
    }
  }
  return map
})

const itemTreeRows = computed(() => buildSelfItemTreeRows(order.value?.items))


const canRetryStock = computed(() => {
  if (!order.value) return false
  if (order.value.stockDeducted) return false
  const hasCallbackOk = shipments.value.some((s) => s.callbackOk)
  return hasCallbackOk && (!!order.value.stockError || ['partial_shipped', 'shipped', 'completed'].includes(order.value.status))
})

const canCancel = computed(() => {
  if (!order.value) return false
  return ['draft', 'ordered', 'confirmed'].includes(order.value.status)
})

const isDraft = computed(() => order.value?.status === 'draft')
/** 完成/取消前可改成本、绑库存 SKU（含已下单手工单） */
const canEditCost = computed(() => {
  const s = order.value?.status
  return !!s && !['completed', 'cancelled'].includes(s)
})
/** 电商订单默认已付款，不展示付款 Tab / 标记付款 */
const isEcommerce = computed(() => (order.value?.sourceChannel || '').toLowerCase() === 'kdzs')
const showPaymentTab = computed(() => !!order.value && !isEcommerce.value)

const costSaving = ref(false)

async function loadData() {
  loading.value = true
  try {
    order.value = await getSelfOrder(soId.value)
    shipments.value = await listSelfShipments(soId.value) as SelfShipment[]
    if (isEcommerce.value && activeTab.value === 'payments') {
      activeTab.value = 'overview'
    }
  } catch (e) {
    ElMessage.error((e as Error).message || '加载失败')
  } finally {
    loading.value = false
  }
}

onMounted(loadData)

async function handleRetryStock() {
  acting.value = true
  try {
    order.value = await retryStock(soId.value)
    ElMessage.success('扣库成功')
  } catch (e) {
    ElMessage.error((e as Error).message || '扣库失败')
    await loadData()
  } finally {
    acting.value = false
  }
}

async function handleRetryCallback(shipmentId?: number) {
  acting.value = true
  try {
    order.value = await retryCallback(soId.value, shipmentId)
    shipments.value = await listSelfShipments(soId.value) as SelfShipment[]
    if (order.value.stockError) {
      ElMessage.warning(`已回传订单中心，但扣库失败：${order.value.stockError}，可重试扣库`)
    } else if (order.value.stockDeducted) {
      ElMessage.success('已回传订单中心并扣减仓储库存')
    } else {
      ElMessage.success('已回传订单中心')
    }
  } catch (e) {
    ElMessage.error((e as Error).message || '回传失败')
  } finally {
    acting.value = false
  }
}

const pendingCallbackShipment = computed(() =>
  shipments.value.find((s) => !s.callbackOk && !!s.trackingNo),
)

const canRetryCallback = computed(() => !!pendingCallbackShipment.value)

async function doAction(label: string, fn: () => Promise<SelfOrderDetail>) {
  acting.value = true
  try {
    order.value = await fn()
    ElMessage.success(`${label}成功`)
  } catch (e) {
    ElMessage.error((e as Error).message || `${label}失败`)
  } finally {
    acting.value = false
  }
}

function itemMissingCost(it: SelfOrderItem) {
  return !(Number(it.costUnitPrice) > 0 || Number(it.costAmount) > 0)
}

async function handleSubmit() {
  if (!order.value) return
  const missing = (order.value.items || []).filter(itemMissingCost)
  if (missing.length) {
    const name = missing[0].skuSpecs || missing[0].productName || '商品'
    ElMessage.warning(`请先填写成本价或绑定库存 SKU：${name}`)
    return
  }
  await doAction('提交下单', () => submitSelfOrder(soId.value))
}

async function onCostChange(row: SelfOrderItem, val: number | undefined) {
  if (!row.id || !canEditCost.value) return
  const price = Number(val)
  if (Number.isNaN(price) || price < 0) {
    ElMessage.warning('成本单价不能为负')
    return
  }
  costSaving.value = true
  try {
    order.value = await updateItemCost(row.id, price)
  } catch (e) {
    ElMessage.error((e as Error).message || '保存成本失败')
    await loadData()
  } finally {
    costSaving.value = false
  }
}

async function handleCancel() {
  try {
    await ElMessageBox.confirm('确定取消此自营单？', '确认')
  } catch {
    return
  }
  await doAction('取消', () => cancelSelfOrder(soId.value))
}

function openBindDialog(row: SelfOrderItem) {
  bindTargetItem.value = row
  bindKeyword.value = ''
  bindResults.value = []
  bindDialogVisible.value = true
}

async function searchBindSkus() {
  bindSearching.value = true
  try {
    const data = await searchWarehouseSkus({
      keyword: bindKeyword.value.trim() || undefined,
      page: 1,
      pageSize: 20,
    })
    bindResults.value = data.list || []
  } catch (e) {
    ElMessage.error((e as Error).message || '搜索失败')
  } finally {
    bindSearching.value = false
  }
}

async function confirmBindSku(sku: WarehouseSku) {
  if (!bindTargetItem.value?.id) return
  bindSaving.value = true
  try {
    order.value = await bindInvSku(bindTargetItem.value.id, {
      invSkuId: sku.id,
      invSkuCode: sku.skuCode,
      costUnitPrice: sku.lastPurchasePrice,
    })
    bindDialogVisible.value = false
    ElMessage.success('已绑定库存 SKU')
  } catch (e) {
    ElMessage.error((e as Error).message || '绑定失败')
  } finally {
    bindSaving.value = false
  }
}
</script>

<template>
  <div v-loading="loading" class="so-detail">
    <div class="top-bar">
      <el-button :icon="ArrowLeft" text @click="router.push('/self-orders')">返回列表</el-button>
      <div v-if="order" class="actions">
        <el-button
          v-if="order.status === 'draft'"
          type="primary"
          :loading="acting"
          @click="handleSubmit"
        >提交下单</el-button>
        <el-button
          v-if="!isEcommerce && (order.status === 'ordered' || order.status === 'confirmed')"
          type="warning"
          :loading="acting"
          @click="doAction('标记已付款', () => markSelfOrderPaid(soId))"
        >快捷标记已付款</el-button>
        <el-button
          v-if="['paid', 'partial_shipped', 'shipped'].includes(order.status)"
          type="success"
          :loading="acting"
          @click="doAction('完成', () => completeSelfOrder(soId))"
        >完成</el-button>
        <el-button
          v-if="canRetryCallback"
          type="warning"
          plain
          :loading="acting"
          @click="handleRetryCallback(pendingCallbackShipment?.id)"
        >重试回传</el-button>
        <el-button
          v-if="canRetryStock"
          type="warning"
          plain
          :loading="acting"
          @click="handleRetryStock"
        >重试扣库</el-button>
        <el-button
          v-if="canCancel"
          type="danger"
          plain
          :loading="acting"
          @click="handleCancel"
        >取消</el-button>
      </div>
    </div>

    <el-card v-if="order">
      <template #header>
        <div class="header-row">
          <span>{{ order.soNo }}</span>
          <el-tag :type="statusType(order.status)">{{ statusLabel(order.status) }}</el-tag>
          <el-tag
            v-if="shipStatusLabel(order.status)"
            :type="shipStatusType(order.status)"
          >{{ shipStatusLabel(order.status) }}</el-tag>
        </div>
      </template>

      <el-tabs v-model="activeTab">
        <el-tab-pane label="基本信息" name="overview">
          <el-alert
            v-if="order.stockError"
            type="error"
            :title="`扣库失败：${order.stockError}`"
            show-icon
            :closable="false"
            class="stock-alert"
          />

          <el-descriptions :column="2" border>
            <el-descriptions-item label="自营单号">{{ order.soNo }}</el-descriptions-item>
            <el-descriptions-item label="单据状态">
              <el-tag size="small" :type="statusType(order.status)">{{ statusLabel(order.status) }}</el-tag>
            </el-descriptions-item>
            <el-descriptions-item label="发货状态">
              <el-tag
                v-if="shipStatusLabel(order.status)"
                size="small"
                :type="shipStatusType(order.status)"
              >{{ shipStatusLabel(order.status) }}</el-tag>
              <span v-else class="muted">—</span>
            </el-descriptions-item>
            <el-descriptions-item label="仓库">{{ order.warehouseId || '—' }}</el-descriptions-item>
            <el-descriptions-item label="扣库">
              <el-tag v-if="order.stockDeducted" size="small" type="success">已扣库</el-tag>
              <el-tag v-else-if="order.stockError" size="small" type="danger">失败</el-tag>
              <span v-else class="muted">未扣</span>
              <el-button
                v-if="canRetryStock"
                link
                type="warning"
                size="small"
                :loading="acting"
                class="retry-btn"
                @click="handleRetryStock"
              >重试</el-button>
            </el-descriptions-item>
            <el-descriptions-item label="销售单号">
              <el-link
                v-if="order.refSoId || order.refTraceId"
                type="primary"
                @click="openOrderCore()"
              >{{ order.refTraceId || `#${order.refSoId}` }}</el-link>
              <span v-else>—</span>
            </el-descriptions-item>
            <el-descriptions-item label="销售金额">
              ¥{{ Number(order.saleAmount || 0).toFixed(2) }}
            </el-descriptions-item>
            <el-descriptions-item label="成本金额">
              ¥{{ Number(order.costAmount || 0).toFixed(2) }}
            </el-descriptions-item>
            <el-descriptions-item label="付款状态">
              {{ SELF_PAY_STATUS_MAP[order.payStatus || 'unpaid'] || order.payStatus || '未付款' }}
            </el-descriptions-item>
            <el-descriptions-item label="付款时间">{{ order.paidAt || '—' }}</el-descriptions-item>
            <el-descriptions-item label="买家">{{ order.buyerName || '—' }}</el-descriptions-item>
            <el-descriptions-item label="手机">{{ order.buyerPhone || '—' }}</el-descriptions-item>
            <el-descriptions-item label="地址" :span="2">{{ order.address || '—' }}</el-descriptions-item>
            <el-descriptions-item label="创建时间">{{ order.createdAt || '—' }}</el-descriptions-item>
            <el-descriptions-item label="下单时间">{{ order.orderedAt || '—' }}</el-descriptions-item>
            <el-descriptions-item label="发货时间">{{ order.shippedAt || '—' }}</el-descriptions-item>
            <el-descriptions-item label="备注" :span="2">{{ order.remark || '—' }}</el-descriptions-item>
          </el-descriptions>

          <h4 class="section-title">商品明细</h4>
          <el-table :data="itemTreeRows" border stripe row-key="key">
            <el-table-column label="图片" width="72" align="center">
              <template #default="{ row }">
                <el-image
                  v-if="!row.fullGroupHeader && row.item.picUrl"
                  :src="row.item.picUrl"
                  :preview-src-list="[row.item.picUrl]"
                  fit="cover"
                  style="width: 40px; height: 40px; border-radius: 4px"
                  preview-teleported
                />
                <span v-else class="muted">—</span>
              </template>
            </el-table-column>
            <el-table-column label="销售单" width="140" show-overflow-tooltip>
              <template #default="{ row }">
                <span v-if="row.isSplitChild || row.fullGroupHeader" class="muted">└</span>
                <template v-else>
                  <span
                    v-if="row.item.refSoId"
                    class="link-text"
                    @click="openOrderCore(row.item.refSoId)"
                  >{{ row.item.refOrderNo || `#${row.item.refSoId}` }}</span>
                  <span v-else>{{ row.item.refOrderNo || '—' }}</span>
                </template>
              </template>
            </el-table-column>
            <el-table-column label="规格" min-width="240" show-overflow-tooltip>
              <template #default="{ row }">
                <div class="spec-cell" :class="{ child: row.isSplitChild || row.fullGroupHeader }">
                  <span v-if="row.isSplitChild" class="tree-prefix">└</span>
                  <span>{{ selfItemTreeTitle(row) }}</span>
                  <el-tag v-if="row.isSplitParent" size="small" type="warning" effect="plain" class="split-tag">已拆分</el-tag>
                  <el-tag v-else-if="row.isSplitChild" size="small" type="info" effect="plain" class="split-tag">拆分</el-tag>
                </div>
              </template>
            </el-table-column>
            <el-table-column label="库存 SKU" width="180">
              <template #default="{ row }">
                <div v-if="!row.isSplitChild && !row.fullGroupHeader" class="inv-sku-cell">
                  <span v-if="row.item.invSkuCode">{{ row.item.invSkuCode }}</span>
                  <span v-else class="muted">未绑定</span>
                  <el-button
                    v-if="canEditCost"
                    link
                    type="primary"
                    size="small"
                    @click="openBindDialog(row.item)"
                  >{{ row.item.invSkuCode ? '更换' : '绑定' }}</el-button>
                </div>
                <span v-else class="muted">—</span>
              </template>
            </el-table-column>
            <el-table-column label="数量" width="70" align="center">
              <template #default="{ row }">{{ row.fullGroupHeader ? '—' : row.item.qty }}</template>
            </el-table-column>
            <el-table-column label="实付金额" width="100" align="right">
              <template #default="{ row }">
                <span v-if="!row.isSplitChild && !row.fullGroupHeader && row.item.saleAmount > 0">¥{{ Number(row.item.saleAmount).toFixed(2) }}</span>
                <span v-else class="muted">—</span>
              </template>
            </el-table-column>
            <el-table-column label="成本单价" width="130" align="right">
              <template #default="{ row }">
                <template v-if="!row.isSplitChild && !row.fullGroupHeader">
                  <el-input-number
                    v-if="canEditCost"
                    :model-value="Number(row.item.costUnitPrice || 0)"
                    :min="0"
                    :precision="2"
                    :step="0.01"
                    controls-position="right"
                    size="small"
                    style="width: 118px"
                    :disabled="costSaving"
                    @change="(v: number | undefined) => onCostChange(row.item, v)"
                  />
                  <span v-else>¥{{ Number(row.item.costUnitPrice || 0).toFixed(2) }}</span>
                </template>
                <span v-else class="muted">—</span>
              </template>
            </el-table-column>
            <el-table-column label="成本小计" width="100" align="right">
              <template #default="{ row }">
                <span v-if="!row.isSplitChild && !row.fullGroupHeader">¥{{ Number(row.item.costAmount || 0).toFixed(2) }}</span>
                <span v-else class="muted">—</span>
              </template>
            </el-table-column>
            <el-table-column label="物流" min-width="140" show-overflow-tooltip>
              <template #default="{ row }">
                <template v-if="row.item.id && logisticsByItem.get(row.item.id)?.length">
                  <div v-for="t in logisticsByItem.get(row.item.id)" :key="t">{{ t }}</div>
                </template>
                <span v-else-if="row.isSplitParent" class="muted">见拆分行</span>
                <span v-else class="muted">未发货</span>
              </template>
            </el-table-column>
            <el-table-column label="备注" min-width="120" show-overflow-tooltip>
              <template #default="{ row }">{{ row.isSplitChild || row.fullGroupHeader ? '—' : (row.item.remark || '—') }}</template>
            </el-table-column>
          </el-table>
        </el-tab-pane>

        <el-tab-pane label="物流" name="shipments">
          <el-alert
            v-if="canRetryCallback"
            type="warning"
            :closable="false"
            show-icon
            class="stock-alert"
            title="本地物流已登记，订单中心回传未成功，请重试回传；回传成功后会自动扣库"
          />
          <el-alert
            v-else-if="order.stockError"
            type="error"
            :title="`已回传订单中心，扣库失败可重试：${order.stockError}`"
            show-icon
            :closable="false"
            class="stock-alert"
          />
          <el-empty v-if="isDraft" description="提交下单后可登记物流" />
          <SelfShipmentTab
            v-else-if="order"
            :self-order-id="soId"
            :order="order"
            :readonly="order.status === 'cancelled'"
            @refresh="loadData"
          />
        </el-tab-pane>

        <el-tab-pane
          v-if="showPaymentTab"
          label="付款"
          name="payments"
          :disabled="order.status === 'cancelled' || isDraft"
        >
          <el-empty v-if="isDraft" description="提交下单后可记录付款" />
          <SelfPaymentTab
            v-else-if="order"
            :self-order-id="soId"
            :order="order"
            :readonly="order.status === 'cancelled'"
            @refresh="loadData"
          />
        </el-tab-pane>
      </el-tabs>
    </el-card>

    <el-dialog v-model="bindDialogVisible" title="绑定库存 SKU" width="520px" destroy-on-close>
      <el-form @submit.prevent="searchBindSkus">
        <el-form-item label="关键词">
          <div class="bind-search">
            <el-input v-model="bindKeyword" clearable placeholder="SKU 编码 / 名称" @keyup.enter="searchBindSkus" />
            <el-button type="primary" :loading="bindSearching" @click="searchBindSkus">搜索</el-button>
          </div>
        </el-form-item>
      </el-form>
      <el-table v-loading="bindSearching" :data="bindResults" border stripe max-height="320">
        <el-table-column prop="skuCode" label="编码" width="120" />
        <el-table-column label="名称" min-width="160" show-overflow-tooltip>
          <template #default="{ row }">{{ row.name || row.pickName || '—' }}</template>
        </el-table-column>
        <el-table-column label="最近进价" width="100" align="right">
          <template #default="{ row }">¥{{ Number(row.lastPurchasePrice || 0).toFixed(2) }}</template>
        </el-table-column>
        <el-table-column label="操作" width="80" align="center">
          <template #default="{ row }">
            <el-button link type="primary" :loading="bindSaving" @click="confirmBindSku(row)">选择</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-dialog>
  </div>
</template>

<style scoped>
.so-detail {
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
.stock-alert {
  margin-bottom: 16px;
}
.muted {
  color: #c0c4cc;
}
.link-text {
  color: var(--el-color-primary);
  cursor: pointer;
}
.retry-btn {
  margin-left: 8px;
}
.bind-search {
  display: flex;
  gap: 8px;
  width: 100%;
}
.inv-sku-cell {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-wrap: wrap;
}

.spec-cell {
  display: flex;
  align-items: center;
  gap: 6px;
}
.spec-cell.child {
  color: var(--el-text-color-regular);
}
.tree-prefix {
  color: var(--el-text-color-placeholder);
}
.split-tag {
  flex-shrink: 0;
}
</style>
