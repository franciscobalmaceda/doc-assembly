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
import type { AutomationKey } from '../api/automation-keys-api'
import { useUpdateAutomationKey } from '../hooks/useAutomationKeys'
import { TenantMultiSelect } from './TenantMultiSelect'

interface ApiKeyEditDialogProps {
  open: boolean
  keyData: AutomationKey | null
  onClose: () => void
}

export function ApiKeyEditDialog({ open, keyData, onClose }: ApiKeyEditDialogProps) {
  return (
    <Dialog open={open} onOpenChange={onClose}>
      {open && keyData && (
        <ApiKeyEditDialogContent keyData={keyData} onClose={onClose} />
      )}
    </Dialog>
  )
}

function ApiKeyEditDialogContent({
  keyData,
  onClose,
}: {
  keyData: AutomationKey
  onClose: () => void
}) {
  const { t } = useTranslation()
  const { toast } = useToast()
  const [name, setName] = useState(keyData.name)
  const [nameError, setNameError] = useState('')
  const [selectedTenants, setSelectedTenants] = useState<string[]>(keyData.allowedTenants ?? [])
  const updateKey = useUpdateAutomationKey()

  const isLoading = updateKey.isPending

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
      await updateKey.mutateAsync({
        id: keyData.id,
        data: {
          name: name.trim(),
          allowedTenants: selectedTenants.length > 0 ? selectedTenants : [],
        },
      })
      toast({ title: t('administration.apiKeys.updateSuccess', 'API key updated') })
      onClose()
    } catch {
      toast({
        variant: 'destructive',
        title: t('common.error', 'Error'),
        description: t('administration.apiKeys.updateError', 'Failed to update API key'),
      })
    }
  }

  return (
    <DialogContent className="sm:max-w-md">
      <DialogHeader>
        <DialogTitle>
          {t('administration.apiKeys.form.editTitle', 'Edit API Key')}
        </DialogTitle>
        <DialogDescription>
          {t(
            'administration.apiKeys.form.editDescription',
            'Update the name or tenant restrictions for this key.'
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
            onClick={onClose}
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
            {isLoading ? t('common.saving', 'Saving...') : t('common.save', 'Save')}
          </button>
        </DialogFooter>
      </form>
    </DialogContent>
  )
}
