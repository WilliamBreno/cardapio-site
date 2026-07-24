import axios from 'axios';
import { useDrenuxAdminStore } from '../store/drenuxAdminStore';
import type { RepasseAfiliado } from './afiliado';

const apiDrenux = axios.create({
  baseURL: import.meta.env.VITE_API_URL || 'http://localhost:8080',
});

apiDrenux.interceptors.request.use((config) => {
  const secret = useDrenuxAdminStore.getState().secret;
  if (secret) {
    config.headers['X-Drenux-Admin-Secret'] = secret;
  }
  return config;
});

// Se o secret estiver errado/vencido, o backend devolve 401 — desloga
// automaticamente em vez de deixar a tela presa em chamadas que nunca
// vão funcionar (mesmo padrão do client.ts principal).
apiDrenux.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response?.status === 401) {
      useDrenuxAdminStore.getState().logout();
    }
    return Promise.reject(error);
  }
);

// AfiliadoComTotais é TODO afiliado cadastrado (mesmo sem nenhum
// lançamento ainda), com quanto já foi pago e quanto está pendente.
export interface AfiliadoComTotais {
  afiliado_id: number;
  nome: string;
  email: string;
  codigo: string;
  comissao_percentual: number;
  total_pendente: number;
  total_pago: number;
  quantidade: number;
}

export async function listarAfiliados(): Promise<AfiliadoComTotais[]> {
  const { data } = await apiDrenux.get<AfiliadoComTotais[]>('/drenux/afiliados');
  return data;
}

export async function buscarRepassesDoAfiliado(afiliadoId: number): Promise<RepasseAfiliado[]> {
  const { data } = await apiDrenux.get<RepasseAfiliado[]>(`/drenux/afiliados/${afiliadoId}/repasses`);
  return data;
}

export async function marcarRepassesComoPago(ids: number[]): Promise<{ marcados: number }> {
  const { data } = await apiDrenux.post<{ marcados: number }>('/drenux/repasses/marcar-pago', { ids });
  return data;
}

export interface CriarAfiliadoInput {
  nome: string;
  email: string;
  senha: string;
  // Fração da taxa de plataforma que esse afiliado recebe (0.376 = 37,6%)
  // — não é porcentagem "crua" (0-100), é a mesma fração usada no cálculo
  // no backend (ver domain.Afiliado.ComissaoPercentual).
  comissao_percentual: number;
}

export interface AfiliadoCriado {
  id: number;
  nome: string;
  email: string;
  codigo: string;
  comissao_percentual: number;
}

// Não existe autocadastro de afiliado — essa é a única forma de criar
// uma conta hoje, restrita a quem tem o secret de /drenux/*.
export async function criarAfiliado(input: CriarAfiliadoInput): Promise<AfiliadoCriado> {
  const { data } = await apiDrenux.post<AfiliadoCriado>('/drenux/afiliados', input);
  return data;
}
