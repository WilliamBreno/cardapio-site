import { useState, type FormEvent } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import axios from 'axios';
import { listarProdutos } from '../../api/admin';
import {
  listarSugestoesProduto, criarSugestaoProduto, deletarSugestaoProduto,
  buscarConfiguracaoPlataforma, type SugestaoProdutoInput,
} from '../../api/sugestoes';
import type { TipoDesconto } from '../../api/types';
import { Campo } from '../../components/Campo';

// N máximo de produtos sugeridos por origem — V1 simples, um número fixo
// em vez de configurável (mesma filosofia de "propositalmente simples" da
// Parte 1/Combos).
const MAX_SUGESTOES_POR_PRODUTO = 5;

export function SugestaoInteligente() {
  const queryClient = useQueryClient();
  const { data: produtos } = useQuery({ queryKey: ['produtos'], queryFn: listarProdutos });
  const { data: sugestoes } = useQuery({ queryKey: ['sugestoes-produto'], queryFn: listarSugestoesProduto });
  const { data: configuracaoPlataforma } = useQuery({ queryKey: ['configuracao-plataforma'], queryFn: buscarConfiguracaoPlataforma });

  const [produtoOrigemId, setProdutoOrigemId] = useState<number | null>(null);
  const [produtoSugeridoId, setProdutoSugeridoId] = useState<number | ''>('');
  const [tipoDesconto, setTipoDesconto] = useState<TipoDesconto | ''>('');
  const [valorDesconto, setValorDesconto] = useState(0);
  const [erro, setErro] = useState<string | null>(null);

  const invalidar = () => queryClient.invalidateQueries({ queryKey: ['sugestoes-produto'] });

  const mutCriar = useMutation({
    mutationFn: criarSugestaoProduto,
    onSuccess: () => { invalidar(); setProdutoSugeridoId(''); setTipoDesconto(''); setValorDesconto(0); setErro(null); },
    onError: (e: unknown) => setErro(
      axios.isAxiosError(e) && e.response?.data?.erro ? e.response.data.erro : 'Não foi possível criar a sugestão.'
    ),
  });
  const mutDeletar = useMutation({ mutationFn: deletarSugestaoProduto, onSuccess: invalidar });

  const produtoOrigem = produtos?.find((p) => p.id === produtoOrigemId) ?? null;
  const sugestoesDoOrigem = sugestoes?.filter((s) => s.produto_origem_id === produtoOrigemId) ?? [];

  // Bloqueia na UI: o próprio produto e qualquer produto de categoria já
  // usada por um vínculo existente dessa origem (o backend rejeita do
  // mesmo jeito, isso só evita o vai-e-volta de erro).
  const categoriasJaUsadas = new Set(sugestoesDoOrigem.map((s) => s.produto_sugerido?.categoria_id).filter(Boolean));
  const opcoesSugerido = (produtos ?? []).filter(
    (p) => p.id !== produtoOrigemId && !categoriasJaUsadas.has(p.categoria_id)
  );

  function salvar(e: FormEvent) {
    e.preventDefault();
    if (!produtoOrigemId || !produtoSugeridoId) return;
    const input: SugestaoProdutoInput = {
      produto_origem_id: produtoOrigemId,
      produto_sugerido_id: produtoSugeridoId,
      tipo_desconto: tipoDesconto || undefined,
      valor_desconto: tipoDesconto ? valorDesconto : undefined,
    };
    mutCriar.mutate(input);
  }

  return (
    <div className="space-y-6">
      <h1 className="font-display text-2xl tracking-wide text-tinta">Sugestão Inteligente</h1>

      <p className="text-sm text-tinta-suave">
        Configure produtos que aparecem como sugestão quando o cliente já tem outro produto no carrinho (estilo totem de fastfood) — não é automático, você escolhe cada vínculo. Pra ativar as sugestões no carrinho do cliente, vá em Configurações.
        {configuracaoPlataforma && (
          <> Recurso pago à parte, R$ {configuracaoPlataforma.sugestao_inteligente_preco_mensal.toFixed(2).replace('.', ',')}/mês.</>
        )}
      </p>

      <div className="grid gap-4 sm:grid-cols-[1fr_1.5fr]">
        <div className="space-y-1 rounded-2xl bg-superficie p-3 shadow-sm">
          <p className="mb-1 px-2 text-xs font-medium uppercase tracking-wide text-tinta-suave">Produtos</p>
          {(produtos ?? []).map((p) => {
            const total = sugestoes?.filter((s) => s.produto_origem_id === p.id).length ?? 0;
            return (
              <button
                key={p.id}
                onClick={() => setProdutoOrigemId(p.id)}
                className={`flex w-full items-center justify-between rounded-lg px-3 py-2 text-left text-sm transition ${
                  produtoOrigemId === p.id ? 'bg-acento/10 text-acento' : 'text-tinta hover:bg-fundo'
                }`}
              >
                <span className="truncate">{p.nome}</span>
                {total > 0 && <span className="ml-2 shrink-0 text-xs text-tinta-suave">{total}</span>}
              </button>
            );
          })}
          {produtos && produtos.length === 0 && (
            <p className="px-2 py-1 text-sm text-tinta-suave">Cadastre produtos primeiro.</p>
          )}
        </div>

        <div className="space-y-4">
          {!produtoOrigem ? (
            <p className="rounded-2xl bg-superficie p-4 text-sm text-tinta-suave shadow-sm">
              Escolhe um produto na lista pra configurar as sugestões dele.
            </p>
          ) : (
            <>
              <div className="rounded-2xl bg-superficie p-4 shadow-sm">
                <h2 className="font-display text-lg tracking-wide text-tinta">Sugestões pra "{produtoOrigem.nome}"</h2>

                {sugestoesDoOrigem.length === 0 ? (
                  <p className="mt-2 text-sm text-tinta-suave">Nenhuma sugestão configurada ainda.</p>
                ) : (
                  <ul className="mt-3 space-y-2">
                    {sugestoesDoOrigem.map((s) => (
                      <li key={s.id} className="flex items-center justify-between rounded-lg bg-fundo px-3 py-2">
                        <div>
                          <p className="text-sm font-medium text-tinta">{s.produto_sugerido?.nome}</p>
                          {s.tipo_desconto && s.valor_desconto && (
                            <p className="text-xs text-emerald-600">
                              {s.tipo_desconto === 'percentual' ? `${s.valor_desconto}% de desconto` : `R$ ${s.valor_desconto.toFixed(2).replace('.', ',')} de desconto`} ao adicionar via sugestão
                            </p>
                          )}
                        </div>
                        <button onClick={() => mutDeletar.mutate(s.id)} className="text-sm text-acento/70 hover:text-acento">
                          Remover
                        </button>
                      </li>
                    ))}
                  </ul>
                )}
              </div>

              {sugestoesDoOrigem.length >= MAX_SUGESTOES_POR_PRODUTO ? (
                <p className="rounded-2xl bg-superficie p-4 text-sm text-tinta-suave shadow-sm">
                  Limite de {MAX_SUGESTOES_POR_PRODUTO} sugestões por produto atingido.
                </p>
              ) : (
                <form onSubmit={salvar} className="space-y-3 rounded-2xl bg-superficie p-4 shadow-sm">
                  <h3 className="font-display text-base tracking-wide text-tinta">Nova sugestão</h3>

                  <Campo label="Produto sugerido">
                    <select
                      required
                      value={produtoSugeridoId}
                      onChange={(e) => setProdutoSugeridoId(parseInt(e.target.value) || '')}
                      className="w-full rounded-lg border border-tinta/20 bg-fundo px-3 py-2 text-sm text-tinta outline-none focus:border-acento"
                    >
                      <option value="">Selecione...</option>
                      {opcoesSugerido.map((p) => (
                        <option key={p.id} value={p.id}>{p.nome}</option>
                      ))}
                    </select>
                    <span className="mt-1 block text-xs text-tinta-suave">
                      O próprio produto e produtos de categorias já usadas em outra sugestão dessa origem não aparecem aqui — só uma sugestão por categoria.
                    </span>
                  </Campo>

                  <div className="flex gap-3">
                    <Campo label="Desconto (opcional)" className="flex-1">
                      <select
                        value={tipoDesconto}
                        onChange={(e) => setTipoDesconto(e.target.value as TipoDesconto | '')}
                        className="w-full rounded-lg border border-tinta/20 bg-fundo px-3 py-2 text-sm text-tinta outline-none focus:border-acento"
                      >
                        <option value="">Sem desconto</option>
                        <option value="percentual">Percentual (%)</option>
                        <option value="fixo">Valor fixo (R$)</option>
                      </select>
                    </Campo>
                    {tipoDesconto && (
                      <Campo label={tipoDesconto === 'percentual' ? 'Desconto (%)' : 'Desconto (R$)'} className="flex-1">
                        <input
                          type="number"
                          step="0.01"
                          min="0.01"
                          max={tipoDesconto === 'percentual' ? 100 : undefined}
                          value={valorDesconto || ''}
                          onChange={(e) => setValorDesconto(parseFloat(e.target.value) || 0)}
                          className="w-full rounded-lg border border-tinta/20 bg-fundo px-3 py-2 text-sm text-tinta outline-none focus:border-acento"
                        />
                      </Campo>
                    )}
                  </div>
                  <p className="text-xs text-tinta-suave">
                    O desconto só se aplica quando o cliente adiciona esse produto através da sugestão — o preço avulso normal não muda.
                  </p>

                  {erro && <p className="text-sm text-acento">{erro}</p>}

                  <button
                    type="submit"
                    disabled={mutCriar.isPending || !produtoSugeridoId}
                    className="rounded-full bg-acento px-4 py-2 text-sm font-semibold text-superficie disabled:opacity-60"
                  >
                    {mutCriar.isPending ? 'Salvando...' : 'Adicionar sugestão'}
                  </button>
                </form>
              )}
            </>
          )}
        </div>
      </div>
    </div>
  );
}
