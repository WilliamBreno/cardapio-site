import type { ReactNode } from "react"
import { X } from "lucide-react"

import { cn } from "@/lib/utils"

interface TagPillProps {
  children: ReactNode
  onRemove: () => void
  className?: string
}

function TagPill({ children, onRemove, className }: TagPillProps) {
  return (
    <span
      data-slot="tag-pill"
      className={cn(
        "inline-flex items-center gap-1.5 rounded-full bg-acento px-3 py-1 text-xs font-semibold text-superficie",
        className
      )}
    >
      {children}
      <button
        type="button"
        onClick={onRemove}
        aria-label={`Remover ${typeof children === "string" ? children : "filtro"}`}
        className="opacity-80 transition hover:opacity-100"
      >
        <X className="size-3" />
      </button>
    </span>
  )
}

export { TagPill }
