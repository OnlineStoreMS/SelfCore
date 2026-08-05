<script setup lang="ts">
import { getOrderCoreUrl } from '../../utils/runtimeConfig'

function openOrderCore() {
  window.open(getOrderCoreUrl(), '_blank', 'noopener,noreferrer')
}
</script>

<template>
  <div class="page">
    <h2 class="page-title">自营订单</h2>
    <p class="desc">
      承接 OrderCore 中自营履约订单（如 <code>alloc_type=self_ship</code>）的只读聚合与跟单入口。
      MVP 阶段先提供骨架；深度履约仍在订单中心 / 物流中心。
    </p>
    <el-card shadow="never" class="card">
      <el-empty description="自营订单列表待接入 OrderCore 聚合">
        <el-button type="primary" @click="openOrderCore">打开订单中心</el-button>
      </el-empty>
    </el-card>
  </div>
</template>

<style scoped>
.page { width: 100%; }
.page-title { margin: 0 0 8px; font-size: 22px; }
.desc { color: #606266; margin: 0 0 20px; line-height: 1.6; max-width: 720px; }
.card { border-radius: 8px; }
code {
  padding: 1px 6px;
  background: #f5f7fa;
  border-radius: 4px;
  font-size: 12px;
}
</style>
