import { useState, useEffect, useRef } from 'react'
import { useTranslation } from 'react-i18next'
import { galleryApi } from '../api/gallery-api'
import { useEditor, EditorContent } from '@tiptap/react'
import StarterKit from '@tiptap/starter-kit'
import { TextStyle, FontFamily, FontSize } from '@tiptap/extension-text-style'
import { Color } from '@tiptap/extension-color'
import TextAlign from '@tiptap/extension-text-align'
import { ImageIcon, X, PanelLeft, PanelRight, LayoutTemplate, Trash2 } from 'lucide-react'
import type { Editor } from '@tiptap/core'
import { cn } from '@/lib/utils'
import { Button } from '@/components/ui/button'
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import { ImageInsertModal, type ImageInsertResult } from './ImageInsertModal'
import { useDocumentHeaderStore, type DocumentHeaderLayout } from '../stores/document-header-store'
import { StoredMarksPersistenceExtension } from '../extensions/StoredMarksPersistence'

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
// Layout constants (reused in the toolbar)
// =============================================================================

const LAYOUTS: { value: DocumentHeaderLayout; icon: typeof PanelLeft; labelKey: string }[] = [
  { value: 'image-left',   icon: PanelLeft,      labelKey: 'editor.documentHeader.layoutImageLeft' },
  { value: 'image-center', icon: LayoutTemplate, labelKey: 'editor.documentHeader.layoutImageCenter' },
  { value: 'image-right',  icon: PanelRight,     labelKey: 'editor.documentHeader.layoutImageRight' },
]

// =============================================================================
// Main component
// =============================================================================

export function DocumentPageHeader({ editable, onTextEditorFocus, onTextEditorBlur }: DocumentPageHeaderProps) {
  const { t } = useTranslation()
  const { layout, imageUrl, imageAlt, content: storeContent, setEnabled, setLayout, setImage, setContent } =
    useDocumentHeaderStore()

  const [imageModalOpen, setImageModalOpen] = useState(false)

  // Show the floating toolbar when hovering the header OR when the text editor is focused.
  const [isHovered, setIsHovered] = useState(false)
  const [isFocused, setIsFocused] = useState(false)
  const showToolbar = editable && (isHovered || isFocused)
  // Small delay on mouse-leave prevents the toolbar from vanishing while moving
  // from the header content up to the toolbar buttons.
  const hoverTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const handleMouseEnter = () => {
    if (hoverTimerRef.current) clearTimeout(hoverTimerRef.current)
    setIsHovered(true)
  }
  const handleMouseLeave = () => {
    hoverTimerRef.current = setTimeout(() => setIsHovered(false), 150)
  }

  // Resolve storage:// gallery URLs to displayable URLs (blob or presigned).
  const [resolvedImageUrl, setResolvedImageUrl] = useState<string>(
    imageUrl?.startsWith('storage://') ? '' : (imageUrl ?? '')
  )
  useEffect(() => {
    if (!imageUrl?.startsWith('storage://')) {
      setResolvedImageUrl(imageUrl ?? '')
      return
    }
    let blobURL = ''
    const key = imageUrl.slice('storage://'.length)
    galleryApi.getSrc(key).then((url) => {
      if (url.startsWith('blob:')) blobURL = url
      setResolvedImageUrl(url)
    }).catch(() => {
      setResolvedImageUrl('')
    })
    return () => {
      if (blobURL) URL.revokeObjectURL(blobURL)
    }
  }, [imageUrl])

  // Track the last content we set from the store so we don't re-apply on our own updates
  const lastExternalContent = useRef<string>(JSON.stringify(storeContent))
  const isExternalUpdate = useRef(false)

  const emptyDoc = { type: 'doc', content: [{ type: 'paragraph' }] }

  const headerEditor = useEditor({
    immediatelyRender: false,
    extensions: [
      StarterKit.configure({ heading: { levels: [1, 2, 3] } }),
      TextStyle,
      Color,
      FontFamily.configure({ types: ['textStyle'] }),
      FontSize.configure({ types: ['textStyle'] }),
      TextAlign.configure({ types: ['heading', 'paragraph'] }),
      StoredMarksPersistenceExtension,
    ],
    content: storeContent ?? emptyDoc,
    editable,
    onUpdate: ({ editor }) => {
      if (isExternalUpdate.current) return
      const json = editor.getJSON()
      lastExternalContent.current = JSON.stringify(json)
      setContent(json)
    },
    onFocus: ({ editor }) => {
      setIsFocused(true)
      onTextEditorFocus?.(editor)
    },
    onBlur: () => {
      setIsFocused(false)
      onTextEditorBlur?.()
    },
    editorProps: {
      attributes: {
        class: 'prose prose-sm dark:prose-invert max-w-none focus:outline-none min-h-[60px] p-3',
      },
    },
  })

  // Sync store content → editor when changed externally (e.g. document import)
  useEffect(() => {
    if (!headerEditor) return
    const serialized = JSON.stringify(storeContent)
    if (serialized === lastExternalContent.current) return
    lastExternalContent.current = serialized
    isExternalUpdate.current = true
    headerEditor.commands.setContent(storeContent ?? emptyDoc)
    isExternalUpdate.current = false
  }, [storeContent, headerEditor])

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
      imageUrl={resolvedImageUrl}
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
      className="flex-1 min-w-0"
    />
  ) : null

  return (
    <>
      <div
        className="relative"
        onMouseEnter={handleMouseEnter}
        onMouseLeave={handleMouseLeave}
      >
        {/* Floating block toolbar — visible on hover or when header text editor is focused */}
        {showToolbar && (
          <div className="absolute top-1 left-1/2 -translate-x-1/2 flex items-center gap-1 bg-background border rounded-lg shadow-lg p-1 z-50">
            <TooltipProvider delayDuration={300}>
              {LAYOUTS.map(({ value, icon: Icon, labelKey }) => (
                <Tooltip key={value}>
                  <TooltipTrigger asChild>
                    <Button
                      variant="ghost"
                      size="icon"
                      className={cn('h-8 w-8', layout === value && 'bg-accent')}
                      onMouseDown={(e) => e.preventDefault()}
                      onClick={() => setLayout(value)}
                    >
                      <Icon className="h-4 w-4" />
                    </Button>
                  </TooltipTrigger>
                  <TooltipContent side="top"><p>{t(labelKey)}</p></TooltipContent>
                </Tooltip>
              ))}

              <div className="w-px h-6 bg-border mx-1" />

              <Tooltip>
                <TooltipTrigger asChild>
                  <Button
                    variant="ghost"
                    size="icon"
                    className="h-8 w-8 text-destructive hover:text-destructive"
                    onMouseDown={(e) => e.preventDefault()}
                    onClick={() => setEnabled(false)}
                  >
                    <Trash2 className="h-4 w-4" />
                  </Button>
                </TooltipTrigger>
                <TooltipContent side="top"><p>{t('editor.toolbar.removeHeader')}</p></TooltipContent>
              </Tooltip>
            </TooltipProvider>
          </div>
        )}

        {layout === 'image-left' && (
          <div className="flex">
            <ImageSlot
              imageUrl={resolvedImageUrl}
              imageAlt={imageAlt}
              editable={editable}
              onOpenModal={() => setImageModalOpen(true)}
              onRemove={() => setImage('', '')}
              className="w-[30%]"
            />
            {textSlot}
          </div>
        )}

        {layout === 'image-right' && (
          <div className="flex">
            {textSlot}
            <ImageSlot
              imageUrl={resolvedImageUrl}
              imageAlt={imageAlt}
              editable={editable}
              onOpenModal={() => setImageModalOpen(true)}
              onRemove={() => setImage('', '')}
              className="w-[30%]"
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
