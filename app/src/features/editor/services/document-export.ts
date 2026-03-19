/**
 * Document Export Service
 *
 * Exports the editor content and metadata into a portable JSON format.
 * This must replicate the exact format from the old system (web-client)
 * because it's the same format used by the backend.
 *
 * Variables are stored as IDs only; full definitions come from the backend.
 */

// @ts-expect-error - tiptap types incompatible with moduleResolution: bundler
import type { Editor, JSONContent } from '@tiptap/core'
import type {
  PortableDocument,
  DocumentMeta,
  PageConfig,
  ExportOptions,
  ExportInfo,
  ProseMirrorDocument,
  ProseMirrorNode,
  SignerRoleDefinition,
  SigningWorkflowConfig,
} from '../types/document-format'
import type { PaginationStore } from '../stores/pagination-store'
import { useDocumentHeaderStore } from '../stores/document-header-store'
import { DOCUMENT_FORMAT_VERSION } from '../types/document-format'
import { PAGE_SIZES } from '../types'

// =============================================================================
// Helper Types
// =============================================================================

interface EditorStoreData {
  pagination: Pick<PaginationStore, 'pageSize' | 'margins'>
  signerRoles: SignerRoleDefinition[]
  workflowConfig: SigningWorkflowConfig
}

// =============================================================================
// Content Extraction
// =============================================================================

/**
 * Extracts ProseMirror JSON content from the editor
 */
function extractContent(editor: Editor): ProseMirrorDocument {
  const json = editor.getJSON() as JSONContent

  return {
    type: 'doc',
    content: (json.content || []) as ProseMirrorNode[],
  }
}

/**
 * Recursively traverses content to find all injector nodes and extract variable references
 */
function findInjectorNodes(content: ProseMirrorNode[]): Set<string> {
  const variableIds = new Set<string>()

  function traverse(nodes: ProseMirrorNode[]) {
    for (const node of nodes) {
      if (node.type === 'injector' && node.attrs?.variableId) {
        variableIds.add(node.attrs.variableId as string)
      }

      // Also check conditional nodes for variable references in conditions
      if (node.type === 'conditional' && node.attrs?.conditions) {
        extractVariablesFromConditions(node.attrs.conditions, variableIds)
      }

      if (node.content) {
        traverse(node.content)
      }
    }
  }

  traverse(content)
  return variableIds
}

/**
 * Extracts variable IDs from conditional logic
 */
function extractVariablesFromConditions(
  conditions: unknown,
  variableIds: Set<string>
): void {
  if (!conditions || typeof conditions !== 'object') return

  const group = conditions as { type?: string; variableId?: string; children?: unknown[] }

  if (group.type === 'rule' && group.variableId) {
    variableIds.add(group.variableId)
  }

  if (group.type === 'group' && Array.isArray(group.children)) {
    for (const child of group.children) {
      extractVariablesFromConditions(child, variableIds)
    }
  }
}

/**
 * Extracts variable IDs directly from the editor instance
 * Used by the preview modal to filter only variables actually used in the document
 */
export function extractVariableIdsFromEditor(editor: Editor): string[] {
  const json = editor.getJSON() as JSONContent
  const content = (json.content || []) as ProseMirrorNode[]
  return Array.from(findInjectorNodes(content))
}

/**
 * Extracts unique variable IDs from the document content
 * Only returns IDs; full definitions come from the backend
 */
function extractVariableIds(content: ProseMirrorDocument): string[] {
  const usedVariableIds = findInjectorNodes(content.content)
  return Array.from(usedVariableIds).sort()
}

// =============================================================================
// Page Configuration
// =============================================================================

/**
 * Gets the page format ID (key) from PAGE_SIZES based on dimensions
 * Returns 'CUSTOM' if no matching format is found
 */
function getPageFormatId(pageSize: { width: number; height: number }): PageConfig['formatId'] {
  const entry = Object.entries(PAGE_SIZES).find(
    ([_, size]) => size.width === pageSize.width && size.height === pageSize.height
  )
  return (entry?.[0] as PageConfig['formatId']) || 'CUSTOM'
}

/**
 * Converts pagination store config to PageConfig format
 */
function extractPageConfig(pagination: EditorStoreData['pagination']): PageConfig {
  const { pageSize, margins } = pagination

  return {
    formatId: getPageFormatId(pageSize),
    width: pageSize.width,
    height: pageSize.height,
    margins: { ...margins },
  }
}

// =============================================================================
// Export Info
// =============================================================================

/**
 * Generates export metadata
 */
function generateExportInfo(options: ExportOptions = {}): ExportInfo {
  const info: ExportInfo = {
    exportedAt: new Date().toISOString(),
    sourceApp: 'doc-assembly-web/1.1.0',
  }

  if (options.exportedBy) {
    info.exportedBy = options.exportedBy
  }

  return info
}

/**
 * Generates a simple checksum for the document
 */
function generateChecksum(content: string): string {
  let hash = 0
  for (let i = 0; i < content.length; i++) {
    const char = content.charCodeAt(i)
    hash = ((hash << 5) - hash) + char
    hash = hash & hash // Convert to 32bit integer
  }
  return Math.abs(hash).toString(16).padStart(8, '0')
}

// =============================================================================
// Main Export Functions
// =============================================================================

/**
 * Exports the complete document with all metadata
 * Variables are stored as IDs only; full definitions come from the backend
 */
export function exportDocument(
  editor: Editor,
  storeData: EditorStoreData,
  meta: DocumentMeta,
  options: ExportOptions = {}
): PortableDocument {
  // Extract content from editor
  const content = extractContent(editor)

  // Extract variable IDs used in the document
  const variableIds = extractVariableIds(content)

  // Extract page configuration
  const pageConfig = extractPageConfig(storeData.pagination)

  // Get signer roles
  const signerRoles = [...storeData.signerRoles]

  // Generate export info
  const exportInfo = generateExportInfo(options)

  // Read header state
  const headerState = useDocumentHeaderStore.getState()

  // Assemble the document
  const document: PortableDocument = {
    version: DOCUMENT_FORMAT_VERSION,
    meta,
    pageConfig,
    variableIds,
    signerRoles,
    signingWorkflow: storeData.workflowConfig,
    content,
    exportInfo,
    header: {
      enabled: headerState.enabled,
      layout: headerState.layout,
      imageUrl: headerState.imageUrl,
      imageAlt: headerState.imageAlt,
      text: headerState.text,
    },
  }

  // Add checksum if requested
  if (options.includeChecksum) {
    const contentString = JSON.stringify(document.content)
    document.exportInfo.checksum = generateChecksum(contentString)
  }

  return document
}

/**
 * Serializes the document to JSON string
 */
export function serializeDocument(
  document: PortableDocument,
  prettyPrint: boolean = true
): string {
  return prettyPrint
    ? JSON.stringify(document, null, 2)
    : JSON.stringify(document)
}

/**
 * Downloads the document as a JSON file
 */
export function downloadAsJson(
  document: PortableDocument,
  filename: string = 'document.json'
): void {
  const json = serializeDocument(document, true)
  const blob = new Blob([json], { type: 'application/json' })
  const url = URL.createObjectURL(blob)

  const a = window.document.createElement('a')
  a.href = url
  a.download = filename.endsWith('.json') ? filename : `${filename}.json`
  window.document.body.appendChild(a)
  a.click()
  window.document.body.removeChild(a)
  URL.revokeObjectURL(url)
}

/**
 * Convenience function to export and download in one step
 */
export function exportAndDownload(
  editor: Editor,
  storeData: EditorStoreData,
  meta: DocumentMeta,
  filename: string = 'document.json',
  options: ExportOptions = {}
): PortableDocument {
  const document = exportDocument(editor, storeData, meta, options)
  downloadAsJson(document, filename)
  return document
}

// =============================================================================
// Utility Functions
// =============================================================================

/**
 * Gets a summary of document contents for preview
 */
export function getDocumentSummary(document: PortableDocument): {
  variableCount: number
  signerRoleCount: number
  pageFormat: string
  hasConditionals: boolean
  hasSignatures: boolean
} {
  const hasConditionals = hasNodeType(document.content.content, 'conditional')
  const hasSignatures = hasNodeType(document.content.content, 'signature')

  return {
    variableCount: document.variableIds.length,
    signerRoleCount: document.signerRoles.length,
    pageFormat: document.pageConfig.formatId,
    hasConditionals,
    hasSignatures,
  }
}

/**
 * Checks if document contains a specific node type
 */
function hasNodeType(content: ProseMirrorNode[], nodeType: string): boolean {
  for (const node of content) {
    if (node.type === nodeType) return true
    if (node.content && hasNodeType(node.content, nodeType)) return true
  }
  return false
}
