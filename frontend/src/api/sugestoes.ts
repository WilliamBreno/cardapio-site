import { api } from './client';
import type { SugestaoProduto, SugestaoCarrinhoItem, TipoDesconto } from './types';

// Admin — configuração dos vínculos produto origem → produto sugerido.
export interface SugestaoProdutoInput {
  produto_origem_id: number;
  produto_sugerido_id: number;
  tipo_desconto?: TipoDesconto | '';
  valor_desconto?: number;
}

export async function listarSugestoesProduto(): Promise<SugestaoProduto[]> {
  const { data } = await api.get<SugestaoProduto[]>('/admin/sugestoes-produto');
  return data;
}

export async function criarSugestaoProduto(input: SugestaoProdutoInput): Promise<SugestaoProduto> {
  const { data } = await api.post<SugestaoProduto>('/admin/sugestoes-produto', input);
  return data;
}

export async function deletarSugestaoProduto(id: number): Promise<void> {
  await api.delete(`/admin/sugestoes-produto/${id}`);
}

export interface ConfiguracaoPlataforma {
  sugestao_inteligente_preco_mensal: number;
}

export async function buscarConfiguracaoPlataforma(): Promise<ConfiguracaoPlataforma> {
  const { data } = await api.get<ConfiguracaoPlataforma>('/admin/configuracao-plataforma');
  return data;
}

// Assinatura da Sugestão Inteligente via Mercado Pago (Fase 6 Parte 3) —
// cobrança recorrente direto na conta da Drenux, independente do plano
// da loja.
export async function assinarSugestaoInteligente(): Promise<{ url: string }> {
  const { data } = await api.post<{ url: string }>('/admin/sugestao-inteligente/assinatura');
  return data;
}

export async function cancelarAssinaturaSugestaoInteligente(): Promise<void> {
  await api.delete('/admin/sugestao-inteligente/assinatura');
}

// Público — seção consolidada de sugestões exibida na revisão do
// carrinho antes do cliente finalizar (ver CarrinhoDrawer).
export async function buscarSugestoesCarrinho(slug: string, produtoIds: number[]): Promise<SugestaoCarrinhoItem[]> {
  if (produtoIds.length === 0) return [];
  const { data } = await api.get<SugestaoCarrinhoItem[]>(`/lojas/${slug}/sugestoes-carrinho`, {
    params: { produtos: produtoIds.join(',') },
  });
  return data;
}
