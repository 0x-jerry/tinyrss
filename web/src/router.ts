import { createRouter, createWebHistory } from 'vue-router'
import { isAuthenticated } from './providers/auth'
import LoginView from './views/LoginView.vue'
import FeedLayout from './views/FeedLayout.vue'
import StatsView from './views/StatsView.vue'

export const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/login', name: 'login', component: LoginView },
    { path: '/stats', name: 'stats', component: StatsView },
    { path: '/', name: 'home', component: FeedLayout },
  ],
})

router.beforeEach((to) => {
  if (!isAuthenticated() && to.path !== '/login') return { path: '/login' }
  if (isAuthenticated() && to.path === '/login') return { path: '/' }
})

export default router
