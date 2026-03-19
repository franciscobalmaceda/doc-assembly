import { useState, useEffect, useRef } from 'react'
import { useTranslation } from 'react-i18next'
import { useEditor, EditorContent } from '@tiptap/react'
import StarterKit from '@tiptap/starter-kit'
import { TextStyle, FontFamily, FontSize } from '@tiptap/extension-text-style'
import { Color } from '@tiptap/extension-color'
import TextAlign from '@tiptap/extension-text-align'
import { ImageIcon, X, PanelLeft, PanelRight, LayoutTemplate } from 'lucide-react'
import type { Editor } from '@tiptap/core'
import { cn } from '@/lib/utils'
import { ImageInsertModal, type ImageInsertResult } from './ImageInsertModal'
import { useDocumentHeaderStore, type DocumentHeaderLayout } from '../stores/document-header-store'

interface DocumentPageHeaderProps {
  editable: boolean
  onTextEditorFocus?: (editor: Editor) => void
  onTextEditorBlur?: () => void
}

// =============================================================================
// Image slot
// =============================================================================

interface ImageSlotProps {
  imageUrl: string | null
  imageAlt: string
  editable: boolean
  onOpenModal: () => void
  onRemove: () => void
  className?: string
}

function ImageSlot({ imageUrl, imageAlt, editable, onOpenModal, onRemove, className }: ImageSlotProps) {
  const { t } = useTranslation()

  return (
    <div
      className={cn(
        'relative flex items-center justify-center',
        'min-h-[80px] max-h-[120px]',
        className
      )}
    >
      {imageUrl ? (
        <>
          <img
            src={imageUrl}
            alt={imageAlt}
            className="object-contain max-h-[120px] w-full p-2"
          />
          {editable && (
            <>
              <button
                type="button"
                onClick={onOpenModal}
                className="absolute inset-0 opacity-0 hover:opacity-100 flex items-center justify-center bg-background/60 transition-opacity"
                title={t('editor.documentHeader.addLogo')}
              >
                <ImageIcon className="h-5 w-5" />
              </button>
              <button
                type="button"
                onClick={onRemove}
                className="absolute top-1 right-1 z-10 rounded-full bg-background/80 p-0.5 text-muted-foreground hover:text-foreground hover:bg-background transition-colors"
                title={t('common.remove')}
              >
                <X className="h-3 w-3" />
              </button>
            </>
          )}
        </>
      ) : editable ? (
        <button
          type="button"
          onClick={onOpenModal}
          className="flex flex-col items-center gap-1 text-muted-foreground hover:text-foreground transition-colors p-4 rounded border-2 border-dashed border-border hover:border-muted-foreground"
        >
          <ImageIcon className="h-5 w-5" />
          <span className="text-xs">{t('editor.documentHeader.addLogo')}</span>
        </button>
      ) : null}
    </div>
  )
}

// =============================================================================
// Layout picker
// =============================================================================

const LAYOUTS: { value: DocumentHeaderLayout; icon: typeof PanelLeft; labelKey: string }[] = [
  { value: 'image-left',   icon: PanelLeft,      labelKey: 'editor.documentHeader.layoutImageLeft' },
  { value: 'image-center', icon: LayoutTemplate, labelKey: 'editor.documentHeader.layoutImageCenter' },
  { value: 'image-right',  icon: PanelRight,     labelKey: 'editor.documentHeader.layoutImageRight' },
]

function LayoutPicker({ current, onChange }: { current: DocumentHeaderLayout; onChange: (l: DocumentHeaderLayout) => void }) {
  const { t } = useTranslation()
  return (
    <div className="flex items-center gap-0.5 rounded border border-border bg-background/80 p-0.5">
      {LAYOUTS.map(({ value, icon: Icon, labelKey }) => (
        <button
          key={value}
          type="button"
          onMouseDown={(e) => e.preventDefault()}
          onClick={() => onChange(value)}
          title={t(labelKey)}
          className={cn(
            'rounded p-1 transition-colors',
            current === value
              ? 'bg-primary text-primary-foreground'
              : 'text-muted-foreground hover:text-foreground hover:bg-muted'
          )}
        >
          <Icon className="h-3 w-3" />
        </button>
      ))}
    </div>
  )
}

// =============================================================================
// Main component
// =============================================================================

export function DocumentPageHeader({ editable, onTextEditorFocus, onTextEditorBlur }: DocumentPageHeaderProps) {
  const { t } = useTranslation()
  const { layout, imageUrl, imageAlt, text, setEnabled, setLayout, setImage, setText } =
    useDocumentHeaderStore()

  const [imageModalOpen, setImageModalOpen] = useState(false)

  // Track the last text we set from the store so we don't re-apply on our own updates
  const lastExternalText = useRef(text)
  const isExternalUpdate = useRef(false)

  const headerEditor = useEditor({
    immediatelyRender: false,
    extensions: [
      StarterKit.configure({ heading: { levels: [1, 2, 3] } }),
      TextStyle,
      Color,
      FontFamily.configure({ types: ['textStyle'] }),
      FontSize.configure({ types: ['textStyle'] }),
      TextAlign.configure({ types: ['heading', 'paragraph'] }),
    ],
    content: text || '<p></p>',
    editable,
    onUpdate: ({ editor }) => {
      if (isExternalUpdate.current) return
      const html = editor.getHTML()
      lastExternalText.current = html
      setText(html)
    },
    onFocus: ({ editor }) => {
      onTextEditorFocus?.(editor)
    },
    onBlur: () => {
      onTextEditorBlur?.()
    },
    editorProps: {
      attributes: {
        class: 'prose prose-sm dark:prose-invert max-w-none focus:outline-none min-h-[60px] p-3',
      },
    },
  })

  // Sync store text → editor when changed externally (e.g. document import)
  useEffect(() => {
    if (!headerEditor || text === lastExternalText.current) return
    lastExternalText.current = text
    isExternalUpdate.current = true
    headerEditor.commands.setContent(text || '<p></p>')
    isExternalUpdate.current = false
  }, [text, headerEditor])

  // Sync editable flag
  useEffect(() => {
    if (!headerEditor) return
    headerEditor.setEditable(editable)
  }, [headerEditor, editable])

  const handleImageInsert = (result: ImageInsertResult) => {
    setImage(result.src, result.injectableLabel ?? '')
    setImageModalOpen(false)
  }

  const imageSlot = (
    <ImageSlot
      imageUrl={imageUrl}
      imageAlt={imageAlt}
      editable={editable}
      onOpenModal={() => setImageModalOpen(true)}
      onRemove={() => setImage('', '')}
      className={layout === 'image-center' ? 'w-full' : 'w-[30%]'}
    />
  )

  const textSlot = headerEditor ? (
    <EditorContent
      editor={headerEditor}
      className="flex-1 overflow-hidden"
    />
  ) : null

  return (
    <>
      <div className="relative border-b border-border">
        {/* Controls row */}
        {editable && (
          <div className="absolute top-1 right-1 z-10 flex items-center gap-1">
            <LayoutPicker current={layout} onChange={setLayout} />
            <button
              type="button"
              onMouseDown={(e) => e.preventDefault()}
              onClick={() => setEnabled(false)}
              className="rounded-full bg-background/80 p-0.5 text-muted-foreground hover:text-foreground hover:bg-background transition-colors"
              title={t('editor.toolbar.removeHeader')}
            >
              <X className="h-3 w-3" />
            </button>
          </div>
        )}

        {layout === 'image-left' && (
          <div className="flex">
            <ImageSlot
              imageUrl={imageUrl}
              imageAlt={imageAlt}
              editable={editable}
              onOpenModal={() => setImageModalOpen(true)}
              onRemove={() => setImage('', '')}
              className="w-[30%] border-r border-border"
            />
            {textSlot}
          </div>
        )}

        {layout === 'image-right' && (
          <div className="flex">
            {textSlot}
            <ImageSlot
              imageUrl={imageUrl}
              imageAlt={imageAlt}
              editable={editable}
              onOpenModal={() => setImageModalOpen(true)}
              onRemove={() => setImage('', '')}
              className="w-[30%] border-l border-border"
            />
          </div>
        )}

        {layout === 'image-center' && (
          <div className="flex justify-center py-2 px-4">
            {imageSlot}
          </div>
        )}
      </div>

      <ImageInsertModal
        open={imageModalOpen}
        onOpenChange={(open) => { if (!open) setImageModalOpen(false) }}
        onInsert={handleImageInsert}
        initialShape="square"
      />
    </>
  )
}
