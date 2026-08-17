import { useEffect, useRef, type ComponentPropsWithoutRef } from "react"
import { useInView, useMotionValue, useSpring } from "motion/react"

import { cn } from "@/lib/utils"

interface NumberTickerProps extends ComponentPropsWithoutRef<"span"> {
  value: number
  startValue?: number
  direction?: "up" | "down"
  delay?: number
  decimalPlaces?: number
}

export function NumberTicker({
  value,
  startValue = 0,
  direction = "up",
  delay = 0,
  className,
  decimalPlaces = 0,
  ...props
}: NumberTickerProps) {
  const ref = useRef<HTMLSpanElement>(null)
  const motionValue = useMotionValue(direction === "down" ? value : startValue)
  // Criticamente amortecido e rígido o bastante pra acompanhar um slider
  // sendo arrastado em tempo real (~0.2s de acomodação) — com os valores
  // originais (damping 60/stiffness 100, bem superamortecido) o número
  // ficava visivelmente atrasado atrás do arraste contínuo do slider.
  const springValue = useSpring(motionValue, {
    damping: 40,
    stiffness: 400,
  })
  const isInView = useInView(ref, { once: true, margin: "0px" })
  const jaRevelou = useRef(false)

  // Primeira aparição: só anima (com o delay configurado) depois que o
  // elemento entra na tela — efeito de "revelar contando" pra uso como
  // estatística estática.
  useEffect(() => {
    if (!isInView || jaRevelou.current) return
    const timer = setTimeout(() => {
      motionValue.set(direction === "down" ? startValue : value)
      jaRevelou.current = true
    }, delay * 1000)
    return () => clearTimeout(timer)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isInView])

  // Qualquer mudança de `value` DEPOIS da primeira revelação atualiza na
  // hora, sem depender do elemento "entrar em vista" de novo —
  // `useInView` com `once: true` só dispara uma vez, então sem este
  // efeito à parte um valor plugado a estado ao vivo (ex: o total de
  // cada plano em Planos.tsx/MeuPlano.tsx, recalculado a cada arrasto do
  // slider) ficava travado no número de quando o elemento entrou na tela
  // pela primeira vez — inclusive travado em 0 se isso nunca tinha
  // acontecido ainda.
  useEffect(() => {
    if (!jaRevelou.current) return
    motionValue.set(direction === "down" ? startValue : value)
  }, [value, direction, startValue, motionValue])

  useEffect(
    () =>
      springValue.on("change", (latest) => {
        if (ref.current) {
          ref.current.textContent = Intl.NumberFormat("en-US", {
            minimumFractionDigits: decimalPlaces,
            maximumFractionDigits: decimalPlaces,
          }).format(Number(latest.toFixed(decimalPlaces)))
        }
      }),
    [springValue, decimalPlaces]
  )

  return (
    <span
      ref={ref}
      className={cn(
        "inline-block tracking-wider text-black tabular-nums dark:text-white",
        className
      )}
      {...props}
    >
      {startValue}
    </span>
  )
}
