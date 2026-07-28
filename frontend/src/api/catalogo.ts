import { api } from './client';
import type { CardapioPublico } from './types';

export async function buscarCardapio(slug: string): Promise<CardapioPublico> {
  const { data } = await api.get<CardapioPublico>(`/lojas/${slug}`);
  // Normaliza campos de lista que podem não vir na resposta (ex: backend
  // ainda não atualizado com a Fase 6, ou qualquer resposta parcial) —
  // sem isso, "undefined.length"/"undefined.map" quebra o render inteiro
  // da página pública, sem nenhum ErrorBoundary pra conter o estrago.
  return {
    ...data,
    combos: data.combos ?? [],
    categorias: data.categorias ?? [],
    subcategorias: data.subcategorias ?? [],
    grupos_cor: data.grupos_cor ?? [],
    produtos: data.produtos ?? [],
  };
}
