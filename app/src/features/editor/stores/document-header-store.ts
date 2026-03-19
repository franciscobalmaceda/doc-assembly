import { create } from 'zustand'

// =============================================================================
// Types
// =============================================================================

export type DocumentHeaderLayout = 'image-left' | 'image-right' | 'image-center'

export interface DocumentHeaderState {
  enabled: boolean
  layout: DocumentHeaderLayout
  imageUrl: string | null
  imageAlt: string
  text: string
}

export interface DocumentHeaderActions {
  setEnabled: (enabled: boolean) => void
  setLayout: (layout: DocumentHeaderLayout) => void
  setImage: (url: string, alt: string) => void
  setText: (text: string) => void
  reset: () => void
  configure: (partial: Partial<DocumentHeaderState>) => void
}

export type DocumentHeaderStore = DocumentHeaderState & DocumentHeaderActions

// =============================================================================
// Initial State
// =============================================================================

const initialState: DocumentHeaderState = {
  enabled: false,
  layout: 'image-left',
  imageUrl: null,
  imageAlt: '',
  text: '',
}

// =============================================================================
// Store
// =============================================================================

export const useDocumentHeaderStore = create<DocumentHeaderStore>()((set) => ({
  ...initialState,

  setEnabled: (enabled) => set({ enabled }),

  setLayout: (layout) => set({ layout }),

  setImage: (imageUrl, imageAlt) => set({ imageUrl, imageAlt }),

  setText: (text) => set({ text }),

  reset: () => set(initialState),

  configure: (partial) => set((state) => ({ ...state, ...partial })),
}))
