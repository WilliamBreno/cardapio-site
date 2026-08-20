import { useState, type FormEvent } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  listarCupons, criarCupom, atualizarCupom, deletarCupom, type CupomInput,
} from '../../api/admin';
import type { Cupom } from '../../api/types';
import { Campo } from '../../components/Campo';
import { InputPrice } from '../../components/ui/input-price';
import { InputDate } from '../../components/ui/input-date';

const formVazio: CupomInput = {
  codigo: '',
  tipo: 'percentual',
  valor: 0,
  ativo: true,
  uso_maximo: null,
  validade: null,
  valor_minimo_pedido: 0,
};

function formatarCodigo(v: string) {
  return v.toUpperCase().replace(/[^A-Z0-9]/g, '');
}

export function Cupons() {
  const queryClient = useQueryClient();
  const { data: cupons, isLoading } = useQuery({ queryKey: ['cupons'], queryFn: listarCupons });

  const [editandoId, setEditandoId] = useState<number | null>(null);
  const [mostrarForm, setMostrarForm] = useState(false);
  const [form, setForm] = useState<CupomInput>(formVazio);
  const [erro, setErro] = useState<string | null>(null);

  const invalidar = () => queryClient.invalidateQueries({ queryKey: ['cupons'] });

  const mutCriar = useMutation({
    mutationFn: criarCupom,
    onSuccess: () => { invalidar(); fecharForm(); },
    onError: () => setErro('Não foi possível salvar — o código pode já estar em uso.'),
  });
  const mutAtualizar = useMutation({
    mutationFn: ({ id, input }: { id: number; input: CupomInput }) => atualizarCupom(id, input),
    onSuccess: () => { invalidar(); fecharForm(); },
    onError: () => setErro('Não foi possível salvar.'),
  });
  const mutDeletar = useMutation({ mutationFn: deletarCupom, onSuccess: invalidar });

  function abrirNovo() {
    setEditandoId(null);
    setForm(formVazio);
    setErro(null);
    setMostrarForm(true);
  }

  function abrirEdicao(c: Cupom) {
    setEditandoId(c.id);
    setForm({
      codigo: c.codigo,
      tipo: c.tipo,
      valor: c.valor,
      ativo: c.ativo,
      uso_maximo: c.uso_maximo,
      validade: c.validade ? c.validade.substring(0, 10) : null,
      valor_minimo_pedido: c.valor_minimo_pedido,
    });
    setErro(null);
    setMostrarForm(true);
  }

  function fecharForm() { setMostrarForm(false); setEditandoId(null); }

  function salvar(e: FormEvent) {
    e.preventDefault();
    if (!form.codigo || form.valor <= 0) { setErro('Preenche código e valor.'); return; }
    setErro(null);
    const payload = {
      ...form,
      validade: form.validade ? new Date(form.validade + 'T23:59:59').toISOString() : null,
    };
    if (editandoId) mutAtualizar.mutate({ id: editandoId, input: payload });
    else mutCriar.mutate(payload);
  }

  const salvando = mutCriar.isPending || mutAtualizar.isPending;

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="font-display text-2xl tracking-wide text-tinta">Cupons</h1>
        {!mostrarForm && (
          <button onClick={abrirNovo} className="btn-neu-primario">
            + Novo cupom
          </button>
        )}
      </div>

      <p className="text-sm text-tinta-suave">
        Crie códigos promocionais pra seus clientes. O desconto é aplicado sobre o subtotal dos produtos.
      </p>

      {mostrarForm && (
        <form onSubmit={salvar} className="space-y-4 rounded-2xl bg-superficie p-5 shadow-sm">
          <h2 className="font-display text-lg tracking-wide text-tinta">
            {editandoId ? 'Editar cupom' : 'Novo cupom'}
          </h2>

          <Campo label="Código do cupom">
            <input
              required
              value={form.codigo}
              onChange={(e) => setForm({ ...form, codigo: formatarCodigo(e.target.value) })}
              placeholder="PROMO10"
              maxLength={20}
              className="w-full rounded-lg border border-tinta/20 bg-fundo px-3 py-2 font-carimbo tracking-widest text-tinta outline-none focus:border-acento"
            />
            <span className="mt-1 block text-xs text-tinta-suave">Só letras e números, automáticamente em maiúsculo.</span>
          </Campo>

          <div className="flex gap-3">
            <Campo label="Tipo de desconto" className="flex-1">
              <select
                value={form.tipo}
                onChange={(e) => setForm({ ...form, tipo: e.target.value as 'percentual' | 'fixo' })}
                className="w-full rounded-lg border border-tinta/20 bg-fundo px-3 py-2 text-tinta outline-none focus:border-acento"
              >
                <option value="percentual">Percentual (%)</option>
                <option value="fixo">Valor fixo (R$)</option>
              </select>
            </Campo>
            <Campo label={form.tipo === 'percentual' ? 'Desconto (%)' : 'Desconto (R$)'} className="flex-1">
              <input
                type="number"
                step="0.01"
                min="0.01"
                max={form.tipo === 'percentual' ? 100 : undefined}
                required
                value={form.valor || ''}
                onChange={(e) => setForm({ ...form, valor: parseFloat(e.target.value) || 0 })}
                className="w-full rounded-lg border border-tinta/20 bg-fundo px-3 py-2 text-tinta outline-none focus:border-acento"
              />
            </Campo>
          </div>

          <div className="grid grid-cols-2 gap-3">
            <Campo label="Validade (opcional)">
              <InputDate value={form.validade ?? ''} onChange={(e) => setForm({ ...form, validade: e.target.value || null })} />
            </Campo>
            <Campo label="Uso máximo (opcional)">
              <input
                type="number"
                min="1"
                value={form.uso_maximo ?? ''}
                onChange={(e) => setForm({ ...form, uso_maximo: e.target.value ? parseInt(e.target.value) : null })}
                placeholder="Ilimitado"
                className="w-full rounded-lg border border-tinta/20 bg-fundo px-3 py-2 text-tinta outline-none focus:border-acento"
              />
            </Campo>
          </div>

          <Campo label="Pedido mínimo pra usar">
            <InputPrice
              step="0.50"
              value={form.valor_minimo_pedido || ''}
              onChange={(e) => setForm({ ...form, valor_minimo_pedido: parseFloat(e.target.value) || 0 })}
              placeholder="Sem mínimo"
            />
          </Campo>

          <label className="flex items-center gap-2 text-sm text-tinta">
            <input
              type="checkbox"
              checked={form.ativo}
              onChange={(e) => setForm({ ...form, ativo: e.target.checked })}
              className="h-4 w-4 accent-acento"
            />
            Cupom ativo
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
      ) : cupons && cupons.length > 0 ? (
        <ul className="space-y-3">
          {cupons.map((cupom) => (
            <li key={cupom.id} className="rounded-2xl bg-superficie p-4 shadow-sm">
              <div className="flex items-start justify-between gap-3">
                <div>
                  <div className="flex items-center gap-2">
                    <span className="font-carimbo text-lg font-semibold tracking-widest text-tinta">
                      {cupom.codigo}
                    </span>
                    <span className={`rounded-full px-2 py-0.5 text-xs font-semibold ${cupom.ativo ? 'bg-emerald-100 text-emerald-700' : 'bg-tinta/10 text-tinta-suave'}`}>
                      {cupom.ativo ? 'Ativo' : 'Inativo'}
                    </span>
                  </div>
                  <p className="mt-1 text-sm text-tinta-suave">
                    {cupom.tipo === 'percentual'
                      ? `${cupom.valor}% de desconto`
                      : `R$ ${cupom.valor.toFixed(2).replace('.', ',')} de desconto`}
                    {cupom.valor_minimo_pedido > 0 && ` · mín. R$ ${cupom.valor_minimo_pedido.toFixed(2).replace('.', ',')}`}
                  </p>
                  <p className="text-xs text-tinta-suave">
                    {cupom.uso_atual} uso{cupom.uso_atual !== 1 ? 's' : ''}
                    {cupom.uso_maximo ? ` / ${cupom.uso_maximo}` : ' (ilimitado)'}
                    {cupom.validade && ` · válido até ${new Date(cupom.validade).toLocaleDateString('pt-BR')}`}
                  </p>
                </div>
                <div className="flex shrink-0 gap-2">
                  <button onClick={() => abrirEdicao(cupom)} className="text-sm font-medium text-acento hover:underline">Editar</button>
                  <button onClick={() => { if (confirm(`Excluir "${cupom.codigo}"?`)) mutDeletar.mutate(cupom.id); }} className="text-sm text-tinta-suave hover:text-acento">Excluir</button>
                </div>
              </div>
            </li>
          ))}
        </ul>
      ) : (
        <p className="text-tinta-suave">Nenhum cupom cadastrado ainda.</p>
      )}
    </div>
  );
}