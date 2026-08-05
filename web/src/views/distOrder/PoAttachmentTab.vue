<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Delete, Upload } from '@element-plus/icons-vue'
import {
  fetchAttachments, createAttachment, deleteAttachment, uploadFile as uploadPoFile,
  ATTACHMENT_TYPE_MAP, type Attachment,
} from '../../api/tracking'

const props = defineProps<{ poId: number; readonly: boolean }>()

const loading = ref(false)
const uploading = ref(false)
const list = ref<Attachment[]>([])
const form = ref({ fileType: 'dist_sales_order', remark: '' })

async function loadData() {
  loading.value = true
  try {
    list.value = await fetchAttachments(props.poId)
  } catch (e) {
    ElMessage.error((e as Error).message || '加载失败')
  } finally {
    loading.value = false
  }
}

onMounted(loadData)

async function onFileChange(uploadFile: { raw?: File }) {
  const file = uploadFile.raw
  if (!file) return
  uploading.value = true
  try {
    const result = await uploadPoFile(file)
    await createAttachment(props.poId, {
      fileType: form.value.fileType,
      fileName: result.fileName,
      fileUrl: result.url,
      remark: form.value.remark,
    })
    ElMessage.success('上传成功')
    form.value.remark = ''
    await loadData()
  } catch (e) {
    ElMessage.error((e as Error).message || '上传失败')
  } finally {
    uploading.value = false
  }
}

async function handleDelete(row: Attachment) {
  try {
    await ElMessageBox.confirm('确定删除此附件？', '确认')
  } catch {
    return
  }
  try {
    await deleteAttachment(props.poId, row.id)
    ElMessage.success('已删除')
    await loadData()
  } catch (e) {
    ElMessage.error((e as Error).message || '删除失败')
  }
}
</script>

<template>
  <div v-loading="loading || uploading">
    <div v-if="!readonly" class="upload-bar">
      <el-select v-model="form.fileType" style="width: 160px">
        <el-option v-for="(label, key) in ATTACHMENT_TYPE_MAP" :key="key" :label="label" :value="key" />
      </el-select>
      <el-input v-model="form.remark" placeholder="备注（可选）" style="width: 200px" />
      <el-upload :auto-upload="false" :show-file-list="false" :on-change="onFileChange">
        <el-button type="primary" :icon="Upload">上传附件</el-button>
      </el-upload>
    </div>
    <el-table :data="list" border stripe>
      <el-table-column label="类型" width="130">
        <template #default="{ row }">{{ ATTACHMENT_TYPE_MAP[row.fileType] || row.fileType }}</template>
      </el-table-column>
      <el-table-column prop="fileName" label="文件名" min-width="180">
        <template #default="{ row }">
          <el-link :href="row.fileUrl" target="_blank" type="primary">{{ row.fileName }}</el-link>
        </template>
      </el-table-column>
      <el-table-column prop="remark" label="备注" min-width="120" />
      <el-table-column prop="createdAt" label="上传时间" width="160" />
      <el-table-column v-if="!readonly" label="操作" width="80">
        <template #default="{ row }">
          <el-button link type="danger" :icon="Delete" @click="handleDelete(row)" />
        </template>
      </el-table-column>
    </el-table>
  </div>
</template>

<style scoped>
.upload-bar {
  display: flex;
  gap: 8px;
  margin-bottom: 12px;
  flex-wrap: wrap;
  align-items: center;
}
</style>
