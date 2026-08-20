import { useRef, type ComponentProps } from "react"
import { Calendar } from "lucide-react"

import { cn } from "@/lib/utils"

function InputDate({ className, ...props }: ComponentProps<"input">) {
  const ref = useRef<HTMLInputElement>(null)

  function abrirSeletor() {
    // showPicker() garante que clicar em QUALQUER parte do campo abra o
    // calendário — o comportamento nativo do <input type="date"> só abre
    // isso clicando exatamente no ícone embutido, dependendo do navegador.
    ref.current?.showPicker?.()
  }

  return (
    <div
      onClick={abrirSeletor}
      data-slot="input-date"
      className={cn(
        "flex cursor-pointer items-center gap-2 rounded-lg border border-tinta/20 bg-fundo px-3 py-2 transition focus-within:border-acento",
        className
      )}
    >
      <Calendar className="size-4 shrink-0 text-tinta-suave" />
      <input
        ref={ref}
        type="date"
        className="w-full cursor-pointer border-none bg-transparent font-carimbo text-sm text-tinta outline-none [&::-webkit-calendar-picker-indicator]:hidden"
        {...props}
      />
    </div>
  )
}

export { InputDate }
