import { api } from './client';

interface CheckoutResponse {
  url: string;
}

interface VerificacaoResponse {
  email: string;
  plano: string;
  token: string;
}

// A confirmação do Mercado Pago é assíncrona (webhook) — enquanto o
// registro existe mas ainda não foi confirmado, o backend devolve 202
// com esse formato em vez de 404, pra o frontend saber que deve tentar
// de novo em vez de mostrar erro (ver FinalizarCadastro.tsx).
interface VerificacaoPendente {
  pendente: true;
}

export async function criarCheckoutAssinatura(plano: 'pro' | 'scale'): Promise<CheckoutResponse> {
  const { data } = await api.post<CheckoutResponse>('/planos/checkout', { plano });
  return data;
}

export async function verificarToken(token: string): Promise<VerificacaoResponse | VerificacaoPendente> {
  const { data } = await api.get<VerificacaoResponse | VerificacaoPendente>('/planos/verificar-token', { params: { token } });
  return data;
}

export async function verificarSessao(sessionId: string): Promise<VerificacaoResponse> {
  const { data } = await api.get<VerificacaoResponse>('/planos/verificar-sessao', { params: { session_id: sessionId } });
  return data;
}