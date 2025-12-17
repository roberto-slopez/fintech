# Fintech Multipaís - Sistema de Gestión de Créditos

Sistema completo para la gestión de solicitudes de crédito en múltiples países, desarrollado con **Go (Gin)** para el backend y **Vue 3 + PrimeVue** para el frontend.

## 🚀 Características Principales

- **Multipaís**: Soporte para N países con configuración dinámica
- **Clean Architecture**: Separación clara de responsabilidades
- **Tiempo Real**: WebSockets para actualizaciones en vivo
- **Procesamiento Asíncrono**: Cola de trabajos con PostgreSQL
- **Caché**: Redis/memoria para optimizar rendimiento
- **JWT Auth**: Autenticación segura con tokens
- **Escalable**: Diseñado para millones de registros

## 📋 Stack Tecnológico

| Componente | Tecnología |
|------------|------------|
| Backend | Go 1.22 + Gin |
| Frontend | Vue 3 + PrimeVue + Pinia |
| Base de Datos | PostgreSQL |
| Cache | Redis / In-Memory |
| WebSocket | Gorilla WebSocket |
| Autenticación | JWT |
| Deploy | Kubernetes |

## 🏗️ Arquitectura

```
┌─────────────────────────────────────────────────────────────────┐
│                        FRONTEND (Vue 3)                          │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐           │
│  │  Views   │ │Components│ │  Stores  │ │ Services │           │
│  └──────────┘ └──────────┘ └──────────┘ └──────────┘           │
└─────────────────────────────────────────────────────────────────┘
                              │
                    ┌─────────▼─────────┐
                    │   API Gateway     │ (Gin Router)
                    │   + WebSocket     │
                    └─────────┬─────────┘
                              │
┌─────────────────────────────▼───────────────────────────────────┐
│                      INTERFACES LAYER                            │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐           │
│  │ Handlers │ │Middleware│ │  Router  │ │ WebSocket│           │
│  └──────────┘ └──────────┘ └──────────┘ └──────────┘           │
└─────────────────────────────────────────────────────────────────┘
                              │
┌─────────────────────────────▼───────────────────────────────────┐
│                     APPLICATION LAYER                            │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐                        │
│  │ UseCases │ │   DTOs   │ │Validators│                        │
│  └──────────┘ └──────────┘ └──────────┘                        │
└─────────────────────────────────────────────────────────────────┘
                              │
┌─────────────────────────────▼───────────────────────────────────┐
│                       DOMAIN LAYER                               │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐                        │
│  │ Entities │ │  Rules   │ │Interfaces│  (Core Business Logic)  │
│  └──────────┘ └──────────┘ └──────────┘                        │
└─────────────────────────────────────────────────────────────────┘
                              │
┌─────────────────────────────▼───────────────────────────────────┐
│                   INFRASTRUCTURE LAYER                           │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐           │
│  │PostgreSQL│ │  Redis   │ │  Queue   │ │ Banking  │           │
│  └──────────┘ └──────────┘ └──────────┘ └──────────┘           │
└─────────────────────────────────────────────────────────────────┘
```

## 📦 Instalación Rápida

### Prerrequisitos

- Go 1.22+
- Node.js 20+
- PostgreSQL (o usar la conexión Neon proporcionada)

### 1. Clonar y configurar

```bash
# Clonar repositorio
git clone <repository-url>
cd fintech

# Configurar variables de entorno
cp backend/.env.example backend/.env
# Editar backend/.env con tus credenciales

# Instalar dependencias
make install
```

### 2. Ejecutar migraciones

```bash
# Las migraciones crean todas las tablas y datos iniciales
make migrate
```

### 3. Iniciar en desarrollo

```bash
# Inicia backend y frontend simultáneamente
make run

# O por separado:
make run-backend  # Puerto 8080
make run-frontend # Puerto 5173
```

### 4. Acceder

- **Frontend**: http://localhost:5173
- **API**: http://localhost:8080/api/v1
- **Usuario demo**: `admin@fintech.com` / `admin123`

## 📊 Modelo de Datos

### Tablas Principales

```
┌─────────────────┐     ┌─────────────────┐
│    countries    │     │ document_types  │
├─────────────────┤     ├─────────────────┤
│ id              │◄────│ country_id      │
│ code (ES,MX...) │     │ code (DNI,CURP) │
│ name            │     │ validation_regex│
│ currency        │     └─────────────────┘
│ config (JSON)   │
└────────┬────────┘
         │
         │      ┌─────────────────────┐
         │      │  country_rules      │
         ├──────│ country_id          │
         │      │ rule_type           │
         │      │ config (JSON)       │
         │      └─────────────────────┘
         │
         │      ┌─────────────────────┐
         └──────│ banking_providers   │
                │ country_id          │
                │ config (JSON)       │
                └──────────┬──────────┘
                           │
┌──────────────────────────▼──────────────────────────┐
│              credit_applications                     │
├──────────────────────────────────────────────────────┤
│ id, country_id, full_name, document_type/number     │
│ email, phone, requested_amount, monthly_income       │
│ status, requires_review, validation_results (JSON)   │
│ risk_score, application_date, created_at, updated_at │
└──────────────────────────────────────────────────────┘
         │
         ├──────► banking_info (1:1)
         ├──────► state_transitions (1:N)
         └──────► audit_logs (1:N)
```

### Índices para Escalabilidad

```sql
-- Índice compuesto para filtrado frecuente (país + estado)
CREATE INDEX idx_applications_country_status ON credit_applications(country_id, status);

-- Índice para búsqueda por documento
CREATE INDEX idx_applications_document ON credit_applications(country_id, document_number);

-- Índice para ordenamiento por fecha (paginación)
CREATE INDEX idx_applications_created ON credit_applications(created_at DESC);

-- Índice para solicitudes que requieren revisión
CREATE INDEX idx_applications_review ON credit_applications(requires_review, status) 
    WHERE requires_review = true;

-- Índice de texto para búsqueda por nombre (trigrams)
CREATE INDEX idx_applications_name_trgm ON credit_applications 
    USING GIN (full_name gin_trgm_ops);
```

### Estrategias de Escalabilidad

1. **Particionamiento por fecha**: Para millones de registros, particionar por `application_date`
2. **Archivado**: Mover solicitudes antiguas (>1 año) a tablas de archivo
3. **Read Replicas**: Separar lectura de escritura
4. **Sharding**: Por país para distribución geográfica

## 🔧 Configuración de Países

El sistema soporta **N países** de forma dinámica. Cada país tiene:

- Tipos de documento válidos (DNI, CURP, CPF, etc.)
- Reglas de validación configurables
- Proveedores bancarios específicos
- Límites de montos y configuración financiera

### Agregar un nuevo país

1. Insertar en tabla `countries`
2. Agregar tipos de documento en `document_types`
3. Configurar reglas en `country_rules`
4. Agregar proveedor bancario en `banking_providers`

## 🔒 Seguridad

- **JWT**: Tokens de acceso (15 min) y refresh (7 días)
- **Roles**: ADMIN, ANALYST, OPERATOR, VIEWER
- **CORS**: Orígenes configurables
- **PII**: Datos sensibles no expuestos en logs
- **Webhooks**: Verificación HMAC de firma

## 📡 Tiempo Real (WebSocket)

El frontend recibe actualizaciones en tiempo real:

```javascript
// Tipos de mensajes
{
  type: 'application_created' | 'application_updated' | 'status_changed',
  data: { ... },
  country_id: 'uuid',
  timestamp: '2024-01-01T00:00:00Z'
}
```

## ⚡ Cola de Trabajos

Procesamiento asíncrono usando PostgreSQL como cola:

- **Tipos de jobs**: DOCUMENT_VALIDATION, BANKING_INFO_FETCH, RISK_EVALUATION, NOTIFICATION
- **Workers**: Configurables (por defecto 5)
- **Retry**: Backoff exponencial con máximo 3 reintentos
- **Triggers**: Automáticos al crear/actualizar solicitudes

## 🗄️ Estrategia de Caché

| Entidad | TTL | Invalidación |
|---------|-----|--------------|
| Countries | 1 hora | Manual |
| Application | 5 min | Al actualizar |
| Rules | 30 min | Manual |

## 🐳 Docker

```bash
# Construir imágenes
make docker-build

# Ejecutar con docker-compose
make docker-up

# Ver logs
make docker-logs
```

## ☸️ Kubernetes

```bash
# Desplegar
make k8s-deploy

# Ver estado
make k8s-status

# Ver logs
make k8s-logs
```

### Componentes desplegados

- **API**: 3 réplicas, HPA (3-10)
- **Worker**: 2 réplicas, HPA (2-8)
- **Frontend**: 2 réplicas
- **Ingress**: NGINX con TLS

## 📝 API Endpoints

### Autenticación
- `POST /api/v1/auth/login` - Iniciar sesión
- `POST /api/v1/auth/register` - Registrar usuario
- `POST /api/v1/auth/refresh` - Refrescar token
- `GET /api/v1/auth/me` - Usuario actual

### Países
- `GET /api/v1/countries` - Listar países
- `GET /api/v1/countries/:code` - Detalles de país
- `GET /api/v1/countries/:code/document-types` - Tipos de documento
- `GET /api/v1/countries/:code/rules` - Reglas (protegido)

### Solicitudes
- `POST /api/v1/applications` - Crear solicitud
- `GET /api/v1/applications` - Listar con filtros
- `GET /api/v1/applications/:id` - Obtener por ID
- `PATCH /api/v1/applications/:id/status` - Actualizar estado
- `GET /api/v1/applications/:id/history` - Historial

### Webhooks
- `POST /api/v1/webhooks/:source` - Recibir webhook

## 🧪 Testing

```bash
# Backend
make test-backend

# Frontend
make test-frontend

# Cobertura
make test-coverage
```

## 📁 Estructura del Proyecto

```
fintech/
├── backend/
│   ├── cmd/
│   │   ├── api/            # Entry point API
│   │   └── worker/         # Entry point Worker
│   ├── internal/
│   │   ├── domain/         # Entidades y reglas de negocio
│   │   │   ├── entity/
│   │   │   ├── repository/ # Interfaces
│   │   │   └── service/    # Interfaces de servicios
│   │   ├── application/    # Casos de uso
│   │   │   └── usecase/
│   │   ├── infrastructure/ # Implementaciones
│   │   │   ├── config/
│   │   │   ├── database/
│   │   │   ├── cache/
│   │   │   ├── queue/
│   │   │   ├── persistence/
│   │   │   └── logger/
│   │   └── interfaces/     # Adaptadores de entrada
│   │       ├── http/
│   │       └── websocket/
│   ├── migrations/
│   └── config/
├── frontend/
│   ├── src/
│   │   ├── assets/
│   │   ├── components/
│   │   ├── composables/
│   │   ├── layouts/
│   │   ├── router/
│   │   ├── services/
│   │   ├── stores/
│   │   ├── types/
│   │   └── views/
│   └── public/
├── k8s/
│   ├── deployments/
│   ├── services/
│   ├── configmap.yaml
│   ├── secrets.yaml
│   └── ingress.yaml
├── Makefile
└── README.md
```

## 🤝 Supuestos y Decisiones

1. **PostgreSQL como cola**: Simplicidad vs Redis/RabbitMQ. Para volumen alto, migrar a Redis Streams.
2. **Caché en memoria**: Fallback cuando Redis no está disponible.
3. **Validación de documentos**: Regex básico. En producción, integrar servicios externos.
4. **Proveedores bancarios simulados**: La integración real requiere credenciales y contratos.
5. **Sin i18n completo**: El frontend está en español. Extensible con vue-i18n.

## 📄 Licencia

MIT

