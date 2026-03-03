import { beforeEach, describe, expect, it, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import { TooltipProvider } from '@/components/ui/tooltip'
import { SignerRolesPanel } from './SignerRolesPanel'
import { useSignerRolesStore } from '../stores/signer-roles-store'

vi.mock('react-i18next', () => ({
  initReactI18next: {
    type: '3rdParty',
    init: () => {},
  },
  useTranslation: () => ({
    t: (key: string) => key,
  }),
}))

describe('SignerRolesPanel collapsed header', () => {
  beforeEach(() => {
    useSignerRolesStore.getState().reset()
    useSignerRolesStore.getState().setRoles([
      {
        id: 'role-1',
        label: 'Signer 1',
        name: { type: 'text', value: '' },
        email: { type: 'text', value: '' },
        order: 1,
      },
    ])
    useSignerRolesStore.getState().toggleCollapsed()
  })

  it('uses toolbar-aligned vertical rhythm and renders full-surface expand control when collapsed', () => {
    const { container } = render(
      <TooltipProvider>
        <SignerRolesPanel variables={[]} />
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
      name: 'editor.roles.panel.collapse.expand',
    })
    expect(expandButton.className).toContain('absolute')
    expect(expandButton.className).toContain('inset-0')
    expect(expandButton.className).toContain('flex')
    expect(expandButton.className).toContain('items-center')
    expect(expandButton.className).toContain('justify-center')

    const panel = container.querySelector('aside')
    expect(panel).toBeDefined()
    expect(panel?.className).toContain('h-full')
    expect(panel?.className).toContain('w-full')
    expect(panel?.className).toContain('min-w-0')
    expect(panel?.className).not.toContain('transition-[width]')
    expect(panel?.getAttribute('data-collapsed-width')).toBeNull()
  })

  it('keeps role cards constrained to available panel width in expanded mode', () => {
    useSignerRolesStore.getState().reset()
    useSignerRolesStore.getState().setRoles([
      {
        id: 'role-1',
        label: 'Signer 1',
        name: { type: 'text', value: '' },
        email: { type: 'text', value: '' },
        order: 1,
      },
    ])

    const { container } = render(
      <TooltipProvider>
        <SignerRolesPanel variables={[]} />
      </TooltipProvider>
    )

    const scrollList = container.querySelector('div.p-4.pb-8.space-y-3')
    expect(scrollList).toBeDefined()
    expect(scrollList?.className).toContain('min-w-0')
    expect(scrollList?.className).toContain('w-full')

    const roleLabelInput = screen.getByDisplayValue('Signer 1')
    const roleCard = roleLabelInput.closest('div.border.border-border.rounded-lg')
    expect(roleCard).toBeDefined()
    expect(roleCard?.className).toContain('w-full')
    expect(roleCard?.className).toContain('min-w-0')
    expect(roleCard?.className).toContain('max-w-full')
    expect(roleCard?.className).toContain('overflow-hidden')

    const panel = container.querySelector('aside')
    expect(panel).toBeDefined()
    expect(panel?.getAttribute('data-expanded-width')).toBeNull()
    expect(panel?.className).toContain('h-full')
    expect(panel?.className).toContain('w-full')
    expect(panel?.className).toContain('min-w-0')
  })
})
