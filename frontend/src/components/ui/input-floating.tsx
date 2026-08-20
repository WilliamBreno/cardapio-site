import { Field as FieldPrimitive } from "@base-ui/react/field"
import { Input as InputPrimitive } from "@base-ui/react/input"
import type { ComponentProps } from "react"

import { cn } from "@/lib/utils"

interface InputFloatingProps extends ComponentProps<typeof InputPrimitive> {
  label: string
  id: string
}

function InputFloating({ label, className, id, ...props }: InputFloatingProps) {
  return (
    <FieldPrimitive.Root className="relative">
      <InputPrimitive
        id={id}
        placeholder=" "
        data-slot="input-floating"
        className={cn(
          "peer w-full rounded-lg border border-tinta/20 bg-fundo px-3 pt-5 pb-1.5 text-sm text-tinta outline-none transition focus:border-acento",
          className
        )}
        {...props}
      />
      <FieldPrimitive.Label
        htmlFor={id}
        className="pointer-events-none absolute left-3 top-4 -translate-y-1/2 bg-fundo px-1 text-sm text-tinta-suave transition-all peer-focus:top-2.5 peer-focus:translate-y-0 peer-focus:text-[11px] peer-focus:font-semibold peer-focus:text-acento peer-[:not(:placeholder-shown)]:top-2.5 peer-[:not(:placeholder-shown)]:translate-y-0 peer-[:not(:placeholder-shown)]:text-[11px] peer-[:not(:placeholder-shown)]:font-semibold peer-[:not(:placeholder-shown)]:text-acento"
      >
        {label}
      </FieldPrimitive.Label>
    </FieldPrimitive.Root>
  )
}

export { InputFloating }
