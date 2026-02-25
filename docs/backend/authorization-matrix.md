# Matriz de Permisos por Endpoint

## Resumen del Sistema de Roles

El sistema tiene **3 niveles de roles** jerárquicos:

1. **SystemRole** (nivel plataforma): `SUPERADMIN` > `PLATFORM_ADMIN`
2. **TenantRole** (nivel tenant): `TENANT_OWNER` > `TENANT_ADMIN`
3. **WorkspaceRole** (nivel workspace): `OWNER` > `ADMIN` > `EDITOR` > `OPERATOR` > `VIEWER`

### Headers Requeridos

| Header | Descripción |
|--------|-------------|
| `Authorization` | `Bearer <JWT_token>` - Requerido para todos los endpoints autenticados |
| `X-Tenant-ID` | UUID del tenant - Requerido para rutas `/tenant/*` |
| `X-Workspace-ID` | UUID del workspace - Requerido para rutas `/workspace/*` y `/content/*` |
| `X-Operation-ID` | UUID de operación (opcional, se genera automáticamente) |

### Elevación Automática de Roles

- `SUPERADMIN` obtiene acceso `OWNER` a cualquier workspace automáticamente
- `SUPERADMIN` obtiene acceso `TENANT_OWNER` a cualquier tenant automáticamente
- `TENANT_OWNER` obtiene acceso `ADMIN` a workspaces dentro de su tenant

---

## Tabla 1: Endpoints de Sistema (SystemRole)

- **Ruta base**: `/api/v1/system`
- **Headers requeridos**: `Authorization`
- **NO requiere**: `X-Tenant-ID`, `X-Workspace-ID`

| Método | Endpoint | Descripción | SUPERADMIN | PLATFORM_ADMIN |
|--------|----------|-------------|:----------:|:--------------:|
| GET | `/system/tenants?page=1&perPage=10&q={query}` | Lista tenants con paginación y búsqueda opcional | ✅ | ✅ |
| POST | `/system/tenants` | Crea un nuevo tenant | ✅ | ❌ |
| GET | `/system/tenants/{tenantId}` | Obtiene información de un tenant específico | ✅ | ✅ |
| PUT | `/system/tenants/{tenantId}` | Actualiza la información de un tenant | ✅ | ✅ |
| DELETE | `/system/tenants/{tenantId}` | Elimina un tenant y todos sus datos | ✅ | ❌ |
| GET | `/system/tenants/{tenantId}/workspaces?page=1&perPage=10&q={query}` | Lista workspaces de un tenant con paginación y búsqueda opcional | ✅ | ✅ |
| GET | `/system/users` | Lista usuarios con roles de sistema asignados | ✅ | ❌ |
| POST | `/system/users/{userId}/role` | Asigna un rol de sistema a un usuario | ✅ | ❌ |
| DELETE | `/system/users/{userId}/role` | Revoca el rol de sistema de un usuario | ✅ | ❌ |

**Archivo fuente**: `internal/adapters/primary/http/controller/admin_controller.go`

### Endpoints de System Injectables (`/api/v1/system/injectables`)

Gestión de inyectores del sistema definidos en código (extensibility system).

| Método | Endpoint | Descripción | SUPERADMIN | PLATFORM_ADMIN |
|--------|----------|-------------|:----------:|:--------------:|
| GET | `/system/injectables` | Lista todos los injectors con su estado (activo/inactivo) | ✅ | ✅ |
| PATCH | `/system/injectables/:key/activate` | Activa un injector globalmente | ✅ | ❌ |
| PATCH | `/system/injectables/:key/deactivate` | Desactiva un injector globalmente | ✅ | ❌ |
| GET | `/system/injectables/:key/assignments` | Lista assignments de un injector | ✅ | ✅ |
| POST | `/system/injectables/:key/assignments` | Crea assignment (asigna a scope) | ✅ | ❌ |
| DELETE | `/system/injectables/:key/assignments/:id` | Elimina un assignment | ✅ | ❌ |
| PATCH | `/system/injectables/:key/assignments/:id/exclude` | Excluye un assignment (is_active=false) | ✅ | ❌ |
| PATCH | `/system/injectables/:key/assignments/:id/include` | Incluye un assignment (is_active=true) | ✅ | ❌ |
| POST | `/system/injectables/bulk/public` | Crea assignments PUBLIC para múltiples keys (bulk) | ✅ | ❌ |
| DELETE | `/system/injectables/bulk/public` | Elimina assignments PUBLIC para múltiples keys (bulk) | ✅ | ❌ |

**Archivo fuente**: `internal/adapters/primary/http/controller/admin_controller.go`

---

## Tabla 2: Endpoints de Tenant (TenantRole)

- **Ruta base**: `/api/v1/tenant`
- **Headers requeridos**: `Authorization`, `X-Tenant-ID`
- **NO requiere**: `X-Workspace-ID`

| Método | Endpoint | Descripción | TENANT_OWNER | TENANT_ADMIN |
|--------|----------|-------------|:------------:|:------------:|
| GET | `/tenant` | Obtiene información del tenant actual | ✅ | ✅ |
| PUT | `/tenant` | Actualiza la información del tenant actual | ✅ | ❌ |
| GET | `/tenant/workspaces?page=1&perPage=10&q={query}` | Lista workspaces con paginación y búsqueda opcional | ✅ | ✅ |
| GET | `/tenant/my-workspaces` | Lista los workspaces a los que el usuario tiene acceso en el tenant | ✅ | ✅ |
| POST | `/tenant/workspaces` | Crea un nuevo workspace en el tenant | ✅ | ❌ |
| DELETE | `/tenant/workspaces/{workspaceId}` | Elimina (archiva) un workspace del tenant | ✅ | ❌ |
| GET | `/tenant/members` | Lista todos los miembros del tenant | ✅ | ✅ |
| POST | `/tenant/members` | Agrega un usuario como miembro del tenant | ✅ | ❌ |
| GET | `/tenant/members/{memberId}` | Obtiene información de un miembro específico | ✅ | ✅ |
| PUT | `/tenant/members/{memberId}` | Actualiza el rol de un miembro del tenant | ✅ | ❌ |
| DELETE | `/tenant/members/{memberId}` | Elimina un miembro del tenant | ✅ | ❌ |

**Archivo fuente**: `internal/adapters/primary/http/controller/tenant_controller.go`

### Endpoint `/tenant/workspaces` - Detalle

Lista workspaces del tenant actual con paginación y búsqueda opcional.

**Parámetros:**
| Param | Tipo | Default | Descripción |
|-------|------|---------|-------------|
| `page` | int | 1 | Número de página |
| `perPage` | int | 10 | Cantidad de items por página |
| `q` | string | (opcional) | Texto de búsqueda por nombre |

**Comportamiento:**
- Solo retorna workspaces del tenant indicado en el header `X-Tenant-ID`
- **Sin parámetro `q`**: Ordenados por historial de acceso (más recientes), luego por nombre
- **Con parámetro `q`**: Ordenados por similitud (pg_trgm), búsqueda fuzzy por nombre
- Incluye metadata de paginación

**Ejemplo de respuesta:**
```json
{
  "data": [
    {
      "id": "uuid-workspace-1",
      "tenantId": "uuid-tenant",
      "name": "Marketing Team",
      "type": "CLIENT",
      "status": "ACTIVE",
      "createdAt": "2024-01-15T10:00:00Z"
    }
  ],
  "pagination": {
    "page": 1,
    "perPage": 10,
    "total": 5,
    "totalPages": 1
  }
}
```

---

## Tabla 3: Endpoints de Workspace y Content (WorkspaceRole)

- **Headers requeridos**: `Authorization`, `X-Workspace-ID`
- **NO requiere**: `X-Tenant-ID`

### Lógica de Roles

| Rol | Peso | Responsabilidad |
|-----|------|-----------------|
| OWNER | 50 | Gestión completa del workspace, miembros y configuración |
| ADMIN | 40 | Administración de contenido, publicación y estructura |
| EDITOR | 30 | Crear y editar contenido (templates, injectables, folders, tags) |
| OPERATOR | 20 | Usar templates para generar documentos (solo lectura de contenido) |
| VIEWER | 10 | Solo lectura |

### Endpoints de Workspace (`/api/v1/workspace`)

| Método | Endpoint | Descripción | OWNER | ADMIN | EDITOR | OPERATOR | VIEWER |
|--------|----------|-------------|:-----:|:-----:|:------:|:--------:|:------:|
| GET | `/workspace` | Obtiene información del workspace actual | ✅ | ✅ | ✅ | ✅ | ✅ |
| PUT | `/workspace` | Actualiza la información del workspace | ✅ | ✅ | ❌ | ❌ | ❌ |
| DELETE | `/workspace` | Archiva el workspace actual | ✅ | ❌ | ❌ | ❌ | ❌ |
| GET | `/workspace/members` | Lista todos los miembros del workspace | ✅ | ✅ | ✅ | ✅ | ✅ |
| POST | `/workspace/members` | Invita un usuario al workspace | ✅ | ✅ | ❌ | ❌ | ❌ |
| GET | `/workspace/members/{memberId}` | Obtiene información de un miembro | ✅ | ✅ | ✅ | ✅ | ✅ |
| PUT | `/workspace/members/{memberId}` | Actualiza el rol de un miembro | ✅ | ❌ | ❌ | ❌ | ❌ |
| DELETE | `/workspace/members/{memberId}` | Elimina un miembro del workspace | ✅ | ✅ | ❌ | ❌ | ❌ |
| GET | `/workspace/folders` | Lista todas las carpetas del workspace | ✅ | ✅ | ✅ | ✅ | ✅ |
| GET | `/workspace/folders/tree` | Obtiene el árbol jerárquico de carpetas | ✅ | ✅ | ✅ | ✅ | ✅ |
| POST | `/workspace/folders` | Crea una nueva carpeta | ✅ | ✅ | ✅ | ❌ | ❌ |
| GET | `/workspace/folders/{folderId}` | Obtiene información de una carpeta | ✅ | ✅ | ✅ | ✅ | ✅ |
| PUT | `/workspace/folders/{folderId}` | Actualiza una carpeta | ✅ | ✅ | ✅ | ❌ | ❌ |
| PATCH | `/workspace/folders/{folderId}/move` | Mueve una carpeta a otro padre | ✅ | ✅ | ✅ | ❌ | ❌ |
| DELETE | `/workspace/folders/{folderId}` | Elimina una carpeta | ✅ | ✅ | ❌ | ❌ | ❌ |
| GET | `/workspace/tags` | Lista todas las etiquetas del workspace | ✅ | ✅ | ✅ | ✅ | ✅ |
| POST | `/workspace/tags` | Crea una nueva etiqueta | ✅ | ✅ | ✅ | ❌ | ❌ |
| GET | `/workspace/tags/{tagId}` | Obtiene información de una etiqueta | ✅ | ✅ | ✅ | ✅ | ✅ |
| PUT | `/workspace/tags/{tagId}` | Actualiza una etiqueta | ✅ | ✅ | ✅ | ❌ | ❌ |
| DELETE | `/workspace/tags/{tagId}` | Elimina una etiqueta | ✅ | ✅ | ❌ | ❌ | ❌ |
| GET | `/workspace/injectables` | Lista injectables propios del workspace | ✅ | ✅ | ✅ | ✅ | ✅ |
| POST | `/workspace/injectables` | Crea un injectable (solo tipo TEXT) | ✅ | ✅ | ✅ | ❌ | ❌ |
| GET | `/workspace/injectables/{injectableId}` | Obtiene un injectable del workspace | ✅ | ✅ | ✅ | ✅ | ✅ |
| PUT | `/workspace/injectables/{injectableId}` | Actualiza un injectable | ✅ | ✅ | ✅ | ❌ | ❌ |
| DELETE | `/workspace/injectables/{injectableId}` | Elimina un injectable (soft delete) | ✅ | ✅ | ❌ | ❌ | ❌ |
| POST | `/workspace/injectables/{injectableId}/activate` | Activa un injectable | ✅ | ✅ | ✅ | ❌ | ❌ |
| POST | `/workspace/injectables/{injectableId}/deactivate` | Desactiva un injectable | ✅ | ✅ | ✅ | ❌ | ❌ |

**Archivo fuente**: `internal/adapters/primary/http/controller/workspace_controller.go`

### Endpoints de Injectables - Lectura (`/api/v1/content/injectables`)

> **Nota**: Estos endpoints son de solo lectura y listan todos los injectables disponibles para el workspace (globales + propios del workspace). Solo se muestran injectables activos (`is_active=true`) y no eliminados (`is_deleted=false`).

| Método | Endpoint | Descripción | OWNER | ADMIN | EDITOR | OPERATOR | VIEWER |
|--------|----------|-------------|:-----:|:-----:|:------:|:--------:|:------:|
| GET | `/content/injectables` | Lista injectables disponibles (globales + workspace, activos y no eliminados) | ✅ | ✅ | ✅ | ✅ | ✅ |
| GET | `/content/injectables/{injectableId}` | Obtiene una definición de injectable | ✅ | ✅ | ✅ | ✅ | ✅ |

**Archivo fuente**: `internal/adapters/primary/http/controller/content_injectable_controller.go`

> Para crear, editar o eliminar injectables del workspace, usar los endpoints de `/workspace/injectables`.

### Endpoints de Templates (`/api/v1/content/templates`)

| Método | Endpoint | Descripción | OWNER | ADMIN | EDITOR | OPERATOR | VIEWER |
|--------|----------|-------------|:-----:|:-----:|:------:|:--------:|:------:|
| GET | `/content/templates` | Lista todos los templates con filtros opcionales | ✅ | ✅ | ✅ | ✅ | ✅ |
| POST | `/content/templates` | Crea un nuevo template con versión draft inicial | ✅ | ✅ | ✅ | ❌ | ❌ |
| GET | `/content/templates/{templateId}` | Obtiene un template con detalles de versión publicada | ✅ | ✅ | ✅ | ✅ | ✅ |
| GET | `/content/templates/{templateId}/all-versions` | Obtiene un template con todas sus versiones | ✅ | ✅ | ✅ | ✅ | ✅ |
| PUT | `/content/templates/{templateId}` | Actualiza los metadatos del template | ✅ | ✅ | ✅ | ❌ | ❌ |
| DELETE | `/content/templates/{templateId}` | Elimina un template y todas sus versiones | ✅ | ✅ | ❌ | ❌ | ❌ |
| POST | `/content/templates/{templateId}/clone` | Clona un template desde su versión publicada | ✅ | ✅ | ✅ | ❌ | ❌ |
| POST | `/content/templates/{templateId}/tags` | Agrega etiquetas a un template | ✅ | ✅ | ✅ | ❌ | ❌ |
| DELETE | `/content/templates/{templateId}/tags/{tagId}` | Elimina una etiqueta de un template | ✅ | ✅ | ✅ | ❌ | ❌ |

**Archivo fuente**: `internal/adapters/primary/http/controller/content_template_controller.go`

### Endpoints de Contract Generation (`/api/v1/content`)

| Método | Endpoint | Descripción | OWNER | ADMIN | EDITOR | OPERATOR | VIEWER |
|--------|----------|-------------|:-----:|:-----:|:------:|:--------:|:------:|
| POST | `/content/generate-contract` | Genera un contrato estructurado desde imagen/PDF/DOCX/texto usando IA | ✅ | ✅ | ✅ | ❌ | ❌ |

**Archivo fuente**: `internal/adapters/primary/http/controller/contract_generator_controller.go`

#### Endpoint `/content/generate-contract` - Detalle

Genera un documento de contrato estructurado (PortableDocument JSON) analizando el contenido proporcionado mediante un modelo de lenguaje (LLM).

**Request body:**
```json
{
  "contentType": "image",
  "content": "<base64_encoded_content>",
  "mimeType": "image/png",
  "outputLang": "es"
}
```

| Campo | Tipo | Requerido | Valores válidos | Descripción |
|-------|------|-----------|-----------------|-------------|
| `contentType` | string | ✅ | `image`, `pdf`, `docx`, `text` | Tipo de contenido de entrada |
| `content` | string | ✅ | - | Contenido base64 (image/pdf/docx) o texto plano (text) |
| `mimeType` | string | ✅* | `image/png`, `image/jpeg`, `application/pdf`, `application/vnd.openxmlformats-officedocument.wordprocessingml.document` | MIME type del contenido (*requerido para image/pdf/docx) |
| `outputLang` | string | ❌ | `es`, `en` | Idioma de salida (default: `es`) |

**Ejemplo de respuesta:**
```json
{
  "document": {
    "version": "1.1.0",
    "meta": {
      "title": "Contrato de Arrendamiento",
      "language": "es"
    },
    "content": { /* ProseMirror document structure */ },
    "signerRoles": [...],
    "variableIds": [...]
  },
  "tokensUsed": 4523,
  "model": "gpt-4o",
  "generatedAt": "2024-01-15T10:30:00Z"
}
```

**Respuestas:**
| Código | Descripción |
|--------|-------------|
| 200 | Contrato generado exitosamente |
| 400 | Request inválido (contentType, mimeType faltante, etc.) |
| 401 | Usuario no autenticado |
| 403 | Usuario sin permisos (requiere rol EDITOR+) |
| 503 | Servicio de IA no disponible |

---

### Endpoints de Template Versions (`/api/v1/content/templates/{templateId}/versions`)

| Método | Endpoint | Descripción | OWNER | ADMIN | EDITOR | OPERATOR | VIEWER |
|--------|----------|-------------|:-----:|:-----:|:------:|:--------:|:------:|
| GET | `/versions` | Lista todas las versiones de un template | ✅ | ✅ | ✅ | ✅ | ✅ |
| POST | `/versions` | Crea una nueva versión del template | ✅ | ✅ | ✅ | ❌ | ❌ |
| POST | `/versions/from-existing` | Crea una versión copiando contenido de otra existente | ✅ | ✅ | ✅ | ❌ | ❌ |
| GET | `/versions/{versionId}` | Obtiene una versión con todos sus detalles | ✅ | ✅ | ✅ | ✅ | ✅ |
| PUT | `/versions/{versionId}` | Actualiza una versión (solo drafts) | ✅ | ✅ | ✅ | ❌ | ❌ |
| DELETE | `/versions/{versionId}` | Elimina una versión draft | ✅ | ✅ | ❌ | ❌ | ❌ |
| POST | `/versions/{versionId}/publish` | Publica una versión draft | ✅ | ✅ | ❌ | ❌ | ❌ |
| POST | `/versions/{versionId}/archive` | Archiva una versión publicada | ✅ | ✅ | ❌ | ❌ | ❌ |
| POST | `/versions/{versionId}/schedule-publish` | Programa una publicación futura | ✅ | ✅ | ❌ | ❌ | ❌ |
| POST | `/versions/{versionId}/schedule-archive` | Programa un archivado futuro | ✅ | ✅ | ❌ | ❌ | ❌ |
| DELETE | `/versions/{versionId}/schedule` | Cancela una acción programada | ✅ | ✅ | ❌ | ❌ | ❌ |
| POST | `/versions/{versionId}/injectables` | Agrega un injectable a la versión | ✅ | ✅ | ✅ | ❌ | ❌ |
| DELETE | `/versions/{versionId}/injectables/{injectableId}` | Elimina un injectable de la versión | ✅ | ✅ | ✅ | ❌ | ❌ |
| POST | `/versions/{versionId}/signer-roles` | Agrega un rol de firmante a la versión | ✅ | ✅ | ✅ | ❌ | ❌ |
| PUT | `/versions/{versionId}/signer-roles/{roleId}` | Actualiza un rol de firmante | ✅ | ✅ | ✅ | ❌ | ❌ |
| DELETE | `/versions/{versionId}/signer-roles/{roleId}` | Elimina un rol de firmante de la versión | ✅ | ✅ | ✅ | ❌ | ❌ |

**Archivo fuente**: `internal/adapters/primary/http/controller/template_version_controller.go`

### Resumen de Roles Mínimos por Operación

| Operación | Rol Mínimo |
|-----------|------------|
| Lectura (GET) | VIEWER |
| Crear contenido (POST templates/versions/injectables) | EDITOR |
| Editar contenido (PUT templates/versions/injectables) | EDITOR |
| Generar contratos con IA (POST generate-contract) | EDITOR |
| Eliminar contenido | ADMIN |
| Publicar/Archivar versiones | ADMIN |
| Gestionar carpetas/tags | EDITOR (crear/editar), ADMIN (eliminar) |
| Gestionar miembros | ADMIN (invitar/eliminar), OWNER (cambiar roles) |
| Configuración workspace | ADMIN (editar), OWNER (archivar) |

---

## Tabla 4: Endpoints sin Contexto (Solo Auth)

**Headers requeridos**: `Authorization`
**NO requiere**: `X-Tenant-ID`, `X-Workspace-ID`

| Método | Endpoint | Descripción | Cualquier usuario autenticado |
|--------|----------|-------------|:-----------------------------:|
| GET | `/me/tenants?page=1&perPage=10&q={query}` | Lista tenants del usuario con paginación y búsqueda opcional | ✅ |
| GET | `/me/roles` | Obtiene los roles del usuario actual (ver detalles abajo) | ✅ |
| POST | `/me/access` | Registra acceso a un tenant o workspace para historial rápido | ✅ |

### Endpoint `/me/tenants` - Detalle

Lista tenants donde el usuario es miembro activo con paginación y búsqueda opcional.

**Parámetros:**
| Param | Tipo | Default | Descripción |
|-------|------|---------|-------------|
| `page` | int | 1 | Número de página |
| `perPage` | int | 10 | Cantidad de items por página |
| `q` | string | (opcional) | Texto de búsqueda por nombre o código |

**Comportamiento:**
- Solo retorna tenants donde el usuario tiene membresía ACTIVE
- **Sin parámetro `q`**: Ordenados por historial de acceso (más recientes), luego por nombre
- **Con parámetro `q`**: Ordenados por similitud (pg_trgm), búsqueda fuzzy por nombre y código
- Incluye metadata de paginación

**Ejemplo de respuesta:**
```json
{
  "data": [
    {
      "id": "uuid-tenant-1",
      "name": "Chile Operations",
      "code": "CL",
      "role": "TENANT_OWNER",
      "createdAt": "2024-01-15T10:00:00Z"
    }
  ],
  "pagination": {
    "page": 1,
    "perPage": 10,
    "total": 5,
    "totalPages": 1
  }
}
```

---

### Endpoint `/me/access` - Detalle

Registra que el usuario accedió a un tenant o workspace, actualizando el historial de accesos rápidos.

**Request body:**
```json
{
  "entityType": "TENANT",
  "entityId": "uuid-del-tenant"
}
```

| Campo | Tipo | Valores válidos | Descripción |
|-------|------|-----------------|-------------|
| `entityType` | string | `TENANT`, `WORKSPACE` | Tipo de recurso accedido |
| `entityId` | UUID | - | ID del tenant o workspace |

**Comportamiento:**
- Si ya existe un registro para ese usuario/tipo/entidad, actualiza el timestamp
- El sistema mantiene automáticamente máximo 10 registros por usuario por tipo
- Verifica que el usuario sea miembro del recurso antes de registrar

**Respuestas:**
| Código | Descripción |
|--------|-------------|
| 204 | Acceso registrado exitosamente |
| 400 | entityType inválido o entityId faltante |
| 401 | Usuario no autenticado |
| 403 | Usuario no es miembro del tenant/workspace |

---

### Endpoint `/me/roles` - Detalle

Este endpoint retorna los roles del usuario autenticado de forma condicional:

| Header | Comportamiento |
|--------|----------------|
| *(ninguno)* | Retorna solo el rol de sistema si existe |
| `X-Tenant-ID` | Agrega el rol del tenant si el usuario es miembro |
| `X-Workspace-ID` | Agrega el rol del workspace si el usuario es miembro |

**Ejemplo de respuesta:**
```json
{
  "roles": [
    { "type": "SYSTEM", "role": "SUPERADMIN", "resourceId": null },
    { "type": "TENANT", "role": "TENANT_OWNER", "resourceId": "uuid-tenant" },
    { "type": "WORKSPACE", "role": "ADMIN", "resourceId": "uuid-workspace" }
  ]
}
```

**Notas:**
- Si el usuario no tiene roles asignados, retorna `{"roles": []}`
- Si el usuario no es miembro del tenant/workspace indicado, ese rol no se incluye (sin error)
- Los headers `X-Tenant-ID` y `X-Workspace-ID` son opcionales e independientes

**Archivo fuente**: `internal/adapters/primary/http/controller/me_controller.go`

---

## Tabla 5: Internal API (API Key Auth)

**Ruta base**: `/api/v1/internal`
**Headers requeridos**:

| Header | Descripción |
|--------|-------------|
| `X-API-Key` | API Key configurada en `internal_api.api_key` |
| `X-Tenant-Code` | Código de tenant (no UUID) |
| `X-Workspace-Code` | Código de workspace (no UUID) |
| `X-Document-Type` | Código del tipo documental |
| `X-External-ID` | ID externo del cliente/entidad (ej: CRM ID) |
| `X-Transactional-ID` | ID de trazabilidad de la transacción |

**NO requiere**: `Authorization`, `X-Tenant-ID`, `X-Workspace-ID`

> **Nota**: Esta API es para comunicación service-to-service. La autenticación se realiza mediante API Key en lugar de JWT.

### Endpoints de Documentos (`/api/v1/internal/documents`)

| Método | Endpoint | Descripción | Requiere API Key |
|--------|----------|-------------|:----------------:|
| POST | `/internal/documents/create` | Crea un documento usando el sistema de extensiones | ✅ |

**Archivo fuente**: `internal/adapters/primary/http/controller/internal_document_controller.go`

### Endpoint `/internal/documents/create` - Detalle

Crea un documento utilizando el sistema de extensiones (Mapper, Init, Injectors).

**Headers requeridos:**
| Header | Descripción |
|--------|-------------|
| `X-API-Key` | API Key para autenticación service-to-service |
| `X-Tenant-Code` | Código de tenant |
| `X-Workspace-Code` | Código de workspace |
| `X-Document-Type` | Código de tipo documental |
| `X-External-ID` | ID externo del cliente/entidad (ej: CRM ID) |
| `X-Transactional-ID` | ID de trazabilidad de la transacción |

**Body:**
Contrato v1 actual (breaking change):

```json
{
  "forceCreate": false,
  "supersedeReason": "optional reason",
  "payload": {
    "customerName": "Juan Pérez",
    "productId": "PROD-001",
    "amount": 50000,
    "quantity": 1
  }
}
```

`payload` es el único bloque enviado al Mapper como `RawBody`.

**Respuestas:**
- `201 Created`: create real
- `200 OK`: replay idempotente

**Ejemplo de respuesta (201/200):**
```json
{
  "id": "doc-uuid",
  "workspaceId": "workspace-uuid",
  "templateVersionId": "version-uuid",
  "externalId": "CRM-123",
  "transactionalId": "TXN-456",
  "status": "DRAFT",
  "idempotentReplay": false,
  "supersededPreviousDocumentId": null,
  "recipients": [
    {
      "id": "recipient-uuid",
      "name": "Juan Pérez",
      "email": "juan@example.com"
    }
  ]
}
```

**Compatibilidad legacy:**
- Requests basadas en `X-Template-ID` sin los headers nuevos retornan `400`.

**Respuestas:**
| Código | Descripción |
|--------|-------------|
| 201 | Documento creado exitosamente |
| 400 | Headers faltantes o error en el Mapper |
| 400 | Injectables requeridos no disponibles para el workspace (`MISSING_INJECTABLES`) |
| 401 | API Key faltante o inválida |
| 404 | Template no encontrado o sin versión publicada |
| 500 | Error interno |

**Flujo de ejecución:**
1. Valida API Key
2. Busca el template y obtiene el workspaceID
3. Obtiene injectables disponibles para el workspace
4. Busca la versión publicada del template
5. Valida que todos los injectables requeridos estén disponibles
6. Ejecuta el Mapper del usuario para parsear el body
7. Ejecuta Init + Injectors para resolver valores
8. Construye recipients desde SignerRoles + valores resueltos
9. Crea el documento con status DRAFT
10. Retorna el documento creado con recipients

**Configuración:**
```yaml
# settings/app.yaml
internal_api:
  enabled: true
  api_key: ""  # DOC_ENGINE_INTERNAL_API_API_KEY
```

---

## Endpoints Públicos (Sin Auth)

| Método | Endpoint | Descripción |
|--------|----------|-------------|
| GET | `/health` | Verifica que el servicio está corriendo |
| GET | `/ready` | Verifica que el servicio está listo para recibir tráfico |
| GET | `/api/v1/ping` | Endpoint de prueba de conectividad de la API |

---

## Archivos de Middleware

| Archivo | Descripción |
|---------|-------------|
| `internal/adapters/primary/http/middleware/jwt_auth.go` | Valida tokens JWT usando JWKS de Keycloak |
| `internal/adapters/primary/http/middleware/identity_context.go` | Obtiene el ID del usuario de la base de datos por email |
| `internal/adapters/primary/http/middleware/system_context.go` | Carga rol de sistema del usuario (opcional) |
| `internal/adapters/primary/http/middleware/tenant_context.go` | Valida X-Tenant-ID y carga rol de tenant |
| `internal/adapters/primary/http/middleware/role_authorization.go` | Autoriza acceso basado en roles de workspace |
| `internal/adapters/primary/http/middleware/apikey_auth.go` | Valida API Key para internal API (service-to-service) |
