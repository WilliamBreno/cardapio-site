import axios from 'axios';
import { useAfiliadoAuthStore } from '../store/afiliadoAuthStore';

const apiAfiliado = axios.create({
  baseURL: import.meta.env.VITE_API_URL,
});

apiAfiliado.interceptors.request.use((config) => {
  const token = useAfiliadoAuthStore.getState().token;
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

interface LoginInput {
  email: string;
  senha: string;
}

interface AuthResponse {
  token: string;
}

interface MensagemResponse {
  mensagem: string;
}

export async function loginAfiliado(input: LoginInput): Promise<AuthResponse> {
  const { data } = await apiAfiliado.post<AuthResponse>('/afiliados/login', input);
  return data;
}

export async function solicitarResetSenhaAfiliado(email: string): Promise<MensagemResponse> {
  const { data } = await apiAfiliado.post<MensagemResponse>('/afiliados/esqueci-senha', { email });
  return data;
}

export async function redefinirSenhaAfiliado(token: string, senha: string): Promise<MensagemResponse> {
  const { data } = await apiAfiliado.post<MensagemResponse>('/afiliados/redefinir-senha', { token, senha });
  return data;
}

export interface LojaIndicada {
  id: number;
  nome: string;
  slug: string;
  created_at: string;
}

export interface DashboardAfiliado {
  codigo: string;
  lojas: LojaIndicada[];
  total_ganho: number;
  stripe_conectado: boolean;
}

export async function buscarDashboardAfiliado(): Promise<DashboardAfiliado> {
  const { data } = await apiAfiliado.get<DashboardAfiliado>('/afiliado/dashboard');
  return data;
}

export async function iniciarOnboardingStripeAfiliado(): Promise<{ url: string }> {
  const { data } = await apiAfiliado.post<{ url: string }>('/afiliado/stripe/onboarding');
  return data;
}

// Repasse manual de comissão pra pedidos pagos via Mercado Pago (Fase 5.5
// do roadmap — o Mercado Pago não faz split automático de 3 partes, então
// esse valor é registrado como pendente e repassado via Pix por fora).
export type StatusRepasse = 'pendente' | 'pago';

export interface RepasseAfiliado {
  id: number;
  afiliado_id: number;
  pedido_id: number;
  loja_id: number;
  loja?: { id: number; nome: string; slug: string };
  valor: number;
  status: StatusRepasse;
  pago_em: string | null;
  created_at: string;
}

export interface ExtratoAfiliado {
  repasses: RepasseAfiliado[];
  total_pendente: number;
}

export async function buscarExtratoAfiliado(): Promise<ExtratoAfiliado> {
  const { data } = await apiAfiliado.get<ExtratoAfiliado>('/afiliado/repasses');
  return data;
}