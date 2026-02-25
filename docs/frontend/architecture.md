# Guía de Arquitectura del Proyecto: Doc-Assembly Web Client

> **Propósito**: Esta guía define cómo crear el proyecto desde cero, las tecnologías a usar, la estructura de carpetas, y los patrones de desarrollo para integrar el diseño del equipo.

---

## 1. Stack Tecnológico

### Core

| Tecnología | Versión | Propósito |
|------------|---------|-----------|
| **React** | 19.x | Framework de UI |
| **TypeScript** | 5.8+ | Tipado estático |
| **Vite** | 7.x | Build tool y dev server |
| **Tailwind CSS** | 4.x | Estilos utility-first |

### Routing y Estado

| Tecnología | Versión | Propósito |
|------------|---------|-----------|
| **TanStack Router** | 1.x | File-based routing |
| **TanStack Query** | 5.x | Server state y caching |
| **Zustand** | 5.x | Client state |

### UI

| Tecnología | Propósito |
|------------|-----------|
| **Radix UI** | Primitivos accesibles |
| **Framer Motion** | Animaciones |
| **Lucide React** | Iconos |
| **class-variance-authority** | Variantes de componentes |
| **clsx + tailwind-merge** | Utilidad para clases CSS |

### Otros

| Tecnología | Propósito |
|------------|-----------|
| **TipTap** | Editor rich text |
| **Zod** | Validación |
| **Axios** | Cliente HTTP |
| **i18next** | Internacionalización |
| **date-fns** | Fechas |
| **dnd-kit** | Drag and drop |
| **Keycloak JS** | Autenticación |

---

## 2. Creación del Proyecto

```bash
# Crear proyecto
pnpm create vite@latest doc-assembly-web --template react-ts
cd doc-assembly-web

# Core
pnpm add react@latest react-dom@latest
pnpm add tailwindcss @tailwindcss/vite

# Routing y Estado
pnpm add @tanstack/react-router zustand @tanstack/react-query
pnpm add -D @tanstack/router-plugin

# UI
pnpm add @radix-ui/react-dialog @radix-ui/react-dropdown-menu @radix-ui/react-select @radix-ui/react-tabs @radix-ui/react-tooltip @radix-ui/react-popover @radix-ui/react-scroll-area @radix-ui/react-separator @radix-ui/react-slot @radix-ui/react-switch @radix-ui/react-label
pnpm add framer-motion lucide-react
pnpm add class-variance-authority clsx tailwind-merge

# Editor
pnpm add @tiptap/react @tiptap/starter-kit @tiptap/extension-placeholder @tiptap/extension-image @tiptap/extension-link @tiptap/extension-underline @tiptap/extension-text-align @tiptap/extension-color @tiptap/extension-text-style @tiptap/extension-highlight

# Utilidades
pnpm add axios i18next react-i18next i18next-browser-languagedetector date-fns zod
pnpm add @dnd-kit/core @dnd-kit/sortable @dnd-kit/utilities
pnpm add keycloak-js

# Dev
pnpm add -D @types/node @tailwindcss/typography tailwindcss-animate
```

---

## 3. Configuración Base

### vite.config.ts

```typescript
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'
import { TanStackRouterVite } from '@tanstack/router-plugin/vite'
import path from 'path'

export default defineConfig({
  plugins: [
    TanStackRouterVite({ target: 'react', autoCodeSplitting: true }),
    react(),
    tailwindcss(),
  ],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
})
```

### tsconfig.app.json

```json
{
  "compilerOptions": {
    "target": "ES2022",
    "lib": ["ES2022", "DOM", "DOM.Iterable"],
    "module": "ESNext",
    "moduleResolution": "bundler",
    "jsx": "react-jsx",
    "strict": true,
    "noUnusedLocals": true,
    "noUnusedParameters": true,
    "baseUrl": ".",
    "paths": {
      "@/*": ["./src/*"]
    }
  },
  "include": ["src"]
}
```

### src/index.css

> **Documentación completa de tokens de diseño:** `design-system.md`

Estructura del archivo:
- `@import "tailwindcss"` + plugins (typography, animate)
- Variables CSS en `:root` (light mode) y `.dark` (dark mode)
- Colores semánticos: primary, secondary, destructive, warning, info, success, etc.

---

## 4. Estructura de Carpetas

```
src/
├── components/
│   ├── common/           # Componentes genéricos (ThemeToggle, UserMenu, etc.)
│   ├── layout/           # Layouts (AppLayout, AdminLayout, Header, Sidebar)
│   └── ui/               # Primitivos UI (button, dialog, input, select, etc.)
│
├── features/
│   ├── auth/
│   │   ├── components/
│   │   ├── hooks/
│   │   ├── rbac/
│   │   └── types/
│   ├── tenants/
│   │   ├── components/
│   │   ├── hooks/
│   │   └── types/
│   ├── workspaces/
│   │   ├── components/
│   │   ├── hooks/
│   │   └── types/
│   ├── templates/
│   │   ├── components/
│   │   ├── hooks/
│   │   └── types/
│   ├── documents/
│   │   ├── components/
│   │   ├── hooks/
│   │   └── types/
│   └── editor/
│       ├── components/
│       ├── extensions/
│       ├── hooks/
│       └── types/
│
├── hooks/                # Hooks globales (use-debounce, use-media-query, etc.)
│
├── lib/                  # Utilidades (utils.ts, i18n.ts, etc.)
│
├── routes/               # TanStack Router (file-based)
│   ├── __root.tsx
│   ├── _app.tsx
│   ├── _app/
│   │   ├── index.tsx
│   │   ├── select-tenant.tsx
│   │   └── workspace/
│   │       └── $workspaceId/
│   └── admin/
│       ├── route.tsx
│       ├── index.tsx
│       ├── tenants.tsx
│       └── users.tsx
│
├── stores/               # Zustand stores
│   ├── auth-store.ts
│   ├── app-context-store.ts
│   └── theme-store.ts
│
├── types/                # Tipos globales
│
├── routeTree.gen.ts      # Auto-generado (NO editar)
├── main.tsx
├── App.tsx
└── index.css
```

---

## 5. Patrones de Código

### Componente UI (Primitivo)

Patrón para crear componentes con variantes usando `cva` (class-variance-authority):

```typescript
// src/components/ui/button.tsx
import { cva, type VariantProps } from 'class-variance-authority'
import { cn } from '@/lib/utils'

// 1. Definir variantes con cva()
const buttonVariants = cva(
  'base-classes...',  // Clases base comunes
  {
    variants: {
      variant: { default: '...', secondary: '...', outline: '...', ghost: '...' },
      size: { default: '...', sm: '...', lg: '...', icon: '...' },
    },
    defaultVariants: { variant: 'default', size: 'default' },
  }
)

// 2. Tipar props con VariantProps
interface ButtonProps
  extends React.ButtonHTMLAttributes<HTMLButtonElement>,
    VariantProps<typeof buttonVariants> {}

// 3. Usar cn() para combinar clases
export const Button = ({ className, variant, size, ...props }: ButtonProps) => (
  <button className={cn(buttonVariants({ variant, size, className }))} {...props} />
)
```

> **Variantes específicas:** Ver `design-system.md` sección Componentes > Botones

### Componente de Feature

```typescript
// src/features/templates/components/TemplateCard.tsx
import { cn } from '@/lib/utils'
import type { Template } from '../types'

interface TemplateCardProps {
  template: Template
  onClick?: () => void
  className?: string
}

export const TemplateCard = ({ template, onClick, className }: TemplateCardProps) => {
  return (
    <div
      className={cn('cursor-pointer rounded-lg border p-4 hover:shadow-md', className)}
      onClick={onClick}
    >
      <h3 className="font-medium">{template.title}</h3>
    </div>
  )
}
```

### Store de Zustand

```typescript
// src/stores/theme-store.ts
import { create } from 'zustand'
import { persist } from 'zustand/middleware'

type Theme = 'light' | 'dark' | 'system'

interface ThemeState {
  theme: Theme
  setTheme: (theme: Theme) => void
}

export const useThemeStore = create<ThemeState>()(
  persist(
    (set) => ({
      theme: 'system',
      setTheme: (theme) => set({ theme }),
    }),
    { name: 'theme-storage' }
  )
)
```

### Tipos de Feature

```typescript
// src/features/templates/types/index.ts
export interface Template {
  id: string
  title: string
  folderId?: string
  createdAt: string
}

export interface TemplateVersion {
  id: string
  templateId: string
  status: 'DRAFT' | 'PUBLISHED' | 'ARCHIVED'
}
```

### Ruta

```typescript
// src/routes/_app/workspace/$workspaceId/templates/index.tsx
import { createFileRoute } from '@tanstack/react-router'
import { TemplatesPage } from '@/features/templates/components/TemplatesPage'

export const Route = createFileRoute('/_app/workspace/$workspaceId/templates/')({
  component: TemplatesPage,
})
```

---

## 6. Utilidades Base

```typescript
// src/lib/utils.ts
import { type ClassValue, clsx } from 'clsx'
import { twMerge } from 'tailwind-merge'

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}
```

---

## 7. Variables de Entorno

```bash
# .env.example
VITE_API_URL=http://localhost:8080/api/v1
VITE_KEYCLOAK_URL=http://localhost:8180
VITE_KEYCLOAK_REALM=doc-assembly
VITE_KEYCLOAK_CLIENT_ID=web-client
VITE_USE_MOCK_AUTH=false
```

---

## 8. Convenciones

### Nombres de Archivos
- **Componentes**: PascalCase (`TemplateCard.tsx`)
- **Hooks**: camelCase con `use` (`useTemplates.ts`)
- **Utilidades**: kebab-case (`date-utils.ts`)
- **Tipos**: `index.ts` en carpeta `types/`

### Imports
- Usar alias `@/` para imports desde `src/`
- Imports relativos solo dentro de la misma feature

### Estilos
- Usar Tailwind CSS para todos los estilos
- Usar `cn()` para combinar clases condicionales
- No usar CSS modules ni styled-components
- **Ver `design-system.md`** para tokens y patrones visuales

---

## 9. Checklist de Implementación

### Fase 1: Setup
- [ ] Crear proyecto con Vite
- [ ] Instalar dependencias
- [ ] Configurar Vite, TypeScript, Tailwind
- [ ] Crear estructura de carpetas

### Fase 2: Componentes UI
- [ ] Migrar componentes UI del diseño
- [ ] Crear layouts (AppLayout, AdminLayout)
- [ ] Implementar navegación

### Fase 3: Features (solo estructura)
- [ ] Crear carpetas de features
- [ ] Definir tipos base
- [ ] Crear componentes placeholder

### Fase 4: Rutas
- [ ] Crear root route
- [ ] Crear rutas de app
- [ ] Crear rutas de admin

---

## 10. Guías de Referencia

| Documento | Contenido |
|-----------|-----------|
| `design-system.md` | Filosofía visual, colores, tipografía, componentes, espaciado |
| `src/components/ui/` | Componentes base reutilizables (Button, Input, Card, Dialog...) |
| `AGENTS.md` | Instrucciones para agentes IA |

### Proceso para Crear Componentes

1. **Revisar `design-system.md`** para entender el estilo visual
2. **Reutilizar componentes** de `src/components/ui/` cuando sea posible
3. **Seguir patrones de código** de esta guía (`cva`, `cn`, tipos TypeScript)

