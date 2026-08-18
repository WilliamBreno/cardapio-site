import { useEffect, useState } from 'react';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import {
  listarInsumos, buscarFichaTecnica, salvarFichaTecnica, type FichaTecnicaItemInput,
} from '../../api/admin';
import type { Produto } from '../../api/types';

function formatarReal(v: number) {
  return `R$ ${v.toFixed(2).replace('.', ',')}`;
}

// Uma linha em edição — guarda o id do insumo como string pra permitir
// "nenhum selecionado ainda" (select vazio) sem precisar de 0 como
// sentinela misturado com ids reais.
interface LinhaEditavel {
  insumoId: string;
  quantidade: string;
}

// FichaTecnicaModal edita os insumos que compõem um produto (Fase 9.1,
// plano Scale) e mostra o CMV calculado ao vivo — a mesma fórmula do
// backend (Σ quantidade × custo por unidade de uso de cada insumo),
// recalculada no cliente a cada edição pra não precisar de round-trip a
// cada tecla, e reconciliada com a resposta do servidor só ao salvar.
export function FichaTecnicaModal({ produto, onFechar }: { produto: Produto; onFechar: () => void }) {
  const queryClient = useQueryClient();
  const { data: insumos } = useQuery({ queryKey: ['insumos'], queryFn: listarInsumos });
  const { data: fichaAtual, isLoading } = useQuery({
    queryKey: ['ficha-tecnica', produto.id],
    queryFn: () => buscarFichaTecnica(produto.id),
  });

  const [linhas, setLinhas] = useState<LinhaEditavel[]>([]);
  const [salvando, setSalvando] = useState(false);
  const [erro, setErro] = useState<string | null>(null);

  useEffect(() => {
    if (fichaAtual) {
      setLinhas(fichaAtual.itens.map((item) => ({ insumoId: String(item.insumo_id), quantidade: String(item.quantidade) })));
    }
  }, [fichaAtual]);

  function adicionarLinha() {
    setLinhas([...linhas, { insumoId: '', quantidade: '' }]);
  }

  function removerLinha(i: number) {
    setLinhas(linhas.filter((_, idx) => idx !== i));
  }

  function atualizarLinha(i: number, campo: keyof LinhaEditavel, valor: string) {
    setLinhas(linhas.map((linha, idx) => (idx === i ? { ...linha, [campo]: valor } : linha)));
  }

  const cmv = linhas.reduce((total, linha) => {
    const insumo = insumos?.find((ins) => ins.id === Number(linha.insumoId));
    const quantidade = parseFloat(linha.quantidade);
    if (!insumo || !quantidade || insumo.fator_conversao <= 0) return total;
    return total + quantidade * (insumo.custo_unidade_compra / insumo.fator_conversao);
  }, 0);
  const margem = produto.preco - cmv;

  async function salvar() {
    const invalidas = linhas.filter((l) => !l.insumoId || !l.quantidade || parseFloat(l.quantidade) <= 0);
    if (invalidas.length > 0) {
      setErro('Toda linha precisa de um insumo e uma quantidade maior que zero.');
      return;
    }
    const ids = linhas.map((l) => l.insumoId);
    if (new Set(ids).size !== ids.length) {
      setErro('O mesmo insumo não pode aparecer duas vezes — some as quantidades numa linha só.');
      return;
    }
    setErro(null);
    setSalvando(true);
    try {
      const itens: FichaTecnicaItemInput[] = linhas.map((l) => ({
        insumo_id: Number(l.insumoId),
        quantidade: parseFloat(l.quantidade),
      }));
      await salvarFichaTecnica(produto.id, itens);
      queryClient.invalidateQueries({ queryKey: ['ficha-tecnica', produto.id] });
      onFechar();
    } catch {
      setErro('Não foi possível salvar a ficha técnica.');
    } finally {
      setSalvando(false);
    }
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 px-4" onClick={onFechar}>
      <div
        className="max-h-[85vh] w-full max-w-lg overflow-y-auto rounded-2xl bg-superficie p-5 shadow-lg"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="mb-4 flex items-start justify-between">
          <div>
            <h2 className="font-display text-lg tracking-wide text-tinta">Ficha técnica — {produto.nome}</h2>
            <p className="text-sm text-tinta-suave">Insumos consumidos por unidade vendida deste produto.</p>
          </div>
          <button onClick={onFechar} className="text-tinta-suave hover:text-tinta" aria-label="Fechar">✕</button>
        </div>

        {isLoading ? (
          <p className="text-sm text-tinta-suave">Carregando...</p>
        ) : !insumos || insumos.length === 0 ? (
          <p className="rounded-xl bg-fundo p-3 text-sm text-tinta-suave">
            Nenhum insumo cadastrado ainda — cadastre insumos na tela de Insumos antes de montar a
            ficha técnica.
          </p>
        ) : (
          <>
            <div className="space-y-2">
              {linhas.map((linha, i) => {
                const insumoSelecionado = insumos.find((ins) => ins.id === Number(linha.insumoId));
                return (
                  <div key={i} className="flex items-center gap-2">
                    <select
                      value={linha.insumoId}
                      onChange={(e) => atualizarLinha(i, 'insumoId', e.target.value)}
                      className="flex-1 rounded-lg border border-tinta/20 bg-fundo px-2 py-1.5 text-sm text-tinta outline-none focus:border-acento"
                    >
                      <option value="">Selecione um insumo...</option>
                      {insumos.map((ins) => (
                        <option key={ins.id} value={ins.id}>{ins.nome}</option>
                      ))}
                    </select>
                    <input
                      type="number"
                      step="0.01"
                      min="0.01"
                      placeholder="qtd."
                      value={linha.quantidade}
                      onChange={(e) => atualizarLinha(i, 'quantidade', e.target.value)}
                      className="w-20 rounded-lg border border-tinta/20 bg-fundo px-2 py-1.5 text-sm text-tinta outline-none focus:border-acento"
                    />
                    <span className="w-10 shrink-0 text-xs text-tinta-suave">{insumoSelecionado?.unidade_uso ?? ''}</span>
                    <button onClick={() => removerLinha(i)} className="shrink-0 text-tinta-suave hover:text-acento" aria-label="Remover">✕</button>
                  </div>
                );
              })}
            </div>

            <button
              onClick={adicionarLinha}
              className="mt-2 text-sm font-medium text-acento hover:underline"
            >
              + Adicionar insumo
            </button>

            <div className="mt-4 space-y-1 rounded-xl bg-fundo p-3 text-sm">
              <div className="flex justify-between text-tinta-suave">
                <span>Preço de venda</span>
                <span>{formatarReal(produto.preco)}</span>
              </div>
              <div className="flex justify-between text-tinta-suave">
                <span>CMV (custo dos insumos)</span>
                <span>{formatarReal(cmv)}</span>
              </div>
              <div className={`flex justify-between font-semibold ${margem < 0 ? 'text-acento' : 'text-tinta'}`}>
                <span>Margem</span>
                <span>{formatarReal(margem)}</span>
              </div>
            </div>

            {erro && <p className="mt-3 text-sm text-acento">{erro}</p>}

            <div className="mt-4 flex gap-3">
              <button type="button" onClick={onFechar} className="rounded-full border border-tinta/20 px-4 py-2 text-sm font-semibold text-tinta">Cancelar</button>
              <button onClick={salvar} disabled={salvando} className="rounded-full bg-acento px-4 py-2 text-sm font-semibold text-superficie disabled:opacity-60">
                {salvando ? 'Salvando...' : 'Salvar ficha técnica'}
              </button>
            </div>
          </>
        )}
      </div>
    </div>
  );
}
