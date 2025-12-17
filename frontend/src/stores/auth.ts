import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { authService } from '@/services/api'
import type { User, LoginCredentials, LoginResponse } from '@/types'

export const useAuthStore = defineStore('auth', () => {
  const user = ref<User | null>(null)
  const accessToken = ref<string | null>(null)
  const refreshToken = ref<string | null>(null)
  const initialized = ref(false)
  const loading = ref(false)

  const isAuthenticated = computed(() => !!accessToken.value && !!user.value)

  async function login(credentials: LoginCredentials): Promise<void> {
    loading.value = true
    try {
      const response = await authService.login(credentials)
      setAuth(response)
    } finally {
      loading.value = false
    }
  }

  async function register(data: { email: string; password: string; full_name: string }): Promise<void> {
    loading.value = true
    try {
      await authService.register(data)
    } finally {
      loading.value = false
    }
  }

  function setAuth(response: LoginResponse) {
    user.value = response.user
    accessToken.value = response.access_token
    refreshToken.value = response.refresh_token
    
    // Persist to localStorage
    localStorage.setItem('accessToken', response.access_token)
    localStorage.setItem('refreshToken', response.refresh_token)
    localStorage.setItem('user', JSON.stringify(response.user))
  }

  function logout() {
    user.value = null
    accessToken.value = null
    refreshToken.value = null
    
    localStorage.removeItem('accessToken')
    localStorage.removeItem('refreshToken')
    localStorage.removeItem('user')
  }

  async function initializeAuth() {
    const storedToken = localStorage.getItem('accessToken')
    const storedRefreshToken = localStorage.getItem('refreshToken')
    const storedUser = localStorage.getItem('user')

    if (storedToken && storedUser) {
      accessToken.value = storedToken
      refreshToken.value = storedRefreshToken
      user.value = JSON.parse(storedUser)
      
      // Verify token is still valid
      try {
        const userData = await authService.me()
        user.value = userData
      } catch (error) {
        // Token expired, try to refresh
        if (storedRefreshToken) {
          try {
            const response = await authService.refresh(storedRefreshToken)
            setAuth(response)
          } catch {
            logout()
          }
        } else {
          logout()
        }
      }
    }
    
    initialized.value = true
  }

  async function refreshTokenIfNeeded() {
    if (refreshToken.value) {
      try {
        const response = await authService.refresh(refreshToken.value)
        setAuth(response)
      } catch {
        logout()
      }
    }
  }

  return {
    user,
    accessToken,
    refreshToken,
    initialized,
    loading,
    isAuthenticated,
    login,
    register,
    logout,
    initializeAuth,
    refreshTokenIfNeeded
  }
})

