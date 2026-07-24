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

export interface PendentePorAfiliado {
  afiliado_id: number;
  nome: string;
  email: string;
  total_pendente: number;
  quantidade: number;
}

export async function listarPendentesPorAfiliado(): Promise<PendentePorAfiliado[]> {
  const { data } = await apiDrenux.get<PendentePorAfiliado[]>('/drenux/afiliados/pendentes');
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
