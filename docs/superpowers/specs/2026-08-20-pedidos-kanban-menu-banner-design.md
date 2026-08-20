# Kanban de Pedidos (drag-and-drop), redesign do menu do admin, banner do cardápio público

> Combinado numa conversa com o Claude (chat) em 20/08/2026, a partir de feedback direto com
> screenshots sobre a Fase 10.2 (Kanban de Pedidos) e a Fase 10.1 (banner do cardápio público).

## Contexto

Três pedidos vieram juntos, cobrindo duas telas diferentes:

1. **Quadro de Pedidos (Kanban)**: hoje só tem um botão "Avançar →" por card, sem
   arrastar-e-soltar (decisão explícita da Fase 10.2: *"sem drag-and-drop de propósito — mais
   frágil de acertar bem e não foi pedido explicitamente"*). Essa decisão está sendo revertida.
   Além disso, o `<main>` do admin tem `max-w-3xl` (768px) — o Kanban de 4 colunas (precisa de
   ~1150px) fica espremido dentro disso e cria scroll horizontal forçado mesmo em tela grande.
2. **Menu de navegação do admin**: barra horizontal simples, só texto + sublinhado no ativo — sem
   ícone, sem visual rico.
3. **Banner do cardápio público** (Fase 10.1): hoje é uma `<img className="h-32 w-full
   object-cover sm:h-48">` colada acima do cabeçalho da loja, cortando a imagem (`object-cover`
   força altura fixa, cortando topo/base de imagens que não batem exatamente com a proporção).

Também um ajuste pequeno, já implementado antes de escrever este spec: removido o filtro "Pagos"
da lista de Pedidos (redundante com os filtros de etapa — se está "a_preparar" já é pago por
definição) e movido "Cancelados" pro fim da lista de filtros, dando ênfase aos pedidos
novos/em andamento primeiro.

## Escopo

**Dentro**: drag-and-drop no Kanban de Pedidos (qualquer direção, qualquer coluna); grid
responsivo pro Kanban (1/2/4 colunas conforme tela); mais densidade de informação por card;
liberar a largura de tela só nessa visão; redesign visual do menu (ícones + pill ativa, mesma
barra horizontal); reposicionar e redesenhar o banner do cardápio público.

**Fora**: sidebar vertical pro menu (decisão explícita — mantém barra horizontal); mudar o
`max-w-3xl` globalmente pras outras telas do admin (só a visão Quadro ganha largura livre).

## Arquitetura geral

**Largura de tela sob demanda**: `frontend/src/store/layoutAdminStore.ts` (Zustand, mesmo padrão
de `authStore.ts`), com `larguraCompleta: boolean`. `Pedidos.tsx` liga isso via `useEffect`
quando `visualizacao === 'quadro'` está ativa, desliga ao sair da tela ou trocar pra Lista.
`Dashboard.tsx` lê o store e só então tira o `max-w-3xl` do `<main>` — cirúrgico, não afeta as
outras ~15 telas do admin.

**Dependência nova**: `@dnd-kit/core` — biblioteca de drag-and-drop mantida ativamente, com
suporte nativo a mouse, toque e teclado (substitui o `react-beautiful-dnd`, hoje sem manutenção
e com problemas de compatibilidade com React 19). Não precisa do `@dnd-kit/sortable` — é
drop em colunas (zonas), não reordenação linear dentro de uma lista só.

## Kanban de Pedidos

**Layout de colunas**: troca `flex overflow-x-auto` (colunas fixas de 288px, scroll horizontal
forçado) por grid responsivo:
```
grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4
```
Mobile: 1 coluna por vez, empilhadas (scroll vertical natural). Tablet: 2 colunas. Desktop: as 4
etapas visíveis ao mesmo tempo, sem scroll.

**Densidade do card**: além do que já existe (nome+#pedido, modo de entrega+total, botões),
adiciona itens do pedido (resumidos), telefone do cliente, horário, e os badges que já existem em
outras telas (cupom aplicado, peso pendente) — mesma informação do card da Lista, replicada aqui.

**Drag-and-drop**: `DndContext` (dnd-kit) envolve o quadro inteiro; cada coluna é uma zona
`useDroppable` (id = etapa); cada card é `useDraggable` (id = pedido). Soltar em **qualquer**
coluna (não só adjacente) chama a mesma mutation que já existe (`atualizarStatusEntrega`, que já
aceita qualquer etapa como destino sem validar ordem) — avança, volta ou pula etapa livremente,
igual um Kanban de verdade. Funciona em mouse, toque e teclado.

**Botão "Avançar →"**: mantido no card como atalho pra quem não quer arrastar (teclado,
acessibilidade, ou clique rápido).

## Menu de navegação do admin

Mantém a barra horizontal (`nav` em `Dashboard.tsx`), sem virar sidebar — mudança só visual:

- Ícone por link (`lucide-react`, já usado no projeto): Início→`Home`, Pedidos→`ClipboardList`,
  Produtos→`Package`, Categorias→`Tags`, Cupons→`Ticket`, Combos/Kits→`Boxes`,
  Sugestão Inteligente→`Sparkles`, Configurações→`Settings`, Meu Plano→`CreditCard`,
  Estoque→`Warehouse`, Insumos→`Beaker`, Guardados→`Archive`.
- Estado ativo: pill preenchida (`bg-acento text-superficie rounded-full`) no lugar do sublinhado
  atual.
- Estado inativo: hover com fundo suave (`hover:bg-tinta/5`) em vez de só mudar cor do texto.
- `overflow-x-auto` continua igual, pra lista de links que já cresce dinamicamente por
  plano/segmento da loja (Estoque, Insumos, Guardados) — lógica de quais links aparecem não muda,
  só o visual de cada item.

## Banner do cardápio público

**Posição**: sai de acima do cabeçalho e vira um card dedicado logo abaixo dele (logo+nome da
loja), antes dos filtros de categoria — cantos arredondados, sombra leve, com respiro ao redor
(não mais edge-to-edge colado nas bordas da tela).

**Exibição**: a imagem aparece inteira (`object-contain`), centralizada, numa altura fixa
consistente; o espaço vazio nas bordas quando a proporção não bate exatamente com o container é
preenchido com uma cópia desfocada (`blur`) da própria imagem como fundo — nunca corta a imagem,
fica visualmente consistente entre lojas com banners de proporções diferentes.

**Onde muda**: `frontend/src/pages/CardapioPublico.tsx`, nos dois lugares onde `banner_url`
aparece hoje (estado loja fechada e estado normal) — os dois recebem o mesmo tratamento.

## Já implementado antes deste spec (não precisa de plano)

Filtros de `Pedidos.tsx`: removido "Pagos" (redundante com os filtros de etapa), "Cancelados"
movido pro fim da lista de pills. `filtrosBase` em `frontend/src/pages/admin/Pedidos.tsx`.
Validado com `npx tsc -b`.

## Validação e testes

Sem suíte de teste automatizado de componente (convenção do projeto) — validação manual no
navegador:

- `npx tsc -b` e `npm run build` limpos.
- Drag-and-drop testado de verdade: mouse (desktop) e emulação de toque (DevTools mobile ou
  Playwright).
- Screenshot antes/depois do Quadro em mobile, tablet e desktop, confirmando a mudança de
  1→2→4 colunas.
- Botões "Avançar →" e "Imprimir comanda" confirmados funcionando junto com o drag.
- Banner testado com pelo menos duas proporções de imagem diferentes (uma larga/baixa, uma mais
  quadrada), confirmando que nenhuma corta e o preenchimento desfocado funciona nos dois casos.
