<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus, Delete } from '@element-plus/icons-vue'
import type { SelfOrderDetail } from '../../api/selfOrder'
import {
  fetchSelfPayments,
  createSelfPayment,
  deleteSelfPayment,
  fetchSelfAttachments,
  createSelfAttachment,
  deleteSelfAttachment,
  type SelfPayment,
  type SelfAttachment,
} from '../../api/selfOrderTracking'
import ScanImageUpload from '../../components/ScanImageUpload.vue'

const props = defineProps<{ selfOrderId: number; order: SelfOrderDetail; readonly: boolean }>()
const emit = defineEmits<{ refresh: [] }>()

const loading = ref(false)
const saving = ref(false)
const list = ref<SelfPayment[]>([])
const attachments = ref<SelfAttachment[]>([])
const dialogVisible = ref(false)
const form = ref({
  payAmount: 0,
  payMethod: 'bank',
  payAccount: '',
  payeeAccount: '',
  payeeName: '',
  remark: '',
})
const pendingShotUrls = ref<string[]>([])

const paidSum = computed(() =>
  list.value.filter((p) => p.payStatus === 'paid').reduce((s, p) => s + p.payAmount, 0),
)

const remainAmount = computed(() => Math.max(0, Number(props.order.saleAmount || 0) - paidSum.value))

const screenshotsByPayment = computed(() => {
  const map = new Map<number, SelfAttachment[]>()
  for (const a of attachments.value) {
    if (a.fileType !== 'payment_screenshot' || !a.paymentId) continue
    const arr = map.get(a.paymentId) || []
    arr.push(a)
    map.set(a.paymentId, arr)
  }
  return map
})

function fileNameFromUrl(url: string) {
  try {
    const path = url.split('?')[0]
    const name = path.split('/').pop() || ''
    return decodeURIComponent(name) || '付款截图.jpg'
  } catch {
    return '付款截图.jpg'
  }
}

async function loadData() {
  loading.value = true
  try {
    const [payments, files] = await Promise.all([
      fetchSelfPayments(props.selfOrderId),
      fetchSelfAttachments(props.selfOrderId),
    ])
    list.value = payments
    attachments.value = files
  } catch (e) {
    ElMessage.error((e as Error).message || '加载失败')
  } finally {
    loading.value = false
  }
}

onMounted(loadData)

function openCreate() {
  form.value = {
    payAmount: remainAmount.value || Number(props.order.saleAmount || 0) || 0.01,
    payMethod: 'bank',
    payAccount: '',
    payeeAccount: '',
    payeeName: '',
    remark: '',
  }
  pendingShotUrls.value = []
  dialogVisible.value = true
}

async function handleSave() {
  if (form.value.payAmount <= 0) {
    ElMessage.warning('请输入付款金额')
    return
  }
  if (Number(props.order.saleAmount || 0) <= 0) {
    try {
      await ElMessageBox.confirm(
        '当前销售金额为 ¥0.00，记录付款后将无法按金额自动标记已付清。是否仍要继续？',
        '提示',
        { type: 'warning' },
      )
    } catch {
      return
    }
  }
  saving.value = true
  try {
    const payment = await createSelfPayment(props.selfOrderId, { ...form.value, payStatus: 'paid' })
    for (const url of pendingShotUrls.value) {
      await createSelfAttachment(props.selfOrderId, {
        fileType: 'payment_screenshot',
        fileName: fileNameFromUrl(url),
        fileUrl: url,
        paymentId: payment.id,
        remark: '付款截图',
      })
    }
    const nextPaid = paidSum.value + form.value.payAmount
    const total = Number(props.order.saleAmount || 0)
    if (total > 0 && nextPaid + 0.001 >= total) {
      ElMessage.success('已记录付款，自营订单已自动标记为已付清')
    } else if (total > 0) {
      ElMessage.success(`已记录付款（已付 ¥${nextPaid.toFixed(2)} / ¥${total.toFixed(2)}）`)
    } else {
      ElMessage.success('已记录付款')
    }
    dialogVisible.value = false
    await loadData()
    emit('refresh')
  } catch (e) {
    ElMessage.error((e as Error).message || '保存失败')
  } finally {
    saving.value = false
  }
}

async function handleDelete(row: SelfPayment) {
  try {
    await ElMessageBox.confirm('确定删除此付款记录？关联的付款截图也会删除。', '确认')
  } catch {
    return
  }
  try {
    const shots = screenshotsByPayment.value.get(row.id) || []
    for (const a of shots) {
      await deleteSelfAttachment(props.selfOrderId, a.id)
    }
    await deleteSelfPayment(props.selfOrderId, row.id)
    ElMessage.success('已删除')
    await loadData()
    emit('refresh')
  } catch (e) {
    ElMessage.error((e as Error).message || '删除失败')
  }
}

const payMethodLabel: Record<string, string> = {
  bank: '银行转账',
  alipay: '支付宝',
  wechat: '微信',
  other: '其他',
}

function payStatusLabel(status: string) {
  if (status === 'partial') return '部分付款'
  if (status === 'paid') return '已付清'
  return '未付款'
}
</script>

<template>
  <div v-loading="loading">
    <div class="summary">
      销售金额 ¥{{ Number(order.saleAmount || 0).toFixed(2) }}
      · 已付 ¥{{ paidSum.toFixed(2) }}
      · 待付 ¥{{ remainAmount.toFixed(2) }}
      · 付款状态 {{ payStatusLabel(order.payStatus || 'unpaid') }}
    </div>
    <div v-if="!readonly" class="toolbar">
      <el-button type="primary" :icon="Plus" @click="openCreate">记录付款</el-button>
    </div>
    <el-table :data="list" border stripe>
      <el-table-column label="金额" width="120" align="right">
        <template #default="{ row }">¥{{ row.payAmount.toFixed(2) }}</template>
      </el-table-column>
      <el-table-column label="方式" width="100">
        <template #default="{ row }">{{ payMethodLabel[row.payMethod || ''] || row.payMethod || '—' }}</template>
      </el-table-column>
      <el-table-column label="付款截图" min-width="160">
        <template #default="{ row }">
          <div v-if="screenshotsByPayment.get(row.id)?.length" class="shots">
            <el-image
              v-for="a in screenshotsByPayment.get(row.id)"
              :key="a.id"
              :src="a.fileUrl"
              :preview-src-list="(screenshotsByPayment.get(row.id) || []).map((x) => x.fileUrl)"
              fit="cover"
              class="shot"
              preview-teleported
            />
          </div>
          <span v-else class="muted">—</span>
        </template>
      </el-table-column>
      <el-table-column prop="payAccount" label="打款账号" min-width="120" />
      <el-table-column prop="payeeAccount" label="收款账号" min-width="120" />
      <el-table-column prop="payeeName" label="收款户名" width="100" />
      <el-table-column prop="paidAt" label="打款时间" width="150" />
      <el-table-column prop="remark" label="备注" min-width="100" />
      <el-table-column v-if="!readonly" label="操作" width="80">
        <template #default="{ row }">
          <el-button link type="danger" :icon="Delete" @click="handleDelete(row)" />
        </template>
      </el-table-column>
    </el-table>

    <el-dialog v-model="dialogVisible" title="记录付款" width="560px">
      <el-form :model="form" label-width="90px">
        <el-form-item label="付款金额" required>
          <el-input-number
            v-model="form.payAmount"
            :min="0.01"
            :precision="2"
            controls-position="right"
            style="width: 100%"
          />
          <div class="form-hint">
            销售金额 ¥{{ Number(order.saleAmount || 0).toFixed(2) }}，待付 ¥{{ remainAmount.toFixed(2) }}；
            累计付清后自动标记已付清
          </div>
        </el-form-item>
        <el-form-item label="付款方式">
          <el-select v-model="form.payMethod" style="width: 100%">
            <el-option label="银行转账" value="bank" />
            <el-option label="支付宝" value="alipay" />
            <el-option label="微信" value="wechat" />
            <el-option label="其他" value="other" />
          </el-select>
        </el-form-item>
        <el-form-item label="付款截图">
          <ScanImageUpload
            v-model="pendingShotUrls"
            subdir="self/payments"
            tip="本机上传"
            scan-title="手机扫码上传付款截图"
          />
        </el-form-item>
        <el-form-item label="打款账号">
          <el-input v-model="form.payAccount" />
        </el-form-item>
        <el-form-item label="收款账号">
          <el-input v-model="form.payeeAccount" />
        </el-form-item>
        <el-form-item label="收款户名">
          <el-input v-model="form.payeeName" />
        </el-form-item>
        <el-form-item label="备注">
          <el-input v-model="form.remark" />
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
.summary {
  margin-bottom: 12px;
  color: #606266;
  font-size: 14px;
}
.toolbar {
  margin-bottom: 12px;
}
.form-hint {
  margin-top: 6px;
  color: var(--el-text-color-secondary);
  font-size: 12px;
  line-height: 1.4;
}
.shots {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}
.shot {
  width: 40px;
  height: 40px;
  border-radius: 4px;
}
.muted {
  color: var(--el-text-color-placeholder);
}
</style>
