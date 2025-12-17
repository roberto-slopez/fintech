<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useApplicationsStore } from '@/stores/applications'
import { useCountriesStore } from '@/stores/countries'
import Card from 'primevue/card'
import Chart from 'primevue/chart'
import DataTable from 'primevue/datatable'
import Column from 'primevue/column'
import Tag from 'primevue/tag'

const applicationsStore = useApplicationsStore()
const countriesStore = useCountriesStore()

const loading = ref(true)

const stats = computed(() => {
  const apps = applicationsStore.applications
  return {
    total: apps.length,
    pending: apps.filter(a => a.status === 'PENDING' || a.status === 'VALIDATING').length,
    approved: apps.filter(a => a.status === 'APPROVED' || a.status === 'DISBURSED').length,
    rejected: apps.filter(a => a.status === 'REJECTED').length,
    underReview: apps.filter(a => a.requires_review).length
  }
})

const recentApplications = computed(() => {
  return applicationsStore.applications.slice(0, 5)
})

const chartData = computed(() => ({
  labels: ['Pendientes', 'Aprobadas', 'Rechazadas', 'En Revisión'],
  datasets: [
    {
      data: [stats.value.pending, stats.value.approved, stats.value.rejected, stats.value.underReview],
      backgroundColor: ['#f59e0b', '#22c55e', '#ef4444', '#8b5cf6'],
      hoverBackgroundColor: ['#fbbf24', '#4ade80', '#f87171', '#a78bfa']
    }
  ]
}))

const chartOptions = {
  plugins: {
    legend: {
      position: 'bottom',
      labels: {
        color: '#94a3b8'
      }
    }
  },
  maintainAspectRatio: false
}

function getStatusSeverity(status: string): string {
  const severities: Record<string, string> = {
    PENDING: 'warning',
    VALIDATING: 'info',
    UNDER_REVIEW: 'info',
    APPROVED: 'success',
    REJECTED: 'danger',
    DISBURSED: 'success',
    CANCELLED: 'secondary'
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

onMounted(async () => {
  try {
    await Promise.all([
      applicationsStore.fetchApplications({ page_size: 100 }),
      countriesStore.fetchCountries()
    ])
  } finally {
    loading.value = false
  }
})
</script>

<template>
  <div class="dashboard">
    <div class="page-header">
      <h1>Dashboard</h1>
      <p class="text-secondary">Resumen de solicitudes de crédito</p>
    </div>

    <!-- Stats Cards -->
    <div class="stats-grid">
      <div class="stat-card">
        <div class="stat-icon primary">
          <i class="pi pi-file"></i>
        </div>
        <div class="stat-value">{{ stats.total }}</div>
        <div class="stat-label">Total Solicitudes</div>
      </div>

      <div class="stat-card">
        <div class="stat-icon warning">
          <i class="pi pi-clock"></i>
        </div>
        <div class="stat-value">{{ stats.pending }}</div>
        <div class="stat-label">Pendientes</div>
      </div>

      <div class="stat-card">
        <div class="stat-icon success">
          <i class="pi pi-check-circle"></i>
        </div>
        <div class="stat-value">{{ stats.approved }}</div>
        <div class="stat-label">Aprobadas</div>
      </div>

      <div class="stat-card">
        <div class="stat-icon danger">
          <i class="pi pi-times-circle"></i>
        </div>
        <div class="stat-value">{{ stats.rejected }}</div>
        <div class="stat-label">Rechazadas</div>
      </div>
    </div>

    <!-- Content Grid -->
    <div class="content-grid">
      <!-- Chart -->
      <Card class="chart-card">
        <template #title>Distribución por Estado</template>
        <template #content>
          <div class="chart-container">
            <Chart type="doughnut" :data="chartData" :options="chartOptions" />
          </div>
        </template>
      </Card>

      <!-- Recent Applications -->
      <Card class="table-card">
        <template #title>Solicitudes Recientes</template>
        <template #content>
          <DataTable
            :value="recentApplications"
            :loading="loading"
            stripedRows
            size="small"
          >
            <Column field="full_name" header="Solicitante">
              <template #body="{ data }">
                <div class="applicant-cell">
                  <span class="name">{{ data.full_name }}</span>
                  <span class="document">{{ data.document_number }}</span>
                </div>
              </template>
            </Column>
            <Column field="country.code" header="País">
              <template #body="{ data }">
                <span class="country-badge">{{ data.country?.code || 'N/A' }}</span>
              </template>
            </Column>
            <Column field="requested_amount" header="Monto">
              <template #body="{ data }">
                {{ countriesStore.formatCurrency(data.requested_amount, data.country?.code || '') }}
              </template>
            </Column>
            <Column field="status" header="Estado">
              <template #body="{ data }">
                <Tag :severity="getStatusSeverity(data.status)">
                  {{ getStatusLabel(data.status) }}
                </Tag>
              </template>
            </Column>
          </DataTable>

          <div class="view-all">
            <router-link to="/applications">Ver todas las solicitudes →</router-link>
          </div>
        </template>
      </Card>
    </div>

    <!-- Countries Overview -->
    <Card class="countries-card">
      <template #title>Países Activos</template>
      <template #content>
        <div class="countries-grid">
          <div
            v-for="country in countriesStore.activeCountries"
            :key="country.code"
            class="country-item"
          >
            <div class="country-code">{{ country.code }}</div>
            <div class="country-name">{{ country.name }}</div>
            <div class="country-currency">{{ country.currency }}</div>
          </div>
        </div>
      </template>
    </Card>
  </div>
</template>

<style scoped lang="scss">
.dashboard {
  animation: slideIn 0.3s ease;
}

.page-header {
  margin-bottom: 2rem;
  
  h1 {
    font-size: 1.75rem;
    font-weight: 700;
    margin-bottom: 0.25rem;
  }
  
  .text-secondary {
    color: var(--color-text-secondary);
  }
}

.stats-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 1.5rem;
  margin-bottom: 2rem;
}

.content-grid {
  display: grid;
  grid-template-columns: 1fr 2fr;
  gap: 1.5rem;
  margin-bottom: 2rem;
  
  @media (max-width: 1024px) {
    grid-template-columns: 1fr;
  }
}

.chart-card {
  .chart-container {
    height: 300px;
    display: flex;
    align-items: center;
    justify-content: center;
  }
}

.applicant-cell {
  display: flex;
  flex-direction: column;
  gap: 0.125rem;
  
  .name {
    font-weight: 500;
  }
  
  .document {
    font-size: 0.75rem;
    color: var(--color-text-muted);
  }
}

.country-badge {
  display: inline-block;
  padding: 0.25rem 0.5rem;
  background: var(--color-surface-lighter);
  border-radius: 4px;
  font-size: 0.75rem;
  font-weight: 600;
}

.view-all {
  margin-top: 1rem;
  text-align: center;
  
  a {
    color: var(--color-primary);
    text-decoration: none;
    font-size: 0.875rem;
    
    &:hover {
      text-decoration: underline;
    }
  }
}

.countries-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
  gap: 1rem;
}

.country-item {
  background: var(--color-surface);
  border: 1px solid var(--color-surface-lighter);
  border-radius: var(--border-radius);
  padding: 1rem;
  text-align: center;
  transition: transform 0.2s ease;
  
  &:hover {
    transform: translateY(-2px);
  }
  
  .country-code {
    font-size: 1.5rem;
    font-weight: 700;
    color: var(--color-primary);
  }
  
  .country-name {
    font-size: 0.875rem;
    color: var(--color-text);
    margin: 0.25rem 0;
  }
  
  .country-currency {
    font-size: 0.75rem;
    color: var(--color-text-muted);
  }
}
</style>

