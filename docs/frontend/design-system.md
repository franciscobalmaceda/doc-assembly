# Design System

Guía de referencia para desarrolladores. Este documento describe los patrones visuales del proyecto para crear componentes consistentes.

---

## Filosofía de Diseño

### Estilo: Minimalista Técnico Corporativo

El diseño de esta aplicación sigue un estilo **austero, preciso y profesional**, optimizado para workflows de trabajo documental.

| Eje                       | Posición        | Descripción                                               |
| ------------------------- | --------------- | --------------------------------------------------------- |
| Minimalista ↔ Ornamentado | **Minimalista** | Sin gradientes, sombras mínimas, bordes planos            |
| Moderno ↔ Clásico         | **Moderno**     | CSS variables, animaciones sutiles, tipografía geométrica |
| Corporativo ↔ Juguetón    | **Corporativo** | Serio, sin emojis ni ilustraciones decorativas            |
| Contraste                 | **Alto**        | Jerarquía visual clara, colores bien definidos            |
| Geometría                 | **Angular**     | Esquinas rectas en elementos clave, formas limpias        |

### Elementos Distintivos

| Elemento          | Implementación                                      | Efecto Visual            |
| ----------------- | --------------------------------------------------- | ------------------------ |
| Labels técnicos   | `font-mono uppercase tracking-widest`               | Formalidad, precisión    |
| Botones de dialog | `rounded-none` (esquinas rectas)                    | Minimalismo, decisión    |
| Botones primarios | `bg-foreground text-background`                     | Alto contraste, claridad |
| Separadores       | Bordes (`border-t`, `border-b`) en lugar de sombras | Estructura plana         |
| Animaciones       | `fade + zoom 95%`                                   | Sutiles, no dramáticas   |

### Palabras Clave

**Austero** · **Técnico** · **Profesional** · **Preciso** · **Funcional** · **Limpio** · **Geométrico** · **Confiable**

### Qué Evitar

- Gradientes decorativos
- Sombras excesivas o difusas
- Emojis en la interfaz
- Colores vibrantes fuera de los tokens semánticos
- Esquinas muy redondeadas en elementos de acción
- Animaciones llamativas o largas

---

## Paleta de Colores

Todos los colores están definidos como variables CSS en formato HSL en `src/index.css`.

### Colores Principales

| Token         | Uso                                              | Referencia     |
| ------------- | ------------------------------------------------ | -------------- |
| `--primary`   | Acciones principales, botones primarios, enlaces | Azul saturado  |
| `--secondary` | Botones secundarios, acciones alternativas       | Gris claro     |
| `--accent`    | Estados hover/focus, resaltados                  | Gris muy claro |
| `--muted`     | Texto secundario, placeholders                   | Gris medio     |

### Colores Semánticos

| Token           | Uso                                    |
| --------------- | -------------------------------------- |
| `--destructive` | Acciones peligrosas, errores, eliminar |
| `--success`     | Confirmaciones, estados exitosos       |
| `--warning`     | Alertas, advertencias                  |
| `--info`        | Información, mensajes informativos     |

### Colores Especiales

| Token           | Uso                                                 |
| --------------- | --------------------------------------------------- |
| `--admin`       | Elementos relacionados con administración (púrpura) |
| `--role`        | Roles de firmantes, inyectables (rosa)              |
| `--accent-blue` | Elementos de workspace                              |

### Neutros

| Token          | Uso                                            |
| -------------- | ---------------------------------------------- |
| `--background` | Fondo de página                                |
| `--foreground` | Texto principal                                |
| `--card`       | Fondo de tarjetas                              |
| `--popover`    | Fondo de popovers/dropdowns                    |
| `--border`     | Bordes estándar                                |
| `--input`      | Bordes de inputs                               |
| `--ring`       | Anillo de focus (gris sutil, no azul primario) |

> **Archivo fuente:** `src/index.css` (secciones `:root` y `.dark`)

---

## Tipografía

### Familias de Fuentes

| Clase          | Familia          | Uso                                      |
| -------------- | ---------------- | ---------------------------------------- |
| (default)      | Inter            | Texto de cuerpo, UI general              |
| `font-display` | Space Grotesk    | Títulos destacados                       |
| `font-mono`    | System monospace | Etiquetas, código, labels de formularios |

### Patrones de Texto

| Patrón               | Clases Tailwind                                                                     | Uso                                |
| -------------------- | ----------------------------------------------------------------------------------- | ---------------------------------- |
| Etiquetas de dialog  | `font-mono text-sm font-medium uppercase tracking-widest`                           | Títulos de diálogos                |
| Labels de formulario | `font-mono text-[10px] font-medium uppercase tracking-widest text-muted-foreground` | Labels en formularios minimalistas |
| Título de card       | `text-2xl font-semibold`                                                            | Encabezados de tarjetas            |
| Descripción          | `text-sm text-muted-foreground`                                                     | Texto secundario                   |
| Badge                | `text-xs font-semibold`                                                             | Etiquetas/badges                   |

> **Archivo fuente:** `src/index.css` (sección `@theme`)

---

## Border Radius

Base: `--radius: 0.5rem` (8px)

| Clase          | Valor | Uso                                        |
| -------------- | ----- | ------------------------------------------ |
| `rounded-sm`   | 4px   | Menu items, checkboxes, elementos pequeños |
| `rounded-md`   | 6px   | Botones, inputs, selects, dropdowns        |
| `rounded-lg`   | 8px   | Cards, alerts, contenedores principales    |
| `rounded-full` | 50%   | Avatars, badges pill, switch toggles       |
| `rounded-none` | 0px   | Botones de dialogs (estilo minimalista)    |

> **Archivo fuente:** `src/index.css` (variables `--radius-*`)

---

## Bordes

| Patrón         | Clases                   | Uso                            |
| -------------- | ------------------------ | ------------------------------ |
| Borde estándar | `border border-border`   | Cards, containers              |
| Borde de input | `border border-input`    | Inputs, selects                |
| Borde inferior | `border-b border-border` | Headers de dialog, separadores |
| Borde superior | `border-t border-border` | Footers de dialog              |

### Sombras

| Clase       | Uso                             |
| ----------- | ------------------------------- |
| `shadow-sm` | Tabs activos, elementos sutiles |
| `shadow-md` | Dropdowns, popovers             |
| `shadow-lg` | Dialogs, modales                |

---

## Espaciado

### Cards

| Sección | Padding                                          |
| ------- | ------------------------------------------------ |
| Header  | `p-6` con `space-y-1.5` entre título/descripción |
| Content | `p-6 pt-0`                                       |
| Footer  | `p-6 pt-0`                                       |

> **Archivo fuente:** `src/components/ui/card.tsx`

### Dialogs

| Sección | Padding                                     |
| ------- | ------------------------------------------- |
| Header  | `p-6` con `border-b`                        |
| Content | `p-6` con `space-y-6` entre campos          |
| Footer  | `p-6` con `border-t`, `gap-3` entre botones |

> **Archivo fuente:** `src/components/ui/dialog.tsx`

### Elementos de Formulario

| Elemento      | Padding       |
| ------------- | ------------- |
| Input/Select  | `px-3 py-2`   |
| Menu item     | `px-2 py-1.5` |
| Botón sm      | `px-3`        |
| Botón default | `px-4 py-2`   |
| Botón lg      | `px-8`        |

---

## Componentes

### Botones

**Variantes:**

| Variant       | Descripción                         |
| ------------- | ----------------------------------- |
| `default`     | Fondo primary, texto blanco         |
| `secondary`   | Fondo secondary, texto oscuro       |
| `outline`     | Borde visible, fondo transparente   |
| `ghost`       | Sin borde ni fondo, solo hover      |
| `destructive` | Fondo rojo para acciones peligrosas |
| `link`        | Estilo de enlace con underline      |

**Tamaños:**

| Size      | Altura    |
| --------- | --------- |
| `sm`      | h-9       |
| `default` | h-10      |
| `lg`      | h-11      |
| `xl`      | h-12      |
| `icon`    | h-10 w-10 |

> **Archivo fuente:** `src/components/ui/button.tsx`

### Badges

**Variantes:**

| Variant       | Uso                        |
| ------------- | -------------------------- |
| `default`     | Badge primario             |
| `secondary`   | Badge secundario/neutro    |
| `destructive` | Errores, estados negativos |
| `outline`     | Solo borde                 |
| `draft`       | Estado borrador (amarillo) |
| `published`   | Estado publicado (verde)   |
| `archived`    | Estado archivado (gris)    |

> **Archivo fuente:** `src/components/ui/badge.tsx`

### Inputs

- Border radius: `rounded-md`
- Altura: Implícita via padding
- Focus: `focus-visible:ring-2 ring-ring`

> **Archivo fuente:** `src/components/ui/input.tsx`

---

## Dialogs y Modales

### Estructura Base

```plaintext
┌─────────────────────────────────┐
│ Header (border-b, p-6)          │
│   - Título (font-mono uppercase)│
│   - Descripción (text-sm muted) │
│   - Botón cerrar (X)            │
├─────────────────────────────────┤
│ Content (p-6, space-y-6)        │
│   - Campos del formulario       │
├─────────────────────────────────┤
│ Footer (border-t, p-6)          │
│   - Botones (justify-end gap-3) │
└─────────────────────────────────┘
```

### Estilos del Container

| Propiedad     | Valor              |
| ------------- | ------------------ |
| Ancho máximo  | `max-w-lg` (512px) |
| Border radius | `sm:rounded-lg`    |
| Sombra        | `shadow-lg`        |
| Borde         | `border`           |
| Z-index       | `z-50`             |

### Overlay

- Fondo: `bg-black/80`
- Animación: fade in/out

### Animaciones

- Apertura: `fade-in-0 zoom-in-95` (0.25s ease-out)
- Cierre: `fade-out-0 zoom-out-95` (0.2s ease-in)

> **Archivo fuente:** `src/components/ui/dialog.tsx`

### Botones de Dialog (Estilo Minimalista)

Los dialogs del proyecto usan un estilo minimalista para botones:

| Tipo     | Características                                                          |
| -------- | ------------------------------------------------------------------------ |
| Cancelar | `rounded-none border bg-background font-mono text-xs uppercase`          |
| Primario | `rounded-none bg-foreground text-background font-mono text-xs uppercase` |

> **Ejemplos:** `src/features/documents/components/CreateFolderDialog.tsx`

---

## Estados Interactivos

### Focus

```plaintext
focus-visible:outline-none
focus-visible:ring-2
focus-visible:ring-ring
focus-visible:ring-offset-2
```

> **Nota**: Usar siempre `focus-visible:` en lugar de `focus:` para elementos interactivos como inputs, botones y selects. Esto evita que aparezca el ring al hacer clic con el mouse, mostrándolo solo cuando se navega con teclado. Los elementos de menú/dropdown usan `focus:` para la navegación con teclado.

### Hover

| Componente          | Patrón                                         |
| ------------------- | ---------------------------------------------- |
| Botón primary       | `hover:bg-primary/90`                          |
| Botón outline/ghost | `hover:bg-accent hover:text-accent-foreground` |
| Menu item           | `focus:bg-accent focus:text-accent-foreground` |

### Disabled

```plaintext
disabled:pointer-events-none
disabled:opacity-50
```

### Transiciones

- General: `transition-colors duration-200`
- Listas/menús: `transition-colors` (más rápido)

---

## Z-Index

| Capa      | Z-Index    | Uso                |
| --------- | ---------- | ------------------ |
| Popovers  | `z-[9999]` | Tooltips, popovers |
| Toasts    | `z-[100]`  | Notificaciones     |
| Dialogs   | `z-50`     | Modales            |
| Dropdowns | `z-50`     | Menús desplegables |

---

## Dark Mode

- Se activa con clase `.dark` en `<html>`
- Todos los colores se ajustan automáticamente via CSS variables
- **Nunca usar colores hardcodeados** - siempre usar tokens semánticos

### Ejemplo Correcto

```tsx
// Correcto - usa tokens
className = "bg-background text-foreground border-border";

// Incorrecto - colores hardcodeados
className = "bg-white text-gray-900 border-gray-200";
```

> **Archivo fuente:** `src/stores/theme-store.ts`

---

## Scrollbar

| Propiedad     | Valor           |
| ------------- | --------------- |
| Ancho         | 6px             |
| Color thumb   | `var(--border)` |
| Border radius | 3px             |

> **Archivo fuente:** `src/index.css` (sección scrollbar)

---

## Checklist para Nuevos Componentes

1. Usar tokens de color semánticos (`--primary`, `--border`, etc.)
2. Aplicar border radius consistente (`rounded-md` para interactivos)
3. Incluir estados focus-visible con ring
4. Agregar transiciones para estados hover
5. Soportar dark mode via CSS variables
6. Seguir escala de espaciado (múltiplos de 4px)
7. Usar `font-mono uppercase tracking-widest` para labels si aplica

---

## Referencias Rápidas

| Recurso             | Ubicación                               |
| ------------------- | --------------------------------------- |
| Variables CSS       | `src/index.css`                         |
| Componentes UI      | `src/components/ui/`                    |
| Store de tema       | `src/stores/theme-store.ts`             |
| Ejemplos de dialogs | `src/features/*/components/*Dialog.tsx` |
