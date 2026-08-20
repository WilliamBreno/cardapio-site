import { Input as InputPrimitive } from "@base-ui/react/input"
import { cva, type VariantProps } from "class-variance-authority"
import { Check, X } from "lucide-react"

import { cn } from "@/lib/utils"

const inputVariants = cva(
  "w-full rounded-lg border bg-fundo px-3 py-2 text-sm text-tinta outline-none transition placeholder:text-tinta-suave/70 disabled:cursor-not-allowed disabled:opacity-60",
  {
    variants: {
      status: {
        default: "border-tinta/20 focus:border-acento",
        success: "border-emerald-500 bg-emerald-500/5 focus:border-emerald-500",
        error: "border-acento bg-acento/5 focus:border-acento",
      },
    },
    defaultVariants: {
      status: "default",
    },
  }
)

export interface InputProps
  extends Omit<InputPrimitive.Props, "className">,
    VariantProps<typeof inputVariants> {
  className?: string
}

function Input({ className, status, ...props }: InputProps) {
  const temIcone = status === "success" || status === "error"
  return (
    <div className="relative">
      <InputPrimitive
        data-slot="input"
        className={cn(inputVariants({ status }), temIcone && "pr-9", className)}
        {...props}
      />
      {status === "success" && (
        <Check className="pointer-events-none absolute right-3 top-1/2 size-4 -translate-y-1/2 text-emerald-500" />
      )}
      {status === "error" && (
        <X className="pointer-events-none absolute right-3 top-1/2 size-4 -translate-y-1/2 text-acento" />
      )}
    </div>
  )
}

export { Input, inputVariants }
