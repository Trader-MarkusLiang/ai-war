import { createRouter, createWebHistory } from 'vue-router'

const routes = [
  { path: '/login', name: 'Login', component: () => import('../views/Login.vue') },
  { path: '/', name: 'Dashboard', component: () => import('../views/Dashboard.vue'), meta: { auth: true } },
  { path: '/services', name: 'Services', component: () => import('../views/Services.vue'), meta: { auth: true } },
  { path: '/logs', name: 'Logs', component: () => import('../views/Logs.vue'), meta: { auth: true } },
  { path: '/deploy', name: 'Deploy', component: () => import('../views/Deploy.vue'), meta: { auth: true } },
  { path: '/diagnose', name: 'Diagnose', component: () => import('../views/Diagnose.vue'), meta: { auth: true } },
  { path: '/backup', name: 'Backup', component: () => import('../views/Backup.vue'), meta: { auth: true } },
  { path: '/config', name: 'Config', component: () => import('../views/Config.vue'), meta: { auth: true } },
  { path: '/settings', name: 'Settings', component: () => import('../views/Settings.vue'), meta: { auth: true } }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

router.beforeEach((to, from, next) => {
  const token = localStorage.getItem('token')
  if (to.meta.auth && !token) {
    next('/login')
  } else {
    next()
  }
})

export default router
