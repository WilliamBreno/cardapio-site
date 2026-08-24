// withOpacity dá suporte a modificadores de opacidade (ex: ring-ring/50,
// bg-destructive/10) em cima de uma CSS variable que já é uma função de cor
// completa (oklch(...), com ou sem alpha embutido) — usa CSS relative color
// syntax pra extrair só L/C/H da variável e aplicar uma alpha nova por
// cima, o que o formato `rgb(var(...) / <alpha-value>)` (usado na paleta
// custom abaixo) não suporta quando a variável já vem com oklch() completo.
function withOpacity(varName) {
  return `oklch(from var(${varName}) l c h / <alpha-value>)`
}

/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{js,ts,jsx,tsx}'],
  // darkMode: 'class' (achado em 24/08/2026, corrige texto preto ilegível
  // no modo noturno do admin): sem essa opção, o Tailwind v3 usa a
  // estratégia padrão 'media' — as classes dark: (number-ticker.tsx,
  // MeuPlano.tsx, badge.tsx, button.tsx, QRCodeCardapio.tsx) respondiam
  // só à preferência do SISTEMA OPERACIONAL, não ao toggle manual do
  // Dashboard.tsx (que só adiciona a classe `.dark` no wrapper). Um
  // computador em modo claro do SO com o toggle do admin em "escuro"
  // deixava esses `dark:text-white` etc. inertes, sobrando `text-black`
  // puro sobre fundo escuro. 'class' faz os `dark:` responderem à mesma
  // classe `.dark` que já controla as variáveis de cor customizadas.
  darkMode: 'class',
  theme: {
    extend: {
      colors: {
        // Cores usam CSS variables pra suportar troca de tema em runtime.
        // O formato 'rgb(var(...) / <alpha-value>)' é o jeito certo no
        // Tailwind v3 de ter variáveis dinâmicas E opacity modifiers
        // funcionando (ex: bg-tinta/10, text-acento/70).
        fundo:        'rgb(var(--color-fundo)       / <alpha-value>)',
        superficie:   'rgb(var(--color-superficie)  / <alpha-value>)',
        tinta:        'rgb(var(--color-tinta)        / <alpha-value>)',
        'tinta-suave':'rgb(var(--color-tinta-suave) / <alpha-value>)',
        acento:       'rgb(var(--color-acento)      / <alpha-value>)',
        douro:        'rgb(var(--color-douro)        / <alpha-value>)',

        // Tokens semânticos do shadcn (usados por components/ui/*, ex:
        // button.tsx, card.tsx, dialog.tsx) — o CLI do shadcn gera esses
        // valores em formato Tailwind v4 (`@theme inline` em index.css),
        // mas este projeto está no Tailwind v3, que não lê `@theme`. Sem
        // esse mapeamento aqui, classes como `border-border`/`bg-primary`
        // não existem e o build inteiro quebra (achado ao testar a Fase 3
        // — não é um problema novo, já afetava MeuPlano.tsx e Planos.tsx).
        border:              withOpacity('--border'),
        input:               withOpacity('--input'),
        ring:                withOpacity('--ring'),
        background:          withOpacity('--background'),
        foreground:          withOpacity('--foreground'),
        primary:             { DEFAULT: withOpacity('--primary'), foreground: withOpacity('--primary-foreground') },
        secondary:           { DEFAULT: withOpacity('--secondary'), foreground: withOpacity('--secondary-foreground') },
        destructive:         withOpacity('--destructive'),
        muted:               { DEFAULT: withOpacity('--muted'), foreground: withOpacity('--muted-foreground') },
        accent:              { DEFAULT: withOpacity('--accent'), foreground: withOpacity('--accent-foreground') },
        popover:             { DEFAULT: withOpacity('--popover'), foreground: withOpacity('--popover-foreground') },
        card:                { DEFAULT: withOpacity('--card'), foreground: withOpacity('--card-foreground') },
        sidebar: {
          DEFAULT:    withOpacity('--sidebar'),
          foreground: withOpacity('--sidebar-foreground'),
          primary:    withOpacity('--sidebar-primary'),
          'primary-foreground': withOpacity('--sidebar-primary-foreground'),
          accent:     withOpacity('--sidebar-accent'),
          'accent-foreground': withOpacity('--sidebar-accent-foreground'),
          border:     withOpacity('--sidebar-border'),
          ring:       withOpacity('--sidebar-ring'),
        },
        'chart-1': withOpacity('--chart-1'),
        'chart-2': withOpacity('--chart-2'),
        'chart-3': withOpacity('--chart-3'),
        'chart-4': withOpacity('--chart-4'),
        'chart-5': withOpacity('--chart-5'),
      },
      fontFamily: {
        display: ['"Anton"', 'sans-serif'],
        corpo: ['"Inter"', 'sans-serif'],
        carimbo: ['"IBM Plex Mono"', 'monospace'],
      },
    },
  },
  plugins: [],
}