import { defineStore } from 'pinia'
import { ref } from 'vue'
import api from '../api/client'
import type { User, LoginRequest, RegisterRequest } from '../types'

export const useAuthStore = defineStore('auth', () => {
  const user = ref<User | null>(null)
  const token = ref<string | null>(localStorage.getItem('token'))

  function setAuth(t: string, u: User) {
    token.value = t
    user.value = u
    localStorage.setItem('token', t)
  }

  function logout() {
    token.value = null
    user.value = null
    localStorage.removeItem('token')
  }

  async function login(req: LoginRequest) {
    const { data } = await api.post('/auth/login', req)
    setAuth(data.token, data.user)
  }

  async function register(req: RegisterRequest) {
    const { data } = await api.post('/auth/register', req)
    setAuth(data.token, data.user)
  }

  const isAuthenticated = () => !!token.value

  return { user, token, login, register, logout, isAuthenticated }
})
