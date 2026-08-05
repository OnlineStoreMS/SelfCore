<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { OfficeBuilding, PriceTag, ShoppingCart, Box } from '@element-plus/icons-vue'
import {
  fetchDashboardStats,
  type DashboardStats,
} from '../api/dashboard'
import { DIST_STATUS_MAP, PAY_STATUS_MAP } from '../api/distOrder'
import { goDistOrders } from '../utils/distOrderListIntent'

const router = useRouter()
const loading = ref(false)
const stats = ref<DashboardStats | null>(null)

const emptyWorkbench = {
  dropshipPO: 0, wholesalePO: 0,
  draftPO: 0, confirmedPO: 0, unpaidPO: 0,
  inTransitPO: 0, partialReceivedPO: 0, activeOffers: 0,
  todayDropshipSaleAmount: 0, todayDropshipWholesaleAmount: 0, todayDropshipProfit: 0,
}
const emptyDistributor = { total: 0, active: 0, offerCount: 0, orderedThisMonth: 0 }
const emptyPO = {
  total: 0, draft: 0, inProgress: 0, completed: 0, cancelled: 0,
  todayCount: 0, weekCount: 0, monthCount: 0,
}
const emptyCost = { todayAmount: 0, weekAmount: 0, monthAmount: 0, unpaidAmount: 0, yearAmount: 0 }

const wb = computed(() => stats.value?.workbench ?? emptyWorkbench)
const distributor = computed(() => stats.value?.distributor ?? emptyDistributor)
const po = computed(() => stats.value?.purchaseOrder ?? emptyPO)
const cost = computed(() => stats.value?.cost ?? emptyCost)

function fmtMoney(n?: number) {
  return (Number(n) || 0).toFixed(2)
}

const workCards = computed(() => [
  {
    key: 'dropship',
    label: '代发订单',
    tip: '今日代发 · 点击查看',
    value: wb.value.dropshipPO,
    color: '#d48806',
    go: () => goDistOrders(router, {
      fulfillmentType: 'dropship',
      today: true,
      excludeStatuses: ['draft', 'cancelled'],
    }),
  },
  {
    key: 'wholesale',
    label: '批发订单',
    tip: '今日批发',
    value: wb.value.wholesalePO,
    color: '#1677ff',
    go: () => goDistOrders(router, {
      fulfillmentType: 'wholesale',
      today: true,
      excludeStatuses: ['draft', 'cancelled'],
    }),
  },
  {
    key: 'draft',
    label: '草稿待提交',
    tip: '今日草稿',
    value: wb.value.draftPO,
    color: '#64748b',
    go: () => goDistOrders(router, { status: 'draft', today: true }),
  },
  {
    key: 'confirmed',
    label: '已确认',
    tip: '今日已确认待推进',
    value: wb.value.confirmedPO,
    color: '#409eff',
    go: () => goDistOrders(router, { status: 'confirmed', today: true }),
  },
  {
    key: 'unpaid',
    label: '待收款',
    tip: '今日未收 / 部分收款',
    value: wb.value.unpaidPO,
    color: '#e6a23c',
    go: () => goDistOrders(router, {
      today: true,
      payStatuses: ['unpaid', 'partial'],
      excludeStatuses: ['draft', 'cancelled'],
    }),
  },
  {
    key: 'transit',
    label: '发货中',
    tip: '今日部分发货 / 已发货',
    value: wb.value.inTransitPO,
    color: '#0f766e',
    go: () => goDistOrders(router, {
      today: true,
      statuses: ['partial_shipped', 'shipped'],
    }),
  },
  {
    key: 'offers',
    label: '有效批发价',
    tip: 'SKU 批发价',
    value: wb.value.activeOffers,
    color: '#67c23a',
    go: () => router.push('/sku-prices'),
  },
])

const financeCards = computed(() => [
  {
    label: '今日毛利润',
    value: wb.value.todayDropshipProfit,
    tip: `代发销售额 ¥${fmtMoney(wb.value.todayDropshipSaleAmount)} − 批发 ¥${fmtMoney(wb.value.todayDropshipWholesaleAmount)}`,
  },
  {
    label: '今日代发销售额',
    value: wb.value.todayDropshipSaleAmount,
    tip: '有批发金额的代发单 · 订单实付',
  },
  {
    label: '今日批发额',
    value: cost.value.todayAmount,
    tip: '今日业务日全部分销金额',
  },
  {
    label: '待收金额',
    value: cost.value.unpaidAmount,
    tip: '未收 / 部分收款合计',
  },
])

const distributorCards = computed(() => [
  { label: '分销商总数', value: distributor.value.total, color: '#409eff', icon: OfficeBuilding },
  { label: '启用中', value: distributor.value.active, color: '#67c23a', icon: OfficeBuilding },
  { label: '本月有订单', value: distributor.value.orderedThisMonth, color: '#e6a23c', icon: ShoppingCart },
  { label: '批发价条数', value: distributor.value.offerCount, color: '#909399', icon: PriceTag },
])

const poCards = computed(() => [
  { label: '订单总数', value: po.value.total, sub: `进行中 ${po.value.inProgress}` },
  { label: '今日新单', value: po.value.todayCount, sub: `近7日 ${po.value.weekCount}` },
  { label: '本月订单', value: po.value.monthCount, sub: `已完成 ${po.value.completed}` },
  { label: '草稿 / 取消', value: po.value.draft, sub: `已取消 ${po.value.cancelled}` },
])

async function loadStats() {
  loading.value = true
  try {
    stats.value = await fetchDashboardStats()
  } catch (e) {
    ElMessage.error((e as Error).message || '加载工作台失败')
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  void loadStats()
})
</script>

<template>
  <div v-loading="loading" class="dashboard">
    <div class="head">
      <div>
        <h2 class="page-title">自营中心工作台</h2>
        <p class="desc">今日分销跟单与收款概览</p>
      </div>
      <el-button :icon="Box" @click="router.push('/self-orders')">自营订单</el-button>
    </div>

    <section class="section">
      <h3 class="section-title">今日工作</h3>
      <div class="card-grid">
        <button
          v-for="card in workCards"
          :key="card.key"
          type="button"
          class="stat-card"
          @click="card.go()"
        >
          <div class="stat-label">{{ card.label }}</div>
          <div class="stat-value" :style="{ color: card.color }">{{ card.value }}</div>
          <div class="stat-tip">{{ card.tip }}</div>
        </button>
      </div>
    </section>

    <section class="section">
      <h3 class="section-title">今日财务</h3>
      <div class="card-grid finance">
        <div v-for="card in financeCards" :key="card.label" class="stat-card static">
          <div class="stat-label">{{ card.label }}</div>
          <div class="stat-value">¥{{ fmtMoney(card.value) }}</div>
          <div class="stat-tip">{{ card.tip }}</div>
        </div>
      </div>
    </section>

    <section class="section">
      <h3 class="section-title">分销商</h3>
      <div class="card-grid">
        <div v-for="card in distributorCards" :key="card.label" class="stat-card static">
          <div class="stat-label">
            <el-icon :style="{ color: card.color }"><component :is="card.icon" /></el-icon>
            {{ card.label }}
          </div>
          <div class="stat-value" :style="{ color: card.color }">{{ card.value }}</div>
        </div>
      </div>
    </section>

    <section class="section">
      <h3 class="section-title">分销订单</h3>
      <div class="card-grid">
        <div v-for="card in poCards" :key="card.label" class="stat-card static">
          <div class="stat-label">{{ card.label }}</div>
          <div class="stat-value">{{ card.value }}</div>
          <div class="stat-tip">{{ card.sub }}</div>
        </div>
      </div>
    </section>

    <section v-if="stats?.recentOrders?.length" class="section">
      <h3 class="section-title">最近订单</h3>
      <el-table :data="stats.recentOrders" stripe size="small">
        <el-table-column prop="distNo" label="分销单号" min-width="140" />
        <el-table-column prop="distributorName" label="分销商" min-width="120" />
        <el-table-column label="状态" width="100">
          <template #default="{ row }">
            {{ DIST_STATUS_MAP[row.status]?.label || row.status }}
          </template>
        </el-table-column>
        <el-table-column label="收款" width="90">
          <template #default="{ row }">
            {{ PAY_STATUS_MAP[row.payStatus] || row.payStatus }}
          </template>
        </el-table-column>
        <el-table-column prop="totalAmount" label="金额" width="100" align="right" />
        <el-table-column label="操作" width="80" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" @click="router.push(`/dist-orders/${row.id}`)">详情</el-button>
          </template>
        </el-table-column>
      </el-table>
    </section>
  </div>
</template>

<style scoped>
.dashboard { width: 100%; }
.head {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 16px;
  margin-bottom: 20px;
}
.page-title { margin: 0 0 6px; font-size: 22px; }
.desc { color: #606266; margin: 0; }
.section { margin-bottom: 24px; }
.section-title {
  margin: 0 0 12px;
  font-size: 15px;
  font-weight: 600;
  color: #303133;
}
.card-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(150px, 1fr));
  gap: 12px;
}
.card-grid.finance {
  grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
}
.stat-card {
  text-align: left;
  border: 1px solid #ebeef5;
  background: #fff;
  border-radius: 8px;
  padding: 14px 16px;
  cursor: pointer;
  transition: border-color 0.15s, box-shadow 0.15s;
}
.stat-card:hover {
  border-color: #c6e2ff;
  box-shadow: 0 2px 8px rgba(64, 158, 255, 0.12);
}
.stat-card.static { cursor: default; }
.stat-card.static:hover { border-color: #ebeef5; box-shadow: none; }
.stat-label {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  color: #909399;
}
.stat-value {
  margin-top: 8px;
  font-size: 24px;
  font-weight: 650;
  color: #303133;
  line-height: 1.2;
}
.stat-tip {
  margin-top: 6px;
  font-size: 12px;
  color: #a8abb2;
  line-height: 1.4;
}
</style>
