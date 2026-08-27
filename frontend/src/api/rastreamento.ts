import { api } from './client';

interface RastrearResponse {
  status_entrega: string;
  entregador_latitude: number;
  entregador_longitude: number;
  entregador_atualizado_em: string | null;
  // disponivel (Fase 7.4): rastreamento em tempo real é Pro/Scale — false
  // não é erro, é o backend avisando que o plano da loja não inclui mapa.
  disponivel: boolean;
  // codigo_confirmacao (24/08/2026): sempre vem preenchido, mesmo em
  // planos sem mapa ao vivo — o cliente mostra pro entregador, que
  // digita na tela de gerenciar entrega pra confirmar.
  codigo_confirmacao: string;
}

// Chamadas administrativas (exigem token — o interceptor do client.ts já
// injeta o Authorization automaticamente quando há um usuário logado).

// EtapaPedido: as 4 etapas do fluxo de preparo/entrega (Fase 10.2) — antes
// só existiam as duas últimas (saiu_para_entrega/entregue).
export type EtapaPedido = 'a_preparar' | 'preparando' | 'saiu_para_entrega' | 'entregue';

// codigoConfirmacao só é necessário (e checado pelo backend) quando
// statusEntrega === 'entregue' num pedido modo "entrega" — ver
// CompartilharLocalizacao.tsx. Nos outros casos, pode ser omitido.
export async function atualizarStatusEntrega(pedidoId: number, statusEntrega: EtapaPedido, codigoConfirmacao?: string): Promise<void> {
  await api.put(`/admin/pedidos/${pedidoId}/status-entrega`, { status_entrega: statusEntrega, codigo_confirmacao: codigoConfirmacao });
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

// --- Link público do entregador (26/08/2026) ---
// O entregador não tem login nesse sistema — essas três chamadas usam o
// token do pedido (na URL) como senha simples, em vez do Bearer que o
// client.ts injeta automaticamente pra quem está logado. São rotas
// públicas (/lojas/:slug/..., não /admin/...), então funcionam mesmo sem
// ninguém logado no navegador de quem está entregando.

export interface EntregadorResponse {
  status_entrega: string;
  cliente_nome: string;
  endereco_entrega: string;
  destino_latitude: number;
  destino_longitude: number;
  disponivel: boolean;
}

export async function buscarParaEntregador(slug: string, pedidoId: number, token: string): Promise<EntregadorResponse> {
  const { data } = await api.get<EntregadorResponse>(`/lojas/${slug}/pedidos/${pedidoId}/entregador`, {
    params: { token },
  });
  return data;
}

export async function atualizarLocalizacaoEntregador(slug: string, pedidoId: number, token: string, latitude: number, longitude: number): Promise<void> {
  await api.post(`/lojas/${slug}/pedidos/${pedidoId}/entregador/localizacao`, { latitude, longitude }, { params: { token } });
}

export async function atualizarStatusEntregador(slug: string, pedidoId: number, token: string, statusEntrega: 'saiu_para_entrega' | 'entregue', codigoConfirmacao?: string): Promise<void> {
  await api.put(
    `/lojas/${slug}/pedidos/${pedidoId}/entregador/status`,
    { status_entrega: statusEntrega, codigo_confirmacao: codigoConfirmacao },
    { params: { token } }
  );
}