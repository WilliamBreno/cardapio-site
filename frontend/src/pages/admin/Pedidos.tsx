import { useState } from 'react';
import { Link } from 'react-router-dom';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { listarPedidos, buscarLoja } from '../../api/admin';
import { atualizarStatusEntrega, type EtapaPedido } from '../../api/rastreamento';
import type { Pedido, StatusPedido, TipoProduto } from '../../api/types';
import { rotuloCombo } from '../../lib/utils';
import { imprimirComanda } from '../../lib/impressoraBluetooth';

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
  { valor: 'aguardando_pagamento', label: 'Aguardando' },
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
        <div className="flex shrink-0 gap-1 rounded-full bg-superficie p-1 shadow-sm">
          <button
            onClick={() => setVisualizacao('lista')}
            className={`rounded-full px-3 py-1 text-xs font-semibold transition ${visualizacao === 'lista' ? 'bg-acento text-superficie' : 'text-tinta-suave'}`}
          >
            Lista
          </button>
          <button
            onClick={() => setVisualizacao('quadro')}
            className={`rounded-full px-3 py-1 text-xs font-semibold transition ${visualizacao === 'quadro' ? 'bg-acento text-superficie' : 'text-tinta-suave'}`}
          >
            Quadro
          </button>
        </div>
      </div>

      {visualizacao === 'lista' ? (
        <>
          <div className="flex gap-2 overflow-x-auto">
            {filtros.map((item) => (
              <button
                key={item.valor}
                onClick={() => setFiltro(item.valor)}
                className={`shrink-0 rounded-full border-2 px-4 py-1.5 text-sm font-medium transition ${
                  filtro === item.valor
                    ? 'border-acento bg-acento text-superficie'
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
            <ul className="space-y-3">
              {pedidosFiltrados.map((pedido) => (
                <PedidoCard key={pedido.id} pedido={pedido} segmentoLoja={loja?.segmento_principal} onAvancar={avancarEtapa} onImprimir={handleImprimir} />
              ))}
            </ul>
          )}
        </>
      ) : (
        <QuadroPedidos pedidos={pedidosPagos} isLoading={isLoading} onAvancar={avancarEtapa} onImprimir={handleImprimir} />
      )}
    </div>
  );
}

// QuadroPedidos é a visão "dinâmica e interativa" (Fase 10.2) — uma
// coluna por etapa, cada card com um botão único de avançar. Sem
// drag-and-drop de propósito (mais frágil de acertar bem e não foi
// pedido explicitamente) — "um clique" já é bem atendido pelo botão.
function QuadroPedidos({ pedidos, isLoading, onAvancar, onImprimir }: { pedidos: Pedido[]; isLoading: boolean; onAvancar: (pedido: Pedido) => void; onImprimir: (pedido: Pedido) => void }) {
  if (isLoading) return <p className="text-tinta-suave">Carregando pedidos...</p>;
  if (pedidos.length === 0) return <p className="text-tinta-suave">Nenhum pedido pago ainda.</p>;

  return (
    <div className="flex gap-4 overflow-x-auto pb-2">
      {ETAPAS.map((etapa) => {
        const pedidosDaEtapa = pedidos.filter((p) => etapaAtual(p) === etapa.valor);
        return (
          <div key={etapa.valor} className="w-72 shrink-0 space-y-3">
            <div className={`rounded-full px-3 py-1.5 text-center text-xs font-semibold ${etapa.classe}`}>
              {etapa.rotuloEntrega === etapa.rotuloRetirada ? etapa.rotuloEntrega : `${etapa.rotuloEntrega} / ${etapa.rotuloRetirada}`}
              {' · '}{pedidosDaEtapa.length}
            </div>
            <div className="space-y-2">
              {pedidosDaEtapa.map((pedido) => {
                const proxima = proximaEtapa(etapaAtual(pedido));
                return (
                  <div key={pedido.id} className="rounded-xl bg-superficie p-3 shadow-sm">
                    <p className="text-sm font-medium text-tinta">
                      {pedido.cliente_nome} <span className="text-tinta-suave">· #{pedido.id}</span>
                    </p>
                    <p className="text-xs text-tinta-suave">
                      {pedido.modo_entrega === 'entrega' ? '🛵 Entrega' : '🏪 Retirada'} · R$ {pedido.total.toFixed(2).replace('.', ',')}
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
        <button onClick={() => onAvancar(pedido)} className="btn-neu-primario mt-3 w-full">
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