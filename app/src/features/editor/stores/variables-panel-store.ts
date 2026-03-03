import { create } from 'zustand'

export interface VariablesPanelStore {
  /**
   * Estado de colapso del panel de variables
   * - false: panel expandido (responsive clamp)
   * - true: panel colapsado (56px)
   */
  isCollapsed: boolean

  /**
   * Alterna el estado de colapso del panel
   */
  toggleCollapsed: () => void

  /**
   * Establece el estado de colapso del panel
   */
  setCollapsed: (collapsed: boolean) => void

  /**
   * Resetea el store al estado inicial
   */
  reset: () => void
}

const initialState = {
  isCollapsed: false,
}

export const useVariablesPanelStore = create<VariablesPanelStore>()((set) => ({
  ...initialState,

  toggleCollapsed: () => {
    set((state) => ({ isCollapsed: !state.isCollapsed }))
  },

  setCollapsed: (collapsed: boolean) => {
    set({ isCollapsed: collapsed })
  },

  reset: () => set(initialState),
}))
