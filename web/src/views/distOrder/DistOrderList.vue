<script setup lang="ts">
import { computed, onMounted, onUnmounted, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus, Search, View, Delete, Connection } from '@element-plus/icons-vue'
import {
  fetchDistOrders,
  fetchDistOrder,
  deleteDistOrder,
  mergeDistOrders,
  DIST_STATUS_MAP,
  PAY_STATUS_MAP,
  FULFILLMENT_TYPE_MAP,
  type DistOrderListItem,
  type DistOrder,
} from '../../api/distOrder'
import { fetchDistributors, type Distributor } from '../../api/distributor'
import { decryptOrders, fetchOrder, type OrderBrief } from '../../api/order'
import { copyToClipboard } from '../../utils/clipboard'
import {
  buildMultiOrderCopyText,
  canDecryptOrder,
  isMaskedReceiver,
} from '../../utils/orderCopy'
import { dateShortcuts, dateRangeDefaultTime, last7DaysDateTimeRange, todayDateTimeRange } from '../../utils/date'
import { onDistOrderListIntent, takeDistOrderListIntent } from '../../utils/distOrderListIntent'
import {
  loadDistOrderListFilters,
  saveDistOrderListFilters,
  type DistOrderListFilterSnapshot,
} from '../../utils/distOrderListFilters'

const route = useRoute()
const router = useRouter()
const tableData = ref<DistOrderListItem[]>([])
const distributors = ref<Distributor[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const status = ref('')
/** 多状态（工作台在途等），优先于 status */
const statusesFilter = ref('')
/** 收款状态（工作台待收款），逗号分隔 */
const payStatusFilter = ref('')
/** 排除状态（代发/采购/待收款卡片） */
const excludeStatusesFilter = ref('')
const distributorId = ref<number | undefined>()
const keyword = ref('')
const refSoId = ref<number | undefined>()
const fulfillmentType = ref('')
const loading = ref(false)
const selected = ref<DistOrderListItem[]>([])
const merging = ref(false)
const decryptRow = reactive<Record<number, boolean>>({})
const copyRow = reactive<Record<number, boolean>>({})
/** 默认按下单时间倒序 */
const sortBy = ref('orderedAt')
const sortOrder = ref<'asc' | 'desc'>('desc')
const defaultSort = { prop: 'orderedAt', order: 'descending' as const }

/** 创建时间范围，默认不限；与下单时间同时筛会过窄 */
const createdRange = ref<[string, string] | null>(null)
/** 下单时间范围，默认近 7 天（与订单中心一致）；清空后查全部 */
const orderedRange = ref<[string, string] | null>(last7DaysDateTimeRange())

/** 列表缩略：单号过多时只展示首单 + 共 N 单 */
function formatRefOrdersBrief(row: DistOrderListItem) {
  const trace = row.refTraceId?.trim()
  if (trace) {
    const nos = trace
      .split(/[,，;\s]+/)
      .map((s) => s.trim())
      .filter(Boolean)
    if (nos.length === 1) return nos[0]
    if (nos.length > 1) return `${nos[0]} 等共 ${nos.length} 单`
  }
  if (row.refSoId) return `#${row.refSoId}`
  return ''
}

function listPathForType(ft: string) {
  if (ft === 'dropship') return '/dist-orders/dropship'
  if (ft === 'wholesale') return '/dist-orders/wholesale'
  return '/dist-orders'
}

const activeTab = computed(() => fulfillmentType.value || 'all')

/** 用户手动切 Tab：先落盘当前筛选，再恢复目标 Tab 记忆 */
function onTypeTabChange(name: string | number) {
  const ft = String(name) === 'all' ? '' : String(name)
  persistFilters()
  fulfillmentType.value = ft
  page.value = 1
  selected.value = []
  const saved = loadDistOrderListFilters(ft || 'all')
  if (saved) {
    applyFilterSnapshot(saved)
  } else {
    status.value = ''
    statusesFilter.value = ''
    payStatusFilter.value = ''
    excludeStatusesFilter.value = ''
    distributorId.value = undefined
    keyword.value = ''
    refSoId.value = undefined
    createdRange.value = null
    orderedRange.value = last7DaysDateTimeRange()
    sortBy.value = 'orderedAt'
    sortOrder.value = 'desc'
  }
  const target = listPathForType(ft)
  if (route.path !== target) {
    void router.replace(target)
    persistFilters()
    return
  }
  void loadData()
}

const pageTitle = computed(() => {
  if (fulfillmentType.value === 'dropship') return '分销直发'
  if (fulfillmentType.value === 'wholesale') return '批发'
  return '分销订单'
})

const createLabel = computed(() => {
  if (fulfillmentType.value === 'dropship') return '新建分销直发单'
  if (fulfillmentType.value === 'wholesale') return '新建分销订单'
  return '新建分销订单'
})

const canMerge = computed(() => {
  if (selected.value.length < 2) return false
  if (!selected.value.every((r) => isMergeableDropship(r))) {
    return false
  }
  const sid = selected.value[0]?.distributorId
  return selected.value.every((r) => r.distributorId === sid)
})

function isMergeableDropship(row: DistOrderListItem) {
  if (row.fulfillmentType !== 'dropship') return false
  if (row.payStatus === 'paid' || row.payStatus === 'partial') return false
  return true
}

function resolveFulfillmentFromRoute(): string {
  const metaFt = route.meta.fulfillmentType
  if (typeof metaFt === 'string' && metaFt) return metaFt
  if (route.path.endsWith('/dropship')) return 'dropship'
  if (route.path.endsWith('/wholesale')) return 'wholesale'
  return ''
}

function filterSnapshot(): DistOrderListFilterSnapshot {
  return {
    status: status.value,
    statusesFilter: statusesFilter.value,
    payStatusFilter: payStatusFilter.value,
    excludeStatusesFilter: excludeStatusesFilter.value,
    distributorId: distributorId.value,
    keyword: keyword.value,
    refSoId: refSoId.value,
    createdRange: createdRange.value,
    orderedRange: orderedRange.value,
    page: page.value,
    pageSize: pageSize.value,
    sortBy: sortBy.value,
    sortOrder: sortOrder.value,
  }
}

function persistFilters() {
  saveDistOrderListFilters(fulfillmentType.value || 'all', filterSnapshot())
}

function applyFilterSnapshot(s: DistOrderListFilterSnapshot) {
  status.value = s.status || ''
  statusesFilter.value = s.statusesFilter || ''
  payStatusFilter.value = s.payStatusFilter || ''
  excludeStatusesFilter.value = s.excludeStatusesFilter || ''
  distributorId.value = s.distributorId
  keyword.value = s.keyword || ''
  refSoId.value = s.refSoId
  createdRange.value = s.createdRange ?? null
  orderedRange.value = s.orderedRange ?? null
  page.value = s.page > 0 ? s.page : 1
  pageSize.value = s.pageSize > 0 ? s.pageSize : 20
  sortBy.value = s.sortBy || 'orderedAt'
  sortOrder.value = s.sortOrder === 'asc' ? 'asc' : 'desc'
}

function resetFilterFields() {
  status.value = ''
  statusesFilter.value = ''
  payStatusFilter.value = ''
  excludeStatusesFilter.value = ''
  distributorId.value = undefined
  keyword.value = ''
  refSoId.value = undefined
  createdRange.value = null
  orderedRange.value = last7DaysDateTimeRange()
  page.value = 1
  sortBy.value = 'orderedAt'
  sortOrder.value = 'desc'
}

/** 从路径 / meta / 进入意图初始化筛选；无意图则恢复记忆 */
function applyRouteContext() {
  fulfillmentType.value = resolveFulfillmentFromRoute()

  const intent = takeDistOrderListIntent()
  const q = route.query
  const hasQuery = !!(q.refSoId || q.status || q.fulfillmentType)

  if (intent) {
    resetFilterFields()
    if (intent.fulfillmentType) {
      fulfillmentType.value = intent.fulfillmentType
    }
    if (intent.statuses?.length) {
      if (intent.statuses.length === 1) {
        status.value = intent.statuses[0]
      } else {
        statusesFilter.value = intent.statuses.join(',')
      }
    } else if (intent.status) {
      status.value = intent.status
    }
    if (intent.payStatuses?.length) {
      payStatusFilter.value = intent.payStatuses.join(',')
    }
    if (intent.excludeStatuses?.length) {
      excludeStatusesFilter.value = intent.excludeStatuses.join(',')
    }
    if (intent.orderedDateStart && intent.orderedDateEnd) {
      orderedRange.value = [
        `${intent.orderedDateStart} 00:00:00`,
        `${intent.orderedDateEnd} 23:59:59`,
      ]
      createdRange.value = null
    } else if (intent.today) {
      orderedRange.value = todayDateTimeRange()
      createdRange.value = null
    }
    if (intent.refSoId) {
      refSoId.value = intent.refSoId
      createdRange.value = null
      orderedRange.value = null
    }
    persistFilters()
  } else if (hasQuery) {
    resetFilterFields()
    if (q.refSoId) {
      refSoId.value = Number(q.refSoId)
      createdRange.value = null
      orderedRange.value = null
    }
    if (typeof q.status === 'string' && q.status) {
      status.value = q.status
    }
    if (typeof q.fulfillmentType === 'string' && q.fulfillmentType && !fulfillmentType.value) {
      fulfillmentType.value = q.fulfillmentType
    }
    persistFilters()
  } else {
    const saved = loadDistOrderListFilters(fulfillmentType.value || 'all')
    if (saved) applyFilterSnapshot(saved)
  }

  if (Object.keys(q).length > 0) {
    void router.replace({ path: listPathForType(fulfillmentType.value), query: {} })
  }
}

async function loadDistributors() {
  try {
    const data = await fetchDistributors({ page: 1, pageSize: 200 })
    distributors.value = data.list
  } catch {
    distributors.value = []
  }
}

async function loadData() {
  loading.value = true
  try {
    const data = await fetchDistOrders({
      status: statusesFilter.value ? undefined : (status.value || undefined),
      statuses: statusesFilter.value || undefined,
      payStatus: payStatusFilter.value || undefined,
      excludeStatuses: excludeStatusesFilter.value || undefined,
      fulfillmentType: fulfillmentType.value || undefined,
      distributorId: distributorId.value,
      refSoId: refSoId.value,
      keyword: keyword.value || undefined,
      createdAtStart: createdRange.value?.[0] || undefined,
      createdAtEnd: createdRange.value?.[1] || undefined,
      orderedAtStart: orderedRange.value?.[0] || undefined,
      orderedAtEnd: orderedRange.value?.[1] || undefined,
      sortBy: sortBy.value || undefined,
      sortOrder: sortOrder.value || undefined,
      page: page.value,
      pageSize: pageSize.value,
    })
    tableData.value = data.list
    total.value = data.total
    selected.value = []
    persistFilters()
  } catch (e) {
    ElMessage.error((e as Error).message || '加载失败')
  } finally {
    loading.value = false
  }
}

onMounted(async () => {
  applyRouteContext()
  await loadDistributors()
  await loadData()
})

const stopIntentListen = onDistOrderListIntent(() => {
  applyRouteContext()
  page.value = 1
  void loadData()
})
onUnmounted(() => stopIntentListen())

watch(
  () => route.path,
  () => {
    if (!route.path.startsWith('/dist-orders')) return
    if (route.path.includes('/create') || /\/\d+/.test(route.path)) return
    applyRouteContext()
    void loadData()
  },
)

const intentTags = computed(() => {
  const tags: { key: string; label: string; clear: () => void }[] = []
  if (excludeStatusesFilter.value) {
    const labels = excludeStatusesFilter.value
      .split(',')
      .map((s) => statusLabel(s.trim()))
      .filter(Boolean)
    if (labels.length) {
      tags.push({
        key: 'exclude',
        label: `排除：${labels.join(' / ')}`,
        clear: () => {
          excludeStatusesFilter.value = ''
          page.value = 1
          void loadData()
        },
      })
    }
  }
  if (statusesFilter.value) {
    const labels = statusesFilter.value
      .split(',')
      .map((s) => statusLabel(s.trim()))
      .filter(Boolean)
    if (labels.length) {
      tags.push({
        key: 'statuses',
        label: `状态：${labels.join(' / ')}`,
        clear: () => {
          statusesFilter.value = ''
          page.value = 1
          void loadData()
        },
      })
    }
  }
  if (payStatusFilter.value) {
    const labels = payStatusFilter.value
      .split(',')
      .map((s) => PAY_STATUS_MAP[s.trim()] || s.trim())
      .filter(Boolean)
    if (labels.length) {
      tags.push({
        key: 'pay',
        label: `收款：${labels.join(' / ')}`,
        clear: () => {
          payStatusFilter.value = ''
          page.value = 1
          void loadData()
        },
      })
    }
  }
  return tags
})

function statusLabel(s: string) {
  return DIST_STATUS_MAP[s]?.label || s
}

function statusType(s: string) {
  return DIST_STATUS_MAP[s]?.type || 'info'
}

function fulfillmentLabel(t: string) {
  return FULFILLMENT_TYPE_MAP[t] || t || '—'
}

function canDelete(row: DistOrderListItem) {
  return row.status !== 'completed'
}

function collectRefSoIds(po: DistOrder): number[] {
  const ids = new Set<number>()
  if (po.refSoId && po.refSoId > 0) ids.add(po.refSoId)
  for (const it of po.items || []) {
    if (it.refSoId && it.refSoId > 0) ids.add(it.refSoId)
  }
  return [...ids]
}

async function loadLinkedOrders(row: DistOrderListItem): Promise<OrderBrief[]> {
  const po = await fetchDistOrder(row.id)
  const ids = collectRefSoIds(po)
  if (!ids.length) {
    throw new Error('该代发单未关联销售订单')
  }
  const orders: OrderBrief[] = []
  for (const id of ids) {
    orders.push(await fetchOrder(id))
  }
  return orders
}

async function handleDecrypt(row: DistOrderListItem) {
  if (row.fulfillmentType !== 'dropship') return
  decryptRow[row.id] = true
  try {
    const orders = await loadLinkedOrders(row)
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
    decryptRow[row.id] = false
  }
}

async function handleCopy(row: DistOrderListItem) {
  if (row.fulfillmentType !== 'dropship') return
  copyRow[row.id] = true
  try {
    let orders = await loadLinkedOrders(row)
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
    copyRow[row.id] = false
  }
}

function openDetail(row: DistOrderListItem) {
  persistFilters()
  router.push(`/dist-orders/${row.id}`)
}

function openCreate() {
  const query: Record<string, string> = {}
  if (fulfillmentType.value) query.fulfillmentType = fulfillmentType.value
  router.push({ path: '/dist-orders/create', query })
}

function onFilterChange() {
  // 手动改筛选时，单状态选择覆盖多状态意图
  if (status.value) {
    statusesFilter.value = ''
  }
  page.value = 1
  void loadData()
}

function onSortChange(payload: { prop: string; order: string | null }) {
  if (!payload.order) {
    sortBy.value = 'orderedAt'
    sortOrder.value = 'desc'
  } else {
    sortBy.value = payload.prop === 'createdAt' ? 'createdAt' : 'orderedAt'
    sortOrder.value = payload.order === 'ascending' ? 'asc' : 'desc'
  }
  page.value = 1
  void loadData()
}

function onSelectionChange(rows: DistOrderListItem[]) {
  selected.value = rows
}

async function handleMerge() {
  if (!canMerge.value) {
    ElMessage.warning('请选择同一分销商下至少 2 张未收款代发单（草稿或已确认）')
    return
  }
  const nos = selected.value.map((r) => r.distNo).join('、')
  try {
    await ElMessageBox.confirm(
      `将合并以下代发单为第一张：${nos}。销售订单关联会一并更新。`,
      '合并代发单',
      { type: 'warning', confirmButtonText: '合并' },
    )
  } catch {
    return
  }
  merging.value = true
  try {
    const ids = selected.value.map((r) => r.id)
    const result = await mergeDistOrders({
      sourceDistOrderIds: ids,
      targetDistOrderId: ids[0],
    })
    ElMessage.success(`已合并为 ${result.distNo}` + (result.relinked ? `（回写 ${result.relinked} 笔销售单）` : ''))
    selected.value = []
    await loadData()
  } catch (e) {
    ElMessage.error((e as Error).message || '合并失败')
  } finally {
    merging.value = false
  }
}

async function handleDelete(row: DistOrderListItem) {
  try {
    await ElMessageBox.confirm(
      `确定删除分销订单「${row.distNo}」？相关物流、收款、附件等记录将一并删除。`,
      '确认删除',
      { type: 'warning', confirmButtonText: '删除', confirmButtonClass: 'el-button--danger' },
    )
  } catch {
    return
  }
  try {
    await deleteDistOrder(row.id)
    ElMessage.success('已删除')
    await loadData()
  } catch (e) {
    ElMessage.error((e as Error).message || '删除失败')
  }
}
</script>

<template>
  <div class="po-page">
    <el-card v-loading="loading">
      <template #header>
        <span>{{ pageTitle }}</span>
        <div class="header-actions">
          <el-button
            :icon="Connection"
            :disabled="!canMerge"
            :loading="merging"
            @click="handleMerge"
          >
            合并代发单
          </el-button>
          <el-button type="primary" :icon="Plus" @click="openCreate">{{ createLabel }}</el-button>
        </div>
      </template>

      <el-tabs :model-value="activeTab" class="type-tabs" @tab-change="onTypeTabChange">
        <el-tab-pane label="全部" name="all" />
        <el-tab-pane label="分销直发" name="dropship" />
        <el-tab-pane label="批发" name="wholesale" />
      </el-tabs>

      <div class="toolbar">
        <el-form inline @submit.prevent>
          <el-form-item>
            <el-input
              v-model="keyword"
              placeholder="订单号"
              :prefix-icon="Search"
              clearable
              style="width: 180px"
              @change="onFilterChange"
            />
          </el-form-item>
          <el-form-item label="状态">
            <el-select
              v-model="status"
              placeholder="状态"
              clearable
              style="width: 130px"
              @change="onFilterChange"
            >
              <el-option v-for="(v, k) in DIST_STATUS_MAP" :key="k" :label="v.label" :value="k" />
            </el-select>
          </el-form-item>
          <el-form-item label="分销商">
            <el-select
              v-model="distributorId"
              placeholder="分销商"
              clearable
              filterable
              style="width: 160px"
              @change="onFilterChange"
            >
              <el-option v-for="s in distributors" :key="s.id" :label="s.name" :value="s.id" />
            </el-select>
          </el-form-item>
          <el-form-item label="下单时间">
            <el-date-picker
              v-model="orderedRange"
              type="datetimerange"
              range-separator="至"
              start-placeholder="开始"
              end-placeholder="结束"
              value-format="YYYY-MM-DD HH:mm:ss"
              :shortcuts="dateShortcuts"
              :default-time="dateRangeDefaultTime"
              clearable
              style="width: 360px"
              @change="onFilterChange"
            />
          </el-form-item>
          <el-form-item label="创建时间">
            <el-date-picker
              v-model="createdRange"
              type="datetimerange"
              range-separator="至"
              start-placeholder="开始"
              end-placeholder="结束"
              value-format="YYYY-MM-DD HH:mm:ss"
              :shortcuts="dateShortcuts"
              :default-time="dateRangeDefaultTime"
              clearable
              style="width: 360px"
              @change="onFilterChange"
            />
          </el-form-item>
        </el-form>
        <div v-if="intentTags.length" class="intent-tags">
          <el-tag
            v-for="tag in intentTags"
            :key="tag.key"
            closable
            type="warning"
            effect="plain"
            @close="tag.clear()"
          >
            {{ tag.label }}
          </el-tag>
        </div>
      </div>

      <el-table
        :data="tableData"
        stripe
        border
        :default-sort="defaultSort"
        @selection-change="onSelectionChange"
        @sort-change="onSortChange"
      >
        <el-table-column
          type="selection"
          width="48"
          :selectable="(row: DistOrderListItem) => isMergeableDropship(row)"
        />
        <el-table-column prop="distNo" label="订单号" width="150">
          <template #default="{ row }">
            <el-link type="primary" @click="openDetail(row)">{{ row.distNo }}</el-link>
          </template>
        </el-table-column>
        <el-table-column prop="distributorName" label="分销商" width="100" show-overflow-tooltip />
        <el-table-column label="订单类型" width="110" align="center">
          <template #default="{ row }">
            <el-tag :type="row.fulfillmentType === 'dropship' ? 'warning' : ''" size="small">
              {{ fulfillmentLabel(row.fulfillmentType) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="关联订单" width="120" show-overflow-tooltip>
          <template #default="{ row }">
            <span v-if="formatRefOrdersBrief(row)">{{ formatRefOrdersBrief(row) }}</span>
            <span v-else class="muted">—</span>
          </template>
        </el-table-column>
        <el-table-column label="规格" min-width="160" show-overflow-tooltip>
          <template #default="{ row }">
            <span v-if="row.skuSpecs">{{ row.skuSpecs }}</span>
            <span v-else class="muted">—</span>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="100" align="center">
          <template #default="{ row }">
            <el-tag :type="statusType(row.status)" size="small">{{ statusLabel(row.status) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="收款" width="90" align="center">
          <template #default="{ row }">
            {{ PAY_STATUS_MAP[row.payStatus] || row.payStatus }}
          </template>
        </el-table-column>
        <el-table-column label="金额" width="120" align="right">
          <template #default="{ row }">¥{{ row.totalAmount.toFixed(2) }}</template>
        </el-table-column>
        <el-table-column prop="itemCount" label="行数" width="70" align="center" />
        <el-table-column prop="orderedAt" label="下单时间" width="160" sortable="custom">
          <template #default="{ row }">{{ row.orderedAt || '—' }}</template>
        </el-table-column>
        <el-table-column prop="createdAt" label="创建时间" width="160" sortable="custom" />
        <el-table-column label="操作" width="220" fixed="right">
          <template #default="{ row }">
            <el-button type="primary" link :icon="View" @click="openDetail(row)">详情</el-button>
            <template v-if="row.fulfillmentType === 'dropship'">
              <el-button
                type="warning"
                link
                :loading="decryptRow[row.id]"
                @click="handleDecrypt(row)"
              >
                解密
              </el-button>
              <el-button
                type="primary"
                link
                :loading="copyRow[row.id]"
                @click="handleCopy(row)"
              >
                复制
              </el-button>
            </template>
            <el-button
              v-if="canDelete(row)"
              type="danger"
              link
              :icon="Delete"
              @click="handleDelete(row)"
            >
              删除
            </el-button>
          </template>
        </el-table-column>
      </el-table>

      <div class="pager">
        <el-pagination
          v-model:current-page="page"
          :page-size="pageSize"
          :total="total"
          layout="total, prev, pager, next"
          @current-change="(p: number) => { page = p; loadData() }"
        />
      </div>
    </el-card>
  </div>
</template>

<style scoped>
.po-page :deep(.el-card__header) {
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.header-actions {
  display: flex;
  gap: 8px;
  align-items: center;
}
.type-tabs {
  margin-bottom: 4px;
}
.toolbar {
  display: flex;
  flex-wrap: wrap;
  margin-bottom: 12px;
  gap: 8px;
  align-items: flex-start;
}
.toolbar :deep(.el-form-item) {
  margin-bottom: 8px;
}
.intent-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: center;
  width: 100%;
  margin-top: 4px;
}
.pager {
  margin-top: 16px;
  display: flex;
  justify-content: flex-end;
}
.muted {
  color: #c0c4cc;
}
</style>
