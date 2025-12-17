<script setup lang="ts">
import { ref, computed } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { useToast } from 'primevue/usetoast'
import InputText from 'primevue/inputtext'
import Password from 'primevue/password'
import Button from 'primevue/button'

const router = useRouter()
const route = useRoute()
const authStore = useAuthStore()
const toast = useToast()

const email = ref('')
const password = ref('')
const isLoading = ref(false)

const isFormValid = computed(() => {
  return email.value.includes('@') && password.value.length >= 6
})

async function handleLogin() {
  if (!isFormValid.value) return

  isLoading.value = true
  try {
    await authStore.login({
      email: email.value,
      password: password.value
    })

    toast.add({
      severity: 'success',
      summary: 'Bienvenido',
      detail: `Hola, ${authStore.user?.full_name}`,
      life: 3000
    })

    const redirect = route.query.redirect as string || '/'
    router.push(redirect)
  } catch (error: any) {
    toast.add({
      severity: 'error',
      summary: 'Error de autenticación',
      detail: error.message || 'Credenciales inválidas',
      life: 5000
    })
  } finally {
    isLoading.value = false
  }
}
</script>

<template>
  <div class="login-container">
    <div class="login-card">
      <div class="login-header">
        <div class="logo">
          <i class="pi pi-credit-card"></i>
          <span>Fintech Multipaís</span>
        </div>
        <p class="subtitle">Sistema de Gestión de Créditos</p>
      </div>

      <form @submit.prevent="handleLogin" class="login-form">
        <div class="form-group">
          <label for="email">Correo electrónico</label>
          <span class="p-input-icon-left">
            <i class="pi pi-envelope" />
            <InputText
              id="email"
              v-model="email"
              type="email"
              placeholder="usuario@ejemplo.com"
              class="w-full"
              :disabled="isLoading"
            />
          </span>
        </div>

        <div class="form-group">
          <label for="password">Contraseña</label>
          <Password
            id="password"
            v-model="password"
            placeholder="••••••••"
            :feedback="false"
            toggleMask
            class="w-full"
            inputClass="w-full"
            :disabled="isLoading"
          />
        </div>

        <Button
          type="submit"
          label="Iniciar sesión"
          icon="pi pi-sign-in"
          class="w-full mt-4"
          :loading="isLoading"
          :disabled="!isFormValid"
        />
      </form>

      <div class="login-footer">
        <p class="hint">
          Usuario demo: <strong>admin@fintech.com</strong><br>
          Contraseña: <strong>admin123</strong>
        </p>
      </div>
    </div>

    <div class="login-decoration">
      <div class="decoration-content">
        <h2>Gestión de Créditos</h2>
        <p>Plataforma multipaís para la gestión integral de solicitudes de crédito.</p>
        
        <div class="features">
          <div class="feature">
            <i class="pi pi-globe"></i>
            <span>6 países</span>
          </div>
          <div class="feature">
            <i class="pi pi-shield"></i>
            <span>Seguro</span>
          </div>
          <div class="feature">
            <i class="pi pi-bolt"></i>
            <span>Tiempo real</span>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped lang="scss">
.login-container {
  min-height: 100vh;
  display: flex;
  
  @media (max-width: 768px) {
    flex-direction: column;
  }
}

.login-card {
  flex: 1;
  display: flex;
  flex-direction: column;
  justify-content: center;
  padding: 3rem;
  max-width: 480px;
  margin: 0 auto;
  
  @media (min-width: 769px) {
    max-width: 50%;
    padding: 4rem;
  }
}

.login-header {
  text-align: center;
  margin-bottom: 3rem;
  
  .logo {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 0.75rem;
    font-size: 1.75rem;
    font-weight: 700;
    color: var(--color-primary);
    margin-bottom: 0.5rem;
    
    i {
      font-size: 2rem;
    }
  }
  
  .subtitle {
    color: var(--color-text-secondary);
    font-size: 1rem;
  }
}

.login-form {
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

.login-footer {
  margin-top: 2rem;
  text-align: center;
  
  .hint {
    font-size: 0.75rem;
    color: var(--color-text-muted);
    background: var(--color-surface-lighter);
    padding: 1rem;
    border-radius: var(--border-radius);
    
    strong {
      color: var(--color-text-secondary);
    }
  }
}

.login-decoration {
  flex: 1;
  background: linear-gradient(135deg, var(--color-primary-dark), var(--color-primary));
  display: none;
  
  @media (min-width: 769px) {
    display: flex;
    align-items: center;
    justify-content: center;
  }
  
  .decoration-content {
    color: white;
    text-align: center;
    padding: 3rem;
    
    h2 {
      font-size: 2.5rem;
      font-weight: 700;
      margin-bottom: 1rem;
    }
    
    p {
      font-size: 1.125rem;
      opacity: 0.9;
      margin-bottom: 2rem;
    }
  }
  
  .features {
    display: flex;
    justify-content: center;
    gap: 2rem;
    
    .feature {
      display: flex;
      flex-direction: column;
      align-items: center;
      gap: 0.5rem;
      
      i {
        font-size: 2rem;
      }
      
      span {
        font-size: 0.875rem;
        font-weight: 500;
      }
    }
  }
}

:deep(.p-inputtext) {
  width: 100%;
}

:deep(.p-password) {
  width: 100%;
}
</style>

