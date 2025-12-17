<script setup lang="ts">
import { ref, computed, watch, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useApplicationsStore } from '@/stores/applications'
import { useCountriesStore } from '@/stores/countries'
import { useToast } from 'primevue/usetoast'
import Card from 'primevue/card'
import InputText from 'primevue/inputtext'
import InputNumber from 'primevue/inputnumber'
import Dropdown from 'primevue/dropdown'
import Button from 'primevue/button'
import Message from 'primevue/message'

const router = useRouter()
const applicationsStore = useApplicationsStore()
const countriesStore = useCountriesStore()
const toast = useToast()

const form = ref({
  country_code: '',
  full_name: '',
  document_type: '',
  document_number: '',
  email: '',
  phone: '',
  requested_amount: 0,
  monthly_income: 0
})

const documentTypes = ref<any[]>([])
const isSubmitting = ref(false)
const errors = ref<Record<string, string>>({})

const selectedCountry = computed(() => {
  return countriesStore.countryByCode[form.value.country_code]
})

const amountLimits = computed(() => {
  if (!selectedCountry.value) return null
  return {
    min: selectedCountry.value.config.min_loan_amount,
    max: selectedCountry.value.config.max_loan_amount,
    reviewThreshold: selectedCountry.value.config.review_threshold
  }
})

const willRequireReview = computed(() => {
  if (!amountLimits.value) return false
  return form.value.requested_amount >= amountLimits.value.reviewThreshold
})

const isFormValid = computed(() => {
  return (
    form.value.country_code &&
    form.value.full_name.length >= 3 &&
    form.value.document_type &&
    form.value.document_number &&
    form.value.email.includes('@') &&
    form.value.requested_amount > 0 &&
    form.value.monthly_income > 0
  )
})

watch(() => form.value.country_code, async (code) => {
  if (code) {
    form.value.document_type = ''
    documentTypes.value = await countriesStore.fetchDocumentTypes(code)
    if (documentTypes.value.length > 0) {
      const required = documentTypes.value.find(d => d.is_required)
      if (required) {
        form.value.document_type = required.code
      }
    }
  }
})

function validateForm(): boolean {
  errors.value = {}
  
  if (!form.value.country_code) {
    errors.value.country_code = 'Seleccione un país'
  }
  
  if (form.value.full_name.length < 3) {
    errors.value.full_name = 'El nombre debe tener al menos 3 caracteres'
  }
  
  if (!form.value.document_type) {
    errors.value.document_type = 'Seleccione un tipo de documento'
  }
  
  if (!form.value.document_number) {
    errors.value.document_number = 'Ingrese el número de documento'
  }
  
  if (!form.value.email.includes('@')) {
    errors.value.email = 'Ingrese un email válido'
  }
  
  if (amountLimits.value) {
    if (form.value.requested_amount < amountLimits.value.min) {
      errors.value.requested_amount = `El monto mínimo es ${countriesStore.formatCurrency(amountLimits.value.min, form.value.country_code)}`
    } else if (form.value.requested_amount > amountLimits.value.max) {
      errors.value.requested_amount = `El monto máximo es ${countriesStore.formatCurrency(amountLimits.value.max, form.value.country_code)}`
    }
  }
  
  if (form.value.monthly_income <= 0) {
    errors.value.monthly_income = 'Ingrese el ingreso mensual'
  }
  
  return Object.keys(errors.value).length === 0
}

async function handleSubmit() {
  if (!validateForm()) return
  
  isSubmitting.value = true
  try {
    const application = await applicationsStore.createApplication(form.value)
    
    toast.add({
      severity: 'success',
      summary: 'Solicitud Creada',
      detail: 'La solicitud se ha creado correctamente',
      life: 5000
    })
    
    router.push(`/applications/${application.id}`)
  } catch (error: any) {
    toast.add({
      severity: 'error',
      summary: 'Error',
      detail: error.message || 'No se pudo crear la solicitud',
      life: 5000
    })
  } finally {
    isSubmitting.value = false
  }
}

onMounted(async () => {
  await countriesStore.fetchCountries()
})
</script>

<template>
  <div class="new-application-view">
    <div class="page-header">
      <div>
        <h1>Nueva Solicitud de Crédito</h1>
        <p class="text-secondary">Complete el formulario para crear una nueva solicitud</p>
      </div>
      <Button
        label="Cancelar"
        severity="secondary"
        text
        @click="router.back()"
      />
    </div>

    <div class="form-container">
      <Card>
        <template #content>
          <form @submit.prevent="handleSubmit" class="application-form">
            <!-- País -->
            <div class="form-section">
              <h3><i class="pi pi-globe"></i> País</h3>
              <div class="form-grid">
                <div class="form-group">
                  <label for="country">País de la solicitud *</label>
                  <Dropdown
                    id="country"
                    v-model="form.country_code"
                    :options="countriesStore.activeCountries"
                    optionLabel="name"
                    optionValue="code"
                    placeholder="Seleccione un país"
                    class="w-full"
                    :class="{ 'p-invalid': errors.country_code }"
                  >
                    <template #option="{ option }">
                      <div class="country-option">
                        <span class="code">{{ option.code }}</span>
                        <span>{{ option.name }}</span>
                        <span class="currency">{{ option.currency }}</span>
                      </div>
                    </template>
                  </Dropdown>
                  <small v-if="errors.country_code" class="p-error">{{ errors.country_code }}</small>
                </div>
              </div>
            </div>

            <!-- Datos Personales -->
            <div class="form-section">
              <h3><i class="pi pi-user"></i> Datos del Solicitante</h3>
              <div class="form-grid">
                <div class="form-group">
                  <label for="full_name">Nombre completo *</label>
                  <InputText
                    id="full_name"
                    v-model="form.full_name"
                    placeholder="Nombre y apellidos"
                    class="w-full"
                    :class="{ 'p-invalid': errors.full_name }"
                  />
                  <small v-if="errors.full_name" class="p-error">{{ errors.full_name }}</small>
                </div>

                <div class="form-group">
                  <label for="email">Correo electrónico *</label>
                  <InputText
                    id="email"
                    v-model="form.email"
                    type="email"
                    placeholder="email@ejemplo.com"
                    class="w-full"
                    :class="{ 'p-invalid': errors.email }"
                  />
                  <small v-if="errors.email" class="p-error">{{ errors.email }}</small>
                </div>

                <div class="form-group">
                  <label for="document_type">Tipo de documento *</label>
                  <Dropdown
                    id="document_type"
                    v-model="form.document_type"
                    :options="documentTypes"
                    optionLabel="name"
                    optionValue="code"
                    placeholder="Seleccione tipo"
                    class="w-full"
                    :disabled="!form.country_code"
                    :class="{ 'p-invalid': errors.document_type }"
                  />
                  <small v-if="errors.document_type" class="p-error">{{ errors.document_type }}</small>
                </div>

                <div class="form-group">
                  <label for="document_number">Número de documento *</label>
                  <InputText
                    id="document_number"
                    v-model="form.document_number"
                    placeholder="Número de documento"
                    class="w-full"
                    :class="{ 'p-invalid': errors.document_number }"
                  />
                  <small v-if="errors.document_number" class="p-error">{{ errors.document_number }}</small>
                </div>

                <div class="form-group">
                  <label for="phone">Teléfono (opcional)</label>
                  <InputText
                    id="phone"
                    v-model="form.phone"
                    placeholder="+34 600 000 000"
                    class="w-full"
                  />
                </div>
              </div>
            </div>

            <!-- Datos Financieros -->
            <div class="form-section">
              <h3><i class="pi pi-wallet"></i> Información Financiera</h3>
              
              <Message v-if="amountLimits" severity="info" :closable="false">
                Límites para {{ selectedCountry?.name }}:
                {{ countriesStore.formatCurrency(amountLimits.min, form.country_code) }} -
                {{ countriesStore.formatCurrency(amountLimits.max, form.country_code) }}
              </Message>

              <div class="form-grid">
                <div class="form-group">
                  <label for="requested_amount">Monto solicitado *</label>
                  <InputNumber
                    id="requested_amount"
                    v-model="form.requested_amount"
                    :min="0"
                    :minFractionDigits="2"
                    :maxFractionDigits="2"
                    mode="currency"
                    :currency="selectedCountry?.currency || 'USD'"
                    class="w-full"
                    :class="{ 'p-invalid': errors.requested_amount }"
                  />
                  <small v-if="errors.requested_amount" class="p-error">{{ errors.requested_amount }}</small>
                </div>

                <div class="form-group">
                  <label for="monthly_income">Ingreso mensual *</label>
                  <InputNumber
                    id="monthly_income"
                    v-model="form.monthly_income"
                    :min="0"
                    :minFractionDigits="2"
                    :maxFractionDigits="2"
                    mode="currency"
                    :currency="selectedCountry?.currency || 'USD'"
                    class="w-full"
                    :class="{ 'p-invalid': errors.monthly_income }"
                  />
                  <small v-if="errors.monthly_income" class="p-error">{{ errors.monthly_income }}</small>
                </div>
              </div>

              <Message v-if="willRequireReview" severity="warn" :closable="false">
                <i class="pi pi-exclamation-triangle"></i>
                Esta solicitud requerirá revisión manual debido al monto solicitado.
              </Message>
            </div>

            <!-- Submit -->
            <div class="form-actions">
              <Button
                type="button"
                label="Cancelar"
                severity="secondary"
                @click="router.back()"
              />
              <Button
                type="submit"
                label="Crear Solicitud"
                icon="pi pi-check"
                :loading="isSubmitting"
                :disabled="!isFormValid"
              />
            </div>
          </form>
        </template>
      </Card>
    </div>
  </div>
</template>

<style scoped lang="scss">
.new-application-view {
  animation: slideIn 0.3s ease;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 1.5rem;
  
  h1 {
    font-size: 1.75rem;
    font-weight: 700;
    margin-bottom: 0.25rem;
  }
  
  .text-secondary {
    color: var(--color-text-secondary);
  }
}

.form-container {
  max-width: 800px;
}

.form-section {
  margin-bottom: 2rem;
  padding-bottom: 2rem;
  border-bottom: 1px solid var(--color-surface-lighter);
  
  &:last-of-type {
    border-bottom: none;
  }
  
  h3 {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    font-size: 1rem;
    font-weight: 600;
    color: var(--color-primary);
    margin-bottom: 1.5rem;
    
    i {
      font-size: 1.125rem;
    }
  }
}

.form-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 1.5rem;
  
  @media (max-width: 768px) {
    grid-template-columns: 1fr;
  }
}

.form-group {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  
  label {
    font-size: 0.875rem;
    font-weight: 500;
    color: var(--color-text-secondary);
  }
}

.country-option {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  
  .code {
    font-weight: 600;
    color: var(--color-primary);
    min-width: 30px;
  }
  
  .currency {
    margin-left: auto;
    font-size: 0.75rem;
    color: var(--color-text-muted);
  }
}

.form-actions {
  display: flex;
  justify-content: flex-end;
  gap: 1rem;
  margin-top: 2rem;
  padding-top: 2rem;
  border-top: 1px solid var(--color-surface-lighter);
}

:deep(.p-message) {
  margin-bottom: 1rem;
}
</style>

