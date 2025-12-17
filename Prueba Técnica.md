# Prueba Técnica
**Software Engineer (Fintech Multipaís)**

## 1. Contexto

Nuestra empresa es una **fintech que opera en 6 países** entre Latinoamérica y Europa:

- España (ES)
- Portugal (PT)
- Italia (IT)
- México (MX)
- Colombia (CO)
- Brasil (BR)

Cada país tiene diferencias operativas, regulatorias y proveedores bancarios distintos. El objetivo es construir un sistema base que admita solicitudes de crédito en múltiples países de forma extensible y **preparado para operar a gran escala**.

## 2. Objetivo del reto

Construir un MVP que permita:

1. Crear solicitudes de crédito.
2. Validar reglas de negocio específicas por país.
3. Integrarse con proveedores bancarios distintos según país.
4. Consultar solicitudes individuales.
5. Listar solicitudes filtradas por país.
6. Actualizar el estado de una solicitud.
7. Procesar lógica de negocio en segundo plano y en paralelo.
8. Mostrar información en tiempo (casi) real en el frontend.

Debes seleccionar **al menos dos países** de la lista anterior para tu implementación principal. Puedes agregar más si lo deseas.

## 3. Funcionalidad requerida

### 3.1 Creación de solicitudes
Cada solicitud debe incluir:

- país
- nombre completo
- documento de identidad
- monto solicitado
- ingreso mensual
- fecha de solicitud
- estado inicial
- información bancaria obtenida del proveedor correspondiente

La creación de una solicitud debe disparar lógica adicional (por ejemplo: validación de riesgo, auditoría o procesamiento en segundo plano).

### 3.2 Validación de reglas por país
Cada país tiene reglas específicas que deben aplicarse durante la creación o actualización de una solicitud de crédito. Implementa las siguientes reglas mínimas, considerando que en cada caso se requiere verificar que el documento y la información asociada sean razonablemente válidos según el país correspondiente:

**España (ES)**
- Documento requerido: DNI.
- La solicitud debe incluir verificaciones del documento.
- Si el monto solicitado supera un umbral definido por ti, debe marcarse como sujeta a revisión adicional.

**Portugal (PT)**
- Documento requerido: NIF.
- Debe existir alguna verificación del documento.
- Debe existir al menos una regla relacionada con el ingreso mensual y el monto solicitado.

**Italia (IT)**
- Documento requerido: Codice Fiscale.
- La solicitud debe incluir verificaciones del documento.
- Debe existir una regla relacionada con estabilidad financiera o ingreso.

**México (MX)**
- Documento requerido: CURP.
- Verificación correspondiente del documento.
- Debe existir alguna regla basada en la relación entre ingreso mensual y monto solicitado.

**Colombia (CO)**
- Documento requerido: Cédula de Ciudadanía (CC).
- Considerar la relación entre deuda total (dato del proveedor bancario) y el ingreso mensual.

**Brasil (BR)**
- Documento requerido: CPF.
- Verificación correspondiente del documento.
- Debe incluirse alguna regla relacionada con score financiero o capacidad de pago.

Puedes extender o agregar reglas si lo consideras necesario.

### 3.3 Integración con proveedor bancario por país
Cada país utiliza un proveedor bancario distinto para obtener información del cliente. Estos proveedores pueden tener diferencias en la forma en que entregan la información, así como en los datos específicos que proporcionan.

Tu solución debe contemplar estas variaciones entre países y permitir que la aplicación procese la información bancaria necesaria para evaluar cada solicitud.

### 3.4 Estados de la solicitud
Define un flujo de estados adecuado por país. El diseño debe permitir agregar nuevos estados o flujos en el futuro.

Las transiciones de estado pueden disparar lógica adicional (por ejemplo: notificaciones, reevaluaciones o auditoría).

### 3.5 Consultar una solicitud
La aplicación debe permitir recuperar los datos completos de una solicitud específica mediante su identificador.

### 3.6 Listar solicitudes
Debe existir una forma de obtener un listado de solicitudes, con la capacidad de filtrarlas por país y otros criterios que consideres relevantes (por ejemplo: estado, rango de fechas).

### 3.7 Procesamiento asíncrono y eventos
El sistema debe incorporar **procesamiento asíncrono**, de forma que ciertas tareas no bloqueen el flujo principal de la API. Considera, por ejemplo:

- Procesos de evaluación de riesgo.
- Generación de registros de auditoría.
- Notificaciones hacia otros sistemas.

**Requisitos:**
- Utilizar capacidades nativas de base de datos (por ejemplo, funciones y mecanismos de disparo en PostgreSQL) para reaccionar a cambios en los datos cuando lo consideres apropiado.
- Incluir al menos un flujo en el que una operación en la base de datos genere trabajo a ser procesado de forma asíncrona (por ejemplo: en una cola de trabajos).

### 3.8 Webhooks y procesos externos
Define al menos un flujo en el que:

- Tu sistema **reciba** información desde un sistema externo vía webhook.
  **o**
- Tu sistema **envíe** una notificación a un endpoint simulado externo para completar parte del flujo.

Este flujo debe estar integrado con el modelo de solicitudes (por ejemplo: actualización de estado, confirmación de datos o registro de eventos externos).

### 3.9 Concurrencia y procesamiento en paralelo
El diseño debe permitir ejecutar múltiples procesos o workers en paralelo (por ejemplo, varios consumidores de cola o procesos que reaccionen a eventos) sin generar inconsistencias de datos evidentes.

No es necesario simular alta concurrencia real, pero sí debes mostrar cómo tu solución permitiría escalar el número de procesos o instancias que ejecutan lógica de negocio concurrente.

### 3.10 Actualización en tiempo real (realtime) en el frontend
Incluye una vista que muestre información relevante (por ejemplo, la lista de solicitudes o los cambios de estado) y que pueda actualizarse en tiempo casi real cuando se produzcan cambios en el sistema.

Puedes usar **Socket.IO u otra tecnología equivalente** de comunicación bidireccional para mantener la interfaz sincronizada con los eventos del backend.

## 4. Requerimientos no funcionales

### 4.1 Arquitectura y niveles de responsabilidad en código
- Código modular y extensible.
- Separación clara de responsabilidades en múltiples capas (por ejemplo: controladores, servicios, repositorios, integración, etc.).
- Diseño que permita agregar países, proveedores o nuevos flujos sin cambios disruptivos en todo el sistema.

### 4.2 Seguridad de APIs
- Manejo seguro de PII.
- Evitar exponer datos bancarios sensibles.
- Implementar al menos un mecanismo de autenticación basado en **JWT** u otra estrategia equivalente.
- Considerar autorización básica (quién puede ver o modificar qué).

### 4.3 Observabilidad
- Logs claros y estructurados.
- Manejo explícito de errores.
- Registros suficientes para entender qué ocurrió en un flujo asíncrono (por ejemplo, trabajos en cola, webhooks, cambios de estado).

### 4.4 Reproducibilidad
- La solución debe poder ejecutarse fácilmente.
- Incluir instrucciones claras en el README.
- El evaluador debe poder instalar y ejecutar en **menos de 5 minutos** (asumiendo herramientas estándar instaladas).

### 4.5 Escalabilidad y manejo de grandes volúmenes de datos
Diseña pensando en que el sistema puede llegar a manejar **millones de solicitudes de crédito**.

Incluye en el README un análisis sobre:
- Índices recomendados.
- Cómo estructurarías las tablas para manejar grandes volúmenes (particionamiento, estrategias que consideres).
- Consultas críticas y cómo evitarías cuellos de botella.
- Posibles estrategias de archivado o compresión si las consideras necesarias.

No es necesario crear millones de registros, pero sí demostrar que el diseño los considera.

### 4.6 Colas y encolamiento de trabajos
El sistema debe ser capaz de encolar tareas para su ejecución asíncrona (por ejemplo, mediante una cola de mensajes o una tabla de trabajos).

- Explica en el README qué tecnología utilizas (o simulas) para la cola.
- Muestra cómo se produce y se consume al menos un tipo de trabajo.

### 4.7 Caching
Incorpora alguna forma de **caché** para mejorar la respuesta de una parte del sistema (por ejemplo, lectura de solicitudes, resultados de evaluación, catálogos, etc.).

- Indica qué decides cachear y por qué.
- Describe en el README qué estrategia de invalidación usas (aunque sea simple).

### 4.8 Despliegue (Kubernetes / k8s)
Incluye archivos de configuración para desplegar tu solución en un entorno de tipo **Kubernetes**. No es necesario realizar un despliegue real, pero sí:

- Manifiestos básicos (YAML) para los componentes principales (por ejemplo: backend, frontend, base de datos si la incluyes en el entorno, workers).
- Variables de entorno y configuración necesaria.
- Cualquier consideración especial (por ejemplo, servicios, ingress, etc.).

Si usas otra herramienta relacionada (Helm, kustomize, etc.), descríbelo en el README.

## 5. Frontend requerido

Incluye una interfaz que permita:

- Crear solicitudes.
- Ver la lista de solicitudes.
- Ver detalles.
- Actualizar estado.
- Visualizar en tiempo casi real los cambios relevantes (por ejemplo, cambios de estado o resultados de procesos asíncronos).

El diseño puede ser sencillo, pero debe mostrar la información de forma clara y manejar errores adecuadamente.

## 6. Entregables

1.  Repositorio con backend, frontend y código relacionado con procesamiento asíncrono, colas, caché y despliegue.
2.  README con:
    - Instrucciones claras para instalar y ejecutar la solución.
    - Supuestos.
    - Modelo de datos.
    - Decisiones técnicas.
    - Consideraciones de seguridad.
    - Análisis de escalabilidad y manejo de grandes volúmenes de datos.
    - Descripción de la estrategia de concurrencia, colas, caché y webhooks.
3.  **Archivos de configuración para despliegue en Kubernetes.**
4.  **Archivo(s) `Makefile` o `Justfile`** con comandos para simplificar tareas frecuentes (por ejemplo: `make run`, `make test`, `make migrate`, `make deploy` o equivalentes).

**Extras opcionales:**
- Implementación de países adicionales.
- Métricas y dashboards.
- Auditoría detallada de cambios.
- Mecanismos avanzados de resiliencia ante fallas de proveedores o colas.

## 7. Tiempo

Completa el reto a tu propio ritmo. Se tomará en cuenta la manera en que priorices, estructures y presentes tu solución.

## 8. Entrega

Envía un repositorio público con toda tu solución.

¡Éxito!

## 9. Glosario

**PII:** “Personally Identifiable Information”, o **información personal identificable que puede usarse para identificar directa o indirectamente a una persona.**