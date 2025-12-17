import axios, { type AxiosInstance, type AxiosError } from 'axios'
import type {
  LoginCredentials,
  LoginResponse,
  User,
  Country,
  DocumentType,
  CreditApplication,
  ApplicationFilter,
  ApplicationListResult,
  StateTransition,
  ApiError
} from '@/types'

// Create axios instance
const api: AxiosInstance = axios.create({
  baseURL: '/api/v1',
  timeout: 30000,
  headers: {
    'Content-Type': 'application/json'
  }
})

// Request interceptor to add auth token
api.interceptors.request.use(
  (config) => {
    const token = localStorage.getItem('accessToken')
    if (token) {
      config.headers.Authorization = `Bearer ${token}`
    }
    return config
  },
  (error) => Promise.reject(error)
)

// Response interceptor for error handling
api.interceptors.response.use(
  (response) => response,
  async (error: AxiosError<ApiError>) => {
    if (error.response?.status === 401) {
      // Token expired, try to refresh
      const refreshToken = localStorage.getItem('refreshToken')
      if (refreshToken && error.config) {
        try {
          const response = await api.post<LoginResponse>('/auth/refresh', {
            refresh_token: refreshToken
          })
          
          localStorage.setItem('accessToken', response.data.access_token)
          localStorage.setItem('refreshToken', response.data.refresh_token)
          
          // Retry original request
          error.config.headers.Authorization = `Bearer ${response.data.access_token}`
          return api.request(error.config)
        } catch {
          // Refresh failed, redirect to login
          localStorage.removeItem('accessToken')
          localStorage.removeItem('refreshToken')
          localStorage.removeItem('user')
          window.location.href = '/login'
        }
      }
    }
    
    const message = error.response?.data?.message || error.message || 'An error occurred'
    return Promise.reject(new Error(message))
  }
)

// Auth Service
export const authService = {
  async login(credentials: LoginCredentials): Promise<LoginResponse> {
    const response = await api.post<LoginResponse>('/auth/login', credentials)
    return response.data
  },

  async register(data: { email: string; password: string; full_name: string }): Promise<User> {
    const response = await api.post<User>('/auth/register', data)
    return response.data
  },

  async refresh(refreshToken: string): Promise<LoginResponse> {
    const response = await api.post<LoginResponse>('/auth/refresh', {
      refresh_token: refreshToken
    })
    return response.data
  },

  async me(): Promise<User> {
    const response = await api.get<User>('/auth/me')
    return response.data
  }
}

// Country Service
export const countryService = {
  async getAll(): Promise<Country[]> {
    const response = await api.get<Country[]>('/countries')
    return response.data
  },

  async getByCode(code: string): Promise<Country> {
    const response = await api.get<Country>(`/countries/${code}`)
    return response.data
  },

  async getDocumentTypes(code: string): Promise<DocumentType[]> {
    const response = await api.get<DocumentType[]>(`/countries/${code}/document-types`)
    return response.data
  },

  async getRules(code: string): Promise<any[]> {
    const response = await api.get(`/countries/${code}/rules`)
    return response.data
  }
}

// Application Service
export const applicationService = {
  async list(filter?: ApplicationFilter): Promise<ApplicationListResult> {
    const params = new URLSearchParams()
    
    if (filter) {
      if (filter.country) params.append('country', filter.country)
      if (filter.status) params.append('status', filter.status)
      if (filter.requires_review !== undefined) params.append('requires_review', String(filter.requires_review))
      if (filter.from_date) params.append('from_date', filter.from_date)
      if (filter.to_date) params.append('to_date', filter.to_date)
      if (filter.min_amount) params.append('min_amount', String(filter.min_amount))
      if (filter.max_amount) params.append('max_amount', String(filter.max_amount))
      if (filter.search) params.append('search', filter.search)
      if (filter.page) params.append('page', String(filter.page))
      if (filter.page_size) params.append('page_size', String(filter.page_size))
      if (filter.sort_by) params.append('sort_by', filter.sort_by)
      if (filter.sort_order) params.append('sort_order', filter.sort_order)
    }

    const response = await api.get<ApplicationListResult>(`/applications?${params.toString()}`)
    return response.data
  },

  async getById(id: string): Promise<CreditApplication> {
    const response = await api.get<CreditApplication>(`/applications/${id}`)
    return response.data
  },

  async create(data: Partial<CreditApplication>): Promise<CreditApplication> {
    const response = await api.post<CreditApplication>('/applications', data)
    return response.data
  },

  async updateStatus(id: string, status: string, reason?: string): Promise<CreditApplication> {
    const response = await api.patch<CreditApplication>(`/applications/${id}/status`, {
      status,
      reason
    })
    return response.data
  },

  async getHistory(id: string): Promise<StateTransition[]> {
    const response = await api.get<StateTransition[]>(`/applications/${id}/history`)
    return response.data
  }
}

export default api

