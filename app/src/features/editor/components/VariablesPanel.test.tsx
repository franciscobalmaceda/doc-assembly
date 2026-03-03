import { beforeEach, describe, expect, it, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import { TooltipProvider } from '@/components/ui/tooltip'
import { VariablesPanel } from './VariablesPanel'
import { useVariablesPanelStore } from '../stores/variables-panel-store'
import { useInjectablesStore } from '../stores/injectables-store'

vi.mock('react-i18next', () => ({
  useTranslation: () => ({
    t: (key: string) => key,
  }),
}))

vi.mock('../hooks/useRoleInjectables', () => ({
  useRoleInjectables: () => ({
    roleInjectables: [],
  }),
}))

describe('VariablesPanel collapsed header', () => {
  beforeEach(() => {
    useVariablesPanelStore.getState().reset()
    useInjectablesStore.getState().reset()

    useVariablesPanelStore.getState().setCollapsed(true)
    useInjectablesStore.getState().setVariables([
      {
        id: 'v-1',
        variableId: 'student_name',
        label: 'Student Name',
        type: 'TEXT',
        sourceType: 'INTERNAL',
      },
    ])
  })

  it('uses toolbar-aligned vertical rhythm and renders full-surface expand control when collapsed', () => {
    const { container } = render(
      <TooltipProvider>
        <VariablesPanel />
      </TooltipProvider>
    )

    const header = container.querySelector('aside > div')
    expect(header).toBeDefined()
    expect(header?.className).toContain('pt-3')
    expect(header?.className).toContain('pb-2')
    expect(header?.className).not.toContain('h-14')

    const collapsedBadge = container.querySelector('span.rounded-full')
    expect(collapsedBadge).toBeNull()

    const expandButton = screen.getByRole('button', {
      name: 'editor.variablesPanel.collapse.expand',
    })
    expect(expandButton.className).toContain('absolute')
    expect(expandButton.className).toContain('inset-0')
    expect(expandButton.className).toContain('justify-center')
  })

  it('keeps inline collapse button styling in expanded mode', () => {
    useVariablesPanelStore.getState().setCollapsed(false)
    useInjectablesStore.getState().setVariables([])

    render(
      <TooltipProvider>
        <VariablesPanel />
      </TooltipProvider>
    )

    const collapseButton = screen.getByRole('button', {
      name: 'editor.variablesPanel.collapse.collapse',
    })

    expect(collapseButton.className).toContain('ml-1')
    expect(collapseButton.className).not.toContain('absolute')
  })

  it('applies anti-overflow classes in expanded header and segmented filter', () => {
    useVariablesPanelStore.getState().setCollapsed(false)
    useInjectablesStore.getState().setVariables([
      {
        id: 'v-internal',
        variableId: 'current_date',
        label: 'Current Date',
        type: 'DATE',
        sourceType: 'INTERNAL',
      },
      {
        id: 'v-external',
        variableId: 'customer_name',
        label: 'Customer Name',
        type: 'TEXT',
        sourceType: 'EXTERNAL',
      },
    ])

    render(
      <TooltipProvider>
        <VariablesPanel />
      </TooltipProvider>
    )

    const panelTitle = screen.getByText('editor.variablesPanel.header')
    expect(panelTitle.className).toContain('truncate')

    const collapseAllButton = screen.getByRole('button', {
      name: 'editor.variablesPanel.expandAll',
    })
    expect(collapseAllButton.parentElement?.className).toContain('shrink-0')

    const internalFilterButton = screen.getByRole('button', { name: 'Internal' })
    const allFilterButton = screen.getByRole('button', { name: 'All' })
    const externalFilterButton = screen.getByRole('button', { name: 'External' })

    expect(internalFilterButton.className).toContain('basis-0')
    expect(internalFilterButton.className).toContain('min-w-0')
    expect(allFilterButton.className).toContain('basis-0')
    expect(allFilterButton.className).toContain('min-w-0')
    expect(externalFilterButton.className).toContain('basis-0')
    expect(externalFilterButton.className).toContain('min-w-0')

    const segmentedControl = internalFilterButton.parentElement
    expect(segmentedControl?.className).toContain('min-w-0')
    expect(segmentedControl?.className).toContain('w-full')
    expect(segmentedControl?.className).toContain('overflow-hidden')

    const externalSectionLabel = screen.getByText(
      'editor.variablesPanel.sections.externalVariables'
    )
    const internalSectionLabel = screen.getByText(
      'editor.variablesPanel.sections.internalVariables'
    )
    expect(externalSectionLabel.className).toContain('flex-1')
    expect(externalSectionLabel.className).toContain('truncate')
    expect(internalSectionLabel.className).toContain('flex-1')
    expect(internalSectionLabel.className).toContain('truncate')
  })

  it('uses 320px width when expanded', () => {
    useVariablesPanelStore.getState().setCollapsed(false)
    useInjectablesStore.getState().setVariables([])

    const { container } = render(
      <TooltipProvider>
        <VariablesPanel />
      </TooltipProvider>
    )

    const panel = container.querySelector('aside')
    expect(panel).toBeDefined()
    expect(panel?.getAttribute('style')).toContain('width: 320px')
  })
})
