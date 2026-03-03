import { Input } from '@/components/ui/input'
import { ScrollArea } from '@/components/ui/scroll-area'
import { cn } from '@/lib/utils'
import { AnimatePresence, motion, type Transition } from 'framer-motion'
import { ChevronLeft, ChevronRight, ChevronsDownUp, ChevronsUpDown, Clock, Database, Loader2, Maximize2, Search, Users, Variable as VariableIcon, X } from 'lucide-react'
import { useCallback, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useRoleInjectables } from '../hooks/useRoleInjectables'
import { useInjectablesStore } from '../stores/injectables-store'
import { useVariablesPanelStore } from '../stores/variables-panel-store'
import type { VariableDragData } from '../types/drag'
import type { RoleInjectable } from '../types/role-injectable'
import type { Variable } from '../types/variables'
import { DraggableVariable } from './DraggableVariable'
import { VariableGroup } from './VariableGroup'
import { VariablesModal } from './VariablesModal'

const COLLAPSE_TRANSITION: Transition = { duration: 0.2, ease: [0.4, 0, 0.2, 1] }

interface VariablesPanelProps {
  /**
   * Optional click handler for variables
   * If provided, clicking a variable will call this handler
   */
  onVariableClick?: (data: VariableDragData) => void

  /**
   * IDs of currently dragging items (for visual feedback)
   */
  draggingIds?: string[]

  className?: string
}

/**
 * Collapsible left sidebar panel displaying all available variables
 * for drag-and-drop or click-to-insert into the editor
 *
 * Features:
 * - Collapsible with smooth animation (320px ↔ 56px)
 * - Global search for variables
 * - Grouped by: Global variables and Role injectables
 * - Visual differentiation for role variables (purple theme)
 * - Icon indicators for configurable format options
 * - Animated section collapse/expand with smooth transitions
 *
 * Layout:
 * ```
 * ┌─────────────────────────────┐
 * │ Variables        [▼] [42]  │ <- Header with collapse btn & count
 * ├─────────────────────────────┤
 * │ 🔍 Search...               │ <- Search input
 * ├─────────────────────────────┤
 * │ 👥 Roles de Firmantes       │ <- Role injectables section (animated)
 * │   ├── Cliente.nombre        │
 * │   └── Cliente.email         │
 * ├─────────────────────────────┤
 * │ 📦 Variables               │ <- Global variables section (animated)
 * │   ├── Client Name           │
 * │   ├── Amount                │
 * │   └── Date                  │
 * └─────────────────────────────┘
 * ```
 */
export function VariablesPanel({
  onVariableClick,
  draggingIds = [],
  className,
}: VariablesPanelProps) {
  const { t } = useTranslation()
  const isCollapsed = useVariablesPanelStore((state) => state.isCollapsed)
  const toggleCollapsed = useVariablesPanelStore((state) => state.toggleCollapsed)
  const [isModalOpen, setIsModalOpen] = useState(false)

  // Get variables from stores
  const { variables: globalVariables, groups, isLoading } = useInjectablesStore()
  const { roleInjectables } = useRoleInjectables()

  // Search state
  const [searchQuery, setSearchQuery] = useState('')

  // Collapsible sections state - all collapsed by default
  const [rolesSectionOpen, setRolesSectionOpen] = useState(false)

  // Filter state for variables by source type
  const [variablesFilter, setVariablesFilter] = useState<'all' | 'internal' | 'external'>('all')

  // Collapsible sections state for internal/external variables - collapsed by default
  const [internalSectionOpen, setInternalSectionOpen] = useState(false)
  const [externalSectionOpen, setExternalSectionOpen] = useState(false)

  // State for grouped variable sections (controlled mode)
  const [groupOpenStates, setGroupOpenStates] = useState<Record<string, boolean>>({})

  // Expand all sections when searching
  const isSearching = searchQuery.trim().length > 0

  // Expand all sections when search begins (adjust state based on derived value)
  const [prevIsSearching, setPrevIsSearching] = useState(isSearching)
  if (isSearching !== prevIsSearching) {
    setPrevIsSearching(isSearching)
    if (isSearching) {
      setRolesSectionOpen(true)
      setExternalSectionOpen(true)
      setInternalSectionOpen(true)
      setGroupOpenStates(() => {
        const newStates: Record<string, boolean> = {}
        for (const group of groups) {
          newStates[group.key] = true
        }
        return newStates
      })
    }
  }

  // Handler for individual group open state changes
  const handleGroupOpenChange = useCallback((groupKey: string, isOpen: boolean) => {
    setGroupOpenStates(prev => ({ ...prev, [groupKey]: isOpen }))
  }, [])

  // Collapse all sections
  const collapseAllSections = useCallback(() => {
    setRolesSectionOpen(false)
    setExternalSectionOpen(false)
    setInternalSectionOpen(false)
    setGroupOpenStates({})
  }, [])

  // Clear search and collapse all
  const handleClearSearch = useCallback(() => {
    setSearchQuery('')
    collapseAllSections()
  }, [collapseAllSections])

  // Handle search input change - collapse all when manually cleared
  const handleSearchChange = useCallback((value: string) => {
    const wasSearching = searchQuery.trim().length > 0
    const willBeEmpty = value.trim().length === 0

    setSearchQuery(value)

    // If user manually clears the search, collapse all
    if (wasSearching && willBeEmpty) {
      collapseAllSections()
    }
  }, [searchQuery, collapseAllSections])

  const lowerSearchQuery = searchQuery.toLowerCase().trim()

  const filteredRoleInjectables = useMemo(() => {
    if (!lowerSearchQuery) return roleInjectables
    return roleInjectables.filter(
      (ri) =>
        ri.label.toLowerCase().includes(lowerSearchQuery) ||
        ri.roleLabel.toLowerCase().includes(lowerSearchQuery) ||
        ri.propertyLabel.toLowerCase().includes(lowerSearchQuery)
    )
  }, [roleInjectables, lowerSearchQuery])

  const { groupedInternalVariables, ungroupedInternalVariables, groupedExternalVariables, ungroupedExternalVariables } = useMemo(() => {
    // Filter variables by source type and search query
    const filterBySourceType = (sourceType: 'INTERNAL' | 'EXTERNAL', excludeFilter: 'internal' | 'external'): Variable[] => {
      if (variablesFilter === excludeFilter) return []
      const filtered = globalVariables.filter(v => v.sourceType === sourceType)
      if (!lowerSearchQuery) return filtered
      return filtered.filter(
        (v) =>
          v.label.toLowerCase().includes(lowerSearchQuery) ||
          v.variableId.toLowerCase().includes(lowerSearchQuery)
      )
    }

    // Separate grouped and ungrouped variables for a given list
    const separateByGroup = (vars: Variable[]) => {
      const grouped = new Map<string, Variable[]>()
      const ungrouped: Variable[] = []

      for (const variable of vars) {
        if (variable.group) {
          const existing = grouped.get(variable.group) || []
          grouped.set(variable.group, [...existing, variable])
        } else {
          ungrouped.push(variable)
        }
      }

      // Sort groups by their order
      const sortedGrouped = Array.from(grouped.entries())
        .sort((a, b) => {
          const groupA = groups.find(g => g.key === a[0])
          const groupB = groups.find(g => g.key === b[0])
          return (groupA?.order ?? 99) - (groupB?.order ?? 99)
        })

      return { grouped: sortedGrouped, ungrouped }
    }

    const internalVars = filterBySourceType('INTERNAL', 'external')
    const externalVars = filterBySourceType('EXTERNAL', 'internal')

    const { grouped: groupedInternal, ungrouped: ungroupedInternal } = separateByGroup(internalVars)
    const { grouped: groupedExternal, ungrouped: ungroupedExternal } = separateByGroup(externalVars)

    return {
      groupedInternalVariables: groupedInternal,
      ungroupedInternalVariables: ungroupedInternal,
      groupedExternalVariables: groupedExternal,
      ungroupedExternalVariables: ungroupedExternal,
    }
  }, [globalVariables, groups, variablesFilter, lowerSearchQuery])

  // Convert Variable to VariableDragData
  const mapVariableToDragData = (v: Variable): VariableDragData => ({
    id: v.variableId,
    itemType: 'variable',
    variableId: v.variableId,
    label: v.label,
    injectorType: v.type,
    formatConfig: v.formatConfig,
    sourceType: v.sourceType,
    description: v.description,
  })

  // Convert RoleInjectable to VariableDragData
  const mapRoleToDragData = (ri: RoleInjectable): VariableDragData => ({
    id: ri.id,
    itemType: 'role-variable',
    variableId: ri.variableId,
    label: ri.label,
    injectorType: ri.type,
    roleId: ri.roleId,
    roleLabel: ri.roleLabel,
    propertyKey: ri.propertyKey,
    propertyLabel: ri.propertyLabel,
  })

  // Track if all sections are expanded (for toggle button)
  const allSectionsExpanded = useMemo(() => {
    const hasRoles = roleInjectables.length > 0
    const hasUngroupedExternal = ungroupedExternalVariables.length > 0
    const hasUngroupedInternal = ungroupedInternalVariables.length > 0
    const groupKeys = groups.map(g => g.key)

    const sectionsOpen = [
      !hasRoles || rolesSectionOpen,
      !hasUngroupedExternal || externalSectionOpen,
      !hasUngroupedInternal || internalSectionOpen,
      ...groupKeys.map(key => groupOpenStates[key] ?? false),
    ]

    return sectionsOpen.every(Boolean)
  }, [rolesSectionOpen, externalSectionOpen, internalSectionOpen, groupOpenStates, roleInjectables, ungroupedExternalVariables, ungroupedInternalVariables, groups])

  // Toggle all sections expand/collapse
  const toggleAllSections = useCallback(() => {
    const newState = !allSectionsExpanded
    setRolesSectionOpen(newState)
    setExternalSectionOpen(newState)
    setInternalSectionOpen(newState)
    setGroupOpenStates(prev => {
      const newStates = { ...prev }
      for (const group of groups) {
        newStates[group.key] = newState
      }
      return newStates
    })
  }, [allSectionsExpanded, groups])

  // Total count for badge
  const totalInternalGrouped = groupedInternalVariables.reduce((acc, [, vars]) => acc + vars.length, 0)
  const totalExternalGrouped = groupedExternalVariables.reduce((acc, [, vars]) => acc + vars.length, 0)
  const totalCount = filteredRoleInjectables.length + totalInternalGrouped + ungroupedInternalVariables.length + totalExternalGrouped + ungroupedExternalVariables.length

  return (
    <>
    <motion.aside
      initial={false}
      animate={{ width: isCollapsed ? 56 : 320 }}
      transition={COLLAPSE_TRANSITION}
      className={cn(
        'border-r border-border bg-card shrink-0 overflow-hidden grid-cols-1',
        className
      )}
    >
      {/* Header */}
      <div className="relative flex items-center pt-3 pb-2 px-3 border-b border-border">
        {!isCollapsed && (
          <>
            <div className="flex items-center gap-2 flex-1 min-w-0">
              <VariableIcon className="h-4 w-4 text-muted-foreground shrink-0" />
              <span className="truncate text-[10px] font-mono uppercase tracking-widest text-muted-foreground overflow-hidden whitespace-nowrap">
                {t('editor.variablesPanel.header')}
              </span>
            </div>

            <div className="shrink-0 flex items-center gap-1 ml-2">
              <span className="shrink-0 text-xs text-muted-foreground/70 min-w-[1ch] text-center">
                {totalCount}
              </span>

              <button
                onClick={toggleAllSections}
                className="shrink-0 p-1 rounded-md hover:bg-muted transition-colors"
                aria-label={allSectionsExpanded ? t('editor.variablesPanel.collapseAll') : t('editor.variablesPanel.expandAll')}
                title={allSectionsExpanded ? t('editor.variablesPanel.collapseAll') : t('editor.variablesPanel.expandAll')}
              >
                {allSectionsExpanded ? (
                  <ChevronsDownUp className="h-4 w-4 text-muted-foreground" />
                ) : (
                  <ChevronsUpDown className="h-4 w-4 text-muted-foreground" />
                )}
              </button>

              <button
                onClick={() => setIsModalOpen(true)}
                className="shrink-0 p-1 rounded-md hover:bg-muted transition-colors"
                aria-label={t('editor.variablesPanel.expandModal')}
                title={t('editor.variablesPanel.expandModal')}
              >
                <Maximize2 className="h-4 w-4 text-muted-foreground" />
              </button>

              <button
                onClick={toggleCollapsed}
                className="shrink-0 p-1 rounded-md hover:bg-muted transition-colors ml-1"
                aria-label={t('editor.variablesPanel.collapse.collapse')}
              >
                <ChevronRight className="h-4 w-4" />
              </button>
            </div>
          </>
        )}

        {isCollapsed ? (
          <button
            onClick={toggleCollapsed}
            className="absolute inset-0 flex items-center justify-center hover:bg-muted/50 transition-colors"
            aria-label={t('editor.variablesPanel.collapse.expand')}
          >
            <ChevronLeft className="h-4 w-4" />
          </button>
        ) : null}
      </div>

      {/* Content */}
      <AnimatePresence mode="wait">
        {!isCollapsed && (
          <motion.div
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            transition={{ duration: 0.15 }}
            className="min-h-0 flex flex-col overflow-hidden"
          >
            {/* Static Search Bar */}
            <div className="shrink-0 p-4 pb-2">
              <div className="relative min-w-0">
                <Search className="absolute left-2 top-2.5 h-4 w-4 text-muted-foreground" />
                <Input
                  placeholder={t('editor.variablesPanel.search.placeholder')}
                  className="pl-8 pr-8 h-9"
                  value={searchQuery}
                  onChange={(e) => handleSearchChange(e.target.value)}
                />
                {/* Clear button - appears when there's text */}
                <AnimatePresence>
                  {searchQuery.length > 0 && (
                    <motion.button
                      initial={{ opacity: 0, scale: 0.8 }}
                      animate={{ opacity: 1, scale: 1 }}
                      exit={{ opacity: 0, scale: 0.8 }}
                      transition={{ duration: 0.15 }}
                      onClick={handleClearSearch}
                      className="absolute right-2 top-2.5 h-4 w-4 text-muted-foreground hover:text-foreground transition-colors"
                      aria-label={t('common.clear')}
                    >
                      <X className="h-4 w-4" />
                    </motion.button>
                  )}
                </AnimatePresence>
              </div>
            </div>

            {/* Variables Filter Toggle - 3-step segmented control */}
            <div className="px-4 pb-2 min-w-0">
              <div className="flex min-w-0 w-full overflow-hidden rounded-none border border-border bg-background p-0.5">
                <button
                  onClick={() => setVariablesFilter('internal')}
                  title="Internal"
                  aria-label="Internal"
                  className={cn(
                    'flex-1 basis-0 min-w-0 flex items-center justify-center gap-1 px-2 py-1.5 text-[10px] font-mono uppercase tracking-wider transition-colors',
                    variablesFilter === 'internal'
                      ? 'bg-foreground text-background'
                      : 'text-muted-foreground hover:text-foreground'
                  )}
                >
                  <Clock className="h-3 w-3 shrink-0" />
                  <span className="truncate">Internal</span>
                </button>
                <button
                  onClick={() => setVariablesFilter('all')}
                  title="All"
                  aria-label="All"
                  className={cn(
                    'flex-1 basis-0 min-w-0 flex items-center justify-center px-2 py-1.5 text-[10px] font-mono uppercase tracking-wider transition-colors',
                    variablesFilter === 'all'
                      ? 'bg-foreground text-background'
                      : 'text-muted-foreground hover:text-foreground'
                  )}
                >
                  <span className="truncate">All</span>
                </button>
                <button
                  onClick={() => setVariablesFilter('external')}
                  title="External"
                  aria-label="External"
                  className={cn(
                    'flex-1 basis-0 min-w-0 flex items-center justify-center gap-1 px-2 py-1.5 text-[10px] font-mono uppercase tracking-wider transition-colors',
                    variablesFilter === 'external'
                      ? 'bg-foreground text-background'
                      : 'text-muted-foreground hover:text-foreground'
                  )}
                >
                  <Database className="h-3 w-3 shrink-0" />
                  <span className="truncate">External</span>
                </button>
              </div>
            </div>

            {/* Scroll container with gradient overlays */}
            <div className="relative flex-1 min-h-0 overflow-hidden">
              {/* Top fade area - solid bg + gradient */}
              <div className="absolute top-0 left-0 right-0 h-10 pointer-events-none z-10 flex flex-col">
                <div className="h-4 bg-card" />
                <div className="h-6 bg-linear-to-b from-card to-transparent" />
              </div>

              <ScrollArea className="h-full w-full [&>div]:overflow-x-hidden!">
                <div className="p-4 pt-8 pb-12 space-y-4 min-w-0 w-full overflow-hidden">
                {/* Loading state */}
                {isLoading && (
                  <div className="flex items-center justify-center py-8 text-muted-foreground">
                    <Loader2 className="h-4 w-4 animate-spin mr-2" />
                    <span className="text-xs">{t('editor.variablesPanel.loading')}</span>
                  </div>
                )}

                 {/* Empty state */}
                 {!isLoading &&
                   filteredRoleInjectables.length === 0 &&
                   groupedInternalVariables.length === 0 &&
                   ungroupedInternalVariables.length === 0 &&
                   groupedExternalVariables.length === 0 &&
                   ungroupedExternalVariables.length === 0 && (
                     <div className="flex flex-col items-center justify-center py-8 text-center">
                       <VariableIcon className="h-8 w-8 text-muted-foreground/40 mb-2" />
                      <p className="text-sm text-muted-foreground">
                        {t('editor.variablesPanel.empty.title')}
                      </p>
                      <p className="text-xs text-muted-foreground/70 mt-1">
                        {searchQuery.trim()
                          ? t('editor.variablesPanel.empty.searchSuggestion')
                          : t('editor.variablesPanel.empty.addSuggestion')}
                      </p>
                    </div>
                  )}

                {/* 1. Role Injectables Section */}
                {!isLoading && filteredRoleInjectables.length > 0 && (
                  <div className="space-y-2 min-w-0">
                    <button
                      onClick={() => setRolesSectionOpen(!rolesSectionOpen)}
                      className="flex min-w-0 items-center gap-2 px-1 text-[10px] font-mono uppercase tracking-widest text-role w-full hover:text-role/80 transition-colors"
                    >
                      <motion.div
                        animate={{ rotate: rolesSectionOpen ? 90 : 0 }}
                        transition={COLLAPSE_TRANSITION}
                        className="shrink-0"
                      >
                        <ChevronRight className="h-3 w-3 shrink-0" />
                      </motion.div>
                      <Users className="h-3 w-3 shrink-0" />
                      <span className="min-w-0 flex-1 truncate text-left">{t('editor.variablesPanel.sections.signerRoles')}</span>
                      <span className="shrink-0 text-[9px] bg-role-muted/50 text-role-foreground px-1.5 rounded">
                        {filteredRoleInjectables.length}
                      </span>
                    </button>

                    <motion.div
                      initial={false}
                      animate={{
                        height: rolesSectionOpen ? 'auto' : 0,
                        opacity: rolesSectionOpen ? 1 : 0,
                      }}
                      transition={{
                        duration: 0.2,
                        ease: [0.4, 0, 0.2, 1],
                      }}
                      style={{ overflow: 'hidden' }}
                    >
                      <div className="space-y-2 pt-2 min-w-0">
                        {filteredRoleInjectables.map((ri, index, array) => {
                          // Check if we need to add a separator before this item
                          // Add separator when:
                          // 1. Not the first item (index > 0)
                          // 2. AND the roleId is different from the previous item
                          const showSeparator = index > 0 && ri.roleId !== array[index - 1].roleId

                          return (
                            <div key={ri.id} className="min-w-0">
                              {/* Dotted line separator between different roles */}
                              {showSeparator && (
                                <div className="border-b border-dashed border-role-border/30 my-2 mx-1" />
                              )}
                              <DraggableVariable
                                data={mapRoleToDragData(ri)}
                                onClick={onVariableClick}
                                isDragging={draggingIds.includes(ri.id)}
                              />
                            </div>
                          )
                        })}
                      </div>
                    </motion.div>
                  </div>
                )}

                  {/* 2a. Grouped External Variables (provider injectables with groups) */}
                  {!isLoading && groupedExternalVariables.map(([groupKey, variables]) => {
                    const group = groups.find(g => g.key === groupKey)
                    if (!group) return null

                    return (
                      <VariableGroup
                        key={`ext-${groupKey}`}
                        group={group}
                        variables={variables}
                        onVariableClick={onVariableClick}
                        draggingIds={draggingIds}
                        isOpen={groupOpenStates[groupKey] ?? false}
                        onOpenChange={(open) => handleGroupOpenChange(groupKey, open)}
                      />
                    )
                  })}

                  {/* 2b. Ungrouped External Variables (provider injectables without groups) */}
                  {!isLoading && ungroupedExternalVariables.length > 0 && (
                    <div className="space-y-2 min-w-0">
                      <button
                        onClick={() => setExternalSectionOpen(!externalSectionOpen)}
                        className="flex min-w-0 items-center gap-2 px-1 text-[10px] font-mono uppercase tracking-widest text-external w-full hover:text-external/80 transition-colors"
                      >
                        <motion.div
                          animate={{ rotate: externalSectionOpen ? 90 : 0 }}
                          transition={COLLAPSE_TRANSITION}
                          className="shrink-0"
                        >
                          <ChevronRight className="h-3 w-3 shrink-0" />
                        </motion.div>
                        <Database className="h-3 w-3 shrink-0" />
                        <span className="min-w-0 flex-1 truncate text-left">{t('editor.variablesPanel.sections.externalVariables')}</span>
                        <span className="shrink-0 text-[9px] bg-external-muted/50 text-external-foreground px-1.5 rounded">
                          {ungroupedExternalVariables.length}
                        </span>
                      </button>

                      <motion.div
                        initial={false}
                        animate={{
                          height: externalSectionOpen ? 'auto' : 0,
                          opacity: externalSectionOpen ? 1 : 0,
                        }}
                        transition={{
                          duration: 0.2,
                          ease: [0.4, 0, 0.2, 1],
                        }}
                        style={{ overflow: 'hidden' }}
                      >
                        <div className="space-y-2 pt-2 min-w-0">
                          {ungroupedExternalVariables.map((v) => (
                            <DraggableVariable
                              key={v.variableId}
                              data={mapVariableToDragData(v)}
                              onClick={onVariableClick}
                              isDragging={draggingIds.includes(v.variableId)}
                            />
                          ))}
                        </div>
                      </motion.div>
                    </div>
                  )}

                  {/* 3. Ungrouped Internal Variables Section (system without group) */}
                  {!isLoading && ungroupedInternalVariables.length > 0 && (
                    <div className="space-y-2 min-w-0">
                      <button
                        onClick={() => setInternalSectionOpen(!internalSectionOpen)}
                        className="flex min-w-0 items-center gap-2 px-1 text-[10px] font-mono uppercase tracking-widest text-internal w-full hover:text-internal/80 transition-colors"
                      >
                        <motion.div
                          animate={{ rotate: internalSectionOpen ? 90 : 0 }}
                          transition={COLLAPSE_TRANSITION}
                          className="shrink-0"
                        >
                          <ChevronRight className="h-3 w-3 shrink-0" />
                        </motion.div>
                        <Clock className="h-3 w-3 shrink-0" />
                        <span className="min-w-0 flex-1 truncate text-left">{t('editor.variablesPanel.sections.internalVariables')}</span>
                        <span className="shrink-0 text-[9px] bg-internal-muted/50 text-internal-foreground px-1.5 rounded">
                          {ungroupedInternalVariables.length}
                        </span>
                      </button>

                      <motion.div
                        initial={false}
                        animate={{
                          height: internalSectionOpen ? 'auto' : 0,
                          opacity: internalSectionOpen ? 1 : 0,
                        }}
                        transition={{
                          duration: 0.2,
                          ease: [0.4, 0, 0.2, 1],
                        }}
                        style={{ overflow: 'hidden' }}
                      >
                        <div className="space-y-2 pt-2 min-w-0">
                          {ungroupedInternalVariables.map((v) => (
                            <DraggableVariable
                              key={v.variableId}
                              data={mapVariableToDragData(v)}
                              onClick={onVariableClick}
                              isDragging={draggingIds.includes(v.variableId)}
                            />
                          ))}
                        </div>
                      </motion.div>
                    </div>
                  )}

                  {/* 4. Grouped Internal Variables - Each group as its own collapsible section (system with group) */}
                  {!isLoading && groupedInternalVariables.map(([groupKey, variables]) => {
                    const group = groups.find(g => g.key === groupKey)
                    if (!group) return null

                    return (
                      <VariableGroup
                        key={groupKey}
                        group={group}
                        variables={variables}
                        onVariableClick={onVariableClick}
                        draggingIds={draggingIds}
                        isOpen={groupOpenStates[groupKey] ?? false}
                        onOpenChange={(open) => handleGroupOpenChange(groupKey, open)}
                      />
                    )
                  })}
                </div>
              </ScrollArea>

              {/* Bottom fade area - solid bg + gradient */}
              <div className="absolute bottom-0 left-0 right-0 h-10 pointer-events-none z-10 flex flex-col">
                <div className="h-6 bg-linear-to-t from-card to-transparent" />
                <div className="h-4 bg-card" />
              </div>
            </div>
          </motion.div>
        )}
      </AnimatePresence>
    </motion.aside>
      <VariablesModal
        open={isModalOpen}
        onOpenChange={setIsModalOpen}
        onVariableClick={onVariableClick}
        draggingIds={draggingIds}
      />
    </>
  )
}
