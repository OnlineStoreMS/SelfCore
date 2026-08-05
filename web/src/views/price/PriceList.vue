<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import type { TableColumnCtx } from 'element-plus'
import { Plus, Edit, Delete, Search } from '@element-plus/icons-vue'
import SkuSearchSelect from '../../components/SkuSearchSelect.vue'
import ProductSkuPicker from '../../components/ProductSkuPicker.vue'
import {
  createSkuPrice,
  deleteSkuPrice,
  fetchSkuPrices,
  fetchDistributorAddresses,
  fetchDistributors,
  updateSkuPrice,
  type SkuPrice,
  type Distributor,
  type DistributorAddress,
} from '../../api/distributor'
import {
  resolveProductSkus,
  skuDisplayPic,
  type ProductSkuSearchItem,
} from '../../api/productSku'

interface OfferRow extends SkuPrice {
  skuInfo?: ProductSkuSearchItem
  _skuSpan?: number
}

const tableData = ref<OfferRow[]>([])
const distributors = ref<Distributor[]>([])
const addresses = ref<DistributorAddress[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const filterSkuId = ref<number | undefined>()
const filterDistributorId = ref<number | undefined>()
const loading = ref(false)
const dialogVisible = ref(false)
const saving = ref(false)
const editing = ref<Partial<SkuPrice>>({
  currency: 'CNY',
  minOrderQty: 1,
  supportsWholesale: true,
  status: 1,
})

const skuSpanMap = computed(() => {
  const map = new Map<number, { first: number; count: number }>()
  tableData.value.forEach((row, index) => {
    const existing = map.get(row.skuId)
    if (existing) {
      existing.count += 1
    } else {
      map.set(row.skuId, { first: index, count: 1 })
    }
  })
  return map
})

function spanMethod(args: {
  row: OfferRow
  column: TableColumnCtx<OfferRow>
  rowIndex: number
  columnIndex: number
}) {
  const prop = String(args.column.property || '')
  if (prop !== 'skuProduct' && prop !== 'skuSpec') {
    return { rowspan: 1, colspan: 1 }
  }
  const info = skuSpanMap.value.get(args.row.skuId)
  if (!info) return { rowspan: 1, colspan: 1 }
  if (args.rowIndex === info.first) {
    return { rowspan: info.count, colspan: 1 }
  }
  return { rowspan: 0, colspan: 0 }
}

async function loadDistributors() {
  try {
    const data = await fetchDistributors({ page: 1, pageSize: 200 })
    distributors.value = data.list
  } catch {
    distributors.value = []
  }
}

async function loadAddresses(distributorId?: number) {
  if (!distributorId) {
    addresses.value = []
    return
  }
  try {
    addresses.value = await fetchDistributorAddresses(distributorId, 'ship')
  } catch {
    addresses.value = []
  }
}

async function loadData() {
  loading.value = true
  try {
    const data = await fetchSkuPrices({
      skuId: filterSkuId.value,
      distributorId: filterDistributorId.value,
      page: page.value,
      pageSize: pageSize.value,
    })
    const skuMap = await resolveProductSkus(data.list.map((item) => item.skuId))
    tableData.value = data.list.map((item) => ({
      ...item,
      skuInfo: skuMap.get(item.skuId),
    }))
    total.value = data.total
  } catch (e) {
    ElMessage.error((e as Error).message || '加载失败')
  } finally {
    loading.value = false
  }
}

onMounted(async () => {
  await loadDistributors()
  await loadData()
})

watch(filterSkuId, () => {
  page.value = 1
  void loadData()
})

async function handleAdd(skuId?: number) {
  editing.value = {
    skuId: skuId || filterSkuId.value,
    distributorId: filterDistributorId.value,
    wholesalePrice: 0,
    currency: 'CNY',
    minOrderQty: 1,
    supportsDropship: false,
    supportsWholesale: true,
    isPrimary: false,
    priority: 0,
    status: 1,
    shipFromAddressId: undefined,
  }
  await loadAddresses(editing.value.distributorId)
  dialogVisible.value = true
}

async function handleEdit(row: SkuPrice) {
  editing.value = {
    ...row,
    shipFromAddressId: row.shipFromAddressId || undefined,
  }
  await loadAddresses(row.distributorId)
  dialogVisible.value = true
}

async function onDistributorChange(distributorId: number) {
  editing.value.shipFromAddressId = undefined
  await loadAddresses(distributorId)
}

async function handleSave() {
  if (!editing.value.skuId || !editing.value.distributorId) {
    ElMessage.warning('请选择商家编码对应的商品与分销商')
    return
  }
  if (editing.value.wholesalePrice == null || editing.value.wholesalePrice < 0) {
    ElMessage.warning('请填写批发价')
    return
  }
  saving.value = true
  try {
    if (editing.value.id) {
      await updateSkuPrice(editing.value.id, editing.value)
    } else {
      await createSkuPrice(editing.value)
    }
    ElMessage.success('已保存')
    dialogVisible.value = false
    await loadData()
  } catch (e) {
    ElMessage.error((e as Error).message || '保存失败')
  } finally {
    saving.value = false
  }
}

async function handleDelete(row: SkuPrice) {
  try {
    await deleteSkuPrice(row.id)
    ElMessage.success('已删除')
    await loadData()
  } catch (e) {
    ElMessage.error((e as Error).message || '删除失败')
  }
}

function onPageChange(p: number) {
  page.value = p
  void loadData()
}

function addressLabel(addr: DistributorAddress) {
  const region = [addr.province, addr.city, addr.district].filter(Boolean).join('')
  const bits = [addr.label, region, addr.address].filter(Boolean)
  return bits.join(' · ') || `地址 #${addr.id}`
}
</script>

<template>
  <div class="offer-page">
    <el-card v-loading="loading">
      <template #header>
        <span>SKU 批发价</span>
        <el-button type="primary" :icon="Plus" @click="handleAdd()">添加报价</el-button>
      </template>

      <div class="toolbar">
        <div class="toolbar-sku">
          <SkuSearchSelect
            v-model="filterSkuId"
            placeholder="按商家编码 / 规格 / 商品名筛选"
            :show-preview="false"
            clearable
          />
        </div>
        <el-select
          v-model="filterDistributorId"
          placeholder="分销商"
          clearable
          filterable
          style="width: 200px"
          @change="() => { page = 1; loadData() }"
        >
          <el-option v-for="s in distributors" :key="s.id" :label="s.name" :value="s.id" />
        </el-select>
        <el-button :icon="Search" @click="loadData">查询</el-button>
      </div>

      <el-table :data="tableData" stripe :span-method="spanMethod" class="offer-table">
        <el-table-column prop="skuProduct" label="商品" min-width="240">
          <template #default="{ row }">
            <div class="sku-cell">
              <el-image
                :src="skuDisplayPic(row.skuInfo)"
                fit="cover"
                class="sku-pic"
              >
                <template #error>
                  <div class="sku-pic placeholder">无图</div>
                </template>
              </el-image>
              <div class="sku-text">
                <div class="sku-name">{{ row.skuInfo?.productName || '商品' }}</div>
                <div class="sku-code">商家编码 {{ row.skuInfo?.skuCode || '—' }}</div>
                <el-button type="primary" link size="small" @click="handleAdd(row.skuId)">
                  + 加报价
                </el-button>
              </div>
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="skuSpec" label="规格" min-width="140">
          <template #default="{ row }">
            <span class="spec-text">{{ row.skuInfo?.specLabel || '—' }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="distributorName" label="分销商" min-width="130" />
        <el-table-column prop="distributorSkuCode" label="对方货号" width="120" />
        <el-table-column label="批发价" width="100" align="right">
          <template #default="{ row }">¥{{ Number(row.wholesalePrice || 0).toFixed(2) }}</template>
        </el-table-column>
        <el-table-column label="发货地" width="110">
          <template #default="{ row }">
            {{ row.shipFromLabel || row.shipFromCity || '—' }}
          </template>
        </el-table-column>
        <el-table-column label="代发" width="70" align="center">
          <template #default="{ row }">
            <el-tag :type="row.supportsDropship ? 'success' : 'info'" size="small">
              {{ row.supportsDropship ? '是' : '否' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="leadTimeDays" label="交期" width="70" align="center" />
        <el-table-column label="主供" width="70" align="center">
          <template #default="{ row }">
            <el-tag v-if="row.isPrimary" type="warning" size="small">主</el-tag>
            <span v-else class="muted">—</span>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="140" fixed="right">
          <template #default="{ row }">
            <el-button type="primary" link :icon="Edit" @click="handleEdit(row)">编辑</el-button>
            <el-popconfirm title="确定删除？" @confirm="handleDelete(row)">
              <template #reference>
                <el-button type="danger" link :icon="Delete">删除</el-button>
              </template>
            </el-popconfirm>
          </template>
        </el-table-column>
      </el-table>

      <div class="pager">
        <el-pagination
          v-model:current-page="page"
          :page-size="pageSize"
          :total="total"
          layout="total, prev, pager, next"
          @current-change="onPageChange"
        />
      </div>
    </el-card>

    <el-dialog
      v-model="dialogVisible"
      :title="editing.id ? '编辑报价' : '添加报价'"
      width="680px"
      destroy-on-close
    >
      <el-form :model="editing" label-width="100px">
        <el-form-item label="商家编码" required>
          <ProductSkuPicker v-if="!editing.id" v-model="editing.skuId" />
          <template v-else>
            <SkuSearchSelect v-model="editing.skuId" disabled />
            <div class="hint">编辑时不可更换商品 SKU</div>
          </template>
        </el-form-item>
        <el-form-item label="分销商" required>
          <el-select
            v-model="editing.distributorId"
            filterable
            style="width: 100%"
            @change="onDistributorChange"
          >
            <el-option
              v-for="s in distributors"
              :key="s.id"
              :label="`${s.name} (${s.code})`"
              :value="s.id"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="对方货号">
          <el-input v-model="editing.distributorSkuCode" placeholder="分销商侧货号（可选）" />
        </el-form-item>
        <el-form-item label="批发价" required>
          <el-input-number
            v-model="editing.wholesalePrice"
            :min="0"
            :precision="2"
            controls-position="right"
            style="width: 100%"
          />
        </el-form-item>
        <el-form-item label="发货地址">
          <el-select
            v-model="editing.shipFromAddressId"
            clearable
            filterable
            placeholder="选择分销商发货地址"
            style="width: 100%"
            :disabled="!editing.distributorId"
          >
            <el-option
              v-for="addr in addresses"
              :key="addr.id"
              :label="addressLabel(addr)"
              :value="addr.id"
            />
          </el-select>
          <div class="hint">先选分销商；地址在分销商详情中维护</div>
        </el-form-item>
        <el-row :gutter="12">
          <el-col :span="12">
            <el-form-item label="起订量">
              <el-input-number
                v-model="editing.minOrderQty"
                :min="1"
                controls-position="right"
                style="width: 100%"
              />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="交期(天)">
              <el-input-number
                v-model="editing.leadTimeDays"
                :min="0"
                controls-position="right"
                style="width: 100%"
              />
            </el-form-item>
          </el-col>
        </el-row>
        <el-row :gutter="12">
          <el-col :span="8">
            <el-form-item label="支持代发">
              <el-switch v-model="editing.supportsDropship" />
            </el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item label="供货到仓">
              <el-switch v-model="editing.supportsWholesale" />
            </el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item label="主分销商">
              <el-switch v-model="editing.isPrimary" />
            </el-form-item>
          </el-col>
        </el-row>
        <el-form-item label="优先级">
          <el-input-number v-model="editing.priority" controls-position="right" style="width: 100%" />
        </el-form-item>
        <el-form-item label="备注">
          <el-input v-model="editing.remark" type="textarea" :rows="2" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="handleSave">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.offer-page :deep(.el-card__header) {
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.toolbar {
  display: flex;
  gap: 8px;
  margin-bottom: 12px;
  align-items: flex-start;
  flex-wrap: wrap;
}
.toolbar-sku {
  width: 360px;
  max-width: 100%;
}
.offer-table :deep(.el-table__cell) {
  vertical-align: top;
}
.sku-cell {
  display: flex;
  gap: 10px;
  align-items: flex-start;
}
.sku-pic {
  width: 56px;
  height: 56px;
  border-radius: 6px;
  flex-shrink: 0;
  background: #f5f7fa;
}
.sku-pic.placeholder {
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 11px;
  color: #c0c4cc;
}
.sku-text {
  min-width: 0;
}
.sku-name {
  font-weight: 500;
  font-size: 13px;
  line-height: 1.4;
  margin-bottom: 2px;
}
.sku-code {
  font-size: 12px;
  color: #909399;
  margin-bottom: 2px;
}
.spec-text {
  color: #409eff;
  font-size: 13px;
  line-height: 1.4;
}
.muted {
  color: #c0c4cc;
}
.pager {
  margin-top: 16px;
  display: flex;
  justify-content: flex-end;
}
.hint {
  font-size: 12px;
  color: #909399;
  margin-top: 4px;
}
</style>
