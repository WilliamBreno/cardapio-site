import { useState } from 'react';
import { Link } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import { listarPedidos, buscarLoja } from '../../api/admin';
import type { Pedido, StatusPedido, TipoProduto } from '../../api/types';
import { rotuloCombo } from '../../lib/utils';

const statusInfo: Record<StatusPedido, { label: string; classe: string }> = {
  aguardando_pagamento: { label: 'Aguardando pagamento', classe: 'bg-douro/20 text-douro' },
  pago: { label: 'Pago', classe: 'bg-emerald-100 text-emerald-700' },
  cancelado: { label: 'Cancelado', classe: 'bg-acento/10 text-acento' },
};

const statusEntregaInfo: Record<string, { label: string; classe: string }> = {
  saiu_para_entrega: { label: '🛵 Saiu para entrega', classe: 'bg-douro/20 text-douro' },
  entregue: { label: '✅ Entregue', classe: 'bg-emerald-100 text-emerald-700' },
};

const PESO_PENDENTE_CLASSE = 'bg-amber-100 text-amber-800';

type FiltroPedido = 'todos' | StatusPedido | 'peso_pendente';

const filtrosBase: { valor: FiltroPedido; label: string }[] = [
  { valor: 'todos', label: 'Todos' },
  { valor: 'pago', label: 'Pagos' },
  { valor: 'aguardando_pagamento', label: 'Aguardando' },
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

export function Pedidos() {
  // Atualiza sozinho a cada 30s — um pedido novo pode chegar a qualquer
  // momento enquanto o dono está com essa tela aberta.
  const { data: pedidos, isLoading } = useQuery({
    queryKey: ['pedidos'],
    queryFn: listarPedidos,
    refetchInterval: 30_000,
  });
  const { data: loja } = useQuery({ queryKey: ['loja'], queryFn: buscarLoja });

  const [filtro, setFiltro] = useState<FiltroPedido>('todos');

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
      return pedido.status === filtro;
    }) ?? [];

  return (
    <div className="space-y-6">
      <h1 className="font-display text-2xl tracking-wide text-tinta">Pedidos</h1>

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
            <PedidoCard key={pedido.id} pedido={pedido} segmentoLoja={loja?.segmento_principal} />
          ))}
        </ul>
      )}
    </div>
  );
}

function PedidoCard({ pedido, segmentoLoja }: { pedido: Pedido; segmentoLoja?: TipoProduto }) {
  const status = statusInfo[pedido.status];
  const statusEntrega = pedido.status_entrega ? statusEntregaInfo[pedido.status_entrega] : null;

  // Só faz sentido gerenciar entrega em pedidos pagos, com modo "entrega",
  // e que ainda não foram marcados como entregues.
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
          {statusEntrega && (
            <span className={`rounded-full px-3 py-1 text-xs font-semibold ${statusEntrega.classe}`}>
              {statusEntrega.label}
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

      {podeGerenciarEntrega && (
        <Link
          to={`/admin/pedidos/${pedido.id}/localizacao`}
          className="mt-3 block rounded-full bg-acento px-4 py-2 text-center text-sm font-semibold text-superficie"
        >
          {pedido.status_entrega === 'saiu_para_entrega' ? '📍 Gerenciar entrega' : '🛵 Iniciar entrega'}
        </Link>
      )}
    </li>
  );
}