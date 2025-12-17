<script setup lang="ts">
import { onMounted } from 'vue'
import { useCountriesStore } from '@/stores/countries'
import Card from 'primevue/card'
import Tag from 'primevue/tag'

const countriesStore = useCountriesStore()

onMounted(async () => {
  await countriesStore.fetchCountries()
})
</script>

<template>
  <div class="countries-view">
    <div class="page-header">
      <h1>Países Configurados</h1>
      <p class="text-secondary">Configuración y reglas por país</p>
    </div>

    <div class="countries-grid">
      <Card
        v-for="country in countriesStore.countries"
        :key="country.code"
        class="country-card"
      >
        <template #header>
          <div class="country-header">
            <div class="country-code">{{ country.code }}</div>
            <Tag :severity="country.is_active ? 'success' : 'secondary'">
              {{ country.is_active ? 'Activo' : 'Inactivo' }}
            </Tag>
          </div>
        </template>
        <template #title>{{ country.name }}</template>
        <template #subtitle>{{ country.currency }} · {{ country.timezone }}</template>
        <template #content>
          <div class="config-grid">
            <div class="config-item">
              <span class="label">Monto Mínimo</span>
              <span class="value">{{ countriesStore.formatCurrency(country.config.min_loan_amount, country.code) }}</span>
            </div>
            <div class="config-item">
              <span class="label">Monto Máximo</span>
              <span class="value">{{ countriesStore.formatCurrency(country.config.max_loan_amount, country.code) }}</span>
            </div>
            <div class="config-item">
              <span class="label">Umbral Revisión</span>
              <span class="value">{{ countriesStore.formatCurrency(country.config.review_threshold, country.code) }}</span>
            </div>
            <div class="config-item">
              <span class="label">Score Mínimo</span>
              <span class="value">{{ country.config.min_credit_score }}</span>
            </div>
            <div class="config-item">
              <span class="label">Ratio Deuda/Ingreso</span>
              <span class="value">{{ (country.config.max_debt_to_income_ratio * 100).toFixed(0) }}%</span>
            </div>
            <div class="config-item">
              <span class="label">Ingreso Mínimo</span>
              <span class="value">{{ countriesStore.formatCurrency(country.config.min_income_required, country.code) }}</span>
            </div>
          </div>
        </template>
      </Card>
    </div>
  </div>
</template>

<style scoped lang="scss">
.countries-view {
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

.countries-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(350px, 1fr));
  gap: 1.5rem;
}

.country-card {
  .country-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 1rem 1.5rem;
    background: var(--color-surface);
  }
  
  .country-code {
    font-size: 2rem;
    font-weight: 700;
    color: var(--color-primary);
  }
}

.config-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 1rem;
}

.config-item {
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
    font-weight: 500;
    color: var(--color-text);
  }
}
</style>

