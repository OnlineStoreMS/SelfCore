<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { Plus, Edit, Delete, Search, Setting, WarningFilled } from '@element-plus/icons-vue'
import {
  createDistributor,
  createDistributorCategory,
  deleteDistributor,
  deleteDistributorCategory,
  fetchDistributorCategories,
  fetchDistributors,
  SETTLEMENT_CYCLE_MAP,
  distributorMobile,
  updateDistributor,
  updateDistributorCategory,
  type Distributor,
  type DistributorCategory,
} from '../../api/distributor'

const router = useRouter()
const tableData = ref<Distributor[]>([])
const categories = ref<DistributorCategory[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const keyword = ref('')
const selectedCategoryId = ref(0)
const loading = ref(false)
const dialogVisible = ref(false)
const categoryDialogVisible = ref(false)
const categoryManageVisible = ref(false)
const editing = ref<Partial<Distributor>>({})
const editingCategory = ref<Partial<DistributorCategory>>({})

const categoryOptions = computed(() =>
  categories.value.filter((c) => c.status === 1),
)

function defaultDistributor(): Partial<Distributor> {
  return {
    code: '',
    name: '',
    status: 1,
    cutOffTime: '00:01',
    settlementCycle: '',
    settlementCustomDays: 7,
    settlementMergeTime: '18:30',
    autoCreateDropshipPO: false,
    syncPurchasePriceFrom: '',
    categoryId: selectedCategoryId.value || undefined,
  }
}

async function loadCategories() {
  try {
    categories.value = await fetchDistributorCategories()
  } catch (e) {
    ElMessage.error((e as Error).message || '加载类别失败')
  }
}

async function loadData() {
  loading.value = true
  try {
    const data = await fetchDistributors({
      keyword: keyword.value || undefined,
      categoryId: selectedCategoryId.value || undefined,
      page: page.value,
      pageSize: pageSize.value,
    })
    tableData.value = data.list
    total.value = data.total
  } catch (e) {
    ElMessage.error((e as Error).message || '加载失败')
  } finally {
    loading.value = false
  }
}

onMounted(async () => {
  await loadCategories()
  await loadData()
})

function selectCategory(id: number) {
  selectedCategoryId.value = id
  page.value = 1
  loadData()
}

function handleAdd() {
  editing.value = defaultDistributor()
  dialogVisible.value = true
}

function handleEdit(row: Distributor) {
  editing.value = {
    ...row,
    mobile: row.mobile || row.phone,
    cutOffTime: row.cutOffTime || '00:01',
    settlementCycle: row.settlementCycle || '',
    settlementCustomDays: row.settlementCustomDays || 7,
    settlementMergeTime: row.settlementMergeTime || '18:30',
    autoCreateDropshipPO: !!row.autoCreateDropshipPO,
    syncPurchasePriceFrom: row.syncPurchasePriceFrom || '',
  }
  dialogVisible.value = true
}

function openDetail(row: Distributor) {
  router.push(`/distributors/${row.id}`)
}

async function handleSave() {
  try {
    const payload = { ...editing.value }
    if (payload.mobile && !payload.phone) {
      payload.phone = payload.mobile
    }
    if (payload.id) {
      await updateDistributor(payload.id, payload)
    } else {
      await createDistributor(payload)
    }
    ElMessage.success('已保存')
    dialogVisible.value = false
    await loadData()
  } catch (e) {
    ElMessage.error((e as Error).message || '保存失败')
  }
}

async function handleDelete(row: Distributor) {
  try {
    await deleteDistributor(row.id)
    ElMessage.success('已删除')
    await loadData()
  } catch (e) {
    ElMessage.error((e as Error).message || '删除失败')
  }
}

function onPageChange(p: number) {
  page.value = p
  loadData()
}

function handleAddCategory() {
  editingCategory.value = { name: '', status: 1, sort: 0, parentId: 0 }
  categoryDialogVisible.value = true
}

function handleEditCategory(row: DistributorCategory) {
  editingCategory.value = { ...row }
  categoryDialogVisible.value = true
}

async function handleSaveCategory() {
  try {
    if (editingCategory.value.id) {
      await updateDistributorCategory(editingCategory.value.id, editingCategory.value)
    } else {
      await createDistributorCategory(editingCategory.value)
    }
    ElMessage.success('已保存')
    categoryDialogVisible.value = false
    await loadCategories()
  } catch (e) {
    ElMessage.error((e as Error).message || '保存失败')
  }
}

async function handleDeleteCategory(row: DistributorCategory) {
  try {
    await deleteDistributorCategory(row.id)
    ElMessage.success('已删除')
    if (selectedCategoryId.value === row.id) {
      selectedCategoryId.value = 0
      await loadData()
    }
    await loadCategories()
  } catch (e) {
    ElMessage.error((e as Error).message || '删除失败')
  }
}
</script>

<template>
  <div class="distributor-page">
    <div class="layout">
      <el-card class="category-panel" shadow="never">
        <template #header>
          <span>分销商类别</span>
          <el-button type="primary" link :icon="Setting" @click="categoryManageVisible = true">
            设置
          </el-button>
        </template>
        <ul class="category-list">
          <li
            :class="{ active: selectedCategoryId === 0 }"
            @click="selectCategory(0)"
          >
            全部类别
          </li>
          <li
            v-for="cat in categories"
            :key="cat.id"
            :class="{ active: selectedCategoryId === cat.id }"
            @click="selectCategory(cat.id)"
          >
            {{ cat.name }}
          </li>
        </ul>
      </el-card>

      <el-card v-loading="loading" class="main-panel">
        <template #header>
          <span>分销商</span>
          <el-button type="primary" :icon="Plus" @click="handleAdd">新增分销商</el-button>
        </template>

        <div class="toolbar">
          <el-input
            v-model="keyword"
            placeholder="搜索名称/编码"
            :prefix-icon="Search"
            clearable
            style="width: 260px"
            @change="() => { page = 1; loadData() }"
          />
        </div>

        <el-table :data="tableData" stripe border table-layout="auto">
          <el-table-column prop="name" label="分销商名称" min-width="140">
            <template #default="{ row }">
              <el-link type="primary" @click="openDetail(row)">{{ row.name }}</el-link>
            </template>
          </el-table-column>
          <el-table-column prop="code" label="编码" width="100" />
          <el-table-column prop="categoryName" label="类别" width="90">
            <template #default="{ row }">{{ row.categoryName || '—' }}</template>
          </el-table-column>
          <el-table-column prop="wangwangId" label="旺旺ID" width="110" show-overflow-tooltip>
            <template #default="{ row }">{{ row.wangwangId || '—' }}</template>
          </el-table-column>
          <el-table-column prop="contactName" label="联系人" width="90">
            <template #default="{ row }">{{ row.contactName || '—' }}</template>
          </el-table-column>
          <el-table-column label="手机" width="120">
            <template #default="{ row }">{{ distributorMobile(row) || '—' }}</template>
          </el-table-column>
          <el-table-column prop="address" label="地址" min-width="140" show-overflow-tooltip>
            <template #default="{ row }">{{ row.address || '—' }}</template>
          </el-table-column>
          <el-table-column prop="buyerName" label="采购员" width="90">
            <template #default="{ row }">{{ row.buyerName || '—' }}</template>
          </el-table-column>
          <el-table-column prop="cutOffTime" label="截单时间" width="90" align="center">
            <template #default="{ row }">{{ row.cutOffTime || '—' }}</template>
          </el-table-column>
          <el-table-column label="结算周期" width="120" align="center">
            <template #default="{ row }">
              <template v-if="row.settlementCycle">
                {{ SETTLEMENT_CYCLE_MAP[row.settlementCycle] || row.settlementCycle }}
                <span v-if="row.settlementCycle === 'custom'">{{ row.settlementCustomDays }}天</span>
                <div class="muted">{{ row.settlementMergeTime || '18:30' }}</div>
              </template>
              <span v-else>—</span>
            </template>
          </el-table-column>
          <el-table-column prop="remark" label="备注" min-width="120" show-overflow-tooltip>
            <template #default="{ row }">{{ row.remark || '—' }}</template>
          </el-table-column>
          <el-table-column label="状态" width="80" align="center">
            <template #default="{ row }">
              <el-tag :type="row.status === 1 ? 'success' : 'info'" size="small">
                {{ row.status === 1 ? '启用' : '停用' }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column label="操作" width="140" fixed="right">
            <template #default="{ row }">
              <el-button type="primary" link @click="openDetail(row)">详情</el-button>
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
    </div>

    <el-dialog
      v-model="dialogVisible"
      :title="editing.id ? '编辑分销商' : '增加分销商'"
      width="720px"
    >
      <el-form :model="editing" label-width="130px">
        <el-row :gutter="16">
          <el-col :span="12">
            <el-form-item label="类别">
              <el-select v-model="editing.categoryId" clearable placeholder="选择类别" style="width: 100%">
                <el-option
                  v-for="cat in categoryOptions"
                  :key="cat.id"
                  :label="cat.name"
                  :value="cat.id"
                />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :span="12" />
          <el-col :span="12">
            <el-form-item label="名称" required>
              <el-input v-model="editing.name" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="编码" required>
              <el-input v-model="editing.code" :disabled="!!editing.id" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="采购员">
              <el-input v-model="editing.buyerName" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="截单时间">
              <el-time-picker
                v-model="editing.cutOffTime"
                format="HH:mm"
                value-format="HH:mm"
                placeholder="选择时间"
                style="width: 100%"
              />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="到货天数">
              <el-input-number v-model="editing.arrivalDays" :min="0" controls-position="right" style="width: 100%" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="账期天数">
              <el-input-number v-model="editing.paymentDays" :min="0" controls-position="right" style="width: 100%" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="结算周期">
              <el-select v-model="editing.settlementCycle" clearable placeholder="不启用" style="width: 100%">
                <el-option label="不启用" value="" />
                <el-option label="按天" value="day" />
                <el-option label="按周" value="week" />
                <el-option label="按月" value="month" />
                <el-option label="自定义天数" value="custom" />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col v-if="editing.settlementCycle === 'custom'" :span="12">
            <el-form-item label="自定义天数">
              <el-input-number v-model="editing.settlementCustomDays" :min="1" :max="365" controls-position="right" style="width: 100%" />
            </el-form-item>
          </el-col>
          <el-col v-if="editing.settlementCycle" :span="12">
            <el-form-item label="合并时刻">
              <el-time-picker
                v-model="editing.settlementMergeTime"
                format="HH:mm"
                value-format="HH:mm"
                placeholder="选择时间"
                style="width: 100%"
              />
            </el-form-item>
          </el-col>
          <el-col :span="24">
            <el-form-item label="自动建代发单">
              <div class="auto-po-field">
                <el-switch v-model="editing.autoCreateDropshipPO" />
                <el-tooltip placement="top" :show-after="200" effect="dark" popper-class="auto-po-tip">
                  <template #content>
                    <div class="tip-body">
                      开启后：仅在订单同步时，自动分配到该分销商的订单会创建代发分销订单。
                      同一批同步中多个订单会合并为一张；不会补建历史已分配但缺单的订单。
                      手工在订单中心改分配为代发时始终建单，不受此开关影响。
                      若同时设置了结算周期，日常逐单产生的代发单会在合并时刻按 T+1 归档合并（按天=合并昨天的单；按周/月=合并上一完整周/月）。
                    </div>
                  </template>
                  <el-icon class="tip-icon"><WarningFilled /></el-icon>
                </el-tooltip>
              </div>
            </el-form-item>
          </el-col>
          <el-col :span="24">
            <el-form-item label="同步批发价">
              <div class="auto-po-field">
                <el-select
                  v-model="editing.syncPurchasePriceFrom"
                  clearable
                  placeholder="不启用"
                  style="width: 220px"
                >
                  <el-option label="不启用" value="" />
                  <el-option label="分发备注" value="fen_fa_remark" />
                  <el-option label="分配备注" value="alloc_remark" />
                  <el-option label="卖家备注" value="seller_remark" />
                  <el-option label="打单备注" value="printer_remark" />
                </el-select>
                <el-tooltip placement="top" :show-after="200" effect="dark" popper-class="auto-po-tip">
                  <template #content>
                    <div class="tip-body">
                      选择备注字段后：合并时刻（及手工合并代发单后）会从该字段解析金额（如「70」），
                      同步为对应销售单的采购小计，并按数量反推分销订单价。建代发单时也会预填。
                      需配合结算周期与合并时刻使用。
                    </div>
                  </template>
                  <el-icon class="tip-icon"><WarningFilled /></el-icon>
                </el-tooltip>
              </div>
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="联系人">
              <el-input v-model="editing.contactName" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="地址">
              <el-input v-model="editing.address" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="办公电话">
              <el-input v-model="editing.officePhone" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="手机">
              <el-input v-model="editing.mobile" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="旺旺ID">
              <el-input v-model="editing.wangwangId" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="QQ">
              <el-input v-model="editing.qq" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="邮箱">
              <el-input v-model="editing.email" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="网址">
              <el-input v-model="editing.website" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="账号">
              <el-input v-model="editing.bankAccount" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="状态">
              <el-switch v-model="editing.status" :active-value="1" :inactive-value="0" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="开户行">
              <el-input v-model="editing.bankName" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="户名">
              <el-input v-model="editing.accountName" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="简称">
              <el-input v-model="editing.shortName" />
            </el-form-item>
          </el-col>
          <el-col :span="24">
            <el-form-item label="账期说明">
              <el-input v-model="editing.defaultPaymentTerms" placeholder="可选文字说明" />
            </el-form-item>
          </el-col>
          <el-col :span="24">
            <el-form-item label="备注">
              <el-input v-model="editing.remark" type="textarea" :rows="2" />
            </el-form-item>
          </el-col>
        </el-row>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" @click="handleSave">保存</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="categoryManageVisible" title="设置分销商类别" width="480px">
      <div class="category-toolbar">
        <el-button type="primary" :icon="Plus" size="small" @click="handleAddCategory">新增类别</el-button>
      </div>
      <el-table :data="categories" stripe border size="small">
        <el-table-column prop="name" label="名称" />
        <el-table-column prop="sort" label="排序" width="70" align="center" />
        <el-table-column label="状态" width="70" align="center">
          <template #default="{ row }">
            <el-tag :type="row.status === 1 ? 'success' : 'info'" size="small">
              {{ row.status === 1 ? '启用' : '停用' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="120">
          <template #default="{ row }">
            <el-button type="primary" link size="small" @click="handleEditCategory(row)">编辑</el-button>
            <el-popconfirm title="确定删除？有分销商的类别不可删" @confirm="handleDeleteCategory(row)">
              <template #reference>
                <el-button type="danger" link size="small">删除</el-button>
              </template>
            </el-popconfirm>
          </template>
        </el-table-column>
      </el-table>
    </el-dialog>

    <el-dialog
      v-model="categoryDialogVisible"
      :title="editingCategory.id ? '编辑类别' : '新增类别'"
      width="400px"
    >
      <el-form :model="editingCategory" label-width="70px">
        <el-form-item label="名称" required>
          <el-input v-model="editingCategory.name" />
        </el-form-item>
        <el-form-item label="排序">
          <el-input-number v-model="editingCategory.sort" :min="0" controls-position="right" style="width: 100%" />
        </el-form-item>
        <el-form-item label="状态">
          <el-switch v-model="editingCategory.status" :active-value="1" :inactive-value="0" />
        </el-form-item>
        <el-form-item label="备注">
          <el-input v-model="editingCategory.remark" type="textarea" :rows="2" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="categoryDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="handleSaveCategory">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.layout {
  display: flex;
  gap: 12px;
  align-items: stretch;
  min-height: calc(100vh - 140px);
}
.category-panel {
  width: 180px;
  flex-shrink: 0;
}
.category-panel :deep(.el-card__body) {
  padding: 8px;
}
.category-panel :deep(.el-card__header) {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 16px;
}
.category-list {
  list-style: none;
  margin: 0;
  padding: 0;
}
.category-list li {
  padding: 8px 12px;
  cursor: pointer;
  border-radius: 4px;
  font-size: 14px;
  color: var(--el-text-color-regular);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.category-list li:hover {
  background: var(--el-fill-color-light);
}
.category-list li.active {
  background: var(--el-color-primary-light-9);
  color: var(--el-color-primary);
  font-weight: 500;
}
.main-panel {
  flex: 1;
  min-width: 0;
  overflow: hidden;
}
.main-panel :deep(.el-card__header) {
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.toolbar {
  margin-bottom: 12px;
}
.pager {
  margin-top: 16px;
  display: flex;
  justify-content: flex-end;
}
.category-toolbar {
  margin-bottom: 12px;
}
.muted {
  color: #909399;
  font-size: 12px;
  line-height: 1.2;
}
.auto-po-field {
  display: inline-flex;
  align-items: center;
  gap: 8px;
}

.tip-icon {
  color: #c0c4cc;
  cursor: pointer;
  font-size: 14px;
  outline: none;
}

.tip-icon:hover {
  color: #909399;
}
</style>

<style>
.auto-po-tip {
  max-width: 360px;
}

.auto-po-tip .tip-body {
  line-height: 1.6;
  font-size: 12px;
}
</style>
