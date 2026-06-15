import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const routes = [
  {
    path: '/auth',
    name: 'auth',
    component: () => import('@/views/AuthView.vue'),
    meta: { guest: true }
  },
  {
    path: '/',
    component: () => import('@/components/layout/AppLayout.vue'),
    meta: { auth: true },
    children: [
      { path: '',      redirect: '/feed' },
      { path: 'chat',  name: 'chat',    component: () => import('@/views/ChatView.vue') },
      { path: 'contacts', name: 'contacts', component: () => import('@/views/ContactsView.vue') },
      { path: 'feed',  name: 'feed',    component: () => import('@/views/FeedView.vue') },
      { path: 'post/:id', name: 'post', component: () => import('@/views/PostDetailView.vue') },
      { path: 'ai',    name: 'ai',      component: () => import('@/views/AIView.vue') },
      { path: 'vip',   name: 'vip',     component: () => import('@/views/VIPView.vue') },
      { path: 'profile', name: 'profile', component: () => import('@/views/ProfileView.vue') },
    ]
  },
  { path: '/:pathMatch(.*)*', redirect: '/' }
]

const router = createRouter({
  history: createWebHistory(),
  routes,
})

router.beforeEach(async (to) => {
  const auth = useAuthStore()

  if (to.meta.auth && !auth.isLoggedIn) return '/auth'
  if (to.meta.guest && auth.isLoggedIn) return '/'

  if (auth.isLoggedIn && !auth.user) {
    await auth.fetchMe()
  }
})

export default router
