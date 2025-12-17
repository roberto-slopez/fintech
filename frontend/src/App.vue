<script setup lang="ts">
import { onMounted, onUnmounted } from 'vue'
import { useAuthStore } from '@/stores/auth'
import { useWebSocket } from '@/composables/useWebSocket'
import Toast from 'primevue/toast'
import ConfirmDialog from 'primevue/confirmdialog'

const authStore = useAuthStore()
const { connect, disconnect } = useWebSocket()

onMounted(async () => {
  // Try to restore session
  await authStore.initializeAuth()
  
  // Connect to WebSocket
  connect()
})

onUnmounted(() => {
  disconnect()
})
</script>

<template>
  <Toast position="top-right" />
  <ConfirmDialog />
  <router-view />
</template>

<style>
#app {
  min-height: 100vh;
}
</style>

