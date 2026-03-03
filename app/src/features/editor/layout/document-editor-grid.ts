export const DOCUMENT_EDITOR_GRID_BASE_CLASS =
  'grid grid-rows-[auto_1fr] h-full w-full min-w-0'

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
