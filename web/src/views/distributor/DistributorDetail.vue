<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { Plus, Edit, Delete, ArrowLeft, Upload } from '@element-plus/icons-vue'
import {
  fetchDistributor,
  fetchDistributorAddresses,
  createDistributorAddress,
  updateDistributorAddress,
  deleteDistributorAddress,
  fetchDistributorPaymentAccounts,
  createDistributorPaymentAccount,
  updateDistributorPaymentAccount,
  deleteDistributorPaymentAccount,
  fetchDistributorPaymentQRs,
  createDistributorPaymentQR,
  updateDistributorPaymentQR,
  deleteDistributorPaymentQR,
  distributorMobile,
  ACCOUNT_TYPE_MAP,
  PAY_TYPE_MAP,
  SETTLEMENT_CYCLE_MAP,
  type Distributor,
  type DistributorAddress,
  type DistributorPaymentAccount,
  type DistributorPaymentQR,
} from '../../api/distributor'
import { uploadFile } from '../../api/tracking'

const SYNC_PURCHASE_PRICE_MAP: Record<string, string> = {
  fen_fa_remark: '分发备注',
  alloc_remark: '分配备注',
  seller_remark: '卖家备注',
  printer_remark: '打单备注',
}

function syncPurchasePriceLabel(v?: string) {
  const key = (v || '').trim()
  if (!key) return '未开启'
  return SYNC_PURCHASE_PRICE_MAP[key] || key
}

const route = useRoute()
const router = useRouter()
const distributorId = computed(() => Number(route.params.id))

const distributor = ref<Distributor | null>(null)
const shipAddresses = ref<DistributorAddress[]>([])
const returnAddresses = ref<DistributorAddress[]>([])
const paymentAccounts = ref<DistributorPaymentAccount[]>([])
const paymentQRs = ref<DistributorPaymentQR[]>([])
const loading = ref(false)

const addrDialogVisible = ref(false)
const editingAddr = ref<Partial<DistributorAddress>>({})
const editingAddrType = ref<'ship' | 'return'>('ship')

const accountDialogVisible = ref(false)
const editingAccount = ref<Partial<DistributorPaymentAccount>>({})

const qrDialogVisible = ref(false)
const editingQR = ref<Partial<DistributorPaymentQR>>({})
const qrUploading = ref(false)
const qrPreviewVisible = ref(false)
const qrPreviewUrl = ref('')

const addrDialogTitle = computed(() => {
  const isReturn = editingAddrType.value === 'return'
  if (editingAddr.value.id) {
    return isReturn ? '编辑退货地址' : '编辑发货地址'
  }
  return isReturn ? '添加退货地址' : '添加发货地址'
})

function openQRPreview(url: string) {
  if (!url) return
  qrPreviewUrl.value = url
  qrPreviewVisible.value = true
}

async function reloadAddresses() {
  const [ship, ret] = await Promise.all([
    fetchDistributorAddresses(distributorId.value, 'ship'),
    fetchDistributorAddresses(distributorId.value, 'return'),
  ])
  shipAddresses.value = ship
  returnAddresses.value = ret
}

async function loadData() {
  loading.value = true
  try {
    const [s, ship, ret, accounts, qrs] = await Promise.all([
      fetchDistributor(distributorId.value),
      fetchDistributorAddresses(distributorId.value, 'ship'),
      fetchDistributorAddresses(distributorId.value, 'return'),
      fetchDistributorPaymentAccounts(distributorId.value),
      fetchDistributorPaymentQRs(distributorId.value),
    ])
    distributor.value = s
    shipAddresses.value = ship
    returnAddresses.value = ret
    paymentAccounts.value = accounts
    paymentQRs.value = qrs
  } catch (e) {
    ElMessage.error((e as Error).message || '加载失败')
  } finally {
    loading.value = false
  }
}

onMounted(loadData)

function handleAddAddress(type: 'ship' | 'return') {
  editingAddrType.value = type
  editingAddr.value = { label: '', addressType: type, status: 1, isDefault: false }
  addrDialogVisible.value = true
}

function handleEditAddress(row: DistributorAddress) {
  editingAddrType.value = row.addressType === 'return' ? 'return' : 'ship'
  editingAddr.value = { ...row, addressType: editingAddrType.value }
  addrDialogVisible.value = true
}

async function handleSaveAddress() {
  try {
    const payload = { ...editingAddr.value, addressType: editingAddrType.value }
    if (payload.id) {
      await updateDistributorAddress(distributorId.value, payload.id, payload)
    } else {
      await createDistributorAddress(distributorId.value, payload)
    }
    ElMessage.success('已保存')
    addrDialogVisible.value = false
    await reloadAddresses()
  } catch (e) {
    ElMessage.error((e as Error).message || '保存失败')
  }
}

async function handleDeleteAddress(row: DistributorAddress) {
  try {
    await deleteDistributorAddress(distributorId.value, row.id)
    ElMessage.success('已删除')
    await reloadAddresses()
  } catch (e) {
    ElMessage.error((e as Error).message || '删除失败')
  }
}

function handleAddAccount() {
  editingAccount.value = { label: '', accountType: 'bank', status: 1, isDefault: false }
  accountDialogVisible.value = true
}

function handleEditAccount(row: DistributorPaymentAccount) {
  editingAccount.value = { ...row }
  accountDialogVisible.value = true
}

async function handleSaveAccount() {
  try {
    if (editingAccount.value.id) {
      await updateDistributorPaymentAccount(distributorId.value, editingAccount.value.id, editingAccount.value)
    } else {
      await createDistributorPaymentAccount(distributorId.value, editingAccount.value)
    }
    ElMessage.success('已保存')
    accountDialogVisible.value = false
    paymentAccounts.value = await fetchDistributorPaymentAccounts(distributorId.value)
  } catch (e) {
    ElMessage.error((e as Error).message || '保存失败')
  }
}

async function handleDeleteAccount(row: DistributorPaymentAccount) {
  try {
    await deleteDistributorPaymentAccount(distributorId.value, row.id)
    ElMessage.success('已删除')
    paymentAccounts.value = await fetchDistributorPaymentAccounts(distributorId.value)
  } catch (e) {
    ElMessage.error((e as Error).message || '删除失败')
  }
}

function handleAddQR() {
  editingQR.value = { label: '', payType: 'wechat', imageUrl: '', status: 1, isDefault: false }
  qrDialogVisible.value = true
}

function handleEditQR(row: DistributorPaymentQR) {
  editingQR.value = { ...row }
  qrDialogVisible.value = true
}

async function onQRFileChange(uploadFileItem: { raw?: File }) {
  const file = uploadFileItem.raw
  if (!file) return
  qrUploading.value = true
  try {
    const result = await uploadFile(file)
    editingQR.value.imageUrl = result.url
    ElMessage.success('图片已上传')
  } catch (e) {
    ElMessage.error((e as Error).message || '上传失败')
  } finally {
    qrUploading.value = false
  }
}

async function handleSaveQR() {
  if (!editingQR.value.imageUrl) {
    ElMessage.warning('请先上传收款码图片')
    return
  }
  try {
    if (editingQR.value.id) {
      await updateDistributorPaymentQR(distributorId.value, editingQR.value.id, editingQR.value)
    } else {
      await createDistributorPaymentQR(distributorId.value, editingQR.value)
    }
    ElMessage.success('已保存')
    qrDialogVisible.value = false
    paymentQRs.value = await fetchDistributorPaymentQRs(distributorId.value)
  } catch (e) {
    ElMessage.error((e as Error).message || '保存失败')
  }
}

async function handleDeleteQR(row: DistributorPaymentQR) {
  try {
    await deleteDistributorPaymentQR(distributorId.value, row.id)
    ElMessage.success('已删除')
    paymentQRs.value = await fetchDistributorPaymentQRs(distributorId.value)
  } catch (e) {
    ElMessage.error((e as Error).message || '删除失败')
  }
}
</script>

<template>
  <div v-loading="loading" class="detail-page">
    <el-card v-if="distributor" class="info-card">
      <template #header>
        <span>{{ distributor.name }}（{{ distributor.code }}）</span>
        <el-button :icon="ArrowLeft" text @click="router.push('/distributors')">返回列表</el-button>
      </template>
      <el-descriptions :column="3" border>
        <el-descriptions-item label="编码">{{ distributor.code }}</el-descriptions-item>
        <el-descriptions-item label="名称">{{ distributor.name }}</el-descriptions-item>
        <el-descriptions-item label="简称">{{ distributor.shortName || '—' }}</el-descriptions-item>
        <el-descriptions-item label="类别">{{ distributor.categoryName || '—' }}</el-descriptions-item>
        <el-descriptions-item label="状态">
          <el-tag :type="distributor.status === 1 ? 'success' : 'info'" size="small">
            {{ distributor.status === 1 ? '启用' : '停用' }}
          </el-tag>
        </el-descriptions-item>
        <el-descriptions-item label="采购员">{{ distributor.buyerName || '—' }}</el-descriptions-item>
        <el-descriptions-item label="截单时间">{{ distributor.cutOffTime || '—' }}</el-descriptions-item>
        <el-descriptions-item label="结算周期">
          <template v-if="distributor.settlementCycle">
            {{ SETTLEMENT_CYCLE_MAP[distributor.settlementCycle] || distributor.settlementCycle }}
            <span v-if="distributor.settlementCycle === 'custom'">（{{ distributor.settlementCustomDays }} 天）</span>
            ，合并时刻 {{ distributor.settlementMergeTime || '18:30' }}（T+1）
          </template>
          <template v-else>—</template>
        </el-descriptions-item>
        <el-descriptions-item label="自动建代发单">
          <el-tag :type="distributor.autoCreateDropshipPO ? 'success' : 'info'" size="small">
            {{ distributor.autoCreateDropshipPO ? '已开启' : '未开启' }}
          </el-tag>
        </el-descriptions-item>
        <el-descriptions-item label="同步批发价">
          {{ syncPurchasePriceLabel(distributor.syncPurchasePriceFrom) }}
        </el-descriptions-item>
        <el-descriptions-item label="到货天数">{{ distributor.arrivalDays ? `${distributor.arrivalDays} 天` : '—' }}</el-descriptions-item>
        <el-descriptions-item label="账期天数">{{ distributor.paymentDays ? `${distributor.paymentDays} 天` : '—' }}</el-descriptions-item>
        <el-descriptions-item label="联系人">{{ distributor.contactName || '—' }}</el-descriptions-item>
        <el-descriptions-item label="手机">{{ distributorMobile(distributor) || '—' }}</el-descriptions-item>
        <el-descriptions-item label="办公电话">{{ distributor.officePhone || '—' }}</el-descriptions-item>
        <el-descriptions-item label="旺旺ID">{{ distributor.wangwangId || '—' }}</el-descriptions-item>
        <el-descriptions-item label="QQ">{{ distributor.qq || '—' }}</el-descriptions-item>
        <el-descriptions-item label="邮箱">{{ distributor.email || '—' }}</el-descriptions-item>
        <el-descriptions-item label="网址" :span="2">{{ distributor.website || '—' }}</el-descriptions-item>
        <el-descriptions-item label="地址" :span="3">{{ distributor.address || '—' }}</el-descriptions-item>
        <el-descriptions-item label="账期说明" :span="3">{{ distributor.defaultPaymentTerms || '—' }}</el-descriptions-item>
        <el-descriptions-item label="备注" :span="3">{{ distributor.remark || '—' }}</el-descriptions-item>
      </el-descriptions>
    </el-card>

    <el-card class="section-card">
      <template #header>
        <span>发货地址</span>
        <el-button type="primary" :icon="Plus" @click="handleAddAddress('ship')">添加地址</el-button>
      </template>
      <el-table :data="shipAddresses" stripe>
        <el-table-column prop="label" label="标签" width="120" />
        <el-table-column label="地区" min-width="180">
          <template #default="{ row }">
            {{ [row.province, row.city, row.district].filter(Boolean).join(' ') }}
          </template>
        </el-table-column>
        <el-table-column prop="address" label="详细地址" min-width="200" />
        <el-table-column prop="contactName" label="联系人" width="100" />
        <el-table-column prop="phone" label="电话" width="120" />
        <el-table-column label="默认" width="70" align="center">
          <template #default="{ row }">
            <el-tag v-if="row.isDefault" type="success" size="small">默认</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="140">
          <template #default="{ row }">
            <el-button type="primary" link :icon="Edit" @click="handleEditAddress(row)">编辑</el-button>
            <el-popconfirm title="确定删除？" @confirm="handleDeleteAddress(row)">
              <template #reference>
                <el-button type="danger" link :icon="Delete">删除</el-button>
              </template>
            </el-popconfirm>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-card class="section-card">
      <template #header>
        <span>退货地址</span>
        <el-button type="primary" :icon="Plus" @click="handleAddAddress('return')">添加地址</el-button>
      </template>
      <el-table :data="returnAddresses" stripe>
        <el-table-column prop="label" label="标签" width="120" />
        <el-table-column label="地区" min-width="180">
          <template #default="{ row }">
            {{ [row.province, row.city, row.district].filter(Boolean).join(' ') }}
          </template>
        </el-table-column>
        <el-table-column prop="address" label="详细地址" min-width="200" />
        <el-table-column prop="contactName" label="联系人" width="100" />
        <el-table-column prop="phone" label="电话" width="120" />
        <el-table-column label="默认" width="70" align="center">
          <template #default="{ row }">
            <el-tag v-if="row.isDefault" type="success" size="small">默认</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="140">
          <template #default="{ row }">
            <el-button type="primary" link :icon="Edit" @click="handleEditAddress(row)">编辑</el-button>
            <el-popconfirm title="确定删除？" @confirm="handleDeleteAddress(row)">
              <template #reference>
                <el-button type="danger" link :icon="Delete">删除</el-button>
              </template>
            </el-popconfirm>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-card class="section-card">
      <template #header>
        <span>收款账户</span>
        <el-button type="primary" :icon="Plus" @click="handleAddAccount">添加账户</el-button>
      </template>
      <el-table :data="paymentAccounts" stripe>
        <el-table-column prop="label" label="标签" width="120" />
        <el-table-column label="类型" width="100">
          <template #default="{ row }">
            {{ ACCOUNT_TYPE_MAP[row.accountType] || row.accountType }}
          </template>
        </el-table-column>
        <el-table-column prop="accountName" label="户名" width="120" />
        <el-table-column prop="bankName" label="开户行" min-width="160" />
        <el-table-column prop="bankAccount" label="账号" min-width="180" />
        <el-table-column prop="remark" label="备注" min-width="120" />
        <el-table-column label="默认" width="70" align="center">
          <template #default="{ row }">
            <el-tag v-if="row.isDefault" type="success" size="small">默认</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="140">
          <template #default="{ row }">
            <el-button type="primary" link :icon="Edit" @click="handleEditAccount(row)">编辑</el-button>
            <el-popconfirm title="确定删除？" @confirm="handleDeleteAccount(row)">
              <template #reference>
                <el-button type="danger" link :icon="Delete">删除</el-button>
              </template>
            </el-popconfirm>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-card class="section-card">
      <template #header>
        <span>收款码</span>
        <el-button type="primary" :icon="Plus" @click="handleAddQR">添加收款码</el-button>
      </template>
      <el-table :data="paymentQRs" stripe>
        <el-table-column prop="label" label="标签" width="120" />
        <el-table-column label="类型" width="100">
          <template #default="{ row }">
            {{ PAY_TYPE_MAP[row.payType] || row.payType }}
          </template>
        </el-table-column>
        <el-table-column label="收款码" width="100" align="center">
          <template #default="{ row }">
            <el-image
              v-if="row.imageUrl"
              :src="row.imageUrl"
              fit="cover"
              class="qr-thumb"
              @click="openQRPreview(row.imageUrl)"
            />
            <span v-else>—</span>
          </template>
        </el-table-column>
        <el-table-column prop="accountName" label="收款人" width="120" />
        <el-table-column prop="remark" label="备注" min-width="140" />
        <el-table-column label="默认" width="70" align="center">
          <template #default="{ row }">
            <el-tag v-if="row.isDefault" type="success" size="small">默认</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="140">
          <template #default="{ row }">
            <el-button type="primary" link :icon="Edit" @click="handleEditQR(row)">编辑</el-button>
            <el-popconfirm title="确定删除？" @confirm="handleDeleteQR(row)">
              <template #reference>
                <el-button type="danger" link :icon="Delete">删除</el-button>
              </template>
            </el-popconfirm>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-dialog v-model="addrDialogVisible" :title="addrDialogTitle" width="520px">
      <el-form :model="editingAddr" label-width="90px">
        <el-form-item label="标签" required>
          <el-input
            v-model="editingAddr.label"
            :placeholder="editingAddrType === 'return' ? '如：退货仓' : '如：深圳仓'"
          />
        </el-form-item>
        <el-form-item label="省">
          <el-input v-model="editingAddr.province" />
        </el-form-item>
        <el-form-item label="市">
          <el-input v-model="editingAddr.city" />
        </el-form-item>
        <el-form-item label="区">
          <el-input v-model="editingAddr.district" />
        </el-form-item>
        <el-form-item label="详细地址">
          <el-input v-model="editingAddr.address" />
        </el-form-item>
        <el-form-item label="联系人">
          <el-input v-model="editingAddr.contactName" />
        </el-form-item>
        <el-form-item label="电话">
          <el-input v-model="editingAddr.phone" />
        </el-form-item>
        <el-form-item :label="editingAddrType === 'return' ? '默认退货地' : '默认发货地'">
          <el-switch v-model="editingAddr.isDefault" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="addrDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="handleSaveAddress">保存</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="accountDialogVisible" :title="editingAccount.id ? '编辑账户' : '添加账户'" width="520px">
      <el-form :model="editingAccount" label-width="90px">
        <el-form-item label="标签" required>
          <el-input v-model="editingAccount.label" placeholder="如：对公账户" />
        </el-form-item>
        <el-form-item label="类型">
          <el-select v-model="editingAccount.accountType" style="width: 100%">
            <el-option v-for="(label, key) in ACCOUNT_TYPE_MAP" :key="key" :label="label" :value="key" />
          </el-select>
        </el-form-item>
        <el-form-item label="户名">
          <el-input v-model="editingAccount.accountName" />
        </el-form-item>
        <el-form-item label="开户行">
          <el-input v-model="editingAccount.bankName" placeholder="银行账户时填写" />
        </el-form-item>
        <el-form-item label="账号">
          <el-input v-model="editingAccount.bankAccount" />
        </el-form-item>
        <el-form-item label="备注">
          <el-input v-model="editingAccount.remark" />
        </el-form-item>
        <el-form-item label="默认账户">
          <el-switch v-model="editingAccount.isDefault" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="accountDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="handleSaveAccount">保存</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="qrDialogVisible" :title="editingQR.id ? '编辑收款码' : '添加收款码'" width="520px">
      <el-form :model="editingQR" label-width="90px">
        <el-form-item label="标签" required>
          <el-input v-model="editingQR.label" placeholder="如：微信收款码" />
        </el-form-item>
        <el-form-item label="类型">
          <el-select v-model="editingQR.payType" style="width: 100%">
            <el-option v-for="(label, key) in PAY_TYPE_MAP" :key="key" :label="label" :value="key" />
          </el-select>
        </el-form-item>
        <el-form-item label="收款人">
          <el-input v-model="editingQR.accountName" />
        </el-form-item>
        <el-form-item label="收款码" required>
          <div class="qr-upload">
            <el-image
              v-if="editingQR.imageUrl"
              :src="editingQR.imageUrl"
              fit="contain"
              class="qr-preview"
              @click="openQRPreview(editingQR.imageUrl)"
            />
            <el-upload :auto-upload="false" :show-file-list="false" accept="image/*" :on-change="onQRFileChange">
              <el-button :icon="Upload" :loading="qrUploading">
                {{ editingQR.imageUrl ? '更换图片' : '上传图片' }}
              </el-button>
            </el-upload>
          </div>
        </el-form-item>
        <el-form-item label="备注">
          <el-input v-model="editingQR.remark" />
        </el-form-item>
        <el-form-item label="默认收款码">
          <el-switch v-model="editingQR.isDefault" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="qrDialogVisible = false">取消</el-button>
        <el-button type="primary" @click="handleSaveQR">保存</el-button>
      </template>
    </el-dialog>

    <el-dialog
      v-model="qrPreviewVisible"
      title="收款码预览"
      width="520px"
      align-center
      append-to-body
      class="qr-preview-dialog"
    >
      <div class="qr-preview-body">
        <img v-if="qrPreviewUrl" :src="qrPreviewUrl" alt="收款码" class="qr-preview-img" />
      </div>
    </el-dialog>
  </div>
</template>

<style scoped>
.detail-page {
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.info-card :deep(.el-card__header),
.section-card :deep(.el-card__header) {
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.section-card :deep(.el-card__body) {
  padding-top: 8px;
}
.qr-thumb {
  width: 48px;
  height: 48px;
  border-radius: 4px;
  cursor: pointer;
}
.qr-upload {
  display: flex;
  flex-direction: column;
  gap: 8px;
  align-items: flex-start;
}
.qr-preview {
  width: 160px;
  height: 160px;
  border-radius: 4px;
  cursor: pointer;
}
.qr-preview-body {
  display: flex;
  justify-content: center;
  align-items: center;
  padding: 12px 0;
}
.qr-preview-img {
  max-width: 440px;
  max-height: 440px;
  width: auto;
  height: auto;
  object-fit: contain;
  border-radius: 4px;
}
</style>
