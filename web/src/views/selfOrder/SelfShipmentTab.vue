<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Delete } from '@element-plus/icons-vue'
import type { SelfOrderDetail, SelfOrderItem } from '../../api/selfOrder'
import {
  fetchSelfShipments,
  createSelfShipment,
  updateSelfShipmentStatus,
  deleteSelfShipment,
  syncSelfShipmentsFromOrders,
  fetchSelfAttachments,
  createSelfAttachment,
  type SelfShipment,
  type SelfAttachment,
} from '../../api/selfOrderTracking'
import { SHIPMENT_STATUS_MAP } from '../../api/tracking'
import { fetchOrder, formatOrderReceiverAddress, shipOrder } from '../../api/order'
import { retryCallback } from '../../api/selfOrder'
import { EXPRESS_COMPANIES, findExpressCompany } from '../../constants/expressCompanies'
import ScanImageUpload from '../../components/ScanImageUpload.vue'

const props = defineProps<{ selfOrderId: number; order: SelfOrderDetail; readonly: boolean }>()
const emit = defineEmits<{ refresh: [] }>()

const loading = ref(false)
const syncing = ref(false)
const callbacking = ref(false)
const saving = ref(false)
const list = ref<SelfShipment[]>([])
const attachments = ref<SelfAttachment[]>([])
const dialogVisible = ref(false)
const callbackVisible = ref(false)
const photoVisible = ref(false)
const photoSaving = ref(false)
const photoTarget = ref<SelfShipment | null>(null)
const photoUrls = ref<string[]>([])
const pendingPhotoUrls = ref<string[]>([])
const loadingAddr = ref(false)
const form = ref({
  carrierCode: '',
  carrierName: '',
  trackingNo: '',
  expectedArrivalDate: '',
  receiverName: '',
  receiverPhone: '',
  receiverAddress: '',
  remark: '',
})
const callbackForm = ref({
  refSoId: 0,
  refOrderNo: '',
  carrierCode: '',
  carrierName: '',
  trackingNo: '',
  remark: '',
})

function onCarrierChange(code: string, target: 'form' | 'callback') {
  const hit = findExpressCompany(code)
  const name = hit?.name || code || ''
  const resolvedCode = hit?.code || code || ''
  if (target === 'form') {
    form.value.carrierCode = resolvedCode
    form.value.carrierName = name
  } else {
    callbackForm.value.carrierCode = resolvedCode
    callbackForm.value.carrierName = name
  }
}

function onFormCarrierChange(v: string | number | null | undefined) {
  onCarrierChange(String(v || ''), 'form')
}

function onCallbackCarrierChange(v: string | number | null | undefined) {
  onCarrierChange(String(v || ''), 'callback')
}

function fileNameFromUrl(url: string, fallback = '物流照片.jpg') {
  try {
    const path = url.split('?')[0]
    const name = path.split('/').pop() || ''
    return decodeURIComponent(name) || fallback
  } catch {
    return fallback
  }
}

const photosByShipment = computed(() => {
  const map = new Map<number, SelfAttachment[]>()
  for (const a of attachments.value) {
    if (a.fileType !== 'shipment_photo' || !a.shipmentId) continue
    const arr = map.get(a.shipmentId) || []
    arr.push(a)
    map.set(a.shipmentId, arr)
  }
  return map
})

async function bindShipmentPhotos(shipmentId: number, urls: string[]) {
  for (const url of urls) {
    await createSelfAttachment(props.selfOrderId, {
      fileType: 'shipment_photo',
      fileName: fileNameFromUrl(url),
      fileUrl: url,
      shipmentId,
      remark: '发货记录/物流单号照片',
    })
  }
}

interface LinePick {
  selfOrderItemId: number
  productName: string
  skuCode: string
  skuSpecs: string
  picUrl?: string
  qty: number
  shippedQty: number
  remaining: number
  selected: boolean
  shipQty: number
  refSoId?: number
  refOrderNo?: string
}

interface SalesOrderGroup {
  key: string
  refSoId: number
  refOrderNo: string
  lines: LinePick[]
  remainingQty: number
  receiverHint: string
  shipStatusHint: string
}

const linePicks = ref<LinePick[]>([])
const soGroups = ref<SalesOrderGroup[]>([])
const activeGroupKey = ref('')
const addressHint = ref('')

const itemLabelMap = computed(() => {
  const map = new Map<number, SelfOrderItem>()
  for (const it of props.order.items || []) {
    if (it.id) map.set(it.id, it)
  }
  return map
})

function shippedQtyByItem(): Map<number, number> {
  const map = new Map<number, number>()
  for (const sh of list.value) {
    for (const it of sh.items || []) {
      map.set(it.selfOrderItemId, (map.get(it.selfOrderItemId) || 0) + it.qty)
    }
  }
  return map
}

function rebuildLinePicks() {
  const shipped = shippedQtyByItem()
  linePicks.value = (props.order.items || [])
    .filter((it) => it.id)
    .map((it) => {
      const shippedQty = shipped.get(it.id!) || 0
      const remaining = Math.max(0, it.qty - shippedQty)
      return {
        selfOrderItemId: it.id!,
        productName: it.productName || '—',
        skuCode: it.skuCode || '',
        skuSpecs: it.skuSpecs || '',
        picUrl: it.picUrl,
        qty: it.qty,
        shippedQty,
        remaining,
        selected: remaining > 0,
        shipQty: remaining > 0 ? remaining : 1,
        refSoId: it.refSoId || 0,
        refOrderNo: it.refOrderNo || '',
      }
    })
}

function rebuildSoGroups() {
  rebuildLinePicks()
  const headerSoId = Number(props.order.refSoId || 0)
  const headerOrderNo = (props.order.refTraceId || '').trim()
  const map = new Map<string, SalesOrderGroup>()
  for (const line of linePicks.value) {
    const orderNo = (line.refOrderNo || '').trim() || (headerOrderNo.includes(',') ? '' : headerOrderNo)
    const soId = line.refSoId || headerSoId || 0
    const key = soId > 0 ? `id:${soId}` : orderNo ? `no:${orderNo}` : `item:${line.selfOrderItemId}`
    let g = map.get(key)
    if (!g) {
      g = {
        key,
        refSoId: soId,
        refOrderNo: orderNo || (soId ? `订单#${soId}` : `明细#${line.selfOrderItemId}`),
        lines: [],
        remainingQty: 0,
        receiverHint: '',
        shipStatusHint: '',
      }
      map.set(key, g)
    }
    g.lines.push(line)
    g.remainingQty += line.remaining
  }
  const groups = [...map.values()]
  for (const g of groups) {
    const shippedAll = g.lines.every((l) => l.remaining <= 0)
    const partial = g.lines.some((l) => l.shippedQty > 0) && !shippedAll
    g.shipStatusHint = shippedAll ? '已登记物流' : partial ? '部分发货' : '待发货'
  }
  soGroups.value = groups
}

/** 一销售单多规格行 → 分行展示（销售单号相同） */
const soGroupRows = computed(() => {
  const rows: {
    key: string
    group: SalesOrderGroup
    line: LinePick
    lineStatus: string
  }[] = []
  for (const g of soGroups.value) {
    for (const line of g.lines) {
      const done = line.remaining <= 0
      const partial = line.shippedQty > 0 && !done
      rows.push({
        key: `${g.key}:${line.selfOrderItemId}`,
        group: g,
        line,
        lineStatus: done ? '已登记物流' : partial ? '部分发货' : '待发货',
      })
    }
  }
  return rows
})

function formatSpecLabel(specs?: string, qty?: number) {
  const s = (specs || '').trim() || '—'
  return qty != null ? `${s} ×${qty}` : s
}

function shipmentSpecText(row: SelfShipment) {
  const items = row.items || []
  if (!items.length) return '—'
  return items
    .map((it) => {
      const orderItem = itemLabelMap.value.get(it.selfOrderItemId)
      return formatSpecLabel(orderItem?.skuSpecs, it.qty)
    })
    .join('；')
}

function shipmentSalesOrders(row: SelfShipment) {
  const nos = new Set<string>()
  const headerOrderNo = (props.order.refTraceId || '').trim()
  const headerSoId = Number(props.order.refSoId || 0)
  for (const it of row.items || []) {
    const orderItem = itemLabelMap.value.get(it.selfOrderItemId)
    const no = orderItem?.refOrderNo?.trim()
    if (no) nos.add(no)
    else if (orderItem?.refSoId) nos.add(`订单#${orderItem.refSoId}`)
  }
  if (!nos.size && headerOrderNo && !headerOrderNo.includes(',')) nos.add(headerOrderNo)
  else if (!nos.size && headerSoId) nos.add(`订单#${headerSoId}`)
  return [...nos].join('、') || '—'
}

function lineLogisticsText(itemId: number) {
  const texts: string[] = []
  for (const sh of list.value) {
    const hit = (sh.items || []).some((it) => it.selfOrderItemId === itemId)
    if (!hit) continue
    const t = [sh.carrierName, sh.trackingNo].filter(Boolean).join(' ')
    if (t && !texts.includes(t)) texts.push(t)
  }
  return texts.join('；') || '—'
}

async function loadData() {
  loading.value = true
  try {
    const [shipments, files] = await Promise.all([
      fetchSelfShipments(props.selfOrderId),
      fetchSelfAttachments(props.selfOrderId),
    ])
    list.value = shipments
    attachments.value = files
    rebuildSoGroups()
  } catch (e) {
    ElMessage.error((e as Error).message || '加载失败')
  } finally {
    loading.value = false
  }
}

onMounted(loadData)
watch(() => props.order.items, () => {
  rebuildSoGroups()
})
watch(() => props.selfOrderId, loadData)

function resetForm() {
  form.value = {
    carrierCode: '', carrierName: '', trackingNo: '', expectedArrivalDate: '',
    receiverName: '', receiverPhone: '', receiverAddress: '', remark: '',
  }
  pendingPhotoUrls.value = []
  addressHint.value = ''
}

async function fillReceiverFromOrder(soId: number) {
  if (!soId) {
    addressHint.value = '该明细未关联销售单 ID，请手工填写收件人'
    return
  }
  loadingAddr.value = true
  addressHint.value = ''
  try {
    const ord = await fetchOrder(soId)
    const addr = ord.address
    const name = addr?.name?.trim() || ord.buyerName?.trim() || ''
    const phone = addr?.phone?.trim() || ord.buyerPhone?.trim() || ''
    const address = formatOrderReceiverAddress(addr)
    form.value.receiverName = name
    form.value.receiverPhone = phone
    form.value.receiverAddress = address
    if (!name && !phone && !address) {
      addressHint.value = '订单中心暂无明文收件人（可能已脱敏），请到订单中心解密后重试，或手工填写'
    } else {
      addressHint.value = `已从订单 ${ord.orderNo} 带入收件人`
    }
  } catch (e) {
    addressHint.value = (e as Error).message || '拉取订单收件人失败，请手工填写'
  } finally {
    loadingAddr.value = false
  }
}

async function openCreateDropship(group: SalesOrderGroup, preferItemId?: number) {
  if (group.remainingQty <= 0) {
    ElMessage.warning('该销售单明细已全部关联物流')
    return
  }
  resetForm()
  activeGroupKey.value = group.key
  const prefer = preferItemId && preferItemId > 0 ? preferItemId : 0
  linePicks.value = group.lines.map((l) => {
    const canShip = l.remaining > 0
    // 指定行发货时默认只勾该行；否则可勾选全部待发（可再取消）
    const selected = canShip && (!prefer || l.selfOrderItemId === prefer)
    return {
      ...l,
      selected,
      shipQty: canShip ? l.remaining : 1,
    }
  })
  dialogVisible.value = true
  await fillReceiverFromOrder(group.refSoId)
}

async function handleSave() {
  const selected = linePicks.value.filter((l) => l.selected && l.remaining > 0)
  if (!selected.length) {
    ElMessage.warning('请勾选要发货的商品明细')
    return
  }
  for (const line of selected) {
    if (!line.shipQty || line.shipQty < 1 || line.shipQty > line.remaining) {
      ElMessage.warning(`「${line.productName}」发货数量需在 1 ~ ${line.remaining} 之间`)
      return
    }
  }
  if (!form.value.trackingNo?.trim()) {
    ElMessage.warning('请填写物流单号')
    return
  }
  if (!form.value.carrierName?.trim() && form.value.carrierCode) {
    onCarrierChange(form.value.carrierCode, 'form')
  }
  if (!form.value.carrierName?.trim()) {
    ElMessage.warning('请选择快递公司')
    return
  }
  saving.value = true
  try {
    const created = await createSelfShipment(props.selfOrderId, {
      ...form.value,
      callback: true,
      items: selected.map((l) => ({ selfOrderItemId: l.selfOrderItemId, qty: l.shipQty })),
    })
    if (pendingPhotoUrls.value.length && created?.id) {
      await bindShipmentPhotos(created.id, pendingPhotoUrls.value)
    }

    if (!created.callbackOk) {
      ElMessage.warning('本地发货已保存，但回传订单中心失败，请重试回传')
      dialogVisible.value = false
      await loadData()
      emit('refresh')
      return
    }
    if (!created.stockDeducted && props.order.stockError) {
      ElMessage.warning(`已回传订单中心，但扣库失败：${props.order.stockError}，可重试扣库`)
    } else {
      ElMessage.success('已添加发货批次，并已回传订单中心')
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

function openUploadPhotos(row: SelfShipment) {
  photoTarget.value = row
  photoUrls.value = []
  photoVisible.value = true
}

async function handleSavePhotos() {
  if (!photoTarget.value) return
  if (!photoUrls.value.length) {
    ElMessage.warning('请先上传照片')
    return
  }
  photoSaving.value = true
  try {
    await bindShipmentPhotos(photoTarget.value.id, photoUrls.value)
    ElMessage.success('已上传物流照片')
    photoVisible.value = false
    await loadData()
  } catch (e) {
    ElMessage.error((e as Error).message || '上传失败')
  } finally {
    photoSaving.value = false
  }
}

async function openCallback(group: SalesOrderGroup) {
  if (!group.refSoId) {
    ElMessage.warning('该销售单缺少订单中心 ID，无法回传')
    return
  }
  const itemIds = new Set(group.lines.map((l) => l.selfOrderItemId))
  const withTracking = list.value.find((s) => {
    if (!s.trackingNo?.trim()) return false
    return (s.items || []).some((it) => itemIds.has(it.selfOrderItemId))
  }) || list.value.find((s) => !!s.trackingNo?.trim())

  callbackForm.value = {
    refSoId: group.refSoId,
    refOrderNo: group.refOrderNo,
    carrierCode: withTracking?.carrierCode || '',
    carrierName: withTracking?.carrierName || '',
    trackingNo: withTracking?.trackingNo || '',
    remark: '',
  }
  if (callbackForm.value.carrierCode && !callbackForm.value.carrierName) {
    onCarrierChange(callbackForm.value.carrierCode, 'callback')
  }
  callbackVisible.value = true
}

async function handleCallbackShip() {
  if (!callbackForm.value.carrierName?.trim() && callbackForm.value.carrierCode) {
    onCarrierChange(callbackForm.value.carrierCode, 'callback')
  }
  if (!callbackForm.value.carrierName?.trim()) {
    ElMessage.warning('请选择快递公司')
    return
  }
  if (!callbackForm.value.trackingNo?.trim()) {
    ElMessage.warning('请填写物流单号')
    return
  }
  callbacking.value = true
  try {
    const itemIds = new Set(
      soGroups.value.find((g) => g.refSoId === callbackForm.value.refSoId)?.lines.map((l) => l.selfOrderItemId) || [],
    )
    const matched = list.value.find((s) => {
      if (!s.trackingNo?.trim()) return false
      return (s.items || []).some((it) => itemIds.has(it.selfOrderItemId))
    })
    if (matched && !matched.callbackOk) {
      await retryCallback(props.selfOrderId, matched.id)
      ElMessage.success('已重试回传订单中心')
    } else {
      await shipOrder(callbackForm.value.refSoId, {
        expressCompany: callbackForm.value.carrierName,
        expressNo: callbackForm.value.trackingNo.trim(),
        remark: callbackForm.value.remark || `自营单 ${props.order.soNo || props.selfOrderId} 回传`,
        callback: true,
      })
      ElMessage.success('已回传订单中心')
      await handleSyncFromOrders(callbackForm.value.refSoId)
    }
    callbackVisible.value = false
    await loadData()
    emit('refresh')
  } catch (e) {
    ElMessage.error((e as Error).message || '回传失败')
  } finally {
    callbacking.value = false
  }
}

async function handleSyncFromOrders(refSoId?: number) {
  syncing.value = true
  try {
    const res = await syncSelfShipmentsFromOrders(props.selfOrderId, refSoId)
    const parts = [
      res.created ? `新建 ${res.created}` : '',
      res.updated ? `更新 ${res.updated}` : '',
      res.skipped ? `跳过 ${res.skipped}` : '',
    ].filter(Boolean)
    ElMessage.success(parts.length ? `同步完成：${parts.join('，')}` : '同步完成')
    if (res.errors?.length) {
      ElMessage.warning(res.errors.slice(0, 3).join('；'))
    }
    await loadData()
    emit('refresh')
  } catch (e) {
    ElMessage.error((e as Error).message || '同步物流失败')
  } finally {
    syncing.value = false
  }
}

async function changeStatus(row: SelfShipment, status: string) {
  try {
    await updateSelfShipmentStatus(props.selfOrderId, row.id, status)
    ElMessage.success('状态已更新')
    await loadData()
    emit('refresh')
  } catch (e) {
    ElMessage.error((e as Error).message || '更新失败')
  }
}

async function handleDelete(row: SelfShipment) {
  try {
    await ElMessageBox.confirm('确定删除此发货批次？', '确认')
  } catch {
    return
  }
  try {
    await deleteSelfShipment(props.selfOrderId, row.id)
    ElMessage.success('已删除')
    await loadData()
    emit('refresh')
  } catch (e) {
    ElMessage.error((e as Error).message || '删除失败')
  }
}

const activeGroup = computed(() =>
  soGroups.value.find((g) => g.key === activeGroupKey.value) || null,
)
</script>

<template>
  <div v-loading="loading">
    <div v-if="!readonly" class="toolbar">
      <el-button type="primary" plain :loading="syncing" @click="handleSyncFromOrders()">同步物流</el-button>
      <span class="hint">支持按商品分批发货（一商品可对应一个快递单号）；全部发出后单据为「已发货」，否则「部分发货」</span>
    </div>

    <el-table :data="soGroupRows" border stripe class="so-group-table" row-key="key">
      <el-table-column label="销售单" width="160" show-overflow-tooltip>
        <template #default="{ row }">{{ row.group.refOrderNo }}</template>
      </el-table-column>
      <el-table-column label="规格" min-width="220" show-overflow-tooltip>
        <template #default="{ row }">{{ formatSpecLabel(row.line.skuSpecs, row.line.qty) }}</template>
      </el-table-column>
      <el-table-column label="待发" width="80" align="center">
        <template #default="{ row }">
          <span :class="{ muted: row.line.remaining <= 0 }">{{ row.line.remaining }}</span>
        </template>
      </el-table-column>
      <el-table-column label="物流" min-width="160" show-overflow-tooltip>
        <template #default="{ row }">{{ lineLogisticsText(row.line.selfOrderItemId) }}</template>
      </el-table-column>
      <el-table-column label="状态" width="100" align="center">
        <template #default="{ row }">{{ row.lineStatus }}</template>
      </el-table-column>
      <el-table-column v-if="!readonly" label="操作" width="180" fixed="right">
        <template #default="{ row }">
          <el-button
            type="primary"
            link
            :disabled="row.line.remaining <= 0"
            @click="openCreateDropship(row.group, row.line.selfOrderItemId)"
          >
            发货
          </el-button>
          <el-button
            type="success"
            link
            :disabled="!row.group.refSoId"
            @click="openCallback(row.group)"
          >
            回传单号
          </el-button>
        </template>
      </el-table-column>
    </el-table>

    <h4 class="section-title">发货批次</h4>

    <el-table :data="list" border stripe>
      <el-table-column prop="shipmentNo" label="批次号" width="140" />
      <el-table-column label="销售单" width="150" show-overflow-tooltip>
        <template #default="{ row }">{{ shipmentSalesOrders(row) }}</template>
      </el-table-column>
      <el-table-column label="状态" width="90">
        <template #default="{ row }">{{ SHIPMENT_STATUS_MAP[row.status] || row.status }}</template>
      </el-table-column>
      <el-table-column prop="carrierName" label="快递" width="100" />
      <el-table-column prop="trackingNo" label="物流单号" min-width="140" />
      <el-table-column label="回传" width="80" align="center">
        <template #default="{ row }">
          <el-tag size="small" :type="row.callbackOk ? 'success' : 'warning'">
            {{ row.callbackOk ? '成功' : '待回传' }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column label="扣库" width="80" align="center">
        <template #default="{ row }">
          <el-tag size="small" :type="row.stockDeducted ? 'success' : 'info'">
            {{ row.stockDeducted ? '已扣' : '未扣' }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column label="发货照片" width="140">
        <template #default="{ row }">
          <div v-if="photosByShipment.get(row.id)?.length" class="shots">
            <el-image
              v-for="a in photosByShipment.get(row.id)"
              :key="a.id"
              :src="a.fileUrl"
              :preview-src-list="(photosByShipment.get(row.id) || []).map((x) => x.fileUrl)"
              fit="cover"
              class="shot"
              preview-teleported
            />
          </div>
          <span v-else class="muted">—</span>
        </template>
      </el-table-column>
      <el-table-column label="对应规格" min-width="220" show-overflow-tooltip>
        <template #default="{ row }">{{ shipmentSpecText(row) }}</template>
      </el-table-column>
      <el-table-column label="收件人" width="100" show-overflow-tooltip>
        <template #default="{ row }">{{ row.receiverName || '—' }}</template>
      </el-table-column>
      <el-table-column prop="expectedArrivalDate" label="预计到货" width="110" />
      <el-table-column prop="shippedAt" label="发货时间" width="150" />
      <el-table-column v-if="!readonly" label="操作" width="280" fixed="right">
        <template #default="{ row }">
          <el-button link type="primary" @click="openUploadPhotos(row)">上传照片</el-button>
          <el-button v-if="row.status === 'pending'" link type="primary" @click="changeStatus(row, 'shipped')">已发货</el-button>
          <el-button v-if="row.status === 'shipped'" link type="primary" @click="changeStatus(row, 'in_transit')">运输中</el-button>
          <el-button v-if="row.status === 'in_transit' || row.status === 'shipped'" link type="success" @click="changeStatus(row, 'delivered')">已签收</el-button>
          <el-button link type="danger" :icon="Delete" @click="handleDelete(row)">删除</el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog v-model="dialogVisible" title="按销售单发货" width="720px">
      <el-form v-loading="loadingAddr" :model="form" label-width="90px">
        <el-form-item v-if="activeGroup" label="销售单">
          <span>{{ activeGroup.refOrderNo }}</span>
        </el-form-item>
        <el-form-item label="快递公司" required>
          <el-select
            v-model="form.carrierCode"
            filterable
            allow-create
            default-first-option
            clearable
            placeholder="选择或搜索快递公司"
            style="width: 100%"
            @change="onFormCarrierChange"
          >
            <el-option
              v-for="c in EXPRESS_COMPANIES"
              :key="c.code"
              :label="c.name"
              :value="c.code"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="物流单号" required>
          <el-input v-model="form.trackingNo" placeholder="填写该销售单对应物流单号" />
        </el-form-item>
        <el-form-item label="发货照片">
          <ScanImageUpload
            v-model="pendingPhotoUrls"
            subdir="self/shipments"
            tip="本机上传"
            scan-title="手机扫码上传发货/单号照片"
          />
          <div class="hint" style="margin-top: 6px">可上传发货记录、物流面单/单号照片等</div>
        </el-form-item>
        <el-form-item label="发货明细" required>
          <el-table :data="linePicks" border size="small" max-height="280">
            <el-table-column width="48" align="center">
              <template #default="{ row }">
                <el-checkbox v-model="row.selected" :disabled="row.remaining <= 0" />
              </template>
            </el-table-column>
            <el-table-column label="图片" width="56" align="center">
              <template #default="{ row }">
                <el-image
                  v-if="row.picUrl"
                  :src="row.picUrl"
                  fit="cover"
                  style="width: 32px; height: 32px; border-radius: 4px"
                />
                <span v-else class="muted">—</span>
              </template>
            </el-table-column>
            <el-table-column label="规格" min-width="160" show-overflow-tooltip>
              <template #default="{ row }">{{ formatSpecLabel(row.skuSpecs) }}</template>
            </el-table-column>
            <el-table-column label="商家编码" width="110" show-overflow-tooltip>
              <template #default="{ row }">{{ row.skuCode || '—' }}</template>
            </el-table-column>
            <el-table-column label="可发" width="70" align="center">
              <template #default="{ row }">
                <span :class="{ muted: row.remaining <= 0 }">{{ row.remaining }}/{{ row.qty }}</span>
              </template>
            </el-table-column>
            <el-table-column label="本单数量" width="110" align="center">
              <template #default="{ row }">
                <el-input-number
                  v-model="row.shipQty"
                  :min="1"
                  :max="Math.max(1, row.remaining)"
                  :disabled="!row.selected || row.remaining <= 0"
                  size="small"
                  controls-position="right"
                  style="width: 96px"
                />
              </template>
            </el-table-column>
          </el-table>
          <div class="hint" style="margin-top: 6px">勾选本次要发的商品；不同商品可分批登记，各自填写快递单号</div>
          <div v-if="!linePicks.some((l) => l.remaining > 0)" class="hint warn">
            全部明细已关联物流，如需改绑请先删除旧发货批次
          </div>
        </el-form-item>
        <el-form-item label="预计到货">
          <el-date-picker v-model="form.expectedArrivalDate" type="date" value-format="YYYY-MM-DD" style="width: 100%" />
        </el-form-item>
        <el-form-item label="收货人">
          <el-input v-model="form.receiverName" />
        </el-form-item>
        <el-form-item label="电话">
          <el-input v-model="form.receiverPhone" />
        </el-form-item>
        <el-form-item label="收货地址">
          <el-input v-model="form.receiverAddress" type="textarea" :rows="2" />
          <div v-if="addressHint" class="hint addr-hint">{{ addressHint }}</div>
        </el-form-item>
        <el-form-item label="备注">
          <el-input v-model="form.remark" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="handleSave">发货并回传</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="callbackVisible" title="回传单号" width="480px">
      <el-form :model="callbackForm" label-width="90px">
        <el-form-item label="销售单">
          <span>{{ callbackForm.refOrderNo }}</span>
        </el-form-item>
        <el-form-item label="快递公司" required>
          <el-select
            v-model="callbackForm.carrierCode"
            filterable
            allow-create
            default-first-option
            clearable
            placeholder="选择或搜索快递公司"
            style="width: 100%"
            @change="onCallbackCarrierChange"
          >
            <el-option
              v-for="c in EXPRESS_COMPANIES"
              :key="c.code"
              :label="c.name"
              :value="c.code"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="物流单号" required>
          <el-input v-model="callbackForm.trackingNo" placeholder="运单号" />
        </el-form-item>
        <el-form-item label="备注">
          <el-input v-model="callbackForm.remark" />
        </el-form-item>
        <div class="hint">已自动带出本单物流（如有）；将写入订单中心并回传电商平台</div>
      </el-form>
      <template #footer>
        <el-button @click="callbackVisible = false">取消</el-button>
        <el-button type="primary" :loading="callbacking" @click="handleCallbackShip">确认回传</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="photoVisible" title="上传发货/物流照片" width="520px">
      <div v-if="photoTarget" class="hint" style="margin-bottom: 12px">
        批次 {{ photoTarget.shipmentNo }} · {{ photoTarget.carrierName || '—' }} {{ photoTarget.trackingNo || '' }}
      </div>
      <ScanImageUpload
        v-model="photoUrls"
        subdir="self/shipments"
        tip="本机上传"
        scan-title="手机扫码上传发货/单号照片"
      />
      <template #footer>
        <el-button @click="photoVisible = false">取消</el-button>
        <el-button type="primary" :loading="photoSaving" @click="handleSavePhotos">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.toolbar {
  margin-bottom: 12px;
  display: flex;
  align-items: center;
  gap: 12px;
}
.hint {
  color: var(--el-text-color-secondary);
  font-size: 13px;
}
.hint.warn {
  margin-top: 8px;
  color: var(--el-color-warning);
}
.addr-hint {
  margin-top: 6px;
}
.section-title {
  margin: 16px 0 10px;
  font-size: 14px;
  font-weight: 600;
}
.so-group-table {
  margin-bottom: 8px;
}
.sub {
  color: var(--el-text-color-secondary);
  font-size: 12px;
}
.muted {
  color: var(--el-text-color-placeholder);
}
.shots {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
}
.shot {
  width: 36px;
  height: 36px;
  border-radius: 4px;
}
</style>
