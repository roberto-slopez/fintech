<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useApplicationsStore } from '@/stores/applications'
import { useCountriesStore } from '@/stores/countries'
import { useToast } from 'primevue/usetoast'
import Card from 'primevue/card'
import Button from 'primevue/button'
import Tag from 'primevue/tag'
import Timeline from 'primevue/timeline'
import Dialog from 'primevue/dialog'
import Dropdown from 'primevue/dropdown'
import Textarea from 'primevue/textarea'
import ProgressSpinner from 'primevue/progressspinner'
import { format } from 'date-fns'
import { es } from 'date-fns/locale'
import type { StateTransition } from '@/types'

const route = useRoute()
const router = useRouter()
const applicationsStore = useApplicationsStore()
const countriesStore = useCountriesStore()
const toast = useToast()

const loading = ref(true)
const history = ref<StateTransition[]>([])
const showStatusDialog = ref(false)
const newStatus = ref('')
const statusReason = ref('')
const isUpdating = ref(false)

const application = computed(() => applicationsStore.currentApplication)

const statusOptions = computed(() => {
  if (!application.value) return []
  
  const transitions: Record<string, string[]> = {
    PENDING: ['VALIDATING', 'CANCELLED'],
    VALIDATING: ['PENDING_BANK_INFO', 'UNDER_REVIEW', 'APPROVED', 'REJECTED'],
    PENDING_BANK_INFO: ['VALIDATING', 'UNDER_REVIEW', 'REJECTED', 'CANCELLED'],
    UNDER_REVIEW: ['APPROVED', 'REJECTED', 'CANCELLED'],
    APPROVED: ['DISBURSED', 'CANCELLED', 'EXPIRED']
  }
  
  const available = transitions[application.value.status] || []
  return available.map(s => ({
    value: s,
    label: getStatusLabel(s)
  }))
})

function getStatusSeverity(status: string): string {
  const severities: Record<string, string> = {
    PENDING: 'warning',
    VALIDATING: 'info',
    PENDING_BANK_INFO: 'info',
    UNDER_REVIEW: 'info',
    APPROVED: 'success',
    REJECTED: 'danger',
    DISBURSED: 'success',
    CANCELLED: 'secondary',
    EXPIRED: 'secondary'
  }
  return severities[status] || 'info'
}

function getStatusLabel(status: string): string {
  const labels: Record<string, string> = {
    PENDING: 'Pendiente',
    VALIDATING: 'Validando',
    PENDING_BANK_INFO: 'Esperando Info Bancaria',
    UNDER_REVIEW: 'En Revisión',
    APPROVED: 'Aprobada',
    REJECTED: 'Rechazada',
    CANCELLED: 'Cancelada',
    EXPIRED: 'Expirada',
    DISBURSED: 'Desembolsada'
  }
  return labels[status] || status
}

function formatDate(date: string): string {
  return format(new Date(date), "d 'de' MMMM yyyy, HH:mm", { locale: es })
}

async function updateStatus() {
  if (!newStatus.value || !application.value) return
  
  isUpdating.value = true
  try {
    await applicationsStore.updateStatus(application.value.id, newStatus.value, statusReason.value)
    
    // Reload history
    history.value = await applicationsStore.fetchHistory(application.value.id)
    
    toast.add({
      severity: 'success',
      summary: 'Estado actualizado',
      detail: `La solicitud ahora está ${getStatusLabel(newStatus.value)}`,
      life: 3000
    })
    
    showStatusDialog.value = false
    newStatus.value = ''
    statusReason.value = ''
  } catch (error: any) {
    toast.add({
      severity: 'error',
      summary: 'Error',
      detail: error.message || 'No se pudo actualizar el estado',
      life: 5000
    })
  } finally {
    isUpdating.value = false
  }
}

onMounted(async () => {
  try {
    const id = route.params.id as string
    await applicationsStore.fetchApplication(id)
    
    // Fetch history with null check
    const historyData = await applicationsStore.fetchHistory(id)
    history.value = historyData || []
    
    await countriesStore.fetchCountries()
  } catch (error) {
    toast.add({
      severity: 'error',
      summary: 'Error',
      detail: 'No se pudo cargar la solicitud',
      life: 5000
    })
    router.push('/applications')
  } finally {
    loading.value = false
  }
})
</script>

<template>
  <div class="application-detail-view">
    <!-- Loading -->
    <div v-if="loading" class="loading-state">
      <ProgressSpinner />
    </div>

    <!-- Content -->
    <template v-else-if="application">
      <!-- Header -->
      <div class="page-header">
        <div class="header-content">
          <Button
            icon="pi pi-arrow-left"
            severity="secondary"
            text
            @click="router.back()"
          />
          <div>
            <h1>{{ application.full_name }}</h1>
            <p class="text-secondary">
              {{ application.document_type }}: {{ application.document_number }}
            </p>
          </div>
        </div>
        <div class="header-actions">
          <Tag :severity="getStatusSeverity(application.status)" class="status-tag">
            {{ getStatusLabel(application.status) }}
          </Tag>
          <Button
            v-if="statusOptions.length > 0"
            label="Cambiar Estado"
            icon="pi pi-pencil"
            @click="showStatusDialog = true"
          />
        </div>
      </div>

      <!-- Main Grid -->
      <div class="detail-grid">
        <!-- Info Cards -->
        <div class="info-column">
          <!-- Datos del Solicitante -->
          <Card>
            <template #title>
              <i class="pi pi-user"></i> Datos del Solicitante
            </template>
            <template #content>
              <div class="info-grid">
                <div class="info-item">
                  <span class="label">Nombre</span>
                  <span class="value">{{ application.full_name }}</span>
                </div>
                <div class="info-item">
                  <span class="label">Documento</span>
                  <span class="value">{{ application.document_type }}: {{ application.document_number }}</span>
                </div>
                <div class="info-item">
                  <span class="label">Email</span>
                  <span class="value">{{ application.email }}</span>
                </div>
                <div class="info-item" v-if="application.phone">
                  <span class="label">Teléfono</span>
                  <span class="value">{{ application.phone }}</span>
                </div>
              </div>
            </template>
          </Card>

          <!-- Información Financiera -->
          <Card>
            <template #title>
              <i class="pi pi-wallet"></i> Información Financiera
            </template>
            <template #content>
              <div class="info-grid">
                <div class="info-item">
                  <span class="label">País</span>
                  <span class="value country">
                    <span class="country-code">{{ application.country?.code }}</span>
                    {{ application.country?.name }}
                  </span>
                </div>
                <div class="info-item">
                  <span class="label">Monto Solicitado</span>
                  <span class="value amount">
                    {{ countriesStore.formatCurrency(application.requested_amount, application.country?.code || '') }}
                  </span>
                </div>
                <div class="info-item">
                  <span class="label">Ingreso Mensual</span>
                  <span class="value">
                    {{ countriesStore.formatCurrency(application.monthly_income, application.country?.code || '') }}
                  </span>
                </div>
                <div class="info-item" v-if="application.risk_score">
                  <span class="label">Score de Riesgo</span>
                  <span class="value">{{ application.risk_score }}</span>
                </div>
              </div>

              <div v-if="application.requires_review" class="review-badge">
                <i class="pi pi-exclamation-triangle"></i>
                Requiere revisión manual
              </div>
            </template>
          </Card>

          <!-- Información Bancaria -->
          <Card v-if="application.banking_info">
            <template #title>
              <i class="pi pi-building"></i> Información Bancaria
            </template>
            <template #content>
              <div class="info-grid">
                <div class="info-item">
                  <span class="label">Proveedor</span>
                  <span class="value">{{ application.banking_info.provider_name }}</span>
                </div>
                <div class="info-item" v-if="application.banking_info.credit_score">
                  <span class="label">Score Crediticio</span>
                  <span class="value">{{ application.banking_info.credit_score }}</span>
                </div>
                <div class="info-item" v-if="application.banking_info.total_debt">
                  <span class="label">Deuda Total</span>
                  <span class="value">
                    {{ countriesStore.formatCurrency(application.banking_info.total_debt, application.country?.code || '') }}
                  </span>
                </div>
                <div class="info-item">
                  <span class="label">Cuentas Bancarias</span>
                  <span class="value">{{ application.banking_info.bank_accounts }}</span>
                </div>
                <div class="info-item">
                  <span class="label">Préstamos Activos</span>
                  <span class="value">{{ application.banking_info.active_loans }}</span>
                </div>
              </div>
            </template>
          </Card>
        </div>

        <!-- Timeline -->
        <div class="timeline-column">
          <Card>
            <template #title>
              <i class="pi pi-history"></i> Historial
            </template>
            <template #content>
              <Timeline :value="history" class="custom-timeline">
                <template #marker="{ item }">
                  <span
                    class="timeline-marker"
                    :class="getStatusSeverity(item.to_status)"
                  >
                    <i class="pi pi-circle-fill"></i>
                  </span>
                </template>
                <template #content="{ item }">
                  <div class="timeline-item">
                    <div class="timeline-header">
                      <Tag :severity="getStatusSeverity(item.to_status)" size="small">
                        {{ getStatusLabel(item.to_status) }}
                      </Tag>
                      <span class="triggered-by">{{ item.triggered_by }}</span>
                    </div>
                    <div class="timeline-date">{{ formatDate(item.created_at) }}</div>
                    <div v-if="item.reason" class="timeline-reason">{{ item.reason }}</div>
                  </div>
                </template>
              </Timeline>

              <div v-if="history.length === 0" class="empty-history">
                <i class="pi pi-clock"></i>
                <p>No hay historial de cambios</p>
              </div>
            </template>
          </Card>

          <!-- Fechas -->
          <Card>
            <template #title>
              <i class="pi pi-calendar"></i> Fechas
            </template>
            <template #content>
              <div class="dates-list">
                <div class="date-item">
                  <span class="label">Fecha de solicitud</span>
                  <span class="value">{{ formatDate(application.application_date) }}</span>
                </div>
                <div class="date-item">
                  <span class="label">Creada</span>
                  <span class="value">{{ formatDate(application.created_at) }}</span>
                </div>
                <div class="date-item">
                  <span class="label">Última actualización</span>
                  <span class="value">{{ formatDate(application.updated_at) }}</span>
                </div>
                <div class="date-item" v-if="application.processed_at">
                  <span class="label">Procesada</span>
                  <span class="value">{{ formatDate(application.processed_at) }}</span>
                </div>
              </div>
            </template>
          </Card>
        </div>
      </div>
    </template>

    <!-- Status Dialog -->
    <Dialog
      v-model:visible="showStatusDialog"
      header="Cambiar Estado"
      modal
      :style="{ width: '450px' }"
    >
      <div class="status-dialog-content">
        <div class="form-group">
          <label>Nuevo Estado</label>
          <Dropdown
            v-model="newStatus"
            :options="statusOptions"
            optionLabel="label"
            optionValue="value"
            placeholder="Seleccione un estado"
            class="w-full"
          />
        </div>
        <div class="form-group">
          <label>Razón (opcional)</label>
          <Textarea
            v-model="statusReason"
            placeholder="Ingrese una razón para el cambio de estado..."
            rows="3"
            class="w-full"
          />
        </div>
      </div>
      <template #footer>
        <Button
          label="Cancelar"
          severity="secondary"
          @click="showStatusDialog = false"
        />
        <Button
          label="Actualizar"
          @click="updateStatus"
          :loading="isUpdating"
          :disabled="!newStatus"
        />
      </template>
    </Dialog>
  </div>
</template>

<style scoped lang="scss">
.application-detail-view {
  animation: slideIn 0.3s ease;
}

.loading-state {
  display: flex;
  justify-content: center;
  align-items: center;
  min-height: 400px;
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 2rem;
  flex-wrap: wrap;
  gap: 1rem;
  
  .header-content {
    display: flex;
    align-items: center;
    gap: 1rem;
    
    h1 {
      font-size: 1.5rem;
      font-weight: 700;
      margin-bottom: 0.125rem;
    }
    
    .text-secondary {
      color: var(--color-text-secondary);
      font-size: 0.875rem;
    }
  }
  
  .header-actions {
    display: flex;
    align-items: center;
    gap: 1rem;
  }
  
  .status-tag {
    font-size: 0.875rem;
    padding: 0.5rem 1rem;
  }
}

.detail-grid {
  display: grid;
  grid-template-columns: 2fr 1fr;
  gap: 1.5rem;
  
  @media (max-width: 1024px) {
    grid-template-columns: 1fr;
  }
}

.info-column, .timeline-column {
  display: flex;
  flex-direction: column;
  gap: 1.5rem;
}

.info-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 1.25rem;
  
  @media (max-width: 768px) {
    grid-template-columns: 1fr;
  }
}

.info-item {
  .label {
    display: block;
    font-size: 0.75rem;
    color: var(--color-text-muted);
    text-transform: uppercase;
    letter-spacing: 0.05em;
    margin-bottom: 0.25rem;
  }
  
  .value {
    font-size: 0.9375rem;
    color: var(--color-text);
    
    &.amount {
      font-weight: 600;
      font-size: 1.125rem;
      color: var(--color-primary);
    }
    
    &.country {
      display: flex;
      align-items: center;
      gap: 0.5rem;
      
      .country-code {
        background: var(--color-surface);
        padding: 0.125rem 0.375rem;
        border-radius: 4px;
        font-weight: 600;
        font-size: 0.75rem;
      }
    }
  }
}

.review-badge {
  display: inline-flex;
  align-items: center;
  gap: 0.5rem;
  margin-top: 1rem;
  padding: 0.5rem 1rem;
  background: rgba(245, 158, 11, 0.15);
  color: var(--color-warning);
  border-radius: var(--border-radius);
  font-size: 0.875rem;
  font-weight: 500;
}

.custom-timeline {
  :deep(.p-timeline-event-content) {
    padding-bottom: 1.5rem;
  }
}

.timeline-marker {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 24px;
  height: 24px;
  border-radius: 50%;
  
  i {
    font-size: 0.5rem;
  }
  
  &.warning {
    background: rgba(245, 158, 11, 0.2);
    color: var(--color-warning);
  }
  
  &.info {
    background: rgba(59, 130, 246, 0.2);
    color: var(--color-info);
  }
  
  &.success {
    background: rgba(34, 197, 94, 0.2);
    color: var(--color-success);
  }
  
  &.danger {
    background: rgba(239, 68, 68, 0.2);
    color: var(--color-danger);
  }
}

.timeline-item {
  .timeline-header {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    margin-bottom: 0.25rem;
    
    .triggered-by {
      font-size: 0.75rem;
      color: var(--color-text-muted);
    }
  }
  
  .timeline-date {
    font-size: 0.75rem;
    color: var(--color-text-secondary);
  }
  
  .timeline-reason {
    margin-top: 0.5rem;
    font-size: 0.875rem;
    color: var(--color-text);
    background: var(--color-surface);
    padding: 0.5rem;
    border-radius: var(--border-radius);
  }
}

.empty-history {
  text-align: center;
  padding: 2rem;
  color: var(--color-text-muted);
  
  i {
    font-size: 2rem;
    margin-bottom: 0.5rem;
  }
}

.dates-list {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.date-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding-bottom: 0.75rem;
  border-bottom: 1px solid var(--color-surface-lighter);
  
  &:last-child {
    border-bottom: none;
    padding-bottom: 0;
  }
  
  .label {
    font-size: 0.875rem;
    color: var(--color-text-secondary);
  }
  
  .value {
    font-size: 0.875rem;
    color: var(--color-text);
  }
}

.status-dialog-content {
  .form-group {
    margin-bottom: 1.5rem;
    
    label {
      display: block;
      margin-bottom: 0.5rem;
      font-size: 0.875rem;
      font-weight: 500;
      color: var(--color-text-secondary);
    }
  }
}
</style>

