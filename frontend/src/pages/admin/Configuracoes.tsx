import { useState, useEffect, type FormEvent, type ChangeEvent } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  buscarLoja,
  atualizarConfiguracoes,
  statusMercadoPago,
  iniciarOnboardingMercadoPago,
} from '../../api/admin';
import { enviarImagem, logoMiniatura } from '../../api/upload';
import { buscarConfiguracaoPlataforma, assinarSugestaoInteligente, cancelarAssinaturaSugestaoInteligente } from '../../api/sugestoes';
import { TEMAS } from '../../themes';
import { Campo } from '../../components/Campo';
import { QRCodeCardapio } from '../../components/QRCodeCardapio';
import { EnderecoCampos, enderecoVazio, enderecoParaTexto, enderecoParaCampos, enderecoPreenchido, type EnderecoValor } from '../../components/EnderecoCampos';
import { rotuloCatalogo } from '../../lib/utils';

const MARGENS = [0, 5, 10, 15, 20, 25, 30];

export function Configuracoes() {
  const queryClient = useQueryClient();

  const { data: loja, isLoading } = useQuery({ queryKey: ['loja'], queryFn: buscarLoja });
  const { data: mercadoPagoStatus } = useQuery({ queryKey: ['mercadopago-status'], queryFn: statusMercadoPago });
  const { data: configuracaoPlataforma } = useQuery({ queryKey: ['configuracao-plataforma'], queryFn: buscarConfiguracaoPlataforma });

  const [whatsapp, setWhatsapp] = useState('');
  const [logoUrl, setLogoUrl] = useState('');
  const [modoPedido, setModoPedido] = useState<'imediato' | 'agendado'>('imediato');
  const [antecedencia, setAntecedencia] = useState(24);
  const [abertura, setAbertura] = useState('');
  const [fechamento, setFechamento] = useState('');
  const [margem, setMargem] = useState(0);
  const [pausado, setPausado] = useState(false);
  const [mensagemPausa, setMensagemPausa] = useState('');
  const [aceitaRetirada, setAceitaRetirada] = useState(true);
  const [aceitaEntrega, setAceitaEntrega] = useState(false);
  const [aceitaGuardarEntregar, setAceitaGuardarEntregar] = useState(false);
  const [segmentoPrincipal, setSegmentoPrincipal] = useState<'alimenticio' | 'mercadoria'>('alimenticio');
  const [taxaTipo, setTaxaTipo] = useState<'fixa' | 'combinado' | 'por_km'>('combinado');
  const [taxaValor, setTaxaValor] = useState(0);
  const [taxaBase, setTaxaBase] = useState(0);
  const [taxaPorKm, setTaxaPorKm] = useState(0);
  const [enderecoValor, setEnderecoValor] = useState<EnderecoValor>(enderecoVazio);
  const [enderecoSalvo, setEnderecoSalvo] = useState('');
  const [valorMinimo, setValorMinimo] = useState(0);
  const [tema, setTema] = useState('kraft');
  const [sugestaoInteligenteAtiva, setSugestaoInteligenteAtiva] = useState(false);
  const [salvo, setSalvo] = useState(false);
  const [erro, setErro] = useState<string | null>(null);
  const [conectandoMercadoPago, setConectandoMercadoPago] = useState(false);
  const [assinandoSugestao, setAssinandoSugestao] = useState(false);
  const [enviandoLogo, setEnviandoLogo] = useState(false);
  const [erroLogo, setErroLogo] = useState<string | null>(null);

  useEffect(() => {
    if (loja) {
      setWhatsapp(loja.whatsapp_numero);
      setLogoUrl(loja.logo_url);
      setModoPedido(loja.modo_pedido ?? 'imediato');
      setAntecedencia(loja.antecedencia_minima_horas || 24);
      setAbertura(loja.horario_abertura ?? '');
      setFechamento(loja.horario_fechamento ?? '');
      setMargem(loja.margem_fechamento_minutos ?? 0);
      setPausado(loja.pausado ?? false);
      setMensagemPausa(loja.mensagem_pausa ?? '');
      setAceitaRetirada(loja.aceita_retirada ?? true);
      setAceitaEntrega(loja.aceita_entrega ?? false);
      setAceitaGuardarEntregar(loja.aceita_guardar_entregar ?? false);
      setSegmentoPrincipal(loja.segmento_principal ?? 'alimenticio');
      setTaxaTipo(loja.taxa_entrega_tipo ?? 'combinado');
      setTaxaValor(loja.taxa_entrega_valor ?? 0);
      setTaxaBase(loja.taxa_entrega_base ?? 0);
      setTaxaPorKm(loja.taxa_entrega_por_km ?? 0);
      setEnderecoSalvo(loja.endereco ?? '');
      setValorMinimo(loja.valor_minimo_pedido ?? 0);
      setTema(loja.tema ?? 'kraft');
      setSugestaoInteligenteAtiva(loja.sugestao_inteligente_ativa ?? false);
    }
  }, [loja]);

  const mutSalvar = useMutation({
    mutationFn: atualizarConfiguracoes,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['loja'] });
      setSalvo(true);
      setErro(null);
      setTimeout(() => setSalvo(false), 3000);
    },
    onError: () => setErro('Não foi possível salvar.'),
  });

  function montarPayload() {
    return {
      whatsapp_numero: whatsapp,
      logo_url: logoUrl,
      modo_pedido: modoPedido,
      antecedencia_minima_horas: antecedencia,
      horario_abertura: abertura,
      horario_fechamento: fechamento,
      margem_fechamento_minutos: margem,
      pausado,
      mensagem_pausa: mensagemPausa,
      aceita_retirada: aceitaRetirada,
      aceita_entrega: aceitaEntrega,
      aceita_guardar_entregar: aceitaGuardarEntregar,
      segmento_principal: segmentoPrincipal,
      taxa_entrega_tipo: taxaTipo,
      taxa_entrega_valor: taxaValor,
      taxa_entrega_base: taxaBase,
      taxa_entrega_por_km: taxaPorKm,
      endereco: enderecoPreenchido(enderecoValor) ? enderecoParaTexto(enderecoValor) : enderecoSalvo,
      ...(enderecoPreenchido(enderecoValor)
        ? {
            endereco_rua: enderecoParaCampos(enderecoValor).rua,
            endereco_numero: enderecoParaCampos(enderecoValor).numero,
            endereco_complemento: enderecoParaCampos(enderecoValor).complemento,
            endereco_bairro: enderecoParaCampos(enderecoValor).bairro,
            endereco_cidade: enderecoParaCampos(enderecoValor).cidade,
            endereco_estado: enderecoParaCampos(enderecoValor).estado,
            endereco_cep: enderecoParaCampos(enderecoValor).cep,
          }
        : {}),
      valor_minimo_pedido: valorMinimo,
      tema,
      sugestao_inteligente_ativa: sugestaoInteligenteAtiva,
    };
  }

  function salvar(e: FormEvent) {
    e.preventDefault();
    mutSalvar.mutate(montarPayload());
  }

  async function selecionarLogo(e: ChangeEvent<HTMLInputElement>) {
    const arquivo = e.target.files?.[0];
    if (!arquivo) return;
    setEnviandoLogo(true);
    setErroLogo(null);
    try {
      const url = await enviarImagem(arquivo);
      setLogoUrl(url);
      await atualizarConfiguracoes({ ...montarPayload(), logo_url: url });
      queryClient.invalidateQueries({ queryKey: ['loja'] });
    } catch {
      setErroLogo('Não foi possível enviar a imagem.');
    } finally {
      setEnviandoLogo(false);
    }
  }

  async function conectarMercadoPago() {
    setConectandoMercadoPago(true);
    try {
      const { url } = await iniciarOnboardingMercadoPago();
      window.location.href = url;
    } catch {
      setConectandoMercadoPago(false);
    }
  }

  async function assinarSugestao() {
    setAssinandoSugestao(true);
    try {
      const { url } = await assinarSugestaoInteligente();
      window.location.href = url;
    } catch {
      setAssinandoSugestao(false);
      setErro('Não foi possível iniciar a assinatura. Tenta de novo em instantes.');
    }
  }

  const mutCancelarSugestao = useMutation({
    mutationFn: cancelarAssinaturaSugestaoInteligente,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['loja'] });
      setSugestaoInteligenteAtiva(false);
    },
    onError: () => setErro('Não foi possível cancelar a assinatura. Tenta de novo em instantes.'),
  });

  if (isLoading) return <p className="text-tinta-suave">Carregando...</p>;

  // O endereço de origem (e a taxa base/por km) é usado tanto pelo frete
  // por quilometragem da entrega imediata quanto pelo frete regional de
  // itens guardados — por isso precisa aparecer se qualquer um dos dois
  // estiver ativo, não só quando taxaTipo === 'por_km'.
  const precisaEnderecoKm = (aceitaEntrega && taxaTipo === 'por_km') || aceitaGuardarEntregar;

  return (
    <div className="space-y-8">
      <h1 className="font-display text-2xl tracking-wide text-tinta">Configurações</h1>

      {/* Pagamento */}
      <section className="rounded-2xl bg-superficie p-5 shadow-sm">
        <h2 className="font-display text-lg tracking-wide text-tinta">Pagamento</h2>
        <p className="mt-1 text-sm text-tinta-suave">
          O Mercado Pago processa os pagamentos dos seus pedidos e te paga direto na sua conta.
        </p>
        <div className="mt-4 flex items-center justify-between rounded-xl bg-fundo px-4 py-3">
          <span className="text-sm font-medium text-tinta">
            {mercadoPagoStatus?.mercadopago_conectado ? 'Conta conectada' : 'Conta não conectada'}
          </span>
          <span className={`h-2.5 w-2.5 rounded-full ${mercadoPagoStatus?.mercadopago_conectado ? 'bg-emerald-500' : 'bg-tinta/20'}`} />
        </div>
        <button
          onClick={conectarMercadoPago}
          disabled={conectandoMercadoPago}
          className="mt-4 rounded-full bg-acento px-4 py-2 text-sm font-semibold text-superficie disabled:opacity-60"
        >
          {conectandoMercadoPago ? 'Abrindo...' : mercadoPagoStatus?.mercadopago_conectado ? 'Revisar dados no Mercado Pago' : 'Conectar conta de pagamento'}
        </button>
      </section>

      <form onSubmit={salvar} className="space-y-6 rounded-2xl bg-superficie p-5 shadow-sm">
        <h2 className="font-display text-lg tracking-wide text-tinta">Loja</h2>

        {/* Logo */}
        <div>
          <span className="mb-2 block text-xs font-medium uppercase tracking-wide text-tinta-suave">Logo da loja</span>
          <div className="flex items-center gap-4">
            <div className="flex h-16 w-16 shrink-0 items-center justify-center overflow-hidden rounded-full border-2 border-dashed border-tinta/25 bg-fundo">
              {logoUrl ? (
                <img src={logoMiniatura(logoUrl)} alt="Logo" className="h-full w-full object-cover" />
              ) : (
                <span className="font-display text-xl text-tinta/30">?</span>
              )}
            </div>
            <label className="cursor-pointer rounded-full border border-tinta/20 px-4 py-2 text-sm font-semibold text-tinta hover:border-acento">
              {enviandoLogo ? 'Enviando...' : logoUrl ? 'Trocar imagem' : 'Enviar logo'}
              <input type="file" accept="image/*" onChange={selecionarLogo} disabled={enviandoLogo} className="hidden" />
            </label>
          </div>
          {erroLogo && <p className="mt-2 text-sm text-acento">{erroLogo}</p>}
        </div>

        {/* WhatsApp */}
        <Campo label="WhatsApp pra receber avisos de pedido">
          <input
            required
            value={whatsapp}
            onChange={(e) => setWhatsapp(e.target.value)}
            placeholder="5579999999999"
            className="w-full rounded-lg border border-tinta/20 bg-fundo px-3 py-2 text-tinta outline-none focus:border-acento"
          />
          <span className="mt-1 block text-xs text-tinta-suave">DDI + DDD + número (ex: 5579999999999).</span>
        </Campo>

        {/* Modo de pedido */}
        <div className="space-y-3 rounded-xl border border-tinta/10 bg-fundo p-4">
          <p className="text-xs font-medium uppercase tracking-wide text-tinta-suave">Modo de pedido</p>
          <div className="flex flex-col gap-2">
            <label className="flex items-start gap-3 cursor-pointer">
              <input type="radio" name="modo" value="imediato" checked={modoPedido === 'imediato'} onChange={() => setModoPedido('imediato')} className="mt-0.5 accent-acento" />
              <div>
                <p className="text-sm font-medium text-tinta">Entrega imediata</p>
                <p className="text-xs text-tinta-suave">Cliente faz o pedido sem agendar data — ideal pra produtos já prontos.</p>
              </div>
            </label>
            <label className="flex items-start gap-3 cursor-pointer">
              <input type="radio" name="modo" value="agendado" checked={modoPedido === 'agendado'} onChange={() => setModoPedido('agendado')} className="mt-0.5 accent-acento" />
              <div>
                <p className="text-sm font-medium text-tinta">Retirada agendada</p>
                <p className="text-xs text-tinta-suave">Cliente escolhe a data com antecedência mínima — ideal pra encomendas.</p>
              </div>
            </label>
          </div>
          {modoPedido === 'agendado' && (
            <Campo label="Antecedência mínima (horas)">
              <input
                type="number"
                min="1"
                value={antecedencia}
                onChange={(e) => setAntecedencia(parseInt(e.target.value) || 1)}
                className="w-full rounded-lg border border-tinta/20 bg-superficie px-3 py-2 text-tinta outline-none focus:border-acento"
              />
              <span className="mt-1 block text-xs text-tinta-suave">Ex: 24 = cliente precisa pedir com pelo menos 24h de antecedência.</span>
            </Campo>
          )}
        </div>

        {/* Horário de funcionamento */}
        <div className="space-y-3 rounded-xl border border-tinta/10 bg-fundo p-4">
          <p className="text-xs font-medium uppercase tracking-wide text-tinta-suave">Horário de funcionamento <span className="normal-case font-normal">(opcional)</span></p>
          <div className="grid grid-cols-2 gap-3">
            <div className="min-w-0">
              <Campo label="Abre">
                <input type="time" value={abertura} onChange={(e) => setAbertura(e.target.value)}
                  className="w-full min-w-0 rounded-lg border border-tinta/20 bg-superficie px-2 py-2 text-sm text-tinta outline-none focus:border-acento" />
              </Campo>
            </div>
            <div className="min-w-0">
              <Campo label="Fecha">
                <input type="time" value={fechamento} onChange={(e) => setFechamento(e.target.value)}
                  className="w-full min-w-0 rounded-lg border border-tinta/20 bg-superficie px-2 py-2 text-sm text-tinta outline-none focus:border-acento" />
              </Campo>
            </div>
          </div>
          <Campo label="Parar de aceitar pedidos antes de fechar">
            <select
              value={margem}
              onChange={(e) => setMargem(parseInt(e.target.value))}
              className="w-full rounded-lg border border-tinta/20 bg-superficie px-3 py-2 text-tinta outline-none focus:border-acento"
            >
              {MARGENS.map((m) => (
                <option key={m} value={m}>{m === 0 ? 'No horário de fechamento' : `${m} min antes`}</option>
              ))}
            </select>
          </Campo>
        </div>

        {/* Pausar produção */}
        <div className="space-y-3 rounded-xl border border-tinta/10 bg-fundo p-4">
          <label className="flex items-center gap-2 cursor-pointer">
            <input type="checkbox" checked={pausado} onChange={(e) => setPausado(e.target.checked)} className="h-4 w-4 accent-acento" />
            <span className="text-sm font-medium text-tinta">Pausar produção</span>
          </label>
          {pausado && (
            <Campo label="Aviso para o cliente">
              <input
                value={mensagemPausa}
                onChange={(e) => setMensagemPausa(e.target.value)}
                placeholder="Ex: Férias até dia 10/07. Voltamos em breve!"
                className="w-full rounded-lg border border-tinta/20 bg-superficie px-3 py-2 text-tinta outline-none focus:border-acento"
              />
            </Campo>
          )}
          <p className="text-xs text-tinta-suave">
            Fecha a loja temporariamente com uma mensagem personalizada exibida pro cliente.
          </p>
        </div>

        {/* Modos de recebimento */}
        <div className="space-y-3 rounded-xl border border-tinta/10 bg-fundo p-4">
          <p className="text-xs font-medium uppercase tracking-wide text-tinta-suave">Modos de recebimento</p>

          <label className="flex items-center gap-2 cursor-pointer">
            <input type="checkbox" checked={aceitaRetirada} onChange={(e) => setAceitaRetirada(e.target.checked)} className="h-4 w-4 accent-acento" />
            <span className="text-sm text-tinta">Retirada no local</span>
          </label>

          <label className="flex items-center gap-2 cursor-pointer">
            <input type="checkbox" checked={aceitaEntrega} onChange={(e) => setAceitaEntrega(e.target.checked)} className="h-4 w-4 accent-acento" />
            <span className="text-sm text-tinta">Entrega em domicílio</span>
          </label>

          {aceitaEntrega && (
            <div className="space-y-3 pt-1">
              <p className="text-xs font-medium text-tinta-suave">Taxa de entrega</p>
              <div className="flex flex-col gap-2">
                <label className="flex items-start gap-3 cursor-pointer">
                  <input type="radio" name="taxaTipo" value="fixa" checked={taxaTipo === 'fixa'} onChange={() => setTaxaTipo('fixa')} className="mt-0.5 accent-acento" />
                  <div>
                    <p className="text-sm font-medium text-tinta">Taxa fixa</p>
                    <p className="text-xs text-tinta-suave">Você define o valor agora e é cobrado automaticamente no checkout.</p>
                  </div>
                </label>
                {taxaTipo === 'fixa' && (
                  <Campo label="Valor da entrega (R$)">
                    <input
                      type="number"
                      step="0.50"
                      min="0"
                      value={taxaValor || ''}
                      onChange={(e) => setTaxaValor(parseFloat(e.target.value) || 0)}
                      placeholder="0,00"
                      className="w-full rounded-lg border border-tinta/20 bg-superficie px-3 py-2 text-tinta outline-none focus:border-acento"
                    />
                  </Campo>
                )}

                <label className="flex items-start gap-3 cursor-pointer">
                  <input type="radio" name="taxaTipo" value="por_km" checked={taxaTipo === 'por_km'} onChange={() => setTaxaTipo('por_km')} className="mt-0.5 accent-acento" />
                  <div>
                    <p className="text-sm font-medium text-tinta">Por quilometragem</p>
                    <p className="text-xs text-tinta-suave">Calculado automaticamente com base na distância até o endereço do cliente. O cliente já vê o valor no carrinho. Configurado logo abaixo, em "Endereço de origem".</p>
                  </div>
                </label>

                <label className="flex items-start gap-3 cursor-pointer">
                  <input type="radio" name="taxaTipo" value="combinado" checked={taxaTipo === 'combinado'} onChange={() => setTaxaTipo('combinado')} className="mt-0.5 accent-acento" />
                  <div>
                    <p className="text-sm font-medium text-tinta">A combinar</p>
                    <p className="text-xs text-tinta-suave">Ideal pra entregas muito variáveis (distâncias longas, itens grandes, casos especiais). O cliente informa o endereço, mas você negocia o valor direto por WhatsApp antes de confirmar.</p>
                  </div>
                </label>
              </div>
            </div>
          )}

          {!aceitaRetirada && !aceitaEntrega && (
            <p className="text-xs text-acento">Ative pelo menos um modo de recebimento.</p>
          )}
        </div>

        {/* Segmento principal — definitivo desde o cadastro (ver
            LojaService no backend). Só um punhado de contas de teste/
            administração consegue mudar depois; pra qualquer outra loja
            o seletor fica travado, só informativo. */}
        <div className="space-y-3 rounded-xl border border-tinta/10 bg-fundo p-4">
          <p className="text-xs font-medium uppercase tracking-wide text-tinta-suave">O que sua loja vende principalmente?</p>
          <div className="grid grid-cols-2 gap-2">
            <button
              type="button"
              disabled={!loja?.pode_editar_segmento}
              onClick={() => setSegmentoPrincipal('alimenticio')}
              className={`rounded-lg border px-3 py-2 text-sm font-medium transition ${
                segmentoPrincipal === 'alimenticio'
                  ? 'border-acento bg-acento/10 text-acento'
                  : 'border-tinta/20 text-tinta-suave hover:border-tinta/40'
              } ${!loja?.pode_editar_segmento ? 'cursor-not-allowed opacity-60 hover:border-tinta/20' : ''}`}
            >
              Comida e bebida
            </button>
            <button
              type="button"
              disabled={!loja?.pode_editar_segmento}
              onClick={() => setSegmentoPrincipal('mercadoria')}
              className={`rounded-lg border px-3 py-2 text-sm font-medium transition ${
                segmentoPrincipal === 'mercadoria'
                  ? 'border-acento bg-acento/10 text-acento'
                  : 'border-tinta/20 text-tinta-suave hover:border-tinta/40'
              } ${!loja?.pode_editar_segmento ? 'cursor-not-allowed opacity-60 hover:border-tinta/20' : ''}`}
            >
              Outros produtos
            </button>
          </div>
          <p className="text-xs text-tinta-suave">
            {loja?.pode_editar_segmento
              ? 'Define o tipo padrão dos novos produtos que você cadastrar.'
              : 'Escolha definida no cadastro e não pode ser alterada depois — muda o comportamento de vários recursos da loja (variações, catálogo, categorias sugeridas). Fale com a Drenux se precisar mudar.'}
          </p>
        </div>

        {/* Guardar e entregar depois */}
        <div className="space-y-3 rounded-xl border border-tinta/10 bg-fundo p-4">
          <label className="flex items-center gap-2 cursor-pointer">
            <input type="checkbox" checked={aceitaGuardarEntregar} onChange={(e) => setAceitaGuardarEntregar(e.target.checked)} className="h-4 w-4 accent-acento" />
            <span className="text-sm font-medium text-tinta">Guardar e entregar depois</span>
          </label>
          <p className="text-xs text-tinta-suave">
            O cliente compra e paga produtos do tipo "Mercadoria" agora, você guarda por tempo indeterminado, e ele volta depois pra escolher o que quer receber e pagar só o frete. Não disponível pra produtos alimentícios (risco de segurança alimentar). Marque o tipo de cada produto em Produtos.
          </p>
        </div>

        {/* Sugestão Inteligente — recurso pago à parte (Fase 6), taxa fixa
            mensal em qualquer plano. O toggle liga/desliga a exibição pro
            cliente final livremente, contratada ou não — sem contratar, a
            loja tem direito a 1 vínculo grátis que aparece de verdade se
            o toggle estiver ligado (a quantidade é limitada em "Sugestão
            Inteligente" no menu, não a exibição aqui). */}
        <div className="space-y-3 rounded-xl border border-tinta/10 bg-fundo p-4">
          <p className="text-xs font-medium uppercase tracking-wide text-tinta-suave">Sugestão Inteligente</p>
          <label className="flex items-center gap-2 cursor-pointer">
            <input
              type="checkbox"
              checked={sugestaoInteligenteAtiva}
              onChange={(e) => setSugestaoInteligenteAtiva(e.target.checked)}
              className="h-4 w-4 accent-acento"
            />
            <span className="text-sm font-medium text-tinta">Mostrar sugestões no carrinho do cliente</span>
          </label>
          {loja?.sugestao_inteligente_contratada ? (
            <>
              <p className="text-xs text-tinta-suave">
                Sugere produtos complementares (estilo totem de fastfood) quando o cliente revisa o carrinho antes de finalizar, com base nos vínculos configurados em "Sugestão Inteligente" no menu.
              </p>
              <button
                type="button"
                onClick={() => { if (confirm('Cancelar a assinatura da Sugestão Inteligente? Os vínculos além do limite grátis ficam ocultos até você assinar de novo.')) mutCancelarSugestao.mutate(); }}
                disabled={mutCancelarSugestao.isPending}
                className="rounded-full border border-acento/30 px-4 py-2 text-sm font-semibold text-acento disabled:opacity-60"
              >
                {mutCancelarSugestao.isPending ? 'Cancelando...' : 'Cancelar assinatura'}
              </button>
            </>
          ) : (
            <>
              <p className="text-xs text-tinta-suave">
                Você pode testar grátis com 1 vínculo (configura em "Sugestão Inteligente" no menu) — funciona de verdade pro cliente final. Recurso completo, sem limite de vínculos, é pago à parte
                {configuracaoPlataforma && (
                  <> — R$ {configuracaoPlataforma.sugestao_inteligente_preco_mensal.toFixed(2).replace('.', ',')}/mês</>
                )}
                .
              </p>
              <button
                type="button"
                onClick={assinarSugestao}
                disabled={assinandoSugestao}
                className="rounded-full bg-acento px-4 py-2 text-sm font-semibold text-superficie disabled:opacity-60"
              >
                {assinandoSugestao ? 'Abrindo pagamento...' : 'Assinar'}
              </button>
            </>
          )}
        </div>

        {/* Endereço de origem — usado pelo frete por km da entrega imediata e pelo frete de itens guardados */}
        {precisaEnderecoKm && (
          <div className="space-y-3 rounded-xl border border-tinta/10 bg-fundo p-4">
            <p className="text-xs font-medium uppercase tracking-wide text-tinta-suave">Endereço de origem</p>
            {enderecoSalvo && !enderecoPreenchido(enderecoValor) && (
              <p className="text-xs text-tinta-suave">
                Endereço atual: <span className="text-tinta">{enderecoSalvo}</span>. Preenche os campos abaixo só se quiser atualizar.
              </p>
            )}
            <EnderecoCampos valor={enderecoValor} onChange={setEnderecoValor} />
            <p className="text-xs text-tinta-suave">
              Usado pra calcular a distância até o cliente. Se souber o CEP, preenche ele primeiro — o resto é preenchido sozinho.
            </p>
            <div className="grid grid-cols-2 gap-3">
              <Campo label="Taxa base (R$)">
                <input
                  type="number"
                  step="0.50"
                  min="0"
                  value={taxaBase || ''}
                  onChange={(e) => setTaxaBase(parseFloat(e.target.value) || 0)}
                  placeholder="0,00"
                  className="w-full rounded-lg border border-tinta/20 bg-superficie px-3 py-2 text-tinta outline-none focus:border-acento"
                />
              </Campo>
              <Campo label="Valor por km (R$)">
                <input
                  type="number"
                  step="0.10"
                  min="0"
                  value={taxaPorKm || ''}
                  onChange={(e) => setTaxaPorKm(parseFloat(e.target.value) || 0)}
                  placeholder="0,00"
                  className="w-full rounded-lg border border-tinta/20 bg-superficie px-3 py-2 text-tinta outline-none focus:border-acento"
                />
              </Campo>
            </div>
            <p className="text-xs text-tinta-suave">
              Exemplo: taxa base R$ 5,00 + R$ 1,50/km → um cliente a 4km paga R$ 11,00 de entrega. Vale tanto pra entrega imediata por km quanto pra entrega de itens guardados dentro da mesma cidade/estado da loja — fora dessa região, o frete de guardados é estimado por peso + distância.
            </p>
          </div>
        )}

        {/* Valor mínimo de pedido */}
        <Campo label="Pedido mínimo (R$)">
          <input
            type="number"
            step="0.50"
            min="0"
            value={valorMinimo || ''}
            onChange={(e) => setValorMinimo(parseFloat(e.target.value) || 0)}
            placeholder="Sem mínimo"
            className="w-full rounded-lg border border-tinta/20 bg-fundo px-3 py-2 text-tinta outline-none focus:border-acento"
          />
          <span className="mt-1 block text-xs text-tinta-suave">
            Deixa em branco pra não ter mínimo. Calculado sobre o subtotal (sem taxa de entrega).
          </span>
        </Campo>

        {/* Seletor de tema */}
        <div className="space-y-3">
          <p className="text-xs font-medium uppercase tracking-wide text-tinta-suave">
            Tema do {rotuloCatalogo(loja?.segmento_principal)}
          </p>
          <div className="grid grid-cols-4 gap-2">
            {TEMAS.map((t) => (
              <button
                key={t.id}
                type="button"
                onClick={() => setTema(t.id)}
                className={`rounded-xl border-2 p-2 text-left transition ${
                  tema === t.id ? 'border-acento' : 'border-tinta/10 hover:border-tinta/25'
                }`}
              >
                <div
                  className="mb-1.5 h-6 w-full rounded-lg"
                  style={{ background: t.acento }}
                />
                <div
                  className="mb-1 h-1.5 w-full rounded"
                  style={{ background: t.fundo }}
                />
                <p className="truncate text-xs font-medium text-tinta">{t.nome}</p>
              </button>
            ))}
          </div>
          {tema && (
            <p className="text-xs text-tinta-suave">
              {TEMAS.find((t) => t.id === tema)?.descricao}
            </p>
          )}
        </div>

        {erro && <p className="text-sm text-acento">{erro}</p>}
        {salvo && <p className="text-sm text-emerald-600">Salvo!</p>}

        <button
          type="submit"
          disabled={mutSalvar.isPending}
          className="rounded-full bg-acento px-4 py-2 text-sm font-semibold text-superficie disabled:opacity-60"
        >
          {mutSalvar.isPending ? 'Salvando...' : 'Salvar'}
        </button>
      </form>
      {loja && <QRCodeCardapio slug={loja.slug} nomeLoja={loja.nome} segmentoLoja={loja.segmento_principal} />}
    </div>
  );
}