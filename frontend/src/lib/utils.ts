import { clsx, type ClassValue } from "clsx"
import { twMerge } from "tailwind-merge"
import type { Produto, TipoProduto, VariacaoProduto } from "../api/types"

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}

// precoItem calcula o preço unitário de um produto com a variação
// selecionada — centraliza a regra dos dois modos de preço de variação:
// "aditivo" soma ao preço base, "absoluto" substitui o preço base.
export function precoItem(produto: Produto, variacao?: VariacaoProduto | null): number {
  if (!variacao) return produto.preco
  if (variacao.modo_preco === 'absoluto') return variacao.preco_adicional
  return produto.preco + variacao.preco_adicional
}

// rotuloCatalogo decide a palavra usada nas telas do admin pra se referir
// à página pública da loja: lojas "mercadoria" usam o layout de e-commerce
// (ver CatalogoGrid) e por isso são chamadas de "catálogo"; "alimenticio"
// mantém "cardápio", nome histórico do produto.
export function rotuloCatalogo(segmento?: TipoProduto): string {
  return segmento === 'mercadoria' ? 'catálogo' : 'cardápio'
}

// rotuloCombo decide a palavra usada pra "Combo" nas telas do admin e no
// cardápio/catálogo público: lojas "mercadoria" usam "Kit" (mais comum
// pra varejo — kit de produtos), "alimenticio" mantém "Combo", nome
// histórico do produto (ex: combo de lanche). Singular, capitalizado —
// quem chama monta plural/minúsculo conforme o contexto.
export function rotuloCombo(segmento?: TipoProduto): string {
  return segmento === 'mercadoria' ? 'Kit' : 'Combo'
}
