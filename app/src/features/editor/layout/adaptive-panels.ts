export const VARIABLES_EXPANDED = 320
export const VARIABLES_COLLAPSED = 56
export const ROLES_EXPANDED = 360
export const ROLES_COLLAPSED = 56
export const EDITOR_SURFACE_PADDING_X = 64
export const LAYOUT_BUFFER = 24

export type AdaptivePanelsPriority = 'roles' | 'variables'
export type AdaptivePanelsReason =
  | 'fits'
  | 'collapse-variables'
  | 'collapse-both'

export interface AdaptivePanelsInput {
  availableWidth: number
  pageSizeWidth: number
  marginsLeft: number
  marginsRight: number
  editable: boolean
  variablesCollapsed: boolean
  rolesCollapsed: boolean
  priority?: AdaptivePanelsPriority
  noAutoExpand?: boolean
}

export interface AdaptivePanelsDecision {
  nextVariablesCollapsed: boolean
  nextRolesCollapsed: boolean
  reason: AdaptivePanelsReason
}

export function decideAdaptivePanels({
  availableWidth,
  pageSizeWidth,
  marginsLeft,
  marginsRight,
  editable,
  variablesCollapsed,
  rolesCollapsed,
  priority = 'roles',
  noAutoExpand = true,
}: AdaptivePanelsInput): AdaptivePanelsDecision {
  const centerMin =
    pageSizeWidth + marginsLeft + marginsRight + EDITOR_SURFACE_PADDING_X

  const expandedVariableWidth = editable ? VARIABLES_EXPANDED : 0
  const collapsedVariableWidth = editable ? VARIABLES_COLLAPSED : 0

  const minBothExpanded =
    centerMin + expandedVariableWidth + ROLES_EXPANDED + LAYOUT_BUFFER
  const minRolesPriority =
    centerMin + collapsedVariableWidth + ROLES_EXPANDED + LAYOUT_BUFFER

  if (availableWidth < minRolesPriority) {
    return {
      nextVariablesCollapsed: editable ? true : variablesCollapsed,
      nextRolesCollapsed: true,
      reason: 'collapse-both',
    }
  }

  if (availableWidth < minBothExpanded) {
    if (priority === 'roles') {
      return {
        nextVariablesCollapsed: editable ? true : variablesCollapsed,
        nextRolesCollapsed: noAutoExpand ? rolesCollapsed : false,
        reason: 'collapse-variables',
      }
    }

    return {
      nextVariablesCollapsed: noAutoExpand ? variablesCollapsed : false,
      nextRolesCollapsed: true,
      reason: 'collapse-variables',
    }
  }

  if (noAutoExpand) {
    return {
      nextVariablesCollapsed: variablesCollapsed,
      nextRolesCollapsed: rolesCollapsed,
      reason: 'fits',
    }
  }

  return {
    nextVariablesCollapsed: editable ? false : variablesCollapsed,
    nextRolesCollapsed: false,
    reason: 'fits',
  }
}
