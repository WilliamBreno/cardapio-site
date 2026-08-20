# Componentes de input sofisticados + modo noturno no admin

> Combinado numa conversa com o Claude (chat) em 20/08/2026. Referência visual: mockup HTML puro
> `drenux-vitrine-botoes.html` (fora do stack real, sem Tailwind/shadcn) — usado só pra reproduzir
> a INTERAÇÃO e o visual, não pra copiar código literal.

## Contexto

O admin do Drenux hoje usa `<input>`/`<select>`/`<textarea>` HTML nativos estilizados manualmente
em cada tela, via um wrapper simples (`Campo.tsx`, label fixo acima + children). Não existe nenhum
componente de formulário no padrão shadcn/Base UI já usado pelo projeto (`button.tsx`, `dialog.tsx`,
`slider.tsx`, `badge.tsx`, `accordion.tsx`, `card.tsx`) — esses 13 componentes novos são os
primeiros do "sistema shadcn" a cobrir formulário de verdade.

O projeto já tem `@base-ui/react` 1.6 instalado, com primitivos prontos pra praticamente tudo que
esse trabalho precisa: `switch`, `number-field`, `field`, `toggle-group`, `input`. A decisão de
design central aqui é: **usar esses primitivos como base semântica (acessibilidade de graça) e só
aplicar visual/interação por cima**, em vez de reconstruir do zero replicando o CSS do mockup.

Junto, o admin ganha modo noturno — só o admin, não o cardápio público (que já tem 16 temas de cor
próprios, fora de escopo aqui).

## Escopo

**Dentro**: os 13 componentes A–M do mockup (preço, stepper, label flutuante, busca, toggle,
segmentado, validação inline, textarea com contador, upload dropzone, data, tema compacto, CEP
autocomplete, tags/pills), aplicados cada um a pelo menos um campo real do sistema. Modo noturno no
admin (toggle, persistência, paleta escura, detecção de `prefers-color-scheme`).

**Fora**: componente N (código OTP) — não existe nenhum fluxo de verificação de telefone/e-mail no
sistema hoje (confirmado por busca no código), então não há onde aplicar; fica de fora até esse
fluxo existir. Os 20 efeitos de botão (hover/clique) do mesmo mockup — o pedido original foi
especificamente sobre inputs, não sobre botões. Migração do resto do admin pros tokens shadcn
`--background`/`--foreground` (só ativados, nunca usados hoje) — decisão explícita de não fazer
(ver seção Modo Noturno).

## Arquitetura da biblioteca de inputs

**Local**: `frontend/src/components/ui/` (mesmo diretório dos componentes shadcn existentes).

**Padrão** (replica exatamente `button.tsx`/`dialog.tsx`/`slider.tsx`): primitivo Base UI
(`@base-ui/react/switch`, `/field`, `/number-field`, `/toggle-group`, `/input`) por baixo, `cva`
pra variantes, `cn()` (clsx + tailwind-merge) pra composição de classe, `data-slot` em cada peça.

**Cor**: só classes da paleta de produto (`bg-fundo`, `text-tinta`, `border-tinta/20`,
`text-acento` etc.) — nunca cor fixa (hex) direto no componente. É isso que faz o modo escuro
funcionar automaticamente em todos os 13 componentes sem lógica própria de tema em cada um.

**Fonte**: reaproveita os papéis que o mockup já definia (`--display`/`--mono`) com as classes que
já existem — `font-display` (Anton) pra números/preço em destaque (componente A), `font-carimbo`
(IBM Plex Mono) pra valores monetários e o stepper (componente B). Nenhuma fonte nova importada.

**Arquivos novos** (13, um por letra em escopo):
`input.tsx`, `field.tsx` (wrapper de label + descrição + erro, base de vários outros),
`input-price.tsx`, `stepper.tsx`, `input-floating.tsx`, `input-search.tsx`, `switch.tsx`,
`segmented.tsx`, `textarea.tsx`, `dropzone.tsx`, `input-date.tsx`, `theme-picker.tsx`,
`tag-pill.tsx`.

G (validação inline) e L (CEP) não ganham arquivo próprio — reaproveitam `input.tsx`/`field.tsx`
com um prop `status: 'default' | 'success' | 'error'`.

## Modo noturno

**Store**: `frontend/src/store/temaAdminStore.ts` (Zustand + `persist`, mesmo padrão de
`authStore.ts`). Guarda `preferencia: 'claro' | 'escuro'` — **binário**, não três estados. A
detecção de `prefers-color-scheme` só decide o valor inicial na primeira visita (antes de
qualquer escolha manual do dono); depois de um clique no toggle, a escolha manual fica salva e
nunca mais é sobrescrita pela preferência do sistema.

**Escopo**: um wrapper em `Dashboard.tsx` (layout do admin) aplica `className="dark"`
condicionalmente em volta de todo o conteúdo — não toca a `<html>`, não vaza pro cardápio público
(árvore de componentes totalmente separada, nunca renderizada dentro de `Dashboard.tsx`).

**Toggle**: ícone sol/lua (`lucide-react`) no cabeçalho do admin, entre o nome da loja e o botão
"Sair" (`Dashboard.tsx`, dentro do `<header>` já existente).

**Decisão de arquitetura — por que reaproveitar as variáveis existentes em vez de migrar pros
tokens shadcn**: o CSS já tem um bloco `.dark` completo com tokens semânticos shadcn
(`--background`, `--foreground` etc., em oklch) gerado pelo CLI, mas **nunca usado** — todo o
código de negócio (admin e cardápio público) usa a camada de produto (`--color-fundo`,
`--color-tinta` etc.). Migrar pra ativar o bloco shadcn exigiria editar dezenas de arquivos do
admin pra trocar `bg-fundo`→`bg-background` etc. — expande o escopo bem além de "13 componentes +
toggle". A abordagem escolhida redefine a **camada de produto** dentro de `.dark`, então toda tela
que já usa `bg-fundo`/`text-tinta` ganha modo escuro automaticamente, sem tocar em nenhuma tela
existente.

**Paleta escura** — redefine as mesmas variáveis já usadas em todo o admin, dentro de `.dark`:

| Variável | Papel | Claro (hoje) | Escuro (novo) |
|---|---|---|---|
| `--color-fundo` | fundo da página | creme claro | marrom quase preto |
| `--color-superficie` | cards/painéis | creme mais claro | marrom escuro, um degrau acima do fundo |
| `--color-tinta` | texto principal | quase preto | quase branco/creme |
| `--color-tinta-suave` | texto secundário | marrom acinzentado | cinza quente claro |
| `--color-douro` | dourado de destaque | dourado | mesmo tom, ajustado só se o contraste pedir |
| `--color-acento` | terracota, cor de ação | terracota | **idêntico** — sem mudar |

Valores RGB exatos definidos na implementação, calibrados por contraste (WCAG AA mínimo texto
principal sobre fundo) — a tabela acima fixa a intenção, não os números finais.

## Os 13 componentes (A–M)

| # | Componente | Primitivo Base UI | Onde aplica de verdade | Nota de interação |
|---|---|---|---|---|
| A | `input-price.tsx` | `input` | Preço de produto, preço de combo, valor mínimo de cupom (3 lugares) | Prefixo "R$" fixo dentro do campo, `font-carimbo` no número |
| B | `stepper.tsx` | `number-field` | Quantidade no carrinho do cliente (produto e combo) | Só -/+ (sem digitação livre, igual hoje) — visual pill com botões maiores |
| C | `input-floating.tsx` | `field` | Nome do produto (`ProdutoFormFields.tsx`) | Label sobe/encolhe ao focar ou preencher |
| D | `input-search.tsx` | `input` | Busca de produtos no admin — **campo novo**, `Produtos.tsx`, filtra a lista por nome | Botão "x" só aparece com texto digitado |
| E | `switch.tsx` | `switch` | Campo "Disponível" do produto (troca o checkbox nativo) | — |
| F | `segmented.tsx` | `toggle-group` | Modo de entrega no carrinho público (Entrega/Retirada/Guardar) | Genérico pra N opções — já suporta 3 hoje, pronto pra "Mesa" sem mudar o componente |
| G | prop `status` em `input.tsx`/`field.tsx` | `field` | Telefone/WhatsApp e CEP (Configurações) | Telefone: valida formato (DDI+DDD+número). CEP: usa o resultado real do ViaCEP (`erroCep`/sucesso já existente) |
| H | `textarea.tsx` | nativo (Base UI não tem textarea) | Descrição do produto | Contador de caracteres restantes |
| I | `dropzone.tsx` | custom | Foto do produto (troca o botão de texto atual) | Arrastar ou clicar em qualquer parte da área |
| J | `input-date.tsx` | nativo `input type=date` | Validade do cupom | Clique em qualquer parte do campo chama `input.showPicker()`, não só no ícone |
| K | `theme-picker.tsx` | custom, semântica de radio | Seletor de tema em Configurações (substitui o grid grande atual) | Bolinhas de cor compactas, mesmos 16 temas |
| L | reaproveita `input.tsx` + `status` | `field` | CEP do endereço da loja (mesmo campo de G) | Junta validação inline com o preenchimento automático já existente (ViaCEP, não Nominatim — correção de premissa do pedido original) |
| M | `tag-pill.tsx` | custom, botão+X | Seleção de Categoria/Subcategoria/Grupo no formulário de produto | Cada nível escolhido vira pill removível; X reseta aquele nível — não muda o modelo de dado |

G e L compartilham a mesma implementação — não são dois componentes separados.

## Fora de escopo, decidido conscientemente

- **Componente N (OTP)**: sem fluxo de verificação de telefone/e-mail no sistema hoje — confirmado
  por busca no código antes de decidir. Fica de fora até esse fluxo existir.
- **Os 20 efeitos de botão** (hover/clique) do mesmo mockup: fora do pedido original, que foi
  especificamente sobre inputs.
- **Migração do resto do admin pros tokens shadcn**: decisão explícita de reaproveitar a camada de
  produto em vez disso (ver "Modo noturno" acima).

## Validação e testes

Sem suíte de teste de componente React no projeto hoje (só o backend Go tem testes automatizados)
— mantida essa convenção, sem introduzir suíte nova só pra isso. Validação manual no navegador,
mesmo padrão do resto do roadmap:

- Cada componente testado nos dois modos (claro/escuro) no lugar real onde foi aplicado.
- `npx tsc -b` e `npm run build` limpos antes de considerar qualquer componente pronto.
- Screenshot de antes/depois em pelo menos um componente por categoria de interação (label
  flutuante, validação inline, dropzone, tema compacto) pra confirmar visualmente.
