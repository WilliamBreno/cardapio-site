import { api } from './client';
import type { Pedido } from './types';

interface ItemPedidoInput {
  produto_id: number;
  variacao_id?: number;
  quantidade: number;
  sugestao_produto_id?: number;
}

interface ComboItemPedidoInput {
  combo_item_id: number;
  variacao_id?: number;
}

interface ComboPedidoInput {
  combo_id: number;
  quantidade: number;
  itens: ComboItemPedidoInput[];
}

interface CriarPedidoInput {
  cliente_nome: string;
  cliente_telefone: string;
  data_retirada: string;
  modo_entrega?: string;
  endereco_entrega?: string;
  endereco_rua?: string;
  endereco_numero?: string;
  endereco_complemento?: string;
  endereco_bairro?: string;
  endereco_cidade?: string;
  endereco_estado?: string;
  endereco_cep?: string;
  cupom_codigo?: string;
  itens: ItemPedidoInput[];
  combos?: ComboPedidoInput[];
}

export async function criarPedido(slug: string, input: CriarPedidoInput): Promise<Pedido> {
  const { data } = await api.post<Pedido>(`/lojas/${slug}/pedidos`, input);
  return data;
}

export async function criarCheckout(pedidoId: number): Promise<{ url: string }> {
  const { data } = await api.post<{ url: string }>(`/pedidos/${pedidoId}/checkout`);
  return data;
}