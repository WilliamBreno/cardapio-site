import { Link } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import { listarSolicitacoes } from '../../api/admin';
import type { SolicitacaoEntrega } from '../../api/types';

const statusEntregaInfo: Record<string, { label: string; classe: string }> = {
  saiu_para_entrega: { label: '🛵 Saiu para entrega', classe: 'bg-douro/20 text-douro' },
  entregue: { label: '✅ Entregue', classe: 'bg-emerald-100 text-emerald-700' },
};

const PESO_PENDENTE_CLASSE = 'bg-amber-100 text-amber-800';

function formatarData(iso: string): string {
  return new Date(iso).toLocaleDateString('pt-BR', {
    day: '2-digit', month: '2-digit', year: 'numeric', hour: '2-digit', minute: '2-digit',
  });
}

export function Solicitacoes() {
  const { data: solicitacoes, isLoading } = useQuery({
    queryKey: ['solicitacoes'],
    queryFn: listarSolicitacoes,
    refetchInterval: 30_000,
  });

  const pesoPendenteCount = solicitacoes?.filter((s) => s.peso_pendente).length ?? 0;

  return (
    <div className="space-y-6">
      <div>
        <h1 className="font-display text-2xl tracking-wide text-tinta">Itens guardados — entregas</h1>
        <p className="mt-1 text-sm text-tinta-suave">
          Pedidos de entrega de itens que já foram pagos e ficaram guardados. O frete já está pago quando aparecem aqui.
        </p>
      </div>

      {pesoPendenteCount > 0 && (
        <div className={`rounded-xl px-4 py-2 text-sm font-medium ${PESO_PENDENTE_CLASSE}`}>
          ⚠️ {pesoPendenteCount} entrega{pesoPendenteCount > 1 ? 's' : ''} com peso pendente — o frete pode estar sub-cobrado.
        </div>
      )}

      {isLoading ? (
        <p className="text-tinta-suave">Carregando...</p>
      ) : !solicitacoes || solicitacoes.length === 0 ? (
        <p className="text-tinta-suave">Nenhuma entrega de itens guardados pendente.</p>
      ) : (
        <ul className="space-y-3">
          {solicitacoes.map((solicitacao) => (
            <SolicitacaoCard key={solicitacao.id} solicitacao={solicitacao} />
          ))}
        </ul>
      )}
    </div>
  );
}

function SolicitacaoCard({ solicitacao }: { solicitacao: SolicitacaoEntrega }) {
  const statusEntrega = solicitacao.status_entrega ? statusEntregaInfo[solicitacao.status_entrega] : null;
  const podeGerenciarEntrega = solicitacao.status_entrega !== 'entregue';

  return (
    <li className="rounded-2xl bg-superficie p-4 shadow-sm">
      <div className="flex items-start justify-between gap-3">
        <div>
          <p className="font-medium text-tinta">
            {solicitacao.cliente_nome} <span className="text-tinta-suave">· #{solicitacao.id}</span>
          </p>
          <p className="text-sm text-tinta-suave">{solicitacao.cliente_telefone}</p>
        </div>
        <div className="flex shrink-0 flex-col items-end gap-1">
          {statusEntrega && (
            <span className={`rounded-full px-3 py-1 text-xs font-semibold ${statusEntrega.classe}`}>
              {statusEntrega.label}
            </span>
          )}
          {solicitacao.peso_pendente && (
            <span className={`rounded-full px-3 py-1 text-xs font-semibold ${PESO_PENDENTE_CLASSE}`}>
              ⚠️ Peso pendente
            </span>
          )}
        </div>
      </div>

      <div className="mt-3 space-y-1 border-t border-tinta/10 pt-3">
        {solicitacao.itens.map((item) => (
          <p key={item.id} className="text-sm text-tinta">
            {item.quantidade}x {item.produto_nome}
            {item.variacao_nome && <span className="text-tinta-suave"> ({item.variacao_nome})</span>}
          </p>
        ))}
      </div>

      <div className="mt-3 flex items-center justify-between border-t border-tinta/10 pt-3 text-sm">
        <div>
          <p className="text-tinta-suave">{solicitacao.endereco_entrega}</p>
          <p className="mt-0.5 text-xs text-tinta-suave">
            {solicitacao.distancia_km.toFixed(1)} km · {solicitacao.tipo_calculo === 'regional' ? 'mesma região' : 'estimado (fora da região)'} · {formatarData(solicitacao.created_at)}
          </p>
        </div>
        <span className="font-carimbo font-semibold text-tinta">
          R$ {solicitacao.valor_frete.toFixed(2).replace('.', ',')}
        </span>
      </div>

      {podeGerenciarEntrega && (
        <Link
          to={`/admin/solicitacoes/${solicitacao.id}/localizacao`}
          className="mt-3 block rounded-full bg-acento px-4 py-2 text-center text-sm font-semibold text-texto-claro"
        >
          {solicitacao.status_entrega === 'saiu_para_entrega' ? '📍 Gerenciar entrega' : '🛵 Iniciar entrega'}
        </Link>
      )}
    </li>
  );
}
