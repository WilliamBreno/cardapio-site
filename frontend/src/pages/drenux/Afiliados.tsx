import { useState } from 'react';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import {
  listarPendentesPorAfiliado,
  buscarRepassesDoAfiliado,
  marcarRepassesComoPago,
  type PendentePorAfiliado,
} from '../../api/drenux';
import { useDrenuxAdminStore } from '../../store/drenuxAdminStore';

function moeda(v: number) {
  return `R$ ${v.toFixed(2).replace('.', ',')}`;
}

function formatarData(iso: string) {
  return new Date(iso).toLocaleDateString('pt-BR', { day: '2-digit', month: '2-digit', year: 'numeric' });
}

// Tela interna de controle de repasse de comissão de afiliado (Fase 5.5
// do roadmap) — pedidos pagos via Mercado Pago não têm split automático
// de 3 partes, então a comissão fica registrada como pendente aqui até
// o repasse via Pix ser feito manualmente e confirmado nesta tela. Não
// existe login de staff da Drenux ainda, então o acesso é só por um
// secret compartilhado (ver middleware.DrenuxAdminRequired no backend).
export function DrenuxAfiliados() {
  const secret = useDrenuxAdminStore((s) => s.secret);

  if (!secret) {
    return <FormularioSecret />;
  }

  return <PainelRepasses />;
}

function FormularioSecret() {
  const setSecret = useDrenuxAdminStore((s) => s.setSecret);
  const [campo, setCampo] = useState('');

  function entrar(e: React.FormEvent) {
    e.preventDefault();
    if (campo.trim()) setSecret(campo.trim());
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-fundo px-4">
      <form onSubmit={entrar} className="w-full max-w-sm space-y-4 rounded-2xl bg-superficie p-6 shadow-sm">
        <div>
          <h1 className="font-display text-xl tracking-wide text-tinta">Área interna — Drenux</h1>
          <p className="mt-1 text-sm text-tinta-suave">Repasse de comissão de afiliados.</p>
        </div>
        <input
          type="password"
          required
          autoFocus
          value={campo}
          onChange={(e) => setCampo(e.target.value)}
          placeholder="Secret de acesso"
          className="w-full rounded-lg border border-tinta/20 bg-fundo px-3 py-2 text-tinta outline-none focus:border-acento"
        />
        <button type="submit" className="w-full rounded-full bg-acento py-2 text-sm font-semibold text-superficie">
          Entrar
        </button>
      </form>
    </div>
  );
}

function PainelRepasses() {
  const logout = useDrenuxAdminStore((s) => s.logout);
  const queryClient = useQueryClient();

  const { data: pendentes, isLoading, isError } = useQuery({
    queryKey: ['drenux-pendentes'],
    queryFn: listarPendentesPorAfiliado,
    retry: false,
  });

  const [afiliadoAberto, setAfiliadoAberto] = useState<number | null>(null);

  if (isError) {
    return (
      <div className="flex min-h-screen flex-col items-center justify-center gap-3 bg-fundo px-4 text-center">
        <p className="text-tinta-suave">Secret inválido ou expirado.</p>
        <button onClick={logout} className="text-sm font-medium text-acento hover:underline">
          Tentar de novo
        </button>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-fundo px-4 py-8">
      <div className="mx-auto max-w-2xl space-y-6">
        <div className="flex items-center justify-between">
          <h1 className="font-display text-2xl tracking-wide text-tinta">Repasse de afiliados</h1>
          <button onClick={logout} className="text-sm text-tinta-suave hover:text-acento">
            Sair
          </button>
        </div>
        <p className="text-sm text-tinta-suave">
          Comissões de pedidos pagos via Mercado Pago, pendentes de repasse manual via Pix — o Mercado Pago
          ainda não faz split automático de 3 partes (Loja + Drenux + Afiliado).
        </p>

        {isLoading ? (
          <p className="text-tinta-suave">Carregando...</p>
        ) : !pendentes || pendentes.length === 0 ? (
          <p className="text-tinta-suave">Nenhum repasse pendente.</p>
        ) : (
          <ul className="space-y-3">
            {pendentes.map((p) => (
              <AfiliadoCard
                key={p.afiliado_id}
                pendente={p}
                aberto={afiliadoAberto === p.afiliado_id}
                onToggle={() => setAfiliadoAberto(afiliadoAberto === p.afiliado_id ? null : p.afiliado_id)}
                onMarcarPago={() => {
                  queryClient.invalidateQueries({ queryKey: ['drenux-pendentes'] });
                  queryClient.invalidateQueries({ queryKey: ['drenux-repasses', p.afiliado_id] });
                }}
              />
            ))}
          </ul>
        )}
      </div>
    </div>
  );
}

function AfiliadoCard({
  pendente,
  aberto,
  onToggle,
  onMarcarPago,
}: {
  pendente: PendentePorAfiliado;
  aberto: boolean;
  onToggle: () => void;
  onMarcarPago: () => void;
}) {
  const { data: repasses, isLoading } = useQuery({
    queryKey: ['drenux-repasses', pendente.afiliado_id],
    queryFn: () => buscarRepassesDoAfiliado(pendente.afiliado_id),
    enabled: aberto,
  });

  const [selecionados, setSelecionados] = useState<number[]>([]);
  const [marcando, setMarcando] = useState(false);

  function alternarSelecao(id: number) {
    setSelecionados((atual) => (atual.includes(id) ? atual.filter((x) => x !== id) : [...atual, id]));
  }

  async function marcarSelecionados() {
    if (selecionados.length === 0) return;
    setMarcando(true);
    try {
      await marcarRepassesComoPago(selecionados);
      setSelecionados([]);
      onMarcarPago();
    } finally {
      setMarcando(false);
    }
  }

  const pendentesDoAfiliado = repasses?.filter((r) => r.status === 'pendente') ?? [];

  return (
    <li className="rounded-2xl bg-superficie p-4 shadow-sm">
      <button onClick={onToggle} className="flex w-full items-center justify-between gap-3 text-left">
        <div>
          <p className="font-medium text-tinta">{pendente.nome}</p>
          <p className="text-xs text-tinta-suave">
            {pendente.email} · {pendente.quantidade} lançamento{pendente.quantidade > 1 ? 's' : ''}
          </p>
        </div>
        <span className="shrink-0 font-carimbo text-lg font-semibold text-tinta">{moeda(pendente.total_pendente)}</span>
      </button>

      {aberto && (
        <div className="mt-3 space-y-2 border-t border-tinta/10 pt-3">
          {isLoading ? (
            <p className="text-sm text-tinta-suave">Carregando...</p>
          ) : pendentesDoAfiliado.length === 0 ? (
            <p className="text-sm text-tinta-suave">Nada pendente.</p>
          ) : (
            <>
              <ul className="space-y-1.5">
                {pendentesDoAfiliado.map((r) => (
                  <li key={r.id} className="flex items-center gap-2 rounded-lg bg-fundo px-3 py-2">
                    <input
                      type="checkbox"
                      checked={selecionados.includes(r.id)}
                      onChange={() => alternarSelecao(r.id)}
                      className="h-4 w-4 accent-acento"
                    />
                    <div className="min-w-0 flex-1">
                      <p className="truncate text-sm text-tinta">
                        {r.loja?.nome ?? `Loja #${r.loja_id}`} <span className="text-tinta-suave">· pedido #{r.pedido_id}</span>
                      </p>
                      <p className="text-xs text-tinta-suave">{formatarData(r.created_at)}</p>
                    </div>
                    <span className="shrink-0 text-sm font-semibold text-tinta">{moeda(r.valor)}</span>
                  </li>
                ))}
              </ul>
              <button
                onClick={marcarSelecionados}
                disabled={selecionados.length === 0 || marcando}
                className="w-full rounded-full bg-acento px-4 py-2 text-sm font-semibold text-superficie disabled:opacity-60"
              >
                {marcando ? 'Marcando...' : selecionados.length > 0 ? `Marcar ${selecionados.length} como pago` : 'Marcar como pago'}
              </button>
            </>
          )}
        </div>
      )}
    </li>
  );
}
