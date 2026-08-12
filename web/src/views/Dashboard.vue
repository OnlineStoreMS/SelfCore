<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { OfficeBuilding, PriceTag, ShoppingCart } from '@element-plus/icons-vue'
import * as echarts from 'echarts/core'
import { LineChart, BarChart } from 'echarts/charts'
import { GridComponent, TooltipComponent, LegendComponent } from 'echarts/components'
import { CanvasRenderer } from 'echarts/renderers'
import {
  fetchDashboardStats,
  fetchDashboardTrend,
  type DashboardStats,
  type DashboardTrend,
  type DashboardTrendPoint,
} from '../api/dashboard'
import { DIST_STATUS_MAP, PAY_STATUS_MAP } from '../api/distOrder'
import { goDistOrders } from '../utils/distOrderListIntent'
import { goSelfOrders } from '../utils/selfOrderListIntent'

echarts.use([LineChart, BarChart, GridComponent, TooltipComponent, LegendComponent, CanvasRenderer])

const router = useRouter()
const loading = ref(false)
const trendLoading = ref(false)
const stats = ref<DashboardStats | null>(null)
const trend = ref<DashboardTrend | null>(null)

function startOfDay(d = new Date()) {
  const x = new Date(d)
  x.setHours(0, 0, 0, 0)
  return x
}

function toYMD(d: Date) {
  const y = d.getFullYear()
  const m = String(d.getMonth() + 1).padStart(2, '0')
  const day = String(d.getDate()).padStart(2, '0')
  return `${y}-${m}-${day}`
}

function defaultRange(): [string, string] {
  const end = startOfDay()
  const start = new Date(end)
  start.setDate(start.getDate() - 6)
  return [toYMD(start), toYMD(end)]
}

const dateRange = ref<[string, string]>(defaultRange())

const emptyWorkbench = {
  selfOrderPO: 0, selfUnpaidPO: 0, selfDraftPO: 0, selfWaitShipPO: 0,
  distOrderPO: 0, dropshipPO: 0, wholesalePO: 0,
  draftPO: 0, unpaidPO: 0, distWaitShipPO: 0,
  orderedPO: 0, confirmedPO: 0, inTransitPO: 0, partialReceivedPO: 0, activeOffers: 0,
  todayDropshipSaleAmount: 0, todayDropshipWholesaleAmount: 0, todayDropshipProfit: 0,
  todayDistSaleAmount: 0, todaySelfSaleAmount: 0,
  weekSelfSaleAmount: 0, monthSelfSaleAmount: 0, monthDistSaleAmount: 0,
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
const trendPoints = computed<DashboardTrendPoint[]>(() => trend.value?.points ?? [])

function fmtMoney(n?: number) {
  return (Number(n) || 0).toFixed(2)
}

type WorkCard = {
  key: string
  label: string
  tip: string
  value: number
  color: string
  highlight?: boolean
  go: () => void
}

/** 第一行：自营 */
const selfWorkCards = computed<WorkCard[]>(() => [
  {
    key: 'self',
    label: '自营订单',
    tip: '今日自营 · 排除已取消',
    value: wb.value.selfOrderPO,
    color: '#d48806',
    highlight: true,
    go: () => goSelfOrders(router, {
      today: true,
      excludeStatuses: ['cancelled'],
    }),
  },
  {
    key: 'self-unpaid',
    label: '自营待收款',
    tip: '今日未收 / 部分收款',
    value: wb.value.selfUnpaidPO,
    color: '#e6a23c',
    highlight: true,
    go: () => goSelfOrders(router, {
      today: true,
      payStatuses: ['unpaid', 'partial'],
      excludeStatuses: ['draft', 'cancelled'],
    }),
  },
  {
    key: 'self-draft',
    label: '自营草稿待提交',
    tip: '今日自营草稿',
    value: wb.value.selfDraftPO,
    color: '#64748b',
    go: () => goSelfOrders(router, { status: 'draft', today: true }),
  },
  {
    key: 'self-wait-ship',
    label: '自营待发货',
    tip: '今日已下单待发货',
    value: wb.value.selfWaitShipPO,
    color: '#0f766e',
    go: () => goSelfOrders(router, {
      today: true,
      shipStatus: 'wait_ship',
    }),
  },
])

/** 第二行：分销（全类型） */
const distWorkCards = computed<WorkCard[]>(() => [
  {
    key: 'dist',
    label: '分销订单',
    tip: '今日全部分销类型 · 排除已取消',
    value: wb.value.distOrderPO,
    color: '#722ed1',
    highlight: true,
    go: () => goDistOrders(router, {
      today: true,
      excludeStatuses: ['cancelled'],
    }),
  },
  {
    key: 'dist-unpaid',
    label: '分销待收款',
    tip: '今日未收 / 部分收款',
    value: wb.value.unpaidPO,
    color: '#e6a23c',
    highlight: true,
    go: () => goDistOrders(router, {
      today: true,
      payStatuses: ['unpaid', 'partial'],
      excludeStatuses: ['draft', 'cancelled'],
    }),
  },
  {
    key: 'dist-draft',
    label: '分销草稿待提交',
    tip: '今日分销草稿',
    value: wb.value.draftPO,
    color: '#64748b',
    go: () => goDistOrders(router, { status: 'draft', today: true }),
  },
  {
    key: 'dist-wait-ship',
    label: '分销待发货',
    tip: '今日已确认 / 已付款',
    value: wb.value.distWaitShipPO,
    color: '#0f766e',
    go: () => goDistOrders(router, {
      today: true,
      statuses: ['confirmed', 'paid'],
    }),
  },
])

const financeCards = computed(() => [
	{
    label: '今日毛利润',
    value: wb.value.todayDropshipProfit,
    tip: `有成本销售额 ¥${fmtMoney(wb.value.todayDropshipSaleAmount)} − 成本 ¥${fmtMoney(wb.value.todayDropshipWholesaleAmount)}（成本为 0 不计入）`,
    highlight: true,
    accent: 'profit' as const,
  },
  {
    label: '今日自营销售额',
    value: wb.value.todaySelfSaleAmount,
    tip: '今日自营单实付合计',
  },
  {
    label: '今日成本额',
    value: cost.value.todayAmount,
    tip: '今日自营成本 + 分销成本',
  },
  {
    label: '今日分销销售额',
    value: wb.value.todayDistSaleAmount,
    tip: '全部分销类型 · 直发实付 / 批发额',
  },
  {
    label: '近7日自营销售额',
    value: wb.value.weekSelfSaleAmount,
    tip: '近 7 日自营销售额累计',
  },
  {
    label: '本月自营销售额',
    value: wb.value.monthSelfSaleAmount,
    tip: '本月自营销售额累计',
  },
  {
    label: '本月分销销售额',
    value: wb.value.monthDistSaleAmount,
    tip: '本月全部分销类型销售额',
  },
  {
    label: '待收金额',
    value: cost.value.unpaidAmount,
    tip: '自营 + 分销 · 待收余额（订单额 − 已付）',
  },
])

const rangeSummaryCards = computed(() => {
  const t = trend.value
  return [
    {
      label: '区间订单量',
      value: String(t?.orderCount ?? 0),
      tip: '自营 + 全部分销 · 排除草稿/取消',
      color: '#d48806',
    },
    {
      label: '区间销售额',
      value: `¥${fmtMoney(t?.saleAmount)}`,
      tip: '自营 + 全部分销销售额',
      color: '#1677ff',
    },
    {
      label: '区间成本额',
      value: `¥${fmtMoney(t?.wholesaleAmount)}`,
      tip: '自营成本 + 分销成本',
      color: '#13c2c2',
    },
    {
      label: '区间毛利润',
      value: `¥${fmtMoney(t?.profit)}`,
      tip: '仅成本额 > 0 的订单 · 销售额 − 成本额',
      color: '#059669',
    },
  ]
})

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

const pickerShortcuts = [
  {
    text: '今天',
    value: () => {
      const d = startOfDay()
      return [d, d] as [Date, Date]
    },
  },
  {
    text: '昨天',
    value: () => {
      const d = startOfDay()
      d.setDate(d.getDate() - 1)
      return [d, d] as [Date, Date]
    },
  },
  {
    text: '最近7天',
    value: () => {
      const end = startOfDay()
      const start = new Date(end)
      start.setDate(start.getDate() - 6)
      return [start, end] as [Date, Date]
    },
  },
  {
    text: '最近14天',
    value: () => {
      const end = startOfDay()
      const start = new Date(end)
      start.setDate(start.getDate() - 13)
      return [start, end] as [Date, Date]
    },
  },
  {
    text: '最近30天',
    value: () => {
      const end = startOfDay()
      const start = new Date(end)
      start.setDate(start.getDate() - 29)
      return [start, end] as [Date, Date]
    },
  },
  {
    text: '本月',
    value: () => {
      const end = startOfDay()
      const start = new Date(end.getFullYear(), end.getMonth(), 1)
      return [start, end] as [Date, Date]
    },
  },
]

const volumeChartEl = ref<HTMLDivElement | null>(null)
const profitChartEl = ref<HTMLDivElement | null>(null)
let volumeChart: echarts.ECharts | null = null
let profitChart: echarts.ECharts | null = null

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

async function loadTrend() {
  if (!dateRange.value || dateRange.value.length !== 2) {
    ElMessage.warning('请选择时间范围')
    return
  }
  const [startDate, endDate] = dateRange.value
  trendLoading.value = true
  try {
    trend.value = await fetchDashboardTrend({ startDate, endDate })
    await nextTick()
    renderCharts()
  } catch (e) {
    ElMessage.error((e as Error).message || '趋势加载失败')
  } finally {
    trendLoading.value = false
  }
}

function axisDates() {
  return trendPoints.value.map((t) => (t.date.length >= 10 ? t.date.slice(5) : t.date))
}

function moneyAxisLabel(v: number) {
  return v >= 10000 ? `${(v / 10000).toFixed(1)}万` : String(v)
}

function renderCharts() {
  const dates = axisDates()
  const selfCounts = trendPoints.value.map((t) => Number(t.selfOrderCount || 0))
  const selfSales = trendPoints.value.map((t) => Number(t.selfSaleAmount || 0))
  const sales = trendPoints.value.map((t) => Number(t.saleAmount || 0))
  const costs = trendPoints.value.map((t) => Number(t.wholesaleAmount || 0))
  const profits = trendPoints.value.map((t) => Number(t.profit || 0))
  const moneyFmt = (v: number) => `¥${fmtMoney(v)}`

  if (volumeChartEl.value) {
    if (!volumeChart) volumeChart = echarts.init(volumeChartEl.value)
    volumeChart.setOption({
      color: ['#d48806', '#1677ff'],
      legend: { data: ['自营单量', '销售额'], top: 0 },
      tooltip: { trigger: 'axis', axisPointer: { type: 'cross' } },
      grid: { left: 48, right: 56, top: 40, bottom: 28 },
      xAxis: { type: 'category', data: dates, boundaryGap: true },
      yAxis: [
        { type: 'value', name: '单', minInterval: 1 },
        {
          type: 'value',
          name: '元',
          splitLine: { show: false },
          axisLabel: { formatter: moneyAxisLabel },
        },
      ],
      series: [
        {
          name: '自营单量',
          type: 'line',
          smooth: true,
          yAxisIndex: 0,
          areaStyle: { opacity: 0.08 },
          data: selfCounts,
        },
        {
          name: '销售额',
          type: 'bar',
          yAxisIndex: 1,
          barMaxWidth: 28,
          data: selfSales,
          tooltip: { valueFormatter: moneyFmt },
        },
      ],
    }, true)
  }

  if (profitChartEl.value) {
    if (!profitChart) profitChart = echarts.init(profitChartEl.value)
    profitChart.setOption({
      color: ['#1677ff', '#13c2c2', '#059669'],
      legend: { data: ['销售额', '成本额', '毛利润'], top: 0 },
      tooltip: { trigger: 'axis' },
      grid: { left: 48, right: 24, top: 40, bottom: 28 },
      xAxis: { type: 'category', data: dates, boundaryGap: false },
      yAxis: {
        type: 'value',
        name: '元',
        axisLabel: { formatter: moneyAxisLabel },
      },
      series: [
        {
          name: '销售额',
          type: 'line',
          smooth: true,
          data: sales,
          tooltip: { valueFormatter: moneyFmt },
        },
        {
          name: '成本额',
          type: 'line',
          smooth: true,
          data: costs,
          tooltip: { valueFormatter: moneyFmt },
        },
        {
          name: '毛利润',
          type: 'line',
          smooth: true,
          areaStyle: { opacity: 0.08 },
          data: profits,
          tooltip: { valueFormatter: moneyFmt },
        },
      ],
    }, true)
  }
}

function onResize() {
  volumeChart?.resize()
  profitChart?.resize()
}

onMounted(() => {
  void loadStats()
  void loadTrend()
  window.addEventListener('resize', onResize)
})

onUnmounted(() => {
  window.removeEventListener('resize', onResize)
  volumeChart?.dispose()
  profitChart?.dispose()
  volumeChart = null
  profitChart = null
})
</script>

<template>
  <div v-loading="loading" class="dashboard">
    <div class="section-head">工作场景 · 今日</div>
    <div class="work-rows">
      <div class="work-row">
        <div class="work-row-tag self">自营</div>
        <div class="work-cards">
          <button
            v-for="card in selfWorkCards"
            :key="card.key"
            type="button"
            class="work-card"
            :class="{ highlight: card.highlight && card.value > 0 }"
            :style="{ '--accent': card.color }"
            @click="card.go()"
          >
            <div class="work-label">{{ card.label }}</div>
            <div class="work-value">{{ card.value }}</div>
            <div class="work-tip">{{ card.tip }}</div>
          </button>
        </div>
      </div>
      <div class="work-row">
        <div class="work-row-tag dist">分销</div>
        <div class="work-cards">
          <button
            v-for="card in distWorkCards"
            :key="card.key"
            type="button"
            class="work-card"
            :class="{ highlight: card.highlight && card.value > 0 }"
            :style="{ '--accent': card.color }"
            @click="card.go()"
          >
            <div class="work-label">{{ card.label }}</div>
            <div class="work-value">{{ card.value }}</div>
            <div class="work-tip">{{ card.tip }}</div>
          </button>
        </div>
      </div>
    </div>

    <div class="section-head">成本与毛利</div>
    <div class="metric-row finance-row">
      <div
        v-for="card in financeCards"
        :key="card.label"
        class="metric-card"
        :class="{ highlight: card.highlight, profit: card.accent === 'profit' }"
      >
        <div class="metric-label">{{ card.label }}</div>
        <div class="metric-value">¥{{ fmtMoney(card.value) }}</div>
        <div class="metric-tip">{{ card.tip }}</div>
      </div>
    </div>

    <div class="section-head row-between">
      <span>趋势分析</span>
      <div class="range-tools">
        <el-date-picker
          v-model="dateRange"
          type="daterange"
          unlink-panels
          range-separator="至"
          start-placeholder="开始日期"
          end-placeholder="结束日期"
          value-format="YYYY-MM-DD"
          :shortcuts="pickerShortcuts"
          :clearable="false"
          @change="loadTrend"
        />
      </div>
    </div>

    <div v-loading="trendLoading" class="trend-block">
      <div class="channel-sales-row">
        <div
          v-for="m in rangeSummaryCards"
          :key="m.label"
          class="channel-sales-card"
          :style="{ '--accent': m.color }"
        >
          <div class="metric-label">{{ m.label }}</div>
          <div class="channel-sales-value">{{ m.value }}</div>
          <div class="metric-tip">{{ m.tip }}</div>
        </div>
      </div>

      <div class="charts">
        <section>
          <h3>自营单量 / 销售额</h3>
          <p class="chart-tip">按创建日；仅统计未取消的自营订单</p>
          <div ref="volumeChartEl" class="chart" />
        </section>
        <section>
          <h3>销售额 / 成本额 / 毛利润</h3>
          <p class="chart-tip">销售额含全部订单；毛利润仅统计成本额 &gt; 0 的订单（自营 + 全部分销）</p>
          <div ref="profitChartEl" class="chart" />
        </section>
      </div>
    </div>

    <div class="section-head">分销商</div>
    <div class="work-cards">
      <div
        v-for="card in distributorCards"
        :key="card.label"
        class="work-card static"
        :style="{ '--accent': card.color }"
      >
        <div class="work-label">
          <el-icon :style="{ color: card.color }"><component :is="card.icon" /></el-icon>
          {{ card.label }}
        </div>
        <div class="work-value">{{ card.value }}</div>
      </div>
    </div>

    <div class="section-head">分销订单</div>
    <div class="work-cards">
      <div v-for="card in poCards" :key="card.label" class="work-card static">
        <div class="work-label">{{ card.label }}</div>
        <div class="work-value">{{ card.value }}</div>
        <div class="work-tip">{{ card.sub }}</div>
      </div>
    </div>

    <section v-if="stats?.recentOrders?.length" class="recent">
      <div class="section-head">最近订单</div>
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
.dashboard {
  display: flex;
  flex-direction: column;
  gap: 14px;
  width: 100%;
}
.section-head {
  font-size: 13px;
  font-weight: 600;
  color: #64748b;
  margin-top: 2px;
}
.section-head.row-between {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  flex-wrap: wrap;
}
.range-tools {
  display: flex;
  align-items: center;
  gap: 8px;
}
.work-rows {
  display: flex;
  flex-direction: column;
  gap: 14px;
}
.work-row {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.work-row-tag {
  align-self: flex-start;
  font-size: 12px;
  font-weight: 600;
  letter-spacing: 0.04em;
  padding: 2px 8px;
  border-radius: 4px;
}
.work-row-tag.self {
  color: #b45309;
  background: #fff7e6;
}
.work-row-tag.dist {
  color: #6b21a8;
  background: #f5f0ff;
}
.work-cards {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 12px;
}
.work-card {
  text-align: left;
  border: 1px solid #e8edf3;
  background: #fff;
  border-radius: 10px;
  padding: 14px 16px;
  cursor: pointer;
  border-top: 3px solid var(--accent, #1677ff);
  transition: box-shadow 0.15s, border-color 0.15s, transform 0.15s;
}
.work-card:hover {
  box-shadow: 0 4px 14px rgba(15, 39, 68, 0.08);
  transform: translateY(-1px);
}
.work-card.static {
  cursor: default;
}
.work-card.static:hover {
  box-shadow: none;
  transform: none;
}
.work-card.highlight {
  border-color: color-mix(in srgb, var(--accent) 35%, #e8edf3);
  background: linear-gradient(180deg, color-mix(in srgb, var(--accent) 10%, #fff) 0%, #fff 65%);
}
.work-label {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  color: #64748b;
}
.work-value {
  margin-top: 6px;
  font-size: 28px;
  font-weight: 700;
  color: #0f172a;
  line-height: 1.1;
}
.work-tip {
  margin-top: 6px;
  font-size: 12px;
  color: #94a3b8;
}
.metric-row {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 12px;
}
.metric-card {
  text-align: left;
  background: #fff;
  border: 1px solid #eef0f3;
  border-radius: 10px;
  padding: 14px 16px;
}
.metric-card.highlight.profit {
  border-color: #6ee7b7;
  background: linear-gradient(180deg, #ecfdf5 0%, #fff 60%);
}
.metric-label {
  font-size: 13px;
  color: #64748b;
}
.metric-value {
  margin-top: 4px;
  font-size: 24px;
  font-weight: 700;
  color: #0f172a;
  line-height: 1.2;
}
.metric-tip {
  margin-top: 6px;
  font-size: 12px;
  color: #94a3b8;
  line-height: 1.4;
}
.trend-block {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.channel-sales-row {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 12px;
}
.channel-sales-card {
  text-align: left;
  background: #fff;
  border: 1px solid #eef0f3;
  border-radius: 10px;
  padding: 14px 16px;
  border-top: 3px solid var(--accent, #1677ff);
}
.channel-sales-value {
  margin-top: 6px;
  font-size: 22px;
  font-weight: 700;
  color: #0f172a;
  line-height: 1.15;
}
.charts {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
}
.charts section {
  background: #fff;
  border: 1px solid #eef0f3;
  border-radius: 10px;
  padding: 14px 16px 8px;
}
.charts h3 {
  margin: 0;
  font-size: 14px;
  font-weight: 600;
  color: #0f172a;
}
.chart-tip {
  margin: 4px 0 0;
  font-size: 12px;
  color: #94a3b8;
}
.chart {
  width: 100%;
  height: 280px;
}
.recent {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
@media (max-width: 1100px) {
  .work-cards,
  .metric-row,
  .channel-sales-row,
  .charts {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}
@media (max-width: 640px) {
  .work-cards,
  .metric-row,
  .channel-sales-row,
  .charts {
    grid-template-columns: 1fr;
  }
}
</style>
