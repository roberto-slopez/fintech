<script setup lang="ts">
import { useRouter } from 'vue-router'
import Button from 'primevue/button'

const router = useRouter()
</script>

<template>
  <div class="not-found-view">
    <div class="content">
      <div class="error-code">404</div>
      <h1>Página no encontrada</h1>
      <p>La página que buscas no existe o ha sido movida.</p>
      <Button
        label="Volver al inicio"
        icon="pi pi-home"
        @click="router.push('/')"
      />
    </div>
  </div>
</template>

<style scoped lang="scss">
.not-found-view {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 2rem;
}

.content {
  text-align: center;
  
  .error-code {
    font-size: 8rem;
    font-weight: 800;
    color: var(--color-primary);
    line-height: 1;
    opacity: 0.5;
  }
  
  h1 {
    font-size: 2rem;
    font-weight: 700;
    margin-bottom: 0.5rem;
  }
  
  p {
    color: var(--color-text-secondary);
    margin-bottom: 2rem;
  }
}
</style>

