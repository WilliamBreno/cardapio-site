import { api } from './client';
import type { Combo } from './types';

export interface ComboItemInput {
  produto_id: number;
  quantidade: number;
}

export interface ComboInput {
  nome: string;
  descricao: string;
  foto_url: string;
  preco: number;
  disponivel: boolean;
  ordem: number;
  itens: ComboItemInput[];
}

export async function listarCombos(): Promise<Combo[]> {
  const { data } = await api.get<Combo[]>('/admin/combos');
  return data;
}

export async function criarCombo(input: ComboInput): Promise<Combo> {
  const { data } = await api.post<Combo>('/admin/combos', input);
  return data;
}

export async function atualizarCombo(id: number, input: ComboInput): Promise<Combo> {
  const { data } = await api.put<Combo>(`/admin/combos/${id}`, input);
  return data;
}

export async function deletarCombo(id: number): Promise<void> {
  await api.delete(`/admin/combos/${id}`);
}
