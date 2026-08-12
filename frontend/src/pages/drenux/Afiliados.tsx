import { useEffect, useState } from 'react';
import { useSearchParams } from 'react-router-dom';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import {
  listarAfiliados,
  buscarRepassesDoAfiliado,
  marcarRepassesComoPago,
  criarAfiliado,
  type AfiliadoComTotais,
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
//
// De propósito, não existe formulário de login visível aqui — qualquer
// um que abrisse essa URL sem saber de nada veria um campo "digite o
// secret", o que já denuncia que existe uma área interna aqui. Em vez
// disso, o acesso é só por link mágico (?key=SECRET): quem não manda
// esse parâmetro (ou não tem o secret salvo de uma visita anterior) cai
// numa página "não encontrada" comum, sem nenhuma pista do que é essa
// rota. Isso é só camuflagem, não é a proteção de verdade — quem chama
// a API direto (o path aparece no bundle JS) ainda esbarra no secret
// checado no backend, que é o que realmente impede o acesso.
export function DrenuxAfiliados() {
  const secret = useDrenuxAdminStore((s) => s.secret);
  const setSecret = useDrenuxAdminStore((s) => s.setSecret);
  const [searchParams, setSearchParams] = useSearchParams();
  const key = searchParams.get('key');

  useEffect(() => {
    if (!key) return;
    setSecret(key);
    const proximos = new URLSearchParams(searchParams);
    proximos.delete('key');
    setSearchParams(proximos, { replace: true });
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [key]);

  if (!secret) {
    return <PaginaNaoEncontrada />;
  }

  return <PainelRepasses />;
}

function PaginaNaoEncontrada() {
  return (
    <div className="flex min-h-screen items-center justify-center bg-fundo px-4 text-center">
      <div>
        <p className="font-display text-2xl text-tinta">404</p>
        <p className="mt-1 text-sm text-tinta-suave">Página não encontrada.</p>
      </div>
    </div>
  );
}

function CriarAfiliadoForm({ onCriado }: { onCriado: () => void }) {
  const [aberto, setAberto] = useState(false);
  const [nome, setNome] = useState('');
  const [email, setEmail] = useState('');
  const [senha, setSenha] = useState('');
  // Percentual "cru" (45), não a fração — converte pra fração (0.45) só
  // na hora de mandar pro backend, que é como domain.Afiliado guarda.
  // Base é o LUCRO LÍQUIDO da comissão (Fase 7.5), não mais o bruto.
  const [comissao, setComissao] = useState('45');
  const [erro, setErro] = useState<string | null>(null);
  const [criado, setCriado] = useState<{ nome: string; codigo: string } | null>(null);
  const [enviando, setEnviando] = useState(false);

  async function criar(e: React.FormEvent) {
    e.preventDefault();
    const comissaoFracao = parseFloat(comissao) / 100;
    if (!comissaoFracao || comissaoFracao <= 0 || comissaoFracao > 1) {
      setErro('Comissão precisa ser um número entre 0 e 100.');
      return;
    }
    setEnviando(true);
    setErro(null);
    try {
      const afiliado = await criarAfiliado({ nome, email, senha, comissao_percentual: comissaoFracao });
      setCriado({ nome: afiliado.nome, codigo: afiliado.codigo });
      setNome('');
      setEmail('');
      setSenha('');
      setComissao('37.6');
      setAberto(false);
      onCriado();
    } catch {
      setErro('Não foi possível criar — confere se o email já não está cadastrado.');
    } finally {
      setEnviando(false);
    }
  }

  if (!aberto) {
    return (
      <div className="space-y-2">
        <button
          onClick={() => { setAberto(true); setCriado(null); }}
          className="w-full rounded-full border border-tinta/20 px-4 py-2 text-sm font-semibold text-tinta hover:border-acento hover:text-acento"
        >
          + Criar afiliado
        </button>
        {criado && (
          <p className="rounded-lg bg-emerald-100 px-3 py-2 text-xs text-emerald-800">
            Afiliado "{criado.nome}" criado — código de indicação: <strong>{criado.codigo}</strong>
          </p>
        )}
      </div>
    );
  }

  return (
    <form onSubmit={criar} className="space-y-3 rounded-2xl bg-superficie p-4 shadow-sm">
      <p className="text-sm font-medium text-tinta">Criar afiliado</p>
      <input
        required
        autoFocus
        value={nome}
        onChange={(e) => setNome(e.target.value)}
        placeholder="Nome"
        className="w-full rounded-lg border border-tinta/20 bg-fundo px-3 py-2 text-sm text-tinta outline-none focus:border-acento"
      />
      <input
        type="email"
        required
        value={email}
        onChange={(e) => setEmail(e.target.value)}
        placeholder="Email"
        className="w-full rounded-lg border border-tinta/20 bg-fundo px-3 py-2 text-sm text-tinta outline-none focus:border-acento"
      />
      <input
        type="password"
        required
        minLength={6}
        value={senha}
        onChange={(e) => setSenha(e.target.value)}
        placeholder="Senha (mínimo 6 caracteres)"
        className="w-full rounded-lg border border-tinta/20 bg-fundo px-3 py-2 text-sm text-tinta outline-none focus:border-acento"
      />
      <div>
        <label className="mb-1 block text-xs font-medium text-tinta-suave">
          Comissão desse afiliado — % do lucro líquido da comissão que ele recebe (padrão: 45%)
        </label>
        <div className="flex items-center gap-2">
          <input
            type="number"
            required
            step="0.1"
            min="0.1"
            max="100"
            value={comissao}
            onChange={(e) => setComissao(e.target.value)}
            className="w-full rounded-lg border border-tinta/20 bg-fundo px-3 py-2 text-sm text-tinta outline-none focus:border-acento"
          />
          <span className="text-sm text-tinta-suave">%</span>
        </div>
      </div>
      {erro && <p className="text-xs text-acento">{erro}</p>}
      <div className="flex gap-2">
        <button
          type="button"
          onClick={() => setAberto(false)}
          className="rounded-full border border-tinta/20 px-4 py-2 text-sm font-semibold text-tinta"
        >
          Cancelar
        </button>
        <button
          type="submit"
          disabled={enviando}
          className="flex-1 rounded-full bg-acento px-4 py-2 text-sm font-semibold text-superficie disabled:opacity-60"
        >
          {enviando ? 'Criando...' : 'Criar'}
        </button>
      </div>
    </form>
  );
}

function PainelRepasses() {
  const logout = useDrenuxAdminStore((s) => s.logout);
  const queryClient = useQueryClient();

  const { data: afiliados, isLoading, isError } = useQuery({
    queryKey: ['drenux-afiliados'],
    queryFn: listarAfiliados,
    retry: false,
  });

  const [afiliadoAberto, setAfiliadoAberto] = useState<number | null>(null);

  if (isError) {
    return (
      <div className="flex min-h-screen flex-col items-center justify-center gap-3 bg-fundo px-4 text-center">
        <p className="text-tinta-suave">Secret inválido, expirado ou bloqueado por tentativas erradas.</p>
        <p className="text-xs text-tinta-suave">Acesse de novo pelo link com ?key= pra entrar com o secret certo.</p>
        <button onClick={logout} className="text-sm font-medium text-acento hover:underline">
          Sair
        </button>
      </div>
    );
  }

  function invalidar() {
    queryClient.invalidateQueries({ queryKey: ['drenux-afiliados'] });
  }

  return (
    <div className="min-h-screen bg-fundo px-4 py-8">
      <div className="mx-auto max-w-2xl space-y-6">
        <div className="flex items-center justify-between">
          <h1 className="font-display text-2xl tracking-wide text-tinta">Afiliados</h1>
          <button onClick={logout} className="text-sm text-tinta-suave hover:text-acento">
            Sair
          </button>
        </div>
        <p className="text-sm text-tinta-suave">
          Todo afiliado cadastrado, com a comissão negociada e quanto já foi pago/está pendente de pedidos
          pagos via Mercado Pago (repasse manual via Pix — sem split automático de 3 partes ainda).
        </p>

        <CriarAfiliadoForm onCriado={invalidar} />

        {isLoading ? (
          <p className="text-tinta-suave">Carregando...</p>
        ) : !afiliados || afiliados.length === 0 ? (
          <p className="text-tinta-suave">Nenhum afiliado cadastrado ainda.</p>
        ) : (
          <ul className="space-y-3">
            {afiliados.map((a) => (
              <AfiliadoCard
                key={a.afiliado_id}
                afiliado={a}
                aberto={afiliadoAberto === a.afiliado_id}
                onToggle={() => setAfiliadoAberto(afiliadoAberto === a.afiliado_id ? null : a.afiliado_id)}
                onMarcarPago={() => {
                  invalidar();
                  queryClient.invalidateQueries({ queryKey: ['drenux-repasses', a.afiliado_id] });
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
  afiliado,
  aberto,
  onToggle,
  onMarcarPago,
}: {
  afiliado: AfiliadoComTotais;
  aberto: boolean;
  onToggle: () => void;
  onMarcarPago: () => void;
}) {
  const { data: repasses, isLoading } = useQuery({
    queryKey: ['drenux-repasses', afiliado.afiliado_id],
    queryFn: () => buscarRepassesDoAfiliado(afiliado.afiliado_id),
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
      <button onClick={onToggle} className="flex w-full items-start justify-between gap-3 text-left">
        <div>
          <p className="font-medium text-tinta">{afiliado.nome}</p>
          <p className="text-xs text-tinta-suave">
            {afiliado.email} · código <span className="font-carimbo">{afiliado.codigo}</span> · comissão{' '}
            {(afiliado.comissao_percentual * 100).toFixed(1)}%
          </p>
          <p className="text-xs text-tinta-suave">
            {afiliado.quantidade} lançamento{afiliado.quantidade !== 1 ? 's' : ''}
          </p>
        </div>
        <div className="shrink-0 text-right">
          <p className="font-carimbo text-lg font-semibold text-tinta">{moeda(afiliado.total_pendente)}</p>
          <p className="text-[11px] text-tinta-suave">pendente</p>
          {afiliado.total_pago > 0 && <p className="mt-1 text-[11px] text-emerald-700">{moeda(afiliado.total_pago)} já pago</p>}
        </div>
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
                        {r.loja?.nome ?? `Loja #${r.loja_id}`}{' '}
                        <span className="text-tinta-suave">
                          · {r.tipo === 'bonus_ativacao' ? 'bônus de ativação' : `pedido #${r.pedido_id}`}
                        </span>
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
