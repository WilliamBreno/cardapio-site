import { useState } from 'react';
import { Link } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import { buscarDashboard, buscarLoja, listarProdutos, statusMercadoPago } from '../../api/admin';
import { PLANOS, planoMaisBarato, custoPlano, NOME_PLANO } from '../../lib/planos';
import { HistoricoClienteModal } from '../../components/admin/HistoricoClienteModal';
import {
  LineChart, Line, BarChart, Bar,
  XAxis, YAxis, CartesianGrid, Tooltip,
  ResponsiveContainer,
} from 'recharts';

function moeda(v: number) {
  return `R$ ${v.toFixed(2).replace('.', ',')}`;
}

// Tradução de chave crua (Fase 10.6) pra rótulo amigável — mesmo espírito
// de Pedidos.tsx. Chave desconhecida cai no fallback (capitaliza a
// própria chave), pra não sumir do gráfico se o Mercado Pago mandar um
// payment_type_id que a gente ainda não mapeou.
const ROTULO_TIPO_ENTREGA: Record<string, string> = {
  retirada: '🏪 Retirada',
  entrega: '🛵 Entrega',
  guardar: '📦 Guardar e entregar depois',
};

const ROTULO_FORMA_PAGAMENTO: Record<string, string> = {
  pix: 'Pix',
  credit_card: 'Cartão de crédito',
  debit_card: 'Cartão de débito',
  ticket: 'Boleto',
  account_money: 'Saldo Mercado Pago',
  bank_transfer: 'Transferência bancária',
};

function rotuloChave(chave: string, mapa: Record<string, string>) {
  return mapa[chave] ?? chave.charAt(0).toUpperCase() + chave.slice(1);
}

// LIMITE_PEDIDOS_START espelha domain.LimitePedidosStart no backend (Fase
// 7.3) — só pra exibição, o limite de verdade é aplicado no backend.
const LIMITE_PEDIDOS_START = 30;

// PassoOnboarding é uma linha do checklist "Primeiros passos" — feito
// (com risco) ou pendente (clicável, leva pra tela de resolver).
function PassoOnboarding({ feito, titulo, descricao, to, cta }: { feito: boolean; titulo: string; descricao: string; to: string; cta: string }) {
  if (feito) {
    return (
      <div className="flex items-center gap-2.5 px-1 py-1 text-sm text-tinta-suave">
        <span className="flex h-5 w-5 shrink-0 items-center justify-center rounded-full bg-emerald-100 text-xs text-emerald-700">✓</span>
        <span className="line-through opacity-70">{titulo}</span>
      </div>
    );
  }
  return (
    <Link
      to={to}
      className="flex items-center justify-between gap-3 rounded-xl bg-superficie px-3 py-2.5 shadow-sm transition hover:-translate-y-0.5"
    >
      <div className="flex items-center gap-2.5">
        <span className="h-5 w-5 shrink-0 rounded-full border-2 border-acento/50" />
        <div>
          <p className="text-sm font-medium text-tinta">{titulo}</p>
          <p className="text-xs text-tinta-suave">{descricao}</p>
        </div>
      </div>
      <span className="shrink-0 text-xs font-semibold text-acento">{cta} →</span>
    </Link>
  );
}

export function Inicio() {
  const { data, isLoading } = useQuery({
    queryKey: ['dashboard'],
    queryFn: buscarDashboard,
    refetchInterval: 60_000, // atualiza a cada 1 min
  });
  const { data: loja } = useQuery({ queryKey: ['loja'], queryFn: buscarLoja });
  const { data: produtos } = useQuery({ queryKey: ['produtos'], queryFn: listarProdutos });
  const { data: mercadoPagoStatus } = useQuery({ queryKey: ['mercadopago-status'], queryFn: statusMercadoPago });

  const [clienteSelecionado, setClienteSelecionado] = useState<{ nome: string; telefone: string } | null>(null);

  if (isLoading) return <p className="text-tinta-suave">Carregando...</p>;
  if (!data) return null;

  const receita7Dias = data.receita_7_dias ?? [];
  const receita4Semanas = data.receita_4_semanas ?? [];
  const topProdutos = data.top_produtos ?? [];
  const topClientesPorPedidos = data.top_clientes_por_pedidos ?? [];
  const topClientesPorValor = data.top_clientes_por_valor ?? [];
  const tiposEntrega = data.tipos_entrega ?? [];
  const formasPagamento = data.formas_pagamento ?? [];
  const totalSemana = data.total_semana ?? 0;
  const totalMes = data.total_mes ?? 0;
  const pedidosSemana = data.pedidos_semana ?? 0;

  // Alerta proativo: mesma conta da calculadora de "Meu Plano", só que
  // aqui aparece sem o lojista precisar entrar na tela pra ver.
  const planoAtual = loja ? PLANOS.find((p) => p.id === loja.plano) : undefined;
  const recomendado = loja ? planoMaisBarato(totalMes) : null;
  const economiaMensal = recomendado && planoAtual
    ? custoPlano(planoAtual, totalMes) - custoPlano(recomendado, totalMes)
    : 0;
  const mostrarAlertaPlano = !!(recomendado && planoAtual && recomendado.id !== planoAtual.id && economiaMensal > 0);

  // Checklist "Primeiros passos" — proativo, não espera um cliente
  // esbarrar no checkout quebrado pra avisar (ver conversa sobre isso).
  // Só aparece enquanto tiver algo pendente; some sozinho quando os dois
  // passos estiverem feitos. Espera as duas queries carregarem antes de
  // decidir se mostra, pra não piscar um passo "pendente" que na
  // verdade já foi feito só porque a resposta ainda não chegou.
  const pagamentoConectado = mercadoPagoStatus?.mercadopago_conectado ?? false;
  const temProdutos = (produtos?.length ?? 0) > 0;
  const onboardingCarregado = mercadoPagoStatus !== undefined && produtos !== undefined;
  const mostrarOnboarding = onboardingCarregado && (!pagamentoConectado || !temProdutos);

  return (
    <div className="space-y-6">
      <h1 className="font-display text-2xl tracking-wide text-tinta">Visão geral</h1>

      {mostrarOnboarding && (
        <div className="space-y-2 rounded-2xl border border-acento/30 bg-acento/5 p-4">
          <p className="text-sm font-semibold text-tinta">Primeiros passos</p>
          <PassoOnboarding
            feito={pagamentoConectado}
            titulo="Conectar Mercado Pago"
            descricao="O mais importante — sem isso, nenhum cliente consegue pagar um pedido."
            to="/admin/configuracoes"
            cta="Conectar"
          />
          <PassoOnboarding
            feito={temProdutos}
            titulo="Cadastrar produtos"
            descricao="Seu cardápio ainda está vazio."
            to="/admin/produtos"
            cta="Cadastrar"
          />
        </div>
      )}

      {mostrarAlertaPlano && recomendado && (
        <Link
          to="/admin/meu-plano"
          className="block rounded-2xl border border-acento/30 bg-acento/5 p-4 shadow-sm transition hover:border-acento/50"
        >
          <p className="text-sm font-medium text-acento">
            Com o faturamento deste mês ({moeda(totalMes)}), o plano {NOME_PLANO[recomendado.id]} sai mais barato pra você — economize {moeda(economiaMensal)}/mês.
          </p>
          <p className="mt-1 text-xs text-tinta-suave">Ver em Meu Plano →</p>
        </Link>
      )}

      {/* Limite de pedidos do plano Start (Fase 7.3) — contador sempre
          visível, nunca só quando perto do limite. */}
      {loja?.plano === 'start' && (
        <Link
          to="/admin/meu-plano"
          className={`block rounded-2xl border p-4 shadow-sm transition ${
            loja.limite_start_bloqueado
              ? 'border-acento/50 bg-acento/10 hover:border-acento/70'
              : 'border-tinta/10 bg-superficie hover:border-tinta/20'
          }`}
        >
          <p className="text-sm font-medium text-tinta">
            {loja.pedidos_mes_atual}/{LIMITE_PEDIDOS_START} pedidos este mês
          </p>
          {loja.limite_start_bloqueado ? (
            <p className="mt-1 text-xs font-medium text-acento">
              Sua loja atingiu o limite do plano grátis — ative o Basic agora (sem custo) e volte a receber
              pedidos na hora.
            </p>
          ) : loja.pedidos_mes_atual > LIMITE_PEDIDOS_START ? (
            <p className="mt-1 text-xs text-tinta-suave">
              Você passou do limite do plano grátis — ative o Basic pra continuar recebendo pedidos sem
              interrupção.
            </p>
          ) : (
            <p className="mt-1 text-xs text-tinta-suave">Plano Start, limite recorrente por mês.</p>
          )}
        </Link>
      )}

      {/* Cards de resumo */}
      <div className="grid grid-cols-3 gap-3">
        <div className="rounded-2xl bg-superficie p-4 shadow-sm">
          <p className="text-xs font-medium uppercase tracking-wide text-tinta-suave">Semana</p>
          <p className="mt-1 font-carimbo text-xl font-semibold text-tinta">
            {moeda(totalSemana)}
          </p>
        </div>
        <div className="rounded-2xl bg-superficie p-4 shadow-sm">
          <p className="text-xs font-medium uppercase tracking-wide text-tinta-suave">Mês</p>
          <p className="mt-1 font-carimbo text-xl font-semibold text-tinta">
            {moeda(totalMes)}
          </p>
        </div>
        <div className="rounded-2xl bg-superficie p-4 shadow-sm">
          <p className="text-xs font-medium uppercase tracking-wide text-tinta-suave">Pedidos</p>
          <p className="mt-1 font-carimbo text-xl font-semibold text-tinta">
            {pedidosSemana}
          </p>
          <p className="text-xs text-tinta-suave">últimos 7 dias</p>
        </div>
      </div>

      {/* Gráfico de receita por dia */}
      <div className="rounded-2xl bg-superficie p-5 shadow-sm">
        <h2 className="mb-4 font-display text-base tracking-wide text-tinta">Receita — últimos 7 dias</h2>
        {receita7Dias.length === 0 || receita7Dias.every((d) => d.total === 0) ? (
          <p className="py-8 text-center text-sm text-tinta-suave">Nenhum pedido pago nos últimos 7 dias.</p>
        ) : (
          <ResponsiveContainer width="100%" height={200}>
            <LineChart data={receita7Dias} margin={{ top: 5, right: 10, left: 0, bottom: 5 }}>
              <CartesianGrid strokeDasharray="3 3" stroke="rgba(43,33,24,0.08)" />
              <XAxis dataKey="data" tick={{ fontSize: 11 }} />
              <YAxis tick={{ fontSize: 11 }} tickFormatter={(v) => `R$${v}`} width={55} />
              <Tooltip formatter={(v: unknown) => moeda(v as number)} />
              <Line
                type="monotone"
                dataKey="total"
                stroke="rgb(var(--color-acento))"
                strokeWidth={2}
                dot={{ r: 4, fill: 'rgb(var(--color-acento))' }}
                activeDot={{ r: 6 }}
              />
            </LineChart>
          </ResponsiveContainer>
        )}
      </div>

      {/* Gráfico semanal */}
      <div className="rounded-2xl bg-superficie p-5 shadow-sm">
        <h2 className="mb-4 font-display text-base tracking-wide text-tinta">Receita — últimas 4 semanas</h2>
        {receita4Semanas.length === 0 || receita4Semanas.every((s) => s.total === 0) ? (
          <p className="py-8 text-center text-sm text-tinta-suave">Nenhum pedido pago nas últimas 4 semanas.</p>
        ) : (
          <ResponsiveContainer width="100%" height={180}>
            <BarChart data={receita4Semanas} margin={{ top: 5, right: 10, left: 0, bottom: 5 }}>
              <CartesianGrid strokeDasharray="3 3" stroke="rgba(43,33,24,0.08)" />
              <XAxis dataKey="semana" tick={{ fontSize: 11 }} />
              <YAxis tick={{ fontSize: 11 }} tickFormatter={(v) => `R$${v}`} width={55} />
              <Tooltip formatter={(v: unknown) => moeda(v as number)} />
              <Bar dataKey="total" fill="rgb(var(--color-acento))" radius={[4, 4, 0, 0]} />
            </BarChart>
          </ResponsiveContainer>
        )}
      </div>

      {/* Top produtos */}
      {topProdutos.length > 0 && (
        <div className="rounded-2xl bg-superficie p-5 shadow-sm">
          <h2 className="mb-4 font-display text-base tracking-wide text-tinta">Mais vendidos — últimos 30 dias</h2>
          <ResponsiveContainer width="100%" height={180}>
            <BarChart
              data={topProdutos}
              layout="vertical"
              margin={{ top: 0, right: 20, left: 0, bottom: 0 }}
            >
              <CartesianGrid strokeDasharray="3 3" stroke="rgba(43,33,24,0.08)" horizontal={false} />
              <XAxis type="number" tick={{ fontSize: 11 }} />
              <YAxis dataKey="nome" type="category" tick={{ fontSize: 11 }} width={100} />
              <Tooltip formatter={(v: unknown) => [`${v as number} vendas`, '']} />
              <Bar dataKey="quantidade" fill="rgb(var(--color-douro))" radius={[0, 4, 4, 0]} />
            </BarChart>
          </ResponsiveContainer>
        </div>
      )}

      {/* Top clientes (Fase 10.4) — os dois rankings, quantidade e valor,
          lado a lado no desktop e empilhados no mobile. */}
      {(topClientesPorPedidos.length > 0 || topClientesPorValor.length > 0) && (
        <div className="grid gap-4 sm:grid-cols-2">
          {topClientesPorPedidos.length > 0 && (
            <div className="rounded-2xl bg-superficie p-5 shadow-sm">
              <h2 className="mb-3 font-display text-base tracking-wide text-tinta">Clientes mais fiéis</h2>
              <ul className="space-y-2">
                {topClientesPorPedidos.map((cliente, i) => (
                  <li key={cliente.cliente_telefone}>
                    <button
                      onClick={() => setClienteSelecionado({ nome: cliente.cliente_nome, telefone: cliente.cliente_telefone })}
                      className="flex w-full items-center justify-between gap-2 rounded-lg py-0.5 text-left text-sm hover:bg-fundo"
                    >
                      <span className="flex items-center gap-2 text-tinta">
                        <span className="flex h-5 w-5 shrink-0 items-center justify-center rounded-full bg-douro/15 text-xs font-semibold text-douro">
                          {i + 1}
                        </span>
                        {cliente.cliente_nome}
                      </span>
                      <span className="shrink-0 text-tinta-suave">
                        {cliente.total_pedidos} pedido{cliente.total_pedidos !== 1 ? 's' : ''}
                      </span>
                    </button>
                  </li>
                ))}
              </ul>
            </div>
          )}

          {topClientesPorValor.length > 0 && (
            <div className="rounded-2xl bg-superficie p-5 shadow-sm">
              <h2 className="mb-3 font-display text-base tracking-wide text-tinta">Maiores clientes</h2>
              <ul className="space-y-2">
                {topClientesPorValor.map((cliente, i) => (
                  <li key={cliente.cliente_telefone}>
                    <button
                      onClick={() => setClienteSelecionado({ nome: cliente.cliente_nome, telefone: cliente.cliente_telefone })}
                      className="flex w-full items-center justify-between gap-2 rounded-lg py-0.5 text-left text-sm hover:bg-fundo"
                    >
                      <span className="flex items-center gap-2 text-tinta">
                        <span className="flex h-5 w-5 shrink-0 items-center justify-center rounded-full bg-douro/15 text-xs font-semibold text-douro">
                          {i + 1}
                        </span>
                        {cliente.cliente_nome}
                      </span>
                      <span className="shrink-0 font-carimbo text-tinta-suave">{moeda(cliente.valor_total)}</span>
                    </button>
                  </li>
                ))}
              </ul>
            </div>
          )}
        </div>
      )}

      {/* Tipo de entrega e forma de pagamento mais usados (Fase 10.6) —
          sem janela de tempo, histórico inteiro da loja. */}
      {(tiposEntrega.length > 0 || formasPagamento.length > 0) && (
        <div className="grid gap-4 sm:grid-cols-2">
          {tiposEntrega.length > 0 && (
            <div className="rounded-2xl bg-superficie p-5 shadow-sm">
              <h2 className="mb-3 font-display text-base tracking-wide text-tinta">Tipo de entrega mais usado</h2>
              <ul className="space-y-2">
                {tiposEntrega.map((item) => (
                  <li key={item.chave} className="flex items-center justify-between gap-2 text-sm">
                    <span className="text-tinta">{rotuloChave(item.chave, ROTULO_TIPO_ENTREGA)}</span>
                    <span className="shrink-0 text-tinta-suave">
                      {item.total} pedido{item.total !== 1 ? 's' : ''}
                    </span>
                  </li>
                ))}
              </ul>
            </div>
          )}

          {formasPagamento.length > 0 ? (
            <div className="rounded-2xl bg-superficie p-5 shadow-sm">
              <h2 className="mb-3 font-display text-base tracking-wide text-tinta">Forma de pagamento mais usada</h2>
              <ul className="space-y-2">
                {formasPagamento.map((item) => (
                  <li key={item.chave} className="flex items-center justify-between gap-2 text-sm">
                    <span className="text-tinta">{rotuloChave(item.chave, ROTULO_FORMA_PAGAMENTO)}</span>
                    <span className="shrink-0 text-tinta-suave">
                      {item.total} pedido{item.total !== 1 ? 's' : ''}
                    </span>
                  </li>
                ))}
              </ul>
            </div>
          ) : (
            <div className="rounded-2xl bg-superficie p-5 shadow-sm">
              <h2 className="mb-3 font-display text-base tracking-wide text-tinta">Forma de pagamento mais usada</h2>
              <p className="text-xs text-tinta-suave">
                Ainda sem dado suficiente — só pedidos pagos a partir de agora entram nessa conta.
              </p>
            </div>
          )}
        </div>
      )}

      {clienteSelecionado && (
        <HistoricoClienteModal
          nome={clienteSelecionado.nome}
          telefone={clienteSelecionado.telefone}
          onFechar={() => setClienteSelecionado(null)}
        />
      )}
    </div>
  );
}