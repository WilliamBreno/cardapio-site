// Dados dos planos (Start/Basic/Pro/Scale), cálculo de custo e o tema
// visual da página comercial de planos — extraído de
// MeuPlano.tsx/Planos.tsx pra reaproveitar também no alerta proativo de
// Início.tsx e manter a mesma cara em Meu Plano, sem duplicar números nem
// cores em vários lugares.
import type { CSSProperties } from 'react';

export interface FaixaComissao {
  /** Teto da faixa em R$ de GMV mensal; 0 = sem teto (última faixa). */
  ate: number;
  /** Fração aplicada sobre a fatia do GMV que cai dentro dessa faixa (0.024 = 2,4%). */
  taxa: number;
}

export interface PlanoInfo {
  id: 'start' | 'basic' | 'pro' | 'scale';
  nome: string;
  mensal: number;
  /** Comissão escalonada por faixa de GMV mensal. Vazio = sem split de pagamento (Start). */
  faixas: FaixaComissao[];
  desc: string;
}

// Mesmas faixas de backend/internal/service/mercadopago_service.go
// (faixasComissaoPorPlano) — qualquer mudança aqui precisa espelhar lá,
// senão a loja vê um número e é cobrada outro.
export const PLANOS: PlanoInfo[] = [
  { id: 'start', nome: 'Start', mensal: 0, faixas: [], desc: 'Sem risco, comece de graça' },
  {
    id: 'basic',
    nome: 'Basic',
    mensal: 0,
    faixas: [
      { ate: 5000, taxa: 0.024 },
      { ate: 20000, taxa: 0.015 },
      { ate: 0, taxa: 0.013 },
    ],
    desc: 'Pagamento automático, sem mensalidade',
  },
  {
    id: 'pro',
    nome: 'Pro',
    mensal: 99,
    faixas: [
      { ate: 5000, taxa: 0.018 },
      { ate: 20000, taxa: 0.012 },
      { ate: 0, taxa: 0.0105 },
    ],
    desc: 'Rastreamento em tempo real + relatório de estoque',
  },
  {
    id: 'scale',
    nome: 'Scale',
    mensal: 349,
    faixas: [{ ate: 0, taxa: 0.011 }],
    desc: 'Volume alto, custo mínimo + controle de estoque completo',
  },
];

export const NOME_PLANO: Record<string, string> = Object.fromEntries(PLANOS.map((p) => [p.id, p.nome]));

// RecursoPlano é uma linha da matriz de funcionalidades por plano (Fase
// 7.4) — cada valor é o texto exibido pra aquele plano; "—" marca que o
// plano NÃO tem esse recurso (usado pra decidir o ícone de check/xis).
// IMPORTANTE: só entra aqui recurso que já existe de verdade e é
// gateado no código — não é a matriz aspiracional completa de
// docs/drenux-planos-comissoes-definido.md § 5 (essa tem linhas como
// "Domínio próprio", "Relatórios avançados", "Múltiplos usuários",
// "Automação", "Multi-loja" e cotas de Sugestão Inteligente por plano
// que ainda não foram construídas — anunciar isso numa página pública
// antes de existir seria propaganda enganosa).
export interface RecursoPlano {
  label: string;
  valores: Record<PlanoInfo['id'], string>;
}

export const RECURSOS_POR_PLANO: RecursoPlano[] = [
  {
    label: 'Cardápio digital (link + QR code)',
    valores: { start: '✓', basic: '✓', pro: '✓', scale: '✓' },
  },
  {
    label: 'Aviso de novo pedido pro dono',
    valores: { start: 'por email', basic: 'WhatsApp', pro: 'WhatsApp', scale: 'WhatsApp' },
  },
  {
    label: 'Produtos cadastrados',
    valores: { start: 'até 30', basic: 'ilimitado', pro: 'ilimitado', scale: 'ilimitado' },
  },
  {
    label: 'Pedidos por mês',
    valores: { start: 'até 30 (renova todo mês)', basic: 'ilimitado', pro: 'ilimitado', scale: 'ilimitado' },
  },
  {
    label: 'Avisos automáticos de status pro cliente (WhatsApp)',
    valores: { start: '—', basic: '✓', pro: '✓', scale: '✓' },
  },
  {
    label: 'Pagamento automático (Pix/cartão via split)',
    valores: { start: '—', basic: '✓', pro: '✓', scale: '✓' },
  },
  {
    label: 'Rastreamento de entrega em tempo real (mapa)',
    valores: { start: '—', basic: '—', pro: '✓', scale: '✓' },
  },
  {
    label: 'Marca "Feito com Drenux"',
    valores: { start: 'visível', basic: 'removível', pro: 'removível', scale: 'removível' },
  },
  {
    label: 'Controle de estoque',
    valores: { start: '—', basic: '—', pro: 'relatório', scale: 'completo (reposição + histórico)' },
  },
];

// comissaoEscalonada aplica cada faixa só sobre a fatia do faturamento que
// cai dentro dela (igual uma tabela progressiva) — mesma lógica de
// calcularComissaoEscalonada no backend.
function comissaoEscalonada(faixas: FaixaComissao[], faturamento: number): number {
  let comissao = 0;
  let piso = 0;
  let restante = faturamento;
  for (const faixa of faixas) {
    const teto = faixa.ate === 0 ? Infinity : faixa.ate;
    const disponivel = Math.max(0, teto - piso);
    const fatia = Math.min(restante, disponivel);
    comissao += fatia * faixa.taxa;
    restante -= fatia;
    piso = teto;
    if (restante <= 0) break;
  }
  return comissao;
}

export function custoPlano(plano: PlanoInfo, faturamento: number): number {
  return plano.mensal + comissaoEscalonada(plano.faixas, faturamento);
}

// taxaEfetiva devolve o percentual médio (comissão ÷ faturamento) — só pra
// exibição. Sem faturamento simulado ainda, usa a taxa da primeira faixa
// como referência do "a partir de".
export function taxaEfetivaPlano(plano: PlanoInfo, faturamento: number): number {
  if (plano.faixas.length === 0) return 0;
  if (faturamento <= 0) return plano.faixas[0].taxa;
  return comissaoEscalonada(plano.faixas, faturamento) / faturamento;
}

// planoMaisBarato compara só planos com split de pagamento ativo (Start
// fica de fora — sem comissão nenhuma, mas também sem processar
// pagamento, não é uma alternativa comparável em "custo de taxa").
export function planoMaisBarato(faturamento: number): PlanoInfo {
  const comparaveis = PLANOS.filter((p) => p.faixas.length > 0);
  return comparaveis.reduce((menor, p) =>
    custoPlano(p, faturamento) < custoPlano(menor, faturamento) ? p : menor
  );
}

// FATURAMENTO_MAX_SIMULADOR precisa cobrir o cruzamento real entre Basic e
// Pro (~R$29.600 com as taxas atuais — confirmado numericamente, não é um
// chute) — com o simulador limitado a R$20.000 como estava antes, o cliente
// nunca via esse cruzamento por mais que arrastasse o slider até o fim.
export const FATURAMENTO_MAX_SIMULADOR = 50000;

export interface FaixaRecomendada {
  de: number;
  ate: number;
  planoId: PlanoInfo['id'];
}

// calcularFaixasRecomendadas varre [0, faturamentoMax] e agrupa em faixas
// contíguas de qual plano com split (Basic/Pro/Scale) sai mais barato —
// é o que alimenta o mapa visual embaixo do slider, pra mostrar de cara em
// que faturamento a recomendação muda, sem o cliente ter que arrastar o
// slider centímetro por centímetro pra descobrir sozinho.
export function calcularFaixasRecomendadas(faturamentoMax: number, passo = 50): FaixaRecomendada[] {
  const comparaveis = PLANOS.filter((p) => p.faixas.length > 0);
  const faixas: FaixaRecomendada[] = [];

  for (let f = 0; f <= faturamentoMax; f += passo) {
    const maisBarato = comparaveis.reduce((menor, p) =>
      custoPlano(p, f) < custoPlano(menor, f) ? p : menor
    );
    const atual = faixas[faixas.length - 1];
    if (atual && atual.planoId === maisBarato.id) {
      atual.ate = f;
    } else {
      faixas.push({ de: f, ate: f, planoId: maisBarato.id });
    }
  }

  return faixas;
}

// Sobrescreve só os tokens de cor do shadcn — preto e dourado, puxados da
// marca — dentro de um wrapper com esse `style`. Não mexe em nada do
// sistema de temas do cardápio público (--color-fundo, --color-tinta
// etc.), que usa variáveis completamente diferentes.
export const temaPlanos: CSSProperties = {
  '--background': '#08080a',
  '--foreground': '#f2efe8',
  '--card': '#131318',
  '--card-foreground': '#f2efe8',
  '--popover': '#131318',
  '--popover-foreground': '#f2efe8',
  '--primary': '#d4af6a',
  '--primary-foreground': '#08080a',
  '--secondary': '#1c1c22',
  '--secondary-foreground': '#f2efe8',
  '--muted': '#1c1c22',
  '--muted-foreground': '#8f8b80',
  '--accent': '#1c1c22',
  '--accent-foreground': '#d4af6a',
  '--border': 'rgba(212, 175, 106, 0.18)',
  '--input': 'rgba(212, 175, 106, 0.18)',
  '--ring': '#d4af6a',
} as CSSProperties;

// Fonte serifada usada nos títulos/preços da área de planos (comercial e
// admin) — injeta o @import + a classe utilitária .drx-serif.
export const FONTE_DRX_SERIF_CSS = `
  @import url('https://fonts.googleapis.com/css2?family=Cormorant+Garamond:wght@400;500;600&display=swap');
  .drx-serif { font-family: 'Cormorant Garamond', serif; }
`;
