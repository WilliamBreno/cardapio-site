import { api } from './client';
import type { Categoria, Subcategoria, GrupoCor, Produto, Pedido, Loja, TipoProduto, SolicitacaoEntrega } from './types';

// Categorias
export async function listarCategorias(): Promise<Categoria[]> {
  const { data } = await api.get<Categoria[]>('/admin/categorias');
  return data;
}

export async function criarCategoria(nome: string): Promise<Categoria> {
  const { data } = await api.post<Categoria>('/admin/categorias', { nome });
  return data;
}

export async function atualizarCategoria(id: number, nome: string): Promise<Categoria> {
  const { data } = await api.put<Categoria>(`/admin/categorias/${id}`, { nome });
  return data;
}

export async function deletarCategoria(id: number): Promise<void> {
  await api.delete(`/admin/categorias/${id}`);
}

// Subcategorias e Grupos — hierarquia Categoria → Subcategoria → Grupo,
// disponível pra qualquer segmento (ver plano-melhorias-drenux.md Fase 3
// e Fase 6, que liberou de "mercadoria" pra geral).
export async function listarSubcategorias(): Promise<Subcategoria[]> {
  const { data } = await api.get<Subcategoria[]>('/admin/subcategorias');
  return data;
}

export async function criarSubcategoria(categoriaId: number, nome: string): Promise<Subcategoria> {
  const { data } = await api.post<Subcategoria>(`/admin/categorias/${categoriaId}/subcategorias`, { nome });
  return data;
}

export async function atualizarSubcategoria(id: number, nome: string): Promise<Subcategoria> {
  const { data } = await api.put<Subcategoria>(`/admin/subcategorias/${id}`, { nome });
  return data;
}

export async function deletarSubcategoria(id: number): Promise<void> {
  await api.delete(`/admin/subcategorias/${id}`);
}

export async function listarGruposCor(): Promise<GrupoCor[]> {
  const { data } = await api.get<GrupoCor[]>('/admin/grupos-cor');
  return data;
}

export async function criarGrupoCor(subcategoriaId: number, nome: string): Promise<GrupoCor> {
  const { data } = await api.post<GrupoCor>(`/admin/subcategorias/${subcategoriaId}/grupos-cor`, { nome });
  return data;
}

export async function atualizarGrupoCor(id: number, nome: string): Promise<GrupoCor> {
  const { data } = await api.put<GrupoCor>(`/admin/grupos-cor/${id}`, { nome });
  return data;
}

export async function deletarGrupoCor(id: number): Promise<void> {
  await api.delete(`/admin/grupos-cor/${id}`);
}

// Produtos
export interface ProdutoInput {
  nome: string;
  descricao: string;
  preco: number;
  foto_url: string;
  disponivel: boolean;
  categoria_id: number;
  subcategoria_id: number | null;
  grupo_cor_id: number | null;
  estoque_atual: number | null;
  estoque_alerta: number | null;
  tipo_produto: TipoProduto;
  peso_gramas: number | null;
}

export async function listarProdutos(): Promise<Produto[]> {
  const { data } = await api.get<Produto[]>('/admin/produtos');
  return data;
}

export async function criarProduto(input: ProdutoInput): Promise<Produto> {
  const { data } = await api.post<Produto>('/admin/produtos', input);
  return data;
}

export async function atualizarProduto(id: number, input: ProdutoInput): Promise<Produto> {
  const { data } = await api.put<Produto>(`/admin/produtos/${id}`, input);
  return data;
}

export async function deletarProduto(id: number): Promise<void> {
  await api.delete(`/admin/produtos/${id}`);
}

// Pedidos
export async function listarPedidos(): Promise<Pedido[]> {
  const { data } = await api.get<Pedido[]>('/admin/pedidos');
  return data;
}

// Solicitações de entrega de itens guardados
export async function listarSolicitacoes(): Promise<SolicitacaoEntrega[]> {
  const { data } = await api.get<SolicitacaoEntrega[]>('/admin/solicitacoes');
  return data;
}

// Loja
export async function buscarLoja(): Promise<Loja> {
  const { data } = await api.get<Loja>('/admin/loja');
  return data;
}

export interface ConfiguracoesInput {
  whatsapp_numero: string;
  logo_url: string;
  modo_pedido: string;
  antecedencia_minima_horas: number;
  horario_abertura: string;
  horario_fechamento: string;
  margem_fechamento_minutos: number;
  pausado: boolean;
  mensagem_pausa: string;
  aceita_retirada: boolean;
  aceita_entrega: boolean;
  aceita_guardar_entregar: boolean;
  segmento_principal: 'alimenticio' | 'mercadoria';
  taxa_entrega_tipo: string;
  taxa_entrega_valor: number;
  valor_minimo_pedido: number;
  tema: string;
  // Precisa ir em TODO save (mesmo que a tela não mexa nisso) — o PUT
  // /admin/loja substitui a configuração inteira de uma vez, e um bool
  // omitido chega como "false" no backend, o que desligaria a Sugestão
  // Inteligente silenciosamente a cada salvamento de qualquer outra
  // configuração se esse campo não fosse explicitamente re-enviado.
  sugestao_inteligente_ativa: boolean;
}

export async function atualizarConfiguracoes(input: ConfiguracoesInput): Promise<void> {
  await api.put('/admin/loja', input);
}

// Plano
export interface MudarPlanoResponse {
  checkout_url: string;
  imediato: boolean;
}

export async function mudarPlano(plano: 'start' | 'pro' | 'scale'): Promise<MudarPlanoResponse> {
  const { data } = await api.post<MudarPlanoResponse>('/admin/plano/mudar', { plano });
  return data;
}

export async function cancelarMudancaAgendada(): Promise<void> {
  await api.delete('/admin/plano/agendamento');
}

// Variações
export interface VariacaoInput {
  nome: string;
  preco_adicional: number;
  disponivel: boolean;
  mostrar_valor_adicional: boolean;
  modo_preco: import('./types').ModoPrecoVariacao;
  estoque_atual: number | null;
  estoque_alerta: number | null;
  ordem: number;
}

export async function listarVariacoes(produtoId: number): Promise<import('./types').VariacaoProduto[]> {
  const { data } = await api.get(`/admin/variacoes/${produtoId}`);
  return data;
}

export async function criarVariacao(produtoId: number, input: VariacaoInput): Promise<import('./types').VariacaoProduto> {
  const { data } = await api.post(`/admin/variacoes/${produtoId}`, input);
  return data;
}

export async function atualizarVariacao(produtoId: number, variacaoId: number, input: VariacaoInput): Promise<import('./types').VariacaoProduto> {
  const { data } = await api.put(`/admin/variacoes/${produtoId}/${variacaoId}`, input);
  return data;
}

export async function deletarVariacao(produtoId: number, variacaoId: number): Promise<void> {
  await api.delete(`/admin/variacoes/${produtoId}/${variacaoId}`);
}

// Dashboard
export async function buscarDashboard(): Promise<import('./types').DashboardData> {
  const { data } = await api.get('/admin/dashboard');
  return data;
}

// Fotos de produto
export async function adicionarFoto(produtoId: number, url: string, ordem: number): Promise<import('./types').FotoProduto> {
  const { data } = await api.post(`/admin/fotos/${produtoId}`, { url, ordem });
  return data;
}

export async function deletarFoto(produtoId: number, fotoId: number): Promise<void> {
  await api.delete(`/admin/fotos/${produtoId}/${fotoId}`);
}

// Reordena a galeria de fotos do produto — a primeira da lista vira a
// foto principal exibida no cardápio.
export async function reordenarFotos(produtoId: number, ids: number[]): Promise<void> {
  await api.put(`/admin/fotos/${produtoId}/reordenar`, { ids });
}

// Fotos de variação (modo de preço "absoluto")
export async function adicionarFotoVariacao(produtoId: number, variacaoId: number, url: string, ordem: number): Promise<import('./types').FotoVariacao> {
  const { data } = await api.post(`/admin/variacoes/${produtoId}/${variacaoId}/fotos`, { url, ordem });
  return data;
}

export async function deletarFotoVariacao(produtoId: number, variacaoId: number, fotoId: number): Promise<void> {
  await api.delete(`/admin/variacoes/${produtoId}/${variacaoId}/fotos/${fotoId}`);
}

// Stripe (mantido só pra assinatura de plano — pedido migrou pro Mercado Pago, ver Fase 5)
export async function iniciarOnboardingStripe(): Promise<{ url: string }> {
  const { data } = await api.post<{ url: string }>('/admin/stripe/onboarding');
  return data;
}

export async function statusStripe(): Promise<{ stripe_conectado: boolean }> {
  const { data } = await api.get<{ stripe_conectado: boolean }>('/admin/stripe/status');
  return data;
}

// Mercado Pago — conexão da loja pra receber pagamento de pedido (Fase 5)
export async function iniciarOnboardingMercadoPago(): Promise<{ url: string }> {
  const { data } = await api.get<{ url: string }>('/admin/mercadopago/onboarding');
  return data;
}

export async function statusMercadoPago(): Promise<{ mercadopago_conectado: boolean }> {
  const { data } = await api.get<{ mercadopago_conectado: boolean }>('/admin/mercadopago/status');
  return data;
}

// Cupons
export interface CupomInput {
  codigo: string;
  tipo: 'percentual' | 'fixo';
  valor: number;
  ativo: boolean;
  uso_maximo: number | null;
  validade: string | null;
  valor_minimo_pedido: number;
}

export async function listarCupons(): Promise<import('./types').Cupom[]> {
  const { data } = await api.get('/admin/cupons');
  return data;
}

export async function criarCupom(input: CupomInput): Promise<import('./types').Cupom> {
  const { data } = await api.post('/admin/cupons', input);
  return data;
}

export async function atualizarCupom(id: number, input: CupomInput): Promise<import('./types').Cupom> {
  const { data } = await api.put(`/admin/cupons/${id}`, input);
  return data;
}

export async function deletarCupom(id: number): Promise<void> {
  await api.delete(`/admin/cupons/${id}`);
}