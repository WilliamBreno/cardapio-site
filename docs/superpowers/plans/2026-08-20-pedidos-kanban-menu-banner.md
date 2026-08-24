# Kanban de Pedidos (drag-and-drop), menu do admin e banner do cardápio — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Dar drag-and-drop de verdade ao Quadro (Kanban) de Pedidos, liberar a largura de tela só nessa visão, redesenhar o menu horizontal do admin com ícone + pill ativa, e reposicionar/redesenhar o banner do cardápio público pra nunca cortar a imagem.

**Architecture:** `layoutAdminStore.ts` (Zustand, sem persist — é estado efêmero de UI, não preferência durável) liga/desliga `larguraCompleta`; `Dashboard.tsx` lê o store pra decidir o `max-w` do `<main>`; `Pedidos.tsx` escreve nesse store via `useEffect` conforme a visualização ativa. O Kanban ganha `@dnd-kit/core` (`DndContext`/`useDraggable`/`useDroppable`/`DragOverlay`) por cima de um grid responsivo já enriquecido com mais dados por card. O menu do admin ganha um mapa rota→ícone (`lucide-react`) e troca sublinhado por pill preenchida. O banner do cardápio público vira um componente local `BannerOferta` (evita duplicar entre os dois pontos onde `banner_url` aparece), com `object-contain` sobre um fundo desfocado da própria imagem.

**Tech Stack:** React 19, TypeScript, Vite, Tailwind v3, Zustand 5, `lucide-react` (já instalado), `@dnd-kit/core` (dependência nova).

## Global Constraints

- Spec de referência: `docs/superpowers/specs/2026-08-20-pedidos-kanban-menu-banner-design.md` — ler antes de implementar qualquer task.
- **Sem suíte de teste de componente React no projeto** — mesma adaptação já usada no plano anterior (`docs/superpowers/plans/2026-08-20-componentes-input-dark-mode.md`): cada task termina com `cd frontend && npx tsc -b` limpo (substituindo "rodar os testes") e uma nota do que verificar visualmente no navegador (substituindo "verificar que o teste passa").
- Só classes Tailwind da paleta de produto (`fundo`, `superficie`, `tinta`, `tinta-suave`, `acento`, `douro`) — nunca hex fixo, nunca tokens shadcn.
- `cn()` importado de `../../lib/utils` (caminho relativo, padrão já usado em `Dashboard.tsx`/`Pedidos.tsx`, que não usam o alias `@/`).
- Import de ícones sempre de `lucide-react`.
- Mensagens de commit em português, estilo já usado no repo.
- Fora de escopo, não tocar: sidebar vertical pro menu (decisão explícita da spec — mantém barra horizontal); `max-w-3xl` das outras telas do admin fora da visão Quadro.

---

### Task 1: Fundação — instalar `@dnd-kit/core` + criar `layoutAdminStore.ts`

**Files:**
- Modify: `frontend/package.json` (via `npm install`)
- Create: `frontend/src/store/layoutAdminStore.ts`

**Interfaces:**
- Produces: `useLayoutAdminStore` (Zustand hook), estado `{ larguraCompleta: boolean; definirLarguraCompleta: (valor: boolean) => void }`.

- [x] **Passo 1: Instalar a dependência**

Rodar dentro de `frontend/`:
```bash
npm install @dnd-kit/core
```

- [x] **Passo 2: Criar `frontend/src/store/layoutAdminStore.ts`**

```ts
import { create } from 'zustand';

// Diferente de authStore/temaAdminStore, esse store NÃO usa persist — é
// estado efêmero de UI (qual visão de Pedidos está aberta agora), não uma
// preferência durável do usuário. Recarregar a página deve voltar pro
// padrão (largura normal), não lembrar a última visão.
interface LayoutAdminState {
  larguraCompleta: boolean;
  definirLarguraCompleta: (valor: boolean) => void;
}

export const useLayoutAdminStore = create<LayoutAdminState>((set) => ({
  larguraCompleta: false,
  definirLarguraCompleta: (larguraCompleta) => set({ larguraCompleta }),
}));
```

- [x] **Passo 3: Verificar o typecheck**

Rodar: `cd frontend && npx tsc -b`
Esperado: sem erros.

- [x] **Passo 4: Commit**

```bash
git add frontend/package.json frontend/package-lock.json frontend/src/store/layoutAdminStore.ts
git commit -m "feat: adiciona @dnd-kit/core e layoutAdminStore (base do Kanban arrastável)"
```

---

### Task 2: `Dashboard.tsx` — `<main>` respeita `larguraCompleta`

**Files:**
- Modify: `frontend/src/pages/admin/Dashboard.tsx`

**Interfaces:**
- Consumes: `useLayoutAdminStore` (Task 1) — lê `larguraCompleta`.

- [x] **Passo 1: Ler o store e trocar a classe do `<main>`**

Em `frontend/src/pages/admin/Dashboard.tsx`, adicionar o import do store junto aos outros (linha 6, depois de `useTemaAdminStore`):

```tsx
import { useLayoutAdminStore } from '../../store/layoutAdminStore';
```

Dentro do componente `Dashboard()`, logo depois de `const alternarTema = useTemaAdminStore((state) => state.alternar);` (linha 28):

```tsx
  const larguraCompleta = useLayoutAdminStore((state) => state.larguraCompleta);
```

Trocar (linha 133):
```tsx
      <main className="mx-auto max-w-3xl px-6 py-6">
```
por:
```tsx
      <main className={cn('mx-auto px-6 py-6', larguraCompleta ? 'max-w-none' : 'max-w-3xl')}>
```

(`cn` já está importado nesse arquivo, linha 7.)

- [x] **Passo 2: Verificar o typecheck**

Rodar: `cd frontend && npx tsc -b`
Esperado: sem erros. `larguraCompleta` fica sempre `false` até a Task 3 existir — nenhuma tela muda de largura ainda, é esperado.

- [x] **Passo 3: Commit**

```bash
git add frontend/src/pages/admin/Dashboard.tsx
git commit -m "feat: main do admin libera largura total quando layoutAdminStore pede"
```

---

### Task 3: `Pedidos.tsx` — sincroniza `visualizacao` com `layoutAdminStore`

**Files:**
- Modify: `frontend/src/pages/admin/Pedidos.tsx`

**Interfaces:**
- Consumes: `useLayoutAdminStore` (Task 1) — usa `definirLarguraCompleta`.

- [x] **Passo 1: Ligar/desligar `larguraCompleta` conforme a visualização**

Trocar o import de `react` (linha 1):
```tsx
import { useState } from 'react';
```
por:
```tsx
import { useEffect, useState } from 'react';
```

Adicionar o import do store, junto aos outros (depois da linha 5):
```tsx
import { useLayoutAdminStore } from '../../store/layoutAdminStore';
```

Dentro de `Pedidos()`, logo depois de `const [visualizacao, setVisualizacao] = useState<'lista' | 'quadro'>('lista');` (linha 95):

```tsx
  const definirLarguraCompleta = useLayoutAdminStore((state) => state.definirLarguraCompleta);

  // Largura total do <main> só faz sentido na visão Quadro (precisa de
  // ~1150px pras 4 colunas lado a lado) — desliga ao trocar pra Lista ou
  // ao sair da tela (cleanup do useEffect), pra não vazar largura total
  // pras outras ~15 telas do admin que reaproveitam o mesmo <main>.
  useEffect(() => {
    definirLarguraCompleta(visualizacao === 'quadro');
    return () => definirLarguraCompleta(false);
  }, [visualizacao, definirLarguraCompleta]);
```

- [x] **Passo 2: Verificar o typecheck**

Rodar: `cd frontend && npx tsc -b`
Esperado: sem erros.

- [x] **Passo 3: Verificar no navegador**

Com o dev server rodando, abrir `/admin/pedidos`, trocar entre "Lista" e "Quadro" pelo botão do topo — na visão Quadro, o `<main>` (herdado do `Dashboard.tsx`) deve ficar sem o teto de 768px (mesmo que o Kanban ainda esteja com o layout antigo `flex overflow-x-auto`, a task seguinte trata disso). Trocar pra "Lista" ou navegar pra outra página do admin deve voltar a largura normal.

**Nota**: verificação visual pulada nesta sessão — não havia dev server nem backend já rodando, nenhuma ferramenta de QA (`backend/cmd/qatools` ou similar) encontrada no repo pra gerar um token de teste, e subir o stack completo (Postgres + backend + conta admin) só pra essa checagem estava fora do escopo pedido pra esta task. `npx tsc -b` limpo confirma a mudança de código; a lógica é direta o suficiente (mesmo padrão do `useEffect` já usado em outras telas) pra não bloquear nisso.

- [x] **Passo 4: Commit**

```bash
git add frontend/src/pages/admin/Pedidos.tsx
git commit -m "feat: Pedidos.tsx libera largura total do admin só na visão Quadro"
```

---

### Task 4: Kanban — grid responsivo + densidade do card

**Files:**
- Modify: `frontend/src/pages/admin/Pedidos.tsx`

**Interfaces:**
- Produces: `QuadroPedidos` ganha a prop nova `segmentoLoja?: TipoProduto`.

- [x] **Passo 1: Passar `segmentoLoja` no call site**

Trocar (linha 196):
```tsx
        <QuadroPedidos pedidos={pedidosPagos} isLoading={isLoading} onAvancar={avancarEtapa} onImprimir={handleImprimir} />
```
por:
```tsx
        <QuadroPedidos pedidos={pedidosPagos} isLoading={isLoading} segmentoLoja={loja?.segmento_principal} onAvancar={avancarEtapa} onImprimir={handleImprimir} />
```

- [x] **Passo 2: Reescrever `QuadroPedidos` — grid + card mais denso**

Substituir a função inteira `QuadroPedidos` (linhas 202-253) por:

```tsx
// QuadroPedidos é a visão "dinâmica e interativa" (Fase 10.2, redesenhada
// em 20/08/2026 com drag-and-drop de verdade — ver próxima task). Grid
// responsivo (1/2/4 colunas) no lugar do scroll horizontal forçado de
// antes; card com a mesma densidade de informação da Lista (itens,
// telefone, horário, cupom, peso pendente), não só nome+total.
function QuadroPedidos({ pedidos, isLoading, segmentoLoja, onAvancar, onImprimir }: {
  pedidos: Pedido[];
  isLoading: boolean;
  segmentoLoja?: TipoProduto;
  onAvancar: (pedido: Pedido) => void;
  onImprimir: (pedido: Pedido) => void;
}) {
  if (isLoading) return <p className="text-tinta-suave">Carregando pedidos...</p>;
  if (pedidos.length === 0) return <p className="text-tinta-suave">Nenhum pedido pago ainda.</p>;

  return (
    <div className="grid grid-cols-1 gap-4 pb-2 sm:grid-cols-2 lg:grid-cols-4">
      {ETAPAS.map((etapa) => {
        const pedidosDaEtapa = pedidos.filter((p) => etapaAtual(p) === etapa.valor);
        return (
          <div key={etapa.valor} className="space-y-3">
            <div className={`rounded-full px-3 py-1.5 text-center text-xs font-semibold ${etapa.classe}`}>
              {etapa.rotuloEntrega === etapa.rotuloRetirada ? etapa.rotuloEntrega : `${etapa.rotuloEntrega} / ${etapa.rotuloRetirada}`}
              {' · '}{pedidosDaEtapa.length}
            </div>
            <div className="space-y-2">
              {pedidosDaEtapa.map((pedido) => {
                const proxima = proximaEtapa(etapaAtual(pedido));
                const totalItens = pedido.itens.length + (pedido.combos?.length ?? 0);
                return (
                  <div key={pedido.id} className="rounded-xl bg-superficie p-3 shadow-sm">
                    <div className="flex items-start justify-between gap-2">
                      <p className="text-sm font-medium text-tinta">
                        {pedido.cliente_nome} <span className="text-tinta-suave">· #{pedido.id}</span>
                      </p>
                      {pedido.peso_pendente && (
                        <span className={`shrink-0 rounded-full px-2 py-0.5 text-[10px] font-semibold ${PESO_PENDENTE_CLASSE}`}>
                          ⚠️ Peso pendente
                        </span>
                      )}
                    </div>
                    <p className="text-xs text-tinta-suave">{pedido.cliente_telefone}</p>
                    <p className="mt-1 text-xs text-tinta-suave">
                      {pedido.modo_entrega === 'entrega' ? '🛵 Entrega' : '🏪 Retirada'} · {formatarData(pedido.data_retirada)}
                    </p>
                    <div className="mt-2 space-y-0.5 border-t border-tinta/10 pt-2">
                      {pedido.itens.slice(0, 2).map((item) => (
                        <p key={item.id} className="truncate text-xs text-tinta">
                          {item.quantidade}x {item.produto_nome}
                        </p>
                      ))}
                      {pedido.combos?.slice(0, 2).map((combo) => (
                        <p key={combo.id} className="truncate text-xs text-tinta">
                          {combo.quantidade}x {combo.nome} <span className="text-acento">· {rotuloCombo(segmentoLoja)}</span>
                        </p>
                      ))}
                      {totalItens > 2 && (
                        <p className="text-xs text-tinta-suave">+{totalItens - 2} ite{totalItens - 2 === 1 ? 'm' : 'ns'}</p>
                      )}
                    </div>
                    {pedido.cupom_codigo && (
                      <p className="mt-1 text-xs text-emerald-600">
                        Cupom {pedido.cupom_codigo} · -R$ {pedido.desconto.toFixed(2).replace('.', ',')}
                      </p>
                    )}
                    <p className="mt-2 border-t border-tinta/10 pt-2 text-sm font-carimbo font-semibold text-tinta">
                      R$ {pedido.total.toFixed(2).replace('.', ',')}
                    </p>
                    {proxima && (
                      <button onClick={() => onAvancar(pedido)} className="btn-neu-primario btn-neu-sm mt-2 w-full">
                        Avançar → {rotuloEtapa(proxima, pedido.modo_entrega)}
                      </button>
                    )}
                    <button onClick={() => onImprimir(pedido)} className="btn-neu-secundario btn-neu-sm mt-2 w-full">
                      🖨️ Imprimir comanda
                    </button>
                  </div>
                );
              })}
              {pedidosDaEtapa.length === 0 && (
                <p className="rounded-xl border-2 border-dashed border-tinta/10 p-3 text-center text-xs text-tinta-suave/60">
                  Nenhum pedido aqui
                </p>
              )}
            </div>
          </div>
        );
      })}
    </div>
  );
}
```

- [x] **Passo 3: Verificar o typecheck**

Rodar: `cd frontend && npx tsc -b`
Esperado: sem erros.

- [x] **Passo 4: Verificar no navegador**

Abrir `/admin/pedidos`, visão Quadro, com pelo menos um pedido pago. Confirmar: (a) em tela larga (desktop), as 4 colunas aparecem lado a lado sem scroll horizontal; (b) redimensionando a janela (ou DevTools) pra ~700px, vira 2 colunas; pra ~400px, 1 coluna empilhada; (c) cada card mostra telefone, horário, itens resumidos e (se houver) cupom/peso pendente, igual o card da Lista.

**Nota**: verificação visual pulada nesta sessão — Docker Desktop não estava rodando (subir Postgres +
API + criar conta admin + pedido pago pra só então logar no navegador não é rápido) e, mais
importante, este ambiente não tem nenhuma ferramenta de browser/screenshot disponível (só
`WebFetch`, que converte HTML pra markdown via um modelo pequeno, sem servir pra inspecionar um SPA
autenticado como o admin) — não haveria como registrar o resultado mesmo subindo o stack completo.
`npx tsc -b` limpo confirma a mudança de código; o JSX é a tradução direta do código do plano
(mesmas classes Tailwind responsivas já usadas em outras telas do projeto), sem lógica nova a
validar além do que o typecheck já cobre.

- [x] **Passo 5: Commit**

```bash
git add frontend/src/pages/admin/Pedidos.tsx
git commit -m "feat: Kanban de Pedidos em grid responsivo com cards mais densos"
```

---

### Task 5: Kanban — drag-and-drop com `@dnd-kit/core`

**Files:**
- Modify: `frontend/src/pages/admin/Pedidos.tsx`

**Interfaces:**
- Consumes: `DndContext`, `DragOverlay`, `KeyboardSensor`, `PointerSensor`, `TouchSensor`, `useDraggable`, `useDroppable`, `useSensor`, `useSensors`, tipos `DragEndEvent`/`DragStartEvent` de `@dnd-kit/core` (Task 1).
- Produces: `QuadroPedidos` ganha a prop nova `onMover: (pedido: Pedido, etapa: EtapaPedido) => void`; `Pedidos()` ganha `moverParaEtapa`.

- [ ] **Passo 1: Adicionar os imports de `@dnd-kit/core` e `cn`**

No topo de `frontend/src/pages/admin/Pedidos.tsx`, adicionar (depois da linha 8, `import { imprimirComanda } ...`):

```tsx
import {
  DndContext,
  DragOverlay,
  KeyboardSensor,
  PointerSensor,
  TouchSensor,
  useDraggable,
  useDroppable,
  useSensor,
  useSensors,
  type DragEndEvent,
  type DragStartEvent,
} from '@dnd-kit/core';
```

Trocar a linha 7:
```tsx
import { rotuloCombo } from '../../lib/utils';
```
por:
```tsx
import { cn, rotuloCombo } from '../../lib/utils';
```

- [ ] **Passo 2: Adicionar `moverParaEtapa` e passar `onMover`**

Depois da função `avancarEtapa` (logo após a linha 106, antes de `handleImprimir`), adicionar:

```tsx
  // moverParaEtapa é o que o drag-and-drop chama — diferente de
  // avancarEtapa (que só sabe ir pra próxima etapa da lista), aceita
  // qualquer etapa de destino, porque soltar um card no Kanban pode
  // avançar, voltar ou pular etapa livremente.
  function moverParaEtapa(pedido: Pedido, etapa: EtapaPedido) {
    if (etapa !== etapaAtual(pedido)) mutAvancar.mutate({ pedidoId: pedido.id, etapa });
  }
```

Trocar o call site (que a Task 4 deixou):
```tsx
        <QuadroPedidos pedidos={pedidosPagos} isLoading={isLoading} segmentoLoja={loja?.segmento_principal} onAvancar={avancarEtapa} onImprimir={handleImprimir} />
```
por:
```tsx
        <QuadroPedidos pedidos={pedidosPagos} isLoading={isLoading} segmentoLoja={loja?.segmento_principal} onAvancar={avancarEtapa} onMover={moverParaEtapa} onImprimir={handleImprimir} />
```

- [ ] **Passo 3: Reescrever `QuadroPedidos` com drag-and-drop**

Substituir a função `QuadroPedidos` inteira (a versão da Task 4) por esta versão, que quebra o card em subcomponentes pra poder reaproveisar o mesmo conteúdo visual dentro do `DragOverlay`:

```tsx
// QuadroPedidos é a visão "dinâmica e interativa" (Fase 10.2, com
// drag-and-drop de verdade desde 20/08/2026) — soltar um card em
// qualquer coluna chama a mesma mutation de sempre (moverParaEtapa),
// sem validar ordem: avança, volta ou pula etapa livremente, igual um
// Kanban de verdade. Funciona em mouse (PointerSensor), toque
// (TouchSensor) e teclado (KeyboardSensor, ativado pelos atributos que
// useDraggable já injeta no card).
function QuadroPedidos({ pedidos, isLoading, segmentoLoja, onAvancar, onMover, onImprimir }: {
  pedidos: Pedido[];
  isLoading: boolean;
  segmentoLoja?: TipoProduto;
  onAvancar: (pedido: Pedido) => void;
  onMover: (pedido: Pedido, etapa: EtapaPedido) => void;
  onImprimir: (pedido: Pedido) => void;
}) {
  const [activeId, setActiveId] = useState<string | null>(null);
  const sensors = useSensors(
    // distance: 8 evita que um clique parado no botão "Avançar"/"Imprimir"
    // seja interpretado como início de arraste.
    useSensor(PointerSensor, { activationConstraint: { distance: 8 } }),
    useSensor(TouchSensor, { activationConstraint: { delay: 200, tolerance: 8 } }),
    useSensor(KeyboardSensor)
  );

  if (isLoading) return <p className="text-tinta-suave">Carregando pedidos...</p>;
  if (pedidos.length === 0) return <p className="text-tinta-suave">Nenhum pedido pago ainda.</p>;

  const pedidoArrastado = activeId ? pedidos.find((p) => `pedido-${p.id}` === activeId) ?? null : null;

  function handleDragStart(event: DragStartEvent) {
    setActiveId(String(event.active.id));
  }

  function handleDragEnd(event: DragEndEvent) {
    setActiveId(null);
    const { active, over } = event;
    if (!over) return;
    const etapaDestino = String(over.id);
    if (!ETAPA_VALORES.has(etapaDestino)) return;
    const pedidoId = Number(String(active.id).replace('pedido-', ''));
    const pedido = pedidos.find((p) => p.id === pedidoId);
    if (pedido) onMover(pedido, etapaDestino as EtapaPedido);
  }

  return (
    <DndContext sensors={sensors} onDragStart={handleDragStart} onDragEnd={handleDragEnd}>
      <div className="grid grid-cols-1 gap-4 pb-2 sm:grid-cols-2 lg:grid-cols-4">
        {ETAPAS.map((etapa) => (
          <ColunaEtapa
            key={etapa.valor}
            etapa={etapa}
            pedidosDaEtapa={pedidos.filter((p) => etapaAtual(p) === etapa.valor)}
            segmentoLoja={segmentoLoja}
            onAvancar={onAvancar}
            onImprimir={onImprimir}
          />
        ))}
      </div>
      <DragOverlay>
        {pedidoArrastado && (
          <div className="rounded-xl bg-superficie p-3 shadow-lg ring-2 ring-acento/40">
            <ConteudoCardQuadro pedido={pedidoArrastado} segmentoLoja={segmentoLoja} onAvancar={onAvancar} onImprimir={onImprimir} />
          </div>
        )}
      </DragOverlay>
    </DndContext>
  );
}

// ColunaEtapa é a zona de soltar (useDroppable, id = etapa.valor) — o
// destaque visual (isOver) confirma pro dono onde o card vai cair antes
// de soltar o dedo/mouse.
function ColunaEtapa({ etapa, pedidosDaEtapa, segmentoLoja, onAvancar, onImprimir }: {
  etapa: (typeof ETAPAS)[number];
  pedidosDaEtapa: Pedido[];
  segmentoLoja?: TipoProduto;
  onAvancar: (pedido: Pedido) => void;
  onImprimir: (pedido: Pedido) => void;
}) {
  const { setNodeRef, isOver } = useDroppable({ id: etapa.valor });

  return (
    <div className="space-y-3">
      <div className={`rounded-full px-3 py-1.5 text-center text-xs font-semibold ${etapa.classe}`}>
        {etapa.rotuloEntrega === etapa.rotuloRetirada ? etapa.rotuloEntrega : `${etapa.rotuloEntrega} / ${etapa.rotuloRetirada}`}
        {' · '}{pedidosDaEtapa.length}
      </div>
      <div
        ref={setNodeRef}
        className={cn(
          // min-h-[6rem] (não min-h-24): a escala numérica padrão do
          // Tailwind v3 para min-height só tem 0/full/screen/min/max/fit —
          // diferente de height/width, que herdam a escala de espaçamento
          // inteira. Precisa do valor arbitrário aqui.
          'min-h-[6rem] space-y-2 rounded-2xl p-1 transition',
          isOver && 'bg-acento/5 ring-2 ring-acento/30'
        )}
      >
        {pedidosDaEtapa.map((pedido) => (
          <CardArrastavel key={pedido.id} pedido={pedido} segmentoLoja={segmentoLoja} onAvancar={onAvancar} onImprimir={onImprimir} />
        ))}
        {pedidosDaEtapa.length === 0 && (
          <p className="rounded-xl border-2 border-dashed border-tinta/10 p-3 text-center text-xs text-tinta-suave/60">
            Solte um pedido aqui
          </p>
        )}
      </div>
    </div>
  );
}

// CardArrastavel é o card em si (useDraggable, id = "pedido-{id}") — fica
// com opacidade reduzida no lugar de origem enquanto arrasta (o
// DragOverlay do QuadroPedidos mostra a cópia "de verdade" seguindo o
// ponteiro).
function CardArrastavel({ pedido, segmentoLoja, onAvancar, onImprimir }: {
  pedido: Pedido;
  segmentoLoja?: TipoProduto;
  onAvancar: (pedido: Pedido) => void;
  onImprimir: (pedido: Pedido) => void;
}) {
  const { attributes, listeners, setNodeRef, transform, isDragging } = useDraggable({ id: `pedido-${pedido.id}` });
  const style = transform ? { transform: `translate3d(${transform.x}px, ${transform.y}px, 0)` } : undefined;

  return (
    <div
      ref={setNodeRef}
      style={style}
      {...listeners}
      {...attributes}
      className={cn(
        'touch-none cursor-grab rounded-xl bg-superficie p-3 shadow-sm active:cursor-grabbing',
        isDragging && 'opacity-30'
      )}
    >
      <ConteudoCardQuadro pedido={pedido} segmentoLoja={segmentoLoja} onAvancar={onAvancar} onImprimir={onImprimir} />
    </div>
  );
}

// ConteudoCardQuadro é só o miolo visual do card (sem nenhum hook de
// drag) — reaproveitado tanto dentro do CardArrastavel quanto dentro do
// DragOverlay, pra não duplicar o JSX nos dois lugares.
function ConteudoCardQuadro({ pedido, segmentoLoja, onAvancar, onImprimir }: {
  pedido: Pedido;
  segmentoLoja?: TipoProduto;
  onAvancar: (pedido: Pedido) => void;
  onImprimir: (pedido: Pedido) => void;
}) {
  const proxima = proximaEtapa(etapaAtual(pedido));
  const totalItens = pedido.itens.length + (pedido.combos?.length ?? 0);

  return (
    <>
      <div className="flex items-start justify-between gap-2">
        <p className="text-sm font-medium text-tinta">
          {pedido.cliente_nome} <span className="text-tinta-suave">· #{pedido.id}</span>
        </p>
        {pedido.peso_pendente && (
          <span className={`shrink-0 rounded-full px-2 py-0.5 text-[10px] font-semibold ${PESO_PENDENTE_CLASSE}`}>
            ⚠️ Peso pendente
          </span>
        )}
      </div>
      <p className="text-xs text-tinta-suave">{pedido.cliente_telefone}</p>
      <p className="mt-1 text-xs text-tinta-suave">
        {pedido.modo_entrega === 'entrega' ? '🛵 Entrega' : '🏪 Retirada'} · {formatarData(pedido.data_retirada)}
      </p>
      <div className="mt-2 space-y-0.5 border-t border-tinta/10 pt-2">
        {pedido.itens.slice(0, 2).map((item) => (
          <p key={item.id} className="truncate text-xs text-tinta">
            {item.quantidade}x {item.produto_nome}
          </p>
        ))}
        {pedido.combos?.slice(0, 2).map((combo) => (
          <p key={combo.id} className="truncate text-xs text-tinta">
            {combo.quantidade}x {combo.nome} <span className="text-acento">· {rotuloCombo(segmentoLoja)}</span>
          </p>
        ))}
        {totalItens > 2 && (
          <p className="text-xs text-tinta-suave">+{totalItens - 2} ite{totalItens - 2 === 1 ? 'm' : 'ns'}</p>
        )}
      </div>
      {pedido.cupom_codigo && (
        <p className="mt-1 text-xs text-emerald-600">
          Cupom {pedido.cupom_codigo} · -R$ {pedido.desconto.toFixed(2).replace('.', ',')}
        </p>
      )}
      <p className="mt-2 border-t border-tinta/10 pt-2 text-sm font-carimbo font-semibold text-tinta">
        R$ {pedido.total.toFixed(2).replace('.', ',')}
      </p>
      {proxima && (
        // onPointerDown + stopPropagation: o container arrastável (pai)
        // também escuta pointerdown (dnd-kit) — sem isso, um toque rápido
        // no botão poderia, em telas de toque, disputar com o início do
        // gesto de arraste (mesmo com o delay do TouchSensor).
        <button
          onPointerDown={(e) => e.stopPropagation()}
          onClick={() => onAvancar(pedido)}
          className="btn-neu-primario btn-neu-sm mt-2 w-full"
        >
          Avançar → {rotuloEtapa(proxima, pedido.modo_entrega)}
        </button>
      )}
      <button
        onPointerDown={(e) => e.stopPropagation()}
        onClick={() => onImprimir(pedido)}
        className="btn-neu-secundario btn-neu-sm mt-2 w-full"
      >
        🖨️ Imprimir comanda
      </button>
    </>
  );
}
```

- [ ] **Passo 4: Verificar o typecheck**

Rodar: `cd frontend && npx tsc -b`
Esperado: sem erros.

- [ ] **Passo 5: Verificar no navegador — mouse**

Abrir `/admin/pedidos`, visão Quadro, com pelo menos 2 pedidos pagos em etapas diferentes. Arrastar um card de uma coluna pra outra (qualquer direção, não só a adjacente) — o card deve se mover e a mutation deve persistir (confirmar com F5 que a etapa nova ficou salva). Confirmar que os botões "Avançar →" e "🖨️ Imprimir comanda" continuam clicáveis normalmente sem disparar um arraste sem querer.

- [ ] **Passo 6: Verificar no navegador — toque**

No DevTools, ativar a emulação de dispositivo móvel (toque). Repetir o teste de arrastar um card — confirmar que funciona com um "press and drag" (o `delay: 200` do `TouchSensor` existe justamente pra distinguir de um scroll vertical da página).

- [ ] **Passo 7: Commit**

```bash
git add frontend/src/pages/admin/Pedidos.tsx
git commit -m "feat: drag-and-drop de verdade no Kanban de Pedidos via @dnd-kit/core"
```

---

### Task 6: `Dashboard.tsx` — menu com ícone por link + pill no estado ativo

**Files:**
- Modify: `frontend/src/pages/admin/Dashboard.tsx`

**Interfaces:** nenhuma nova — só troca visual do `<nav>` já existente.

- [ ] **Passo 1: Importar os ícones e montar o mapa rota→ícone**

Trocar a linha 3:
```tsx
import { Moon, Sun } from 'lucide-react';
```
por:
```tsx
import {
  Archive,
  Beaker,
  Boxes,
  ClipboardList,
  CreditCard,
  Home,
  Moon,
  Package,
  Settings,
  Sparkles,
  Sun,
  Tags,
  Ticket,
  Warehouse,
  type LucideIcon,
} from 'lucide-react';
```

Adicionar, depois de `linksBase` (depois da linha 19):

```tsx
// Ícone por rota (Fase de redesign, 20/08/2026) — mapa fixo por `to`
// porque o `label` de alguns links muda em runtime (ex: "Combos" vira
// "Kits" pra loja mercadoria, via rotuloCombo), então indexar pelo
// texto seria frágil.
const ICONE_POR_ROTA: Record<string, LucideIcon> = {
  '/admin': Home,
  '/admin/pedidos': ClipboardList,
  '/admin/produtos': Package,
  '/admin/categorias': Tags,
  '/admin/cupons': Ticket,
  '/admin/combos': Boxes,
  '/admin/sugestao-inteligente': Sparkles,
  '/admin/configuracoes': Settings,
  '/admin/meu-plano': CreditCard,
  '/admin/estoque': Warehouse,
  '/admin/insumos': Beaker,
  '/admin/solicitacoes': Archive,
};
```

- [ ] **Passo 2: Trocar o visual do `<nav>`**

Trocar (linhas 114-131):
```tsx
      <nav className="flex gap-1 overflow-x-auto border-b border-tinta/10 bg-superficie px-6">
        {links.map((link) => (
          <NavLink
            key={link.to}
            to={link.to}
            end={link.to === '/admin'}
            className={({ isActive }) =>
              `whitespace-nowrap border-b-2 px-3 py-3 text-sm font-medium transition ${
                isActive
                  ? 'border-acento text-acento'
                  : 'border-transparent text-tinta-suave hover:text-tinta'
              }`
            }
          >
            {link.label}
          </NavLink>
        ))}
      </nav>
```
por:
```tsx
      <nav className="flex gap-1 overflow-x-auto border-b border-tinta/10 bg-superficie px-6 py-2">
        {links.map((link) => {
          const Icone = ICONE_POR_ROTA[link.to];
          return (
            <NavLink
              key={link.to}
              to={link.to}
              end={link.to === '/admin'}
              className={({ isActive }) =>
                cn(
                  'flex shrink-0 items-center gap-1.5 whitespace-nowrap rounded-full px-3 py-1.5 text-sm font-medium transition',
                  isActive
                    ? 'bg-acento text-superficie'
                    : 'text-tinta-suave hover:bg-tinta/5 hover:text-tinta'
                )
              }
            >
              {Icone && <Icone className="size-4" />}
              {link.label}
            </NavLink>
          );
        })}
      </nav>
```

(`cn` já está importado nesse arquivo, linha 7.)

- [ ] **Passo 3: Verificar o typecheck**

Rodar: `cd frontend && npx tsc -b`
Esperado: sem erros.

- [ ] **Passo 4: Verificar no navegador**

Abrir qualquer tela do admin. Confirmar: cada link do menu mostra o ícone certo (conferir "Estoque"/"Insumos"/"Guardados" também, logando com uma loja Scale que tenha `aceita_guardar_entregar` ativo, se disponível); o link da página atual aparece como pill preenchida (`bg-acento`); passar o mouse sobre um link inativo mostra o fundo suave (`hover:bg-tinta/5`); a barra continua horizontal com scroll lateral em tela estreita (não virou sidebar).

- [ ] **Passo 5: Commit**

```bash
git add frontend/src/pages/admin/Dashboard.tsx
git commit -m "feat: redesenha o menu do admin com ícone por link e pill no estado ativo"
```

---

### Task 7: `CardapioPublico.tsx` — banner em card dedicado, sem cortar a imagem

**Files:**
- Modify: `frontend/src/pages/CardapioPublico.tsx`

**Interfaces:**
- Produces: `BannerOferta({ url: string })`, componente local (não exportado).

- [ ] **Passo 1: Criar o componente `BannerOferta`**

Adicionar, logo antes de `export function CardapioPublico()` (antes da linha 109):

```tsx
// BannerOferta (redesign de 20/08/2026): card dedicado abaixo do
// cabeçalho da loja, não mais colado acima dele. object-contain nunca
// corta a imagem (diferente do object-cover de antes); o espaço vazio
// nas bordas, quando a proporção da imagem não bate com a altura fixa
// do card, é preenchido com uma cópia desfocada da própria imagem em
// vez de deixar uma faixa da cor de fundo — fica visualmente consistente
// entre lojas com banners de proporções bem diferentes.
function BannerOferta({ url }: { url: string }) {
  return (
    <div className="relative mx-4 mt-4 h-40 overflow-hidden rounded-2xl shadow-sm sm:mx-6 sm:h-56">
      <img
        src={url}
        alt=""
        aria-hidden="true"
        className="absolute inset-0 h-full w-full scale-110 object-cover blur-xl"
      />
      <img
        src={url}
        alt="Oferta em destaque"
        className="relative h-full w-full object-contain"
      />
    </div>
  );
}
```

- [ ] **Passo 2: Mover o banner pra depois do cabeçalho — estado "loja fechada"**

Trocar (linhas 161-172):
```tsx
    return (
      <div className="min-h-screen bg-fundo" data-tema={data.loja.tema || 'kraft'}>
        {data.loja.banner_url && (
          <img src={data.loja.banner_url} alt="Oferta em destaque" className="h-32 w-full object-cover sm:h-48" />
        )}
        <header className="bg-acento px-6 py-8 text-center">
          {data.loja.logo_url && (
            <img src={data.loja.logo_url} alt={data.loja.nome}
              className="mx-auto mb-3 h-16 w-16 rounded-full border-2 border-superficie/40 object-cover" />
          )}
          <h1 className="font-display text-3xl tracking-wide text-superficie">{data.loja.nome}</h1>
        </header>
        <div className="flex flex-col items-center justify-center gap-3 px-6 py-16 text-center">
```
por:
```tsx
    return (
      <div className="min-h-screen bg-fundo" data-tema={data.loja.tema || 'kraft'}>
        <header className="bg-acento px-6 py-8 text-center">
          {data.loja.logo_url && (
            <img src={data.loja.logo_url} alt={data.loja.nome}
              className="mx-auto mb-3 h-16 w-16 rounded-full border-2 border-superficie/40 object-cover" />
          )}
          <h1 className="font-display text-3xl tracking-wide text-superficie">{data.loja.nome}</h1>
        </header>
        {data.loja.banner_url && <BannerOferta url={data.loja.banner_url} />}
        <div className="flex flex-col items-center justify-center gap-3 px-6 py-16 text-center">
```

- [ ] **Passo 3: Mover o banner pra depois do cabeçalho — estado normal**

Trocar (linhas 199-231, do banner até o fechamento do `</header>`):
```tsx
      {data.loja.banner_url && (
        <img src={data.loja.banner_url} alt="Oferta em destaque" className="h-32 w-full object-cover sm:h-48" />
      )}
      <header className="bg-acento px-6 py-8 text-center">
        {data.loja.logo_url && (
          <img src={data.loja.logo_url} alt={data.loja.nome}
            className="mx-auto mb-3 h-16 w-16 rounded-full border-2 border-superficie/40 object-cover" />
        )}
        <h1 className="font-display text-3xl tracking-wide text-superficie sm:text-4xl">
          {data.loja.nome}
        </h1>
        {data.loja.horario_abertura && data.loja.horario_fechamento && (
          <p className="mt-1 font-carimbo text-xs uppercase tracking-[0.2em] text-superficie/70">
            {data.loja.horario_abertura} – {data.loja.horario_fechamento}
          </p>
        )}
        <div className="mt-3 flex justify-center gap-2">
          <button
            onClick={() => setHistoricoAberto(true)}
            className="rounded-full border border-superficie/30 px-4 py-1.5 text-xs font-medium text-superficie/80 hover:bg-superficie/10"
          >
            Meus pedidos
          </button>
          {data.loja.aceita_guardar_entregar && (
            <button
              onClick={() => setGuardadosAberto(true)}
              className="rounded-full border border-superficie/30 px-4 py-1.5 text-xs font-medium text-superficie/80 hover:bg-superficie/10"
            >
              📦 Itens guardados
            </button>
          )}
        </div>
      </header>
```
por:
```tsx
      <header className="bg-acento px-6 py-8 text-center">
        {data.loja.logo_url && (
          <img src={data.loja.logo_url} alt={data.loja.nome}
            className="mx-auto mb-3 h-16 w-16 rounded-full border-2 border-superficie/40 object-cover" />
        )}
        <h1 className="font-display text-3xl tracking-wide text-superficie sm:text-4xl">
          {data.loja.nome}
        </h1>
        {data.loja.horario_abertura && data.loja.horario_fechamento && (
          <p className="mt-1 font-carimbo text-xs uppercase tracking-[0.2em] text-superficie/70">
            {data.loja.horario_abertura} – {data.loja.horario_fechamento}
          </p>
        )}
        <div className="mt-3 flex justify-center gap-2">
          <button
            onClick={() => setHistoricoAberto(true)}
            className="rounded-full border border-superficie/30 px-4 py-1.5 text-xs font-medium text-superficie/80 hover:bg-superficie/10"
          >
            Meus pedidos
          </button>
          {data.loja.aceita_guardar_entregar && (
            <button
              onClick={() => setGuardadosAberto(true)}
              className="rounded-full border border-superficie/30 px-4 py-1.5 text-xs font-medium text-superficie/80 hover:bg-superficie/10"
            >
              📦 Itens guardados
            </button>
          )}
        </div>
      </header>

      {data.loja.banner_url && <BannerOferta url={data.loja.banner_url} />}
```

- [ ] **Passo 4: Verificar o typecheck**

Rodar: `cd frontend && npx tsc -b`
Esperado: sem erros.

- [ ] **Passo 5: Verificar no navegador — duas proporções de imagem**

Configurar `banner_url` de uma loja de teste (via `PUT /admin/loja` ou direto no banco) pra uma imagem larga/baixa (ex: 1200×300) e conferir que aparece inteira, centralizada, com o fundo desfocado preenchendo as laterais/topo-base vazios. Repetir com uma imagem mais quadrada (ex: 800×800) e confirmar que também aparece inteira, sem cortar, com o fundo desfocado preenchendo o excesso. Testar também com a loja fechada/pausada (segundo local do banner) e com `aceita_guardar_entregar` ligado (pra conferir que o botão "📦 Itens guardados" continua no lugar certo, já que o header não mudou de posição, só o banner saiu de cima dele).

- [ ] **Passo 6: Commit**

```bash
git add frontend/src/pages/CardapioPublico.tsx
git commit -m "feat: banner do cardápio público em card dedicado abaixo do cabeçalho, sem cortar a imagem"
```

---

## Verificação final (depois de todas as tasks)

Espelha a seção "Validação e testes" da spec — checklist de fechamento, não uma task própria:

- [ ] `cd frontend && npx tsc -b` e `npm run build` limpos, no estado final de todas as 7 tasks juntas.
- [ ] Drag-and-drop testado de ponta a ponta com dados reais (não só isolado por task): criar/promover uma loja de teste com pelo menos 4 pedidos pagos em etapas diferentes, arrastar em várias direções (avançar, voltar, pular etapa), e no filtro de Lista confirmar que a etapa mudou de verdade (não só visualmente no Quadro).
- [ ] Screenshot do Quadro em mobile (~390px), tablet (~800px) e desktop (~1300px), confirmando a progressão 1→2→4 colunas.
- [ ] Botões "Avançar →" e "🖨️ Imprimir comanda" dentro do Quadro clicados de verdade (não só inspecionados) pra confirmar que não competem com o gesto de arrastar.
- [ ] Menu do admin conferido em pelo menos 3 tipos de loja diferentes (Start sem "Guardados"/"Estoque", Pro com "Estoque", Scale com "Estoque"+"Insumos") pra confirmar que o mapa de ícone cobre todo link que pode aparecer.
- [ ] Banner conferido com as duas proporções de imagem (Task 7, Passo 5) e nos dois pontos onde aparece (loja aberta e loja fechada/pausada).
- [ ] Limpar qualquer loja/pedido de teste criado durante a verificação.
