import { Toggle } from "@base-ui/react/toggle"
import { ToggleGroup } from "@base-ui/react/toggle-group"

import { cn } from "@/lib/utils"

export interface SegmentedOption<T extends string> {
  valor: T
  label: string
}

interface SegmentedProps<T extends string> {
  opcoes: SegmentedOption<T>[]
  valor: T
  onValorChange: (valor: T) => void
  className?: string
}

function Segmented<T extends string>({ opcoes, valor, onValorChange, className }: SegmentedProps<T>) {
  return (
    <ToggleGroup
      value={[valor]}
      onValueChange={(novoValor) => {
        const escolhido = novoValor[novoValor.length - 1]
        if (escolhido) onValorChange(escolhido as T)
      }}
      data-slot="segmented"
      className={cn("inline-flex gap-1 rounded-full border border-tinta/15 bg-fundo p-1", className)}
    >
      {opcoes.map((opcao) => (
        <Toggle
          key={opcao.valor}
          value={opcao.valor}
          className="flex-1 rounded-full px-4 py-1.5 text-sm font-semibold text-tinta-suave transition data-[pressed]:bg-acento data-[pressed]:text-superficie"
        >
          {opcao.label}
        </Toggle>
      ))}
    </ToggleGroup>
  )
}

export { Segmented }
