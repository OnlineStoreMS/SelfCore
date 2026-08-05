<script setup lang="ts">
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import {
  HomeFilled, OfficeBuilding, ShoppingCart, Box,
} from '@element-plus/icons-vue'

const route = useRoute()
const router = useRouter()
const collapsed = defineModel<boolean>('collapsed', { default: false })

const openMenus = computed(() => {
  const p = route.path
  if (p.startsWith('/distributors') || p.startsWith('/sku-prices')) return ['distributor']
  if (p.startsWith('/dist-orders')) return ['dist-orders']
  return []
})

const logoText = computed(() => (collapsed.value ? 'SC' : '自营中心'))

const activeMenu = computed(() => {
  if (route.path.startsWith('/dist-orders/dropship')) return '/dist-orders/dropship'
  if (route.path.startsWith('/dist-orders/wholesale')) return '/dist-orders/wholesale'
  if (route.path === '/dist-orders') return '/dist-orders'
  if (route.path.startsWith('/dist-orders')) return '/dist-orders'
  if (route.path.startsWith('/distributors')) return '/distributors'
  return route.path
})

function navigate(path: string) {
  router.push(path)
}
</script>

<template>
  <aside class="sidebar" :class="{ collapsed }">
    <div class="logo">{{ logoText }}</div>
    <el-menu
      :key="activeMenu"
      :default-active="activeMenu"
      :default-openeds="openMenus"
      :collapse="collapsed"
      background-color="#001529"
      text-color="#ffffffa6"
      active-text-color="#fff"
    >
      <el-menu-item index="/dashboard" @click="navigate('/dashboard')">
        <el-icon><HomeFilled /></el-icon>
        <span>工作台</span>
      </el-menu-item>

      <el-menu-item index="/self-orders" @click="navigate('/self-orders')">
        <el-icon><Box /></el-icon>
        <span>自营订单</span>
      </el-menu-item>

      <el-sub-menu index="dist-orders">
        <template #title>
          <el-icon><ShoppingCart /></el-icon>
          <span>分销订单</span>
        </template>
        <el-menu-item index="/dist-orders" @click="navigate('/dist-orders')">全部订单</el-menu-item>
        <el-menu-item index="/dist-orders/dropship" @click="navigate('/dist-orders/dropship')">
          代发订单
        </el-menu-item>
        <el-menu-item index="/dist-orders/wholesale" @click="navigate('/dist-orders/wholesale')">
          批发订单
        </el-menu-item>
      </el-sub-menu>

      <el-sub-menu index="distributor">
        <template #title>
          <el-icon><OfficeBuilding /></el-icon>
          <span>分销商</span>
        </template>
        <el-menu-item index="/distributors" @click="navigate('/distributors')">分销商信息</el-menu-item>
        <el-menu-item index="/sku-prices" @click="navigate('/sku-prices')">SKU 批发价</el-menu-item>
      </el-sub-menu>
    </el-menu>
  </aside>
</template>

<style scoped>
.sidebar {
  width: 220px;
  background: #001529;
  transition: width 0.2s;
  flex-shrink: 0;
  overflow: auto;
}
.sidebar.collapsed {
  width: 64px;
}
.logo {
  height: 56px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
  font-weight: 600;
  font-size: 16px;
  border-bottom: 1px solid #ffffff14;
}
.sidebar :deep(.el-menu) {
  border-right: none;
}
</style>
