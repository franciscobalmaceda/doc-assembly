import { useState, useCallback, useMemo } from 'react'
import { v4 as uuidv4 } from 'uuid'
import {
  DndContext,
  DragOverlay,
  useSensor,
  useSensors,
  MouseSensor,
  TouchSensor,
} from '@dnd-kit/core'
import type { DragStartEvent, DragEndEvent } from '@dnd-kit/core'
import { snapCenterToCursor } from '@dnd-kit/modifiers'
import { GripVertical } from 'lucide-react'
import { cn } from '@/lib/utils'
import type {
  LogicGroup,
  LogicRule,
  ConditionalSchema,
  RuleValue,
} from '../ConditionalExtension'
import { LogicBuilderContext } from './LogicBuilderContext'
import { LogicGroupItem } from './LogicGroup'
import { FormulaSummary } from './FormulaSummary'
import { LogicBuilderVariablesPanel } from './LogicBuilderVariablesPanel'
import type { InjectorType } from '../../../types/variables'
import type { VariableDragData } from '../../../types/drag'
import { useInjectablesStore } from '../../../stores/injectables-store'
import { VARIABLE_ICONS } from '../../Mentions/variables'

const ALLOWED_TYPES: InjectorType[] = [
  'TEXT',
  'NUMBER',
  'CURRENCY',
  'DATE',
  'BOOLEAN',
]

const SOURCE_TYPE_STYLES: Record<string, string> = {
  INTERNAL: 'border-internal-border/50 bg-internal-muted/30 text-internal-foreground',
  EXTERNAL: 'border-external-border/50 bg-external-muted/30 text-external-foreground',
  default: 'border-border bg-card text-foreground',
}

// Pure function - moved outside component for better memoization
const updateNodeRecursively = (
  current: LogicGroup,
  nodeId: string,
  changes: Partial<LogicRule | LogicGroup>
): LogicGroup => {
  if (current.id === nodeId) {
    return { ...current, ...changes } as LogicGroup
  }
  return {
    ...current,
    children: current.children.map((child) => {
      if (child.id === nodeId) {
        return { ...child, ...changes } as LogicRule | LogicGroup
      }
      if (child.type === 'group') {
        return updateNodeRecursively(child as LogicGroup, nodeId, changes)
      }
      return child
    }),
  }
}

interface LogicBuilderProps {
  initialData: ConditionalSchema
  onChange: (data: ConditionalSchema) => void
}

export const LogicBuilder = ({ initialData, onChange }: LogicBuilderProps) => {
  const [data, setData] = useState<ConditionalSchema>(
    initialData || {
      id: 'root',
      type: 'group',
      logic: 'AND',
      children: [],
    }
  )
  const [activeDragData, setActiveDragData] = useState<VariableDragData | null>(null)

  // Get variables from store
  const storeVariables = useInjectablesStore((s) => s.variables)

  const sensors = useSensors(
    useSensor(MouseSensor, { activationConstraint: { distance: 5 } }),
    useSensor(TouchSensor)
  )

  // Map store variables to LogicBuilder context format (used by rule components)
  const allVariables = useMemo(() => {
    return storeVariables
      .filter((v) => ALLOWED_TYPES.includes(v.type))
      .map((v) => ({
        id: v.variableId,
        label: v.label,
        type: v.type,
      }))
  }, [storeVariables])

  // --- ACTIONS ---

  const updateNode = useCallback(
    (nodeId: string, changes: Partial<LogicRule | LogicGroup>) => {
      const newData = updateNodeRecursively(data, nodeId, changes)
      setData(newData)
      onChange(newData)
    },
    [data, onChange]
  )

  const addRule = useCallback(
    (parentId: string) => {
      const newRule: LogicRule = {
        id: uuidv4(),
        type: 'rule',
        variableId: '',
        operator: 'eq',
        value: { mode: 'text', value: '' } as RuleValue,
      }
      const insertInto = (group: LogicGroup): LogicGroup => {
        if (group.id === parentId) {
          return { ...group, children: [...group.children, newRule] }
        }
        return {
          ...group,
          children: group.children.map((c) =>
            c.type === 'group' ? insertInto(c as LogicGroup) : c
          ),
        }
      }
      const newData = insertInto(data)
      setData(newData)
      onChange(newData)
    },
    [data, onChange]
  )

  const addGroup = useCallback(
    (parentId: string) => {
      const newGroup: LogicGroup = {
        id: uuidv4(),
        type: 'group',
        logic: 'AND',
        children: [],
      }
      const insertInto = (group: LogicGroup): LogicGroup => {
        if (group.id === parentId) {
          return { ...group, children: [...group.children, newGroup] }
        }
        return {
          ...group,
          children: group.children.map((c) =>
            c.type === 'group' ? insertInto(c as LogicGroup) : c
          ),
        }
      }
      const newData = insertInto(data)
      setData(newData)
      onChange(newData)
    },
    [data, onChange]
  )

  const removeNode = useCallback(
    (nodeId: string, parentId: string) => {
      const removeFrom = (group: LogicGroup): LogicGroup => {
        if (group.id === parentId) {
          return {
            ...group,
            children: group.children.filter((c) => c.id !== nodeId),
          }
        }
        return {
          ...group,
          children: group.children.map((c) =>
            c.type === 'group' ? removeFrom(c as LogicGroup) : c
          ),
        }
      }
      const newData = removeFrom(data)
      setData(newData)
      onChange(newData)
    },
    [data, onChange]
  )

  // --- DRAG HANDLERS ---
  const handleDragStart = (event: DragStartEvent) => {
    const dragData = event.active.data.current as VariableDragData | undefined
    setActiveDragData(dragData ?? null)
  }

  const handleDragEnd = (event: DragEndEvent) => {
    const { active, over } = event
    setActiveDragData(null)

    if (!over) return

    // Check if dropped on a rule variable field
    // ID format: rule-var-{ruleId}
    if (over.id.toString().startsWith('rule-var-')) {
      const ruleId = over.id.toString().replace('rule-var-', '')
      const variableId = active.id.toString()
      updateNode(ruleId, {
        variableId,
        value: { mode: 'text', value: '' } as RuleValue,
        operator: 'eq',
      })
    }
  }

  return (
    <DndContext
      sensors={sensors}
      onDragStart={handleDragStart}
      onDragEnd={handleDragEnd}
      modifiers={[snapCenterToCursor]}
    >
      <LogicBuilderContext.Provider
        value={{
          variables: allVariables,
          updateNode,
          addRule,
          addGroup,
          removeNode,
        }}
      >
        <div className="flex flex-row h-full bg-background overflow-hidden">
          {/* Variables Sidebar */}
          <LogicBuilderVariablesPanel className="w-80" />

          {/* Builder Area */}
          <div className="flex-1 flex flex-col min-w-0 overflow-hidden">
            <div className="flex-1 p-4 overflow-y-auto overflow-x-hidden bg-muted/50">
              <LogicGroupItem group={data} />
            </div>

            {/* Formula Summary */}
            <div className="border-t border-border p-3 bg-card">
              <FormulaSummary schema={data} />
            </div>
          </div>
        </div>

        <DragOverlay zIndex={100} dropAnimation={null}>
          {activeDragData ? (
            <DragOverlayItem data={activeDragData} />
          ) : null}
        </DragOverlay>
      </LogicBuilderContext.Provider>
    </DndContext>
  )
}

const DragOverlayItem = ({ data }: { data: VariableDragData }) => {
  const Icon = VARIABLE_ICONS[data.injectorType] || VARIABLE_ICONS.TEXT
  const style = SOURCE_TYPE_STYLES[data.sourceType ?? 'default'] || SOURCE_TYPE_STYLES.default

  return (
    <div
      className={cn(
        'flex items-center gap-2 px-3 py-2 text-sm border rounded-md bg-card shadow-lg cursor-grabbing select-none',
        style
      )}
    >
      <GripVertical className="h-3.5 w-3.5" />
      <Icon className="h-3.5 w-3.5" />
      <span className="truncate font-medium">{data.label}</span>
      <span className="text-[10px] font-mono uppercase tracking-wider opacity-70 whitespace-nowrap">
        {data.injectorType}
      </span>
    </div>
  )
}
