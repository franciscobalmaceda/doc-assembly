/**
 * Portable Document Format for Document Assembly Editor
 *
 * This format is compatible with ProseMirror/TipTap JSON structure
 * and includes all metadata needed to fully restore a document.
 *
 * NOTE: This must replicate the exact format from the old system (web-client)
 * because it's the same format used by the backend.
 */

import type { LogicGroup } from '../extensions/Conditional/ConditionalExtension'
import type { SignatureCount, SignatureItem, SignatureLayout, SignatureLineWidth } from '../extensions/Signature/types'
import type { PageMargins } from './index'
import type {
  SigningWorkflowConfig,
  SignerRoleDefinition,
} from './signer-roles'

// =============================================================================
// Document Format Version
// =============================================================================

/**
 * Current document format version
 * - Major: Breaking changes to schema structure
 * - Minor: New optional fields added
 * - Patch: Bug fixes, clarifications
 *
 * Changelog:
 * - 1.1.0: Added signingWorkflow (orderMode, notifications)
 * - 1.0.0: Initial version
 */
export const DOCUMENT_FORMAT_VERSION = '1.1.0'

// =============================================================================
// Document Envelope
// =============================================================================

/**
 * Complete portable document format
 * Contains all data needed to fully restore a document in the editor
 *
 * NOTE: Variables are stored as IDs only. Full definitions come from the backend.
 */
export interface PortableDocument {
  /** Format version for migration support */
  version: string

  /** Document metadata */
  meta: DocumentMeta

  /** Page configuration */
  pageConfig: PageConfig

  /** Variable IDs used in document (references to backend variables) */
  variableIds: string[]

  /** Signer role definitions */
  signerRoles: SignerRoleDefinition[]

  /** Signing workflow configuration (order mode, notifications) */
  signingWorkflow: SigningWorkflowConfig

  /** ProseMirror-compatible content structure */
  content: ProseMirrorDocument

  /** Export metadata (auto-generated) */
  exportInfo: ExportInfo

  /** Optional document header (letterhead) above page margins */
  header?: {
    enabled: boolean
    layout?: 'image-left' | 'image-right' | 'image-center'
    imageUrl?: string | null
    imageAlt?: string
    text?: string
  }
}

// =============================================================================
// Document Metadata
// =============================================================================

export interface DocumentMeta {
  /** Document title */
  title: string

  /** Optional description */
  description?: string

  /** Document language (ISO 639-1) */
  language: 'en' | 'es'

  /** Custom metadata key-value pairs */
  customFields?: Record<string, string>
}

// =============================================================================
// Page Configuration
// =============================================================================

export type PageFormatId = 'A4' | 'LETTER' | 'LEGAL' | 'CUSTOM'

export interface PageConfig {
  /** Page format preset ID or 'CUSTOM' */
  formatId: PageFormatId

  /** Page width in pixels (96 DPI) */
  width: number

  /** Page height in pixels (96 DPI) */
  height: number

  /** Page margins in pixels */
  margins: PageMargins
}

// =============================================================================
// Backend Variable Types (source of truth from API)
// =============================================================================

export type VariableType = 'TEXT' | 'NUMBER' | 'DATE' | 'CURRENCY' | 'BOOLEAN' | 'IMAGE' | 'TABLE' | 'ROLE_TEXT'

export interface VariableValidation {
  /** Minimum value (NUMBER, DATE) */
  min?: number | string

  /** Maximum value (NUMBER, DATE) */
  max?: number | string

  /** Regex pattern (TEXT) */
  pattern?: string

  /** Allowed values (enum) */
  allowedValues?: string[]
}

/**
 * Variable definition from backend API
 * The document only stores variableIds; full definitions come from the backend
 */
export interface BackendVariable {
  /** Unique identifier */
  id: string

  /** Variable key (used in templates and stored in document) */
  variableId: string

  /** Human-readable label */
  label: string

  /** Data type */
  type: VariableType

  /** Whether this variable is required */
  required?: boolean

  /** Default value (type-appropriate) */
  defaultValue?: string | number | boolean

  /** Format specification (for DATE, CURRENCY) */
  format?: string

  /** Validation rules */
  validation?: VariableValidation
}

/**
 * Result of resolving variable IDs against backend variables
 */
export interface VariableResolutionResult {
  /** Variables that were found in the backend */
  resolved: BackendVariable[]

  /** Variable IDs in the document that don't exist in the backend */
  orphaned: string[]
}

// =============================================================================
// Export Info
// =============================================================================

export interface ExportInfo {
  /** ISO 8601 timestamp */
  exportedAt: string

  /** User who exported (optional, for audit) */
  exportedBy?: string

  /** Source application identifier */
  sourceApp: string

  /** Checksum for integrity verification */
  checksum?: string
}

// =============================================================================
// ProseMirror Document Structure
// =============================================================================

/**
 * Standard ProseMirror JSON document format
 * Compatible with editor.getJSON() output
 */
export interface ProseMirrorDocument {
  type: 'doc'
  content: ProseMirrorNode[]
}

export interface ProseMirrorMark {
  type: string
  attrs?: Record<string, unknown>
}

export interface ProseMirrorNode {
  type: string
  attrs?: Record<string, unknown>
  content?: ProseMirrorNode[]
  marks?: ProseMirrorMark[]
  text?: string
}

// =============================================================================
// Node Attribute Types (for type safety)
// =============================================================================

/** Heading node attributes */
export interface HeadingAttrs {
  level: 1 | 2 | 3
}

/** Ordered list node attributes */
export interface OrderedListAttrs {
  start?: number
}

/** Task item node attributes */
export interface TaskItemAttrs {
  checked: boolean
}

/** Code block node attributes */
export interface CodeBlockAttrs {
  language?: string
}

/** Page break node attributes */
export interface PageBreakAttrs {
  id: string
}

/** Image display mode */
export type ImageDisplayMode = 'block' | 'inline'

/** Image alignment */
export type ImageAlign = 'left' | 'center' | 'right'

/** Image shape */
export type ImageShape = 'square' | 'circle'

/** Image node attributes */
export interface ImageAttrs {
  /** Base64 data URI or URL */
  src: string
  alt?: string
  title?: string
  width?: number
  height?: number
  displayMode: ImageDisplayMode
  align: ImageAlign
  shape: ImageShape
}

/** Signature node attributes */
export interface SignatureAttrs {
  count: SignatureCount
  layout: SignatureLayout
  lineWidth: SignatureLineWidth
  signatures: SignatureItem[]
}

/** Conditional node attributes */
export interface ConditionalAttrs {
  conditions: LogicGroup
  expression: string
}

/**
 * Injector (variable placeholder) node attributes
 * Only variableId is stored; type, label, etc. come from backend
 */
export interface InjectorAttrs {
  /** Reference to backend variable */
  variableId: string
}

/** Link mark attributes */
export interface LinkMarkAttrs {
  href: string
  target?: string
}

/** Highlight mark attributes */
export interface HighlightMarkAttrs {
  color?: string
}

// =============================================================================
// Validation Result Types
// =============================================================================

export interface ValidationError {
  /** Error code for programmatic handling */
  code: string

  /** JSON path to the error location */
  path: string

  /** Human-readable error message */
  message: string
}

export interface ValidationWarning {
  /** Warning code for programmatic handling */
  code: string

  /** JSON path to the warning location */
  path: string

  /** Human-readable warning message */
  message: string

  /** Suggested fix */
  suggestion?: string
}

export interface ValidationResult {
  /** Whether the document is valid */
  valid: boolean

  /** Critical errors that prevent import */
  errors: ValidationError[]

  /** Non-critical warnings */
  warnings: ValidationWarning[]
}

// =============================================================================
// Import Result Types
// =============================================================================

export interface ImportResult {
  /** Whether import was successful */
  success: boolean

  /** Validation result */
  validation: ValidationResult

  /** Imported document (if successful) */
  document?: PortableDocument

  /** Variable IDs in document that don't exist in the backend */
  orphanedVariables?: string[]
}

// =============================================================================
// Export Options
// =============================================================================

export interface ExportOptions {
  /** Include checksum for integrity verification */
  includeChecksum?: boolean

  /** Pretty print JSON output */
  prettyPrint?: boolean

  /** User identifier for audit trail */
  exportedBy?: string
}

// =============================================================================
// Import Options
// =============================================================================

export interface ImportOptions {
  /** Whether to validate semantic references */
  validateReferences?: boolean

  /** Whether to auto-migrate older versions */
  autoMigrate?: boolean

  /** Maximum allowed image size in bytes (default: 5MB) */
  maxImageSize?: number
}

// =============================================================================
// Re-export related types for convenience
// =============================================================================

// Re-export signer role types
export type { SignerRoleDefinition, SignerRoleFieldValue, SignerRoleFieldType } from './signer-roles'
export type {
  SigningOrderMode,
  NotificationTrigger,
  NotificationScope,
  NotificationTriggerSettings,
  NotificationTriggerMap,
  PreviousRolesConfig,
  RoleNotificationConfig,
  SigningNotificationConfig,
  SigningWorkflowConfig,
} from './signer-roles'

// Re-export conditional logic types
export type { LogicGroup, LogicRule, RuleValue, RuleOperator } from '../extensions/Conditional/ConditionalExtension'

// Re-export signature types
export type { SignatureCount, SignatureLayout, SignatureLineWidth, SignatureItem } from '../extensions/Signature/types'
