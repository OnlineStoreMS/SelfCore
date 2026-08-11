<script setup lang="ts">
import { computed, onUnmounted, ref } from 'vue'
import { ElMessage, type UploadRequestOptions } from 'element-plus'
import { Iphone, Plus, Delete } from '@element-plus/icons-vue'
import QRCode from 'qrcode'
import {
  createPhotoUploadSession,
  getPhotoUploadSession,
  uploadImage,
} from '../api/upload'

const props = withDefaults(
  defineProps<{
    modelValue?: string[]
    subdir?: string
    tip?: string
    scanTitle?: string
    max?: number
  }>(),
  {
    modelValue: () => [],
    subdir: 'do',
    tip: '本机上传',
    scanTitle: '手机扫码上传',
    max: 9,
  },
)

const emit = defineEmits<{
  (e: 'update:modelValue', v: string[]): void
}>()

const urls = computed({
  get: () => props.modelValue || [],
  set: (v) => emit('update:modelValue', v),
})

const uploading = ref(false)
const scanVisible = ref(false)
const scanLoading = ref(false)
const qrDataUrl = ref('')
const scanToken = ref('')
const scanStatus = ref<'idle' | 'waiting' | 'done' | 'expired'>('idle')
let pollTimer: ReturnType<typeof setInterval> | null = null

function stopPoll() {
  if (pollTimer) {
    clearInterval(pollTimer)
    pollTimer = null
  }
}

function addUrl(url: string) {
  if (!url || urls.value.includes(url)) return
  if (urls.value.length >= props.max) {
    ElMessage.warning(`最多上传 ${props.max} 张`)
    return
  }
  urls.value = [...urls.value, url]
}

function removeAt(idx: number) {
  urls.value = urls.value.filter((_, i) => i !== idx)
}

async function doUpload(opt: UploadRequestOptions) {
  if (urls.value.length >= props.max) {
    ElMessage.warning(`最多上传 ${props.max} 张`)
    return
  }
  uploading.value = true
  try {
    const file = opt.file as File
    const res = await uploadImage(file, props.subdir)
    addUrl(res.url)
    ElMessage.success('已上传')
  } catch (e) {
    ElMessage.error((e as Error).message || '上传失败')
  } finally {
    uploading.value = false
  }
}

async function openScan() {
  if (urls.value.length >= props.max) {
    ElMessage.warning(`最多上传 ${props.max} 张`)
    return
  }
  scanVisible.value = true
  scanLoading.value = true
  scanStatus.value = 'idle'
  qrDataUrl.value = ''
  scanToken.value = ''
  stopPoll()
  try {
    const session = await createPhotoUploadSession(props.subdir)
    scanToken.value = session.token
    const pageUrl = `${window.location.origin}${import.meta.env.BASE_URL || '/'}m/photo-upload?token=${encodeURIComponent(session.token)}`
    qrDataUrl.value = await QRCode.toDataURL(pageUrl, {
      width: 220,
      margin: 2,
      errorCorrectionLevel: 'M',
    })
    scanStatus.value = 'waiting'
    pollTimer = setInterval(async () => {
      try {
        const s = await getPhotoUploadSession(scanToken.value)
        if (s.status === 'done' && s.url) {
          addUrl(s.url)
          scanStatus.value = 'done'
          stopPoll()
          ElMessage.success('手机照片已上传')
          setTimeout(() => {
            scanVisible.value = false
          }, 500)
        }
      } catch {
        scanStatus.value = 'expired'
        stopPoll()
      }
    }, 2000)
  } catch (e) {
    ElMessage.error((e as Error).message || '创建扫码会话失败')
    scanVisible.value = false
  } finally {
    scanLoading.value = false
  }
}

function closeScan() {
  scanVisible.value = false
  stopPoll()
}

onUnmounted(stopPoll)
</script>

<template>
  <div class="scan-upload">
    <div class="thumbs">
      <div v-for="(u, idx) in urls" :key="u" class="thumb">
        <el-image :src="u" fit="cover" :preview-src-list="urls" :initial-index="idx" preview-teleported />
        <button type="button" class="rm" title="移除" @click="removeAt(idx)">
          <el-icon :size="12"><Delete /></el-icon>
        </button>
      </div>
      <el-upload
        v-if="urls.length < max"
        :show-file-list="false"
        accept="image/*"
        :disabled="uploading"
        :http-request="doUpload"
      >
        <div class="thumb placeholder" v-loading="uploading">
          <el-icon><Plus /></el-icon>
          <span>{{ tip }}</span>
        </div>
      </el-upload>
    </div>
    <div class="actions">
      <el-button type="primary" plain :icon="Iphone" :disabled="urls.length >= max" @click="openScan">
        手机扫码上传
      </el-button>
      <span class="hint">也可本机选图；扫码后电脑端自动回填</span>
    </div>
  </div>

  <el-dialog
    v-model="scanVisible"
    :title="scanTitle"
    width="360px"
    append-to-body
    destroy-on-close
    @closed="closeScan"
  >
    <div class="scan-body" v-loading="scanLoading">
      <img v-if="qrDataUrl" :src="qrDataUrl" alt="扫码上传" class="qr" />
      <p v-if="scanStatus === 'waiting'" class="scan-hint">请用手机扫描二维码，拍照或从相册选择</p>
      <p v-else-if="scanStatus === 'done'" class="scan-hint ok">上传成功</p>
      <p v-else-if="scanStatus === 'expired'" class="scan-hint err">会话已过期，请关闭后重试</p>
      <p v-else class="scan-hint">正在生成二维码…</p>
    </div>
    <template #footer>
      <el-button @click="closeScan">关闭</el-button>
      <el-button type="primary" :disabled="scanLoading" @click="openScan">刷新二维码</el-button>
    </template>
  </el-dialog>
</template>

<style scoped>
.scan-upload { width: 100%; }
.thumbs { display: flex; flex-wrap: wrap; gap: 8px; }
.thumb {
  position: relative;
  width: 88px;
  height: 88px;
  border-radius: 8px;
  border: 1px dashed var(--el-border-color);
  overflow: hidden;
  background: #fafafa;
}
.thumb :deep(.el-image) { width: 100%; height: 100%; }
.thumb.placeholder {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 4px;
  color: #909399;
  font-size: 12px;
  cursor: pointer;
}
.rm {
  position: absolute;
  top: 2px;
  right: 2px;
  width: 20px;
  height: 20px;
  border: none;
  border-radius: 50%;
  background: rgba(0, 0, 0, 0.55);
  color: #fff;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  padding: 0;
}
.actions {
  margin-top: 10px;
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
}
.hint { color: #909399; font-size: 12px; }
.scan-body {
  display: flex;
  flex-direction: column;
  align-items: center;
  min-height: 260px;
  justify-content: center;
}
.qr { width: 220px; height: 220px; border: 1px solid #ebeef5; border-radius: 8px; }
.scan-hint { margin: 12px 0 0; font-size: 13px; color: #606266; text-align: center; line-height: 1.5; }
.scan-hint.ok { color: #67c23a; }
.scan-hint.err { color: #f56c6c; }
</style>
