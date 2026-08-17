// Tipos espelhando os modelos do backend (internal/domain). Mantém o
// front em sincronia com o que a API realmente devolve — se um campo
// mudar lá, é só atualizar aqui também.

export interface Categoria {
  id: number;
  loja_id: number;
  nome: string;
  created_at: string;
  updated_at: string;
}

// Subcategoria e GrupoCor são exclusivos do segmento "mercadoria" — drill-
// down opcional Categoria → Subcategoria → Grupo de Cor pra organizar
// catálogo de varejo (ex: tamanho → cor). Grupo de Cor é sempre aninhado
// numa Subcategoria, nunca solto direto na Categoria. Não confundir com
// VariacaoProduto (aditiva sobre o preço, recurso de cardápio).
export interface Subcategoria {
  id: number;
  categoria_id: number;
  nome: string;
  ordem: number;
  created_at: string;
  updated_at: string;
}

export interface GrupoCor {
  id: number;
  subcategoria_id: number;
  nome: string;
  ordem: number;
  created_at: string;
  updated_at: string;
}

// ModoPrecoVariacao: "aditivo" soma preco_adicional ao preço base do
// produto (comportamento original); "absoluto" faz preco_adicional ser o
// preço final da variação (ignora o preço base) — pensado pro segmento
// "mercadoria", onde cada variação pode ter preço e fotos próprios.
export type ModoPrecoVariacao = 'aditivo' | 'absoluto';

export interface VariacaoProduto {
  id: number;
  produto_id: number;
  nome: string;
  preco_adicional: number;
  disponivel: boolean;
  mostrar_valor_adicional: boolean;
  modo_preco: ModoPrecoVariacao;
  fotos?: FotoVariacao[];
  estoque_atual: number | null;
  estoque_alerta: number | null;
  ordem: number;
}

export interface FotoProduto {
  id: number;
  produto_id: number;
  url: string;
  ordem: number;
}

export interface FotoVariacao {
  id: number;
  variacao_id: number;
  url: string;
  ordem: number;
}

// Tipo do produto: "alimenticio" (perecível, nunca pode ser guardado) ou
// "mercadoria" (roupas, artesanato etc. — único tipo elegível pro fluxo
// de "guardar e entregar depois", já que reter comida por tempo
// indeterminado é um risco de segurança alimentar).
export type TipoProduto = 'alimenticio' | 'mercadoria';

export interface Produto {
  id: number;
  loja_id: number;
  categoria_id: number;
  categoria?: Categoria;
  subcategoria_id: number | null;
  subcategoria?: Subcategoria;
  grupo_cor_id: number | null;
  grupo_cor?: GrupoCor;
  nome: string;
  descricao: string;
  preco: number;
  foto_url: string;
  fotos?: FotoProduto[];
  disponivel: boolean;
  estoque_atual: number | null;
  estoque_alerta: number | null;
  variacoes?: VariacaoProduto[];
  tipo_produto: TipoProduto;
  peso_gramas: number | null;
  created_at: string;
  updated_at: string;
}

// Combo/Kit (Fase 6) — pacote fixo com preço definido diretamente pelo
// lojista (não calculado por desconto sobre a soma dos itens). O cliente
// escolhe a variação de cada componente ao montar no carrinho, igual
// comprando avulso.
export interface ComboItem {
  id: number;
  combo_id: number;
  produto_id: number;
  produto?: Produto;
  quantidade: number;
}

export interface Combo {
  id: number;
  loja_id: number;
  nome: string;
  descricao: string;
  foto_url: string;
  preco: number;
  disponivel: boolean;
  ordem: number;
  itens: ComboItem[];
  created_at: string;
  updated_at: string;
}

// Cópia dos itens do combo no momento da compra — guardada no pedido pra
// não depender do combo (que pode mudar ou ser excluído depois).
export interface PedidoComboItem {
  id: number;
  pedido_combo_id: number;
  produto_id: number;
  produto_nome: string;
  variacao_id: number | null;
  variacao_nome: string;
  quantidade: number;
}

export interface PedidoCombo {
  id: number;
  pedido_id: number;
  combo_id: number;
  nome: string;
  foto_url: string;
  preco: number;
  quantidade: number;
  itens: PedidoComboItem[];
}

// Sugestão Inteligente (Fase 6) — vínculo manual produto origem → produto
// sugerido, configurado pelo lojista (não é algoritmo automático).
export type TipoDesconto = 'percentual' | 'fixo';

export interface SugestaoProduto {
  id: number;
  loja_id: number;
  produto_origem_id: number;
  produto_sugerido_id: number;
  produto_sugerido?: Produto;
  tipo_desconto: TipoDesconto | null;
  valor_desconto: number | null;
  created_at: string;
  // Sem a Sugestão Inteligente contratada, só o vínculo mais antigo da
  // loja fica ativo (o "gostinho grátis") — os demais continuam salvos,
  // só ficam ocultos até uma nova assinatura.
  ativo: boolean;
}

// Versão pronta pra exibir na revisão do carrinho — já resolvida (uma por
// categoria, sem produto repetido) e com o preço com desconto calculado.
export interface SugestaoCarrinhoItem {
  sugestao_id: number;
  produto_id: number;
  nome: string;
  foto_url: string;
  preco: number;
  preco_com_desconto: number;
  tipo_desconto?: TipoDesconto;
  valor_desconto?: number;
}

export interface DashboardData {
  total_semana: number;
  total_mes: number;
  pedidos_semana: number;
  receita_7_dias: { data: string; total: number }[];
  receita_4_semanas: { semana: string; total: number }[];
  top_produtos: { nome: string; quantidade: number }[];
}

// Tipo de cálculo da taxa de entrega:
// "fixa"      → valor fixo definido pelo dono
// "combinado" → cliente informa endereço, dono combina fora do sistema
// "por_km"    → calculado automaticamente: taxa_entrega_base + (distância_km * taxa_entrega_por_km)
export type TaxaEntregaTipo = 'fixa' | 'combinado' | 'por_km';

export interface Loja {
  id: number;
  usuario_id: number;
  nome: string;
  slug: string;
  whatsapp_numero: string;
  logo_url: string;
  modo_pedido: 'imediato' | 'agendado';
  antecedencia_minima_horas: number;
  horario_abertura: string;
  horario_fechamento: string;
  margem_fechamento_minutos: number;
  pausado: boolean;
  mensagem_pausa: string;
  aceita_retirada: boolean;
  aceita_entrega: boolean;
  aceita_guardar_entregar: boolean;
  segmento_principal: TipoProduto;
  taxa_entrega_tipo: TaxaEntregaTipo;
  taxa_entrega_valor: number;
  taxa_entrega_base: number;
  taxa_entrega_por_km: number;
  endereco: string;
  latitude: number;
  longitude: number;
  cidade: string;
  estado: string;
  valor_minimo_pedido: number;
  tema: string;
  created_at: string;
  updated_at: string;
  plano: string; // "start" | "basic" | "pro" | "scale"
  plano_agendado: string | null;
  // Limite de pedidos/mês do plano Start (Fase 7.3) — sempre presentes,
  // mesmo pra quem não é Start (o painel decide se mostra com base no
  // plano). pedidos_mes_atual conta pelo mês-calendário corrente.
  pedidos_mes_atual: number;
  limite_start_bloqueado: boolean;
  // Sugestão Inteligente (Fase 6) — recurso pago à parte. "ativa" liga/
  // desliga a exibição no carrinho do cliente livremente, contratada ou
  // não (sem contratar, a loja tem direito a 1 vínculo grátis que
  // aparece de verdade se "ativa" estiver true — ver SugestaoProduto.ativo).
  sugestao_inteligente_contratada: boolean;
  sugestao_inteligente_contratada_em: string | null;
  sugestao_inteligente_ativa: boolean;
  // segmento_principal é definitivo desde o cadastro — só true pra um
  // punhado de contas de teste/administração (ver LojaService no
  // backend). Pra qualquer outra loja, o seletor de segmento em
  // Configurações fica travado.
  pode_editar_segmento: boolean;
}

export interface CardapioPublico {
  loja: {
    nome: string;
    slug: string;
    logo_url: string;
    modo_pedido: 'imediato' | 'agendado';
    antecedencia_minima_horas: number;
    horario_abertura: string;
    horario_fechamento: string;
    margem_fechamento_minutos: number;
    pausado: boolean;
    mensagem_pausa: string;
    aceita_retirada: boolean;
    aceita_entrega: boolean;
    aceita_guardar_entregar: boolean;
    segmento_principal: TipoProduto;
    taxa_entrega_tipo: TaxaEntregaTipo;
    taxa_entrega_valor: number;
    taxa_entrega_base: number;
    taxa_entrega_por_km: number;
    valor_minimo_pedido: number;
    tema: string;
    sugestao_inteligente_ativa: boolean;
    // mostrar_selo_drenux: marca "Feito com Drenux" — visível só no Start,
    // removível a partir do Basic (já computado no backend).
    mostrar_selo_drenux: boolean;
    // avisos_whatsapp_cliente: false no Start — cliente não recebe
    // confirmação de pagamento nem aviso de saída pra entrega via
    // WhatsApp nesse plano (já computado no backend).
    avisos_whatsapp_cliente: boolean;
  };
  categorias: Categoria[];
  subcategorias: Subcategoria[];
  grupos_cor: GrupoCor[];
  produtos: Produto[];
  combos: Combo[];
}

export type StatusPedido = 'aguardando_pagamento' | 'pago' | 'cancelado';

export interface ItemPedido {
  id: number;
  pedido_id: number;
  produto_id: number;
  produto_nome: string;
  variacao_id: number | null;
  variacao_nome: string;
  quantidade: number;
  preco_unit: number;
  tipo_produto: TipoProduto;
  peso_gramas: number;
  solicitacao_entrega_id: number | null;
}

// ItemGuardado é um ItemPedido comprado no modo "guardar" que ainda não
// foi reivindicado por nenhuma entrega — com a data da compra original.
export interface ItemGuardado extends ItemPedido {
  guardado_desde: string;
}

export type StatusSolicitacao = 'aguardando_pagamento' | 'paga' | 'cancelada';

export interface SolicitacaoEntrega {
  id: number;
  loja_id: number;
  cliente_nome: string;
  cliente_telefone: string;
  endereco_entrega: string;
  latitude: number;
  longitude: number;
  distancia_km: number;
  tipo_calculo: string;
  peso_total_gramas: number;
  valor_frete: number;
  peso_pendente: boolean;
  status: StatusSolicitacao;
  itens: ItemPedido[];
  status_entrega: string;
  entregador_latitude: number;
  entregador_longitude: number;
  entregador_atualizado_em: string | null;
  created_at: string;
  updated_at: string;
}

export interface Cupom {
  id: number;
  loja_id: number;
  codigo: string;
  tipo: 'percentual' | 'fixo';
  valor: number;
  ativo: boolean;
  uso_maximo: number | null;
  uso_atual: number;
  validade: string | null;
  valor_minimo_pedido: number;
  created_at: string;
  updated_at: string;
}

export interface Pedido {
  id: number;
  loja_id: number;
  cliente_nome: string;
  cliente_telefone: string;
  data_retirada: string;
  status: StatusPedido;
  total: number;
  modo_entrega: 'retirada' | 'entrega' | 'guardar';
  endereco_entrega: string;
  cupom_codigo: string;
  desconto: number;
  peso_pendente: boolean;
  itens: ItemPedido[];
  combos?: PedidoCombo[];
  created_at: string;
  updated_at: string;
  status_entrega: string;
  entregador_latitude: number;
  entregador_longitude: number;
  entregador_atualizado_em: string | null;
}

// Carrinho — estado só do front, nunca enviado direto pra API. Na hora
// de criar o pedido, viram { produto_id, variacao_id, quantidade } no payload.
export interface ItemCarrinho {
  produto: Produto;
  variacao?: VariacaoProduto; // undefined = produto sem variação
  quantidade: number;
  // Preenchidos só quando o item entrou no carrinho via Sugestão
  // Inteligente (Fase 6) — o preço com desconto é calculado no backend
  // (MontarSugestoesCarrinho) e só guardado aqui pra exibir/somar no
  // carrinho; o backend recalcula e revalida tudo de novo no checkout.
  sugestaoProdutoId?: number;
  precoComDesconto?: number;
}

// Seleção de variação de um componente específico do combo, feita pelo
// cliente ao montar o combo no carrinho (igual escolher variação de um
// produto avulso).
export interface SelecaoComboItem {
  comboItemId: number;
  produtoId: number;
  variacao?: VariacaoProduto;
}

export interface ItemCarrinhoCombo {
  combo: Combo;
  quantidade: number;
  selecoes: SelecaoComboItem[];
}