import { useState } from 'react';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import {
  buscarLoja, buscarRelatorioEstoque, buscarMovimentacoesEstoque, reporEstoque, ajustarEstoque,
  buscarListaDeCompras, buscarRelatoriosAvancados,
  type ItemEstoque, type MovimentacaoEstoque,
} from '../../api/admin';

// Rótulo + tom de cada tipo de movimentação, pro histórico do modal.
const ROTULO_MOVIMENTO: Record<MovimentacaoEstoque['tipo'], string> = {
  venda: 'Venda',
  reposicao: 'Reposição',
  ajuste: 'Ajuste manual',
};

function formatarData(iso: string) {
  return new Date(iso).toLocaleString('pt-BR', { day: '2-digit', month: '2-digit', hour: '2-digit', minute: '2-digit' });
}

function tituloItem(item: ItemEstoque) {
  return item.variacao_id ? `${item.produto_nome} — ${item.variacao_nome}` : item.produto_nome;
}

export function Estoque() {
  const queryClient = useQueryClient();
  const { data: loja } = useQuery({ queryKey: ['loja'], queryFn: buscarLoja });
  const { data: itens, isLoading } = useQuery({ queryKey: ['estoque'], queryFn: buscarRelatorioEstoque });

  const completo = loja?.plano === 'scale';
  const [itemSelecionado, setItemSelecionado] = useState<ItemEstoque | null>(null);

  const { data: listaCompras } = useQuery({ queryKey: ['lista-compras'], queryFn: buscarListaDeCompras, enabled: completo });
  const { data: relatorios } = useQuery({ queryKey: ['relatorios-estoque'], queryFn: buscarRelatoriosAvancados, enabled: completo });

  if (loja && loja.plano !== 'pro' && loja.plano !== 'scale') {
    return (
      <div className="space-y-3">
        <h1 className="font-display text-2xl tracking-wide text-tinta">Estoque</h1>
        <div className="rounded-2xl bg-superficie p-6 text-center shadow-sm">
          <p className="text-sm text-tinta">
            O relatório de estoque é um recurso do plano Pro — o controle completo, com reposição
            guiada e histórico de movimentação, é exclusivo do Scale.
          </p>
          <p className="mt-2 text-xs text-tinta-suave">Ative o Pro ou o Scale em Meu Plano pra liberar.</p>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      <h1 className="font-display text-2xl tracking-wide text-tinta">Estoque</h1>

      {!completo && (
        <div className="rounded-xl bg-douro/10 px-4 py-3 text-sm text-tinta">
          Reposição guiada e histórico completo de movimentação — disponível no plano Scale.
        </div>
      )}

      {isLoading && <p className="text-tinta-suave">Carregando...</p>}

      {itens && itens.length === 0 && (
        <p className="rounded-2xl bg-superficie p-6 text-center text-sm text-tinta-suave shadow-sm">
          Nenhum produto com controle de estoque ativado ainda. Ative "estoque atual" ao cadastrar
          ou editar um produto/variação pra ele aparecer aqui.
        </p>
      )}

      {itens && itens.length > 0 && (
        <ul className="space-y-2">
          {itens.map((item) => (
            <li
              key={`${item.produto_id}-${item.variacao_id ?? 'geral'}`}
              className={`flex items-center justify-between gap-3 rounded-xl bg-superficie px-4 py-3 shadow-sm ${
                item.estoque_atual === 0 ? 'ring-1 ring-acento/50' : ''
              }`}
            >
              <div>
                <p className="text-sm font-medium text-tinta">{tituloItem(item)}</p>
                <p className="text-xs text-tinta-suave">
                  {item.estoque_atual === 0
                    ? 'Esgotado'
                    : item.critico
                      ? `Estoque baixo — ${item.estoque_atual} unidade(s)`
                      : `${item.estoque_atual} unidade(s)`}
                  {item.estoque_alerta !== null && ` · alerta em ${item.estoque_alerta}`}
                </p>
              </div>
              {completo && (
                <button
                  onClick={() => setItemSelecionado(item)}
                  className="shrink-0 rounded-full border border-tinta/20 px-3 py-1.5 text-xs font-semibold text-tinta hover:border-acento hover:text-acento"
                >
                  Gerenciar
                </button>
              )}
            </li>
          ))}
        </ul>
      )}

      {itemSelecionado && (
        <ModalEstoque
          item={itemSelecionado}
          onFechar={() => setItemSelecionado(null)}
          onAtualizado={() => queryClient.invalidateQueries({ queryKey: ['estoque'] })}
        />
      )}

      {completo && listaCompras && listaCompras.length > 0 && (
        <div className="space-y-2">
          <h2 className="font-display text-lg tracking-wide text-tinta">Lista de compras</h2>
          <p className="text-xs text-tinta-suave">Insumos abaixo do alerta configurado.</p>
          <ul className="space-y-2">
            {listaCompras.map((item) => (
              <li key={item.insumo_id} className="rounded-xl bg-superficie px-4 py-3 shadow-sm">
                <p className="text-sm font-medium text-tinta">{item.nome}</p>
                <p className="text-xs text-tinta-suave">
                  {item.estoque_atual}{item.unidade_uso} atual · alerta em {item.estoque_alerta}{item.unidade_uso} · comprar
                  pelo menos {item.deficit_unidade_compra.toFixed(2)} {item.unidade_compra}
                </p>
                {item.produtos_afetados.length > 0 && (
                  <p className="text-xs text-tinta-suave">Usado em: {item.produtos_afetados.join(', ')}</p>
                )}
              </li>
            ))}
          </ul>
        </div>
      )}

      {completo && relatorios && (
        <div className="space-y-4">
          <h2 className="font-display text-lg tracking-wide text-tinta">Relatórios avançados</h2>

          <div className="rounded-xl bg-superficie p-4 shadow-sm">
            <p className="text-xs font-semibold uppercase tracking-wide text-tinta-suave">Valor parado em estoque (insumos)</p>
            <p className="font-carimbo text-lg text-tinta">R$ {relatorios.valor_parado_estoque.toFixed(2).replace('.', ',')}</p>
            <p className="mt-1 text-xs text-tinta-suave">{relatorios.valor_parado_observacao}</p>
          </div>

          <div className="rounded-xl bg-superficie p-4 shadow-sm">
            <p className="mb-2 text-xs font-semibold uppercase tracking-wide text-tinta-suave">Insumos que mais saem (30 dias)</p>
            {relatorios.insumos_que_mais_saem.length === 0 ? (
              <p className="text-sm text-tinta-suave">Nenhum consumo registrado nos últimos 30 dias.</p>
            ) : (
              <ul className="space-y-1 text-sm text-tinta">
                {relatorios.insumos_que_mais_saem.map((i) => (
                  <li key={i.insumo_id} className="flex justify-between">
                    <span>{i.nome}</span>
                    <span className="text-tinta-suave">{i.consumido_30d}{i.unidade_uso}</span>
                  </li>
                ))}
              </ul>
            )}
          </div>

          <div className="rounded-xl bg-superficie p-4 shadow-sm">
            <p className="mb-2 text-xs font-semibold uppercase tracking-wide text-tinta-suave">Giro de estoque (30 dias)</p>
            {relatorios.giro_estoque.length === 0 ? (
              <p className="text-sm text-tinta-suave">Nenhum produto com controle de estoque ativo.</p>
            ) : (
              <ul className="space-y-1 text-sm text-tinta">
                {relatorios.giro_estoque.map((g) => (
                  <li key={`${g.produto_id}-${g.variacao_nome}`} className="flex justify-between">
                    <span>{g.produto_nome}{g.variacao_nome && ` — ${g.variacao_nome}`}</span>
                    <span className="text-tinta-suave">{g.giro.toFixed(1)}x ({g.vendido_30d} vendido(s) / {g.estoque_atual} em estoque)</span>
                  </li>
                ))}
              </ul>
            )}
          </div>

          <div className="rounded-xl bg-superficie p-4 shadow-sm">
            <p className="mb-2 text-xs font-semibold uppercase tracking-wide text-tinta-suave">Produtos parados (30 dias)</p>
            {relatorios.produtos_parados.length === 0 ? (
              <p className="text-sm text-tinta-suave">Nenhum produto disponível ficou sem vender nos últimos 30 dias.</p>
            ) : (
              <ul className="space-y-1 text-sm text-tinta">
                {relatorios.produtos_parados.map((p) => (
                  <li key={p.produto_id}>{p.produto_nome}</li>
                ))}
              </ul>
            )}
          </div>
        </div>
      )}
    </div>
  );
}

function ModalEstoque({ item, onFechar, onAtualizado }: { item: ItemEstoque; onFechar: () => void; onAtualizado: () => void }) {
  const [quantidadeRepor, setQuantidadeRepor] = useState('');
  const [motivoRepor, setMotivoRepor] = useState('');
  const [novoValor, setNovoValor] = useState('');
  const [motivoAjuste, setMotivoAjuste] = useState('');
  const [enviando, setEnviando] = useState(false);
  const [erro, setErro] = useState<string | null>(null);

  const { data: movimentacoes } = useQuery({
    queryKey: ['estoque-movimentacoes', item.produto_id],
    queryFn: () => buscarMovimentacoesEstoque(item.produto_id),
  });

  async function repor(e: React.FormEvent) {
    e.preventDefault();
    const quantidade = parseInt(quantidadeRepor, 10);
    if (!quantidade || quantidade <= 0) {
      setErro('Informe uma quantidade maior que zero.');
      return;
    }
    setEnviando(true);
    setErro(null);
    try {
      await reporEstoque({
        produto_id: item.produto_id,
        variacao_id: item.variacao_id,
        quantidade,
        motivo: motivoRepor || undefined,
      });
      setQuantidadeRepor('');
      setMotivoRepor('');
      onAtualizado();
    } catch {
      setErro('Não foi possível repor o estoque. Tenta de novo.');
    } finally {
      setEnviando(false);
    }
  }

  async function ajustar(e: React.FormEvent) {
    e.preventDefault();
    const valor = parseInt(novoValor, 10);
    if (isNaN(valor) || valor < 0) {
      setErro('Informe um valor válido (0 ou mais).');
      return;
    }
    if (!motivoAjuste.trim()) {
      setErro('Motivo é obrigatório num ajuste manual.');
      return;
    }
    setEnviando(true);
    setErro(null);
    try {
      await ajustarEstoque({
        produto_id: item.produto_id,
        variacao_id: item.variacao_id,
        novo_valor: valor,
        motivo: motivoAjuste,
      });
      setNovoValor('');
      setMotivoAjuste('');
      onAtualizado();
    } catch {
      setErro('Não foi possível ajustar o estoque. Tenta de novo.');
    } finally {
      setEnviando(false);
    }
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 px-4" onClick={onFechar}>
      <div
        className="max-h-[85vh] w-full max-w-md overflow-y-auto rounded-2xl bg-superficie p-5 shadow-lg"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="mb-4 flex items-start justify-between">
          <div>
            <h2 className="font-display text-lg tracking-wide text-tinta">{tituloItem(item)}</h2>
            <p className="text-sm text-tinta-suave">Estoque atual: {item.estoque_atual}</p>
          </div>
          <button onClick={onFechar} className="text-tinta-suave hover:text-tinta" aria-label="Fechar">✕</button>
        </div>

        {erro && <p className="mb-3 text-sm text-acento">{erro}</p>}

        <form onSubmit={repor} className="mb-4 space-y-2 rounded-xl bg-fundo p-3">
          <p className="text-xs font-semibold uppercase tracking-wide text-tinta-suave">Repor estoque</p>
          <div className="flex gap-2">
            <input
              type="number"
              min={1}
              placeholder="+ quantidade"
              value={quantidadeRepor}
              onChange={(e) => setQuantidadeRepor(e.target.value)}
              className="w-28 rounded-lg border border-tinta/20 bg-superficie px-2 py-1.5 text-sm text-tinta outline-none focus:border-acento"
            />
            <input
              type="text"
              placeholder="Motivo (opcional)"
              value={motivoRepor}
              onChange={(e) => setMotivoRepor(e.target.value)}
              className="flex-1 rounded-lg border border-tinta/20 bg-superficie px-2 py-1.5 text-sm text-tinta outline-none focus:border-acento"
            />
          </div>
          <button
            type="submit"
            disabled={enviando}
            className="w-full rounded-full bg-acento py-2 text-sm font-semibold text-superficie disabled:opacity-60"
          >
            {enviando ? 'Salvando...' : 'Repor'}
          </button>
        </form>

        <form onSubmit={ajustar} className="mb-4 space-y-2 rounded-xl bg-fundo p-3">
          <p className="text-xs font-semibold uppercase tracking-wide text-tinta-suave">
            Ajustar pra um valor (contagem física)
          </p>
          <div className="flex gap-2">
            <input
              type="number"
              min={0}
              placeholder="novo valor"
              value={novoValor}
              onChange={(e) => setNovoValor(e.target.value)}
              className="w-28 rounded-lg border border-tinta/20 bg-superficie px-2 py-1.5 text-sm text-tinta outline-none focus:border-acento"
            />
            <input
              type="text"
              placeholder="Motivo (obrigatório)"
              value={motivoAjuste}
              onChange={(e) => setMotivoAjuste(e.target.value)}
              className="flex-1 rounded-lg border border-tinta/20 bg-superficie px-2 py-1.5 text-sm text-tinta outline-none focus:border-acento"
            />
          </div>
          <button
            type="submit"
            disabled={enviando}
            className="w-full rounded-full border border-tinta/20 py-2 text-sm font-semibold text-tinta disabled:opacity-60"
          >
            {enviando ? 'Salvando...' : 'Ajustar'}
          </button>
        </form>

        <div>
          <p className="mb-2 text-xs font-semibold uppercase tracking-wide text-tinta-suave">
            Histórico de movimentação
          </p>
          {!movimentacoes || movimentacoes.length === 0 ? (
            <p className="text-sm text-tinta-suave">Nenhuma movimentação registrada ainda.</p>
          ) : (
            <ul className="space-y-1.5">
              {movimentacoes.map((m) => (
                <li key={m.id} className="flex items-center justify-between text-xs text-tinta-suave">
                  <span>
                    {ROTULO_MOVIMENTO[m.tipo]}
                    {m.motivo ? ` — ${m.motivo}` : ''}
                    {m.pedido_id ? ` (pedido #${m.pedido_id})` : ''}
                  </span>
                  <span className="shrink-0">
                    {m.quantidade > 0 ? `+${m.quantidade}` : m.quantidade} · {formatarData(m.created_at)}
                  </span>
                </li>
              ))}
            </ul>
          )}
        </div>
      </div>
    </div>
  );
}
