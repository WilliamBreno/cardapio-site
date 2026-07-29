import { useState, type ReactNode } from 'react';
import type { Produto, Categoria, Subcategoria, GrupoCor } from '../api/types';
import { ProdutoCardGrid } from './ProdutoCardGrid';

interface Props {
  produtos: Produto[];
  categorias: Categoria[];
  subcategorias: Subcategoria[];
  gruposCor: GrupoCor[];
}

// Catálogo em formato de e-commerce (grid de produtos + navegação
// hierárquica Categoria → Subcategoria → Grupo de Cor) — layout exclusivo
// de lojas "mercadoria". Lojas "alimenticio" continuam usando o layout de
// lista com AbasCategorias (ver CardapioPublico.tsx).
export function CatalogoGrid({ produtos, categorias, subcategorias, gruposCor }: Props) {
  const [categoriaAtiva, setCategoriaAtiva] = useState<number | null>(null);
  const [subcategoriaAtiva, setSubcategoriaAtiva] = useState<number | null>(null);
  const [grupoCorAtivo, setGrupoCorAtivo] = useState<number | null>(null);

  const subsDaCategoria = categoriaAtiva !== null ? subcategorias.filter((s) => s.categoria_id === categoriaAtiva) : [];
  const gruposDaSubcategoria = subcategoriaAtiva !== null ? gruposCor.filter((g) => g.subcategoria_id === subcategoriaAtiva) : [];

  function selecionarCategoria(id: number | null) {
    setCategoriaAtiva(id);
    setSubcategoriaAtiva(null);
    setGrupoCorAtivo(null);
  }

  function selecionarSubcategoria(id: number | null) {
    setSubcategoriaAtiva(id);
    setGrupoCorAtivo(null);
  }

  const produtosFiltrados = produtos.filter((p) => {
    if (categoriaAtiva !== null && p.categoria_id !== categoriaAtiva) return false;
    if (subcategoriaAtiva !== null && p.subcategoria_id !== subcategoriaAtiva) return false;
    if (grupoCorAtivo !== null && p.grupo_cor_id !== grupoCorAtivo) return false;
    return true;
  });

  return (
    <div className="space-y-3 pt-2">
      <div className="flex gap-2 overflow-x-auto px-4 py-1 [scrollbar-width:none] [&::-webkit-scrollbar]:hidden">
        <FiltroChip ativo={categoriaAtiva === null} onClick={() => selecionarCategoria(null)}>Tudo</FiltroChip>
        {categorias.map((c) => (
          <FiltroChip key={c.id} ativo={categoriaAtiva === c.id} onClick={() => selecionarCategoria(categoriaAtiva === c.id ? null : c.id)}>
            {c.nome}
          </FiltroChip>
        ))}
      </div>

      {subsDaCategoria.length > 0 && (
        <div className="flex gap-2 overflow-x-auto px-4 py-1 [scrollbar-width:none] [&::-webkit-scrollbar]:hidden">
          <FiltroChip pequeno ativo={subcategoriaAtiva === null} onClick={() => selecionarSubcategoria(null)}>Todos</FiltroChip>
          {subsDaCategoria.map((s) => (
            <FiltroChip key={s.id} pequeno ativo={subcategoriaAtiva === s.id} onClick={() => selecionarSubcategoria(subcategoriaAtiva === s.id ? null : s.id)}>
              {s.nome}
            </FiltroChip>
          ))}
        </div>
      )}

      {gruposDaSubcategoria.length > 0 && (
        <div className="flex gap-2 overflow-x-auto px-4 py-1 [scrollbar-width:none] [&::-webkit-scrollbar]:hidden">
          <FiltroChip pequeno ativo={grupoCorAtivo === null} onClick={() => setGrupoCorAtivo(null)}>Todos os grupos</FiltroChip>
          {gruposDaSubcategoria.map((g) => (
            <FiltroChip key={g.id} pequeno ativo={grupoCorAtivo === g.id} onClick={() => setGrupoCorAtivo(grupoCorAtivo === g.id ? null : g.id)}>
              {g.nome}
            </FiltroChip>
          ))}
        </div>
      )}

      {produtosFiltrados.length === 0 ? (
        <p className="py-12 text-center text-tinta-suave">Nenhum produto por aqui ainda.</p>
      ) : (
        <div className="grid grid-cols-2 gap-3 px-4 sm:grid-cols-3">
          {produtosFiltrados.map((produto) => <ProdutoCardGrid key={produto.id} produto={produto} />)}
        </div>
      )}
    </div>
  );
}

function FiltroChip({
  children,
  ativo,
  onClick,
  pequeno,
}: {
  children: ReactNode;
  ativo: boolean;
  onClick: () => void;
  pequeno?: boolean;
}) {
  return (
    <button
      onClick={onClick}
      className={`shrink-0 rounded-full border-2 font-display tracking-wide transition ${
        pequeno ? 'px-3 py-1 text-xs' : 'px-5 py-1.5 text-sm'
      } ${
        ativo
          ? 'border-acento bg-acento text-superficie'
          : 'border-tinta/25 bg-superficie text-tinta-suave hover:border-tinta/40'
      }`}
    >
      {children}
    </button>
  );
}
