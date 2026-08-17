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
  MessageCircle, Truck, Sparkles, BarChart3, TicketPercent, ArrowDown, Check, X,
} from 'lucide-react';
import { criarCheckoutAssinatura } from '../api/planos';
import { PLANOS, custoPlano, taxaEfetivaPlano, temaPlanos, FONTE_DRX_SERIF_CSS, RECURSOS_POR_PLANO } from '../lib/planos';

function fmt(v: number) {
  return v.toLocaleString('pt-BR', { style: 'currency', currency: 'BRL', maximumFractionDigits: 0 });
}

// PROVAS são blocos com tela real do sistema (dado fictício, sem cliente
// real — ver docs/prints-apresentacao/) em vez de ícone genérico. Convence
// mais que "ícone + frase solta" porque mostra o produto de verdade.
const PROVAS = [
  {
    imagem: '/planos/prova-cardapio.png',
    titulo: 'Seu cardápio, do seu jeito',
    texto: 'Link e QR code prontos em minutos — sem aplicativo de terceiro escondendo seu cliente e cobrando comissão por fora.',
  },
  {
    imagem: '/planos/prova-pedidos.png',
    titulo: 'Todo pedido, num painel só',
    texto: 'Status, cliente e valor de cada pedido organizados na hora — sem planilha, sem grupo de WhatsApp bagunçado.',
  },
  {
    imagem: '/planos/prova-carrinho.png',
    titulo: 'O carrinho vende por você',
    texto: 'Sugestão certa na hora certa e cupom aplicado ali mesmo — ticket médio maior sem precisar convencer ninguém.',
  },
];

const FUNCOES = [
  {
    icone: Truck,
    titulo: 'Frete calculado sozinho',
    texto: 'Cliente digita o endereço, o frete sai calculado na hora — sem tabela manual, sem prejuízo escondido.',
  },
  {
    icone: BarChart3,
    titulo: 'Números em tempo real',
    texto: 'Faturamento, pedidos e mais vendidos num painel só — decisão com dado, não com "achismo".',
  },
  {
    icone: TicketPercent,
    titulo: 'Cupons de desconto',
    texto: 'Crie um cupom em segundos pra atrair cliente novo ou recompensar quem já compra.',
  },
];

// Prévia das duas mensagens automáticas reais que o cliente recebe —
// texto fiel ao que sai de verdade (ver notification/templates.go), só com
// nome/loja/pedido trocados por dado fictício. Só 2 estágios: não existe
// aviso de "preparando" no sistema hoje.
const CONVERSA_WHATSAPP = [
  {
    quando: 'Pagamento confirmado',
    texto: 'Oi, Maria! Seu pedido #128 na Cardápio Exemplo foi confirmado.\n\n1x Hambúrguer Bacon — R$ 26,00\n\nTotal: R$ 26,00\nRetirada: hoje às 19h30\n\nObrigado pela preferência!',
  },
  {
    quando: 'Saiu para entrega',
    texto: 'Oba, Maria! Seu pedido #128 na Cardápio Exemplo saiu para entrega. 🛵\n\nAcompanhe em tempo real: drenux.com.br/…/rastrear',
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
    () =>
      PLANOS.map((p) => ({
        ...p,
        valorTaxa: custoPlano(p, faturamento) - p.mensal,
        total: custoPlano(p, faturamento),
      })),
    [faturamento]
  );
  const menorCusto = Math.min(...custos.filter((c) => c.faixas.length > 0).map((c) => c.total));

  async function escolherPlano(planoId: string) {
    if (planoId === 'start' || planoId === 'basic') {
      // plano_desejado (não `plano`) pra não colidir com o `?plano=` que
      // Cadastro.tsx já usa pro banner de "pagamento confirmado" no
      // retorno do checkout de Pro/Scale — aqui não houve pagamento nenhum.
      navigate(`/cadastro?plano_desejado=${planoId}`);
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
        <span className="mt-5 inline-flex items-center gap-2 rounded-full border border-primary/30 bg-primary/10 px-4 py-1.5 text-xs font-medium text-primary">
          <Sparkles className="h-3.5 w-3.5" strokeWidth={1.75} />
          Zero comissão até você mesmo ligar o Pix automático
        </span>
      </section>

      {/* Barra de confiança — sinaliza stack sério sem precisar de número
          de cliente que ainda não temos. */}
      <section className="border-y border-border">
        <div className="mx-auto flex max-w-4xl flex-wrap items-center justify-center gap-x-10 gap-y-3 px-6 py-6 text-xs font-medium uppercase tracking-widest text-muted-foreground">
          <span>Pagamento via Mercado Pago</span>
          <span className="text-primary/40">•</span>
          <span>Pix automático</span>
          <span className="text-primary/40">•</span>
          <span>Avisos por WhatsApp</span>
        </div>
      </section>

      {/* Prova em tela real — em vez de ícone genérico, mostra a tela de
          verdade (dado fictício, sem cliente real). */}
      <section className="mx-auto max-w-5xl px-6 pb-16 pt-20">
        <h2 className="drx-serif mb-10 text-center text-2xl font-medium sm:text-3xl">
          Tudo que sua loja precisa pra vender mais, num lugar só
        </h2>
        <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
          {PROVAS.map((p) => (
            <Card key={p.titulo} className="overflow-hidden border-0 ring-1 ring-border">
              <img src={p.imagem} alt={p.titulo} className="aspect-[4/3] w-full object-cover object-top" />
              <CardContent className="pt-5">
                <p className="mb-1.5 text-sm font-semibold text-foreground">{p.titulo}</p>
                <p className="text-sm leading-relaxed text-muted-foreground">{p.texto}</p>
              </CardContent>
            </Card>
          ))}
        </div>

        <div className="mt-6 grid grid-cols-1 gap-4 sm:grid-cols-3">
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

      {/* Prévia de conversa — mostra os dois avisos automáticos reais que
          o cliente recebe (texto fiel ao que o sistema manda de verdade). */}
      <section className="border-y border-border bg-card/40">
        <div className="mx-auto max-w-md px-6 py-16">
          <div className="mb-8 flex items-center justify-center gap-2 text-center">
            <MessageCircle className="h-5 w-5 text-primary" strokeWidth={1.75} />
            <h2 className="drx-serif text-2xl font-medium sm:text-3xl">Seu cliente avisado, sem você mexer um dedo</h2>
          </div>
          <div className="space-y-3">
            {CONVERSA_WHATSAPP.map((msg) => (
              <div key={msg.quando}>
                <p className="mb-1.5 text-[11px] font-medium uppercase tracking-wide text-muted-foreground">
                  {msg.quando}
                </p>
                <div className="whitespace-pre-line rounded-2xl rounded-tl-sm bg-[#1f2c25] px-4 py-3 text-sm leading-relaxed text-[#e9f5ee] ring-1 ring-[#2f7a52]/40">
                  {msg.texto}
                </div>
              </div>
            ))}
          </div>
          <p className="mt-5 text-center text-xs text-muted-foreground">
            Exemplo ilustrativo com dado fictício — a partir do Basic. No Start, o aviso de novo pedido chega por
            email pro dono, sem esses avisos automáticos pro cliente.
          </p>
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
            const semSplit = p.faixas.length === 0;
            const recomendado = !semSplit && p.total === menorCusto;
            const efetivo = taxaEfetivaPlano(p, faturamento) * 100;

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
                        <p className="text-lg font-semibold">
                          {semSplit ? 'Sem split' : `a partir de ${(p.faixas[0].taxa * 100).toFixed(1)}%`}
                        </p>
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
                          {semSplit ? 'Sem taxa de pagamento' : `${fmt(p.valorTaxa)} de taxa`}
                          {' + '}
                          {p.mensal === 0 ? 'R$ 0' : fmt(p.mensal)} de mensalidade
                        </p>
                        <p className="drx-serif text-2xl font-medium text-primary">
                          R$ <NumberTicker value={p.total} className="text-primary" />
                          <span className="drx-serif text-sm font-normal text-muted-foreground"> /mês*</span>
                        </p>
                      </div>
                      {!semSplit && (
                        <span className="rounded-lg bg-secondary px-3 py-1.5 text-sm font-semibold">
                          {efetivo.toFixed(1)}%
                        </span>
                      )}
                    </div>
                  </div>

                  <div>
                    <p className="mb-2 text-[11px] font-medium uppercase tracking-wide text-muted-foreground">
                      O que está incluso
                    </p>
                    <ul className="space-y-1.5 border-t border-border pt-3">
                      {RECURSOS_POR_PLANO.map((r) => {
                        const valor = r.valores[p.id];
                        const incluso = valor !== '—';
                        return (
                          <li key={r.label} className="flex items-start gap-2 text-sm">
                            {incluso ? (
                              <Check className="mt-0.5 h-4 w-4 shrink-0 text-primary" />
                            ) : (
                              <X className="mt-0.5 h-4 w-4 shrink-0 text-muted-foreground/40" />
                            )}
                            <span className={incluso ? 'text-foreground' : 'text-muted-foreground/50 line-through'}>
                              {r.label}
                              {incluso && valor !== '✓' ? ` — ${valor}` : ''}
                            </span>
                          </li>
                        );
                      })}
                    </ul>
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
                Cardápio digital, frete calculado automaticamente, guardar e entregar depois e o programa de
                afiliados valem pra qualquer plano. O Start tem limite de 30 produtos e 30 pedidos por mês (renova
                todo mês) e não manda aviso automático de status pro cliente — a partir do Basic isso já é
                ilimitado e o aviso vem incluso. Rastreamento de entrega em tempo real (mapa ao vivo) é exclusivo
                do Pro e do Scale. A lista completa está em cada plano acima, em "O que está incluso".
              </AccordionContent>
            </AccordionItem>
          </Accordion>
        </div>
      </section>
    </div>
  );
}