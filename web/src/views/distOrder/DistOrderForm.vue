<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { ArrowLeft, Plus, Delete } from '@element-plus/icons-vue'
import {
  createDistOrder,
  fetchDistOrder,
  updateDistOrder,
  type DistOrderInput,
} from '../../api/distOrder'
import {
  createSkuPrice,
  fetchDistributors,
  fetchSkuPrices,
  type Distributor,
  type SkuPrice,
} from '../../api/distributor'
import SkuSearchSelect from '../../components/SkuSearchSelect.vue'
import OrderSearchSelect from '../../components/OrderSearchSelect.vue'
import {
  resolveProductSkus,
  type ProductSkuSearchItem,
} from '../../api/productSku'
import { fetchOrder, type OrderBrief } from '../../api/order'

const route = useRoute()
const router = useRouter()
const isEdit = computed(() => route.name === 'DistOrderEdit')
const poId = computed(() => (isEdit.value ? Number(route.params.id) : 0))

const loading = ref(false)
const saving = ref(false)
const orderLoading = ref(false)
const distributors = ref<Distributor[]>([])
const offers = ref<SkuPrice[]>([])
const offerSkuMap = ref<Map<number, ProductSkuSearchItem>>(new Map())

const offerDialogVisible = ref(false)
const offerSaving = ref(false)
const offerLineIndex = ref(-1)
const offerDraft = ref({
  distributorSkuCode: '',
  wholesalePrice: 0,
  supportsDropship: true,
  supportsWholesale: false,
})

function defaultPurchaseAt() {
  // 手工新建默认：当天常见采购时刻 10:00
  const d = new Date()
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} 10:00:00`
}

function formatOrderedAt(raw?: string) {
  if (!raw) return ''
  // 兼容 RFC3339 / 本地格式
  const d = new Date(raw)
  if (Number.isNaN(d.getTime())) {
    return raw.replace('T', ' ').slice(0, 19)
  }
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`
}

const form = ref<DistOrderInput>({
  distributorId: undefined,
  fulfillmentType: 'wholesale',
  currency: 'CNY',
  remark: '',
  orderedAt: defaultPurchaseAt(),
  items: [{ qty: 1, unitPrice: 0 }],
})

function roundMoney(n: number) {
  return Math.round((n + Number.EPSILON) * 100) / 100
}

function mapOrderToLines(order: OrderBrief) {
  const items = order.items || []
  if (!items.length) return [{ qty: 1, unitPrice: 0 }]

  let pay = Number(order.payAmount || 0)
  if (pay <= 0) pay = Number(order.totalAmount || 0)

  const weights = items.map((it) => {
    let w = Number(it.totalAmount || 0)
    if (w <= 0) {
      const qty = it.quantity > 0 ? it.quantity : 1
      w = Number(it.price || 0) * qty
    }
    return w > 0 ? w : 1
  })
  const sumW = weights.reduce((a, b) => a + b, 0)

  let allocated = 0
  const remarkParts: string[] = []
  if (order.remark?.trim()) remarkParts.push(`买家备注：${order.remark.trim()}`)
  if (order.sellerRemark?.trim()) remarkParts.push(`卖家备注：${order.sellerRemark.trim()}`)
  const lineRemark = remarkParts.join('；')

  return items.map((it, i) => {
    const qty = it.quantity > 0 ? it.quantity : 1
    let saleAmt = 0
    if (pay > 0 && sumW > 0) {
      if (i === items.length - 1) {
        saleAmt = roundMoney(pay - allocated)
      } else {
        saleAmt = roundMoney((pay * weights[i]) / sumW)
        allocated += saleAmt
      }
    } else {
      saleAmt = roundMoney(Number(it.totalAmount || 0) || Number(it.price || 0) * qty)
    }
    if (saleAmt < 0) saleAmt = 0
    const saleUnit = qty > 0 ? roundMoney(saleAmt / qty) : 0

    let unitPrice = 0
    let offerId: number | undefined
    let distributorSkuCode: string | undefined
    const skuId = it.skuId || undefined
    if (skuId && form.value.distributorId) {
      const offer = offers.value.find((o) => o.skuId === skuId)
      if (offer) {
        offerId = offer.id
        unitPrice = offer.wholesalePrice
        distributorSkuCode = offer.distributorSkuCode
      }
    }

    return {
      skuId,
      offerId,
      productName: it.productName,
      skuCode: it.skuCode,
      skuSpecs: it.skuSpecs,
      picUrl: it.picUrl,
      distributorSkuCode,
      qty,
      saleUnitPrice: saleUnit,
      saleAmount: saleAmt,
      unitPrice,
      refSoId: order.id || undefined,
      refOrderNo: order.orderNo || undefined,
      remark: lineRemark || undefined,
    }
  })
}

async function onOrderSelect(item: OrderBrief | undefined) {
  if (!item) {
    form.value.refSoId = undefined
    form.value.refTraceId = undefined
    return
  }
  form.value.refTraceId = item.orderNo
  form.value.refSoId = item.id || undefined

  if (!item.id) {
    ElMessage.warning('未找到订单详情，请重新搜索选择')
    return
  }

  orderLoading.value = true
  try {
    const order = await fetchOrder(item.id)
    form.value.refTraceId = order.orderNo
    form.value.refSoId = order.id
    const pay = Number(order.payAmount || order.totalAmount || 0)
    if (pay > 0) form.value.saleAmount = pay
    // 下单时间保持表单默认/用户所选，不随销售单下单时间覆盖
    form.value.items = mapOrderToLines(order)
    ElMessage.success(`已带入订单 ${order.orderNo} 的 ${form.value.items.length} 行明细`)
  } catch (e) {
    ElMessage.error((e as Error).message || '加载订单明细失败')
  } finally {
    orderLoading.value = false
  }
}

async function loadDistributors() {
  const data = await fetchDistributors({ page: 1, pageSize: 200 })
  distributors.value = data.list
}

async function loadOffers(distributorId: number) {
  if (!distributorId) {
    offers.value = []
    offerSkuMap.value = new Map()
    return
  }
  try {
    const data = await fetchSkuPrices({ distributorId, page: 1, pageSize: 500 })
    offers.value = data.list
    offerSkuMap.value = await resolveProductSkus(data.list.map((o) => o.skuId))
  } catch {
    offers.value = []
    offerSkuMap.value = new Map()
  }
}

function offerLabel(o: SkuPrice) {
  const info = offerSkuMap.value.get(o.skuId)
  const code = info?.skuCode?.trim() || '未编码'
  const bits = [code, `¥${Number(o.wholesalePrice || 0).toFixed(2)}`]
  if (o.distributorSkuCode) bits.push(`对方 ${o.distributorSkuCode}`)
  return bits.join(' · ')
}

async function loadPO() {
  if (!isEdit.value || !poId.value) return
  loading.value = true
  try {
    const po = await fetchDistOrder(poId.value)
    if (po.status !== 'draft') {
      ElMessage.warning('仅草稿可编辑')
      router.replace(`/dist-orders/${poId.value}`)
      return
    }
    form.value = {
      distributorId: po.distributorId,
      fulfillmentType: po.fulfillmentType,
      currency: po.currency,
      expectedArrivalDate: po.expectedArrivalDate,
      warehouseId: po.warehouseId,
      refSoId: po.refSoId,
      refTraceId: po.refTraceId,
      orderedAt: formatOrderedAt(po.orderedAt) || defaultPurchaseAt(),
      remark: po.remark,
      items: po.items.map((it) => ({
        skuId: it.skuId || undefined,
        offerId: it.offerId || undefined,
        productName: it.productName,
        skuCode: it.skuCode,
        skuSpecs: it.skuSpecs,
        picUrl: it.picUrl,
        distributorSkuCode: it.distributorSkuCode,
        qty: it.qty,
        saleUnitPrice: it.saleUnitPrice,
        saleAmount: it.saleAmount,
        unitPrice: it.unitPrice,
        remark: it.remark,
      })),
    }
    await loadOffers(po.distributorId)
  } catch (e) {
    ElMessage.error((e as Error).message || '加载失败')
  } finally {
    loading.value = false
  }
}

onMounted(async () => {
  if (!isEdit.value) {
    const ft = route.query.fulfillmentType
    if (ft === 'dropship' || ft === 'wholesale') {
      form.value.fulfillmentType = ft
    }
  }
  await loadDistributors()
  await loadPO()
})

watch(() => form.value.distributorId, async (id) => {
  await loadOffers(id || 0)
  if (!id) return
  // 已选订单明细时，按分销商报价自动带入批发价
  for (const line of form.value.items) {
    if (!line.skuId) continue
    const offer = offers.value.find((o) => o.skuId === line.skuId)
    if (!offer) continue
    line.offerId = offer.id
    line.unitPrice = offer.wholesalePrice
    if (offer.distributorSkuCode) line.distributorSkuCode = offer.distributorSkuCode
  }
})

function addLine() {
  form.value.items.push({ qty: 1, unitPrice: 0 })
}

function removeLine(index: number) {
  if (form.value.items.length <= 1) return
  form.value.items.splice(index, 1)
}

function applyOffer(index: number, offerId: number) {
  const offer = offers.value.find((o) => o.id === offerId)
  if (!offer) return
  const line = form.value.items[index]
  line.offerId = offer.id
  line.skuId = offer.skuId
  line.distributorSkuCode = offer.distributorSkuCode
  line.unitPrice = offer.wholesalePrice
  const info = offerSkuMap.value.get(offer.skuId)
  if (info?.productName) line.productName = info.productName
  if (info?.skuCode) line.skuCode = info.skuCode
  if (info?.specLabel) line.skuSpecs = info.specLabel
  if (info?.pic || info?.productPic) line.picUrl = info.pic || info.productPic
}

function onSkuPicked(index: number, item: ProductSkuSearchItem | undefined) {
  const line = form.value.items[index]
  if (!item) {
    // 清空商家编码选择时，保留 OMS 已带入的商品名/图片/规格，避免编辑页被冲掉
    return
  }
  line.productName = item.productName
  line.skuCode = item.skuCode
  line.skuSpecs = item.specLabel || line.skuSpecs
  line.picUrl = item.pic || item.productPic || line.picUrl
  if (line.offerId) {
    const offer = offers.value.find((o) => o.id === line.offerId)
    if (offer && offer.skuId !== item.skuId) {
      line.offerId = undefined
    }
  }
}

function openCreateOffer(index: number) {
  const line = form.value.items[index]
  if (!form.value.distributorId) {
    ElMessage.warning('请先选择分销商')
    return
  }
  if (!line.skuId) {
    ElMessage.warning('请先选择商家编码对应的商品')
    return
  }
  offerLineIndex.value = index
  offerDraft.value = {
    distributorSkuCode: line.distributorSkuCode || '',
    wholesalePrice: Number(line.unitPrice || 0),
    supportsDropship: form.value.fulfillmentType === 'dropship',
    supportsWholesale: form.value.fulfillmentType !== 'dropship',
  }
  offerDialogVisible.value = true
}

async function saveInlineOffer() {
  const index = offerLineIndex.value
  const line = form.value.items[index]
  if (!line?.skuId || !form.value.distributorId) return
  if (offerDraft.value.wholesalePrice < 0) {
    ElMessage.warning('请填写批发价')
    return
  }
  offerSaving.value = true
  try {
    const created = await createSkuPrice({
      skuId: line.skuId,
      distributorId: form.value.distributorId,
      distributorSkuCode: offerDraft.value.distributorSkuCode,
      wholesalePrice: offerDraft.value.wholesalePrice,
      currency: 'CNY',
      minOrderQty: 1,
      supportsDropship: offerDraft.value.supportsDropship,
      supportsWholesale: offerDraft.value.supportsWholesale,
      status: 1,
    })
    await loadOffers(form.value.distributorId)
    line.offerId = created.id
    line.unitPrice = created.wholesalePrice
    line.distributorSkuCode = created.distributorSkuCode || offerDraft.value.distributorSkuCode
    offerDialogVisible.value = false
    ElMessage.success('已保存到 SKU 批发价并应用到本行')
  } catch (e) {
    ElMessage.error((e as Error).message || '保存报价失败')
  } finally {
    offerSaving.value = false
  }
}

const lineTotal = computed(() =>
  form.value.items.reduce((sum, it) => sum + it.qty * (it.unitPrice || 0), 0),
)

async function handleSave() {
  if (!form.value.distributorId) {
    ElMessage.warning('请选择分销商')
    return
  }
  const dropship = form.value.fulfillmentType === 'dropship'
  for (const it of form.value.items) {
    if (it.qty <= 0) {
      ElMessage.warning('请完善明细数量')
      return
    }
    if (!dropship && !it.skuId) {
      ElMessage.warning('请选择商家编码')
      return
    }
    if (!dropship && (it.unitPrice == null || it.unitPrice <= 0) && !it.offerId) {
      ElMessage.warning('请填写分销订单价或选择批发价')
      return
    }
    if (dropship && !it.skuId && !it.productName) {
      ElMessage.warning('代发明细需有商品名称或商家编码')
      return
    }
  }
  saving.value = true
  try {
    if (isEdit.value) {
      await updateDistOrder(poId.value, form.value)
      ElMessage.success('已保存')
      router.push(`/dist-orders/${poId.value}`)
    } else {
      const po = await createDistOrder(form.value)
      ElMessage.success('已创建')
      router.push(`/dist-orders/${po.id}`)
    }
  } catch (e) {
    ElMessage.error((e as Error).message || '保存失败')
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <div v-loading="loading || orderLoading" class="po-form">
    <el-button :icon="ArrowLeft" text @click="router.push('/dist-orders')">返回列表</el-button>

    <el-card>
      <template #header>{{ isEdit ? '编辑分销订单' : '新建分销订单' }}</template>

      <el-form label-width="100px">
        <el-row :gutter="16">
          <el-col :span="12">
            <el-form-item label="分销商" required>
              <el-select
                v-model="form.distributorId"
                filterable
                clearable
                placeholder="搜索分销商名称 / 编码"
                style="width: 100%"
              >
                <el-option
                  v-for="s in distributors"
                  :key="s.id"
                  :label="`${s.name}（${s.code}）`"
                  :value="s.id"
                />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="订单类型">
              <el-select v-model="form.fulfillmentType" style="width: 100%">
                <el-option label="批发" value="wholesale" />
                <el-option label="代发" value="dropship" />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="预计到货">
              <el-date-picker
                v-model="form.expectedArrivalDate"
                type="date"
                value-format="YYYY-MM-DD"
                placeholder="可选"
                style="width: 100%"
              />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="关联订单">
              <OrderSearchSelect
                v-model="form.refTraceId"
                @select="onOrderSelect"
              />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="下单时间">
              <el-date-picker
                v-model="form.orderedAt"
                type="datetime"
                value-format="YYYY-MM-DD HH:mm:ss"
                format="YYYY-MM-DD HH:mm"
                placeholder="默认当天 10:00，可改"
                style="width: 100%"
              />
            </el-form-item>
          </el-col>
          <el-col :span="24">
            <el-form-item label="备注">
              <el-input v-model="form.remark" type="textarea" :rows="2" />
            </el-form-item>
          </el-col>
        </el-row>
      </el-form>

      <div class="lines-header">
        <span>采购明细</span>
        <el-button type="primary" link :icon="Plus" @click="addLine">添加行</el-button>
      </div>
      <div class="hint-bar">
        可不选批发价，直接填写分销订单价；也可「存为报价」写入 SKU 批发价后继续下单。
      </div>

      <el-table :data="form.items" border size="small" class="lines-table">
        <el-table-column label="图片" width="72" align="center">
          <template #default="{ row }">
            <el-image
              v-if="row.picUrl"
              :src="row.picUrl"
              :preview-src-list="[row.picUrl]"
              fit="cover"
              class="sku-pic"
              preview-teleported
            />
            <span v-else class="muted">—</span>
          </template>
        </el-table-column>
        <el-table-column label="商品" min-width="140">
          <template #default="{ row }">
            <div class="name">{{ row.productName || '—' }}</div>
            <div v-if="row.remark" class="sub">{{ row.remark }}</div>
          </template>
        </el-table-column>
        <el-table-column label="规格" width="120">
          <template #default="{ row }">
            <el-input v-model="row.skuSpecs" placeholder="规格" />
          </template>
        </el-table-column>
        <el-table-column label="批发价" width="200">
          <template #default="{ row, $index }">
            <el-select
              :model-value="row.offerId"
              placeholder="可选"
              clearable
              filterable
              style="width: 100%"
              @update:model-value="(v: number) => applyOffer($index, v)"
            >
              <el-option v-for="o in offers" :key="o.id" :label="offerLabel(o)" :value="o.id" />
            </el-select>
            <el-button type="primary" link size="small" @click="openCreateOffer($index)">存为报价</el-button>
          </template>
        </el-table-column>
        <el-table-column label="商家编码" min-width="200">
          <template #default="{ row, $index }">
            <SkuSearchSelect
              v-model="row.skuId"
              :show-preview="false"
              placeholder="搜索商家编码（可选）"
              @select="(item) => onSkuPicked($index, item)"
            />
          </template>
        </el-table-column>
        <el-table-column label="对方货号" width="110">
          <template #default="{ row }">
            <el-input v-model="row.distributorSkuCode" placeholder="分销商货号" />
          </template>
        </el-table-column>
        <el-table-column label="数量" width="90">
          <template #default="{ row }">
            <el-input-number v-model="row.qty" :min="1" controls-position="right" style="width: 100%" />
          </template>
        </el-table-column>
        <el-table-column label="实付金额" width="100" align="right">
          <template #default="{ row }">
            <span v-if="row.saleAmount">¥{{ Number(row.saleAmount).toFixed(2) }}</span>
            <span v-else class="muted">—</span>
          </template>
        </el-table-column>
        <el-table-column label="分销订单价" width="120">
          <template #default="{ row }">
            <el-input-number
              v-model="row.unitPrice"
              :min="0"
              :precision="2"
              controls-position="right"
              style="width: 100%"
            />
          </template>
        </el-table-column>
        <el-table-column label="采购小计" width="90" align="right">
          <template #default="{ row }">¥{{ (row.qty * (row.unitPrice || 0)).toFixed(2) }}</template>
        </el-table-column>
        <el-table-column width="50" align="center">
          <template #default="{ $index }">
            <el-button type="danger" link :icon="Delete" @click="removeLine($index)" />
          </template>
        </el-table-column>
      </el-table>

      <div class="footer">
        <span class="total">合计：¥{{ lineTotal.toFixed(2) }}</span>
        <div>
          <el-button @click="router.push('/dist-orders')">取消</el-button>
          <el-button type="primary" :loading="saving" @click="handleSave">保存草稿</el-button>
        </div>
      </div>
    </el-card>

    <el-dialog v-model="offerDialogVisible" title="现场新增批发价" width="440px" destroy-on-close>
      <el-form label-width="90px">
        <el-form-item label="对方货号">
          <el-input v-model="offerDraft.distributorSkuCode" placeholder="分销商侧货号（可选）" />
        </el-form-item>
        <el-form-item label="批发价" required>
          <el-input-number
            v-model="offerDraft.wholesalePrice"
            :min="0"
            :precision="2"
            controls-position="right"
            style="width: 100%"
          />
        </el-form-item>
        <el-form-item label="支持代发">
          <el-switch v-model="offerDraft.supportsDropship" />
        </el-form-item>
        <el-form-item label="供货到仓">
          <el-switch v-model="offerDraft.supportsWholesale" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="offerDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="offerSaving" @click="saveInlineOffer">保存并应用</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.po-form {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.lines-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin: 16px 0 4px;
  font-weight: 600;
}
.hint-bar {
  font-size: 12px;
  color: #909399;
  margin-bottom: 8px;
}
.sku-pic {
  width: 40px;
  height: 40px;
  border-radius: 4px;
}
.name {
  font-size: 13px;
  line-height: 1.35;
}
.sub {
  font-size: 12px;
  color: #909399;
  margin-top: 2px;
}
.muted {
  color: #c0c4cc;
}
.footer {
  margin-top: 16px;
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.total {
  font-size: 16px;
  font-weight: 600;
  color: #303133;
}
.lines-table :deep(.el-table__cell) {
  vertical-align: top;
}
</style>
