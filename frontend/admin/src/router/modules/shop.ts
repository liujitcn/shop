import { RouteRecordRaw } from 'vue-router'

const shopRoutes: RouteRecordRaw = {
  path: '/shop',
  component: () => import('@/layout/index.vue'),
  redirect: '/shop/service',
  name: 'Shop',
  meta: {
    title: '商城管理',
    icon: 'shop',
    roles: ['admin']
  },
  children: [
    {
      path: 'service',
      component: () => import('@/views/shop/service/index.vue'),
      name: 'ShopService',
      meta: {
        title: '商城服务',
        icon: 'service',
        roles: ['admin']
      }
    }
  ]
}

export default shopRoutes 