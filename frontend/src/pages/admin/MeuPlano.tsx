import { useMemo, useState } from 'react';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { buscarLoja, buscarDashboard, mudarPlano, cancelarMudancaAgendada } from '../../api/admin';
import { Card, CardHeader, CardTitle, CardDescription, CardContent, CardFooter } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Slider } from '@/components/ui/slider';
import { NumberTicker } from '@/components/ui/number-ticker';
import { PLANOS, custoPlano, taxaEfetivaPlano, NOME_PLANO } from '../../lib/planos';

function fmt(v: number) {
  return v.toLocaleString('pt-BR', { style: 'currency', currency: 'BRL', maximumFractionDigits: 0 });
}

export function MeuPlano() {
  const queryClient = useQueryClient();
  const { data: loja } = useQuery({ queryKey: ['loja'], queryFn: buscarLoja });
  const { data: dashboard } = useQuery({ queryKey: ['dashboard'], queryFn: buscarDashboard });

  // Pré-preenche com o faturamento real do mês, mas o dono pode ajustar
  // o slider pra simular outros cenários.
  const [faturamento, setFaturamento] = useState<number | null>(null);
  const faturamentoUsado = faturamento ?? dashboard?.total_mes ?? 0;

  const [trocando, setTrocando] = useState<string | null>(null);
  const [cancelando, setCancelando] = useState(false);
  const [erro, setErro] = useState<string | null>(null);

  const custos = useMemo(
    () =>
      PLANOS.map((p) => ({
        ...p,
        valorTaxa: custoPlano(p, faturamentoUsado) - p.mensal,
        total: custoPlano(p, faturamentoUsado),
      })),
    [faturamentoUsado]
  );
  const menorCusto = Math.min(...custos.filter((c) => c.faixas.length > 0).map((c) => c.total));

  async function escolherPlano(planoId: string) {
    setErro(null);
    setTrocando(planoId);
    try {
      const resultado = await mudarPlano(planoId as 'start' | 'basic' | 'pro' | 'scale');
      if (resultado.checkout_url) {
        window.location.href = resultado.checkout_url;
        return;
      }
      queryClient.invalidateQueries({ queryKey: ['loja'] });
    } catch {
      setErro('Não foi possível trocar de plano agora. Tenta de novo em instantes.');
    } finally {
      setTrocando(null);
    }
  }

  async function desfazerAgendamento() {
    setCancelando(true);
    try {
      await cancelarMudancaAgendada();
      queryClient.invalidateQueries({ queryKey: ['loja'] });
    } catch {
      setErro('Não foi possível cancelar a mudança agendada.');
    } finally {
      setCancelando(false);
    }
  }

  if (!loja) return <p className="text-tinta-suave">Carregando...</p>;

  return (
    <div className="space-y-6">
      <h1 className="font-display text-2xl tracking-wide text-tinta">Meu Plano</h1>

      {/* Plano atual */}
      <Card className="border-none bg-superficie ring-1 ring-tinta/10">
        <CardHeader>
          <CardDescription className="text-tinta-suave">Plano atual</CardDescription>
          <CardTitle className="font-display text-2xl tracking-wide text-tinta">{NOME_PLANO[loja.plano] ?? 'Start'}</CardTitle>
        </CardHeader>
        {loja.plano_agendado && (
          <CardContent>
            <div className="flex items-center justify-between gap-3 rounded-lg bg-douro/10 px-4 py-3">
              <p className="text-sm text-tinta">
                Você vai mudar pro plano <strong>{NOME_PLANO[loja.plano_agendado]}</strong> na próxima renovação.
              </p>
              <button
                type="button"
                onClick={desfazerAgendamento}
                disabled={cancelando}
                className="rounded-full border border-tinta/20 px-3 py-1.5 text-sm font-semibold text-tinta disabled:opacity-60"
              >
                {cancelando ? 'Cancelando...' : 'Cancelar'}
              </button>
            </div>
          </CardContent>
        )}
      </Card>

      {/* Calculadora */}
      <Card className="border-none bg-superficie ring-1 ring-tinta/10">
        <CardHeader>
          <CardDescription className="text-tinta-suave">
            {dashboard ? 'Faturamento do mês (ajuste pra simular outros cenários)' : 'Simule seu faturamento mensal'}
          </CardDescription>
          <CardTitle className="font-carimbo text-2xl font-semibold text-acento">
            R$ <NumberTicker value={faturamentoUsado} className="text-tinta dark:text-tinta" />
          </CardTitle>
        </CardHeader>
        <CardContent>
          <Slider
            value={[faturamentoUsado]}
            onValueChange={(v) => setFaturamento(Array.isArray(v) ? v[0] : v)}
            min={0}
            max={20000}
            step={100}
          />
        </CardContent>
      </Card>

      {erro && <p className="text-sm text-acento">{erro}</p>}

      {/* Cards de plano */}
      <div className="space-y-4">
        {custos.map((p) => {
          const semSplit = p.faixas.length === 0;
          const ehAtual = p.id === loja.plano;
          const recomendado = !semSplit && p.total === menorCusto && !ehAtual;
          const efetivo = taxaEfetivaPlano(p, faturamentoUsado) * 100;

          return (
            <Card
              key={p.id}
              className={`border-none bg-superficie ring-1 ${recomendado ? 'ring-2 ring-acento' : 'ring-tinta/10'}`}
            >
              <CardHeader>
                <div className="flex items-start justify-between">
                  <div>
                    <CardTitle className="font-display text-lg tracking-wide text-tinta">{p.nome}</CardTitle>
                    <CardDescription className="text-tinta-suave">{p.desc}</CardDescription>
                  </div>
                  {ehAtual && <Badge className="bg-douro/20 text-douro">Plano atual</Badge>}
                  {recomendado && <Badge className="bg-acento/10 text-acento">Mais barato pra você</Badge>}
                </div>
              </CardHeader>

              <CardContent className="space-y-4">
                <div>
                  <p className="mb-2 text-[11px] font-medium uppercase tracking-wide text-tinta-suave">
                    Valores fixos do plano
                  </p>
                  <div className="grid grid-cols-2 gap-2">
                    <div className="rounded-lg bg-fundo px-3 py-2.5">
                      <p className="text-xs text-tinta-suave">Mensalidade</p>
                      <p className="font-carimbo text-lg font-semibold text-tinta">{p.mensal === 0 ? 'R$ 0' : fmt(p.mensal)}</p>
                    </div>
                    <div className="rounded-lg bg-fundo px-3 py-2.5">
                      <p className="text-xs text-tinta-suave">Taxa por pedido</p>
                      <p className="font-carimbo text-lg font-semibold text-tinta">
                        {semSplit ? 'Sem split' : `a partir de ${(p.faixas[0].taxa * 100).toFixed(1)}%`}
                      </p>
                    </div>
                  </div>
                </div>

                <div>
                  <p className="mb-2 text-[11px] font-medium uppercase tracking-wide text-tinta-suave">
                    Projeção com o faturamento simulado
                  </p>
                  <div className="flex items-center justify-between border-t border-tinta/10 pt-3">
                    <div>
                      <p className="text-xs text-tinta-suave">
                        {semSplit ? 'Sem taxa de pagamento' : `${fmt(p.valorTaxa)} de taxa`}
                        {' + '}
                        {p.mensal === 0 ? 'R$ 0' : fmt(p.mensal)} de mensalidade
                      </p>
                      <p className="font-carimbo text-xl font-semibold text-tinta">
                        R$ <NumberTicker value={p.total} className="text-tinta dark:text-tinta" />
                        <span className="text-sm font-normal text-tinta-suave"> /mês*</span>
                      </p>
                    </div>
                    {!semSplit && (
                      <span className="rounded-lg bg-fundo px-3 py-1.5 text-sm font-semibold text-tinta">
                        {efetivo.toFixed(1)}%
                      </span>
                    )}
                  </div>
                </div>
              </CardContent>

              {!ehAtual && (
                <CardFooter className="border-tinta/10 bg-transparent">
                  <button
                    type="button"
                    onClick={() => escolherPlano(p.id)}
                    disabled={trocando === p.id}
                    className="w-full rounded-full bg-acento px-4 py-2 text-sm font-semibold text-white transition hover:bg-acento/90 disabled:opacity-60"
                  >
                    {trocando === p.id ? 'Processando...' : `Mudar pro ${p.nome}`}
                  </button>
                </CardFooter>
              )}
            </Card>
          );
        })}
      </div>

      {/* Aviso */}
      <div className="flex items-start gap-2 rounded-xl bg-superficie px-4 py-3 ring-1 ring-tinta/10">
        <span className="text-tinta-suave">ⓘ</span>
        <p className="text-xs text-tinta-suave">
          Mensalidade e taxa são os valores fixos de cada plano. O total mostrado é uma projeção com base no
          faturamento simulado acima — o valor real muda conforme suas vendas no mês.
        </p>
      </div>
    </div>
  );
}
