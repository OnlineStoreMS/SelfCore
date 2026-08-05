<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import {
  formatSkuOptionLabel,
  searchProductSkus,
  skuDisplayPic,
  type ProductSkuSearchItem,
} from '../api/productSku'

const model = defineModel<number | undefined>({ default: undefined })

withDefaults(
  defineProps<{
    placeholder?: string
    disabled?: boolean
    clearable?: boolean
    showPreview?: boolean
  }>(),
  {
    placeholder: '输入编码 / 规格 / 商品名搜索',
    disabled: false,
    clearable: true,
    showPreview: true,
  },
)

const emit = defineEmits<{
  select: [item: ProductSkuSearchItem | undefined]
}>()

const loading = ref(false)
const options = ref<ProductSkuSearchItem[]>([])
let debounceTimer: ReturnType<typeof setTimeout> | undefined

const selected = computed(() =>
  model.value ? options.value.find((item) => item.skuId === model.value) : undefined,
)

function mergeOptions(items: ProductSkuSearchItem[]) {
  const map = new Map<number, ProductSkuSearchItem>()
  for (const item of options.value) {
    map.set(item.skuId, item)
  }
  for (const item of items) {
    map.set(item.skuId, item)
  }
  options.value = Array.from(map.values())
}

async function remoteSearch(keyword: string) {
  const q = keyword.trim()
  if (!q) return
  loading.value = true
  try {
    const data = await searchProductSkus({ keyword: q, page: 1, pageSize: 20 })
    mergeOptions(data.list)
  } catch (e) {
    ElMessage.error((e as Error).message || 'SKU 搜索失败')
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

async function loadSelected(id?: number) {
  if (!id) {
    // 无 skuId 时不向上通知，避免编辑页冲掉 OMS 已带入的商品信息
    return
  }
  const existing = options.value.find((item) => item.skuId === id)
  if (existing) {
    emit('select', existing)
    return
  }
  loading.value = true
  try {
    const data = await searchProductSkus({ keyword: String(id), page: 1, pageSize: 10 })
    const hit = data.list.find((item) => item.skuId === id) || data.list[0]
    if (hit && hit.skuId === id) {
      mergeOptions([hit])
      emit('select', hit)
    }
  } catch {
    // ignore hydrate failure
  } finally {
    loading.value = false
  }
}

function onChange(value: number | undefined) {
  if (!value) {
    emit('select', undefined)
    return
  }
  const hit = options.value.find((item) => item.skuId === value)
  emit('select', hit)
}

onMounted(() => {
  void loadSelected(model.value)
})

watch(
  () => model.value,
  (id) => {
    void loadSelected(id)
  },
)
</script>

<template>
  <div class="sku-search">
    <el-select
      v-model="model"
      filterable
      remote
      reserve-keyword
      :remote-method="onSearch"
      :loading="loading"
      :placeholder="placeholder"
      :disabled="disabled"
      :clearable="clearable"
      popper-class="sku-search-dropdown"
      style="width: 100%"
      @change="onChange"
    >
      <el-option
        v-for="item in options"
        :key="item.skuId"
        :label="formatSkuOptionLabel(item)"
        :value="item.skuId"
      >
        <div class="sku-option">
          <el-image
            :src="skuDisplayPic(item)"
            fit="cover"
            class="sku-option-pic"
          >
            <template #error>
              <div class="sku-option-pic placeholder">无图</div>
            </template>
          </el-image>
          <div class="sku-option-text">
            <div class="sku-option-main">
              <span class="code">{{ item.skuCode || '未编码' }}</span>
              <span class="spec">{{ item.specLabel || '无规格' }}</span>
            </div>
            <div class="sku-option-sub">{{ item.productName }}</div>
          </div>
        </div>
      </el-option>
    </el-select>

    <div v-if="showPreview && selected" class="sku-preview">
      <el-image :src="skuDisplayPic(selected)" fit="cover" class="sku-preview-pic">
        <template #error>
          <div class="sku-preview-pic placeholder">无图</div>
        </template>
      </el-image>
      <div class="sku-preview-text">
        <div class="name">{{ selected.productName }}</div>
        <div class="meta">
          <span>{{ selected.skuCode || '未编码' }}</span>
          <span v-if="selected.specLabel">{{ selected.specLabel }}</span>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.sku-search {
  width: 100%;
}
.sku-option {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 4px 0;
  min-height: 52px;
}
.sku-option-pic {
  width: 40px;
  height: 40px;
  border-radius: 6px;
  flex-shrink: 0;
  background: #f5f7fa;
}
.sku-option-pic.placeholder,
.sku-preview-pic.placeholder {
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 11px;
  color: #c0c4cc;
}
.sku-option-text {
  min-width: 0;
  flex: 1;
}
.sku-option-main {
  display: flex;
  gap: 8px;
  align-items: baseline;
  flex-wrap: wrap;
}
.sku-option-main .code {
  font-weight: 600;
  color: var(--el-text-color-primary);
}
.sku-option-main .spec {
  font-size: 12px;
  color: #409eff;
}
.sku-option-sub {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.sku-preview {
  margin-top: 10px;
  display: flex;
  gap: 10px;
  align-items: center;
  padding: 10px;
  background: #f5f7fa;
  border-radius: 8px;
}
.sku-preview-pic {
  width: 56px;
  height: 56px;
  border-radius: 6px;
  flex-shrink: 0;
  background: #fff;
}
.sku-preview-text {
  min-width: 0;
}
.sku-preview-text .name {
  font-weight: 500;
  font-size: 14px;
  margin-bottom: 4px;
}
.sku-preview-text .meta {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  font-size: 12px;
  color: #606266;
}
</style>

<style>
.sku-search-dropdown.el-select-dropdown {
  min-width: 420px !important;
}
.sku-search-dropdown .el-select-dropdown__item {
  height: auto;
  padding: 6px 12px;
  line-height: 1.35;
}
</style>
