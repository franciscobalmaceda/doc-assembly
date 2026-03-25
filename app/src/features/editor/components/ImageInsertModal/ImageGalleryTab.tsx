import { useState, useCallback, useEffect, useRef } from 'react'
import { ImageIcon, Upload, Search, ChevronLeft, ChevronRight, Loader2, Check } from 'lucide-react'
import { cn } from '@/lib/utils'
import { galleryApi, type GalleryAsset } from '../../api/gallery-api'
import type { ImageGalleryTabProps } from './types'

const PER_PAGE = 20

export function ImageGalleryTab({ onSelect }: ImageGalleryTabProps) {
  const [assets, setAssets] = useState<GalleryAsset[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [query, setQuery] = useState('')
  const [debouncedQuery, setDebouncedQuery] = useState('')
  const [loading, setLoading] = useState(false)
  const [uploading, setUploading] = useState(false)
  const [selectedKey, setSelectedKey] = useState<string | null>(null)
  const [resolvedURLs, setResolvedURLs] = useState<Record<string, string>>({})
  const [error, setError] = useState<string | null>(null)
  const fileInputRef = useRef<HTMLInputElement>(null)
  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null)

  const totalPages = Math.ceil(total / PER_PAGE)

  const handleQueryChange = useCallback((value: string) => {
    setQuery(value)
    if (debounceRef.current) clearTimeout(debounceRef.current)
    debounceRef.current = setTimeout(() => {
      setDebouncedQuery(value)
      setPage(1)
    }, 300)
  }, [])

  useEffect(() => {
    let cancelled = false
    setLoading(true)
    setError(null)

    const fetch = debouncedQuery.trim()
      ? galleryApi.search(debouncedQuery, page, PER_PAGE)
      : galleryApi.list(page, PER_PAGE)

    fetch
      .then((result) => {
        if (cancelled) return
        setAssets(result.items)
        setTotal(result.total)
      })
      .catch(() => {
        if (cancelled) return
        setError('Error al cargar la galería.')
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })

    return () => {
      cancelled = true
    }
  }, [debouncedQuery, page])

  useEffect(() => {
    const blobsCreated: string[] = []
    for (const asset of assets) {
      if (resolvedURLs[asset.key]) continue
      galleryApi
        .getSrc(asset.key)
        .then((url) => {
          if (url.startsWith('blob:')) blobsCreated.push(url)
          setResolvedURLs((prev) => ({ ...prev, [asset.key]: url }))
        })
        .catch(() => {
          // ignore thumbnail resolution failures
        })
    }
    return () => {
      blobsCreated.forEach((url) => URL.revokeObjectURL(url))
    }
  }, [assets]) // eslint-disable-line react-hooks/exhaustive-deps

  const handleSelectAsset = useCallback(
    (asset: GalleryAsset) => {
      setSelectedKey(asset.key)
      onSelect({
        src: `storage://${asset.key}`,
        isBase64: false,
      })
    },
    [onSelect],
  )

  const handleUploadClick = useCallback(() => {
    fileInputRef.current?.click()
  }, [])

  const handleFileChange = useCallback(
    async (e: React.ChangeEvent<HTMLInputElement>) => {
      const file = e.target.files?.[0]
      if (!file) return
      e.target.value = ''

      setUploading(true)
      setError(null)
      try {
        const asset = await galleryApi.upload(file)
        setAssets((prev) => [asset, ...prev])
        setTotal((prev) => prev + 1)
        setSelectedKey(asset.key)
        onSelect({ src: `storage://${asset.key}`, isBase64: false })
      } catch {
        setError('Error al subir la imagen. Verifica el formato y tamaño (máx. 10 MB).')
      } finally {
        setUploading(false)
      }
    },
    [onSelect],
  )

  return (
    <div className="min-h-[280px] flex flex-col gap-3">
      {/* Toolbar */}
      <div className="flex items-center gap-2">
        <div className="relative flex-1">
          <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 h-3.5 w-3.5 text-muted-foreground" />
          <input
            type="text"
            placeholder="Buscar imágenes..."
            value={query}
            onChange={(e) => handleQueryChange(e.target.value)}
            className="w-full rounded-none border border-border bg-background py-1.5 pl-8 pr-3 font-mono text-xs text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-1 focus:ring-foreground"
          />
        </div>
        <button
          type="button"
          onClick={handleUploadClick}
          disabled={uploading}
          className="flex items-center gap-1.5 rounded-none border border-border bg-background px-3 py-1.5 font-mono text-xs uppercase tracking-wider text-muted-foreground transition-colors hover:border-foreground hover:text-foreground disabled:opacity-50"
        >
          {uploading ? (
            <Loader2 className="h-3.5 w-3.5 animate-spin" />
          ) : (
            <Upload className="h-3.5 w-3.5" />
          )}
          Subir
        </button>
        <input
          ref={fileInputRef}
          type="file"
          accept="image/*"
          className="hidden"
          onChange={handleFileChange}
        />
      </div>

      {error && (
        <p className="font-mono text-xs text-destructive">{error}</p>
      )}

      {/* Grid */}
      <div className="flex-1">
        {loading ? (
          <div className="flex items-center justify-center h-40">
            <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
          </div>
        ) : assets.length === 0 ? (
          <div className="flex flex-col items-center justify-center h-40 text-muted-foreground">
            <ImageIcon className="h-10 w-10 mb-2 opacity-40" />
            <p className="font-mono text-xs">
              {debouncedQuery ? 'Sin resultados' : 'La galería está vacía'}
            </p>
          </div>
        ) : (
          <div className="grid grid-cols-4 gap-2">
            {assets.map((asset) => (
              <button
                key={asset.key}
                type="button"
                onClick={() => handleSelectAsset(asset)}
                className={cn(
                  'relative aspect-square overflow-hidden rounded-none border border-border bg-muted transition-all hover:border-foreground focus:outline-none',
                  selectedKey === asset.key && 'border-foreground ring-1 ring-foreground',
                )}
                title={asset.filename}
              >
                {resolvedURLs[asset.key] ? (
                  <img
                    src={resolvedURLs[asset.key]}
                    alt={asset.filename}
                    className="h-full w-full object-cover"
                  />
                ) : (
                  <div className="flex h-full items-center justify-center">
                    <ImageIcon className="h-5 w-5 opacity-30" />
                  </div>
                )}
                {selectedKey === asset.key && (
                  <div className="absolute inset-0 flex items-center justify-center bg-foreground/20">
                    <Check className="h-5 w-5 text-white drop-shadow" />
                  </div>
                )}
              </button>
            ))}
          </div>
        )}
      </div>

      {/* Pagination */}
      {totalPages > 1 && (
        <div className="flex items-center justify-between pt-1">
          <span className="font-mono text-xs text-muted-foreground">
            {total} imagen{total !== 1 ? 'es' : ''}
          </span>
          <div className="flex items-center gap-1">
            <button
              type="button"
              disabled={page <= 1}
              onClick={() => setPage((p) => p - 1)}
              className="rounded-none border border-border p-1 text-muted-foreground transition-colors hover:border-foreground hover:text-foreground disabled:opacity-40"
            >
              <ChevronLeft className="h-3.5 w-3.5" />
            </button>
            <span className="font-mono text-xs text-muted-foreground px-1">
              {page} / {totalPages}
            </span>
            <button
              type="button"
              disabled={page >= totalPages}
              onClick={() => setPage((p) => p + 1)}
              className="rounded-none border border-border p-1 text-muted-foreground transition-colors hover:border-foreground hover:text-foreground disabled:opacity-40"
            >
              <ChevronRight className="h-3.5 w-3.5" />
            </button>
          </div>
        </div>
      )}
    </div>
  )
}
