import { useQuery } from '@tanstack/react-query';
import { buscarHistoricoCliente } from '../../api/admin';

function formatarData(iso: string): string {
  return new Date(iso).toLocaleDateString('pt-BR', {
    day: '2-digit',
    month: '2-digit',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  });
}

// HistoricoClienteModal (Fase 10.6) — aberto ao clicar num cliente da
// lista de top clientes em Inicio.tsx. Reaproveita GET
// /admin/clientes/:telefone/pedidos, que é a mesma consulta do "Meus
// pedidos" público (ListarPorTelefone), só que sem exigir o telefone
// como senha — o dono já está autenticado.
export function HistoricoClienteModal({ nome, telefone, onFechar }: { nome: string; telefone: string; onFechar: () => void }) {
  const { data: pedidos, isLoading } = useQuery({
    queryKey: ['historico-cliente', telefone],
    queryFn: () => buscarHistoricoCliente(telefone),
  });

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 px-4" onClick={onFechar}>
      <div
        className="max-h-[85vh] w-full max-w-md overflow-y-auto rounded-2xl bg-superficie p-5 shadow-lg"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="mb-4 flex items-start justify-between">
          <div>
            <h2 className="font-display text-lg tracking-wide text-tinta">{nome}</h2>
            <p className="text-sm text-tinta-suave">{telefone}</p>
          </div>
          <button onClick={onFechar} className="text-tinta-suave hover:text-tinta" aria-label="Fechar">✕</button>
        </div>

        {isLoading ? (
          <p className="text-sm text-tinta-suave">Carregando...</p>
        ) : !pedidos || pedidos.length === 0 ? (
          <p className="text-sm text-tinta-suave">Nenhum pedido pago encontrado pra esse cliente.</p>
        ) : (
          <ul className="space-y-3">
            {pedidos.map((pedido) => (
              <li key={pedido.id} className="rounded-xl bg-fundo p-3">
                <div className="flex items-center justify-between text-sm">
                  <span className="font-medium text-tinta">#{pedido.id}</span>
                  <span className="font-carimbo font-semibold text-tinta">
                    R$ {pedido.total.toFixed(2).replace('.', ',')}
                  </span>
                </div>
                <p className="text-xs text-tinta-suave">{formatarData(pedido.data_retirada)}</p>
                <p className="mt-1 text-xs text-tinta-suave">
                  {pedido.itens.map((item) => `${item.quantidade}x ${item.produto_nome}`).join(', ')}
                  {pedido.combos && pedido.combos.length > 0 && (pedido.itens.length > 0 ? ', ' : '') + pedido.combos.map((c) => `${c.quantidade}x ${c.nome}`).join(', ')}
                </p>
              </li>
            ))}
          </ul>
        )}
      </div>
    </div>
  );
}
