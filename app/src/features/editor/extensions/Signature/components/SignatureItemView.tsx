import { cn } from '@/lib/utils'
import { useMemo, useRef, useCallback, useState, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import { User } from 'lucide-react'
import Moveable from 'react-moveable'
import type { SignatureItem, SignatureLineWidth } from '../types'
import { getLineWidthClasses } from '../signature-layouts'
import { useSignerRolesContextSafe } from '../../../context/SignerRolesContext'

interface ImageTransform {
  imageRotation: number
  imageScale: number
  imageX: number
  imageY: number
}

interface SignatureItemViewProps {
  signature: SignatureItem
  lineWidth: SignatureLineWidth
  className?: string
  editable?: boolean
  isImageSelected?: boolean
  onImageSelect?: () => void
  onImageDeselect?: () => void
  onImageTransformChange?: (transform: Partial<ImageTransform>) => void
}

export function SignatureItemView({
  signature,
  lineWidth,
  className,
  editable = false,
  isImageSelected = false,
  onImageSelect,
  onImageDeselect,
  onImageTransformChange,
}: SignatureItemViewProps) {
  const { t } = useTranslation()
  const lineWidthClasses = getLineWidthClasses(lineWidth)
  const rolesContext = useSignerRolesContextSafe()
  const containerRef = useRef<HTMLDivElement>(null)
  const imageRef = useRef<HTMLImageElement>(null)
  const [imageElement, setImageElement] = useState<HTMLImageElement | null>(
    null
  )

  // Sync ref to state for Moveable target
  useEffect(() => {
    setImageElement(imageRef.current)
  }, [signature.imageData])

  // Store last clamped position for onDragEnd
  const lastClampedPos = useRef({ x: 0, y: 0 })

  // Clamp position so image center stays within container
  const clampPosition = useCallback(
    (x: number, y: number): { x: number; y: number } => {
      if (!containerRef.current) return { x, y }

      const container = containerRef.current.getBoundingClientRect()

      // Bounds for keeping center inside container
      const maxX = container.width / 2
      const maxY = container.height / 2

      const clamped = {
        x: Math.max(-maxX, Math.min(maxX, x)),
        y: Math.max(-maxY, Math.min(maxY, y)),
      }

      lastClampedPos.current = clamped
      return clamped
    },
    []
  )

  const assignedRole = useMemo(() => {
    if (!signature.roleId || !rolesContext) return null
    return rolesContext.getRoleById(signature.roleId)
  }, [signature.roleId, rolesContext])

  // Compute transform from signature values
  const imageTransform = useMemo(() => {
    const rotation = signature.imageRotation ?? 0
    const scale = signature.imageScale ?? 1
    const x = signature.imageX ?? 0
    const y = signature.imageY ?? 0
    return `translate(${x}px, ${y}px) rotate(${rotation}deg) scale(${scale})`
  }, [
    signature.imageRotation,
    signature.imageScale,
    signature.imageX,
    signature.imageY,
  ])

  const imageStyles = useMemo(() => {
    if (!signature.imageData) return {}
    return {
      opacity: (signature.imageOpacity ?? 100) / 100,
      mixBlendMode: 'multiply' as const,
      transform: imageTransform,
    }
  }, [signature.imageData, signature.imageOpacity, imageTransform])

  const handleImageClick = useCallback(() => {
    if (editable && onImageSelect) {
      onImageSelect()
    }
  }, [editable, onImageSelect])

  const handleClickOutside = useCallback(() => {
    if (onImageDeselect) {
      onImageDeselect()
    }
  }, [onImageDeselect])

  return (
    <div
      className={cn('relative flex flex-col items-center pt-24', className)}
      onClick={handleClickOutside}
    >
      {/* Badge de rol asignado */}
      {assignedRole && (
        <div className="absolute top-2 left-1/2 -translate-x-1/2 z-10">
          <span className="inline-flex items-center gap-1 text-[10px] bg-foreground/5 text-muted-foreground px-2 py-0.5 rounded-full border border-border">
            <User className="h-2.5 w-2.5" />
            {assignedRole.label}
          </span>
        </div>
      )}

      {/* Imagen de firma — always reserve slot height so lines stay aligned */}
      <div
        ref={containerRef}
        className="mb-2 h-20 w-full flex items-end justify-center relative"
        onClick={(e) => e.stopPropagation()}
      >
        {signature.imageData && (
          <>
            <img
              ref={imageRef}
              src={signature.imageData}
              alt={t('editor.signature.title')}
              className={cn(
                'max-h-20 max-w-full object-contain',
                editable && !isImageSelected && 'cursor-pointer',
                editable && isImageSelected && 'cursor-move'
              )}
              style={imageStyles}
              onClick={handleImageClick}
            />

            {/* Moveable controls - only when editable and selected */}
            {editable && isImageSelected && imageElement && (
              <Moveable
                target={imageElement}
                draggable={true}
                rotatable={true}
                scalable={true}
                keepRatio={true}
                throttleDrag={0}
                throttleRotate={0}
                throttleScale={0}
                onDrag={({ target, translate }) => {
                  // Clamp position to keep center within container
                  const clamped = clampPosition(translate[0], translate[1])
                  const rotation = signature.imageRotation ?? 0
                  const scale = signature.imageScale ?? 1
                  target.style.transform = `translate(${clamped.x}px, ${clamped.y}px) rotate(${rotation}deg) scale(${scale})`
                }}
                onDragEnd={() => {
                  // Save last clamped position
                  if (onImageTransformChange) {
                    onImageTransformChange({
                      imageX: lastClampedPos.current.x,
                      imageY: lastClampedPos.current.y,
                    })
                  }
                }}
                onRotate={({ target, transform }) => {
                  target.style.transform = transform
                }}
                onRotateEnd={({ target }) => {
                  const style = target.style.transform
                  const rotateMatch = style.match(/rotate\(([^)]+)deg\)/)
                  if (rotateMatch && onImageTransformChange) {
                    onImageTransformChange({
                      imageRotation: parseFloat(rotateMatch[1]),
                    })
                  }
                }}
                onScale={({ target, transform }) => {
                  target.style.transform = transform
                }}
                onScaleEnd={({ target }) => {
                  const style = target.style.transform
                  const scaleMatch = style.match(/scale\(([^)]+)\)/)
                  if (scaleMatch && onImageTransformChange) {
                    onImageTransformChange({
                      imageScale: parseFloat(scaleMatch[1]),
                    })
                  }
                }}
              />
            )}
          </>
        )}
      </div>

      {/* Línea de firma */}
      <div className={cn('h-px bg-foreground', lineWidthClasses)} />

      {/* Label + subtitle — subtitle is absolute so it never affects item height */}
      <div className="mt-2 text-center relative">
        <p className="text-sm font-medium text-foreground">
          {signature.label || t('editor.signature.title')}
        </p>
        {signature.subtitle && (
          <p className="absolute left-0 right-0 text-xs text-muted-foreground mt-0.5">{signature.subtitle}</p>
        )}
      </div>
    </div>
  )
}
