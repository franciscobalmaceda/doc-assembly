import { describe, expect, it } from 'vitest'
import {
  DOCUMENT_EDITOR_GRID_BASE_CLASS,
  DOCUMENT_EDITOR_GRID_EDITABLE_CLASS,
  DOCUMENT_EDITOR_GRID_READ_ONLY_CLASS,
  getDocumentEditorGridClass,
} from './document-editor-grid'

describe('getDocumentEditorGridClass', () => {
  it('uses minmax center column for editable mode', () => {
    const className = getDocumentEditorGridClass(true)

    expect(className).toContain(DOCUMENT_EDITOR_GRID_BASE_CLASS)
    expect(className).toContain(DOCUMENT_EDITOR_GRID_EDITABLE_CLASS)
    expect(className).toContain('w-full')
    expect(className).toContain('min-w-0')
  })

  it('uses minmax center column for read-only mode', () => {
    const className = getDocumentEditorGridClass(false)

    expect(className).toContain(DOCUMENT_EDITOR_GRID_BASE_CLASS)
    expect(className).toContain(DOCUMENT_EDITOR_GRID_READ_ONLY_CLASS)
    expect(className).toContain('w-full')
    expect(className).toContain('min-w-0')
  })
})
