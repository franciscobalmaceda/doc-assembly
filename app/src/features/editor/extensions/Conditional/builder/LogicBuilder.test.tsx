import { describe, expect, it, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import { LogicBuilder } from './LogicBuilder'

vi.mock('../../../stores/injectables-store', () => ({
  useInjectablesStore: (
    selector: (state: { variables: [] }) => unknown
  ) => selector({ variables: [] }),
}))

vi.mock('./LogicGroup', () => ({
  LogicGroupItem: () => <div data-testid="logic-group-item" />,
}))

vi.mock('./FormulaSummary', () => ({
  FormulaSummary: () => <div data-testid="formula-summary" />,
}))

vi.mock('./LogicBuilderVariablesPanel', () => ({
  LogicBuilderVariablesPanel: ({ className }: { className?: string }) => (
    <div data-testid="logic-builder-variables-panel" className={className} />
  ),
}))

describe('LogicBuilder', () => {
  it('mounts variables sidebar using w-80 width class', () => {
    render(
      <LogicBuilder
        initialData={{
          id: 'root',
          type: 'group',
          logic: 'AND',
          children: [],
        }}
        onChange={vi.fn()}
      />
    )

    const sidebar = screen.getByTestId('logic-builder-variables-panel')
    expect(sidebar.className).toContain('w-80')
  })
})
