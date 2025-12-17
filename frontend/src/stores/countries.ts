import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { countryService } from '@/services/api'
import type { Country, DocumentType } from '@/types'

export const useCountriesStore = defineStore('countries', () => {
  const countries = ref<Country[]>([])
  const documentTypes = ref<Record<string, DocumentType[]>>({})
  const loading = ref(false)
  const error = ref<string | null>(null)

  const activeCountries = computed(() => countries.value.filter(c => c.is_active))

  const countryByCode = computed(() => {
    const map: Record<string, Country> = {}
    countries.value.forEach(c => {
      map[c.code] = c
    })
    return map
  })

  async function fetchCountries() {
    if (countries.value.length > 0) return // Already loaded
    
    loading.value = true
    error.value = null
    
    try {
      countries.value = await countryService.getAll()
    } catch (e: any) {
      error.value = e.message || 'Error loading countries'
      throw e
    } finally {
      loading.value = false
    }
  }

  async function fetchCountryDetails(code: string) {
    loading.value = true
    error.value = null
    
    try {
      return await countryService.getByCode(code)
    } catch (e: any) {
      error.value = e.message || 'Error loading country details'
      throw e
    } finally {
      loading.value = false
    }
  }

  async function fetchDocumentTypes(countryCode: string) {
    if (documentTypes.value[countryCode]) {
      return documentTypes.value[countryCode]
    }
    
    try {
      const types = await countryService.getDocumentTypes(countryCode)
      documentTypes.value[countryCode] = types
      return types
    } catch (e: any) {
      error.value = e.message || 'Error loading document types'
      throw e
    }
  }

  function getCountryName(code: string): string {
    return countryByCode.value[code]?.name || code
  }

  function getCountryCurrency(code: string): string {
    return countryByCode.value[code]?.currency || ''
  }

  function formatCurrency(amount: number, countryCode: string): string {
    const currency = getCountryCurrency(countryCode)
    
    const locales: Record<string, string> = {
      ES: 'es-ES',
      PT: 'pt-PT',
      IT: 'it-IT',
      MX: 'es-MX',
      CO: 'es-CO',
      BR: 'pt-BR'
    }
    
    try {
      return new Intl.NumberFormat(locales[countryCode] || 'en-US', {
        style: 'currency',
        currency: currency || 'USD'
      }).format(amount)
    } catch {
      return `${currency} ${amount.toFixed(2)}`
    }
  }

  return {
    countries,
    documentTypes,
    loading,
    error,
    activeCountries,
    countryByCode,
    fetchCountries,
    fetchCountryDetails,
    fetchDocumentTypes,
    getCountryName,
    getCountryCurrency,
    formatCurrency
  }
})

