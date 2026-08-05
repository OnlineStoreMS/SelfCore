<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import {
  cacheSkuSearchItem,
  fetchProductSkus,
  formatSkuSpecLabel,
  searchProducts,
  searchProductSkus,
  type ProductBrief,
  type ProductSkuItem,
  type ProductSkuSearchItem,
  type ProductSkusPayload,
} from '../api/productSku'

const model = defineModel<number | undefined>({ default: undefined })

withDefaults(
  defineProps<{
    disabled?: boolean
  }>(),
  {
    disabled: false,
  },
)

const emit = defineEmits<{
  select: [item: ProductSkuSearchItem | undefined]
}>()

const productLoading = ref(false)
const skuLoading = ref(false)
const productOptions = ref<ProductBrief[]>([])
const selectedProductId = ref<number | undefined>()
const productDetail = ref<ProductSkusPayload | null>(null)
const skus = ref<ProductSkuItem[]>([])
let productDebounce: ReturnType<typeof setTimeout> | undefined

const selectedSku = computed(() => skus.value.find((s) => s.id === model.value))

const selectedProduct = computed(() =>
  productOptions.value.find((p) => p.id === selectedProductId.value),
)

function productLabel(p: ProductBrief) {
  const bits = [p.name]
  if (p.materialCode) bits.push(p.materialCode)
  return bits.join(' · ')
}

function skuLabel(sku: ProductSkuItem) {
  const spec = formatSkuSpecLabel(sku.specs) || '无规格'
  const code = sku.skuCode?.trim() || '未编码'
  return `${code} · ${spec}`
}

function toSearchItem(
  productId: number,
  productName: string,
  productPic: string | undefined,
  materialCode: string | undefined,
  sku: ProductSkuItem,
): ProductSkuSearchItem {
  return {
    productId,
    productName,
    materialCode,
    productPic,
    skuId: sku.id,
    skuCode: sku.skuCode,
    specs: sku.specs || {},
    specLabel: formatSkuSpecLabel(sku.specs),
    price: sku.price,
    stock: sku.stock,
    pic: sku.pic || productPic,
  }
}

function keepSelectedInOptions(list: ProductBrief[]) {
  const selected = selectedProduct.value
  if (!selected) return list
  if (list.some((p) => p.id === selected.id)) return list
  return [selected, ...list]
}

async function remoteSearchProducts(keyword: string) {
  const q = keyword.trim()
  if (!q) {
    // 未输入关键字时不拉全量，只保留已选商品
    productOptions.value = selectedProduct.value ? [selectedProduct.value] : []
    return
  }
  productLoading.value = true
  try {
    const data = await searchProducts({
      keyword: q,
      page: 1,
      pageSize: 20,
    })
    productOptions.value = keepSelectedInOptions(data.list)
  } catch (e) {
    ElMessage.error((e as Error).message || '商品搜索失败')
  } finally {
    productLoading.value = false
  }
}

function onProductSearch(keyword: string) {
  clearTimeout(productDebounce)
  productDebounce = setTimeout(() => {
    void remoteSearchProducts(keyword)
  }, 280)
}

async function loadSkus(productId?: number) {
  if (!productId) {
    productDetail.value = null
    skus.value = []
    return
  }
  skuLoading.value = true
  try {
    const data = await fetchProductSkus(productId)
    productDetail.value = data
    skus.value = data.skus || []
    const brief: ProductBrief = {
      id: data.id,
      name: data.name,
      materialCode: data.materialCode,
      pic: data.pic,
      skuCount: data.skuCount,
    }
    if (!productOptions.value.some((p) => p.id === brief.id)) {
      productOptions.value = [brief, ...productOptions.value]
    } else {
      productOptions.value = productOptions.value.map((p) => (p.id === brief.id ? { ...p, ...brief } : p))
    }
  } catch (e) {
    ElMessage.error((e as Error).message || '加载 SKU 失败')
    skus.value = []
  } finally {
    skuLoading.value = false
  }
}

async function onProductChange(productId: number | undefined) {
  selectedProductId.value = productId
  model.value = undefined
  emit('select', undefined)
  await loadSkus(productId)
}

function onSkuChange(skuId: number | undefined) {
  if (!skuId || !productDetail.value) {
    emit('select', undefined)
    return
  }
  const sku = skus.value.find((s) => s.id === skuId)
  if (!sku) {
    emit('select', undefined)
    return
  }
  const item = toSearchItem(
    productDetail.value.id,
    productDetail.value.name,
    selectedProduct.value?.pic || productDetail.value.pic,
    productDetail.value.materialCode,
    sku,
  )
  cacheSkuSearchItem(item)
  emit('select', item)
}

async function hydrateFromSkuId(skuId?: number) {
  if (!skuId) {
    if (!selectedProductId.value) {
      productDetail.value = null
      skus.value = []
    }
    emit('select', undefined)
    return
  }
  if (skus.value.some((s) => s.id === skuId) && selectedProductId.value) {
    onSkuChange(skuId)
    return
  }
  skuLoading.value = true
  try {
    const data = await searchProductSkus({ keyword: String(skuId), page: 1, pageSize: 10 })
    const hit = data.list.find((item) => item.skuId === skuId)
    if (!hit) {
      emit('select', undefined)
      return
    }
    cacheSkuSearchItem(hit)
    selectedProductId.value = hit.productId
    productOptions.value = [
      {
        id: hit.productId,
        name: hit.productName,
        materialCode: hit.materialCode,
        productSn: hit.productSn,
        pic: hit.productPic,
      },
    ]
    await loadSkus(hit.productId)
    if (model.value !== skuId) {
      model.value = skuId
    }
    emit('select', hit)
  } catch {
    emit('select', undefined)
  } finally {
    skuLoading.value = false
  }
}

onMounted(() => {
  void hydrateFromSkuId(model.value)
})

watch(
  () => model.value,
  (id, prev) => {
    if (id === prev) return
    void hydrateFromSkuId(id)
  },
)
</script>

<template>
  <div class="product-sku-picker" :class="{ disabled }">
    <div class="picker-row">
      <div class="picker-label">商品</div>
      <el-select
        :model-value="selectedProductId"
        filterable
        remote
        reserve-keyword
        clearable
        :remote-method="onProductSearch"
        :loading="productLoading"
        :disabled="disabled"
        placeholder="输入关键字搜索商品"
        no-data-text="请输入商品名称 / 货号搜索"
        popper-class="product-sku-picker-dropdown"
        style="width: 100%"
        @update:model-value="onProductChange"
      >
        <el-option
          v-for="item in productOptions"
          :key="item.id"
          :label="productLabel(item)"
          :value="item.id"
        >
          <div class="option-row">
            <el-image :src="item.pic" fit="cover" class="option-pic">
              <template #error>
                <div class="option-pic placeholder">无图</div>
              </template>
            </el-image>
            <div class="option-text">
              <div class="option-main">{{ item.name }}</div>
              <div class="option-sub">
                <span v-if="item.materialCode">{{ item.materialCode }}</span>
                <span v-if="item.skuCount != null">SKU {{ item.skuCount }}</span>
                <span>#{{ item.id }}</span>
              </div>
            </div>
          </div>
        </el-option>
      </el-select>
    </div>

    <div class="picker-row">
      <div class="picker-label">SKU</div>
      <el-select
        v-model="model"
        filterable
        clearable
        :loading="skuLoading"
        :disabled="disabled || !selectedProductId"
        :placeholder="selectedProductId ? '选择规格 SKU' : '请先选择商品'"
        popper-class="product-sku-picker-dropdown"
        style="width: 100%"
        @change="onSkuChange"
      >
        <el-option
          v-for="sku in skus"
          :key="sku.id"
          :label="skuLabel(sku)"
          :value="sku.id"
        >
          <div class="option-row">
            <el-image :src="sku.pic || selectedProduct?.pic" fit="cover" class="option-pic">
              <template #error>
                <div class="option-pic placeholder">无图</div>
              </template>
            </el-image>
            <div class="option-text">
              <div class="option-main">
                <span class="code">{{ sku.skuCode || '未编码' }}</span>
                <span class="spec">{{ formatSkuSpecLabel(sku.specs) || '无规格' }}</span>
              </div>
              <div class="option-sub">
                <span>¥{{ Number(sku.price || 0).toFixed(2) }}</span>
                <span>库存 {{ sku.stock }}</span>
              </div>
            </div>
          </div>
        </el-option>
      </el-select>
    </div>

    <div v-if="selectedSku && (selectedProduct || productDetail)" class="preview">
      <el-image
        :src="selectedSku.pic || selectedProduct?.pic || productDetail?.pic"
        fit="cover"
        class="preview-pic"
      >
        <template #error>
          <div class="preview-pic placeholder">无图</div>
        </template>
      </el-image>
      <div class="preview-text">
        <div class="name">{{ productDetail?.name || selectedProduct?.name }}</div>
        <div class="meta">
          <span>{{ selectedSku.skuCode || '未编码' }}</span>
          <span>{{ formatSkuSpecLabel(selectedSku.specs) || '无规格' }}</span>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.product-sku-picker {
  width: 100%;
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.picker-row {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.picker-label {
  font-size: 12px;
  color: #909399;
}
.option-row {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 4px 0;
  min-height: 52px;
}
.option-pic,
.preview-pic {
  width: 40px;
  height: 40px;
  border-radius: 6px;
  flex-shrink: 0;
  background: #f5f7fa;
}
.preview-pic {
  width: 56px;
  height: 56px;
}
.option-pic.placeholder,
.preview-pic.placeholder {
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 11px;
  color: #c0c4cc;
}
.option-text,
.preview-text {
  min-width: 0;
  flex: 1;
}
.option-main {
  display: flex;
  gap: 8px;
  align-items: baseline;
  flex-wrap: wrap;
  font-weight: 500;
}
.option-main .code {
  color: var(--el-text-color-primary);
}
.option-main .spec {
  font-size: 12px;
  color: #409eff;
  font-weight: 400;
}
.option-sub,
.preview-text .meta {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
  font-size: 12px;
  color: #909399;
  margin-top: 2px;
}
.preview {
  display: flex;
  gap: 10px;
  align-items: center;
  padding: 10px;
  background: #f5f7fa;
  border-radius: 8px;
}
.preview-text .name {
  font-weight: 500;
  font-size: 14px;
}
</style>

<style>
.product-sku-picker-dropdown.el-select-dropdown {
  min-width: 420px !important;
}
.product-sku-picker-dropdown .el-select-dropdown__item {
  height: auto;
  padding: 6px 12px;
  line-height: 1.35;
}
</style>
