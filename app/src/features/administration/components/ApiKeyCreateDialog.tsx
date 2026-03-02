import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { cn } from '@/lib/utils'
import { Loader2 } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useToast } from '@/components/ui/use-toast'
import type { CreateAutomationKeyResponse } from '../api/automation-keys-api'
import { useCreateAutomationKey } from '../hooks/useAutomationKeys'
import { TenantMultiSelect } from './TenantMultiSelect'

interface ApiKeyCreateDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  onCreated: (result: CreateAutomationKeyResponse) => void
}

export function ApiKeyCreateDialog({ open, onOpenChange, onCreated }: ApiKeyCreateDialogProps) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      {open && (
        <ApiKeyCreateDialogContent onOpenChange={onOpenChange} onCreated={onCreated} />
      )}
    </Dialog>
  )
}

function ApiKeyCreateDialogContent({
  onOpenChange,
  onCreated,
}: {
  onOpenChange: (open: boolean) => void
  onCreated: (result: CreateAutomationKeyResponse) => void
}) {
  const { t } = useTranslation()
  const { toast } = useToast()
  const [name, setName] = useState('')
  const [nameError, setNameError] = useState('')
  const [selectedTenants, setSelectedTenants] = useState<string[]>([])
  const createKey = useCreateAutomationKey()

  const isLoading = createKey.isPending

  const validateForm = (): boolean => {
    if (!name.trim()) {
      setNameError(t('administration.apiKeys.form.nameRequired', 'Name is required'))
      return false
    }
    setNameError('')
    return true
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!validateForm()) return

    try {
      const result = await createKey.mutateAsync({
        name: name.trim(),
        ...(selectedTenants.length > 0 && { allowedTenants: selectedTenants }),
      })
      onOpenChange(false)
      onCreated(result)
    } catch {
      toast({
        variant: 'destructive',
        title: t('common.error', 'Error'),
        description: t('administration.apiKeys.createError', 'Failed to create API key'),
      })
    }
  }

  return (
    <DialogContent className="sm:max-w-md">
      <DialogHeader>
        <DialogTitle>
          {t('administration.apiKeys.form.createTitle', 'Create API Key')}
        </DialogTitle>
        <DialogDescription>
          {t(
            'administration.apiKeys.form.createDescription',
            'Create a new automation API key for machine-to-machine access.'
          )}
        </DialogDescription>
      </DialogHeader>

      <form onSubmit={handleSubmit} className="space-y-4">
        <div>
          <label className="mb-1.5 block text-sm font-medium">
            {t('administration.apiKeys.form.name', 'Key Name')} *
          </label>
          <input
            type="text"
            value={name}
            onChange={(e) => {
              setName(e.target.value)
              setNameError('')
            }}
            placeholder={t('administration.apiKeys.form.namePlaceholder', 'e.g. CI/CD Pipeline')}
            className={cn(
              'w-full rounded-sm border bg-transparent px-3 py-2 text-sm outline-none transition-colors focus:border-foreground',
              nameError ? 'border-destructive' : 'border-border'
            )}
            disabled={isLoading}
            autoFocus
          />
          {nameError && <p className="mt-1 text-xs text-destructive">{nameError}</p>}
        </div>

        <TenantMultiSelect
          value={selectedTenants}
          onChange={setSelectedTenants}
          disabled={isLoading}
        />

        <DialogFooter>
          <button
            type="button"
            onClick={() => onOpenChange(false)}
            disabled={isLoading}
            className="rounded-sm border border-border px-4 py-2 text-sm font-medium transition-colors hover:bg-muted disabled:opacity-50"
          >
            {t('common.cancel', 'Cancel')}
          </button>
          <button
            type="submit"
            disabled={!name.trim() || isLoading}
            className="inline-flex items-center gap-2 rounded-sm bg-foreground px-4 py-2 text-sm font-medium text-background transition-colors hover:bg-foreground/90 disabled:opacity-50"
          >
            {isLoading && <Loader2 size={16} className="animate-spin" />}
            {isLoading
              ? t('common.creating', 'Creating...')
              : t('common.create', 'Create')}
          </button>
        </DialogFooter>
      </form>
    </DialogContent>
  )
}
