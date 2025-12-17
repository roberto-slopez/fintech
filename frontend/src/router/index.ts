import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    {
      path: '/login',
      name: 'login',
      component: () => import('@/views/LoginView.vue'),
      meta: { requiresAuth: false }
    },
    {
      path: '/',
      component: () => import('@/layouts/MainLayout.vue'),
      meta: { requiresAuth: true },
      children: [
        {
          path: '',
          name: 'dashboard',
          component: () => import('@/views/DashboardView.vue')
        },
        {
          path: 'applications',
          name: 'applications',
          component: () => import('@/views/ApplicationsView.vue')
        },
        {
          path: 'applications/new',
          name: 'new-application',
          component: () => import('@/views/NewApplicationView.vue')
        },
        {
          path: 'applications/:id',
          name: 'application-detail',
          component: () => import('@/views/ApplicationDetailView.vue'),
          props: true
        },
        {
          path: 'countries',
          name: 'countries',
          component: () => import('@/views/CountriesView.vue')
        }
      ]
    },
    {
      path: '/:pathMatch(.*)*',
      name: 'not-found',
      component: () => import('@/views/NotFoundView.vue')
    }
  ]
})

// Navigation guard
router.beforeEach(async (to, from, next) => {
  const authStore = useAuthStore()
  
  // Initialize auth if not done
  if (!authStore.initialized) {
    await authStore.initializeAuth()
  }
  
  const requiresAuth = to.meta.requiresAuth !== false
  
  if (requiresAuth && !authStore.isAuthenticated) {
    next({ name: 'login', query: { redirect: to.fullPath } })
  } else if (to.name === 'login' && authStore.isAuthenticated) {
    next({ name: 'dashboard' })
  } else {
    next()
  }
})

export default router

