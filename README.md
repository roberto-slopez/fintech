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

- Go 1.22+ https://go.dev/dl/
- Node.js 20+ https://nodejs.org/en/download
- PostgreSQL (o usar la conexión Neon proporcionada) https://neon.new/ `npx get-db --yes`

### 1. Clonar y configurar

```bash
# Clonar repositorio
git clone <repository-url>
cd fintech

# Configurar variables de entorno
cp .env.example backend/.env
# Editar backend/.env con tus credenciales

# Instalar dependencias
make install
# comandos manuales 
cd backend && go mod download && go mod tidy
cd frontend && pnpm install # Puede usar npm sin problemas 
```

### 2. Ejecutar migraciones

```bash
# Las migraciones crean todas las tablas y datos iniciales
make migrate
# comandos manuales
cd backend && go run cmd/migrate/main.go up
```

### 3. Iniciar en desarrollo

```bash
# Inicia backend y frontend simultáneamente
make run
# comandos manuales
go run cmd/api/main.go  # Puerto 8080
pnpm run dev # Puerto 5173 tambien se puede utlizar npm si se instalo con npm
```

### 3.1 Worker

Iniciar los workers

```bash
go run cmd\worker\main.go
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

## 📈 Escalabilidad y Manejo de Grandes Volúmenes de Datos

El sistema está diseñado para manejar **millones de solicitudes de crédito** con las siguientes estrategias:

### Índices Recomendados

```sql
-- ═══════════════════════════════════════════════════════════════════
-- ÍNDICES PARA CONSULTAS FRECUENTES
-- ═══════════════════════════════════════════════════════════════════

-- 1. Índice compuesto para filtrado por país + estado (CONSULTA MÁS FRECUENTE)
-- Usado en: GET /applications?country=ES&status=PENDING
-- Cardinalidad estimada: Alta selectividad
CREATE INDEX idx_applications_country_status 
    ON credit_applications(country_id, status);

-- 2. Índice para búsqueda por documento (validación de duplicados)
-- Usado en: Verificar si ya existe solicitud para el documento
CREATE INDEX idx_applications_document 
    ON credit_applications(country_id, document_number);

-- 3. Índice para ordenamiento por fecha DESC (paginación)
-- Usado en: Listados ordenados por fecha más reciente
CREATE INDEX idx_applications_created 
    ON credit_applications(created_at DESC);

-- 4. Índice para filtrado por fecha de aplicación
-- Usado en: Reportes por rango de fechas
CREATE INDEX idx_applications_date 
    ON credit_applications(application_date DESC);

-- 5. Índice parcial para solicitudes que requieren revisión
-- Usado en: Dashboard de analistas (solo ~5% de registros)
CREATE INDEX idx_applications_review 
    ON credit_applications(requires_review, status) 
    WHERE requires_review = true;

-- 6. Índice GIN para búsqueda de texto (nombre del solicitante)
-- Usado en: Búsqueda fuzzy por nombre
-- Requiere extensión: pg_trgm
CREATE INDEX idx_applications_name_trgm 
    ON credit_applications USING GIN (full_name gin_trgm_ops);

-- 7. Índice para filtrado por rango de montos
-- Usado en: Reportes financieros
CREATE INDEX idx_applications_amount 
    ON credit_applications(requested_amount);

-- ═══════════════════════════════════════════════════════════════════
-- ÍNDICES PARA COLA DE TRABAJOS
-- ═══════════════════════════════════════════════════════════════════

-- Índice para obtener trabajos pendientes (workers)
-- Crítico para rendimiento de la cola
CREATE INDEX idx_jobs_pending 
    ON jobs_queue(status, priority DESC, scheduled_at ASC) 
    WHERE status IN ('PENDING', 'RETRYING');

-- Índice para limpieza de trabajos completados
CREATE INDEX idx_jobs_completed 
    ON jobs_queue(completed_at) 
    WHERE status IN ('COMPLETED', 'FAILED');
```

### Consultas Críticas y Optimización

| Consulta | Frecuencia | Índice Usado | Tiempo Esperado |
|----------|------------|--------------|-----------------|
| Listar por país + estado | Muy alta | `idx_applications_country_status` | < 10ms |
| Buscar por documento | Alta | `idx_applications_document` | < 5ms |
| Listar paginado por fecha | Alta | `idx_applications_created` | < 20ms |
| Buscar por nombre | Media | `idx_applications_name_trgm` | < 50ms |
| Solicitudes en revisión | Media | `idx_applications_review` | < 10ms |
| Dequeue trabajo | Muy alta | `idx_jobs_pending` | < 5ms |

### Estrategias de Particionamiento

Para manejar millones de registros, se recomienda particionar la tabla principal:

```sql
-- ═══════════════════════════════════════════════════════════════════
-- PARTICIONAMIENTO POR RANGO DE FECHAS
-- Recomendado cuando: > 10 millones de registros
-- ═══════════════════════════════════════════════════════════════════

-- 1. Crear tabla particionada
CREATE TABLE credit_applications_partitioned (
    id UUID NOT NULL,
    country_id UUID NOT NULL,
    full_name VARCHAR(200) NOT NULL,
    -- ... otros campos ...
    application_date TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
) PARTITION BY RANGE (application_date);

-- 2. Crear particiones por año/mes
CREATE TABLE applications_2024_q1 PARTITION OF credit_applications_partitioned
    FOR VALUES FROM ('2024-01-01') TO ('2024-04-01');

CREATE TABLE applications_2024_q2 PARTITION OF credit_applications_partitioned
    FOR VALUES FROM ('2024-04-01') TO ('2024-07-01');

-- 3. Automatizar creación de particiones con pg_partman
-- SELECT partman.create_parent('public.credit_applications_partitioned', 
--                               'application_date', 'time', 'monthly');
```

```sql
-- ═══════════════════════════════════════════════════════════════════
-- PARTICIONAMIENTO POR PAÍS (Sharding lógico)
-- Recomendado cuando: Distribución geográfica de servidores
-- ═══════════════════════════════════════════════════════════════════

CREATE TABLE credit_applications_by_country (
    id UUID NOT NULL,
    country_id UUID NOT NULL,
    -- ... otros campos ...
) PARTITION BY LIST (country_id);

-- Partición para España
CREATE TABLE applications_es PARTITION OF credit_applications_by_country
    FOR VALUES IN ('uuid-de-españa');

-- Partición para México  
CREATE TABLE applications_mx PARTITION OF credit_applications_by_country
    FOR VALUES IN ('uuid-de-mexico');
```

### Estrategias de Archivado

```sql
-- ═══════════════════════════════════════════════════════════════════
-- ARCHIVADO DE REGISTROS ANTIGUOS
-- Mover solicitudes > 2 años a tabla de archivo
-- ═══════════════════════════════════════════════════════════════════

-- 1. Crear tabla de archivo (sin índices pesados)
CREATE TABLE credit_applications_archive (
    LIKE credit_applications INCLUDING ALL
);

-- 2. Procedimiento de archivado mensual
CREATE OR REPLACE FUNCTION archive_old_applications() RETURNS void AS $$
BEGIN
    -- Mover a archivo
    INSERT INTO credit_applications_archive
    SELECT * FROM credit_applications
    WHERE application_date < NOW() - INTERVAL '2 years'
    AND status IN ('APPROVED', 'REJECTED', 'CANCELLED', 'DISBURSED');
    
    -- Eliminar de tabla principal
    DELETE FROM credit_applications
    WHERE application_date < NOW() - INTERVAL '2 years'
    AND status IN ('APPROVED', 'REJECTED', 'CANCELLED', 'DISBURSED');
    
    -- Actualizar estadísticas
    ANALYZE credit_applications;
END;
$$ LANGUAGE plpgsql;

-- 3. Programar ejecución mensual (con pg_cron)
-- SELECT cron.schedule('archive-monthly', '0 2 1 * *', 
--                      'SELECT archive_old_applications()');
```

### Evitar Cuellos de Botella

| Problema | Solución Implementada |
|----------|----------------------|
| **Bloqueos en la cola** | `FOR UPDATE SKIP LOCKED` para concurrencia sin bloqueos |
| **Escrituras frecuentes** | Batch inserts, conexiones pooled |
| **Lecturas pesadas** | Caché con TTL, índices parciales |
| **Conteo de registros** | Tablas de estadísticas pre-calculadas |
| **JOIN pesados** | Desnormalización selectiva (validation_results JSONB) |
| **Búsquedas de texto** | Índice GIN con pg_trgm |

### Métricas de Escalabilidad Esperada

```
┌─────────────────────────────────────────────────────────────────────┐
│                    CAPACIDAD ESTIMADA                                │
├─────────────────────────────────────────────────────────────────────┤
│  Registros totales:     10+ millones                                │
│  Inserciones/segundo:   1,000+                                      │
│  Lecturas/segundo:      10,000+                                     │
│  Workers concurrentes:  10-50                                       │
│  Latencia P99 lectura:  < 100ms                                     │
│  Latencia P99 escritura: < 200ms                                    │
└─────────────────────────────────────────────────────────────────────┘
```

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

## 🏦 Integración con Proveedores Bancarios por País

El sistema implementa una arquitectura extensible para integrarse con diferentes proveedores bancarios según el país de la solicitud.

### Arquitectura de Proveedores

```
┌─────────────────────────────────────────────────────────────────────┐
│                      BANKING PROVIDER SYSTEM                         │
└─────────────────────────────────────────────────────────────────────┘
                                   │
        ┌──────────────────────────┼──────────────────────────┐
        │                          │                          │
        ▼                          ▼                          ▼
┌───────────────┐          ┌───────────────┐          ┌───────────────┐
│  ESPAÑA (ES)  │          │  MÉXICO (MX)  │          │ COLOMBIA (CO) │
│   Equifax     │          │Buró de Crédito│          │ DataCrédito   │
│   (OAUTH2)    │          │   (API_KEY)   │          │   (API_KEY)   │
└───────────────┘          └───────────────┘          └───────────────┘
        │                          │                          │
        ▼                          ▼                          ▼
┌───────────────┐          ┌───────────────┐          ┌───────────────┐
│  BRASIL (BR)  │          │ PORTUGAL (PT) │          │  ITALIA (IT)  │
│Serasa Experian│          │Banco de Portugal│        │     CRIF      │
│   (OAUTH2)    │          │   (OAUTH2)    │          │   (API_KEY)   │
└───────────────┘          └───────────────┘          └───────────────┘
                                   │
                                   ▼
                    ┌─────────────────────────────┐
                    │   BankingInfoResponse       │
                    │   (Formato Normalizado)     │
                    │   - credit_score            │
                    │   - total_debt              │
                    │   - available_credit        │
                    │   - payment_history         │
                    │   - bank_accounts           │
                    │   - active_loans            │
                    │   - months_employed         │
                    └─────────────────────────────┘
```

### Tipos de Proveedores Soportados

| Tipo | Descripción | Ejemplo |
|------|-------------|---------|
| `CREDIT_BUREAU` | Burós de crédito tradicionales | Equifax, Serasa, CRIF |
| `BANK_API` | APIs bancarias directas | APIs de bancos específicos |
| `OPEN_BANKING` | Plataformas Open Banking | PSD2 en Europa |
| `AGGREGATOR` | Agregadores financieros | Plaid, Belvo |

### Proveedores Configurados por País

```sql
-- Los proveedores se configuran en la tabla banking_providers
| País     | Proveedor           | Tipo         | Auth     |
|----------|---------------------|--------------|----------|
| España   | Equifax España      | CREDIT_BUREAU| OAUTH2   |
| México   | Buró de Crédito     | CREDIT_BUREAU| API_KEY  |
| Colombia | DataCrédito         | CREDIT_BUREAU| API_KEY  |
| Brasil   | Serasa Experian     | CREDIT_BUREAU| OAUTH2   |
| Portugal | Banco de Portugal   | CREDIT_BUREAU| OAUTH2   |
| Italia   | CRIF Italia         | CREDIT_BUREAU| API_KEY  |
```

### Configuración del Proveedor

Cada proveedor tiene configuración flexible en formato JSON:

```json
{
  "base_url": "https://api.equifax.es",
  "timeout_seconds": 30,
  "retry_attempts": 3,
  "retry_delay_ms": 1000,
  "rate_limit_per_min": 100,
  "cache_ttl_minutes": 60,
  "auth_type": "OAUTH2",
  "response_mapping": {
    "score_field": "credit_score",
    "debt_field": "total_debt"
  }
}
```

### Flujo de Obtención de Información Bancaria

```
1. Se crea una solicitud de crédito
         │
         ▼
2. Trigger PostgreSQL crea job BANKING_INFO_FETCH
         │
         ▼
3. Worker procesa el job
         │
         ▼
4. ProviderService.GetProviderForCountry()
   → Obtiene proveedor activo por país (ordenado por prioridad)
         │
         ▼
5. ProviderService.FetchBankingInfo()
   → Llama al API del proveedor (simulado en MVP)
   → Maneja timeout, retry y rate limiting
         │
         ▼
6. Normalización de respuesta a BankingInfoResponse
         │
         ▼
7. ProviderService.SaveBankingInfo()
   → Guarda en tabla banking_info
         │
         ▼
8. Se actualiza la solicitud y se crea job de validación
```

### Estructura del Código

```
backend/internal/
├── domain/entity/
│   └── banking_provider.go    # Entidades: BankingProvider, BankingInfoResponse
├── infrastructure/banking/
│   └── provider_service.go    # Servicio de integración con proveedores
└── infrastructure/queue/
    └── postgres_queue.go      # Worker que procesa BANKING_INFO_FETCH
```

### Componentes Principales

**1. Entidad BankingProvider** (`banking_provider.go`):
```go
type BankingProvider struct {
    ID          uuid.UUID
    CountryID   uuid.UUID
    Code        string           // Identificador único (ES_EQUIFAX, MX_BURO, etc.)
    Name        string
    Type        ProviderType     // CREDIT_BUREAU, BANK_API, OPEN_BANKING, AGGREGATOR
    IsActive    bool
    Priority    int              // Orden de preferencia si hay múltiples
    Config      ProviderConfig   // Configuración flexible
    Credentials map[string]string // Credenciales (no expuestas)
}
```

**2. Respuesta Normalizada** (`BankingInfoResponse`):
```go
type BankingInfoResponse struct {
    Success         bool
    ProviderCode    string
    CreditScore     *int      // Score crediticio (300-850)
    TotalDebt       *float64  // Deuda total
    AvailableCredit *float64  // Crédito disponible
    PaymentHistory  *string   // GOOD, REGULAR, BAD
    BankAccounts    int       // Número de cuentas
    ActiveLoans     int       // Préstamos activos
    MonthsEmployed  *int      // Meses de empleo
    RawData         map[string]interface{} // Datos crudos del proveedor
}
```

**3. ProviderService** (`provider_service.go`):
- `GetProviderForCountry()`: Selecciona el proveedor activo para un país
- `FetchBankingInfo()`: Obtiene información del proveedor (con manejo de errores)
- `SaveBankingInfo()`: Persiste la información bancaria normalizada

### Agregar un Nuevo Proveedor

1. **Insertar en base de datos**:
```sql
INSERT INTO banking_providers (country_id, code, name, type, config) VALUES
((SELECT id FROM countries WHERE code = 'AR'), 
 'AR_VERAZ', 
 'Veraz Argentina', 
 'CREDIT_BUREAU', 
 '{"base_url": "https://api.veraz.com.ar", "auth_type": "API_KEY"}'::jsonb);
```

2. **Implementar adaptador** (si el proveedor tiene formato diferente):
```go
// En provider_service.go, extender simulateProviderResponse
// o crear adaptadores específicos por proveedor
```

### Consideraciones de Producción

- **Credenciales**: Almacenadas encriptadas, nunca expuestas en logs
- **Rate Limiting**: Configurado por proveedor para respetar límites de API
- **Caché**: TTL configurable para evitar llamadas repetidas (24h por defecto)
- **Retry**: Backoff exponencial con máximo de intentos
- **Fallback**: Sistema de prioridad permite proveedores de respaldo

## 🔒 Seguridad

- **JWT**: Tokens de acceso (15 min) y refresh (7 días)
- **Roles**: ADMIN, ANALYST, OPERATOR, VIEWER
- **CORS**: Orígenes configurables
- **PII**: Datos sensibles no expuestos en logs
- **Webhooks**: Verificación HMAC-SHA256 de firma

## 🔗 Webhooks y Procesos Externos

El sistema implementa un sistema completo de webhooks bidireccional que permite tanto **recibir** eventos de sistemas externos como **enviar** notificaciones a endpoints configurados.

### Arquitectura de Webhooks

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         WEBHOOK SYSTEM                                       │
└─────────────────────────────────────────────────────────────────────────────┘

                    WEBHOOKS ENTRANTES (Recibir)
                    ════════════════════════════
                              
┌──────────────────┐    POST /webhooks/:source    ┌──────────────────────────┐
│ Sistemas         │ ─────────────────────────────▶│ WebhookHandler           │
│ Externos         │    X-Webhook-Signature       │                          │
│                  │    Content-Type: json        │ • Verificar firma HMAC   │
│ • banking_provider│                              │ • Guardar en webhook_events│
│ • payment_gateway│                              │ • Procesar asíncrono     │
│ • verification   │                              │                          │
└──────────────────┘                              └────────────┬─────────────┘
                                                               │
                                                               ▼
                                                  ┌──────────────────────────┐
                                                  │ Procesadores por Source  │
                                                  │                          │
                                                  │ banking_provider:        │
                                                  │  • credit_report_ready   │
                                                  │  • verification_complete │
                                                  │                          │
                                                  │ payment_gateway:         │
                                                  │  • payment_confirmed     │
                                                  │  • disbursement_complete │
                                                  └──────────────────────────┘

                    WEBHOOKS SALIENTES (Enviar)
                    ════════════════════════════

┌──────────────────┐                              ┌──────────────────────────┐
│ Eventos del      │                              │ Endpoints Externos       │
│ Sistema          │                              │ (por país)               │
│                  │    POST con firma HMAC       │                          │
│ • application.   │ ─────────────────────────────▶│ ES: webhook.example/ES   │
│   created        │    X-Webhook-Signature       │ MX: webhook.example/MX   │
│ • application.   │    X-Webhook-Event           │ CO: webhook.example/CO   │
│   approved       │    X-Webhook-ID              │ ...                      │
│ • application.   │    X-Webhook-Timestamp       │                          │
│   rejected       │                              │                          │
└──────────────────┘                              └──────────────────────────┘
```

### Webhooks Entrantes (Recibir de Sistemas Externos)

El sistema puede **recibir** webhooks de sistemas externos como proveedores bancarios, gateways de pago, o servicios de verificación.

**Endpoint:**
```
POST /api/v1/webhooks/:source
```

**Parámetro `source`:**

El parámetro `:source` identifica el sistema externo que envía el webhook. Valores soportados:

| Source | Descripción | Eventos Soportados |
|--------|-------------|-------------------|
| `banking_provider` | Proveedores bancarios (Equifax, Buró, etc.) | `credit_report_ready`, `verification_complete` |
| `payment_gateway` | Gateway de pagos | `payment_confirmed`, `disbursement_complete` |
| `verification` | Servicios de verificación de identidad | `identity_verified`, `document_validated` |

**Headers Requeridos:**
```http
Content-Type: application/json
X-Webhook-Signature: <HMAC-SHA256 del payload>
```

**Ejemplo de Payload Entrante:**
```json
{
  "event_type": "credit_report_ready",
  "application_id": "550e8400-e29b-41d4-a716-446655440000",
  "timestamp": "2024-01-15T10:30:00Z",
  "data": {
    "credit_score": 720,
    "report_id": "RPT-2024-001",
    "provider": "ES_EQUIFAX"
  }
}
```

**Respuesta:**
```json
{
  "success": true,
  "event_id": "123e4567-e89b-12d3-a456-426614174000",
  "message": "Event received and queued for processing"
}
```

**Flujo de Procesamiento:**
```
1. Recibir POST en /webhooks/:source
         │
         ▼
2. Verificar firma HMAC-SHA256 (si está configurada)
         │
         ▼
3. Parsear payload JSON
         │
         ▼
4. Guardar en tabla webhook_events (status: RECEIVED)
         │
         ▼
5. Retornar respuesta inmediata (202 Accepted)
         │
         ▼
6. Procesar evento asíncronamente según source:
   ├── banking_provider → processBankingProviderEvent()
   │   ├── credit_report_ready → Actualizar estado a VALIDATING
   │   └── verification_complete → Aprobar o rechazar según resultado
   │
   └── payment_gateway → processPaymentGatewayEvent()
       └── (extensible)
         │
         ▼
7. Actualizar webhook_events (status: PROCESSED o FAILED)
```

### Webhooks Salientes (Enviar a Sistemas Externos)

El sistema **envía** webhooks a endpoints configurados cuando ocurren eventos importantes.

**Tipos de Eventos Enviados:**

| Evento | Descripción | Trigger |
|--------|-------------|---------|
| `application.created` | Nueva solicitud creada | Al insertar en credit_applications |
| `application.updated` | Solicitud actualizada | Al actualizar datos de solicitud |
| `application.approved` | Solicitud aprobada | Cambio de estado a APPROVED |
| `application.rejected` | Solicitud rechazada | Cambio de estado a REJECTED |
| `application.disbursed` | Crédito desembolsado | Cambio de estado a DISBURSED |
| `banking_info.received` | Info bancaria recibida | Al completar job BANKING_INFO_FETCH |

**Configuración de Endpoints (por país):**

Los endpoints se configuran en la tabla `webhook_endpoints`:

```sql
-- Cada país puede tener sus propios endpoints
SELECT * FROM webhook_endpoints WHERE country_id = '<country_uuid>';

-- Resultado ejemplo:
| id  | country_id | url                                  | event_types                          |
|-----|------------|--------------------------------------|--------------------------------------|
| ... | ES         | https://webhook.example.com/fintech/ES | {application.created, application.approved} |
| ... | MX         | https://api.partner.mx/webhooks      | {application.created, application.rejected} |
```

**Headers Enviados:**
```http
POST /webhook-endpoint HTTP/1.1
Content-Type: application/json
X-Webhook-Signature: abc123def456...  (HMAC-SHA256)
X-Webhook-Event: application.created
X-Webhook-ID: 550e8400-e29b-41d4-a716-446655440000
X-Webhook-Timestamp: 2024-01-15T10:30:00Z
User-Agent: Fintech-Multipass-Webhook/1.0
```

**Ejemplo de Payload Enviado:**
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "event_type": "application.approved",
  "application_id": "123e4567-e89b-12d3-a456-426614174000",
  "country_code": "ES",
  "timestamp": "2024-01-15T10:30:00Z",
  "data": {
    "application_id": "123e4567-e89b-12d3-a456-426614174000",
    "status": "APPROVED",
    "status_reason": "All validations passed",
    "requested_amount": 15000.00,
    "requires_review": false
  }
}
```

### Verificación de Firma (Seguridad)

Todos los webhooks (entrantes y salientes) usan **HMAC-SHA256** para verificar la autenticidad:

```go
// Generar firma
signature := HMAC-SHA256(payload, secret_key)

// El receptor verifica comparando:
expected := HMAC-SHA256(received_payload, shared_secret)
valid := hmac.Equal(received_signature, expected)
```

**Configuración del Secret:**
```yaml
# backend/config/config.yaml
webhook:
  secret: "your-webhook-secret-key"
  timeout: 30s
  max_retries: 3
  retry_delay: 5s
```

### Modelo de Datos de Webhooks

```sql
-- Eventos de webhook recibidos
CREATE TABLE webhook_events (
    id UUID PRIMARY KEY,
    source VARCHAR(100) NOT NULL,      -- banking_provider, payment_gateway, etc.
    event_type VARCHAR(100) NOT NULL,  -- credit_report_ready, payment_confirmed
    payload JSONB NOT NULL,
    signature VARCHAR(255),
    status VARCHAR(20) DEFAULT 'RECEIVED',  -- RECEIVED, PROCESSED, FAILED
    error_message TEXT,
    processed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Endpoints configurados para webhooks salientes
CREATE TABLE webhook_endpoints (
    id UUID PRIMARY KEY,
    country_id UUID REFERENCES countries(id),
    url VARCHAR(500) NOT NULL,
    secret VARCHAR(255),
    event_types VARCHAR(100)[] NOT NULL,  -- Array de eventos suscritos
    is_active BOOLEAN DEFAULT true,
    max_retries INT DEFAULT 3,
    retry_delay_seconds INT DEFAULT 60,
    created_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ
);

-- Registro de entregas de webhooks salientes
CREATE TABLE webhook_deliveries (
    id UUID PRIMARY KEY,
    endpoint_id UUID REFERENCES webhook_endpoints(id),
    event_type VARCHAR(100) NOT NULL,
    payload JSONB NOT NULL,
    status VARCHAR(20) DEFAULT 'PENDING',  -- PENDING, SENT, FAILED
    http_status INT,
    response_body TEXT,
    attempts INT DEFAULT 0,
    last_attempt TIMESTAMPTZ,
    created_at TIMESTAMPTZ
);
```

### Estructura del Código

```
backend/internal/
├── infrastructure/webhook/
│   └── service.go              # WebhookService - envío de webhooks salientes
│       ├── DeliverWebhook()    # Enviar webhook a endpoint
│       ├── signPayload()       # Firmar con HMAC-SHA256
│       ├── GetEndpointsForEvent() # Obtener endpoints suscritos
│       └── PublishApplicationEvent() # Publicar evento de aplicación
│
├── interfaces/http/handler/
│   └── webhook_handler.go      # Handler para webhooks entrantes
│       ├── HandleIncoming()    # Recibir POST /webhooks/:source
│       ├── verifySignature()   # Verificar firma HMAC
│       ├── processEvent()      # Procesar evento asíncronamente
│       ├── processBankingProviderEvent() # Procesar eventos bancarios
│       └── processPaymentGatewayEvent()  # Procesar eventos de pago
│
└── infrastructure/queue/
    └── postgres_queue.go
        └── handleWebhookCall() # Worker para webhooks en cola
```

### Agregar un Nuevo Source de Webhook Entrante

1. **Agregar case en processEvent():**
```go
// webhook_handler.go
func (h *WebhookHandler) processEvent(event *entity.WebhookEvent) {
    switch event.Source {
    case "banking_provider":
        err = h.processBankingProviderEvent(ctx, event)
    case "payment_gateway":
        err = h.processPaymentGatewayEvent(ctx, event)
    case "new_source":  // ← Nuevo source
        err = h.processNewSourceEvent(ctx, event)
    }
}
```

2. **Implementar procesador específico:**
```go
func (h *WebhookHandler) processNewSourceEvent(ctx context.Context, event *entity.WebhookEvent) error {
    switch event.EventType {
    case "event_type_1":
        return h.handleEventType1(ctx, event.Payload)
    case "event_type_2":
        return h.handleEventType2(ctx, event.Payload)
    }
    return nil
}
```

### Probar Webhooks

**Enviar webhook de prueba (curl):**
```bash
# Calcular firma HMAC-SHA256
PAYLOAD='{"event_type":"credit_report_ready","application_id":"uuid-here"}'
SECRET="your-secret"
SIGNATURE=$(echo -n "$PAYLOAD" | openssl dgst -sha256 -hmac "$SECRET" | cut -d' ' -f2)

# Enviar webhook
curl -X POST http://localhost:8080/api/v1/webhooks/banking_provider \
  -H "Content-Type: application/json" \
  -H "X-Webhook-Signature: $SIGNATURE" \
  -d "$PAYLOAD"
```

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

## ⚡ Cola de Trabajos y Procesamiento Asíncrono

El sistema implementa una **cola de trabajos basada en PostgreSQL** para procesamiento asíncrono, diseñada para escalar horizontalmente con múltiples workers.

### ¿Por qué PostgreSQL como Cola?

| Aspecto | PostgreSQL | Redis/RabbitMQ |
|---------|------------|----------------|
| **Simplicidad** | ✅ Sin infraestructura adicional | ❌ Servicio separado |
| **Transaccionalidad** | ✅ ACID completo | ⚠️ Limitado |
| **Durabilidad** | ✅ Garantizada | ⚠️ Configurable |
| **Escalabilidad** | ⚠️ Buena (hasta ~10k jobs/seg) | ✅ Excelente |
| **Complejidad** | ✅ Baja | ❌ Alta |

**Decisión**: Para el MVP usamos PostgreSQL. Para alto volumen (>10k jobs/seg), migrar a Redis Streams o RabbitMQ.

### Arquitectura de la Cola

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         QUEUE SYSTEM                                         │
└─────────────────────────────────────────────────────────────────────────────┘

    PRODUCTORES                         COLA                        CONSUMIDORES
    ═══════════                    ═══════════════                  ════════════

┌──────────────┐                 ┌─────────────────┐              ┌─────────────┐
│   Trigger    │──INSERT───────▶│                 │              │  Worker 1   │
│  PostgreSQL  │                │   jobs_queue    │──DEQUEUE────▶│             │
└──────────────┘                │                 │              └─────────────┘
                                │  ┌───────────┐  │              
┌──────────────┐                │  │ PENDING   │  │              ┌─────────────┐
│     API      │──INSERT───────▶│  │ PROCESSING│  │──DEQUEUE────▶│  Worker 2   │
│   Handler    │                │  │ COMPLETED │  │              │             │
└──────────────┘                │  │ FAILED    │  │              └─────────────┘
                                │  │ RETRYING  │  │              
┌──────────────┐                │  └───────────┘  │              ┌─────────────┐
│   Webhook    │──INSERT───────▶│                 │──DEQUEUE────▶│  Worker N   │
│   Handler    │                │                 │              │             │
└──────────────┘                └─────────────────┘              └─────────────┘
                                                                        │
                                    FOR UPDATE SKIP LOCKED              │
                                    (Sin bloqueos entre workers)        │
                                                                        ▼
                                                                 ┌─────────────┐
                                                                 │  Handlers   │
                                                                 │             │
                                                                 │ • RiskEval  │
                                                                 │ • BankInfo  │
                                                                 │ • DocValid  │
                                                                 │ • Notify    │
                                                                 │ • Webhook   │
                                                                 └─────────────┘
```

### Modelo de Datos de la Cola

```sql
CREATE TABLE jobs_queue (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    type VARCHAR(50) NOT NULL,           -- Tipo de trabajo
    status VARCHAR(20) NOT NULL DEFAULT 'PENDING',
    priority INT NOT NULL DEFAULT 0,     -- Mayor = más prioritario
    payload JSONB NOT NULL,              -- Datos del trabajo
    result JSONB,                        -- Resultado (si completado)
    error_message TEXT,                  -- Error (si falló)
    attempts INT NOT NULL DEFAULT 0,     -- Intentos realizados
    max_attempts INT NOT NULL DEFAULT 3, -- Máximo de reintentos
    worker_id VARCHAR(100),              -- Worker que lo procesa
    scheduled_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

### Tipos de Trabajos (JobTypes)

| Tipo | Descripción | Trigger | Prioridad |
|------|-------------|---------|-----------|
| `DOCUMENT_VALIDATION` | Valida formato de documento de identidad | Al crear solicitud | 10 |
| `BANKING_INFO_FETCH` | Obtiene info del proveedor bancario | Al crear solicitud | 8 |
| `RISK_EVALUATION` | Evalúa riesgo crediticio | Al completar BANKING_INFO_FETCH | 10 |
| `NOTIFICATION` | Envía notificaciones (email/SMS) | Al cambiar estado | 5 |
| `AUDIT_LOG` | Crea registros de auditoría | En operaciones críticas | 3 |
| `WEBHOOK_CALL` | Llama webhooks externos | En eventos configurados | 5 |

### Cómo se Producen los Trabajos

**1. Automáticamente via Triggers PostgreSQL:**

```sql
-- Trigger: Al crear solicitud → encola DOCUMENT_VALIDATION + BANKING_INFO_FETCH
CREATE OR REPLACE FUNCTION on_application_created()
RETURNS TRIGGER AS $$
BEGIN
    -- Job de validación de documento
    INSERT INTO jobs_queue (type, priority, payload)
    VALUES (
        'DOCUMENT_VALIDATION',
        10,
        jsonb_build_object(
            'application_id', NEW.id,
            'country_id', NEW.country_id,
            'document_type', NEW.document_type,
            'document_number', NEW.document_number
        )
    );
    
    -- Job de obtención de info bancaria
    INSERT INTO jobs_queue (type, priority, payload)
    VALUES (
        'BANKING_INFO_FETCH',
        8,
        jsonb_build_object(
            'application_id', NEW.id,
            'country_id', NEW.country_id,
            'document_type', NEW.document_type,
            'document_number', NEW.document_number
        )
    );
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_application_created
    AFTER INSERT ON credit_applications
    FOR EACH ROW EXECUTE FUNCTION on_application_created();
```

**2. Programáticamente desde el código:**

```go
// Encolar trabajo manualmente
job := &entity.Job{
    Type:     entity.JobTypeRiskEvaluation,
    Priority: 10,
    Payload:  json.RawMessage(`{"application_id": "uuid-here"}`),
}
err := queue.Enqueue(ctx, job)

// Encolar con delay (para reintentos)
err := queue.EnqueueWithDelay(ctx, job, 60) // 60 segundos de delay
```

### Cómo se Consumen los Trabajos

**Dequeue con `FOR UPDATE SKIP LOCKED`** (concurrencia sin bloqueos):

```go
// Dequeue: Obtiene el siguiente trabajo disponible SIN bloquear otros workers
func (q *PostgresQueue) Dequeue(ctx context.Context, workerID string) (*entity.Job, error) {
    query := `
        UPDATE jobs_queue
        SET status = 'PROCESSING', 
            started_at = NOW(),
            worker_id = $1,
            attempts = attempts + 1
        WHERE id = (
            SELECT id FROM jobs_queue
            WHERE status IN ('PENDING', 'RETRYING')
            AND scheduled_at <= NOW()
            ORDER BY priority DESC, scheduled_at ASC
            FOR UPDATE SKIP LOCKED  -- ← CLAVE: No bloquea otros workers
            LIMIT 1
        )
        RETURNING id, type, payload, attempts, max_attempts
    `
    // ...
}
```

**Ciclo del Worker:**

```go
func (w *Worker) Start(ctx context.Context) {
    ticker := time.NewTicker(1 * time.Second)
    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            // 1. Obtener trabajo
            job, _ := w.queue.Dequeue(ctx, w.id)
            if job == nil {
                continue // No hay trabajos
            }
            
            // 2. Buscar handler registrado
            handler := w.queue.handlers[job.Type]
            
            // 3. Ejecutar con timeout
            jobCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
            err := handler(jobCtx, job)
            cancel()
            
            // 4. Marcar resultado
            if err != nil {
                w.queue.Fail(ctx, job.ID, err.Error())
            } else {
                w.queue.Complete(ctx, job.ID, nil)
            }
        }
    }
}
```

### Estrategia de Reintentos (Backoff Exponencial)

```go
func (q *PostgresQueue) Fail(ctx context.Context, jobID uuid.UUID, errorMsg string) error {
    var attempts, maxAttempts int
    // ... obtener intentos actuales ...
    
    if attempts < maxAttempts {
        // Reintento con backoff exponencial: 30s, 120s, 270s...
        delay := time.Duration(attempts * attempts * 30) * time.Second
        scheduledAt := time.Now().Add(delay)
        
        // Actualizar a RETRYING con nuevo scheduled_at
        query := `UPDATE jobs_queue SET status = 'RETRYING', scheduled_at = $2 WHERE id = $1`
        return q.db.Exec(ctx, query, jobID, scheduledAt)
    }
    
    // Sin más reintentos → FAILED
    query := `UPDATE jobs_queue SET status = 'FAILED', completed_at = NOW() WHERE id = $1`
    return q.db.Exec(ctx, query, jobID)
}
```

### Flujo Completo de un Trabajo

```
┌─────────┐     ┌─────────┐     ┌────────────┐     ┌───────────┐     ┌───────────┐
│ PENDING │────▶│PROCESSING│────▶│ COMPLETED  │     │  RETRYING │     │  FAILED   │
└─────────┘     └─────────┘     └────────────┘     └───────────┘     └───────────┘
     │               │                                    │                │
     │               │         (éxito)                    │                │
     │               ├────────────────────────────────────┘                │
     │               │                                                     │
     │               │         (error + attempts < max)                    │
     │               ├─────────────────────────────────────────────────────┤
     │               │                                                     │
     │               │         (error + attempts >= max)                   │
     │               └─────────────────────────────────────────────────────┘
     │
     │   (scheduled_at <= NOW)
     └─────────────────────────▶ Worker selecciona
```

### Configuración de Workers

```yaml
# config/config.yaml
queue:
  workers: 5              # Número de workers concurrentes
  poll_interval: 1s       # Intervalo de polling
  job_timeout: 5m         # Timeout por trabajo
  max_retries: 3          # Reintentos máximos
```

```go
// Iniciar workers
queue.StartWorkers(ctx, 5)

// Detener workers (graceful shutdown)
queue.StopWorkers()
```

### Monitoreo de la Cola

```go
// Obtener estadísticas
stats, _ := queue.Stats(ctx)
// Resultado: map[JobStatus]int64{
//   "PENDING": 42,
//   "PROCESSING": 3,
//   "COMPLETED": 1520,
//   "FAILED": 12,
//   "RETRYING": 5,
// }
```

```sql
-- Query para dashboard de monitoreo
SELECT 
    type,
    status,
    COUNT(*) as count,
    AVG(EXTRACT(EPOCH FROM (completed_at - created_at))) as avg_duration_secs
FROM jobs_queue
WHERE created_at > NOW() - INTERVAL '24 hours'
GROUP BY type, status
ORDER BY type, status;
```

## 🗄️ Estrategia de Caché

> **Resumen Ejecutivo:**
> - **Tecnología**: Redis (producción) + MemoryCache (fallback/desarrollo)
> - **Qué se cachea**: Países (1h), Solicitudes (5min), Reglas (30min)
> - **Invalidación**: TTL automático + invalidación explícita al actualizar
> - **Implementación**: `backend/internal/infrastructure/cache/cache.go`
> - **Uso**: `CountryUseCase` y `ApplicationUseCase` usan caché activamente

El sistema implementa una capa de caché con **Redis** como almacenamiento principal y **caché en memoria** como fallback.

### Arquitectura de Caché

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           CACHE SYSTEM                                       │
└─────────────────────────────────────────────────────────────────────────────┘

    REQUEST                   CACHE LAYER                      DATABASE
    ═══════                   ═══════════                      ════════

┌──────────┐             ┌─────────────────┐              ┌─────────────┐
│          │──GET───────▶│                 │              │             │
│  Client  │             │   CacheService  │              │  PostgreSQL │
│          │◀──RESPONSE──│                 │              │             │
└──────────┘             └────────┬────────┘              └──────┬──────┘
                                  │                              │
                         ┌────────▼────────┐                     │
                         │   Cache Hit?    │                     │
                         └────────┬────────┘                     │
                                  │                              │
                    ┌─────────────┴─────────────┐                │
                    │                           │                │
                ┌───▼───┐                   ┌───▼───┐            │
                │  HIT  │                   │ MISS  │────GET────▶│
                │       │                   │       │◀───DATA────│
                │Return │                   │ Cache │            │
                │ Data  │                   │ + Ret │            │
                └───────┘                   └───────┘            │
                                                                 │
                                                                 
┌─────────────────────────────────────────────────────────────────────────────┐
│                         IMPLEMENTACIONES                                     │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│   ┌───────────────────┐         ┌───────────────────┐                      │
│   │    RedisCache     │         │   MemoryCache     │                      │
│   │   (Producción)    │         │    (Fallback)     │                      │
│   │                   │         │                   │                      │
│   │ • Distribuido     │         │ • Local a proceso │                      │
│   │ • Persistente     │         │ • Sin dependencias│                      │
│   │ • Escalable       │         │ • Auto-limpieza   │                      │
│   └───────────────────┘         └───────────────────┘                      │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Qué se Cachea y Por Qué

| Entidad | Prefijo | TTL | Razón | Invalidación |
|---------|---------|-----|-------|--------------|
| **Countries** | `country:` | 1 hora | Datos estáticos, consultados frecuentemente | Manual (raro cambio) |
| **All Countries** | `countries:all` | 1 hora | Lista completa para dropdowns | Manual |
| **Application** | `app:` | 5 min | Lecturas frecuentes, escrituras moderadas | Al actualizar |
| **Rules** | `rules:` | 30 min | Configuración semi-estática | Manual |
| **User Session** | `user:` | 15 min | Datos de sesión | Al logout |

### Implementación del Caché

**Interface CacheService:**

```go
type CacheService interface {
    // Operaciones genéricas
    Get(ctx context.Context, key string, dest interface{}) error
    Set(ctx context.Context, key string, value interface{}, ttlSeconds int) error
    Delete(ctx context.Context, key string) error
    Exists(ctx context.Context, key string) (bool, error)
    Close() error

    // Métodos específicos del dominio (type-safe)
    GetApplication(ctx context.Context, id uuid.UUID) (*entity.CreditApplication, error)
    SetApplication(ctx context.Context, app *entity.CreditApplication) error
    InvalidateApplication(ctx context.Context, id uuid.UUID) error

    GetCountry(ctx context.Context, code string) (*entity.Country, error)
    SetCountry(ctx context.Context, country *entity.Country) error
    GetAllCountries(ctx context.Context) ([]entity.Country, error)
    SetAllCountries(ctx context.Context, countries []entity.Country) error
}
```

**Prefijos y TTLs definidos:**

```go
// Prefijos de cache
const (
    prefixApplication = "app:"
    prefixCountry     = "country:"
    prefixCountries   = "countries:all"
    prefixUser        = "user:"
    prefixRules       = "rules:"
)

// TTLs por defecto (en segundos)
const (
    ttlApplication = 300  // 5 minutos
    ttlCountry     = 3600 // 1 hora
    ttlRules       = 1800 // 30 minutos
)
```

### Estrategia de Invalidación

**1. Invalidación Explícita (al actualizar):**

```go
// En ApplicationUseCase.UpdateStatus()
func (uc *ApplicationUseCase) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
    // 1. Actualizar en base de datos
    err := uc.repo.UpdateStatus(ctx, id, status)
    if err != nil {
        return err
    }
    
    // 2. Invalidar caché
    err = uc.cache.InvalidateApplication(ctx, id)
    if err != nil {
        uc.log.Warn().Err(err).Msg("Failed to invalidate cache")
        // No fallar la operación por error de caché
    }
    
    return nil
}
```

**2. TTL (Time-To-Live):**

```go
// El dato expira automáticamente después del TTL
func (c *RedisCache) SetApplication(ctx context.Context, app *entity.CreditApplication) error {
    return c.Set(ctx, prefixApplication+app.ID.String(), app, ttlApplication) // 5 min
}
```

**3. Limpieza Automática (MemoryCache):**

```go
// Goroutine de limpieza cada minuto
func (c *MemoryCache) cleanup() {
    ticker := time.NewTicker(1 * time.Minute)
    for range ticker.C {
        c.mu.Lock()
        now := time.Now()
        for key, item := range c.data {
            if now.After(item.expiresAt) {
                delete(c.data, key)
            }
        }
        c.mu.Unlock()
    }
}
```

### Patrón Cache-Aside Implementado

```go
// Ejemplo en CountryUseCase.GetByCode()
func (uc *CountryUseCase) GetByCode(ctx context.Context, code string) (*entity.Country, error) {
    // 1. Intentar obtener del caché
    country, err := uc.cache.GetCountry(ctx, code)
    if err == nil {
        return country, nil // ✅ Cache HIT
    }
    
    // 2. Cache MISS → consultar base de datos
    country, err = uc.repo.GetByCode(ctx, code)
    if err != nil {
        return nil, err
    }
    
    // 3. Guardar en caché para próximas consultas
    if err := uc.cache.SetCountry(ctx, country); err != nil {
        uc.log.Warn().Err(err).Msg("Failed to cache country")
    }
    
    return country, nil
}
```

### Configuración

```yaml
# config/config.yaml
cache:
  type: redis          # redis | memory
  host: localhost
  port: 6379
  password: ""
  db: 0
  ttl: 300             # TTL por defecto en segundos
```

### Redis vs Memory Cache

| Aspecto | RedisCache | MemoryCache |
|---------|------------|-------------|
| **Uso** | Producción | Desarrollo / Fallback |
| **Distribución** | ✅ Compartido entre instancias | ❌ Local por proceso |
| **Persistencia** | ✅ Sobrevive reinicios | ❌ Se pierde al reiniciar |
| **Escalabilidad** | ✅ Cluster Redis | ❌ Limitado |
| **Latencia** | ~1ms (red local) | ~0.01ms |
| **Configuración** | Requiere servidor Redis | Sin dependencias |

### Fallback Automático

```go
// En la inicialización de la aplicación
func initCache(cfg config.CacheConfig) cache.CacheService {
    // Intentar conectar a Redis
    redisCache, err := cache.NewRedisCache(cfg)
    if err == nil {
        log.Info().Msg("Connected to Redis cache")
        return redisCache
    }
    
    // Fallback a memoria si Redis no está disponible
    log.Warn().Err(err).Msg("Redis unavailable, using memory cache")
    return cache.NewMemoryCache()
}
```

### Métricas de Caché Recomendadas

```go
// Para monitoreo en producción
type CacheMetrics struct {
    Hits       int64   // Consultas exitosas desde caché
    Misses     int64   // Consultas que fueron a DB
    HitRate    float64 // Hits / (Hits + Misses)
    AvgLatency time.Duration
}

// Hit rate esperado por entidad:
// Countries:    ~99% (datos muy estáticos)
// Applications: ~60-80% (lecturas frecuentes de mismas apps)
// Rules:        ~95% (cambios poco frecuentes)
```

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
- `POST /api/v1/webhooks/:source` - Recibir webhook de sistema externo
  - `:source` = identificador del sistema (ej: `banking_provider`, `payment_gateway`, `verification`)
  - Headers: `X-Webhook-Signature` (HMAC-SHA256), `Content-Type: application/json`
  - Ver sección "Webhooks y Procesos Externos" para detalles completos

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

## 🤝 Supuestos del Sistema

### Supuestos de Negocio

| Supuesto | Descripción | Impacto |
|----------|-------------|---------|
| **Operación multipaís** | Cada país tiene regulaciones, documentos y proveedores bancarios distintos | Arquitectura modular por país |
| **Volumen de solicitudes** | Sistema diseñado para millones de solicitudes | Índices, particionamiento, caché |
| **Validación de documentos** | Formato validado con regex; en producción, integrar servicios externos (RENIEC, SAT, etc.) | Simplificación para MVP |
| **Proveedores bancarios** | APIs simuladas; la integración real requiere contratos y credenciales | Abstracción con interfaces |
| **Flujo de estados** | Las solicitudes siguen un flujo lineal (PENDING → VALIDATING → APPROVED/REJECTED) | Extensible a flujos más complejos |
| **Usuarios internos** | El sistema es para operadores internos, no para solicitantes directos | Sin registro público |

### Supuestos Técnicos

| Supuesto | Descripción |
|----------|-------------|
| **Base de datos disponible** | PostgreSQL siempre accesible; sin modo offline |
| **Conectividad de red** | Workers y API en la misma red; latencia baja |
| **Zona horaria** | Todas las fechas en UTC; conversión en frontend |
| **Idioma** | Backend en inglés, frontend en español |
| **Concurrencia** | Máximo ~50 workers simultáneos por instancia |

## 🔧 Decisiones Técnicas

### Backend: Go + Gin

| Aspecto | Decisión | Alternativas Consideradas | Razón |
|---------|----------|---------------------------|-------|
| **Lenguaje** | Go 1.22 | Node.js, Java, Python | Alto rendimiento, bajo consumo de memoria, concurrencia nativa |
| **Framework** | Gin | Echo, Fiber, Chi | Madurez, documentación, middleware ecosystem |
| **ORM** | SQL directo (pgx) | GORM, sqlx | Control total sobre queries, mejor rendimiento |

```
Ventajas de Go para este caso:
✅ Goroutines para workers concurrentes
✅ Compilación a binario único (fácil deployment)
✅ Bajo uso de memoria (~20MB por instancia)
✅ Tipado estático reduce errores en runtime
```

### Frontend: Vue 3 + PrimeVue

| Aspecto | Decisión | Alternativas | Razón |
|---------|----------|--------------|-------|
| **Framework** | Vue 3 | React, Angular, Svelte | Composition API, curva de aprendizaje suave |
| **UI Library** | PrimeVue | Vuetify, Element Plus | Componentes enterprise-ready, DataTable potente |
| **State** | Pinia | Vuex, Composables | API moderna, TypeScript nativo |
| **Build** | Vite | Webpack, Rollup | HMR instantáneo, builds rápidos |

### Base de Datos: PostgreSQL

| Aspecto | Decisión | Razón |
|---------|----------|-------|
| **RDBMS** | PostgreSQL 15+ | JSONB para flexibilidad, extensiones (pg_trgm, uuid-ossp) |
| **Cola de trabajos** | Tabla `jobs_queue` | Simplicidad, transaccionalidad, sin infraestructura adicional |
| **Búsqueda de texto** | pg_trgm + GIN | Búsqueda fuzzy sin Elasticsearch |

```sql
-- ¿Por qué PostgreSQL como cola?
-- Ventajas:
✅ ACID completo (trabajos no se pierden)
✅ FOR UPDATE SKIP LOCKED (concurrencia sin bloqueos)
✅ Sin servicio adicional que mantener
✅ Transacciones con datos de negocio

-- Desventajas (aceptables para MVP):
⚠️ Polling (1 query/segundo por worker)
⚠️ Límite práctico ~10k jobs/segundo
```

### Caché: Redis + Fallback en Memoria

| Decisión | Razón |
|----------|-------|
| Redis como primario | Distribuido, persistente, rápido |
| MemoryCache como fallback | Desarrollo sin dependencias, resiliencia |
| TTL por entidad | Balance entre frescura y rendimiento |
| Cache-aside pattern | Control explícito de invalidación |

### Arquitectura: Clean Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                     DECISIÓN DE CAPAS                        │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  Domain Layer (Entidades + Interfaces)                      │
│  └─ Sin dependencias externas                               │
│  └─ Reglas de negocio puras                                 │
│                                                             │
│  Application Layer (UseCases)                               │
│  └─ Orquesta flujos de negocio                              │
│  └─ Depende solo de Domain                                  │
│                                                             │
│  Infrastructure Layer (Implementaciones)                    │
│  └─ PostgreSQL, Redis, HTTP clients                         │
│  └─ Implementa interfaces de Domain                         │
│                                                             │
│  Interfaces Layer (HTTP Handlers, WebSocket)                │
│  └─ Adapta requests externos a UseCases                     │
│                                                             │
└─────────────────────────────────────────────────────────────┘

Beneficios:
✅ Testeable (mock de interfaces)
✅ Extensible (nuevos países sin cambiar core)
✅ Mantenible (cambios aislados por capa)
```

### Autenticación: JWT

| Decisión | Configuración |
|----------|---------------|
| Access Token | 15 minutos de vida |
| Refresh Token | 7 días de vida |
| Algoritmo | HS256 (simétrico) |
| Storage | HttpOnly cookies (frontend) |

```
¿Por qué JWT y no sesiones?
✅ Stateless (escalabilidad horizontal)
✅ No requiere storage de sesiones
✅ Fácil para microservicios futuros
```

### Comunicación en Tiempo Real: WebSocket

| Decisión | Razón |
|----------|-------|
| Gorilla WebSocket | Librería madura, bien mantenida |
| Hub centralizado | Broadcast eficiente a múltiples clientes |
| Reconexión automática | Frontend resiliente a desconexiones |

## 🔐 Consideraciones de Seguridad

### Protección de PII (Información Personal Identificable)

| Dato | Clasificación | Protección |
|------|---------------|------------|
| Nombre completo | PII | Almacenado, no en logs |
| Documento de identidad | PII Sensible | Almacenado, nunca en logs ni responses completos |
| Email | PII | Almacenado, enmascarado en logs |
| Teléfono | PII | Almacenado, enmascarado en logs |
| Información bancaria | PII Sensible | Almacenado encriptado, nunca expuesto completo |

```go
// Ejemplo: Enmascaramiento en logs
log.Info().
    Str("email", maskEmail(user.Email)).     // j***@example.com
    Str("document", maskDocument(doc)).       // ****4567X
    Msg("Processing application")
```

### Seguridad de APIs

| Mecanismo | Implementación |
|-----------|----------------|
| **Autenticación** | JWT con refresh tokens |
| **Autorización** | Roles (ADMIN, ANALYST, OPERATOR, VIEWER) |
| **Rate Limiting** | Por IP y por usuario (configurable) |
| **CORS** | Orígenes permitidos configurables |
| **Headers de seguridad** | X-Content-Type-Options, X-Frame-Options |
| **Input validation** | Validación en handler + usecase |

### Seguridad de Webhooks

| Mecanismo | Descripción |
|-----------|-------------|
| **Firma HMAC-SHA256** | Todos los webhooks firmados |
| **Verificación de firma** | Rechazo si firma inválida |
| **Timestamp validation** | Rechazo si muy antiguo (replay attack) |
| **Secret por endpoint** | Cada integración tiene su propio secret |

```go
// Verificación de webhook entrante
func verifySignature(payload []byte, signature, secret string) bool {
    mac := hmac.New(sha256.New, []byte(secret))
    mac.Write(payload)
    expected := hex.EncodeToString(mac.Sum(nil))
    return hmac.Equal([]byte(signature), []byte(expected))
}
```

### Seguridad de Base de Datos

| Mecanismo | Implementación |
|-----------|----------------|
| **Conexión SSL** | Requerida en producción |
| **Prepared statements** | Prevención de SQL injection |
| **Credenciales** | Variables de entorno, nunca en código |
| **Principio de menor privilegio** | Usuario de app sin permisos de DDL |

### Datos Bancarios

| Aspecto | Protección |
|---------|------------|
| **Credenciales de proveedores** | Almacenadas en secrets de K8s, nunca en código |
| **Respuestas de APIs** | Datos sensibles no loggeados |
| **Credit scores** | Almacenados, no expuestos en listados |
| **Información financiera** | Visible solo para roles autorizados |

```go
// Entidad BankingProvider - credenciales nunca serializadas
type BankingProvider struct {
    // ...
    Credentials map[string]string `json:"-"` // ← Nunca en JSON
}
```

### Auditoría de Seguridad

| Evento | Registrado |
|--------|------------|
| Login exitoso/fallido | ✅ |
| Cambios de estado de solicitud | ✅ |
| Acceso a datos sensibles | ✅ |
| Modificaciones de configuración | ✅ |
| Webhooks recibidos | ✅ |

```sql
-- Tabla de auditoría
CREATE TABLE audit_logs (
    id UUID PRIMARY KEY,
    entity_type VARCHAR(50) NOT NULL,  -- APPLICATION, USER, CONFIG
    entity_id UUID NOT NULL,
    action VARCHAR(30) NOT NULL,       -- CREATE, UPDATE, DELETE, VIEW
    actor_type VARCHAR(20) NOT NULL,   -- USER, SYSTEM, WEBHOOK
    actor_id UUID,
    old_values JSONB,                  -- Estado anterior
    new_values JSONB,                  -- Estado nuevo
    ip_address INET,
    user_agent TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);
```

## 🚀 Extras Implementados

### ✅ Países Adicionales (6 de 6)

Todos los países de la lista están implementados con:
- Tipos de documento específicos
- Reglas de validación por país
- Proveedores bancarios configurados

| País | Documento | Proveedor |
|------|-----------|-----------|
| 🇪🇸 España | DNI, NIE | Equifax |
| 🇲🇽 México | CURP, RFC | Buró de Crédito |
| 🇨🇴 Colombia | CC, CE | DataCrédito |
| 🇧🇷 Brasil | CPF, RG | Serasa |
| 🇵🇹 Portugal | NIF, CC | Banco de Portugal |
| 🇮🇹 Italia | CF, CI | CRIF |

### ✅ Auditoría Detallada

- Trigger automático al crear/actualizar solicitudes
- Registro de cambios de estado
- Historial completo por solicitud (`GET /applications/:id/history`)

### ✅ Resiliencia ante Fallas

| Mecanismo | Implementación |
|-----------|----------------|
| Retry con backoff | Exponencial: 30s, 120s, 270s |
| Circuit breaker | Configurable por proveedor |
| Fallback de caché | MemoryCache si Redis no disponible |
| Graceful shutdown | Workers terminan jobs en curso |
| Dead letter queue | Jobs fallidos marcados para revisión |

## 📄 Licencia

MIT

