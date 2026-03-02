import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { ChevronDown, Search, X } from 'lucide-react'
import { cn } from '@/lib/utils'
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover'
import { Checkbox } from '@/components/ui/checkbox'
import { Label } from '@/components/ui/label'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Badge } from '@/components/ui/badge'
import { useSystemTenants } from '@/features/system-injectables/hooks/useSystemTenants'

interface TenantMultiSelectProps {
  value: string[]
  onChange: (ids: string[]) => void
  disabled?: boolean
}

export function TenantMultiSelect({ value, onChange, disabled }: TenantMultiSelectProps) {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)
  const [search, setSearch] = useState('')
  const [debouncedSearch, setDebouncedSearch] = useState('')

  useEffect(() => {
    const timer = setTimeout(() => setDebouncedSearch(search), 300)
    return () => clearTimeout(timer)
  }, [search])

  const { data, isLoading } = useSystemTenants(1, 50, debouncedSearch || undefined)
  const tenants = data?.data ?? []

  const handleToggle = (tenantId: string) => {
    onChange(
      value.includes(tenantId)
        ? value.filter((id) => id !== tenantId)
        : [...value, tenantId]
    )
  }

  const resolvedSelected = tenants.filter((tenant) => value.includes(tenant.id))
  const unresolvedIds = value.filter((id) => !tenants.some((tenant) => tenant.id === id))

  return (
    <div>
      <label className="mb-1.5 block text-sm font-medium">
        {t('administration.apiKeys.form.allowedTenants', 'Allowed Tenants')}
      </label>

      <Popover open={open} onOpenChange={setOpen}>
        <PopoverTrigger asChild>
          <button
            type="button"
            disabled={disabled}
            className={cn(
              'flex w-full items-center justify-between rounded-sm border border-border bg-transparent px-3 py-2 text-sm transition-colors focus:border-foreground disabled:opacity-50',
              !value.length && 'text-muted-foreground'
            )}
          >
            <span>
              {value.length > 0
                ? t('administration.apiKeys.form.tenantsSelected', '{{count}} tenant(s) selected', { count: value.length })
                : t('administration.apiKeys.form.allowedTenantsPlaceholder', 'Select tenants...')}
            </span>
            <ChevronDown size={14} className="text-muted-foreground" />
          </button>
        </PopoverTrigger>

        <PopoverContent align="start" className="w-[var(--radix-popover-trigger-width)] p-0">
          <div className="flex items-center gap-2 border-b border-border px-3 py-2">
            <Search size={14} className="text-muted-foreground" />
            <input
              type="text"
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder={t('common.search', 'Search...')}
              className="flex-1 bg-transparent text-sm outline-none placeholder:text-muted-foreground"
              autoFocus
            />
          </div>

          <ScrollArea className="max-h-48">
            <div className="p-2 space-y-1">
              {isLoading ? (
                <p className="px-2 py-3 text-center text-xs text-muted-foreground">
                  {t('common.loading', 'Loading...')}
                </p>
              ) : tenants.length === 0 ? (
                <p className="px-2 py-3 text-center text-xs text-muted-foreground">
                  {t('common.noResults', 'No results')}
                </p>
              ) : (
                tenants.map((tenant) => (
                  <div
                    key={tenant.id}
                    className="flex items-center gap-2 rounded-sm px-2 py-1.5 hover:bg-muted cursor-pointer"
                    onClick={() => handleToggle(tenant.id)}
                  >
                    <Checkbox
                      id={`tenant-${tenant.id}`}
                      checked={value.includes(tenant.id)}
                      onCheckedChange={() => handleToggle(tenant.id)}
                      onClick={(e) => e.stopPropagation()}
                    />
                    <Label
                      htmlFor={`tenant-${tenant.id}`}
                      className="flex-1 cursor-pointer text-xs"
                    >
                      {tenant.name}
                      <span className="ml-1.5 text-muted-foreground">({tenant.code})</span>
                    </Label>
                  </div>
                ))
              )}
            </div>
          </ScrollArea>
        </PopoverContent>
      </Popover>

      {value.length > 0 && (
        <div className="mt-2 flex flex-wrap gap-1">
          {resolvedSelected.map((tenant) => (
            <Badge key={tenant.id} variant="secondary" className="gap-1 pr-1">
              {tenant.name}
              <button
                type="button"
                onClick={() => handleToggle(tenant.id)}
                className="rounded-full p-0.5 hover:bg-muted-foreground/20"
              >
                <X size={10} />
              </button>
            </Badge>
          ))}
          {unresolvedIds.map((id) => (
            <Badge key={id} variant="outline" className="gap-1 pr-1">
              {id.slice(0, 8)}...
              <button
                type="button"
                onClick={() => handleToggle(id)}
                className="rounded-full p-0.5 hover:bg-muted-foreground/20"
              >
                <X size={10} />
              </button>
            </Badge>
          ))}
        </div>
      )}

      <p className="mt-1.5 text-xs text-muted-foreground">
        {t(
          'administration.apiKeys.form.allowedTenantsHint',
          'Leave empty for global access (all tenants).'
        )}
      </p>
    </div>
  )
}
