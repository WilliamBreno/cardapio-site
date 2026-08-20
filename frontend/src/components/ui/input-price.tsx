import { Input as InputPrimitive } from "@base-ui/react/input"
import type { ComponentProps } from "react"

import { cn } from "@/lib/utils"
import { inputVariants } from "./input"

function InputPrice({ className, ...props }: ComponentProps<typeof InputPrimitive>) {
  return (
    <div className="relative">
      <span className="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 font-carimbo text-sm font-semibold text-tinta-suave">
        R$
      </span>
      <InputPrimitive
        type="number"
        step="0.01"
        min="0"
        data-slot="input-price"
        className={cn(inputVariants({ status: "default" }), "pl-10 font-carimbo", className)}
        {...props}
      />
    </div>
  )
}

export { InputPrice }
