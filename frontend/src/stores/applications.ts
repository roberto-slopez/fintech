import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { applicationService } from '@/services/api'
import type { CreditApplication, ApplicationFilter, ApplicationListResult } from '@/types'

export const useApplicationsStore = defineStore('applications', () => {
  const applications = ref<CreditApplication[]>([])
  const currentApplication = ref<CreditApplication | null>(null)
  const pagination = ref({
    total: 0,
    page: 1,
    pageSize: 20,
    totalPages: 0
  })
  const filters = ref<ApplicationFilter>({})
  const loading = ref(false)
  const error = ref<string | null>(null)

  // Real-time updates counter for visual feedback
  const realtimeUpdates = ref(0)

  const filteredApplications = computed(() => applications.value)

  async function fetchApplications(filter?: ApplicationFilter) {
    loading.value = true
    error.value = null
    
    try {
      const result = await applicationService.list({ ...filters.value, ...filter })
      applications.value = result.applications
      pagination.value = {
        total: result.total,
        page: result.page,
        pageSize: result.page_size,
        totalPages: result.total_pages
      }
    } catch (e: any) {
      error.value = e.message || 'Error loading applications'
      throw e
    } finally {
      loading.value = false
    }
  }

  async function fetchApplication(id: string) {
    loading.value = true
    error.value = null
    
    try {
      currentApplication.value = await applicationService.getById(id)
    } catch (e: any) {
      error.value = e.message || 'Error loading application'
      throw e
    } finally {
      loading.value = false
    }
  }

  async function createApplication(data: Partial<CreditApplication>) {
    loading.value = true
    error.value = null
    
    try {
      const newApp = await applicationService.create(data)
      applications.value.unshift(newApp)
      return newApp
    } catch (e: any) {
      error.value = e.message || 'Error creating application'
      throw e
    } finally {
      loading.value = false
    }
  }

  async function updateStatus(id: string, status: string, reason?: string) {
    loading.value = true
    error.value = null
    
    try {
      const updated = await applicationService.updateStatus(id, status, reason)
      
      // Update in list
      const index = applications.value.findIndex(a => a.id === id)
      if (index !== -1) {
        applications.value[index] = updated
      }
      
      // Update current if viewing
      if (currentApplication.value?.id === id) {
        currentApplication.value = updated
      }
      
      return updated
    } catch (e: any) {
      error.value = e.message || 'Error updating status'
      throw e
    } finally {
      loading.value = false
    }
  }

  async function fetchHistory(id: string) {
    try {
      return await applicationService.getHistory(id)
    } catch (e: any) {
      error.value = e.message || 'Error loading history'
      throw e
    }
  }

  // Handle real-time updates from WebSocket
  function handleRealtimeUpdate(data: any) {
    realtimeUpdates.value++
    
    if (data.type === 'application_created') {
      // Add new application to the top of the list
      const exists = applications.value.find(a => a.id === data.application.id)
      if (!exists) {
        applications.value.unshift(data.application)
      }
    } else if (data.type === 'application_updated' || data.type === 'status_changed') {
      // Update existing application
      const index = applications.value.findIndex(a => a.id === data.application_id)
      if (index !== -1) {
        applications.value[index] = { ...applications.value[index], ...data.data }
      }
      
      // Update current application if viewing
      if (currentApplication.value?.id === data.application_id) {
        currentApplication.value = { ...currentApplication.value, ...data.data }
      }
    }
  }

  function setFilters(newFilters: ApplicationFilter) {
    filters.value = { ...filters.value, ...newFilters }
  }

  function clearFilters() {
    filters.value = {}
  }

  return {
    applications,
    currentApplication,
    pagination,
    filters,
    loading,
    error,
    realtimeUpdates,
    filteredApplications,
    fetchApplications,
    fetchApplication,
    createApplication,
    updateStatus,
    fetchHistory,
    handleRealtimeUpdate,
    setFilters,
    clearFilters
  }
})

