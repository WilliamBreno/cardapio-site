import { useEffect, useRef, useState } from 'react';

interface Props {
  texto: string;
  className?: string;
}

// Ícone "i" clicável que explica uma função que não é óbvia só pelo nome
// do botão (ex: "Cadastro em massa"). É por clique, não só hover — assim
// funciona em toque (celular) e não só com mouse. Fecha ao clicar fora ou
// no próprio ícone de novo.
export function InfoTooltip({ texto, className = '' }: Props) {
  const [aberto, setAberto] = useState(false);
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!aberto) return;
    function aoClicarFora(e: MouseEvent) {
      if (ref.current && !ref.current.contains(e.target as Node)) setAberto(false);
    }
    document.addEventListener('mousedown', aoClicarFora);
    return () => document.removeEventListener('mousedown', aoClicarFora);
  }, [aberto]);

  return (
    <div ref={ref} className={`relative inline-flex ${className}`}>
      <button
        type="button"
        onClick={() => setAberto((v) => !v)}
        aria-label="Mais informações"
        className="flex h-4 w-4 shrink-0 items-center justify-center rounded-full border border-tinta/30 text-[10px] font-semibold leading-none text-tinta-suave hover:border-acento hover:text-acento"
      >
        i
      </button>
      {aberto && (
        <div
          role="tooltip"
          className="absolute left-1/2 top-full z-20 mt-2 w-64 -translate-x-1/2 rounded-xl bg-tinta px-3 py-2 text-xs leading-relaxed text-texto-claro shadow-lg"
        >
          {texto}
          <div className="absolute -top-1 left-1/2 h-2 w-2 -translate-x-1/2 rotate-45 bg-tinta" />
        </div>
      )}
    </div>
  );
}
