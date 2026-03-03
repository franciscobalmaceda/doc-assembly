import { describe, expect, it, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import { SaveStatusIndicator } from './SaveStatusIndicator'

vi.mock('react-i18next', () => ({
  useTranslation: () => ({
    t: (key: string) => key,
  }),
}))

describe('SaveStatusIndicator', () => {
  it('applies overflow-safe classes to container and text', () => {
    const { container } = render(
      <SaveStatusIndicator
        status="saved"
        lastSavedAt={null}
        error={null}
      />
    )

    const wrapper = container.firstElementChild
    expect(wrapper).toBeDefined()
    expect(wrapper?.className).toContain('min-w-0')
    expect(wrapper?.className).toContain('max-w-[240px]')
    expect(wrapper?.className).toContain('overflow-hidden')

    const text = screen.getByText('editor.autoSave.saved')
    expect(text.className).toContain('truncate')
    expect(text.className).toContain('overflow-hidden')
    expect(text.className).toContain('max-w-[160px]')
  })

  it('renders retry action without shrinking in error state', () => {
    const { container } = render(
      <SaveStatusIndicator
        status="error"
        lastSavedAt={null}
        error={new Error('save failed')}
        onRetry={() => {}}
      />
    )

    const retryButton = screen.getByRole('button', { name: 'common.retry' })
    expect(retryButton).toBeDefined()
    expect(retryButton.parentElement?.className).toContain('shrink-0')
    expect(container.firstElementChild?.className).toContain('max-w-[240px]')
  })
})
