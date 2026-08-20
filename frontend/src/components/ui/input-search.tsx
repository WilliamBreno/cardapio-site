import { Input as InputPrimitive } from "@base-ui/react/input"
import { Search, X } from "lucide-react"

import { cn } from "@/lib/utils"

export interface InputSearchProps {
  value: string
  onValueChange: (value: string) => void
  placeholder?: string
  className?: string
}

function InputSearch({ value, onValueChange, placeholder = "Buscar...", className }: InputSearchProps) {
  return (
    <div className={cn("relative", className)}>
      <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-tinta-suave" />
      <InputPrimitive
        value={value}
        onValueChange={onValueChange}
        placeholder={placeholder}
        data-slot="input-search"
        className="w-full rounded-full border border-tinta/20 bg-fundo py-2 pl-9 pr-9 text-sm text-tinta outline-none transition focus:border-acento"
      />
      {value.length > 0 && (
        <button
          type="button"
          onClick={() => onValueChange("")}
          aria-label="Limpar busca"
          className="absolute right-2.5 top-1/2 flex size-5 -translate-y-1/2 items-center justify-center rounded-full bg-tinta/10 text-tinta-suave transition hover:bg-tinta/20"
        >
          <X className="size-3" />
        </button>
      )}
    </div>
  )
}

export { InputSearch }
