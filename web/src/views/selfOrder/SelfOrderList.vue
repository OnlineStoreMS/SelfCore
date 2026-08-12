<script setup lang="ts">
import { computed, onMounted, onUnmounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { View, Delete } from '@element-plus/icons-vue'
import {
  listSelfOrders,
  getSelfOrder,
  deleteSelfOrder,
  SELF_ORDER_STATUS_MAP,
  SELF_ORDER_DOC_STATUS_OPTIONS,
  SELF_SHIP_STATUS_MAP,
  deriveSelfDocStatus,
  deriveSelfShipStatus,
  type SelfOrderListItem,
  type SelfOrderDetail,
} from '../../api/selfOrder'
import { SELF_PAY_STATUS_MAP } from '../../api/selfOrderTracking'
import { decryptOrders, fetchOrder, formatDateTime, formatPlatformShop, formatRemarkLines, labelSource, type OrderBrief } from '../../api/order'
import { dateShortcuts, dateRangeDefaultTime, last7DaysDateTimeRange, todayDateTimeRange } from '../../utils/date'
import { copyToClipboard } from '../../utils/clipboard'
import {
  buildMultiOrderCopyText,
  canDecryptOrder,
  isMaskedReceiver,
} from '../../utils/orderCopy'
import { onSelfOrderListIntent, takeSelfOrderListIntent } from '../../utils/selfOrderListIntent'

const router = useRouter()
const loading = ref(false)
const list = ref<SelfOrderListItem[]>([])
const total = ref(0)
const decryptRow = reactive<Record<number, boolean>>({})
const copyRow = reactive<Record<number, boolean>>({})
const statusFilter = ref('')
const statusesFilter = ref('')
const payStatusesFilter = ref('')
const excludeStatusesFilter = ref('')

const filters = reactive({
  page: 1,
  pageSize: 20,
  keyword: '',
  shipStatus: '',
  status: '',
  orderedRange: last7DaysDateTimeRange() as [string, string] | null,
  shippedRange: null as [string, string] | null,
})

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
  return ship ? (SELF_SHIP_STATUS_MAP[ship]?.label || ship) : '—'
}

function shipStatusType(s: string) {
  const ship = deriveSelfShipStatus(s)
  return ship ? (SELF_SHIP_STATUS_MAP[ship]?.type || 'info') : 'info'
}

function formatRefOrder(row: SelfOrderListItem) {
  const trace = row.refTraceId?.trim()
  if (trace) return trace
  if (row.refSoId) return `#${row.refSoId}`
  return '—'
}

function remarkLines(row: SelfOrderListItem) {
  return formatRemarkLines({
    remark: row.buyerRemark,
    sellerRemark: row.sellerRemark,
    fenFaRemark: row.fenFaRemark,
    printerRemark: row.printerRemark,
  })
}

function canDelete(row: SelfOrderListItem) {
  return row.status !== 'completed'
}

function collectRefSoIds(detail: SelfOrderDetail): number[] {
  const ids = new Set<number>()
  if (detail.refSoId && detail.refSoId > 0) ids.add(detail.refSoId)
  for (const it of detail.items || []) {
    if (it.refSoId && it.refSoId > 0) ids.add(it.refSoId)
  }
  return [...ids]
}

async function loadLinkedOrders(row: SelfOrderListItem): Promise<OrderBrief[]> {
  const detail = await getSelfOrder(row.id)
  const ids = collectRefSoIds(detail)
  if (!ids.length) {
    throw new Error('该自营单未关联销售订单')
  }
  const orders: OrderBrief[] = []
  for (const id of ids) {
    orders.push(await fetchOrder(id))
  }
  return orders
}

async function handleDecrypt(row: SelfOrderListItem) {
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

async function handleCopy(row: SelfOrderListItem) {
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

async function handleDelete(row: SelfOrderListItem) {
  try {
    await ElMessageBox.confirm(
      `确定删除自营订单「${row.soNo}」？相关物流、附件等记录将一并删除。`,
      '确认删除',
      { type: 'warning', confirmButtonText: '删除', confirmButtonClass: 'el-button--danger' },
    )
  } catch {
    return
  }
  try {
    await deleteSelfOrder(row.id)
    ElMessage.success('已删除')
    await load()
  } catch (e) {
    ElMessage.error((e as Error).message || '删除失败')
  }
}

function openDetail(row: SelfOrderListItem) {
  router.push(`/self-orders/${row.id}`)
}

function applyListIntent() {
  const intent = takeSelfOrderListIntent()
  if (!intent) return
  statusFilter.value = ''
  statusesFilter.value = ''
  payStatusesFilter.value = ''
  excludeStatusesFilter.value = ''
  filters.status = ''
  filters.shipStatus = ''
  if (intent.statuses?.length) {
    if (intent.statuses.length === 1) {
      statusFilter.value = intent.statuses[0]
      filters.status = intent.statuses[0]
    } else {
      statusesFilter.value = intent.statuses.join(',')
    }
  } else if (intent.status) {
    statusFilter.value = intent.status
    filters.status = intent.status
  }
  if (intent.payStatuses?.length) {
    payStatusesFilter.value = intent.payStatuses.join(',')
  }
  if (intent.shipStatus) {
    filters.shipStatus = intent.shipStatus
  }
  if (intent.excludeStatuses?.length) {
    excludeStatusesFilter.value = intent.excludeStatuses.join(',')
  }
  if (intent.orderedDateStart && intent.orderedDateEnd) {
    filters.orderedRange = [
      `${intent.orderedDateStart} 00:00:00`,
      `${intent.orderedDateEnd} 23:59:59`,
    ]
  } else if (intent.today) {
    filters.orderedRange = todayDateTimeRange()
  }
  filters.page = 1
}

async function load() {
  loading.value = true
  try {
    const params: Record<string, unknown> = {
      page: filters.page,
      pageSize: filters.pageSize,
      keyword: filters.keyword || undefined,
      shipStatus: filters.shipStatus || undefined,
      status: statusesFilter.value ? undefined : (statusFilter.value || filters.status || undefined),
      statuses: statusesFilter.value || undefined,
      payStatus: payStatusesFilter.value || undefined,
      excludeStatuses: excludeStatusesFilter.value || undefined,
    }
    if (filters.orderedRange?.length === 2) {
      params.orderedAtStart = filters.orderedRange[0]
      params.orderedAtEnd = filters.orderedRange[1]
    }
    if (filters.shippedRange?.length === 2) {
      params.shippedAtStart = filters.shippedRange[0]
      params.shippedAtEnd = filters.shippedRange[1]
    }
    const data = await listSelfOrders(params as Parameters<typeof listSelfOrders>[0])
    list.value = data.list || []
    total.value = data.total || 0
  } catch (e) {
    ElMessage.error((e as Error).message || '加载失败')
  } finally {
    loading.value = false
  }
}

function onFilterChange() {
  filters.page = 1
  void load()
}

function onStatusFilterChange() {
  statusFilter.value = filters.status
  statusesFilter.value = ''
  onFilterChange()
}

function resetFilters() {
  filters.keyword = ''
  filters.shipStatus = ''
  filters.status = ''
  statusFilter.value = ''
  statusesFilter.value = ''
  payStatusesFilter.value = ''
  excludeStatusesFilter.value = ''
  filters.orderedRange = last7DaysDateTimeRange()
  filters.shippedRange = null
  onFilterChange()
}

const intentTags = computed(() => {
  const tags: { key: string; label: string; clear: () => void }[] = []
  if (statusFilter.value || filters.status) {
    const s = statusFilter.value || filters.status
    tags.push({
      key: 'status',
      label: `状态：${statusLabel(s)}`,
      clear: () => {
        statusFilter.value = ''
        filters.status = ''
        filters.page = 1
        void load()
      },
    })
  }
  if (statusesFilter.value) {
    const labels = statusesFilter.value.split(',').map((s) => statusLabel(s.trim())).filter(Boolean)
    tags.push({
      key: 'statuses',
      label: `状态：${labels.join(' / ')}`,
      clear: () => {
        statusesFilter.value = ''
        filters.page = 1
        void load()
      },
    })
  }
  if (payStatusesFilter.value) {
    const labels = payStatusesFilter.value
      .split(',')
      .map((s) => SELF_PAY_STATUS_MAP[s.trim()] || s.trim())
      .filter(Boolean)
    tags.push({
      key: 'pay',
      label: `付款：${labels.join(' / ')}`,
      clear: () => {
        payStatusesFilter.value = ''
        filters.page = 1
        void load()
      },
    })
  }
  if (filters.shipStatus) {
    const label = SELF_SHIP_STATUS_MAP[filters.shipStatus]?.label || filters.shipStatus
    tags.push({
      key: 'ship',
      label: `发货：${label}`,
      clear: () => {
        filters.shipStatus = ''
        filters.page = 1
        void load()
      },
    })
  }
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
          filters.page = 1
          void load()
        },
      })
    }
  }
  return tags
})

onMounted(() => {
  applyListIntent()
  void load()
})

const stopIntentListen = onSelfOrderListIntent(() => {
  applyListIntent()
  void load()
})
onUnmounted(() => stopIntentListen())
</script>

<template>
  <div class="page">
    <div class="head">
      <h2 class="page-title">自营订单</h2>
    </div>
    <p class="desc">
      本地自营履约单据，承接订单中心「自营发货」分配；发货、扣库与物流回传在此处理。
    </p>

    <div class="toolbar">
      <el-form :inline="true" class="filters" @submit.prevent="onFilterChange">
        <el-form-item label="单据状态">
          <el-select
            v-model="filters.status"
            clearable
            style="width: 120px"
            @change="onStatusFilterChange"
          >
            <el-option
              v-for="opt in SELF_ORDER_DOC_STATUS_OPTIONS"
              :key="opt.value"
              :label="opt.label"
              :value="opt.value"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="发货状态">
          <el-select v-model="filters.shipStatus" clearable style="width: 120px" @change="onFilterChange">
            <el-option label="待发货" value="wait_ship" />
            <el-option label="部分发货" value="partial_shipped" />
            <el-option label="已发货" value="shipped" />
          </el-select>
        </el-form-item>
        <el-form-item label="下单时间">
          <el-date-picker
            v-model="filters.orderedRange"
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
        <el-form-item label="发货时间">
          <el-date-picker
            v-model="filters.shippedRange"
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
        <el-form-item>
          <el-input
            v-model="filters.keyword"
            clearable
            placeholder="单号/关联销售单"
            style="width: 180px"
            @keyup.enter="onFilterChange"
          />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="onFilterChange">查询</el-button>
          <el-button @click="resetFilters">重置</el-button>
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
      v-loading="loading"
      :data="list"
      stripe
      class="table"
      :default-sort="{ prop: 'orderedAt', order: 'descending' }"
      @row-click="openDetail"
    >
      <el-table-column prop="soNo" label="自营单号" min-width="130" show-overflow-tooltip />
      <el-table-column label="订单类型" width="80" align="center">
        <template #default="{ row }">{{ labelSource(row.sourceChannel) }}</template>
      </el-table-column>
      <el-table-column label="平台" min-width="120" show-overflow-tooltip>
        <template #default="{ row }">{{ formatPlatformShop(row) }}</template>
      </el-table-column>
      <el-table-column label="关联订单" min-width="120" show-overflow-tooltip>
        <template #default="{ row }">
          <span v-if="formatRefOrder(row) !== '—'">{{ formatRefOrder(row) }}</span>
          <span v-else class="muted">—</span>
        </template>
      </el-table-column>
      <el-table-column label="规格" min-width="200" show-overflow-tooltip>
        <template #default="{ row }">
          <span v-if="row.skuSpecs">{{ row.skuSpecs }}</span>
          <span v-else class="muted">—</span>
        </template>
      </el-table-column>
      <el-table-column label="留言备注" min-width="140">
        <template #default="{ row }">
          <div v-if="remarkLines(row).length" class="remark-lines">
            <div v-for="(line, idx) in remarkLines(row)" :key="idx">{{ line }}</div>
          </div>
          <span v-else class="muted">—</span>
        </template>
      </el-table-column>
      <el-table-column label="单据状态" width="88" align="center">
        <template #default="{ row }">
          <el-tag size="small" :type="statusType(row.status)">{{ statusLabel(row.status) }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="发货状态" width="88" align="center">
        <template #default="{ row }">
          <el-tag
            v-if="deriveSelfShipStatus(row.status)"
            size="small"
            :type="shipStatusType(row.status)"
          >{{ shipStatusLabel(row.status) }}</el-tag>
          <span v-else class="muted">—</span>
        </template>
      </el-table-column>
      <el-table-column label="金额" width="96" align="right">
        <template #default="{ row }">
          ¥{{ Number(row.saleAmount || 0).toFixed(2) }}
        </template>
      </el-table-column>
      <el-table-column label="付款" width="84" align="center">
        <template #default="{ row }">
          {{ SELF_PAY_STATUS_MAP[row.payStatus || 'unpaid'] || row.payStatus || '未付清' }}
        </template>
      </el-table-column>
      <el-table-column prop="itemCount" label="行数" width="56" align="center" />
      <el-table-column label="扣库" width="72" align="center">
        <template #default="{ row }">
          <el-tag v-if="row.stockDeducted" size="small" type="success">已扣</el-tag>
          <el-tag v-else-if="row.stockError" size="small" type="danger">失败</el-tag>
          <span v-else class="muted">—</span>
        </template>
      </el-table-column>
      <el-table-column label="下单时间" prop="orderedAt" width="168" fixed="right" class-name="col-nowrap">
        <template #default="{ row }">{{ formatDateTime(row.orderedAt || row.createdAt) }}</template>
      </el-table-column>
      <el-table-column label="操作" width="200" fixed="right">
        <template #default="{ row }">
          <el-button link type="primary" :icon="View" @click.stop="openDetail(row)">详情</el-button>
          <el-button
            type="warning"
            link
            :loading="decryptRow[row.id]"
            @click.stop="handleDecrypt(row)"
          >
            解密
          </el-button>
          <el-button
            type="primary"
            link
            :loading="copyRow[row.id]"
            @click.stop="handleCopy(row)"
          >
            复制
          </el-button>
          <el-button
            v-if="canDelete(row)"
            type="danger"
            link
            :icon="Delete"
            @click.stop="handleDelete(row)"
          >
            删除
          </el-button>
        </template>
      </el-table-column>
    </el-table>

    <div class="pager">
      <el-pagination
        v-model:current-page="filters.page"
        v-model:page-size="filters.pageSize"
        :total="total"
        layout="total, prev, pager, next"
        @current-change="load"
      />
    </div>
  </div>
</template>

<style scoped>
.page {
  width: 100%;
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}
.page-title {
  margin: 0;
  font-size: 22px;
}
.desc {
  margin: 0;
  color: #606266;
  line-height: 1.5;
}
.toolbar {
  background: #fff;
  border: 1px solid #eef0f3;
  border-radius: 8px;
  padding: 10px 12px 0;
}
.filters :deep(.el-form-item) {
  margin-bottom: 10px;
}
.intent-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  padding: 0 0 10px;
}
.table {
  background: #fff;
}
.table :deep(.col-nowrap .cell) {
  white-space: nowrap;
}
.remark-lines {
  font-size: 12px;
  line-height: 1.45;
  color: #606266;
  white-space: pre-wrap;
  word-break: break-word;
}
.muted {
  color: #909399;
}
.pager {
  display: flex;
  justify-content: flex-end;
  padding: 8px 0;
}
</style>
