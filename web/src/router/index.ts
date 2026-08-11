import { createRouter, createWebHistory } from 'vue-router'
import AdminLayout from '../layouts/AdminLayout.vue'
import {redirectToPortal, ensureSession, clearToken} from '../utils/auth'

const APP_TITLE = 'SelfCore - 自营中心'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/m/photo-upload',
      name: 'MobilePhotoUpload',
      component: () => import('../views/MobilePhotoUpload.vue'),
      meta: { public: true },
    },
    {
      path: '/auth/callback',
      name: 'AuthCallback',
      component: () => import('../views/AuthCallback.vue'),
      meta: { public: true },
    },
    {
      path: '/auth/logout',
      name: 'AuthLogout',
      component: () => import('../views/AuthLogout.vue'),
      meta: { public: true },
    },
    {
      path: '/',
      component: AdminLayout,
      redirect: '/dashboard',
      children: [
        { path: 'dashboard', name: 'Dashboard', component: () => import('../views/Dashboard.vue'), meta: { title: '工作台' } },
        { path: 'self-orders', name: 'SelfOrderList', component: () => import('../views/selfOrder/SelfOrderList.vue'), meta: { title: '自营订单' } },
        { path: 'self-orders/:id(\\d+)', name: 'SelfOrderDetail', component: () => import('../views/selfOrder/SelfOrderDetail.vue'), meta: { title: '自营订单详情' } },
        {
          path: 'dist-orders/dropship',
          name: 'DropshipOrderList',
          component: () => import('../views/distOrder/DistOrderList.vue'),
          meta: { title: '分销直发', fulfillmentType: 'dropship' },
        },
        {
          path: 'dist-orders/wholesale',
          name: 'WholesaleOrderList',
          component: () => import('../views/distOrder/DistOrderList.vue'),
          meta: { title: '批发订单', fulfillmentType: 'wholesale' },
        },
        { path: 'dist-orders', name: 'DistOrderList', component: () => import('../views/distOrder/DistOrderList.vue'), meta: { title: '分销订单' } },
        { path: 'dist-orders/create', name: 'DistOrderCreate', component: () => import('../views/distOrder/DistOrderForm.vue'), meta: { title: '新建分销订单' } },
        { path: 'dist-orders/:id(\\d+)/edit', name: 'DistOrderEdit', component: () => import('../views/distOrder/DistOrderForm.vue'), meta: { title: '编辑分销订单' } },
        { path: 'dist-orders/:id(\\d+)', name: 'DistOrderDetail', component: () => import('../views/distOrder/DistOrderDetail.vue'), meta: { title: '分销订单详情' } },
        { path: 'distributors', name: 'DistributorList', component: () => import('../views/distributor/DistributorList.vue'), meta: { title: '分销商信息' } },
        { path: 'distributors/:id', name: 'DistributorDetail', component: () => import('../views/distributor/DistributorDetail.vue'), meta: { title: '分销商详情' } },
        { path: 'sku-prices', name: 'PriceList', component: () => import('../views/price/PriceList.vue'), meta: { title: 'SKU 批发价' } },
      ],
    },
  ],
})

router.beforeEach(async (to) => {
  if (to.meta.public) return true
  const ok = await ensureSession()
  if (!ok) {
    clearToken()
    redirectToPortal()
    return false
  }
  return true
})

router.afterEach(() => {
  document.title = APP_TITLE
})

export default router
