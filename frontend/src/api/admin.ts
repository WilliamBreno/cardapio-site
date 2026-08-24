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

// Histórico de pedidos de um cliente específico (Fase 10.6) — usado no
// Dashboard, ao clicar num cliente da lista de top clientes.
export async function buscarHistoricoCliente(telefone: string): Promise<Pedido[]> {
  const { data } = await api.get<Pedido[]>(`/admin/clientes/${telefone}/pedidos`);
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
  // banner_url (campo único antigo, opcional) foi substituído pelo
  // carrossel de fotos — ver listarBanners/adicionarBanner/deletarBanner/
  // reordenarBanners abaixo. Não é mais enviado por Configuracoes.tsx.
  banner_url?: string;
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
  taxa_entrega_base?: number;
  taxa_entrega_por_km?: number;
  endereco?: string;
  endereco_rua?: string;
  endereco_numero?: string;
  endereco_complemento?: string;
  endereco_bairro?: string;
  endereco_cidade?: string;
  endereco_estado?: string;
  endereco_cep?: string;
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

// Carrossel de fotos do banner (redesign de 24/08/2026) — mesmo padrão
// de adicionarFoto/deletarFoto/reordenarFotos (produto), só que
// escopado direto pela loja do token, sem precisar de um :produtoId.
export async function listarBanners(): Promise<import('./types').FotoBanner[]> {
  const { data } = await api.get('/admin/banners');
  return data;
}

export async function adicionarBanner(url: string, ordem: number): Promise<import('./types').FotoBanner> {
  const { data } = await api.post('/admin/banners', { url, ordem });
  return data;
}

export async function deletarBanner(fotoId: number): Promise<void> {
  await api.delete(`/admin/banners/${fotoId}`);
}

export async function reordenarBanners(ids: number[]): Promise<void> {
  await api.put('/admin/banners/reordenar', { ids });
}

// Plano
export interface MudarPlanoResponse {
  checkout_url: string;
  imediato: boolean;
}

export async function mudarPlano(plano: 'start' | 'basic' | 'pro' | 'scale'): Promise<MudarPlanoResponse> {
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

// Resumo de período exato pra relatório via WhatsApp (Fase 10.5) — data no
// formato AAAA-MM-DD.
export async function buscarResumoPeriodo(tipo: 'dia' | 'semana' | 'mes', data: string): Promise<import('./types').PeriodoResumo> {
  const { data: resumo } = await api.get<import('./types').PeriodoResumo>('/admin/dashboard/periodo', {
    params: { tipo, data },
  });
  return resumo;
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

// Estoque (Fase 8) — relatório é Pro/Scale; reposição/ajuste/histórico
// são exclusivos do Scale (o backend recusa com 403 fora desses planos).
export interface ItemEstoque {
  produto_id: number;
  produto_nome: string;
  variacao_id: number | null;
  variacao_nome: string;
  estoque_atual: number;
  estoque_alerta: number | null;
  critico: boolean;
}

export type TipoMovimentoEstoque = 'venda' | 'reposicao' | 'ajuste';

export interface MovimentacaoEstoque {
  id: number;
  loja_id: number;
  produto_id: number;
  variacao_id: number | null;
  tipo: TipoMovimentoEstoque;
  quantidade: number;
  estoque_resultante: number;
  motivo: string;
  pedido_id: number | null;
  created_at: string;
}

export async function buscarRelatorioEstoque(): Promise<ItemEstoque[]> {
  const { data } = await api.get<ItemEstoque[]>('/admin/estoque');
  return data;
}

export async function buscarMovimentacoesEstoque(produtoId?: number): Promise<MovimentacaoEstoque[]> {
  const { data } = await api.get<MovimentacaoEstoque[]>('/admin/estoque/movimentacoes', {
    params: produtoId ? { produto_id: produtoId } : undefined,
  });
  return data;
}

export interface ReporEstoqueInput {
  produto_id: number;
  variacao_id?: number | null;
  quantidade: number;
  motivo?: string;
}

export async function reporEstoque(input: ReporEstoqueInput): Promise<{ estoque_atual: number }> {
  const { data } = await api.post<{ estoque_atual: number }>('/admin/estoque/repor', input);
  return data;
}

export interface AjustarEstoqueInput {
  produto_id: number;
  variacao_id?: number | null;
  novo_valor: number;
  motivo: string;
}

export async function ajustarEstoque(input: AjustarEstoqueInput): Promise<{ estoque_atual: number }> {
  const { data } = await api.post<{ estoque_atual: number }>('/admin/estoque/ajustar', input);
  return data;
}

// Lista de compras + relatórios avançados (Fase 9.3) — exclusivo do
// plano Scale.
export interface ItemListaCompras {
  insumo_id: number;
  nome: string;
  estoque_atual: number;
  estoque_alerta: number;
  unidade_uso: string;
  unidade_compra: string;
  deficit_unidade_uso: number;
  deficit_unidade_compra: number;
  produtos_afetados: string[];
}

export async function buscarListaDeCompras(): Promise<ItemListaCompras[]> {
  const { data } = await api.get<ItemListaCompras[]>('/admin/estoque/lista-compras');
  return data;
}

export interface ItemProdutoParado {
  produto_id: number;
  produto_nome: string;
}

export interface ItemGiroEstoque {
  produto_id: number;
  produto_nome: string;
  variacao_nome: string;
  estoque_atual: number;
  vendido_30d: number;
  giro: number;
}

export interface ItemInsumoMaisSai {
  insumo_id: number;
  nome: string;
  consumido_30d: number;
  unidade_uso: string;
}

export interface RelatoriosAvancados {
  produtos_parados: ItemProdutoParado[];
  giro_estoque: ItemGiroEstoque[];
  valor_parado_estoque: number;
  valor_parado_observacao: string;
  insumos_que_mais_saem: ItemInsumoMaisSai[];
}

export async function buscarRelatoriosAvancados(): Promise<RelatoriosAvancados> {
  const { data } = await api.get<RelatoriosAvancados>('/admin/estoque/relatorios');
  return data;
}

// Insumos + Ficha técnica + CMV automático (Fase 9.1) — exclusivo do
// plano Scale, o backend recusa com 403 fora desse plano.
export interface Insumo {
  id: number;
  loja_id: number;
  nome: string;
  unidade_compra: string;
  unidade_uso: string;
  fator_conversao: number;
  custo_unidade_compra: number;
  estoque_atual: number | null;
  estoque_alerta: number | null;
  created_at: string;
  updated_at: string;
}

export interface InsumoInput {
  nome: string;
  unidade_compra: string;
  unidade_uso: string;
  fator_conversao: number;
  custo_unidade_compra: number;
  estoque_atual: number | null;
  estoque_alerta: number | null;
}

export async function listarInsumos(): Promise<Insumo[]> {
  const { data } = await api.get<Insumo[]>('/admin/insumos');
  return data;
}

export async function criarInsumo(input: InsumoInput): Promise<Insumo> {
  const { data } = await api.post<Insumo>('/admin/insumos', input);
  return data;
}

export async function atualizarInsumo(id: number, input: InsumoInput): Promise<Insumo> {
  const { data } = await api.put<Insumo>(`/admin/insumos/${id}`, input);
  return data;
}

export async function deletarInsumo(id: number): Promise<void> {
  await api.delete(`/admin/insumos/${id}`);
}

export interface FichaTecnicaItem {
  id: number;
  produto_id: number;
  insumo_id: number;
  insumo: Insumo;
  quantidade: number;
}

export interface FichaTecnica {
  itens: FichaTecnicaItem[];
  cmv: number;
  preco: number;
  margem: number;
}

export interface FichaTecnicaItemInput {
  insumo_id: number;
  quantidade: number;
}

export async function buscarFichaTecnica(produtoId: number): Promise<FichaTecnica> {
  const { data } = await api.get<FichaTecnica>(`/admin/produtos/${produtoId}/ficha-tecnica`);
  return data;
}

export async function salvarFichaTecnica(produtoId: number, itens: FichaTecnicaItemInput[]): Promise<FichaTecnica> {
  const { data } = await api.put<FichaTecnica>(`/admin/produtos/${produtoId}/ficha-tecnica`, { itens });
  return data;
}

// Importação de insumo via XML de NF-e (Fase 9.2) — exclusivo do plano
// Scale. O XML chega como texto puro (lido no navegador via
// File.text()), sem upload multipart.
export interface ItemNFeImportado {
  nome: string;
  unidade: string;
  quantidade: number;
  valor_unitario: number;
  insumo_sugerido: number | null;
}

export interface PreviewNFe {
  numero_nota: string;
  fornecedor: string;
  itens: ItemNFeImportado[];
}

export async function previewImportacaoNFe(xml: string): Promise<PreviewNFe> {
  const { data } = await api.post<PreviewNFe>('/admin/insumos/importar-nfe/preview', { xml });
  return data;
}

export interface ConfirmarItemNFeInput {
  acao: 'vincular' | 'criar' | 'ignorar';
  insumo_id?: number | null;
  nome?: string;
  unidade_compra?: string;
  unidade_uso?: string;
  fator_conversao?: number;
  quantidade: number;
  valor_unitario: number;
}

export async function confirmarImportacaoNFe(numeroNota: string, itens: ConfirmarItemNFeInput[]): Promise<Insumo[]> {
  const { data } = await api.post<Insumo[]>('/admin/insumos/importar-nfe/confirmar', { numero_nota: numeroNota, itens });
  return data;
}