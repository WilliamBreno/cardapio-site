import { NumberField } from "@base-ui/react/number-field"
import { Minus, Plus } from "lucide-react"

import { cn } from "@/lib/utils"

export interface StepperProps {
  value: number
  onValueChange: (value: number) => void
  min?: number
  className?: string
}

function Stepper({ value, onValueChange, min = 0, className }: StepperProps) {
  return (
    <NumberField.Root
      value={value}
      min={min}
      onValueChange={(novoValor) => {
        if (novoValor !== null) onValueChange(novoValor)
      }}
      className={cn("inline-flex", className)}
    >
      <NumberField.Group
        data-slot="stepper"
        className="flex items-center gap-1 rounded-full border border-tinta/15 bg-superficie px-1 py-1"
      >
        <NumberField.Decrement
          aria-label="Diminuir quantidade"
          className="flex size-6 items-center justify-center rounded-full text-tinta transition hover:bg-fundo disabled:opacity-40"
        >
          <Minus className="size-3.5" />
        </NumberField.Decrement>
        <NumberField.Input
          readOnly
          className="w-6 border-none bg-transparent text-center font-carimbo text-sm text-tinta outline-none"
        />
        <NumberField.Increment
          aria-label="Aumentar quantidade"
          className="flex size-6 items-center justify-center rounded-full text-tinta transition hover:bg-fundo disabled:opacity-40"
        >
          <Plus className="size-3.5" />
        </NumberField.Increment>
      </NumberField.Group>
    </NumberField.Root>
  )
}

export { Stepper }
