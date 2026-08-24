import { useState } from 'react';
import { Link } from 'react-router-dom';
import { buscarHistorico } from '../api/historico';
import { useCartStore } from '../store/cartStore';
import type { Pedido, Produto } from '../api/types';

interface Props {
  aberto: boolean;
  onFechar: () => void;
  slug: string;
  produtos: Produto[]; // lista atual do cardápio pra validar disponibilidade
  onAbrirCarrinho: () => void;
}

function formatarData(iso: string) {
  return new Date(iso).toLocaleDateString('pt-BR', {
    day: '2-digit', month: '2-digit', year: 'numeric',
  });
}

function normalizarTelefone(v: string) {
  const d = v.replace(/\D/g, '');
  return d.startsWith('55') ? d : `55${d}`;
}

export function HistoricoDrawer({ aberto, onFechar, slug, produtos, onAbrirCarrinho }: Props) {
  const adicionar = useCartStore((s) => s.adicionar);

  const [telefone, setTelefone] = useState('');
  const [pedidos, setPedidos] = useState<Pedido[] | null>(null);
  const [buscando, setBuscando] = useState(false);
  const [erro, setErro] = useState<string | null>(null);

  if (!aberto) return null;

  async function buscar() {
    if (!telefone.trim()) return;
    setBuscando(true);
    setErro(null);
    try {
      const resultado = await buscarHistorico(slug, normalizarTelefone(telefone));
      setPedidos(resultado);
      if (resultado.length === 0) setErro('Nenhum pedido encontrado pra esse número.');
    } catch {
      setErro('Não foi possível buscar o histórico.');
    } finally {
      setBuscando(false);
    }
  }

  function repetirPedido(pedido: Pedido) {
    let adicionados = 0;
    for (const item of pedido.itens) {
      const produto = produtos.find((p) => p.id === item.produto_id && p.disponivel);
      if (!produto) continue;
      const variacao = item.variacao_nome
        ? produto.variacoes?.find((v) => v.nome === item.variacao_nome && v.disponivel)
        : undefined;
      for (let i = 0; i < item.quantidade; i++) {
        adicionar(produto, variacao);
      }
      adicionados++;
    }
    onFechar();
    onAbrirCarrinho();
    if (adicionados === 0) {
      // todos os produtos foram removidos da loja
    }
  }

  return (
    <div className="fixed inset-0 z-30 flex items-end justify-center sm:items-center">
      <button aria-label="Fechar" onClick={onFechar} className="absolute inset-0 bg-tinta/50" />

      <div className="relative flex max-h-[85vh] w-full flex-col rounded-t-3xl bg-superficie sm:max-w-md sm:rounded-3xl">
        {/* Cabeçalho */}
        <div className="flex items-center justify-between border-b border-tinta/10 px-6 py-4">
          <h2 className="font-display text-xl tracking-wide text-tinta">Meus pedidos</h2>
          <button onClick={onFechar} className="text-xl text-tinta-suave hover:text-tinta">×</button>
        </div>

        <div className="flex-1 overflow-y-auto px-6 py-4 space-y-4">
          {/* Campo de telefone */}
          <div className="flex gap-2">
            <input
              value={telefone}
              onChange={(e) => { setTelefone(e.target.value); setPedidos(null); setErro(null); }}
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
            Digite o WhatsApp usado no pedido pra ver seu histórico.
          </p>

          {erro && <p className="text-sm text-acento">{erro}</p>}

          {/* Lista de pedidos */}
          {pedidos && pedidos.length > 0 && (
            <ul className="space-y-3">
              {pedidos.map((pedido) => (
                <li key={pedido.id} className="rounded-2xl border border-tinta/10 bg-fundo p-4">
                  <div className="flex items-start justify-between gap-3">
                    <div>
                      <p className="text-xs text-tinta-suave">{formatarData(pedido.created_at)}</p>
                      <div className="mt-1 space-y-0.5">
                        {pedido.itens.map((item) => (
                          <p key={item.id} className="text-sm text-tinta">
                            {item.quantidade}× {item.produto_nome}
                            {item.variacao_nome && (
                              <span className="text-tinta-suave"> ({item.variacao_nome})</span>
                            )}
                          </p>
                        ))}
                      </div>
                      <p className="mt-1.5 font-carimbo text-sm font-semibold text-tinta">
                        R$ {pedido.total.toFixed(2).replace('.', ',')}
                      </p>
                    </div>
                    <button
                      onClick={() => repetirPedido(pedido)}
                      className="shrink-0 rounded-full bg-acento px-3 py-1.5 text-xs font-semibold text-texto-claro"
                    >
                      Repetir
                    </button>
                  </div>

                  {/* Rastreamento — só aparece pra pedidos de entrega que
                      já saíram (ou já foram entregues) */}
                  {pedido.modo_entrega === 'entrega' &&
                    (pedido.status_entrega === 'saiu_para_entrega' || pedido.status_entrega === 'entregue') && (
                      <Link
                        to={`/${slug}/pedido/${pedido.id}/rastrear?telefone=${normalizarTelefone(telefone)}`}
                        onClick={onFechar}
                        className="mt-3 block rounded-full border border-acento/30 px-3 py-1.5 text-center text-xs font-semibold text-acento"
                      >
                        {pedido.status_entrega === 'entregue' ? '✅ Pedido entregue' : '🛵 Rastrear entrega'}
                      </Link>
                    )}
                </li>
              ))}
            </ul>
          )}
        </div>
      </div>
    </div>
  );
}