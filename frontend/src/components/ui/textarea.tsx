import type { ComponentProps } from "react"

import { cn } from "@/lib/utils"

interface TextareaProps extends ComponentProps<"textarea"> {
  maxLength: number
  value: string
}

function Textarea({ className, maxLength, value, ...props }: TextareaProps) {
  const restantes = maxLength - value.length
  return (
    <div className="relative">
      <textarea
        value={value}
        maxLength={maxLength}
        data-slot="textarea"
        className={cn(
          "min-h-[90px] w-full resize-y rounded-lg border border-tinta/20 bg-fundo px-3 py-2 pb-6 text-sm text-tinta outline-none transition focus:border-acento",
          className
        )}
        {...props}
      />
      <span className="pointer-events-none absolute bottom-2 right-3 rounded bg-fundo px-1 font-carimbo text-[11px] text-tinta-suave">
        {restantes} restantes
      </span>
    </div>
  )
}

export { Textarea }
