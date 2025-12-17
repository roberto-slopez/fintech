<script setup lang="ts">
import { ref, onMounted, watch } from 'vue'
import { useRouter } from 'vue-router'
import { useApplicationsStore } from '@/stores/applications'
import { useCountriesStore } from '@/stores/countries'
import DataTable from 'primevue/datatable'
import Column from 'primevue/column'
import Button from 'primevue/button'
import InputText from 'primevue/inputtext'
import Dropdown from 'primevue/dropdown'
import Tag from 'primevue/tag'
import { format } from 'date-fns'
import { es } from 'date-fns/locale'

const router = useRouter()
const applicationsStore = useApplicationsStore()
const countriesStore = useCountriesStore()

const search = ref('')
const selectedCountry = ref<string | null>(null)
const selectedStatus = ref<string | null>(null)

const statusOptions = [
  { label: 'Todos', value: null },
  { label: 'Pendiente', value: 'PENDING' },
  { label: 'Validando', value: 'VALIDATING' },
  { label: 'En Revisión', value: 'UNDER_REVIEW' },
  { label: 'Aprobada', value: 'APPROVED' },
  { label: 'Rechazada', value: 'REJECTED' },
  { label: 'Desembolsada', value: 'DISBURSED' }
]

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
    PENDING_BANK_INFO: 'Esperando Info',
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
  return format(new Date(date), "d MMM yyyy, HH:mm", { locale: es })
}

function viewApplication(id: string) {
  router.push(`/applications/${id}`)
}

async function loadApplications() {
  await applicationsStore.fetchApplications({
    search: search.value || undefined,
    country: selectedCountry.value || undefined,
    status: selectedStatus.value || undefined,
    page: applicationsStore.pagination.page,
    page_size: applicationsStore.pagination.pageSize
  })
}

function onPage(event: any) {
  applicationsStore.fetchApplications({
    ...applicationsStore.filters,
    page: event.page + 1,
    page_size: event.rows
  })
}

// Debounced search
let searchTimeout: number | null = null
watch(search, () => {
  if (searchTimeout) clearTimeout(searchTimeout)
  searchTimeout = window.setTimeout(loadApplications, 300)
})

watch([selectedCountry, selectedStatus], () => {
  loadApplications()
})

onMounted(async () => {
  await countriesStore.fetchCountries()
  await loadApplications()
})
</script>

<template>
  <div class="applications-view">
    <div class="page-header">
      <div>
        <h1>Solicitudes de Crédito</h1>
        <p class="text-secondary">Gestión y seguimiento de solicitudes</p>
      </div>
      <router-link to="/applications/new">
        <Button label="Nueva Solicitud" icon="pi pi-plus" />
      </router-link>
    </div>

    <!-- Filters -->
    <div class="filters-bar card">
      <span class="p-input-icon-left flex-1">
        <i class="pi pi-search" />
        <InputText
          v-model="search"
          placeholder="Buscar por nombre o documento..."
          class="w-full"
        />
      </span>
      
      <Dropdown
        v-model="selectedCountry"
        :options="[{ code: null, name: 'Todos los países' }, ...countriesStore.activeCountries]"
        optionLabel="name"
        optionValue="code"
        placeholder="País"
        class="filter-dropdown"
      />
      
      <Dropdown
        v-model="selectedStatus"
        :options="statusOptions"
        optionLabel="label"
        optionValue="value"
        placeholder="Estado"
        class="filter-dropdown"
      />
    </div>

    <!-- Realtime indicator -->
    <div v-if="applicationsStore.realtimeUpdates > 0" class="realtime-indicator">
      <i class="pi pi-bolt"></i>
      {{ applicationsStore.realtimeUpdates }} actualizaciones en tiempo real
    </div>

    <!-- Data Table -->
    <div class="card">
      <DataTable
        :value="applicationsStore.applications"
        :loading="applicationsStore.loading"
        :paginator="true"
        :rows="applicationsStore.pagination.pageSize"
        :totalRecords="applicationsStore.pagination.total"
        :lazy="true"
        @page="onPage"
        stripedRows
        removableSort
        :rowsPerPageOptions="[10, 20, 50]"
        paginatorTemplate="FirstPageLink PrevPageLink PageLinks NextPageLink LastPageLink RowsPerPageDropdown"
        currentPageReportTemplate="Mostrando {first} a {last} de {totalRecords}"
        responsiveLayout="scroll"
      >
        <Column field="full_name" header="Solicitante" sortable style="min-width: 200px">
          <template #body="{ data }">
            <div class="applicant-cell">
              <span class="name">{{ data.full_name }}</span>
              <span class="document">{{ data.document_type }}: {{ data.document_number }}</span>
            </div>
          </template>
        </Column>
        
        <Column field="country.code" header="País" sortable style="min-width: 100px">
          <template #body="{ data }">
            <span class="country-badge">{{ data.country?.code || 'N/A' }}</span>
          </template>
        </Column>
        
        <Column field="requested_amount" header="Monto" sortable style="min-width: 150px">
          <template #body="{ data }">
            <span class="amount">
              {{ countriesStore.formatCurrency(data.requested_amount, data.country?.code || '') }}
            </span>
          </template>
        </Column>
        
        <Column field="monthly_income" header="Ingreso Mensual" style="min-width: 150px">
          <template #body="{ data }">
            {{ countriesStore.formatCurrency(data.monthly_income, data.country?.code || '') }}
          </template>
        </Column>
        
        <Column field="status" header="Estado" sortable style="min-width: 130px">
          <template #body="{ data }">
            <Tag :severity="getStatusSeverity(data.status)">
              {{ getStatusLabel(data.status) }}
            </Tag>
          </template>
        </Column>
        
        <Column field="requires_review" header="Revisión" style="min-width: 100px">
          <template #body="{ data }">
            <i
              v-if="data.requires_review"
              class="pi pi-exclamation-triangle review-icon"
              v-tooltip="'Requiere revisión manual'"
            ></i>
          </template>
        </Column>
        
        <Column field="application_date" header="Fecha" sortable style="min-width: 150px">
          <template #body="{ data }">
            <span class="date">{{ formatDate(data.application_date) }}</span>
          </template>
        </Column>
        
        <Column header="Acciones" style="min-width: 100px">
          <template #body="{ data }">
            <Button
              icon="pi pi-eye"
              severity="secondary"
              text
              rounded
              @click="viewApplication(data.id)"
              v-tooltip="'Ver detalles'"
            />
          </template>
        </Column>

        <template #empty>
          <div class="empty-state">
            <i class="pi pi-inbox"></i>
            <p>No se encontraron solicitudes</p>
          </div>
        </template>
      </DataTable>
    </div>
  </div>
</template>

<style scoped lang="scss">
.applications-view {
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

.filters-bar {
  display: flex;
  gap: 1rem;
  align-items: center;
  margin-bottom: 1.5rem;
  flex-wrap: wrap;
  
  .flex-1 {
    flex: 1;
    min-width: 200px;
  }
  
  .filter-dropdown {
    min-width: 180px;
  }
}

.realtime-indicator {
  display: inline-flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.5rem 1rem;
  background: rgba(20, 184, 166, 0.15);
  color: var(--color-primary);
  border-radius: 9999px;
  font-size: 0.75rem;
  font-weight: 500;
  margin-bottom: 1rem;
  animation: pulse 2s infinite;
  
  @keyframes pulse {
    0%, 100% { opacity: 1; }
    50% { opacity: 0.7; }
  }
}

.applicant-cell {
  display: flex;
  flex-direction: column;
  gap: 0.125rem;
  
  .name {
    font-weight: 500;
    color: var(--color-text);
  }
  
  .document {
    font-size: 0.75rem;
    color: var(--color-text-muted);
  }
}

.country-badge {
  display: inline-block;
  padding: 0.25rem 0.5rem;
  background: var(--color-surface);
  border-radius: 4px;
  font-size: 0.75rem;
  font-weight: 600;
}

.amount {
  font-weight: 600;
  color: var(--color-text);
}

.date {
  font-size: 0.875rem;
  color: var(--color-text-secondary);
}

.review-icon {
  color: var(--color-warning);
}

.empty-state {
  padding: 3rem;
  text-align: center;
  color: var(--color-text-muted);
  
  i {
    font-size: 3rem;
    margin-bottom: 1rem;
  }
  
  p {
    font-size: 1rem;
  }
}
</style>

