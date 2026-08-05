<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { searchOrders, type OrderBrief } from '../api/order'

const model = defineModel<string | undefined>({ default: undefined })

withDefaults(
  defineProps<{
    placeholder?: string
    disabled?: boolean
    clearable?: boolean
  }>(),
  {
    placeholder: '输入订单号搜索，如 OC202607270013',
    disabled: false,
    clearable: true,
  },
)

const emit = defineEmits<{
  select: [item: OrderBrief | undefined]
}>()

const loading = ref(false)
const options = ref<OrderBrief[]>([])
let debounceTimer: ReturnType<typeof setTimeout> | undefined

const selected = computed(() =>
  model.value ? options.value.find((item) => item.orderNo === model.value) : undefined,
)

function orderLabel(o: OrderBrief) {
  const bits = [o.orderNo]
  if (o.platformOrderId) bits.push(o.platformOrderId)
  if (o.buyerName || o.buyerNick) bits.push(o.buyerName || o.buyerNick || '')
  return bits.filter(Boolean).join(' · ')
}

async function remoteSearch(keyword: string) {
  const q = keyword.trim()
  if (!q) {
    options.value = selected.value ? [selected.value] : []
    return
  }
  loading.value = true
  try {
    const data = await searchOrders({ keyword: q, page: 1, pageSize: 20 })
    const list = data.list || []
    if (selected.value && !list.some((o) => o.orderNo === selected.value!.orderNo)) {
      options.value = [selected.value, ...list]
    } else {
      options.value = list
    }
  } catch (e) {
    ElMessage.error((e as Error).message || '订单搜索失败')
  } finally {
    loading.value = false
  }
}

function onSearch(keyword: string) {
  clearTimeout(debounceTimer)
  debounceTimer = setTimeout(() => {
    void remoteSearch(keyword)
  }, 280)
}

function onChange(orderNo: string | undefined) {
  if (!orderNo) {
    emit('select', undefined)
    return
  }
  const hit = options.value.find((o) => o.orderNo === orderNo)
  emit('select', hit)
}

watch(
  () => model.value,
  (orderNo) => {
    if (!orderNo) return
    if (options.value.some((o) => o.orderNo === orderNo)) return
    options.value = [
      {
        id: 0,
        orderNo,
      },
      ...options.value,
    ]
    void remoteSearch(orderNo)
  },
  { immediate: true },
)
</script>

<template>
  <el-select
    v-model="model"
    filterable
    remote
    reserve-keyword
    :clearable="clearable"
    :disabled="disabled"
    :remote-method="onSearch"
    :loading="loading"
    :placeholder="placeholder"
    no-data-text="请输入订单号搜索"
    popper-class="order-search-select-dropdown"
    style="width: 100%"
    @change="onChange"
  >
    <el-option
      v-for="item in options"
      :key="item.orderNo"
      :label="orderLabel(item)"
      :value="item.orderNo"
    >
      <div class="order-option">
        <div class="main">{{ item.orderNo }}</div>
        <div class="sub">
          <span v-if="item.platformOrderId">平台 {{ item.platformOrderId }}</span>
          <span v-if="item.shopName">{{ item.shopName }}</span>
          <span v-if="item.buyerName || item.buyerNick">{{ item.buyerName || item.buyerNick }}</span>
          <span v-if="item.payAmount != null">¥{{ Number(item.payAmount || 0).toFixed(2) }}</span>
        </div>
      </div>
    </el-option>
  </el-select>
</template>

<style scoped>
.order-option {
  padding: 2px 0;
  line-height: 1.35;
  white-space: normal;
}
.order-option .main {
  font-weight: 500;
  word-break: break-all;
}
.order-option .sub {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 2px;
  font-size: 12px;
  color: #909399;
}
</style>

<style>
.order-search-select-dropdown.el-select-dropdown {
  min-width: 480px !important;
  max-width: min(640px, 92vw) !important;
}
.order-search-select-dropdown .el-select-dropdown__item {
  height: auto !important;
  min-height: 40px;
  padding: 8px 12px;
  line-height: 1.35;
  white-space: normal;
}
</style>
