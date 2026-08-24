import { useState, useEffect, useRef } from 'react';
import { listarGuardados, cotarFreteGuardados, solicitarEntregaGuardados, criarCheckoutFrete } from '../api/guardados';
import type { ItemGuardado } from '../api/types';
import { Campo } from './Campo';
import { EnderecoCampos, enderecoVazio, enderecoParaTexto, enderecoPreenchido, type EnderecoValor } from './EnderecoCampos';

interface Props {
  aberto: boolean;
  onFechar: () => void;
  slug: string;
}

function normalizarTelefone(v: string) {
  const d = v.replace(/\D/g, '');
  return d.startsWith('55') ? d : `55${d}`;
}

function formatarData(iso: string) {
  return new Date(iso).toLocaleDateString('pt-BR', { day: '2-digit', month: '2-digit', year: 'numeric' });
}

export function GuardadosDrawer({ aberto, onFechar, slug }: Props) {
  const [etapa, setEtapa] = useState<'buscar' | 'entrega'>('buscar');

  const [telefone, setTelefone] = useState('');
  const [itens, setItens] = useState<ItemGuardado[] | null>(null);
  const [selecionados, setSelecionados] = useState<Set<number>>(new Set());
  const [buscando, setBuscando] = useState(false);
  const [erro, setErro] = useState<string | null>(null);

  const [nome, setNome] = useState('');
  const [enderecoValor, setEnderecoValor] = useState<EnderecoValor>(enderecoVazio);
  const enderecoTexto = enderecoParaTexto(enderecoValor);
  const [freteCalculado, setFreteCalculado] = useState<number | null>(null);
  const [distanciaKm, setDistanciaKm] = useState<number | null>(null);
  const [mesmaRegiao, setMesmaRegiao] = useState<boolean | null>(null);
  const [calculandoFrete, setCalculandoFrete] = useState(false);
  const [erroFrete, setErroFrete] = useState<string | null>(null);
  const [confirmando, setConfirmando] = useState(false);
  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  useEffect(() => {
    if (etapa !== 'entrega') return;
    if (debounceRef.current) clearTimeout(debounceRef.current);

    if (!enderecoPreenchido(enderecoValor) || selecionados.size === 0) {
      setFreteCalculado(null);
      setDistanciaKm(null);
      setMesmaRegiao(null);
      setErroFrete(null);
      return;
    }

    debounceRef.current = setTimeout(async () => {
      setCalculandoFrete(true);
      setErroFrete(null);
      try {
        const resultado = await cotarFreteGuardados(slug, normalizarTelefone(telefone), enderecoTexto, enderecoValor, Array.from(selecionados));
        setFreteCalculado(resultado.valor_frete);
        setDistanciaKm(resultado.distancia_km);
        setMesmaRegiao(resultado.mesma_regiao);
      } catch {
        setFreteCalculado(null);
        setDistanciaKm(null);
        setMesmaRegiao(null);
        setErroFrete('Não conseguimos calcular o frete pra esse endereço. Confere se está completo (rua, número, bairro, cidade).');
      } finally {
        setCalculandoFrete(false);
      }
    }, 900);

    return () => {
      if (debounceRef.current) clearTimeout(debounceRef.current);
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [enderecoTexto, etapa]);

  if (!aberto) return null;

  async function buscar() {
    if (!telefone.trim()) return;
    setBuscando(true);
    setErro(null);
    try {
      const resultado = await listarGuardados(slug, normalizarTelefone(telefone));
      setItens(resultado);
      setSelecionados(new Set());
      if (resultado.length === 0) setErro('Nenhum item guardado pra esse número.');
    } catch {
      setErro('Não foi possível buscar seus itens guardados.');
    } finally {
      setBuscando(false);
    }
  }

  function alternarSelecao(itemId: number) {
    setSelecionados((atual) => {
      const novo = new Set(atual);
      if (novo.has(itemId)) novo.delete(itemId);
      else novo.add(itemId);
      return novo;
    });
  }

  function reiniciar() {
    setEtapa('buscar');
    setItens(null);
    setSelecionados(new Set());
    setTelefone('');
    setNome('');
    setEnderecoValor(enderecoVazio);
    setFreteCalculado(null);
    setDistanciaKm(null);
    setMesmaRegiao(null);
    setErro(null);
  }

  function fecharTudo() {
    reiniciar();
    onFechar();
  }

  async function confirmarEntrega() {
    if (!nome.trim()) { setErro('Preenche seu nome.'); return; }
    if (!enderecoPreenchido(enderecoValor)) { setErro('Preenche o endereço de entrega (rua e cidade, pelo menos).'); return; }
    if (freteCalculado === null) { setErro('Aguarda o cálculo do frete antes de continuar.'); return; }

    setConfirmando(true);
    setErro(null);
    try {
      const solicitacao = await solicitarEntregaGuardados(slug, {
        cliente_nome: nome.trim(),
        cliente_telefone: normalizarTelefone(telefone),
        endereco: enderecoTexto,
        item_ids: Array.from(selecionados),
      }, enderecoValor);
      const { url } = await criarCheckoutFrete(solicitacao.id);
      window.location.href = url;
    } catch {
      setErro('Não foi possível confirmar a entrega. Alguns itens podem já ter sido reivindicados — tenta buscar de novo.');
      setConfirmando(false);
    }
  }

  const itensSelecionados = itens?.filter((item) => selecionados.has(item.id)) ?? [];

  return (
    <div className="fixed inset-0 z-30 flex items-end justify-center sm:items-center">
      <button aria-label="Fechar" onClick={fecharTudo} className="absolute inset-0 bg-tinta/50" />

      <div className="relative flex max-h-[85vh] w-full flex-col rounded-t-3xl bg-superficie sm:max-w-md sm:rounded-3xl">
        <div className="flex items-center justify-between border-b border-tinta/10 px-6 py-4">
          <h2 className="font-display text-xl tracking-wide text-tinta">
            {etapa === 'buscar' ? 'Itens guardados' : 'Entregar itens selecionados'}
          </h2>
          <button onClick={fecharTudo} className="text-xl text-tinta-suave hover:text-tinta">×</button>
        </div>

        <div className="flex-1 overflow-y-auto px-6 py-4 space-y-4">
          {etapa === 'buscar' && (
            <>
              <div className="flex gap-2">
                <input
                  value={telefone}
                  onChange={(e) => { setTelefone(e.target.value); setItens(null); setErro(null); }}
                  onKeyDown={(e) => e.key === 'Enter' && buscar()}
                  placeholder="(79) 99999-9999"
                  className="min-w-0 flex-1 rounded-lg border border-tinta/20 bg-fundo px-3 py-2 text-tinta outline-none focus:border-acento"
                />
                <button
                  onClick={buscar}
                  disabled={buscando || !telefone.trim()}
                  className="btn-neu-primario disabled:opacity-50"
                >
                  {buscando ? '...' : 'Buscar'}
                </button>
              </div>
              <p className="text-xs text-tinta-suave">
                Digite o WhatsApp usado na compra pra ver o que você tem guardado aqui.
              </p>

              {erro && <p className="text-sm text-acento">{erro}</p>}

              {itens && itens.length > 0 && (
                <ul className="space-y-2">
                  {itens.map((item) => (
                    <li key={item.id}>
                      <label className="flex cursor-pointer items-center gap-3 rounded-xl border border-tinta/10 bg-fundo p-3">
                        <input
                          type="checkbox"
                          checked={selecionados.has(item.id)}
                          onChange={() => alternarSelecao(item.id)}
                          className="h-4 w-4 accent-acento"
                        />
                        <div className="flex-1">
                          <p className="text-sm font-medium text-tinta">
                            {item.quantidade}× {item.produto_nome}
                            {item.variacao_nome && <span className="text-tinta-suave"> ({item.variacao_nome})</span>}
                          </p>
                          <p className="text-xs text-tinta-suave">Guardado desde {formatarData(item.guardado_desde)}</p>
                        </div>
                      </label>
                    </li>
                  ))}
                </ul>
              )}
            </>
          )}

          {etapa === 'entrega' && (
            <div className="space-y-4">
              <div className="rounded-xl bg-fundo p-3">
                <p className="mb-1 text-xs font-medium uppercase tracking-wide text-tinta-suave">Itens selecionados</p>
                {itensSelecionados.map((item) => (
                  <p key={item.id} className="text-sm text-tinta">
                    {item.quantidade}× {item.produto_nome}
                    {item.variacao_nome && <span className="text-tinta-suave"> ({item.variacao_nome})</span>}
                  </p>
                ))}
              </div>

              <Campo label="Seu nome">
                <input value={nome} onChange={(e) => setNome(e.target.value)} placeholder="Maria Silva" className="w-full rounded-lg border border-tinta/20 bg-fundo px-3 py-2 text-tinta outline-none focus:border-acento" />
              </Campo>

              <div>
                <span className="mb-2 block text-xs font-medium uppercase tracking-wide text-tinta-suave">Endereço de entrega</span>
                <EnderecoCampos valor={enderecoValor} onChange={setEnderecoValor} />
                <div className="mt-1.5">
                  {calculandoFrete && <p className="text-xs text-tinta-suave">Calculando frete...</p>}
                  {!calculandoFrete && freteCalculado !== null && distanciaKm !== null && (
                    <p className="text-xs text-emerald-600">
                      {distanciaKm.toFixed(1)} km ({mesmaRegiao ? 'mesma região' : 'fora da região'}) — frete de R$ {freteCalculado.toFixed(2).replace('.', ',')}
                    </p>
                  )}
                  {!calculandoFrete && erroFrete && <p className="text-xs text-acento">{erroFrete}</p>}
                </div>
              </div>

              {erro && <p className="text-sm text-acento">{erro}</p>}
            </div>
          )}
        </div>

        <div className="space-y-3 border-t border-tinta/10 px-6 py-4">
          {etapa === 'buscar' ? (
            <button
              onClick={() => setEtapa('entrega')}
              disabled={selecionados.size === 0}
              className="w-full rounded-full bg-acento py-3 font-semibold text-texto-claro transition disabled:opacity-40"
            >
              Continuar ({selecionados.size} {selecionados.size === 1 ? 'item' : 'itens'})
            </button>
          ) : (
            <div className="flex gap-3">
              <button onClick={() => setEtapa('buscar')} className="rounded-full border border-tinta/20 px-4 py-3 font-semibold text-tinta">
                Voltar
              </button>
              <button
                onClick={confirmarEntrega}
                disabled={confirmando}
                className="flex-1 rounded-full bg-acento py-3 font-semibold text-texto-claro transition disabled:opacity-60"
              >
                {confirmando ? 'Confirmando...' : 'Confirmar e pagar o frete'}
              </button>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
