import { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
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
import { listarPedidos, buscarLoja } from '../../api/admin';
import { atualizarStatusEntrega, type EtapaPedido } from '../../api/rastreamento';
import type { Pedido, StatusPedido, TipoProduto } from '../../api/types';
import { cn, rotuloCombo } from '../../lib/utils';
import { imprimirComanda, bluetoothSuportado, conectarImpressora, tentarReconectarSilenciosamente } from '../../lib/impressoraBluetooth';
import { useLayoutAdminStore } from '../../store/layoutAdminStore';
import { useImpressoraStore } from '../../store/impressoraStore';

// ESTILO_AVANCAR varia o botão "Avançar" pela etapa ATUAL do pedido (não
// a de destino) — sinaliza visualmente o que precisa de atenção agora.
// "A preparar" pulsa (pedido novo, ainda não começou) e mantém a cor de
// marca; as demais têm cor própria, sem pulsar. "entregue" nunca chega a
// renderizar o botão (não tem próxima etapa) — mantido só por
// completude do Record.
const ESTILO_AVANCAR: Record<EtapaPedido, string> = {
  a_preparar: 'bg-acento btn-neu-pulsa',
  preparando: 'bg-amber-600',
  saiu_para_entrega: 'bg-sky-600',
  entregue: 'bg-emerald-600',
};

const statusInfo: Record<StatusPedido, { label: string; classe: string }> = {
  aguardando_pagamento: { label: 'Aguardando pagamento', classe: 'bg-douro/20 text-douro' },
  pago: { label: 'Pago', classe: 'bg-emerald-100 text-emerald-700' },
  cancelado: { label: 'Cancelado', classe: 'bg-acento/10 text-acento' },
};

// ETAPAS (Fase 10.2): as 4 etapas do fluxo de preparo/entrega, em ordem —
// "Avançar" sempre pula pra próxima da lista. rotuloEntrega é o texto pra
// pedido modo_entrega === 'entrega'; rotuloRetirada é o mesmo passo, mas
// pra quem só retira no balcão (não faz sentido "saiu pra entrega" nesse
// caso — decisão confirmada com o William).
const ETAPAS: { valor: EtapaPedido; rotuloEntrega: string; rotuloRetirada: string; classe: string }[] = [
  { valor: 'a_preparar', rotuloEntrega: '🧾 A preparar', rotuloRetirada: '🧾 A preparar', classe: 'bg-tinta/10 text-tinta-suave' },
  { valor: 'preparando', rotuloEntrega: '👨‍🍳 Preparando', rotuloRetirada: '👨‍🍳 Preparando', classe: 'bg-amber-100 text-amber-800' },
  { valor: 'saiu_para_entrega', rotuloEntrega: '🛵 Saiu para entrega', rotuloRetirada: '🏪 Pronto pra retirada', classe: 'bg-douro/20 text-douro' },
  { valor: 'entregue', rotuloEntrega: '✅ Entregue', rotuloRetirada: '✅ Entregue', classe: 'bg-emerald-100 text-emerald-700' },
];

// etapaAtual trata '' (pedido pago antes da Fase 10.2, sem etapa gravada)
// como "a_preparar" — a primeira etapa — em vez de deixar o pedido sem
// nenhuma etapa visível.
function etapaAtual(pedido: Pedido): EtapaPedido {
  return (pedido.status_entrega || 'a_preparar') as EtapaPedido;
}

function infoEtapa(etapa: EtapaPedido) {
  return ETAPAS.find((e) => e.valor === etapa) ?? ETAPAS[0];
}

function rotuloEtapa(etapa: EtapaPedido, modoEntrega: string) {
  const info = infoEtapa(etapa);
  return modoEntrega === 'entrega' ? info.rotuloEntrega : info.rotuloRetirada;
}

// proximaEtapa devolve a etapa seguinte, ou null se já é a última — usado
// tanto pro botão "Avançar" (lista e quadro) quanto pra decidir se ele
// aparece.
function proximaEtapa(etapa: EtapaPedido): EtapaPedido | null {
  const i = ETAPAS.findIndex((e) => e.valor === etapa);
  return i === -1 || i === ETAPAS.length - 1 ? null : ETAPAS[i + 1].valor;
}

const PESO_PENDENTE_CLASSE = 'bg-amber-100 text-amber-800';

type FiltroPedido = 'todos' | StatusPedido | 'peso_pendente' | EtapaPedido;

// "Pagos" saiu da lista (Fase 10.2 feedback do William, 20/08/2026) —
// redundante com os filtros de etapa (a_preparar/preparando/etc já
// implicam pedido pago, não faz sentido ter os dois). "Cancelados" foi
// pro fim, dando ênfase aos pedidos novos/em andamento primeiro.
const filtrosBase: { valor: FiltroPedido; label: string }[] = [
  { valor: 'todos', label: 'Todos' },
  { valor: 'aguardando_pagamento', label: 'Aguardando pagamento' },
  { valor: 'a_preparar', label: '🧾 A preparar' },
  { valor: 'preparando', label: '👨‍🍳 Preparando' },
  { valor: 'saiu_para_entrega', label: '🛵 Saiu p/ entrega' },
  { valor: 'entregue', label: '✅ Entregue' },
  { valor: 'cancelado', label: 'Cancelados' },
];

function formatarData(iso: string): string {
  return new Date(iso).toLocaleDateString('pt-BR', {
    day: '2-digit',
    month: '2-digit',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  });
}

const ETAPA_VALORES = new Set<string>(ETAPAS.map((e) => e.valor));

// StatusImpressoraBluetooth (24/08/2026) — fica entre o título "Pedidos"
// e os botões Lista/Quadro, pedido do William. Serve dois propósitos no
// mesmo lugar: (1) indicador de "buscando..." quando a preferência
// "buscar automaticamente" (Configurações) está ligada — tenta
// reconectar numa impressora já pareada antes, sem abrir o seletor
// (Web Bluetooth não permite abrir o seletor sem clique do usuário,
// então "automático" aqui é reconexão silenciosa, não descoberta de
// impressora nova); (2) botão de conectar/buscar manual — sempre
// disponível no mesmo lugar, pra quem prefere ligar na hora ou não usa
// o automático. Não aparece em navegador sem suporte a Web Bluetooth
// (ex: iOS/Safari) — não tem nada útil a mostrar/fazer ali.
function StatusImpressoraBluetooth() {
  const autoBuscar = useImpressoraStore((state) => state.autoBuscar);
  const [status, setStatus] = useState<'ocioso' | 'buscando' | 'conectado' | 'nao_encontrada'>('ocioso');

  useEffect(() => {
    if (!autoBuscar || !bluetoothSuportado()) return;
    let cancelado = false;
    setStatus('buscando');
    tentarReconectarSilenciosamente().then((conectou) => {
      if (cancelado) return;
      setStatus(conectou ? 'conectado' : 'nao_encontrada');
    });
    return () => {
      cancelado = true;
    };
  }, [autoBuscar]);

  if (!bluetoothSuportado()) return null;

  async function conectarManual() {
    setStatus('buscando');
    try {
      await conectarImpressora();
      setStatus('conectado');
    } catch {
      setStatus('nao_encontrada');
    }
  }

  const rotulo =
    status === 'buscando' ? 'Buscando impressora...' : status === 'conectado' ? '🖨️ Impressora conectada' : '🖨️ Conectar impressora';

  return (
    <button
      onClick={conectarManual}
      disabled={status === 'buscando'}
      title="Impressora Bluetooth pra comandas"
      className={cn(
        'flex shrink-0 items-center gap-1.5 whitespace-nowrap rounded-full px-3 py-1.5 text-xs font-medium transition disabled:cursor-wait',
        status === 'conectado'
          ? 'bg-emerald-500/10 text-emerald-600'
          : status === 'buscando'
          ? 'bg-douro/10 text-douro'
          : 'bg-tinta/5 text-tinta-suave hover:bg-tinta/10'
      )}
    >
      <span
        className={cn(
          'h-1.5 w-1.5 rounded-full',
          status === 'conectado' ? 'bg-emerald-500' : status === 'buscando' ? 'animate-pulse bg-douro' : 'bg-tinta-suave'
        )}
      />
      {rotulo}
    </button>
  );
}

export function Pedidos() {
  const queryClient = useQueryClient();

  // Atualiza sozinho a cada 30s — um pedido novo pode chegar a qualquer
  // momento enquanto o dono está com essa tela aberta.
  const { data: pedidos, isLoading } = useQuery({
    queryKey: ['pedidos'],
    queryFn: listarPedidos,
    refetchInterval: 30_000,
  });
  const { data: loja } = useQuery({ queryKey: ['loja'], queryFn: buscarLoja });

  const [filtro, setFiltro] = useState<FiltroPedido>('todos');
  const [visualizacao, setVisualizacao] = useState<'lista' | 'quadro'>('lista');

  const definirLarguraCompleta = useLayoutAdminStore((state) => state.definirLarguraCompleta);

  // Largura total do <main> — Lista também precisa (a barra de filtros
  // tinha 7+ pills forçando scroll horizontal em max-w-3xl) e Quadro
  // precisa (4 colunas, ~1150px) — então liga pra qualquer visão de
  // Pedidos, não só Quadro. Desliga ao sair da tela (cleanup), pra não
  // vazar largura total pras outras ~15 telas do admin que reaproveitam
  // o mesmo <main>. A lista de pedidos em si mantém uma largura de
  // leitura confortável própria (ver `<ul className="max-w-3xl">`
  // abaixo) mesmo com o <main> liberado.
  useEffect(() => {
    definirLarguraCompleta(true);
    return () => definirLarguraCompleta(false);
  }, [definirLarguraCompleta]);

  // avancarEtapa é compartilhado pela lista e pelo quadro — os dois
  // chamam o mesmo endpoint, só muda onde o botão aparece.
  const mutAvancar = useMutation({
    mutationFn: ({ pedidoId, etapa }: { pedidoId: number; etapa: EtapaPedido }) => atualizarStatusEntrega(pedidoId, etapa),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['pedidos'] }),
  });
  function avancarEtapa(pedido: Pedido) {
    const proxima = proximaEtapa(etapaAtual(pedido));
    if (proxima) mutAvancar.mutate({ pedidoId: pedido.id, etapa: proxima });
  }

  // moverParaEtapa é o que o drag-and-drop chama — diferente de
  // avancarEtapa (que só sabe ir pra próxima etapa da lista), aceita
  // qualquer etapa de destino, porque soltar um card no Kanban pode
  // avançar, voltar ou pular etapa livremente.
  function moverParaEtapa(pedido: Pedido, etapa: EtapaPedido) {
    if (etapa !== etapaAtual(pedido)) mutAvancar.mutate({ pedidoId: pedido.id, etapa });
  }

  // Impressão de comanda via Bluetooth (Fase 10.7) — 100% client-side, sem
  // chamada de backend. Erro aqui é normal na primeira tentativa (usuário
  // cancelou o seletor de pareamento, impressora desligada etc.), avisa
  // com alert() mesmo padrão já usado em Insumos.tsx pra erro de mutação.
  async function handleImprimir(pedido: Pedido) {
    try {
      await imprimirComanda(pedido, loja?.nome ?? 'Comanda');
    } catch (e) {
      alert(e instanceof Error ? e.message : 'Não foi possível imprimir a comanda.');
    }
  }

  const pesoPendenteCount = pedidos?.filter((p) => p.peso_pendente).length ?? 0;

  // O filtro de peso pendente só aparece quando há algum — evita poluir a
  // barra pra lojas que nunca usam o modo "guardar e entregar depois".
  const filtros = pesoPendenteCount > 0
    ? [...filtrosBase, { valor: 'peso_pendente' as const, label: `⚠️ Peso pendente (${pesoPendenteCount})` }]
    : filtrosBase;

  const pedidosFiltrados =
    pedidos?.filter((pedido) => {
      if (filtro === 'todos') return true;
      if (filtro === 'peso_pendente') return pedido.peso_pendente;
      // Etapas de preparo/entrega só existem em pedido pago — os filtros
      // de etapa não devolvem nada de aguardando_pagamento/cancelado.
      if (ETAPA_VALORES.has(filtro)) return pedido.status === 'pago' && etapaAtual(pedido) === filtro;
      return pedido.status === filtro;
    }) ?? [];

  // Quadro (Kanban) só faz sentido pra pedido pago — não tem etapa de
  // preparo antes de pagar. Ignora o filtro de status/peso quando o
  // quadro está ativo (ele já filtra por etapa através das colunas).
  const pedidosPagos = pedidos?.filter((p) => p.status === 'pago') ?? [];

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between gap-3">
        <h1 className="font-display text-2xl tracking-wide text-tinta">Pedidos</h1>
        <StatusImpressoraBluetooth />
        <div className="flex shrink-0 gap-1 rounded-full bg-superficie p-1 shadow-sm">
          <button
            onClick={() => setVisualizacao('lista')}
            className={`rounded-full px-3 py-1 text-xs font-semibold transition ${visualizacao === 'lista' ? 'bg-acento text-texto-claro' : 'text-tinta-suave'}`}
          >
            Lista
          </button>
          <button
            onClick={() => setVisualizacao('quadro')}
            className={`rounded-full px-3 py-1 text-xs font-semibold transition ${visualizacao === 'quadro' ? 'bg-acento text-texto-claro' : 'text-tinta-suave'}`}
          >
            Quadro
          </button>
        </div>
      </div>

      {visualizacao === 'lista' ? (
        <>
          <div className="flex flex-wrap gap-2">
            {filtros.map((item) => (
              <button
                key={item.valor}
                onClick={() => setFiltro(item.valor)}
                className={`shrink-0 rounded-full border-2 px-4 py-1.5 text-sm font-medium transition ${
                  filtro === item.valor
                    ? 'border-acento bg-acento text-texto-claro'
                    : item.valor === 'peso_pendente'
                    ? 'border-amber-300 bg-amber-50 text-amber-800 hover:border-amber-400'
                    : 'border-tinta/15 bg-superficie text-tinta-suave hover:border-tinta/30'
                }`}
              >
                {item.label}
              </button>
            ))}
          </div>

          {isLoading ? (
            <p className="text-tinta-suave">Carregando pedidos...</p>
          ) : pedidosFiltrados.length === 0 ? (
            <p className="text-tinta-suave">Nenhum pedido por aqui ainda.</p>
          ) : (
            <ul className="max-w-3xl space-y-3">
              {pedidosFiltrados.map((pedido) => (
                <PedidoCard key={pedido.id} pedido={pedido} segmentoLoja={loja?.segmento_principal} onAvancar={avancarEtapa} onImprimir={handleImprimir} />
              ))}
            </ul>
          )}
        </>
      ) : (
        <QuadroPedidos pedidos={pedidosPagos} isLoading={isLoading} segmentoLoja={loja?.segmento_principal} onAvancar={avancarEtapa} onMover={moverParaEtapa} onImprimir={handleImprimir} />
      )}
    </div>
  );
}

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
          className={cn('btn-neu-primario btn-neu-sm mt-2 w-full', ESTILO_AVANCAR[etapaAtual(pedido)])}
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

function PedidoCard({ pedido, segmentoLoja, onAvancar, onImprimir }: { pedido: Pedido; segmentoLoja?: TipoProduto; onAvancar: (pedido: Pedido) => void; onImprimir: (pedido: Pedido) => void }) {
  const status = statusInfo[pedido.status];
  // Etapa de preparo/entrega (Fase 10.2) só existe em pedido pago.
  const etapa = pedido.status === 'pago' ? infoEtapa(etapaAtual(pedido)) : null;
  const proxima = pedido.status === 'pago' ? proximaEtapa(etapaAtual(pedido)) : null;

  // Só faz sentido gerenciar entrega (link pra tela de GPS/rastreamento)
  // em pedidos pagos, com modo "entrega", e que ainda não foram marcados
  // como entregues — o botão "Avançar" genérico abaixo é o caminho pra
  // todo mundo, esse link é um atalho a mais só pra quem tem entrega de
  // verdade (compartilha localização).
  const podeGerenciarEntrega =
    pedido.status === 'pago' &&
    pedido.modo_entrega === 'entrega' &&
    pedido.status_entrega !== 'entregue';

  return (
    <li className="rounded-2xl bg-superficie p-4 shadow-sm">
      <div className="flex items-start justify-between gap-3">
        <div>
          <p className="font-medium text-tinta">
            {pedido.cliente_nome} <span className="text-tinta-suave">· #{pedido.id}</span>
          </p>
          <p className="text-sm text-tinta-suave">{pedido.cliente_telefone}</p>
        </div>
        <div className="flex shrink-0 flex-col items-end gap-1">
          <span className={`rounded-full px-3 py-1 text-xs font-semibold ${status.classe}`}>
            {status.label}
          </span>
          {etapa && (
            <span className={`rounded-full px-3 py-1 text-xs font-semibold ${etapa.classe}`}>
              {rotuloEtapa(etapa.valor, pedido.modo_entrega)}
            </span>
          )}
          {pedido.peso_pendente && (
            <span className={`rounded-full px-3 py-1 text-xs font-semibold ${PESO_PENDENTE_CLASSE}`}>
              ⚠️ Peso pendente
            </span>
          )}
        </div>
      </div>

      <div className="mt-3 space-y-1 border-t border-tinta/10 pt-3">
        {pedido.itens.map((item) => (
          <p key={item.id} className="text-sm text-tinta">
            {item.quantidade}x {item.produto_nome}{' '}
            {item.sugestao_produto_id !== null && (
              <span className="rounded-full bg-douro/10 px-1.5 py-0.5 text-xs text-douro">💡 Sugestão</span>
            )}{' '}
            <span className="text-tinta-suave">
              · R$ {(item.preco_unit * item.quantidade).toFixed(2).replace('.', ',')}
            </span>
          </p>
        ))}
        {pedido.combos?.map((combo) => (
          <div key={combo.id} className="text-sm text-tinta">
            <p>
              {combo.quantidade}x {combo.nome}{' '}
              <span className="rounded-full bg-acento/10 px-1.5 py-0.5 text-xs text-acento">{rotuloCombo(segmentoLoja)}</span>{' '}
              <span className="text-tinta-suave">
                · R$ {(combo.preco * combo.quantidade).toFixed(2).replace('.', ',')}
              </span>
            </p>
            <p className="pl-3 text-xs text-tinta-suave">
              {combo.itens.map((item) => `${item.quantidade}x ${item.produto_nome}${item.variacao_nome ? ` (${item.variacao_nome})` : ''}`).join(', ')}
            </p>
          </div>
        ))}
      </div>

      <div className="mt-3 flex items-center justify-between border-t border-tinta/10 pt-3 text-sm">
        <div>
          <span className="text-tinta-suave">
            {pedido.modo_entrega === 'entrega' ? '🛵 Entrega' : '🏪 Retirada'}
          </span>
          {pedido.modo_entrega === 'entrega' && pedido.endereco_entrega && (
            <p className="mt-0.5 text-xs text-tinta-suave">{pedido.endereco_entrega}</p>
          )}
          <p className="mt-0.5 text-xs text-tinta-suave">{formatarData(pedido.data_retirada)}</p>
          {pedido.cupom_codigo && (
            <p className="mt-0.5 text-xs text-emerald-600">
              Cupom {pedido.cupom_codigo} · -R$ {pedido.desconto.toFixed(2).replace('.', ',')}
            </p>
          )}
        </div>
        <span className="font-carimbo font-semibold text-tinta">
          R$ {pedido.total.toFixed(2).replace('.', ',')}
        </span>
      </div>

      {proxima && (
        <button onClick={() => onAvancar(pedido)} className={cn('btn-neu-primario mt-3 w-full', ESTILO_AVANCAR[etapaAtual(pedido)])}>
          Avançar → {rotuloEtapa(proxima, pedido.modo_entrega)}
        </button>
      )}

      {pedido.status === 'pago' && (
        <button onClick={() => onImprimir(pedido)} className="btn-neu-secundario mt-2 w-full">
          🖨️ Imprimir comanda
        </button>
      )}

      {podeGerenciarEntrega && (
        <Link
          to={`/admin/pedidos/${pedido.id}/localizacao`}
          className="btn-neu-secundario mt-2 block text-center"
        >
          {pedido.status_entrega === 'saiu_para_entrega' ? '📍 Gerenciar entrega' : '🛵 Iniciar entrega'}
        </Link>
      )}
    </li>
  );
}