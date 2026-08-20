import type { Tema } from "@/themes"
import { cn } from "@/lib/utils"

interface ThemePickerProps {
  temas: Tema[]
  valor: string
  onValorChange: (id: string) => void
  className?: string
}

function ThemePicker({ temas, valor, onValorChange, className }: ThemePickerProps) {
  const temaAtual = temas.find((t) => t.id === valor)
  return (
    <div className={cn("space-y-2", className)}>
      <div className="flex flex-wrap gap-2">
        {temas.map((tema) => (
          <button
            key={tema.id}
            type="button"
            onClick={() => onValorChange(tema.id)}
            aria-label={tema.nome}
            aria-pressed={valor === tema.id}
            data-slot="theme-dot"
            className={cn(
              "size-7 shrink-0 rounded-full border-2 transition",
              valor === tema.id ? "scale-110 border-tinta" : "border-transparent hover:scale-105"
            )}
            style={{ background: tema.acento }}
          />
        ))}
      </div>
      {temaAtual && <p className="text-xs text-tinta-suave">{temaAtual.descricao}</p>}
    </div>
  )
}

export { ThemePicker }
