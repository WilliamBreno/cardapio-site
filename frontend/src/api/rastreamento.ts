import { api } from './client';

interface RastrearResponse {
  status_entrega: string;
  entregador_latitude: number;
  entregador_longitude: number;
  entregador_atualizado_em: string | null;
  // disponivel (Fase 7.4): rastreamento em tempo real é Pro/Scale — false
  // não é erro, é o backend avisando que o plano da loja não inclui mapa.
  disponivel: boolean;
}

// Chamadas administrativas (exigem token — o interceptor do client.ts já
// injeta o Authorization automaticamente quando há um usuário logado).

export async function atualizarStatusEntrega(pedidoId: number, statusEntrega: 'saiu_para_entrega' | 'entregue'): Promise<void> {
  await api.put(`/admin/pedidos/${pedidoId}/status-entrega`, { status_entrega: statusEntrega });
}

export async function atualizarLocalizacao(pedidoId: number, latitude: number, longitude: number): Promise<void> {
  await api.post(`/admin/pedidos/${pedidoId}/localizacao`, { latitude, longitude });
}

// Chamada pública — usada pelo cliente final na página de rastreamento.
// O telefone funciona como "senha simples": só quem sabe o telefone
// usado no pedido consegue ver a localização.
export async function rastrearPedido(slug: string, pedidoId: number, telefone: string): Promise<RastrearResponse> {
  const { data } = await api.get<RastrearResponse>(`/lojas/${slug}/pedidos/${pedidoId}/rastrear`, {
    params: { telefone },
  });
  return data;
}

// Mesmo padrão, só que pra entrega de itens guardados (SolicitacaoEntrega
// em vez de Pedido) — ver Fase 3.
export async function atualizarStatusEntregaSolicitacao(solicitacaoId: number, statusEntrega: 'saiu_para_entrega' | 'entregue'): Promise<void> {
  await api.put(`/admin/solicitacoes/${solicitacaoId}/status-entrega`, { status_entrega: statusEntrega });
}

export async function atualizarLocalizacaoSolicitacao(solicitacaoId: number, latitude: number, longitude: number): Promise<void> {
  await api.post(`/admin/solicitacoes/${solicitacaoId}/localizacao`, { latitude, longitude });
}

export async function rastrearSolicitacao(slug: string, solicitacaoId: number, telefone: string): Promise<RastrearResponse> {
  const { data } = await api.get<RastrearResponse>(`/lojas/${slug}/solicitacoes/${solicitacaoId}/rastrear`, {
    params: { telefone },
  });
  return data;
}