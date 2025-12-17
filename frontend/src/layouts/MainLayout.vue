<script setup lang="ts">
import { computed } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { useWebSocket } from '@/composables/useWebSocket'
import Button from 'primevue/button'

const router = useRouter()
const route = useRoute()
const authStore = useAuthStore()
const { isConnected } = useWebSocket()

const menuItems = [
  { label: 'Dashboard', icon: 'pi pi-home', route: '/' },
  { label: 'Solicitudes', icon: 'pi pi-file', route: '/applications' },
  { label: 'Nueva Solicitud', icon: 'pi pi-plus', route: '/applications/new' },
  { label: 'Países', icon: 'pi pi-globe', route: '/countries' }
]

const currentRoute = computed(() => route.path)

function isActive(path: string): boolean {
  if (path === '/') return route.path === '/'
  return route.path.startsWith(path)
}

function logout() {
  authStore.logout()
  router.push('/login')
}
</script>

<template>
  <div class="layout">
    <!-- Sidebar -->
    <aside class="sidebar">
      <div class="sidebar-header">
        <div class="logo">
          <i class="pi pi-credit-card"></i>
          <span>Fintech</span>
        </div>
        <div class="connection-status" :class="{ connected: isConnected }">
          <i class="pi" :class="isConnected ? 'pi-wifi' : 'pi-wifi-off'"></i>
          <span>{{ isConnected ? 'En línea' : 'Desconectado' }}</span>
        </div>
      </div>

      <nav class="sidebar-nav">
        <router-link
          v-for="item in menuItems"
          :key="item.route"
          :to="item.route"
          class="nav-item"
          :class="{ active: isActive(item.route) }"
        >
          <i :class="item.icon"></i>
          <span>{{ item.label }}</span>
        </router-link>
      </nav>

      <div class="sidebar-footer">
        <div class="user-info">
          <div class="user-avatar">
            <i class="pi pi-user"></i>
          </div>
          <div class="user-details">
            <div class="user-name">{{ authStore.user?.full_name }}</div>
            <div class="user-role">{{ authStore.user?.role }}</div>
          </div>
        </div>
        <Button
          icon="pi pi-sign-out"
          severity="secondary"
          text
          @click="logout"
          v-tooltip.right="'Cerrar sesión'"
        />
      </div>
    </aside>

    <!-- Main Content -->
    <main class="main-content">
      <router-view v-slot="{ Component }">
        <transition name="fade" mode="out-in">
          <component :is="Component" />
        </transition>
      </router-view>
    </main>
  </div>
</template>

<style scoped lang="scss">
.sidebar {
  display: flex;
  flex-direction: column;
  
  &-header {
    padding: 1rem;
    border-bottom: 1px solid var(--color-surface-lighter);
    margin-bottom: 1rem;
  }
  
  .logo {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    font-size: 1.25rem;
    font-weight: 700;
    color: var(--color-primary);
    margin-bottom: 0.75rem;
    
    i {
      font-size: 1.5rem;
    }
  }
  
  .connection-status {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    font-size: 0.75rem;
    color: var(--color-danger);
    
    &.connected {
      color: var(--color-success);
    }
    
    i {
      font-size: 0.75rem;
    }
  }
}

.sidebar-nav {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.nav-item {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 0.75rem 1rem;
  color: var(--color-text-secondary);
  text-decoration: none;
  border-radius: var(--border-radius);
  transition: all 0.2s ease;
  
  &:hover {
    background: var(--color-surface-lighter);
    color: var(--color-text);
  }
  
  &.active {
    background: rgba(20, 184, 166, 0.15);
    color: var(--color-primary);
    font-weight: 500;
  }
  
  i {
    font-size: 1.125rem;
    width: 24px;
    text-align: center;
  }
}

.sidebar-footer {
  padding: 1rem;
  border-top: 1px solid var(--color-surface-lighter);
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.user-info {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.user-avatar {
  width: 40px;
  height: 40px;
  border-radius: 50%;
  background: var(--color-surface-lighter);
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--color-primary);
}

.user-details {
  .user-name {
    font-size: 0.875rem;
    font-weight: 500;
    color: var(--color-text);
  }
  
  .user-role {
    font-size: 0.75rem;
    color: var(--color-text-muted);
    text-transform: uppercase;
  }
}

.main-content {
  background: transparent;
}

.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.2s ease;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}
</style>

