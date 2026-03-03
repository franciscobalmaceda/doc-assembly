import { useMemo, useState, useEffect, useCallback } from 'react'
import { useTranslation } from 'react-i18next'
import {
  DndContext,
  DragOverlay,
  closestCenter,
  KeyboardSensor,
  PointerSensor,
  useSensor,
  useSensors,
  type DragEndEvent,
  type DragStartEvent,
  type DropAnimation,
  defaultDropAnimationSideEffects,
} from '@dnd-kit/core'
import {
  SortableContext,
  sortableKeyboardCoordinates,
  verticalListSortingStrategy,
} from '@dnd-kit/sortable'
import { CSS } from '@dnd-kit/utilities'
import { AnimatePresence, motion } from 'framer-motion'
import { ChevronLeft, LayoutList, Plus, Rows3, Users, Trash2, AlertTriangle, CheckSquare, X } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { ScrollArea } from '@/components/ui/scroll-area'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { cn } from '@/lib/utils'
import { useSignerRolesStore } from '../stores/signer-roles-store'
import { SignerRoleItem } from './SignerRoleItem'
import { WorkflowConfigButton } from './workflow'
import type { Variable } from '../types/variables'

interface SignerRolesPanelProps {
  variables: Variable[]
  className?: string
  /** Whether the panel is editable (false when document is published) */
  editable?: boolean
}

export function SignerRolesPanel({
  variables,
  className,
  editable = true,
}: SignerRolesPanelProps) {
  const { t } = useTranslation()
  // Access raw roles and sort with useMemo to avoid infinite loop
  const rawRoles = useSignerRolesStore((state) => state.roles)
  const roles = useMemo(
    () => [...rawRoles].sort((a, b) => a.order - b.order),
    [rawRoles]
  )
  const addRole = useSignerRolesStore((state) => state.addRole)
  const updateRole = useSignerRolesStore((state) => state.updateRole)
  const deleteRole = useSignerRolesStore((state) => state.deleteRole)
  const reorderRoles = useSignerRolesStore((state) => state.reorderRoles)
  const isCollapsed = useSignerRolesStore((state) => state.isCollapsed)
  const toggleCollapsed = useSignerRolesStore((state) => state.toggleCollapsed)
  const isCompactMode = useSignerRolesStore((state) => state.isCompactMode)
  const toggleCompactMode = useSignerRolesStore((state) => state.toggleCompactMode)
  // Selection mode
  const isSelectionMode = useSignerRolesStore((state) => state.isSelectionMode)
  const selectedRoleIds = useSignerRolesStore((state) => state.selectedRoleIds)
  const enterSelectionMode = useSignerRolesStore((state) => state.enterSelectionMode)
  const exitSelectionMode = useSignerRolesStore((state) => state.exitSelectionMode)
  const toggleRoleSelection = useSignerRolesStore((state) => state.toggleRoleSelection)
  const deleteSelectedRoles = useSignerRolesStore((state) => state.deleteSelectedRoles)

  // Bulk delete confirmation dialog
  const [showBulkDeleteConfirmation, setShowBulkDeleteConfirmation] = useState(false)

  // Handle bulk delete confirmation
  const handleBulkDelete = useCallback(() => {
    setShowBulkDeleteConfirmation(true)
  }, [])

  const confirmBulkDelete = useCallback(() => {
    deleteSelectedRoles()
    setShowBulkDeleteConfirmation(false)
  }, [deleteSelectedRoles])

  // Escape key to exit selection mode
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && isSelectionMode) {
        exitSelectionMode()
      }
    }
    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [isSelectionMode, exitSelectionMode])

  // Estado para tracking del item siendo arrastrado
  const [activeId, setActiveId] = useState<string | null>(null)
  const activeRole = useMemo(
    () => roles.find((r) => r.id === activeId),
    [roles, activeId]
  )
  const activeIndex = useMemo(
    () => roles.findIndex((r) => r.id === activeId),
    [roles, activeId]
  )

  // Only enable sensors when editable - disables drag when read-only
  const sensors = useSensors(
    useSensor(PointerSensor, {
      activationConstraint: {
        distance: editable ? 8 : Infinity, // Disable drag when not editable
      },
    }),
    useSensor(KeyboardSensor, {
      coordinateGetter: sortableKeyboardCoordinates,
    })
  )

  // Animación suave al soltar el overlay con fade-out
  const dropAnimation: DropAnimation = {
    sideEffects: defaultDropAnimationSideEffects({
      styles: {
        active: {
          opacity: '0.5',
        },
      },
    }),
    duration: 250,
    easing: 'cubic-bezier(0.4, 0, 0.2, 1)',
    keyframes: ({ transform }) => [
      { transform: CSS.Transform.toString(transform.initial), opacity: 1 },
      { transform: CSS.Transform.toString(transform.final), opacity: 0 },
    ],
  }

  const handleDragStart = (event: DragStartEvent) => {
    setActiveId(event.active.id as string)
  }

  const handleDragEnd = (event: DragEndEvent) => {
    const { active, over } = event

    if (over && active.id !== over.id) {
      const oldIndex = roles.findIndex((role) => role.id === active.id)
      const newIndex = roles.findIndex((role) => role.id === over.id)
      reorderRoles(oldIndex, newIndex)
    }

    // Delay para que la animación del overlay termine antes de mostrar el item original
    setTimeout(() => setActiveId(null), 250)
  }

  const handleDragCancel = () => {
    setTimeout(() => setActiveId(null), 250)
  }

  const roleIds = useMemo(() => roles.map((role) => role.id), [roles])

  return (
    <motion.aside
      initial={false}
      className={cn(
        'flex h-full w-full min-w-0 flex-col border-l border-border bg-card shrink-0 overflow-hidden',
        className
      )}
    >
      {/* Header */}
      <div
        className={cn(
          'relative flex items-center pt-3 pb-2 border-b border-border shrink-0',
          isCollapsed ? 'px-2' : 'px-3'
        )}
      >
        {/* Título */}
        <div className="flex items-center gap-2 flex-1 min-w-0">
          <Users
            className={cn(
              'h-4 w-4 text-muted-foreground shrink-0 transition-opacity',
              isCollapsed && 'opacity-0'
            )}
          />
          <motion.span
            initial={false}
            animate={{
              opacity: isCollapsed ? 0 : 1,
              width: isCollapsed ? 0 : 'auto',
            }}
            transition={{ duration: 0.15, ease: [0.4, 0, 0.2, 1] }}
            className="text-[10px] font-mono uppercase tracking-widest text-muted-foreground overflow-hidden whitespace-nowrap"
          >
            {t('editor.signerRoles')}
          </motion.span>
        </div>

        {/* Role count - hide when collapsed */}
        <motion.span
          initial={false}
          animate={{
            opacity: isCollapsed ? 0 : 1,
            width: isCollapsed ? 0 : 'auto',
          }}
          transition={{ duration: 0.15, ease: [0.4, 0, 0.2, 1] }}
          className="text-xs text-muted-foreground/70 min-w-[1ch] text-center overflow-hidden"
        >
          {roles.length}
        </motion.span>

        {/* Acciones agrupadas */}
        <motion.div
          initial={false}
          animate={{
            opacity: isCollapsed ? 0 : 1,
            width: isCollapsed ? 0 : 'auto',
          }}
          transition={{ duration: 0.15, ease: [0.4, 0, 0.2, 1] }}
          className="flex items-center gap-0.5 ml-2 overflow-hidden"
        >
          {/* Selection mode toggle with tooltip - only show when editable */}
          {editable && roles.length > 0 && (
            <Tooltip>
              <TooltipTrigger asChild>
                <button
                  onClick={isSelectionMode ? exitSelectionMode : () => enterSelectionMode()}
                  className={cn(
                    'shrink-0 p-1 rounded-md transition-colors',
                    isSelectionMode
                      ? 'bg-muted text-foreground'
                      : 'hover:bg-muted text-muted-foreground'
                  )}
                  aria-label={isSelectionMode ? t('editor.roles.panel.selection.disable') : t('editor.roles.panel.selection.enable')}
                >
                  <AnimatePresence mode="wait">
                    {isSelectionMode ? (
                      <motion.div
                        key="close"
                        initial={{ scale: 0.5, opacity: 0 }}
                        animate={{ scale: 1, opacity: 1 }}
                        exit={{ scale: 0.5, opacity: 0 }}
                        transition={{ duration: 0.15 }}
                      >
                        <X className="h-4 w-4" />
                      </motion.div>
                    ) : (
                      <motion.div
                        key="select"
                        initial={{ scale: 0.5, opacity: 0 }}
                        animate={{ scale: 1, opacity: 1 }}
                        exit={{ scale: 0.5, opacity: 0 }}
                        transition={{ duration: 0.15 }}
                      >
                        <CheckSquare className="h-4 w-4" />
                      </motion.div>
                    )}
                  </AnimatePresence>
                </button>
              </TooltipTrigger>
              <TooltipContent side="bottom">
                {isSelectionMode ? t('editor.roles.panel.selection.disable') : t('editor.roles.panel.selection.enable')}
              </TooltipContent>
            </Tooltip>
          )}

          {/* Compact mode toggle with tooltip - hide in selection mode */}
          <AnimatePresence>
            {!isSelectionMode && (
              <motion.div
                initial={{ opacity: 0, width: 0 }}
                animate={{ opacity: 1, width: 'auto' }}
                exit={{ opacity: 0, width: 0 }}
                transition={{ duration: 0.15 }}
              >
                <Tooltip>
                  <TooltipTrigger asChild>
                    <button
                      onClick={toggleCompactMode}
                      className="shrink-0 p-1 rounded-md hover:bg-muted transition-colors"
                      aria-label={isCompactMode ? t('editor.roles.panel.view.expanded') : t('editor.roles.panel.view.compact')}
                    >
                      {isCompactMode ? (
                        <Rows3 className="h-4 w-4 text-muted-foreground" />
                      ) : (
                        <LayoutList className="h-4 w-4 text-muted-foreground" />
                      )}
                    </button>
                  </TooltipTrigger>
                  <TooltipContent side="bottom">
                    {isCompactMode ? t('editor.roles.panel.view.expandAll') : t('editor.roles.panel.view.compact')}
                  </TooltipContent>
                </Tooltip>
              </motion.div>
            )}
          </AnimatePresence>
        </motion.div>

        {!isCollapsed && (
          <button
            onClick={toggleCollapsed}
            className="shrink-0 p-1 rounded-md hover:bg-muted transition-colors ml-0.5"
            aria-label={t('editor.roles.panel.collapse.collapse')}
          >
            <ChevronLeft className="h-4 w-4" />
          </button>
        )}

        {isCollapsed && (
          <button
            onClick={toggleCollapsed}
            className="absolute inset-0 z-20 flex items-center justify-center hover:bg-muted/50 transition-colors"
            aria-label={t('editor.roles.panel.collapse.expand')}
          >
            <ChevronLeft className="h-4 w-4 text-muted-foreground" />
          </button>
        )}
      </div>

      {/* Content */}
      <AnimatePresence mode="wait">
        {!isCollapsed && (
          <motion.div
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            transition={{ duration: 0.15 }}
            className="flex-1 flex flex-col min-h-0"
          >
            {/* Workflow Configuration Button - only show when editable */}
            {editable && (
              <div className="p-4 pb-2">
                <WorkflowConfigButton />
              </div>
            )}

            {roles.length === 0 ? (
              // Empty state - centered content
              <div className="flex-1 flex flex-col items-center justify-center py-8 text-center px-4">
                <Users className="h-8 w-8 text-muted-foreground/40 mb-2" />
                <p className="text-sm text-muted-foreground">{t('editor.roles.empty.title')}</p>
                <p className="text-xs text-muted-foreground/70 mt-1">
                  {t('editor.roles.empty.description')}
                </p>
                {editable && (
                  <Button
                    variant="outline"
                    size="sm"
                    className="mt-4 border-border text-muted-foreground hover:text-foreground hover:border-foreground"
                    onClick={addRole}
                  >
                    <Plus className="h-4 w-4 mr-2" />
                    {t('editor.roles.addFirst')}
                  </Button>
                )}
              </div>
            ) : (
              <>
                {/* Scrollable role list */}
                <ScrollArea className="flex-1 min-w-0">
                  <div className="p-4 pb-8 space-y-3 min-w-0 w-full">
                    <DndContext
                      sensors={sensors}
                      collisionDetection={closestCenter}
                      onDragStart={handleDragStart}
                      onDragEnd={handleDragEnd}
                      onDragCancel={handleDragCancel}
                    >
                      <SortableContext
                        items={roleIds}
                        strategy={verticalListSortingStrategy}
                      >
                        <AnimatePresence mode="popLayout">
                          {roles.map((role, index) => (
                            <motion.div
                              key={role.id}
                              layout={!activeId}
                              initial={{ opacity: 0, y: -16 }}
                              animate={{ opacity: 1, y: 0 }}
                              exit={{ opacity: 0, y: -8, scale: 0.95 }}
                              transition={{
                                duration: 0.25,
                                ease: [0.4, 0, 0.2, 1],
                                layout: { duration: 0.2 },
                              }}
                              className="min-w-0 w-full"
                            >
                              <SignerRoleItem
                                role={role}
                                index={index}
                                isCompactMode={isCompactMode}
                                isDragging={role.id === activeId}
                                variables={variables}
                                allRoles={roles}
                                isSelectionMode={isSelectionMode}
                                isSelected={selectedRoleIds.includes(role.id)}
                                onToggleSelection={toggleRoleSelection}
                                onUpdate={updateRole}
                                onDelete={deleteRole}
                                editable={editable}
                              />
                            </motion.div>
                          ))}
                        </AnimatePresence>
                      </SortableContext>

                      {/* Overlay que sigue al cursor durante el drag */}
                      <DragOverlay dropAnimation={dropAnimation}>
                        {activeRole && (
                          <SignerRoleItem
                            role={activeRole}
                            index={activeIndex}
                            isCompactMode={isCompactMode}
                            isOverlay
                            variables={variables}
                            allRoles={roles}
                            isSelectionMode={isSelectionMode}
                            isSelected={selectedRoleIds.includes(activeRole.id)}
                            onToggleSelection={() => {}}
                            onUpdate={() => {}}
                            onDelete={() => {}}
                            editable={editable}
                          />
                        )}
                      </DragOverlay>
                    </DndContext>
                  </div>
                </ScrollArea>

                {/* Sticky footer with gradient - only show when editable */}
                {editable && (
                  <div className="relative shrink-0">
                    <div className="absolute -top-6 left-0 right-0 h-6 bg-gradient-to-t from-card to-transparent pointer-events-none" />
                    <AnimatePresence mode="wait">
                      {selectedRoleIds.length > 0 ? (
                        <motion.div
                          key="delete-action"
                          initial={{ opacity: 0, y: 20 }}
                          animate={{ opacity: 1, y: 0 }}
                          exit={{ opacity: 0, y: 20 }}
                          transition={{ duration: 0.2 }}
                          className="p-4 pt-2 bg-card"
                        >
                          <Button
                            variant="outline"
                            size="sm"
                            className="w-full border-border text-muted-foreground hover:text-foreground hover:border-foreground"
                            onClick={handleBulkDelete}
                          >
                            <Trash2 className="h-4 w-4 mr-2" />
                            {t('editor.roles.panel.delete.title', { count: selectedRoleIds.length })}
                          </Button>
                        </motion.div>
                      ) : (
                        <motion.div
                          key="add-action"
                          initial={{ opacity: 0, y: 20 }}
                          animate={{ opacity: 1, y: 0 }}
                          exit={{ opacity: 0, y: 20 }}
                          transition={{ duration: 0.2 }}
                          className="p-4 pt-2 bg-card"
                        >
                          <Button
                            variant="outline"
                            size="sm"
                            className="w-full border-border text-muted-foreground hover:text-foreground hover:border-foreground"
                            onClick={addRole}
                          >
                            <Plus className="h-4 w-4 mr-2" />
                            {t('editor.roles.panel.add')}
                          </Button>
                        </motion.div>
                      )}
                    </AnimatePresence>
                  </div>
                )}
              </>
            )}
          </motion.div>
        )}
      </AnimatePresence>

      {/* Bulk delete confirmation dialog */}
      <Dialog
        open={showBulkDeleteConfirmation}
        onOpenChange={setShowBulkDeleteConfirmation}
      >
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2">
              <AlertTriangle className="h-5 w-5 text-destructive" />
              {t('editor.roles.panel.delete.title', { count: selectedRoleIds.length })}
            </DialogTitle>
            <DialogDescription className="pt-2">
              {t('editor.roles.panel.delete.confirmation')}
            </DialogDescription>
          </DialogHeader>
          <DialogFooter className="gap-2 sm:gap-0">
            <Button
              variant="outline"
              onClick={() => setShowBulkDeleteConfirmation(false)}
            >
              {t('common.cancel')}
            </Button>
            <Button
              variant="outline"
              className="border-border hover:bg-muted"
              onClick={confirmBulkDelete}
            >
              {t('editor.roles.panel.delete.title', { count: selectedRoleIds.length })}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </motion.aside>
  )
}
