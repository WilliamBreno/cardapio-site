import { useState, useEffect, useMemo, useRef } from 'react';
import { useQuery } from '@tanstack/react-query';
import { useCartStore } from '../store/cartStore';
import { criarPedido, criarCheckout } from '../api/pedidos';
import { validarCupom } from '../api/cupons';
import { cotarFrete } from '../api/frete';
import { buscarSugestoesCarrinho } from '../api/sugestoes';
import { SugestaoPreviewItem } from './SugestaoPreviewItem';
import { Campo } from './Campo';
import { EnderecoCampos, enderecoVazio, enderecoParaTexto, enderecoPreenchido, type EnderecoValor } from './EnderecoCampos';
import { Stepper } from './ui/stepper';
import { Segmented } from './ui/segmented';
import { precoItem, rotuloCombo } from '../lib/utils';
import type { Produto, TipoProduto } from '../api/types';

interface Props {
  aberto: boolean;
  onFechar: () => void;
  slug: string;
  modoPedido: 'imediato' | 'agendado';
  antecedenciaMinimaHoras: number;
  aceitaRetirada: boolean;
  aceitaEntrega: boolean;
  aceitaGuardarEntregar: boolean;
  taxaEntregaTipo: string;
  taxaEntregaValor: number;
  valorMinimoPedido: number;
  sugestaoInteligenteAtiva: boolean;
  produtos: Produto[];
  segmentoLoja?: TipoProduto;
}

function formatarDataLocal(data: Date): string {
  const ano = data.getFullYear();
  const mes = String(data.getMonth() + 1).padStart(2, '0');
  const dia = String(data.getDate()).padStart(2, '0');
  return `${ano}-${mes}-${dia}`;
}

function formatarHoraLocal(data: Date): string {
  return `${String(data.getHours()).padStart(2, '0')}:${String(data.getMinutes()).padStart(2, '0')}`;
}

function dataMinima(antecedenciaHoras: number): string {
  const data = new Date();
  data.setHours(data.getHours() + antecedenciaHoras);
  return formatarDataLocal(data);
}

function horaMinima(dataSelecionada: string, antecedenciaHoras: number): string {
  if (!dataSelecionada) return '00:00';

  const agora = new Date();
  const minimo = new Date(agora.getTime() + antecedenciaHoras * 60 * 60 * 1000);
  const diaMinimo = formatarDataLocal(minimo);

  if (dataSelecionada === diaMinimo) {
    const mins = minimo.getMinutes();
    const arredondado = Math.ceil(mins / 15) * 15;
    minimo.setMinutes(arredondado, 0, 0);
    return formatarHoraLocal(minimo);
  }

  return '00:00';
}

function normalizarTelefone(valor: string): string {
  const digitos = valor.replace(/\D/g, '');
  return digitos.startsWith('55') ? digitos : `55${digitos}`;
}

export function CarrinhoDrawer({ aberto, onFechar, slug, modoPedido, antecedenciaMinimaHoras, aceitaRetirada, aceitaEntrega, aceitaGuardarEntregar, taxaEntregaTipo, taxaEntregaValor, valorMinimoPedido, sugestaoInteligenteAtiva, produtos, segmentoLoja }: Props) {
  const itens = useCartStore((state) => state.itens);
  const combos = useCartStore((state) => state.combos);
  const total = useCartStore((state) => state.total());
  const alterarQuantidade = useCartStore((state) => state.alterarQuantidade);
  const remover = useCartStore((state) => state.remover);
  const alterarQuantidadeCombo = useCartStore((state) => state.alterarQuantidadeCombo);
  const removerCombo = useCartStore((state) => state.removerCombo);
  const adicionarSugestao = useCartStore((state) => state.adicionarSugestao);

  // Calculado localmente (não via seletor do Zustand) de propósito: a
  // versão anterior chamava state.produtosNoCarrinho() dentro do seletor,
  // que devolve um array novo a cada chamada — o useSyncExternalStore por
  // baixo do Zustand detecta a referência diferente a cada render como
  // "mudou de novo" e entra em loop infinito de atualização (React error
  // #185, "Maximum update depth exceeded"), quebrando a página pública
  // inteira sem nenhum ErrorBoundary pra conter. useMemo aqui só recalcula
  // quando itens/combos (já selecionados de forma estável) realmente mudam.
  const produtosNoCarrinho = useMemo(() => {
    const idsAvulso = itens.map((item) => item.produto.id);
    const idsCombo = combos.flatMap((item) => item.combo.itens.map((i) => i.produto_id));
    return Array.from(new Set([...idsAvulso, ...idsCombo]));
  }, [itens, combos]);

  // Seção consolidada de sugestões — juntando os vínculos de TODOS os
  // produtos já no carrinho (avulso ou componente de combo), exibida na
  // revisão do carrinho antes do cliente finalizar (não popup a cada
  // produto adicionado, ver docs/plano-melhorias-drenux.md Fase 6).
  const { data: sugestoesCarrinho } = useQuery({
    queryKey: ['sugestoes-carrinho', slug, produtosNoCarrinho],
    queryFn: () => buscarSugestoesCarrinho(slug, produtosNoCarrinho),
    enabled: sugestaoInteligenteAtiva && aberto && produtosNoCarrinho.length > 0,
  });

  // "Guardar pra depois" só aparece se a loja ativou o recurso, todo mundo
  // no carrinho é "mercadoria" E não tem combo — combo não guarda pra
  // entregar depois (ver o guard equivalente em PedidoService.CriarPorSlug,
  // combo não tem TipoProduto/PesoGramas por componente pra sustentar esse
  // fluxo).
  const todosMercadoria =
    combos.length === 0 &&
    itens.length > 0 &&
    itens.every((item) => item.produto.tipo_produto === 'mercadoria');
  const opcoesModoEntrega: { valor: 'retirada' | 'entrega' | 'guardar'; label: string }[] = [
    ...(aceitaRetirada ? [{ valor: 'retirada' as const, label: '🏪 Retirada' }] : []),
    ...(aceitaEntrega ? [{ valor: 'entrega' as const, label: '🛵 Entrega' }] : []),
    ...(aceitaGuardarEntregar && todosMercadoria ? [{ valor: 'guardar' as const, label: '📦 Guardar' }] : []),
  ];

  const [etapa, setEtapa] = useState<'carrinho' | 'dados'>('carrinho');
  const [nome, setNome] = useState('');
  const [telefone, setTelefone] = useState('');
  const [data, setData] = useState('');
  const [hora, setHora] = useState('');
  const [modoEntrega, setModoEntrega] = useState<'retirada' | 'entrega' | 'guardar'>(
    aceitaRetirada ? 'retirada' : 'entrega'
  );
  const [enderecoValor, setEnderecoValor] = useState<EnderecoValor>(enderecoVazio);
  const enderecoTexto = enderecoParaTexto(enderecoValor);
  const [cupomCodigo, setCupomCodigo] = useState('');
  const [desconto, setDesconto] = useState(0);
  const [cupomMsg, setCupomMsg] = useState<{ tipo: 'ok' | 'erro'; texto: string } | null>(null);
  const [validandoCupom, setValidandoCupom] = useState(false);
  const [enviando, setEnviando] = useState(false);
  const [erro, setErro] = useState<string | null>(null);

  // Cotação de frete por km — calculada em tempo real conforme o cliente
  // digita o endereço, com debounce pra não disparar uma chamada a cada
  // letra digitada.
  const [freteCalculado, setFreteCalculado] = useState<number | null>(null);
  const [distanciaKm, setDistanciaKm] = useState<number | null>(null);
  const [calculandoFrete, setCalculandoFrete] = useState(false);
  const [erroFrete, setErroFrete] = useState<string | null>(null);
  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  useEffect(() => {
    // Se o carrinho deixou de ser 100% mercadoria (ex: cliente adicionou
    // um item alimentício depois de já ter escolhido "guardar"), volta
    // pro modo padrão em vez de deixar uma opção que sumiu selecionada.
    if (modoEntrega === 'guardar' && !opcoesModoEntrega.some((o) => o.valor === 'guardar')) {
      setModoEntrega(aceitaRetirada ? 'retirada' : 'entrega');
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [todosMercadoria]);

  useEffect(() => {
    if (taxaEntregaTipo !== 'por_km' || modoEntrega !== 'entrega') {
      return;
    }
    if (debounceRef.current) clearTimeout(debounceRef.current);

    if (!enderecoPreenchido(enderecoValor)) {
      setFreteCalculado(null);
      setDistanciaKm(null);
      setErroFrete(null);
      return;
    }

    debounceRef.current = setTimeout(async () => {
      setCalculandoFrete(true);
      setErroFrete(null);
      try {
        const resultado = await cotarFrete(slug, enderecoTexto, enderecoValor);
        setFreteCalculado(resultado.valor_frete);
        setDistanciaKm(resultado.distancia_km);
      } catch {
        setFreteCalculado(null);
        setDistanciaKm(null);
        setErroFrete('Não conseguimos calcular o frete pra esse endereço. Confere se está completo (rua, número, bairro, cidade).');
      } finally {
        setCalculandoFrete(false);
      }
    }, 900); // espera o cliente parar de digitar por ~0.9s antes de cotar

    return () => {
      if (debounceRef.current) clearTimeout(debounceRef.current);
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [enderecoTexto, taxaEntregaTipo, modoEntrega, slug]);

  if (!aberto) return null;

  const taxaEntrega =
    modoEntrega === 'entrega' && taxaEntregaTipo === 'fixa'
      ? taxaEntregaValor
      : modoEntrega === 'entrega' && taxaEntregaTipo === 'por_km' && freteCalculado !== null
      ? freteCalculado
      : 0;
  const totalComEntrega = total + taxaEntrega - desconto;

  async function aplicarCupom() {
    if (!cupomCodigo.trim()) return;
    setValidandoCupom(true);
    setCupomMsg(null);
    try {
      const resultado = await validarCupom(slug, cupomCodigo, total);
      setDesconto(resultado.desconto);
      setCupomMsg({
        tipo: 'ok',
        texto: resultado.tipo === 'percentual'
          ? `Cupom aplicado — ${resultado.valor}% de desconto`
          : `Cupom aplicado — R$ ${resultado.desconto.toFixed(2).replace('.', ',')} de desconto`,
      });
    } catch {
      setDesconto(0);
      setCupomMsg({ tipo: 'erro', texto: 'Cupom inválido ou não aplicável.' });
    } finally {
      setValidandoCupom(false);
    }
  }

  function removerCupom() {
    setCupomCodigo('');
    setDesconto(0);
    setCupomMsg(null);
  }

  function handleDataChange(novaData: string) {
    setData(novaData);
    const min = horaMinima(novaData, antecedenciaMinimaHoras);
    if (!hora || hora < min) {
      setHora(min);
    }
  }

  async function confirmarPedido() {
    if (!nome.trim() || !telefone.trim()) {
      setErro('Preenche nome e WhatsApp.');
      return;
    }
    if (modoEntrega === 'entrega' && !enderecoPreenchido(enderecoValor)) {
      setErro('Preenche o endereço de entrega (rua e cidade, pelo menos).');
      return;
    }
    if (modoEntrega === 'entrega' && taxaEntregaTipo === 'por_km' && freteCalculado === null) {
      setErro('Aguarda o cálculo do frete antes de continuar (ou confere se o endereço está completo).');
      return;
    }
    if (modoPedido === 'agendado' && modoEntrega !== 'guardar') {
      if (!data) { setErro('Escolhe a data de retirada.'); return; }
      if (!hora) { setErro('Escolhe a hora.'); return; }
      const escolhida = new Date(`${data}T${hora}:00`);
      const minimo = new Date(Date.now() + antecedenciaMinimaHoras * 60 * 60 * 1000);
      if (escolhida < minimo) {
        const horaMin = horaMinima(data, antecedenciaMinimaHoras);
        setErro(`Horário inválido — essa loja exige pelo menos ${antecedenciaMinimaHoras}h de antecedência. Escolhe a partir das ${horaMin}.`);
        setHora(horaMin);
        return;
      }
    }

    setEnviando(true);
    setErro(null);

    try {
      const dataRetirada = modoPedido === 'agendado' && modoEntrega !== 'guardar'
        ? new Date(`${data}T${hora}:00`).toISOString()
        : new Date().toISOString();

      const pedido = await criarPedido(slug, {
        cliente_nome: nome.trim(),
        cliente_telefone: normalizarTelefone(telefone),
        data_retirada: dataRetirada,
        modo_entrega: modoEntrega,
        endereco_entrega: modoEntrega === 'entrega' ? enderecoTexto : undefined,
        ...(modoEntrega === 'entrega' ? {
          endereco_rua: enderecoValor.rua.trim(),
          endereco_numero: enderecoValor.numero.trim(),
          endereco_complemento: enderecoValor.complemento.trim(),
          endereco_bairro: enderecoValor.bairro.trim(),
          endereco_cidade: enderecoValor.cidade.trim(),
          endereco_estado: enderecoValor.estado.trim(),
          endereco_cep: enderecoValor.cep.trim(),
        } : {}),
        cupom_codigo: desconto > 0 ? cupomCodigo : undefined,
        itens: itens.map((item) => ({
          produto_id: item.produto.id,
          variacao_id: item.variacao?.id,
          quantidade: item.quantidade,
          sugestao_produto_id: item.sugestaoProdutoId,
        })),
        combos: combos.map((item) => ({
          combo_id: item.combo.id,
          quantidade: item.quantidade,
          itens: item.selecoes.map((selecao) => ({
            combo_item_id: selecao.comboItemId,
            variacao_id: selecao.variacao?.id,
          })),
        })),
      });

      const { url } = await criarCheckout(pedido.id);
      window.location.href = url;
    } catch {
      setErro('Não foi possível confirmar o pedido. Confere os dados e tenta de novo.');
      setEnviando(false);
    }
  }

  return (
    <div className="fixed inset-0 z-30 flex items-end justify-center sm:items-center">
      <button
        aria-label="Fechar carrinho"
        onClick={onFechar}
        className="absolute inset-0 bg-tinta/50"
      />

      <div className="relative flex max-h-[85vh] w-full flex-col rounded-t-3xl bg-superficie sm:max-w-md sm:rounded-3xl">
        <div className="flex items-center justify-between border-b border-tinta/10 px-6 py-4">
          <h2 className="font-display text-xl tracking-wide text-tinta">
            {etapa === 'carrinho' ? 'Seu pedido' : 'Dados pra retirada'}
          </h2>
          <button
            onClick={onFechar}
            className="text-xl leading-none text-tinta-suave hover:text-tinta"
            aria-label="Fechar"
          >
            ×
          </button>
        </div>

        <div className="flex-1 overflow-y-auto px-6 py-4">
          {etapa === 'carrinho' ? (
            <div className="space-y-6">
              <ul className="space-y-4">
                {itens.map((item) => {
                  const chave = item.variacao ? `${item.produto.id}-${item.variacao.id}` : `${item.produto.id}`;
                  const precoUnit = item.precoComDesconto ?? precoItem(item.produto, item.variacao);
                  return (
                    <li key={chave} className="flex items-center gap-3">
                      <div className="flex-1">
                        <p className="font-medium text-tinta">
                          {item.produto.nome}
                          {item.variacao && (
                            <span className="ml-1.5 rounded-full bg-fundo px-2 py-0.5 text-xs text-tinta-suave">
                              {item.variacao.nome}
                            </span>
                          )}
                          {item.sugestaoProdutoId && (
                            <span className="ml-1.5 rounded-full bg-emerald-100 px-2 py-0.5 text-xs text-emerald-700">
                              Sugestão
                            </span>
                          )}
                        </p>
                        <p className="font-carimbo text-sm text-tinta-suave">
                          R$ {precoUnit.toFixed(2).replace('.', ',')}
                        </p>
                      </div>

                      <Stepper
                        value={item.quantidade}
                        min={1}
                        onValueChange={(nova) => alterarQuantidade(item.produto.id, nova, item.variacao?.id)}
                      />

                      <button
                        onClick={() => remover(item.produto.id, item.variacao?.id)}
                        className="text-sm text-acento/70 hover:text-acento"
                        aria-label={`Remover ${item.produto.nome}`}
                      >
                        Remover
                      </button>
                    </li>
                  );
                })}

                {combos.map((item) => (
                  <li key={`combo-${item.combo.id}`} className="flex items-center gap-3">
                    <div className="flex-1">
                      <p className="font-medium text-tinta">
                        {item.combo.nome}
                        <span className="ml-1.5 rounded-full bg-acento/10 px-2 py-0.5 text-xs text-acento">
                          {rotuloCombo(segmentoLoja)}
                        </span>
                      </p>
                      <p className="font-carimbo text-sm text-tinta-suave">
                        R$ {item.combo.preco.toFixed(2).replace('.', ',')}
                      </p>
                    </div>

                    <Stepper
                      value={item.quantidade}
                      min={1}
                      onValueChange={(nova) => alterarQuantidadeCombo(item.combo.id, nova)}
                    />

                    <button
                      onClick={() => removerCombo(item.combo.id)}
                      className="text-sm text-acento/70 hover:text-acento"
                      aria-label={`Remover ${item.combo.nome}`}
                    >
                      Remover
                    </button>
                  </li>
                ))}
              </ul>

              {sugestaoInteligenteAtiva && sugestoesCarrinho && sugestoesCarrinho.length > 0 && (
                <div className="space-y-2 rounded-2xl bg-fundo p-4">
                  <p className="text-xs font-medium uppercase tracking-wide text-tinta-suave">
                    Quem pediu isso também levou
                  </p>
                  <ul className="space-y-2">
                    {sugestoesCarrinho.map((sugestao) => {
                      const produto = produtos.find((p) => p.id === sugestao.produto_id);
                      if (!produto) return null;
                      return (
                        <SugestaoPreviewItem
                          key={sugestao.sugestao_id}
                          nome={sugestao.nome}
                          fotoUrl={sugestao.foto_url}
                          preco={sugestao.preco}
                          precoComDesconto={sugestao.preco_com_desconto}
                          onAdicionar={() => adicionarSugestao(sugestao, produto)}
                        />
                      );
                    })}
                  </ul>
                </div>
              )}
            </div>
          ) : (
            <div className="space-y-4">
              {opcoesModoEntrega.length > 1 && (
                <Segmented opcoes={opcoesModoEntrega} valor={modoEntrega} onValorChange={setModoEntrega} className="w-full" />
              )}

              {modoEntrega === 'guardar' && (
                <p className="rounded-xl bg-fundo px-3 py-2 text-xs text-tinta-suave">
                  Você paga só os produtos agora. Eles ficam guardados aqui até você voltar, escolher o que quer receber e pagar o frete separadamente.
                </p>
              )}

              <Campo label="Seu nome">
                <input value={nome} onChange={(e) => setNome(e.target.value)} placeholder="Maria Silva" className="w-full rounded-lg border border-tinta/20 bg-fundo px-3 py-2 text-tinta outline-none focus:border-acento" />
              </Campo>

              <Campo label="WhatsApp">
                <input value={telefone} onChange={(e) => setTelefone(e.target.value)} placeholder="(79) 99999-9999" className="w-full rounded-lg border border-tinta/20 bg-fundo px-3 py-2 text-tinta outline-none focus:border-acento" />
              </Campo>

              {modoEntrega === 'entrega' && (
                <div>
                  <span className="mb-2 block text-xs font-medium uppercase tracking-wide text-tinta-suave">Endereço de entrega</span>
                  <EnderecoCampos valor={enderecoValor} onChange={setEnderecoValor} />
                  {taxaEntregaTipo === 'combinado' && (
                    <p className="mt-1.5 text-xs text-tinta-suave">Taxa de entrega a combinar com a loja.</p>
                  )}
                  {taxaEntregaTipo === 'por_km' && (
                    <div className="mt-1.5">
                      {calculandoFrete && (
                        <p className="text-xs text-tinta-suave">Calculando frete...</p>
                      )}
                      {!calculandoFrete && freteCalculado !== null && distanciaKm !== null && (
                        <p className="text-xs text-emerald-600">
                          {distanciaKm.toFixed(1)} km — frete de R$ {freteCalculado.toFixed(2).replace('.', ',')}
                        </p>
                      )}
                      {!calculandoFrete && erroFrete && (
                        <p className="text-xs text-acento">{erroFrete}</p>
                      )}
                    </div>
                  )}
                </div>
              )}

              {modoPedido === 'agendado' && modoEntrega !== 'guardar' && (
                <div className="flex gap-3">
                  <Campo label="Data de retirada" className="flex-1">
                    <input
                      type="date"
                      value={data}
                      min={dataMinima(antecedenciaMinimaHoras)}
                      onChange={(e) => handleDataChange(e.target.value)}
                      className="w-full rounded-lg border border-tinta/20 bg-fundo px-3 py-2 text-tinta outline-none focus:border-acento"
                    />
                  </Campo>
                  <Campo label="Hora" className="w-28">
                    <input
                      type="time"
                      value={hora}
                      min={horaMinima(data, antecedenciaMinimaHoras)}
                      onChange={(e) => setHora(e.target.value)}
                      className="w-full rounded-lg border border-tinta/20 bg-fundo px-3 py-2 text-tinta outline-none focus:border-acento"
                    />
                  </Campo>
                </div>
              )}

              {modoPedido === 'agendado' && modoEntrega !== 'guardar' && antecedenciaMinimaHoras > 0 && (
                <p className="text-xs text-tinta-suave">
                  Essa loja exige pelo menos {antecedenciaMinimaHoras}h de antecedência.
                </p>
              )}
              {erro && <p className="text-sm text-acento">{erro}</p>}
            </div>
          )}
        </div>

        <div className="space-y-3 border-t border-tinta/10 px-6 py-4">
          <div className="space-y-1">
            {etapa === 'carrinho' && (
              <div className="mb-3 flex gap-2">
                <input
                  value={cupomCodigo}
                  onChange={(e) => { setCupomCodigo(e.target.value.toUpperCase()); setDesconto(0); setCupomMsg(null); }}
                  placeholder="Código do cupom"
                  className="min-w-0 flex-1 rounded-lg border border-tinta/20 bg-fundo px-3 py-2 font-carimbo text-sm tracking-widest text-tinta outline-none focus:border-acento"
                />
                {desconto > 0 ? (
                  <button onClick={removerCupom} className="rounded-lg border border-acento/30 px-3 py-2 text-xs text-acento">
                    Remover
                  </button>
                ) : (
                  <button onClick={aplicarCupom} disabled={validandoCupom || !cupomCodigo.trim()} className="rounded-lg bg-tinta px-3 py-2 text-xs font-semibold text-superficie disabled:opacity-40">
                    {validandoCupom ? '...' : 'Aplicar'}
                  </button>
                )}
              </div>
            )}
            {cupomMsg && (
              <p className={`mb-2 text-xs ${cupomMsg.tipo === 'ok' ? 'text-emerald-600' : 'text-acento'}`}>
                {cupomMsg.texto}
              </p>
            )}

            <div className="flex items-center justify-between text-sm">
              <span className="text-tinta-suave">Subtotal</span>
              <span className="text-tinta">R$ {total.toFixed(2).replace('.', ',')}</span>
            </div>
            {modoEntrega === 'entrega' && taxaEntregaTipo === 'fixa' && taxaEntregaValor > 0 && (
              <div className="flex items-center justify-between text-sm">
                <span className="text-tinta-suave">Taxa de entrega</span>
                <span className="text-tinta">R$ {taxaEntregaValor.toFixed(2).replace('.', ',')}</span>
              </div>
            )}
            {modoEntrega === 'entrega' && taxaEntregaTipo === 'combinado' && (
              <div className="flex items-center justify-between text-sm">
                <span className="text-tinta-suave">Taxa de entrega</span>
                <span className="text-tinta-suave italic">a combinar</span>
              </div>
            )}
            {modoEntrega === 'entrega' && taxaEntregaTipo === 'por_km' && freteCalculado !== null && (
              <div className="flex items-center justify-between text-sm">
                <span className="text-tinta-suave">Taxa de entrega ({distanciaKm?.toFixed(1)} km)</span>
                <span className="text-tinta">R$ {freteCalculado.toFixed(2).replace('.', ',')}</span>
              </div>
            )}
            {desconto > 0 && (
              <div className="flex items-center justify-between text-sm">
                <span className="text-emerald-600">Desconto ({cupomCodigo})</span>
                <span className="text-emerald-600">- R$ {desconto.toFixed(2).replace('.', ',')}</span>
              </div>
            )}
            <div className="flex items-center justify-between border-t border-tinta/10 pt-2">
              <span className="text-tinta-suave">Total</span>
              <span className="font-carimbo text-lg font-semibold text-tinta">
                R$ {totalComEntrega.toFixed(2).replace('.', ',')}
              </span>
            </div>
          </div>

          {etapa === 'carrinho' ? (
            <>
              {valorMinimoPedido > 0 && total < valorMinimoPedido && (
                <p className="text-center text-xs text-acento">
                  Pedido mínimo de R$ {valorMinimoPedido.toFixed(2).replace('.', ',')} — faltam R$ {(valorMinimoPedido - total).toFixed(2).replace('.', ',')}
                </p>
              )}
              <button
                onClick={() => setEtapa('dados')}
                disabled={(itens.length === 0 && combos.length === 0) || (valorMinimoPedido > 0 && total < valorMinimoPedido)}
                className="btn-neu-primario w-full py-3"
              >
                Continuar
              </button>
            </>
          ) : (
            <div className="flex gap-3">
              <button
                onClick={() => setEtapa('carrinho')}
                className="btn-neu-secundario py-3"
              >
                Voltar
              </button>
              <button
                onClick={confirmarPedido}
                disabled={enviando}
                className="btn-neu-primario flex-1 py-3"
              >
                {enviando ? 'Confirmando...' : 'Confirmar e pagar'}
              </button>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}