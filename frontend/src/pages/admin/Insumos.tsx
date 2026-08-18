import { useState, type FormEvent } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  buscarLoja, listarInsumos, criarInsumo, atualizarInsumo, deletarInsumo, type Insumo, type InsumoInput,
} from '../../api/admin';
import { Campo } from '../../components/Campo';
import { ImportarNFeModal } from '../../components/admin/ImportarNFeModal';

const formVazio: InsumoInput = {
  nome: '',
  unidade_compra: 'kg',
  unidade_uso: 'g',
  fator_conversao: 1000,
  custo_unidade_compra: 0,
  estoque_atual: null,
  estoque_alerta: null,
};

function formatarCusto(v: number) {
  return `R$ ${v.toFixed(2).replace('.', ',')}`;
}

export function Insumos() {
  const queryClient = useQueryClient();
  const { data: loja } = useQuery({ queryKey: ['loja'], queryFn: buscarLoja });
  const { data: insumos, isLoading } = useQuery({ queryKey: ['insumos'], queryFn: listarInsumos, enabled: loja?.plano === 'scale' });

  const [editandoId, setEditandoId] = useState<number | null>(null);
  const [mostrarForm, setMostrarForm] = useState(false);
  const [form, setForm] = useState<InsumoInput>(formVazio);
  const [comEstoque, setComEstoque] = useState(false);
  const [erro, setErro] = useState<string | null>(null);
  const [mostrarImportarNFe, setMostrarImportarNFe] = useState(false);

  const invalidar = () => queryClient.invalidateQueries({ queryKey: ['insumos'] });

  const mutCriar = useMutation({
    mutationFn: criarInsumo,
    onSuccess: () => { invalidar(); fecharForm(); },
    onError: () => setErro('Não foi possível salvar o insumo.'),
  });
  const mutAtualizar = useMutation({
    mutationFn: ({ id, input }: { id: number; input: InsumoInput }) => atualizarInsumo(id, input),
    onSuccess: () => { invalidar(); fecharForm(); },
    onError: () => setErro('Não foi possível salvar o insumo.'),
  });
  const mutDeletar = useMutation({
    mutationFn: deletarInsumo,
    onSuccess: invalidar,
    onError: () => alert('Não foi possível excluir — esse insumo está sendo usado em alguma ficha técnica.'),
  });

  function abrirNovo() {
    setEditandoId(null);
    setForm(formVazio);
    setComEstoque(false);
    setErro(null);
    setMostrarForm(true);
  }

  function abrirEdicao(i: Insumo) {
    setEditandoId(i.id);
    setForm({
      nome: i.nome,
      unidade_compra: i.unidade_compra,
      unidade_uso: i.unidade_uso,
      fator_conversao: i.fator_conversao,
      custo_unidade_compra: i.custo_unidade_compra,
      estoque_atual: i.estoque_atual,
      estoque_alerta: i.estoque_alerta,
    });
    setComEstoque(i.estoque_atual !== null);
    setErro(null);
    setMostrarForm(true);
  }

  function fecharForm() { setMostrarForm(false); setEditandoId(null); }

  function salvar(e: FormEvent) {
    e.preventDefault();
    if (!form.nome || !form.unidade_compra || !form.unidade_uso || form.fator_conversao <= 0) {
      setErro('Preenche nome, unidades e um fator de conversão maior que zero.');
      return;
    }
    setErro(null);
    const payload: InsumoInput = {
      ...form,
      estoque_atual: comEstoque ? (form.estoque_atual ?? 0) : null,
      estoque_alerta: comEstoque ? form.estoque_alerta : null,
    };
    if (editandoId) mutAtualizar.mutate({ id: editandoId, input: payload });
    else mutCriar.mutate(payload);
  }

  const salvando = mutCriar.isPending || mutAtualizar.isPending;

  if (loja && loja.plano !== 'scale') {
    return (
      <div className="space-y-3">
        <h1 className="font-display text-2xl tracking-wide text-tinta">Insumos</h1>
        <div className="rounded-2xl bg-superficie p-6 text-center shadow-sm">
          <p className="text-sm text-tinta">
            Ficha técnica e CMV automático (custo do produto calculado a partir dos insumos) são um
            recurso exclusivo do plano Scale.
          </p>
          <p className="mt-2 text-xs text-tinta-suave">Ative o Scale em Meu Plano pra liberar.</p>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="font-display text-2xl tracking-wide text-tinta">Insumos</h1>
        {!mostrarForm && (
          <div className="flex gap-2">
            <button
              onClick={() => setMostrarImportarNFe(true)}
              className="rounded-full border border-tinta/20 px-4 py-2 text-sm font-semibold text-tinta"
            >
              Importar NF-e
            </button>
            <button onClick={abrirNovo} className="rounded-full bg-acento px-4 py-2 text-sm font-semibold text-superficie">
              + Novo insumo
            </button>
          </div>
        )}
      </div>

      <p className="text-sm text-tinta-suave">
        Insumos são os ingredientes/matérias-primas usados na ficha técnica dos seus produtos (ex:
        carne, pão, queijo) — não aparecem no cardápio, só são consumidos quando um produto que os
        usa é vendido. Monte a ficha técnica de um produto na tela de Produtos.
      </p>

      {mostrarForm && (
        <form onSubmit={salvar} className="space-y-4 rounded-2xl bg-superficie p-5 shadow-sm">
          <h2 className="font-display text-lg tracking-wide text-tinta">
            {editandoId ? 'Editar insumo' : 'Novo insumo'}
          </h2>

          <Campo label="Nome">
            <input
              required
              value={form.nome}
              onChange={(e) => setForm({ ...form, nome: e.target.value })}
              placeholder="Ex: Carne bovina moída"
              className="w-full rounded-lg border border-tinta/20 bg-fundo px-3 py-2 text-tinta outline-none focus:border-acento"
            />
          </Campo>

          <div className="grid grid-cols-2 gap-3">
            <Campo label="Unidade de compra">
              <input
                required
                value={form.unidade_compra}
                onChange={(e) => setForm({ ...form, unidade_compra: e.target.value })}
                placeholder="kg, fardo, caixa, litro..."
                className="w-full rounded-lg border border-tinta/20 bg-fundo px-3 py-2 text-tinta outline-none focus:border-acento"
              />
            </Campo>
            <Campo label="Unidade de uso (na receita)">
              <input
                required
                value={form.unidade_uso}
                onChange={(e) => setForm({ ...form, unidade_uso: e.target.value })}
                placeholder="g, ml, unidade..."
                className="w-full rounded-lg border border-tinta/20 bg-fundo px-3 py-2 text-tinta outline-none focus:border-acento"
              />
            </Campo>
          </div>

          <div className="grid grid-cols-2 gap-3">
            <Campo label="1 unidade de compra equivale a quantas de uso?">
              <input
                type="number"
                step="0.01"
                min="0.01"
                required
                value={form.fator_conversao || ''}
                onChange={(e) => setForm({ ...form, fator_conversao: parseFloat(e.target.value) || 0 })}
                placeholder="Ex: 1kg = 1000g → 1000"
                className="w-full rounded-lg border border-tinta/20 bg-fundo px-3 py-2 text-tinta outline-none focus:border-acento"
              />
            </Campo>
            <Campo label="Custo por unidade de compra (R$)">
              <input
                type="number"
                step="0.01"
                min="0"
                value={form.custo_unidade_compra || ''}
                onChange={(e) => setForm({ ...form, custo_unidade_compra: parseFloat(e.target.value) || 0 })}
                className="w-full rounded-lg border border-tinta/20 bg-fundo px-3 py-2 text-tinta outline-none focus:border-acento"
              />
            </Campo>
          </div>

          {form.fator_conversao > 0 && (
            <p className="text-xs text-tinta-suave">
              Custo por {form.unidade_uso || 'unidade de uso'}: {formatarCusto(form.custo_unidade_compra / form.fator_conversao)}
            </p>
          )}

          <label className="flex items-center gap-2 text-sm text-tinta">
            <input
              type="checkbox"
              checked={comEstoque}
              onChange={(e) => setComEstoque(e.target.checked)}
              className="h-4 w-4 accent-acento"
            />
            Controlar estoque desse insumo
          </label>

          {comEstoque && (
            <div className="grid grid-cols-2 gap-3">
              <Campo label={`Estoque atual (${form.unidade_uso || 'unidade de uso'})`}>
                <input
                  type="number"
                  step="0.01"
                  min="0"
                  value={form.estoque_atual ?? ''}
                  onChange={(e) => setForm({ ...form, estoque_atual: e.target.value ? parseFloat(e.target.value) : 0 })}
                  className="w-full rounded-lg border border-tinta/20 bg-fundo px-3 py-2 text-tinta outline-none focus:border-acento"
                />
              </Campo>
              <Campo label="Avisar quando chegar em (opcional)">
                <input
                  type="number"
                  step="0.01"
                  min="0"
                  value={form.estoque_alerta ?? ''}
                  onChange={(e) => setForm({ ...form, estoque_alerta: e.target.value ? parseFloat(e.target.value) : null })}
                  placeholder="Sem alerta"
                  className="w-full rounded-lg border border-tinta/20 bg-fundo px-3 py-2 text-tinta outline-none focus:border-acento"
                />
              </Campo>
            </div>
          )}

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
      ) : insumos && insumos.length > 0 ? (
        <ul className="space-y-3">
          {insumos.map((insumo) => (
            <li key={insumo.id} className="rounded-2xl bg-superficie p-4 shadow-sm">
              <div className="flex items-start justify-between gap-3">
                <div>
                  <p className="text-sm font-medium text-tinta">{insumo.nome}</p>
                  <p className="mt-1 text-xs text-tinta-suave">
                    {formatarCusto(insumo.custo_unidade_compra)}/{insumo.unidade_compra} · {formatarCusto(insumo.custo_unidade_compra / insumo.fator_conversao)}/{insumo.unidade_uso}
                  </p>
                  {insumo.estoque_atual !== null && (
                    <p className={`text-xs ${insumo.estoque_atual === 0 ? 'text-acento' : 'text-tinta-suave'}`}>
                      {insumo.estoque_atual === 0 ? 'Esgotado' : `${insumo.estoque_atual} ${insumo.unidade_uso} em estoque`}
                    </p>
                  )}
                </div>
                <div className="flex shrink-0 gap-2">
                  <button onClick={() => abrirEdicao(insumo)} className="text-sm font-medium text-acento hover:underline">Editar</button>
                  <button onClick={() => { if (confirm(`Excluir "${insumo.nome}"?`)) mutDeletar.mutate(insumo.id); }} className="text-sm text-tinta-suave hover:text-acento">Excluir</button>
                </div>
              </div>
            </li>
          ))}
        </ul>
      ) : (
        <p className="text-tinta-suave">Nenhum insumo cadastrado ainda.</p>
      )}

      {mostrarImportarNFe && <ImportarNFeModal onFechar={() => setMostrarImportarNFe(false)} />}
    </div>
  );
}
