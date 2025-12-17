import { ref, onUnmounted } from 'vue'
import { useApplicationsStore } from '@/stores/applications'
import { useToast } from 'primevue/usetoast'

const socket = ref<WebSocket | null>(null)
const isConnected = ref(false)
const reconnectAttempts = ref(0)
const maxReconnectAttempts = 5

let reconnectTimeout: number | null = null
let pingInterval: number | null = null

export function useWebSocket() {
  const applicationsStore = useApplicationsStore()
  const toast = useToast()

  function connect() {
    if (socket.value?.readyState === WebSocket.OPEN) {
      return
    }

    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const wsUrl = `${protocol}//${window.location.host}/ws`

    try {
      socket.value = new WebSocket(wsUrl)

      socket.value.onopen = () => {
        console.log('WebSocket connected')
        isConnected.value = true
        reconnectAttempts.value = 0

        // Start ping interval
        pingInterval = window.setInterval(() => {
          if (socket.value?.readyState === WebSocket.OPEN) {
            socket.value.send(JSON.stringify({ type: 'ping' }))
          }
        }, 30000)
      }

      socket.value.onmessage = (event) => {
        try {
          const message = JSON.parse(event.data)
          handleMessage(message)
        } catch (e) {
          console.error('Failed to parse WebSocket message:', e)
        }
      }

      socket.value.onclose = () => {
        console.log('WebSocket disconnected')
        isConnected.value = false
        cleanup()
        scheduleReconnect()
      }

      socket.value.onerror = (error) => {
        console.error('WebSocket error:', error)
        isConnected.value = false
      }
    } catch (e) {
      console.error('Failed to connect WebSocket:', e)
      scheduleReconnect()
    }
  }

  function disconnect() {
    cleanup()
    if (socket.value) {
      socket.value.close()
      socket.value = null
    }
    isConnected.value = false
  }

  function cleanup() {
    if (pingInterval) {
      clearInterval(pingInterval)
      pingInterval = null
    }
    if (reconnectTimeout) {
      clearTimeout(reconnectTimeout)
      reconnectTimeout = null
    }
  }

  function scheduleReconnect() {
    if (reconnectAttempts.value >= maxReconnectAttempts) {
      console.log('Max reconnect attempts reached')
      return
    }

    const delay = Math.min(1000 * Math.pow(2, reconnectAttempts.value), 30000)
    reconnectAttempts.value++

    console.log(`Scheduling reconnect in ${delay}ms (attempt ${reconnectAttempts.value})`)
    reconnectTimeout = window.setTimeout(connect, delay)
  }

  function handleMessage(message: any) {
    switch (message.type) {
      case 'pong':
        // Server acknowledged ping
        break

      case 'application_created':
        applicationsStore.handleRealtimeUpdate(message)
        toast.add({
          severity: 'info',
          summary: 'Nueva Solicitud',
          detail: `Se ha creado una nueva solicitud de crédito`,
          life: 5000
        })
        break

      case 'application_updated':
      case 'status_changed':
        applicationsStore.handleRealtimeUpdate(message)
        toast.add({
          severity: 'info',
          summary: 'Actualización',
          detail: `Una solicitud ha sido actualizada`,
          life: 3000
        })
        break

      case 'notification':
        toast.add({
          severity: message.data?.severity || 'info',
          summary: message.data?.title || 'Notificación',
          detail: message.data?.message || '',
          life: 5000
        })
        break

      default:
        console.log('Unknown message type:', message.type)
    }
  }

  function send(data: any) {
    if (socket.value?.readyState === WebSocket.OPEN) {
      socket.value.send(JSON.stringify(data))
    }
  }

  onUnmounted(() => {
    cleanup()
  })

  return {
    socket,
    isConnected,
    connect,
    disconnect,
    send
  }
}

