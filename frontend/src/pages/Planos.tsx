import { useMemo, useRef, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { Card, CardHeader, CardTitle, CardDescription, CardContent, CardFooter } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Slider } from '@/components/ui/slider';
import { Button } from '@/components/ui/button';
import { Accordion, AccordionItem, AccordionTrigger, AccordionContent } from '@/components/ui/accordion';
import { NumberTicker } from '@/components/ui/number-ticker';
import { ShimmerButton } from '@/components/ui/shimmer-button';
import {
  Store, MessageCircle, Truck, Sparkles, BarChart3, Users, ArrowDown,
} from 'lucide-react';
import { criarCheckoutAssinatura } from '../api/planos';
import { PLANOS, temaPlanos, FONTE_DRX_SERIF_CSS } from '../lib/planos';

function fmt(v: number) {
  return v.toLocaleString('pt-BR', { style: 'currency', currency: 'BRL', maximumFractionDigits: 0 });
}

const FUNCOES = [
  {
    icone: Store,
    titulo: 'Cardápio com a sua cara',
    texto: 'Link e QR code próprios, no ar em minutos — sem app de terceiro cobrando por fora e escondendo seu cliente.',
  },
  {
    icone: MessageCircle,
    titulo: 'Pedido direto no seu WhatsApp',
    texto: 'Cada venda cai automática no seu número. Nenhum pedido se perde, nenhuma plataforma fica no meio do caminho.',
  },
  {
    icone: Truck,
    titulo: 'Frete calculado sozinho',
    texto: 'O cliente digita o endereço e o sistema calcula a distância e o valor na hora — sem tabela manual, sem prejuízo escondido.',
  },
  {
    icone: Sparkles,
    titulo: 'Sugestão inteligente e cupons',
    texto: 'O carrinho sugere o produto certo na hora certa. Ticket médio maior sem você precisar convencer ninguém.',
  },
  {
    icone: BarChart3,
    titulo: 'Números em tempo real',
    texto: 'Faturamento, pedidos e mais vendidos num painel só. Decisão com dado — não com "acho que vendeu bem".',
  },
  {
    icone: Users,
    titulo: 'Programa de afiliados',
    texto: 'Gente divulgando sua loja por comissão. Você cresce sem tirar dinheiro do bolso pra anúncio.',
  },
];

export function Planos() {
  const navigate = useNavigate();
  const [faturamento, setFaturamento] = useState(6000);
  const [carregando, setCarregando] = useState<string | null>(null);
  const [destacarCalculadora, setDestacarCalculadora] = useState(false);
  const calculadoraRef = useRef<HTMLDivElement>(null);

  function irParaCalculadora() {
    calculadoraRef.current?.scrollIntoView({ behavior: 'smooth', block: 'center' });
    setDestacarCalculadora(true);
    setTimeout(() => setDestacarCalculadora(false), 2200);
  }

  const custos = useMemo(
    () => PLANOS.map((p) => ({ ...p, valorTaxa: p.taxa * faturamento, total: p.mensal + p.taxa * faturamento })),
    [faturamento]
  );
  const menorCusto = Math.min(...custos.map((c) => c.total));

  async function escolherPlano(planoId: string) {
    if (planoId === 'start') {
      navigate('/cadastro');
      return;
    }
    setCarregando(planoId);
    try {
      const { url } = await criarCheckoutAssinatura(planoId as 'pro' | 'scale');
      window.location.href = url;
    } catch {
      setCarregando(null);
    }
  }

  return (
    <div style={temaPlanos} className="min-h-screen bg-background text-foreground">
      <style>{`
        ${FONTE_DRX_SERIF_CSS}
        @keyframes drx-girar { from { transform: rotate(0deg); } to { transform: rotate(360deg); } }
        @keyframes drx-pulsar { 0%, 100% { opacity: 1; } 50% { opacity: 0.75; } }
        .drx-estrela { animation: drx-girar 22s linear infinite, drx-pulsar 4s ease-in-out infinite; }
        @keyframes drx-brilho { 0%, 100% { box-shadow: 0 0 0 0 rgba(212, 175, 106, 0); } 50% { box-shadow: 0 0 24px 4px rgba(212, 175, 106, 0.25); } }
        .drx-destaque { animation: drx-brilho 1.1s ease-in-out 2; }
      `}</style>

      {/* Header */}
      <header className="border-b border-border">
        <div className="mx-auto flex max-w-4xl items-center justify-between px-6 py-5">
          <button onClick={() => navigate('/inicio')} className="text-sm font-semibold tracking-widest text-foreground">
            DRENUX
          </button>
          <div className="flex items-center gap-3">
            <Button variant="ghost" size="sm" onClick={() => navigate('/login')}>
              Entrar
            </Button>
            <Button size="sm" onClick={() => navigate('/cadastro')}>
              Criar minha loja
            </Button>
          </div>
        </div>
      </header>

      {/* Hero */}
      <section className="flex flex-col items-center px-6 pb-16 pt-20 text-center">
        <svg className="drx-estrela mb-7" width="88" height="88" viewBox="0 0 100 100">
          <defs>
            <linearGradient id="ouroGrad" x1="0%" y1="0%" x2="100%" y2="100%">
              <stop offset="0%" stopColor="#e8cd94" />
              <stop offset="100%" stopColor="#d4af6a" />
            </linearGradient>
          </defs>
          <path
            d="M50 4 C50 30 30 50 4 50 C30 50 50 70 50 96 C50 70 70 50 96 50 C70 50 50 30 50 4 Z"
            fill="url(#ouroGrad)"
          />
        </svg>

        <h1 className="drx-serif max-w-xl text-4xl font-medium leading-tight sm:text-5xl">
          Um plano pra cada fase da sua loja
        </h1>
        <p className="mt-4 max-w-md text-sm text-muted-foreground">
          Comece de graça, sem risco. Migre quando o faturamento pedir — sem multa, sem fidelidade.
        </p>
      </section>

      {/* O que o Drenux resolve pro dono do negócio */}
      <section className="mx-auto max-w-4xl px-6 pb-20">
        <h2 className="drx-serif mb-10 text-center text-2xl font-medium sm:text-3xl">
          Tudo que sua loja precisa pra vender mais, num lugar só
        </h2>
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {FUNCOES.map((f) => (
            <Card key={f.titulo} className="border-0 ring-1 ring-border">
              <CardContent className="pt-6">
                <f.icone className="mb-3 h-6 w-6 text-primary" strokeWidth={1.75} />
                <p className="mb-1.5 text-sm font-semibold text-foreground">{f.titulo}</p>
                <p className="text-sm leading-relaxed text-muted-foreground">{f.texto}</p>
              </CardContent>
            </Card>
          ))}
        </div>
      </section>

      {/* Chamada reflexiva pra calculadora */}
      <section className="border-y border-border bg-card/40">
        <div className="mx-auto max-w-xl px-6 py-16 text-center">
          <p className="drx-serif text-2xl font-medium leading-snug sm:text-3xl">
            Quanto da sua venda vira taxa sem você nem perceber?
          </p>
          <p className="mx-auto mt-4 max-w-md text-sm text-muted-foreground">
            A maioria dos donos de loja sente que "sobra pouco" no fim do mês, mas não sabe dizer o número
            exato. Simule o faturamento real da sua loja e veja, em segundos, o quanto cada plano custaria
            pra você — e qual sai mais barato.
          </p>
          <Button size="lg" className="mt-7 gap-2" onClick={irParaCalculadora}>
            Simular meu faturamento
            <ArrowDown className="h-4 w-4" />
          </Button>
        </div>
      </section>

      {/* Calculadora + planos */}
      <section className="mx-auto max-w-2xl px-6 pb-24 pt-16">
        <Card
          ref={calculadoraRef}
          className={`mb-8 transition-shadow duration-500 ${destacarCalculadora ? 'drx-destaque ring-2 ring-primary' : ''}`}
        >
          <CardHeader>
            <CardDescription>Quanto sua loja fatura por mês?</CardDescription>
            <CardTitle className="drx-serif text-3xl font-medium text-primary">
              R$ <NumberTicker value={faturamento} className="text-primary" />
            </CardTitle>
          </CardHeader>
          <CardContent>
            <Slider
              value={[faturamento]}
              onValueChange={(v) => setFaturamento(Array.isArray(v) ? v[0] : v)}
              min={0}
              max={20000}
              step={100}
            />
          </CardContent>
        </Card>

        <div className="space-y-4">
          {custos.map((p) => {
            const recomendado = p.total === menorCusto;
            const efetivo = faturamento > 0 ? (p.total / faturamento) * 100 : p.taxa * 100;

            return (
              <Card key={p.id} className={recomendado ? 'ring-2 ring-primary' : ''}>
                <CardHeader>
                  <div className="flex items-start justify-between">
                    <div>
                      <CardTitle className="drx-serif text-xl font-medium">{p.nome}</CardTitle>
                      <CardDescription>{p.desc}</CardDescription>
                    </div>
                    {recomendado && <Badge className="bg-primary text-primary-foreground">Mais barato</Badge>}
                  </div>
                </CardHeader>

                <CardContent className="space-y-4">
                  <div>
                    <p className="mb-2 text-[11px] font-medium uppercase tracking-wide text-muted-foreground">
                      Valores fixos do plano
                    </p>
                    <div className="grid grid-cols-2 gap-2">
                      <div className="rounded-lg bg-secondary px-3 py-2.5">
                        <p className="text-xs text-muted-foreground">Mensalidade</p>
                        <p className="text-lg font-semibold">{p.mensal === 0 ? 'R$ 0' : fmt(p.mensal)}</p>
                      </div>
                      <div className="rounded-lg bg-secondary px-3 py-2.5">
                        <p className="text-xs text-muted-foreground">Taxa por pedido</p>
                        <p className="text-lg font-semibold">{(p.taxa * 100).toFixed(1)}%</p>
                      </div>
                    </div>
                  </div>

                  <div>
                    <p className="mb-2 text-[11px] font-medium uppercase tracking-wide text-muted-foreground">
                      Projeção com o faturamento simulado
                    </p>
                    <div className="flex items-center justify-between border-t border-border pt-3">
                      <div>
                        <p className="text-xs text-muted-foreground">
                          {fmt(p.valorTaxa)} de taxa + {p.mensal === 0 ? 'R$ 0' : fmt(p.mensal)} de mensalidade
                        </p>
                        <p className="drx-serif text-2xl font-medium text-primary">
                          R$ <NumberTicker value={p.total} className="text-primary" />
                          <span className="drx-serif text-sm font-normal text-muted-foreground"> /mês*</span>
                        </p>
                      </div>
                      <span className="rounded-lg bg-secondary px-3 py-1.5 text-sm font-semibold">
                        {efetivo.toFixed(1)}%
                      </span>
                    </div>
                  </div>
                </CardContent>

                <CardFooter>
                  {recomendado ? (
                    <ShimmerButton
                      onClick={() => escolherPlano(p.id)}
                      disabled={carregando === p.id}
                      background="#d4af6a"
                      shimmerColor="#ffffff"
                      className="w-full text-sm font-semibold text-background"
                    >
                      {carregando === p.id ? 'Abrindo pagamento...' : `Escolher ${p.nome}`}
                    </ShimmerButton>
                  ) : (
                    <Button
                      className="w-full"
                      variant="secondary"
                      onClick={() => escolherPlano(p.id)}
                      disabled={carregando === p.id}
                    >
                      {carregando === p.id ? 'Abrindo pagamento...' : `Escolher ${p.nome}`}
                    </Button>
                  )}
                </CardFooter>
              </Card>
            );
          })}
        </div>

        {/* Aviso */}
        <div className="mt-4 flex items-start gap-2 rounded-xl bg-card px-4 py-3 ring-1 ring-border">
          <span className="text-muted-foreground">ⓘ</span>
          <p className="text-xs text-muted-foreground">
            Mensalidade e taxa são os valores fixos de cada plano. O total mostrado é uma projeção com base no
            faturamento que você simulou acima — o valor real muda conforme suas vendas no mês.
          </p>
        </div>

        {/* FAQ */}
        <div className="mt-16">
          <h2 className="drx-serif mb-6 text-center text-2xl font-medium">Perguntas frequentes</h2>
          <Accordion>
            <AccordionItem value="troca">
              <AccordionTrigger>Posso trocar de plano depois?</AccordionTrigger>
              <AccordionContent>
                Sim, a qualquer momento, direto no painel — sem multa e sem fidelidade.
              </AccordionContent>
            </AccordionItem>
            <AccordionItem value="cobranca">
              <AccordionTrigger>A taxa sai automática ou eu pago à parte?</AccordionTrigger>
              <AccordionContent>
                A taxa é descontada automaticamente de cada pedido pago. A mensalidade (quando houver) é cobrada
                uma vez por mês no cartão cadastrado.
              </AccordionContent>
            </AccordionItem>
            <AccordionItem value="recursos">
              <AccordionTrigger>O que muda entre os planos em termos de recursos?</AccordionTrigger>
              <AccordionContent>
                Nada. Rastreamento em tempo real, frete dinâmico, guardar e entregar depois, sistema de afiliados —
                todos os recursos do Drenux estão disponíveis em qualquer plano. Só muda como você paga.
              </AccordionContent>
            </AccordionItem>
          </Accordion>
        </div>
      </section>
    </div>
  );
}