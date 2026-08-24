import type { ReactNode } from 'react';
import type { Categoria } from '../api/types';

interface Props {
  categorias: Categoria[];
  ativa: number | null;
  onSelecionar: (categoriaId: number | null) => void;
}

export function AbasCategorias({ categorias, ativa, onSelecionar }: Props) {
  return (
    <div className="flex gap-3 overflow-x-auto px-4 py-1 [scrollbar-width:none] [&::-webkit-scrollbar]:hidden">
      <BotaoAba ativa={ativa === null} onClick={() => onSelecionar(null)}>
        Tudo
      </BotaoAba>
      {categorias.map((categoria) => (
        <BotaoAba
          key={categoria.id}
          ativa={ativa === categoria.id}
          onClick={() => onSelecionar(categoria.id)}
        >
          {categoria.nome}
        </BotaoAba>
      ))}
    </div>
  );
}

function BotaoAba({
  children,
  ativa,
  onClick,
}: {
  children: ReactNode;
  ativa: boolean;
  onClick: () => void;
}) {
  return (
    <button
      onClick={onClick}
      className={`shrink-0 rounded-full border-2 px-5 py-1.5 font-display text-sm tracking-wide transition ${
        ativa
          ? 'border-acento bg-acento text-texto-claro'
          : 'border-tinta/25 bg-superficie text-tinta-suave hover:border-tinta/40'
      }`}
    >
      {children}
    </button>
  );
}