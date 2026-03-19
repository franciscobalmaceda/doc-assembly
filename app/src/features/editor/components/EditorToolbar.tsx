import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import type { Editor } from '@tiptap/react'
import { Button } from '@/components/ui/button'
import { Separator } from '@/components/ui/separator'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import {
  Bold,
  Italic,
  Strikethrough,
  AlignLeft,
  AlignCenter,
  AlignRight,
  AlignJustify,
  List,
  ListOrdered,
  Quote,
  Undo,
  Redo,
  Heading1,
  Heading2,
  Heading3,
  Minus,
  PenLine,
  GitBranch,
  ImageIcon,
  Table2,
  Download,
  Upload,
  ChevronLeft,
  ChevronRight,
  MoreHorizontal,
  Paintbrush,
  PanelTop,
} from 'lucide-react'
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import { PreviewButton } from './preview'
import { TextColorPicker } from './TextColorPicker'
import { FontFamilyPicker } from './FontFamilyPicker'
import { FontSizePicker } from './FontSizePicker'
import { useOverflowScroll } from '@/hooks/use-overflow-scroll'
import { cn } from '@/lib/utils'
import { useDocumentHeaderStore } from '../stores/document-header-store'

interface EditorToolbarProps {
  editor: Editor | null
  onExport?: () => void
  onImport?: () => void
  templateId?: string
  versionId?: string
  /** Hide document-specific block buttons (image, signature, table, etc.) when editing header text */
  showSpecialBlocks?: boolean
}

export function EditorToolbar({ editor, onExport, onImport, templateId, versionId, showSpecialBlocks = true }: EditorToolbarProps) {
  const { t } = useTranslation()
  const { enabled: headerEnabled, setEnabled: setHeaderEnabled } = useDocumentHeaderStore()
  // Force re-render when editor state changes (for undo/redo buttons)
  const [, forceUpdate] = useState({})

  const {
    containerRef,
    scrollRef,
    canScrollLeft,
    canScrollRight,
    isOverflowing,
    scrollLeft,
    scrollRight,
  } = useOverflowScroll()

  useEffect(() => {
    if (!editor) return

    const handler = () => forceUpdate({})
    editor.on('transaction', handler)

    return () => {
      editor.off('transaction', handler)
    }
  }, [editor])

  if (!editor) return null

  return (
    <TooltipProvider delayDuration={300}>
      <div
        ref={containerRef}
        className="relative flex-1 min-w-0 flex items-center pt-3 pb-2 px-2 bg-card overflow-y-visible overflow-x-clip"
      >
        {/* Left scroll arrow */}
        {canScrollLeft && (
          <button
            type="button"
            onClick={scrollLeft}
            className={cn(
              'absolute left-0 z-10 h-full px-1',
              'flex items-center justify-center',
              'bg-gradient-to-r from-card via-card/95 to-transparent',
              'text-muted-foreground hover:text-foreground',
              'transition-colors duration-150'
            )}
            aria-label={t('editor.toolbar.scrollLeft')}
          >
            <ChevronLeft className="h-4 w-4" />
          </button>
        )}

        {/* Scrollable toolbar content */}
        <div
          ref={scrollRef}
          className={cn(
            'flex items-center gap-1 overflow-x-auto overflow-y-visible scrollbar-hide scroll-smooth touch-pan-x pt-1',
            canScrollLeft && 'pl-6',
            canScrollRight && 'pr-6'
          )}
        >
          {/* Undo/Redo */}
          <ToolbarButton
            onClick={() => editor.chain().focus().undo().run()}
            disabled={!editor.can().undo()}
            tooltip={t('editor.toolbar.undo')}
          >
            <Undo className="h-4 w-4" />
          </ToolbarButton>
          <ToolbarButton
            onClick={() => editor.chain().focus().redo().run()}
            disabled={!editor.can().redo()}
            tooltip={t('editor.toolbar.redo')}
          >
            <Redo className="h-4 w-4" />
          </ToolbarButton>

          <Separator orientation="vertical" className="h-6 mx-1" />

          {/* Font Family */}
          <FontFamilyPicker editor={editor} />

          {/* Font Size */}
          <FontSizePicker editor={editor} />

          <Separator orientation="vertical" className="h-6 mx-1" />

          {/* Headings */}
          <ToolbarButton
            onClick={() => editor.chain().focus().toggleHeading({ level: 1 }).run()}
            isActive={editor.isActive('heading', { level: 1 })}
            disabled={editor.isActive('table')}
            tooltip={t('editor.toolbar.heading1')}
          >
            <Heading1 className="h-4 w-4" />
          </ToolbarButton>
          <ToolbarButton
            onClick={() => editor.chain().focus().toggleHeading({ level: 2 }).run()}
            isActive={editor.isActive('heading', { level: 2 })}
            disabled={editor.isActive('table')}
            tooltip={t('editor.toolbar.heading2')}
          >
            <Heading2 className="h-4 w-4" />
          </ToolbarButton>
          <ToolbarButton
            onClick={() => editor.chain().focus().toggleHeading({ level: 3 }).run()}
            isActive={editor.isActive('heading', { level: 3 })}
            disabled={editor.isActive('table')}
            tooltip={t('editor.toolbar.heading3')}
          >
            <Heading3 className="h-4 w-4" />
          </ToolbarButton>

          <Separator orientation="vertical" className="h-6 mx-1" />

          {/* Text formatting */}
          <ToolbarButton
            onClick={() => editor.chain().focus().toggleBold().run()}
            isActive={editor.isActive('bold')}
            tooltip={t('editor.toolbar.bold')}
          >
            <Bold className="h-4 w-4" />
          </ToolbarButton>
          <ToolbarButton
            onClick={() => editor.chain().focus().toggleItalic().run()}
            isActive={editor.isActive('italic')}
            tooltip={t('editor.toolbar.italic')}
          >
            <Italic className="h-4 w-4" />
          </ToolbarButton>
          <ToolbarButton
            onClick={() => editor.chain().focus().toggleStrike().run()}
            isActive={editor.isActive('strike')}
            tooltip={t('editor.toolbar.strikethrough')}
          >
            <Strikethrough className="h-4 w-4" />
          </ToolbarButton>

          {/* Text Color */}
          <TextColorPicker editor={editor} />

          <Separator orientation="vertical" className="h-6 mx-1" />

          {/* Text alignment */}
          <ToolbarButton
            onClick={() => editor.chain().focus().setTextAlign('left').run()}
            isActive={editor.isActive({ textAlign: 'left' })}
            tooltip={t('editor.toolbar.alignLeft')}
          >
            <AlignLeft className="h-4 w-4" />
          </ToolbarButton>
          <ToolbarButton
            onClick={() => editor.chain().focus().setTextAlign('center').run()}
            isActive={editor.isActive({ textAlign: 'center' })}
            tooltip={t('editor.toolbar.alignCenter')}
          >
            <AlignCenter className="h-4 w-4" />
          </ToolbarButton>
          <ToolbarButton
            onClick={() => editor.chain().focus().setTextAlign('right').run()}
            isActive={editor.isActive({ textAlign: 'right' })}
            tooltip={t('editor.toolbar.alignRight')}
          >
            <AlignRight className="h-4 w-4" />
          </ToolbarButton>
          <ToolbarButton
            onClick={() => editor.chain().focus().setTextAlign('justify').run()}
            isActive={editor.isActive({ textAlign: 'justify' })}
            tooltip={t('editor.toolbar.alignJustify')}
          >
            <AlignJustify className="h-4 w-4" />
          </ToolbarButton>

          <Separator orientation="vertical" className="h-6 mx-1" />

          {/* Lists */}
          <ToolbarButton
            onClick={() => editor.chain().focus().toggleBulletList().run()}
            isActive={editor.isActive('bulletList')}
            tooltip={t('editor.toolbar.bulletList')}
          >
            <List className="h-4 w-4" />
          </ToolbarButton>
          <ToolbarButton
            onClick={() => editor.chain().focus().toggleOrderedList().run()}
            isActive={editor.isActive('orderedList')}
            tooltip={t('editor.toolbar.orderedList')}
          >
            <ListOrdered className="h-4 w-4" />
          </ToolbarButton>

          <Separator orientation="vertical" className="h-6 mx-1" />

          {/* Block elements */}
          <ToolbarButton
            onClick={() => editor.chain().focus().toggleBlockquote().run()}
            isActive={editor.isActive('blockquote')}
            tooltip={t('editor.toolbar.blockquote')}
          >
            <Quote className="h-4 w-4" />
          </ToolbarButton>
          <ToolbarButton
            onClick={() => editor.chain().focus().setHorizontalRule().run()}
            tooltip={t('editor.toolbar.horizontalRule')}
          >
            <Minus className="h-4 w-4" />
          </ToolbarButton>

          <Separator orientation="vertical" className="h-6 mx-1" />

          {/* Document elements - Special tools section */}
          {showSpecialBlocks && (
            <>
              <div className="relative flex items-center gap-1 px-2 py-1 bg-muted/60 dark:bg-muted/40 rounded-lg border border-border">
                {/* Floating label */}
                <span className="absolute -top-2 left-2 px-1.5 text-[10px] font-medium tracking-wide uppercase text-muted-foreground bg-card rounded">
                  {t('editor.toolbar.blocks')}
                </span>

                <ToolbarButton
                  onClick={() => {
                    editor.view.dom.dispatchEvent(
                      new CustomEvent('editor:open-image-modal', { bubbles: true })
                    )
                  }}
                  tooltip={t('editor.toolbar.insertImage')}
                >
                  <ImageIcon className="h-4 w-4 text-success-foreground dark:text-success" />
                </ToolbarButton>
                <ToolbarButton
                  onClick={() => editor.chain().focus().setSignature().run()}
                  tooltip={t('editor.toolbar.signatureBlock')}
                >
                  <PenLine className="h-4 w-4 text-info-foreground dark:text-info" />
                </ToolbarButton>
                <ToolbarButton
                  onClick={() => editor.chain().focus().setConditional({}).run()}
                  tooltip={t('editor.toolbar.conditionalBlock')}
                >
                  <GitBranch className="h-4 w-4 text-warning-foreground dark:text-warning" />
                </ToolbarButton>
                <ToolbarButton
                  onClick={() => setHeaderEnabled(!headerEnabled)}
                  isActive={headerEnabled}
                  tooltip={headerEnabled ? t('editor.toolbar.removeHeader') : t('editor.toolbar.addHeader')}
                >
                  <PanelTop className="h-4 w-4" />
                </ToolbarButton>
                <ToolbarButton
                  onClick={() => editor.chain().focus().insertTable({ rows: 3, cols: 3, withHeaderRow: true }).run()}
                  tooltip={t('editor.insertTable')}
                >
                  <Table2 className="h-4 w-4 text-primary" />
                </ToolbarButton>
              </div>

              <Separator orientation="vertical" className="h-6 mx-1" />
            </>
          )}

          {/* Export/Import/Preview */}
          {(onExport || onImport || (templateId && versionId)) && (
            <>
              {onExport && (
                <ToolbarButton
                  onClick={onExport}
                  tooltip={t('editor.toolbar.exportDocument')}
                >
                  <Download className="h-4 w-4" />
                </ToolbarButton>
              )}
              {onImport && (
                <ToolbarButton
                  onClick={onImport}
                  tooltip={t('editor.toolbar.importDocument')}
                >
                  <Upload className="h-4 w-4" />
                </ToolbarButton>
              )}
              {templateId && versionId && (
                <PreviewButton
                  templateId={templateId}
                  versionId={versionId}
                  editor={editor}
                />
              )}
            </>
          )}
        </div>

        {/* Right scroll arrow */}
        {canScrollRight && (
          <button
            type="button"
            onClick={scrollRight}
            className={cn(
              'absolute right-8 z-10 h-full px-1',
              'flex items-center justify-center',
              'bg-gradient-to-l from-card via-card/95 to-transparent',
              'text-muted-foreground hover:text-foreground',
              'transition-colors duration-150'
            )}
            aria-label={t('editor.toolbar.scrollRight')}
          >
            <ChevronRight className="h-4 w-4" />
          </button>
        )}

        {/* Overflow dropdown menu */}
        {isOverflowing && (
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button
                variant="ghost"
                size="sm"
                className="h-8 w-8 p-0 shrink-0 ml-1 text-muted-foreground hover:text-foreground"
              >
                <MoreHorizontal className="h-4 w-4" />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end" className="w-56">
              {/* History */}
              <DropdownMenuLabel className="text-xs font-medium text-muted-foreground">
                {t('editor.toolbar.history')}
              </DropdownMenuLabel>
              <DropdownMenuItem
                onClick={() => editor.chain().focus().undo().run()}
                disabled={!editor.can().undo()}
              >
                <Undo className="mr-2 h-4 w-4" />
                {t('editor.toolbar.undo')}
              </DropdownMenuItem>
              <DropdownMenuItem
                onClick={() => editor.chain().focus().redo().run()}
                disabled={!editor.can().redo()}
              >
                <Redo className="mr-2 h-4 w-4" />
                {t('editor.toolbar.redo')}
              </DropdownMenuItem>

              <DropdownMenuSeparator />

              {/* Headings */}
              <DropdownMenuLabel className="text-xs font-medium text-muted-foreground">
                {t('editor.toolbar.headings')}
              </DropdownMenuLabel>
              <DropdownMenuItem
                onClick={() => editor.chain().focus().toggleHeading({ level: 1 }).run()}
                className={cn(editor.isActive('heading', { level: 1 }) && 'bg-accent')}
                disabled={editor.isActive('table')}
              >
                <Heading1 className="mr-2 h-4 w-4" />
                {t('editor.toolbar.heading1')}
              </DropdownMenuItem>
              <DropdownMenuItem
                onClick={() => editor.chain().focus().toggleHeading({ level: 2 }).run()}
                className={cn(editor.isActive('heading', { level: 2 }) && 'bg-accent')}
                disabled={editor.isActive('table')}
              >
                <Heading2 className="mr-2 h-4 w-4" />
                {t('editor.toolbar.heading2')}
              </DropdownMenuItem>
              <DropdownMenuItem
                onClick={() => editor.chain().focus().toggleHeading({ level: 3 }).run()}
                className={cn(editor.isActive('heading', { level: 3 }) && 'bg-accent')}
                disabled={editor.isActive('table')}
              >
                <Heading3 className="mr-2 h-4 w-4" />
                {t('editor.toolbar.heading3')}
              </DropdownMenuItem>

              <DropdownMenuSeparator />

              {/* Text formatting */}
              <DropdownMenuLabel className="text-xs font-medium text-muted-foreground">
                {t('editor.toolbar.textFormatting')}
              </DropdownMenuLabel>
              <DropdownMenuItem
                onClick={() => editor.chain().focus().toggleBold().run()}
                className={cn(editor.isActive('bold') && 'bg-accent')}
              >
                <Bold className="mr-2 h-4 w-4" />
                {t('editor.toolbar.bold')}
              </DropdownMenuItem>
              <DropdownMenuItem
                onClick={() => editor.chain().focus().toggleItalic().run()}
                className={cn(editor.isActive('italic') && 'bg-accent')}
              >
                <Italic className="mr-2 h-4 w-4" />
                {t('editor.toolbar.italic')}
              </DropdownMenuItem>
              <DropdownMenuItem
                onClick={() => editor.chain().focus().toggleStrike().run()}
                className={cn(editor.isActive('strike') && 'bg-accent')}
              >
                <Strikethrough className="mr-2 h-4 w-4" />
                {t('editor.toolbar.strikethrough')}
              </DropdownMenuItem>
              <DropdownMenuItem asChild>
                <label className="flex items-center gap-2 cursor-pointer">
                  <Paintbrush className="h-4 w-4" style={{ color: editor.getAttributes('textStyle').color || undefined }} />
                  {t('editor.toolbar.textColor')}
                  <input
                    type="color"
                    value={editor.getAttributes('textStyle').color || '#000000'}
                    onChange={(e) => editor.chain().focus().setColor(e.target.value).run()}
                    className="w-6 h-6 p-0 border border-border rounded-sm cursor-pointer ml-auto"
                  />
                </label>
              </DropdownMenuItem>

              <DropdownMenuSeparator />

              {/* Text alignment */}
              <DropdownMenuLabel className="text-xs font-medium text-muted-foreground">
                {t('editor.toolbar.alignment')}
              </DropdownMenuLabel>
              <DropdownMenuItem
                onClick={() => editor.chain().focus().setTextAlign('left').run()}
                className={cn(editor.isActive({ textAlign: 'left' }) && 'bg-accent')}
              >
                <AlignLeft className="mr-2 h-4 w-4" />
                {t('editor.toolbar.alignLeft')}
              </DropdownMenuItem>
              <DropdownMenuItem
                onClick={() => editor.chain().focus().setTextAlign('center').run()}
                className={cn(editor.isActive({ textAlign: 'center' }) && 'bg-accent')}
              >
                <AlignCenter className="mr-2 h-4 w-4" />
                {t('editor.toolbar.alignCenter')}
              </DropdownMenuItem>
              <DropdownMenuItem
                onClick={() => editor.chain().focus().setTextAlign('right').run()}
                className={cn(editor.isActive({ textAlign: 'right' }) && 'bg-accent')}
              >
                <AlignRight className="mr-2 h-4 w-4" />
                {t('editor.toolbar.alignRight')}
              </DropdownMenuItem>
              <DropdownMenuItem
                onClick={() => editor.chain().focus().setTextAlign('justify').run()}
                className={cn(editor.isActive({ textAlign: 'justify' }) && 'bg-accent')}
              >
                <AlignJustify className="mr-2 h-4 w-4" />
                {t('editor.toolbar.alignJustify')}
              </DropdownMenuItem>

              <DropdownMenuSeparator />

              {/* Lists */}
              <DropdownMenuLabel className="text-xs font-medium text-muted-foreground">
                {t('editor.toolbar.lists')}
              </DropdownMenuLabel>
              <DropdownMenuItem
                onClick={() => editor.chain().focus().toggleBulletList().run()}
                className={cn(editor.isActive('bulletList') && 'bg-accent')}
              >
                <List className="mr-2 h-4 w-4" />
                {t('editor.toolbar.bulletList')}
              </DropdownMenuItem>
              <DropdownMenuItem
                onClick={() => editor.chain().focus().toggleOrderedList().run()}
                className={cn(editor.isActive('orderedList') && 'bg-accent')}
              >
                <ListOrdered className="mr-2 h-4 w-4" />
                {t('editor.toolbar.orderedList')}
              </DropdownMenuItem>

              <DropdownMenuSeparator />

              {/* Block elements */}
              <DropdownMenuLabel className="text-xs font-medium text-muted-foreground">
                {t('editor.toolbar.blockElements')}
              </DropdownMenuLabel>
              <DropdownMenuItem
                onClick={() => editor.chain().focus().toggleBlockquote().run()}
                className={cn(editor.isActive('blockquote') && 'bg-accent')}
              >
                <Quote className="mr-2 h-4 w-4" />
                {t('editor.toolbar.blockquote')}
              </DropdownMenuItem>
              <DropdownMenuItem
                onClick={() => editor.chain().focus().setHorizontalRule().run()}
              >
                <Minus className="mr-2 h-4 w-4" />
                {t('editor.toolbar.horizontalRule')}
              </DropdownMenuItem>
              {showSpecialBlocks && (
                <>
                  <DropdownMenuItem
                    onClick={() => {
                      editor.view.dom.dispatchEvent(
                        new CustomEvent('editor:open-image-modal', { bubbles: true })
                      )
                    }}
                  >
                    <ImageIcon className="mr-2 h-4 w-4 text-success-foreground dark:text-success" />
                    {t('editor.toolbar.insertImage')}
                  </DropdownMenuItem>
                  <DropdownMenuItem
                    onClick={() => editor.chain().focus().setSignature().run()}
                  >
                    <PenLine className="mr-2 h-4 w-4 text-info-foreground dark:text-info" />
                    {t('editor.toolbar.signatureBlock')}
                  </DropdownMenuItem>
                  <DropdownMenuItem
                    onClick={() => editor.chain().focus().setConditional({}).run()}
                  >
                    <GitBranch className="mr-2 h-4 w-4 text-warning-foreground dark:text-warning" />
                    {t('editor.toolbar.conditionalBlock')}
                  </DropdownMenuItem>
                  <DropdownMenuItem
                    onClick={() => setHeaderEnabled(!headerEnabled)}
                    className={cn(headerEnabled && 'bg-accent')}
                  >
                    <PanelTop className="mr-2 h-4 w-4" />
                    {headerEnabled ? t('editor.toolbar.removeHeader') : t('editor.toolbar.addHeader')}
                  </DropdownMenuItem>
                  <DropdownMenuItem
                    onClick={() => editor.chain().focus().insertTable({ rows: 3, cols: 3, withHeaderRow: true }).run()}
                  >
                    <Table2 className="mr-2 h-4 w-4 text-primary" />
                    {t('editor.insertTable')}
                  </DropdownMenuItem>
                </>
              )}

              {/* Export/Import/Preview */}
              {(onExport || onImport || (templateId && versionId)) && (
                <>
                  <DropdownMenuSeparator />
                  <DropdownMenuLabel className="text-xs font-medium text-muted-foreground">
                    {t('editor.toolbar.actions')}
                  </DropdownMenuLabel>
                  {onExport && (
                    <DropdownMenuItem onClick={onExport}>
                      <Download className="mr-2 h-4 w-4" />
                      {t('editor.toolbar.exportDocumentShort')}
                    </DropdownMenuItem>
                  )}
                  {onImport && (
                    <DropdownMenuItem onClick={onImport}>
                      <Upload className="mr-2 h-4 w-4" />
                      {t('editor.toolbar.importDocumentShort')}
                    </DropdownMenuItem>
                  )}
                </>
              )}
            </DropdownMenuContent>
          </DropdownMenu>
        )}
      </div>
    </TooltipProvider>
  )
}

interface ToolbarButtonProps {
  children: React.ReactNode
  onClick: () => void
  isActive?: boolean
  disabled?: boolean
  tooltip: string
}

function ToolbarButton({
  children,
  onClick,
  isActive = false,
  disabled = false,
  tooltip,
}: ToolbarButtonProps) {
  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <Button
          type="button"
          variant={isActive ? 'secondary' : 'ghost'}
          size="sm"
          onClick={onClick}
          disabled={disabled}
          onMouseDown={(e) => e.preventDefault()}
          className="h-8 w-8 p-0"
        >
          {children}
        </Button>
      </TooltipTrigger>
      <TooltipContent side="bottom" className="text-xs">
        {tooltip}
      </TooltipContent>
    </Tooltip>
  )
}
