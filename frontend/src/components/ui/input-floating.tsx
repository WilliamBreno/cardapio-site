import { Field as FieldPrimitive } from "@base-ui/react/field"
import { Input as InputPrimitive } from "@base-ui/react/input"
import type { ComponentProps } from "react"

import { cn } from "@/lib/utils"

interface InputFloatingProps extends ComponentProps<typeof InputPrimitive> {
  label: string
  id: string
}

function InputFloating({ label, className, id, value, ...props }: InputFloatingProps) {
  // O estado "preenchido" é calculado a partir do `value` controlado (React),
  // não do pseudo-seletor CSS `:placeholder-shown` — testado ao vivo e
  // confirmado que a variante arbitrária `peer-[:not(:placeholder-shown)]`
  // não reagia de forma confiável nesse projeto (Tailwind v3), deixando o
  // label sobreposto ao texto digitado em vez de subir. Calcular em JS é
  // mais robusto e não depende de nenhuma sutileza de seletor CSS.
  const preenchido = typeof value === "string" && value.length > 0

  return (
    <FieldPrimitive.Root className="relative">
      <InputPrimitive
        id={id}
        placeholder=" "
        value={value}
        data-slot="input-floating"
        className={cn(
          "peer w-full rounded-lg border border-tinta/20 bg-fundo px-3 pt-5 pb-1.5 text-sm text-tinta outline-none transition focus:border-acento",
          className
        )}
        {...props}
      />
      <FieldPrimitive.Label
        htmlFor={id}
        className={cn(
          "pointer-events-none absolute left-3 top-4 -translate-y-1/2 bg-fundo px-1 text-sm text-tinta-suave transition-all peer-focus:top-2.5 peer-focus:translate-y-0 peer-focus:text-[11px] peer-focus:font-semibold peer-focus:text-acento",
          preenchido && "top-2.5 translate-y-0 text-[11px] font-semibold text-acento"
        )}
      >
        {label}
      </FieldPrimitive.Label>
    </FieldPrimitive.Root>
  )
}

export { InputFloating }
