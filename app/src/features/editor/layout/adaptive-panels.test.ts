import { describe, expect, it } from 'vitest'
import {
  decideAdaptivePanels,
  EDITOR_SURFACE_PADDING_X,
  LAYOUT_BUFFER,
  ROLES_EXPANDED,
  VARIABLES_COLLAPSED,
  VARIABLES_EXPANDED,
} from './adaptive-panels'

const pageSizeWidth = 794
const marginsLeft = 96
const marginsRight = 96

const centerMin =
  pageSizeWidth + marginsLeft + marginsRight + EDITOR_SURFACE_PADDING_X
const minBothExpanded =
  centerMin + VARIABLES_EXPANDED + ROLES_EXPANDED + LAYOUT_BUFFER
const minRolesPriority =
  centerMin + VARIABLES_COLLAPSED + ROLES_EXPANDED + LAYOUT_BUFFER

describe('decideAdaptivePanels', () => {
  it('does not collapse when width is sufficient', () => {
    const result = decideAdaptivePanels({
      availableWidth: minBothExpanded + 1,
      pageSizeWidth,
      marginsLeft,
      marginsRight,
      editable: true,
      variablesCollapsed: false,
      rolesCollapsed: false,
    })

    expect(result).toEqual({
      nextVariablesCollapsed: false,
      nextRolesCollapsed: false,
      reason: 'fits',
    })
  })

  it('collapses variables first when width is intermediate', () => {
    const result = decideAdaptivePanels({
      availableWidth: minBothExpanded - 1,
      pageSizeWidth,
      marginsLeft,
      marginsRight,
      editable: true,
      variablesCollapsed: false,
      rolesCollapsed: false,
    })

    expect(result).toEqual({
      nextVariablesCollapsed: true,
      nextRolesCollapsed: false,
      reason: 'collapse-variables',
    })
  })

  it('collapses both when width is critical', () => {
    const result = decideAdaptivePanels({
      availableWidth: minRolesPriority - 1,
      pageSizeWidth,
      marginsLeft,
      marginsRight,
      editable: true,
      variablesCollapsed: false,
      rolesCollapsed: false,
    })

    expect(result).toEqual({
      nextVariablesCollapsed: true,
      nextRolesCollapsed: true,
      reason: 'collapse-both',
    })
  })

  it('does not auto-expand when width recovers and noAutoExpand is true', () => {
    const result = decideAdaptivePanels({
      availableWidth: minBothExpanded + 200,
      pageSizeWidth,
      marginsLeft,
      marginsRight,
      editable: true,
      variablesCollapsed: true,
      rolesCollapsed: true,
      noAutoExpand: true,
    })

    expect(result).toEqual({
      nextVariablesCollapsed: true,
      nextRolesCollapsed: true,
      reason: 'fits',
    })
  })

  it('ignores variables panel in read-only mode and collapses roles if needed', () => {
    const minReadOnly =
      centerMin + ROLES_EXPANDED + LAYOUT_BUFFER

    const result = decideAdaptivePanels({
      availableWidth: minReadOnly - 1,
      pageSizeWidth,
      marginsLeft,
      marginsRight,
      editable: false,
      variablesCollapsed: false,
      rolesCollapsed: false,
    })

    expect(result).toEqual({
      nextVariablesCollapsed: false,
      nextRolesCollapsed: true,
      reason: 'collapse-both',
    })
  })
})
