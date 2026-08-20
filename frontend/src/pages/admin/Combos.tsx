import { useState, type ChangeEvent, type FormEvent } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import axios from 'axios';
import { listarCombos, criarCombo, atualizarCombo, deletarCombo, type ComboInput } from '../../api/combos';
import { listarProdutos, buscarLoja } from '../../api/admin';
import { enviarImagem, logoMiniatura } from '../../api/upload';
import type { Combo } from '../../api/types';
import { Campo } from '../../components/Campo';
import { rotuloCombo, rotuloCatalogo } from '../../lib/utils';
import { InputPrice } from '../../components/ui/input-price';

// Arredonda pra centavos — evita que o valor guardado saia com sobras de
// ponto flutuante (ex: 29.999999999997 em vez de 30) e garante que o que
// o lojista digita bate exatamente com o que é registrado e exibido.
function arredondarCentavos(valor: number): number {
  return Math.round(valor * 100) / 100;
}

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
  const { data: loja } = useQuery({ queryKey: ['loja'], queryFn: buscarLoja });

  // "Combo" vira "Kit" pra loja "mercadoria" — só o texto muda, o
  // domain/API/rotas continuam se chamando Combo (ver rotuloCombo).
  const rotulo = rotuloCombo(loja?.segmento_principal);
  const rotuloMin = rotulo.toLowerCase();

  const [editandoId, setEditandoId] = useState<number | null>(null);
  const [mostrarForm, setMostrarForm] = useState(false);
  const [form, setForm] = useState<ComboInput>(formVazio);
  const [erro, setErro] = useState<string | null>(null);
  const [enviandoFoto, setEnviandoFoto] = useState(false);

  const invalidar = () => queryClient.invalidateQueries({ queryKey: ['combos'] });

  function mensagemErro(err: unknown): string {
    return axios.isAxiosError(err) && err.response?.data?.erro
      ? err.response.data.erro
      : `Não foi possível salvar o ${rotuloMin}.`;
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

  async function selecionarFoto(e: ChangeEvent<HTMLInputElement>) {
    const arquivo = e.target.files?.[0];
    if (!arquivo) return;
    setEnviandoFoto(true);
    setErro(null);
    try {
      const url = await enviarImagem(arquivo);
      setForm((atual) => ({ ...atual, foto_url: url }));
    } catch {
      setErro('Não foi possível enviar a foto.');
    } finally {
      setEnviandoFoto(false);
    }
  }

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
    if (!form.nome.trim() || form.preco <= 0) { setErro(`Preenche o nome e o preço do ${rotuloMin}.`); return; }
    if (form.itens.length === 0) { setErro(`Adiciona pelo menos um produto ao ${rotuloMin}.`); return; }
    setErro(null);
    if (editandoId) mutAtualizar.mutate({ id: editandoId, input: form });
    else mutCriar.mutate(form);
  }

  const salvando = mutCriar.isPending || mutAtualizar.isPending;

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="font-display text-2xl tracking-wide text-tinta">{rotulo}s</h1>
        {!mostrarForm && (
          <button onClick={abrirNovo} className="btn-neu-primario">
            + Novo {rotuloMin}
          </button>
        )}
      </div>

      <p className="text-sm text-tinta-suave">
        Monte um pacote fixo de produtos com um preço final único (não é calculado por desconto sobre a soma dos itens — você define o valor direto). O cliente escolhe a variação de cada produto ao adicionar o {rotuloMin} no carrinho, igual comprando avulso.
      </p>

      {mostrarForm && (
        <form onSubmit={salvar} className="space-y-4 rounded-2xl bg-superficie p-5 shadow-sm">
          <h2 className="font-display text-lg tracking-wide text-tinta">
            {editandoId ? `Editar ${rotuloMin}` : `Novo ${rotuloMin}`}
          </h2>

          <Campo label={`Nome do ${rotuloMin}`}>
            <input
              required
              value={form.nome}
              onChange={(e) => setForm({ ...form, nome: e.target.value })}
              placeholder={rotulo === 'Kit' ? 'Kit Presente' : 'Combo X-Tudo'}
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

          <Campo label={`Foto do ${rotuloMin} (opcional)`}>
            <div className="flex items-center gap-3">
              {form.foto_url && (
                <img
                  src={logoMiniatura(form.foto_url)}
                  alt={`Prévia do ${rotuloMin}`}
                  className="h-12 w-12 shrink-0 rounded-lg object-cover"
                />
              )}
              <label className="cursor-pointer btn-neu-secundario hover:border-acento">
                {enviandoFoto ? 'Enviando...' : form.foto_url ? 'Trocar imagem' : 'Enviar imagem'}
                <input type="file" accept="image/*" onChange={selecionarFoto} disabled={enviandoFoto} className="hidden" />
              </label>
            </div>
          </Campo>

          <Campo label={`Preço final do ${rotuloMin}`}>
            <InputPrice min="0.01" required value={form.preco || ''} onChange={(e) => setForm({ ...form, preco: arredondarCentavos(parseFloat(e.target.value) || 0) })} />
          </Campo>

          <div className="space-y-2">
            <div className="flex items-center justify-between">
              <span className="text-xs font-medium uppercase tracking-wide text-tinta-suave">Produtos do {rotuloMin}</span>
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
            {rotulo} disponível no {rotuloCatalogo(loja?.segmento_principal)}
          </label>

          {erro && <p className="text-sm text-acento">{erro}</p>}

          <div className="flex gap-3">
            <button type="button" onClick={fecharForm} className="btn-neu-secundario">Cancelar</button>
            <button type="submit" disabled={salvando} className="btn-neu-primario">
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
        <p className="text-tinta-suave">Nenhum {rotuloMin} cadastrado ainda.</p>
      )}
    </div>
  );
}
