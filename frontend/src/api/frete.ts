import { api } from './client';
import type { EnderecoValor } from '../components/EnderecoCampos';
import { enderecoParaCampos } from '../components/EnderecoCampos';

interface CotarFreteResponse {
  distancia_km: number;
  valor_frete: number;
}

export async function cotarFrete(slug: string, endereco: string, enderecoValor: EnderecoValor): Promise<CotarFreteResponse> {
  const { data } = await api.post<CotarFreteResponse>(`/lojas/${slug}/cotar-frete`, {
    endereco,
    ...enderecoParaCampos(enderecoValor),
  });
  return data;
}
