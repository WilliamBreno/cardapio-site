// Dados dos planos (Start/Pro/Scale), cálculo de custo e o tema visual da
// página comercial de planos — extraído de MeuPlano.tsx/Planos.tsx pra
// reaproveitar também no alerta proativo de Início.tsx e manter a mesma
// cara em Meu Plano, sem duplicar números nem cores em vários lugares.
import type { CSSProperties } from 'react';

export interface PlanoInfo {
  id: 'start' | 'pro' | 'scale';
  nome: string;
  mensal: number;
  taxa: number;
  desc: string;
}

export const PLANOS: PlanoInfo[] = [
  { id: 'start', nome: 'Start', mensal: 0, taxa: 0.065, desc: 'Sem risco, comece de graça' },
  { id: 'pro', nome: 'Pro', mensal: 129, taxa: 0.04, desc: 'Pra loja em crescimento' },
  { id: 'scale', nome: 'Scale', mensal: 349, taxa: 0.015, desc: 'Volume alto, custo mínimo' },
];

export function custoPlano(plano: PlanoInfo, faturamento: number): number {
  return plano.mensal + plano.taxa * faturamento;
}

// planoMaisBarato devolve, entre os três planos, o de menor custo total
// pro faturamento informado (em caso de empate, o primeiro da lista).
export function planoMaisBarato(faturamento: number): PlanoInfo {
  return PLANOS.reduce((menor, p) =>
    custoPlano(p, faturamento) < custoPlano(menor, faturamento) ? p : menor
  );
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
