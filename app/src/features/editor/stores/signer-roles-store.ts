import { create } from 'zustand'
import { v4 as uuid } from 'uuid'
import type {
  SignerRoleDefinition,
  SignerRolesStore,
  SigningOrderMode,
  NotificationScope,
  NotificationTriggerMap,
  SigningWorkflowConfig,
} from '../types/signer-roles'
import {
  createEmptyRole,
  createDefaultWorkflowConfig,
  getDefaultParallelTriggers,
  getDefaultSequentialTriggers,
} from '../types/signer-roles'

const initialState = {
  roles: [] as SignerRoleDefinition[],
  isCollapsed: false,
  isCompactMode: false,
  workflowConfig: createDefaultWorkflowConfig(),
  activeInjectionTarget: null,
  // Selection mode
  isSelectionMode: false,
  selectedRoleIds: [] as string[],
}

export const useSignerRolesStore = create<SignerRolesStore>()((set, get) => ({
  ...initialState,

  setRoles: (roles) => set({ roles }),

  addRole: () => {
    const { roles } = get()
    const newOrder = roles.length + 1
    const newRole: SignerRoleDefinition = {
      id: uuid(),
      ...createEmptyRole(newOrder),
    }
    set({ roles: [...roles, newRole] })
  },

  updateRole: (id, updates) => {
    set((state) => ({
      roles: state.roles.map((role) =>
        role.id === id ? { ...role, ...updates } : role
      ),
    }))
  },

  deleteRole: (id) => {
    set((state) => {
      const filteredRoles = state.roles.filter((role) => role.id !== id)
      // Reordenar los roles restantes
      const reorderedRoles = filteredRoles.map((role, index) => ({
        ...role,
        order: index + 1,
      }))
      return {
        roles: reorderedRoles,
        activeInjectionTarget:
          state.activeInjectionTarget?.roleId === id
            ? null
            : state.activeInjectionTarget,
      }
    })
  },

  reorderRoles: (startIndex, endIndex) => {
    set((state) => {
      const roles = [...state.roles]
      const [removed] = roles.splice(startIndex, 1)
      roles.splice(endIndex, 0, removed)
      // Actualizar orden
      const reorderedRoles = roles.map((role, index) => ({
        ...role,
        order: index + 1,
      }))
      return { roles: reorderedRoles }
    })
  },

  setCollapsed: (collapsed) => {
    set({ isCollapsed: collapsed })
  },

  toggleCollapsed: () => {
    set((state) => ({ isCollapsed: !state.isCollapsed }))
  },

  toggleCompactMode: () => {
    set((state) => ({ isCompactMode: !state.isCompactMode }))
  },

  setActiveInjectionTarget: (target) => {
    set({ activeInjectionTarget: target })
  },

  clearActiveInjectionTarget: () => {
    set({ activeInjectionTarget: null })
  },

  reset: () => set(initialState),

  // Selection mode actions
  enterSelectionMode: (initialId?: string) => {
    set({
      isSelectionMode: true,
      selectedRoleIds: initialId ? [initialId] : [],
    })
  },

  exitSelectionMode: () => {
    set({
      isSelectionMode: false,
      selectedRoleIds: [],
    })
  },

  toggleRoleSelection: (id: string) => {
    set((state) => {
      const isSelected = state.selectedRoleIds.includes(id)
      const newSelectedIds = isSelected
        ? state.selectedRoleIds.filter((rid) => rid !== id)
        : [...state.selectedRoleIds, id]

      // Auto-exit selection mode if no items selected
      if (newSelectedIds.length === 0) {
        return {
          isSelectionMode: false,
          selectedRoleIds: [],
        }
      }

      return { selectedRoleIds: newSelectedIds }
    })
  },

  deleteSelectedRoles: () => {
    set((state) => {
      const filteredRoles = state.roles.filter(
        (role) => !state.selectedRoleIds.includes(role.id)
      )
      // Reorder remaining roles
      const reorderedRoles = filteredRoles.map((role, index) => ({
        ...role,
        order: index + 1,
      }))
      return {
        roles: reorderedRoles,
        activeInjectionTarget:
          state.activeInjectionTarget &&
          state.selectedRoleIds.includes(state.activeInjectionTarget.roleId)
            ? null
            : state.activeInjectionTarget,
        isSelectionMode: false,
        selectedRoleIds: [],
      }
    })
  },

  // Workflow actions
  setOrderMode: (mode: SigningOrderMode) => {
    set((state) => {
      const defaultTriggers =
        mode === 'parallel'
          ? getDefaultParallelTriggers()
          : getDefaultSequentialTriggers()

      return {
        workflowConfig: {
          ...state.workflowConfig,
          orderMode: mode,
          notifications: {
            ...state.workflowConfig.notifications,
            globalTriggers: defaultTriggers,
            // Reset individual configs with new mode defaults
            roleConfigs: state.workflowConfig.notifications.roleConfigs.map(
              (rc) => ({
                ...rc,
                triggers: defaultTriggers,
              })
            ),
          },
        },
      }
    })
  },

  setNotificationScope: (scope: NotificationScope) => {
    set((state) => {
      const { workflowConfig, roles } = state
      const defaultTriggers =
        workflowConfig.orderMode === 'parallel'
          ? getDefaultParallelTriggers()
          : getDefaultSequentialTriggers()

      // When switching to individual, initialize roleConfigs for all roles
      const roleConfigs =
        scope === 'individual'
          ? roles.map((role) => ({
              roleId: role.id,
              triggers:
                workflowConfig.notifications.roleConfigs.find(
                  (rc) => rc.roleId === role.id
                )?.triggers || defaultTriggers,
            }))
          : workflowConfig.notifications.roleConfigs

      return {
        workflowConfig: {
          ...workflowConfig,
          notifications: {
            ...workflowConfig.notifications,
            scope,
            roleConfigs,
          },
        },
      }
    })
  },

  updateGlobalTriggers: (triggers: NotificationTriggerMap) => {
    set((state) => ({
      workflowConfig: {
        ...state.workflowConfig,
        notifications: {
          ...state.workflowConfig.notifications,
          globalTriggers: triggers,
        },
      },
    }))
  },

  updateRoleTriggers: (roleId: string, triggers: NotificationTriggerMap) => {
    set((state) => {
      const { roleConfigs } = state.workflowConfig.notifications
      const existingIndex = roleConfigs.findIndex((rc) => rc.roleId === roleId)

      const newRoleConfigs =
        existingIndex >= 0
          ? roleConfigs.map((rc, i) =>
              i === existingIndex ? { ...rc, triggers } : rc
            )
          : [...roleConfigs, { roleId, triggers }]

      return {
        workflowConfig: {
          ...state.workflowConfig,
          notifications: {
            ...state.workflowConfig.notifications,
            roleConfigs: newRoleConfigs,
          },
        },
      }
    })
  },

  setWorkflowConfig: (config: SigningWorkflowConfig) => {
    set({ workflowConfig: config })
  },
}))

/**
 * Selector para obtener roles ordenados
 */
export const selectOrderedRoles = (state: SignerRolesStore) =>
  [...state.roles].sort((a, b) => a.order - b.order)

/**
 * Selector para obtener un rol por ID
 */
export const selectRoleById = (state: SignerRolesStore, id: string) =>
  state.roles.find((role) => role.id === id)
