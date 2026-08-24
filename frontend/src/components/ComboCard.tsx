import { useState } from 'react';
import { useCartStore } from '../store/cartStore';
import type { Combo, SelecaoComboItem, TipoProduto, VariacaoProduto } from '../api/types';
import { rotuloCombo } from '../lib/utils';
import { ImagemModal } from './ImagemModal';

interface Props {
  combo: Combo;
  segmentoLoja?: TipoProduto;
}

// ComboCard espelha o ProdutoCard, mas pra um pacote fixo de produtos: o
// cliente escolhe a variação de cada componente que tiver (igual comprando
// avulso) antes de adicionar o combo inteiro como uma linha só no carrinho.
export function ComboCard({ combo, segmentoLoja }: Props) {
  const adicionarCombo = useCartStore((state) => state.adicionarCombo);

  const [selecoes, setSelecoes] = useState<Record<number, VariacaoProduto | null>>({});
  const [modalAberto, setModalAberto] = useState(false);

  function selecionarVariacao(comboItemId: number, variacao: VariacaoProduto | null) {
    setSelecoes((atual) => ({ ...atual, [comboItemId]: variacao }));
  }

  function handleAdicionar() {
    const selecoesFinais: SelecaoComboItem[] = combo.itens.map((item) => ({
      comboItemId: item.id,
      produtoId: item.produto_id,
      variacao: selecoes[item.id] ?? undefined,
    }));
    adicionarCombo(combo, selecoesFinais);
    setSelecoes({});
  }

  return (
    <>
      <div className="group flex gap-4 rounded-2xl bg-superficie p-4 shadow-[0_2px_0_0_rgba(43,33,24,0.08)] transition hover:-translate-y-0.5 hover:shadow-[0_6px_0_0_rgba(43,33,24,0.08)]">
        <div className="shrink-0">
          {combo.foto_url ? (
            <button
              onClick={() => setModalAberto(true)}
              className="block overflow-hidden rounded-full focus:outline-none"
              aria-label="Ampliar foto"
            >
              <img
                src={combo.foto_url}
                alt={combo.nome}
                className="h-20 w-20 rounded-full object-cover transition hover:brightness-90"
              />
            </button>
          ) : (
            <div className="flex h-20 w-20 items-center justify-center rounded-full border-2 border-dashed border-tinta/25 bg-fundo">
              <span className="font-display text-2xl text-tinta/40">
                {combo.nome.charAt(0).toUpperCase()}
              </span>
            </div>
          )}
        </div>

        <div className="flex flex-1 flex-col justify-between gap-2">
          <div>
            <div className="flex items-center gap-1.5">
              <span className="rounded-full bg-acento/10 px-2 py-0.5 text-[10px] font-semibold uppercase tracking-wide text-acento">
                {rotuloCombo(segmentoLoja)}
              </span>
            </div>
            <h3 className="font-display text-lg leading-none tracking-wide text-tinta">
              {combo.nome}
            </h3>
            {combo.descricao && (
              <p className="mt-1.5 text-sm text-tinta-suave">{combo.descricao}</p>
            )}
            <p className="mt-1 text-xs text-tinta-suave">
              {combo.itens.map((item) => `${item.quantidade}x ${item.produto?.nome ?? ''}`).join(' + ')}
            </p>
          </div>

          {combo.itens.some((item) => (item.produto?.variacoes ?? []).some((v) => v.disponivel)) && (
            <div className="space-y-1.5">
              {combo.itens.map((item) => {
                const variacoes = (item.produto?.variacoes ?? []).filter((v) => v.disponivel);
                if (variacoes.length === 0) return null;
                return (
                  <div key={item.id}>
                    <span className="text-xs text-tinta-suave">{item.produto?.nome}:</span>
                    <div className="mt-1 flex flex-wrap gap-1.5">
                      {variacoes.map((v) => (
                        <button
                          key={v.id}
                          onClick={() => selecionarVariacao(item.id, selecoes[item.id]?.id === v.id ? null : v)}
                          className={`rounded-full border px-2.5 py-0.5 text-xs font-semibold transition ${
                            selecoes[item.id]?.id === v.id
                              ? 'border-acento bg-acento text-texto-claro'
                              : 'border-tinta/20 text-tinta hover:border-acento/50'
                          }`}
                        >
                          {v.nome}
                        </button>
                      ))}
                    </div>
                  </div>
                );
              })}
            </div>
          )}

          <div className="flex items-end justify-between gap-2">
            <span className="inline-flex items-center rounded-full border-2 border-acento px-3 py-0.5 font-carimbo text-sm font-semibold text-acento">
              R$ {combo.preco.toFixed(2).replace('.', ',')}
            </span>

            <button
              onClick={handleAdicionar}
              className="btn-neu-primario btn-neu-sm"
            >
              Adicionar
            </button>
          </div>
        </div>
      </div>

      {/* Modal lightbox */}
      {modalAberto && combo.foto_url && (
        <ImagemModal
          fotos={[{ id: 0, url: combo.foto_url }]}
          onFechar={() => setModalAberto(false)}
        />
      )}
    </>
  );
}
