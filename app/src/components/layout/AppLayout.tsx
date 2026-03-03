import { Outlet } from '@tanstack/react-router'
import { motion } from 'framer-motion'
import { useCallback, useEffect, useRef } from 'react'
import { AppSidebar } from './AppSidebar'
import { MobileSidebar } from './MobileSidebar'
import { AppHeader } from './AppHeader'
import { useSidebarStore, useSidebarResizeSync } from '@/stores/sidebar-store'

// Variantes de animación - sidebar aparece inmediatamente
const sidebarVariants = {
  initial: { opacity: 1 },
  animate: { opacity: 1 },
}

export function AppLayout() {
  const { toggleMobileOpen, isPinned, setHovering } = useSidebarStore()
  const hoverTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null)

  // Sync mobile sidebar state on resize (close when crossing to desktop)
  useSidebarResizeSync()

  const handleMouseEnter = useCallback(() => {
    if (isPinned) return

    if (hoverTimeoutRef.current) {
      clearTimeout(hoverTimeoutRef.current)
      hoverTimeoutRef.current = null
    }

    setHovering(true)
  }, [isPinned, setHovering])

  const handleMouseLeave = useCallback(() => {
    if (isPinned) return

    hoverTimeoutRef.current = setTimeout(() => {
      setHovering(false)
    }, 150)
  }, [isPinned, setHovering])

  useEffect(() => {
    return () => {
      if (hoverTimeoutRef.current) {
        clearTimeout(hoverTimeoutRef.current)
      }
    }
  }, [])

  return (
    <div className="flex h-screen overflow-hidden bg-background">
      {/* Header con botón de menú móvil integrado */}
      <AppHeader
        variant="full"
        showMobileMenu={true}
        onMobileMenuToggle={toggleMobileOpen}
      />

      {/* Mobile sidebar - Sheet based, only renders on mobile */}
      <MobileSidebar />

      {/* Desktop sidebar - hidden on mobile */}
      <motion.div
        variants={sidebarVariants}
        initial="initial"
        animate="animate"
        className="hidden lg:block lg:relative"
        onMouseEnter={handleMouseEnter}
        onMouseLeave={handleMouseLeave}
      >
        <AppSidebar />
      </motion.div>

      {/* Contenido principal */}
      <main className="flex flex-1 min-w-0 flex-col overflow-hidden pt-16">
        <div className="flex-1 min-w-0 overflow-auto">
          <Outlet />
        </div>
      </main>
    </div>
  )
}
