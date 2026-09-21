import { createRouter, createWebHistory } from 'vue-router'
import { isAuthenticated } from './providers/auth'

// Views are lazy-loaded so the initial bundle stays small and each page's chunk
// (and its heavy deps, e.g. the feed stats chart) is fetched only on demand.
const LoginView = () => import('./views/LoginView.vue')
const FeedLayout = () => import('./views/FeedLayout.vue')
const StatsView = () => import('./views/StatsView.vue')
const NotFoundView = () => import('./views/NotFoundView.vue')

export const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/login', name: 'login', component: LoginView },
    { path: '/stats', name: 'stats', component: StatsView },
    { path: '/', name: 'home', component: FeedLayout },
    { path: '/:pathMatch(.*)*', name: 'not-found', component: NotFoundView },
  ],
})

router.beforeEach((to) => {
  if (!isAuthenticated() && to.path !== '/login') return { path: '/login' }
  if (isAuthenticated() && to.path === '/login') return { path: '/' }
})

export default router
