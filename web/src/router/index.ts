import { createRouter, createWebHistory } from 'vue-router'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/login',
      name: 'login',
      component: () => import('../views/LoginView.vue'),
    },
    {
      path: '/register',
      name: 'register',
      component: () => import('../views/RegisterView.vue'),
    },
    {
      path: '/',
      component: () => import('../views/MailLayout.vue'),
      meta: { requiresAuth: true },
      children: [
        {
          path: '',
          name: 'inbox',
          component: () => import('../views/MessageList.vue'),
        },
        {
          path: 'folder/:name',
          name: 'folder',
          component: () => import('../views/MessageList.vue'),
        },
        {
          path: 'message/:id',
          name: 'message',
          component: () => import('../views/MessageView.vue'),
        },
        {
          path: 'compose',
          name: 'compose',
          component: () => import('../views/ComposeView.vue'),
        },
        {
          path: 'settings',
          name: 'settings',
          component: () => import('../views/SettingsView.vue'),
        },
      ],
    },
  ],
})

router.beforeEach((to) => {
  const token = localStorage.getItem('token')
  if (to.meta.requiresAuth && !token) {
    return { name: 'login' }
  }
})

export default router
