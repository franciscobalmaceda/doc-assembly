import {
  PANEL_COLLAPSED_WIDTH,
  ROLES_EXPANDED_WIDTH,
  VARIABLES_EXPANDED_WIDTH,
} from './panel-widths'

export const DOCUMENT_EDITOR_GRID_BASE_CLASS =
  'grid grid-rows-[auto_1fr] h-full w-full min-w-0 overflow-hidden transition-[grid-template-columns] duration-200 ease-[cubic-bezier(0.4,0,0.2,1)]'

export const DOCUMENT_EDITOR_GRID_EDITABLE_CLASS =
  'grid-cols-[auto_minmax(0,1fr)_auto]'

export const DOCUMENT_EDITOR_GRID_READ_ONLY_CLASS =
  'grid-cols-[minmax(0,1fr)_auto]'

export function getDocumentEditorGridClass(editable: boolean): string {
  return [
    DOCUMENT_EDITOR_GRID_BASE_CLASS,
    editable
      ? DOCUMENT_EDITOR_GRID_EDITABLE_CLASS
      : DOCUMENT_EDITOR_GRID_READ_ONLY_CLASS,
  ].join(' ')
}

interface DocumentEditorGridTemplateColumnsParams {
  editable: boolean
  variablesCollapsed: boolean
  rolesCollapsed: boolean
}

const CENTER_COLUMN = 'minmax(0,1fr)'

export function getDocumentEditorGridTemplateColumns({
  editable,
  variablesCollapsed,
  rolesCollapsed,
}: DocumentEditorGridTemplateColumnsParams): string {
  const rolesWidth = rolesCollapsed ? PANEL_COLLAPSED_WIDTH : ROLES_EXPANDED_WIDTH

  if (!editable) {
    return `${CENTER_COLUMN} ${rolesWidth}px`
  }

  const variablesWidth = variablesCollapsed
    ? PANEL_COLLAPSED_WIDTH
    : VARIABLES_EXPANDED_WIDTH

  return `${variablesWidth}px ${CENTER_COLUMN} ${rolesWidth}px`
}
