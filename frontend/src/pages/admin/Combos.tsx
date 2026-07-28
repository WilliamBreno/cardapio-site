import { useState, type FormEvent } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import axios from 'axios';
import { listarCombos, criarCombo, atualizarCombo, deletarCombo, type ComboInput } from '../../api/combos';
import { listarProdutos } from '../../api/admin';
import type { Combo } from '../../api/types';
import { Campo } from '../../components/Campo';

const formVazio: ComboInput = {
  nome: '',
  descricao: '',
  foto_url: '',
  preco: 0,
  disponivel: true,
  ordem: 0,
  itens: [],
};

export function Combos() {
  const queryClient = useQueryClient();
  const { data: combos, isLoading } = useQuery({ queryKey: ['combos'], queryFn: listarCombos });
  const { data: produtos } = useQuery({ queryKey: ['produtos'], queryFn: listarProdutos });

  const [editandoId, setEditandoId] = useState<number | null>(null);
  const [mostrarForm, setMostrarForm] = useState(false);
  const [form, setForm] = useState<ComboInput>(formVazio);
  const [erro, setErro] = useState<string | null>(null);

  const invalidar = () => queryClient.invalidateQueries({ queryKey: ['combos'] });

  function mensagemErro(err: unknown): string {
    return axios.isAxiosError(err) && err.response?.data?.erro
      ? err.response.data.erro
      : 'Não foi possível salvar o combo.';
  }

  const mutCriar = useMutation({
    mutationFn: criarCombo,
    onSuccess: () => { invalidar(); fecharForm(); },
    onError: (e: unknown) => setErro(mensagemErro(e)),
  });
  const mutAtualizar = useMutation({
    mutationFn: ({ id, input }: { id: number; input: ComboInput }) => atualizarCombo(id, input),
    onSuccess: () => { invalidar(); fecharForm(); },
    onError: (e: unknown) => setErro(mensagemErro(e)),
  });
  const mutDeletar = useMutation({ mutationFn: deletarCombo, onSuccess: invalidar });

  function abrirNovo() {
    setEditandoId(null);
    setForm(formVazio);
    setErro(null);
    setMostrarForm(true);
  }

  function abrirEdicao(combo: Combo) {
    setEditandoId(combo.id);
    setForm({
      nome: combo.nome,
      descricao: combo.descricao,
      foto_url: combo.foto_url,
      preco: combo.preco,
      disponivel: combo.disponivel,
      ordem: combo.ordem,
      itens: combo.itens.map((item) => ({ produto_id: item.produto_id, quantidade: item.quantidade })),
    });
    setErro(null);
    setMostrarForm(true);
  }

  function fecharForm() { setMostrarForm(false); setEditandoId(null); }

  function adicionarItem() {
    if (!produtos || produtos.length === 0) return;
    setForm((atual) => ({
      ...atual,
      itens: [...atual.itens, { produto_id: produtos[0].id, quantidade: 1 }],
    }));
  }

  function atualizarItem(indice: number, produtoId: number, quantidade: number) {
    setForm((atual) => ({
      ...atual,
      itens: atual.itens.map((item, i) => (i === indice ? { produto_id: produtoId, quantidade } : item)),
    }));
  }

  function removerItem(indice: number) {
    setForm((atual) => ({ ...atual, itens: atual.itens.filter((_, i) => i !== indice) }));
  }

  function salvar(e: FormEvent) {
    e.preventDefault();
    if (!form.nome.trim() || form.preco <= 0) { setErro('Preenche o nome e o preço do combo.'); return; }
    if (form.itens.length === 0) { setErro('Adiciona pelo menos um produto ao combo.'); return; }
    setErro(null);
    if (editandoId) mutAtualizar.mutate({ id: editandoId, input: form });
    else mutCriar.mutate(form);
  }

  const salvando = mutCriar.isPending || mutAtualizar.isPending;

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="font-display text-2xl tracking-wide text-tinta">Combos e Kits</h1>
        {!mostrarForm && (
          <button onClick={abrirNovo} className="rounded-full bg-acento px-4 py-2 text-sm font-semibold text-superficie">
            + Novo combo
          </button>
        )}
      </div>

      <p className="text-sm text-tinta-suave">
        Monte um pacote fixo de produtos com um preço final único (não é calculado por desconto sobre a soma dos itens — você define o valor direto). O cliente escolhe a variação de cada produto ao adicionar o combo no carrinho, igual comprando avulso.
      </p>

      {mostrarForm && (
        <form onSubmit={salvar} className="space-y-4 rounded-2xl bg-superficie p-5 shadow-sm">
          <h2 className="font-display text-lg tracking-wide text-tinta">
            {editandoId ? 'Editar combo' : 'Novo combo'}
          </h2>

          <Campo label="Nome do combo">
            <input
              required
              value={form.nome}
              onChange={(e) => setForm({ ...form, nome: e.target.value })}
              placeholder="Combo X-Tudo"
              className="w-full rounded-lg border border-tinta/20 bg-fundo px-3 py-2 text-tinta outline-none focus:border-acento"
            />
          </Campo>

          <Campo label="Descrição (opcional)">
            <textarea
              value={form.descricao}
              onChange={(e) => setForm({ ...form, descricao: e.target.value })}
              rows={2}
              className="w-full rounded-lg border border-tinta/20 bg-fundo px-3 py-2 text-tinta outline-none focus:border-acento"
            />
          </Campo>

          <Campo label="URL da foto (opcional)">
            <input
              value={form.foto_url}
              onChange={(e) => setForm({ ...form, foto_url: e.target.value })}
              placeholder="https://..."
              className="w-full rounded-lg border border-tinta/20 bg-fundo px-3 py-2 text-tinta outline-none focus:border-acento"
            />
          </Campo>

          <Campo label="Preço final do combo (R$)">
            <input
              type="number"
              step="0.50"
              min="0.01"
              required
              value={form.preco || ''}
              onChange={(e) => setForm({ ...form, preco: parseFloat(e.target.value) || 0 })}
              className="w-full rounded-lg border border-tinta/20 bg-fundo px-3 py-2 text-tinta outline-none focus:border-acento"
            />
          </Campo>

          <div className="space-y-2">
            <div className="flex items-center justify-between">
              <span className="text-xs font-medium uppercase tracking-wide text-tinta-suave">Produtos do combo</span>
              <button type="button" onClick={adicionarItem} className="text-sm font-medium text-acento hover:underline">
                + Adicionar produto
              </button>
            </div>
            {form.itens.length === 0 && (
              <p className="text-sm text-tinta-suave">Nenhum produto adicionado ainda.</p>
            )}
            {form.itens.map((item, indice) => (
              <div key={indice} className="flex items-center gap-2">
                <select
                  value={item.produto_id}
                  onChange={(e) => atualizarItem(indice, parseInt(e.target.value), item.quantidade)}
                  className="min-w-0 flex-1 rounded-lg border border-tinta/20 bg-fundo px-3 py-2 text-sm text-tinta outline-none focus:border-acento"
                >
                  {(produtos ?? []).map((p) => (
                    <option key={p.id} value={p.id}>{p.nome}</option>
                  ))}
                </select>
                <input
                  type="number"
                  min="1"
                  value={item.quantidade}
                  onChange={(e) => atualizarItem(indice, item.produto_id, parseInt(e.target.value) || 1)}
                  className="w-16 rounded-lg border border-tinta/20 bg-fundo px-2 py-2 text-center text-sm text-tinta outline-none focus:border-acento"
                />
                <button type="button" onClick={() => removerItem(indice)} className="text-sm text-acento/70 hover:text-acento">
                  Remover
                </button>
              </div>
            ))}
          </div>

          <label className="flex items-center gap-2 text-sm text-tinta">
            <input
              type="checkbox"
              checked={form.disponivel}
              onChange={(e) => setForm({ ...form, disponivel: e.target.checked })}
              className="h-4 w-4 accent-acento"
            />
            Combo disponível no cardápio
          </label>

          {erro && <p className="text-sm text-acento">{erro}</p>}

          <div className="flex gap-3">
            <button type="button" onClick={fecharForm} className="rounded-full border border-tinta/20 px-4 py-2 text-sm font-semibold text-tinta">Cancelar</button>
            <button type="submit" disabled={salvando} className="rounded-full bg-acento px-4 py-2 text-sm font-semibold text-superficie disabled:opacity-60">
              {salvando ? 'Salvando...' : 'Salvar'}
            </button>
          </div>
        </form>
      )}

      {isLoading ? (
        <p className="text-tinta-suave">Carregando...</p>
      ) : combos && combos.length > 0 ? (
        <ul className="space-y-3">
          {combos.map((combo) => (
            <li key={combo.id} className="rounded-2xl bg-superficie p-4 shadow-sm">
              <div className="flex items-start justify-between gap-3">
                <div>
                  <div className="flex items-center gap-2">
                    <span className="font-display text-base text-tinta">{combo.nome}</span>
                    <span className={`rounded-full px-2 py-0.5 text-xs font-semibold ${combo.disponivel ? 'bg-emerald-100 text-emerald-700' : 'bg-tinta/10 text-tinta-suave'}`}>
                      {combo.disponivel ? 'Disponível' : 'Indisponível'}
                    </span>
                  </div>
                  <p className="mt-1 font-carimbo text-sm text-tinta-suave">
                    R$ {combo.preco.toFixed(2).replace('.', ',')}
                  </p>
                  <p className="text-xs text-tinta-suave">
                    {combo.itens.map((item) => `${item.quantidade}x ${item.produto?.nome ?? 'produto removido'}`).join(' + ')}
                  </p>
                </div>
                <div className="flex shrink-0 gap-2">
                  <button onClick={() => abrirEdicao(combo)} className="text-sm font-medium text-acento hover:underline">Editar</button>
                  <button onClick={() => { if (confirm(`Excluir "${combo.nome}"?`)) mutDeletar.mutate(combo.id); }} className="text-sm text-tinta-suave hover:text-acento">Excluir</button>
                </div>
              </div>
            </li>
          ))}
        </ul>
      ) : (
        <p className="text-tinta-suave">Nenhum combo cadastrado ainda.</p>
      )}
    </div>
  );
}
