import { useState } from 'react';
import { useCartStore } from '../store/cartStore';
import type { Produto, VariacaoProduto } from '../api/types';
import { ImagemModal } from './ImagemModal';
import { precoItem } from '../lib/utils';

interface Props {
  produto: Produto;
}

// Card vertical (foto em cima, preço/botão embaixo) — layout de e-commerce
// usado no catálogo público de lojas "mercadoria" (ver CatalogoGrid). O
// ProdutoCard "horizontal" continua sendo o padrão pra "alimenticio".
export function ProdutoCardGrid({ produto }: Props) {
  const adicionar = useCartStore((state) => state.adicionar);
  const variacoes = produto.variacoes?.filter((v) => v.disponivel) ?? [];
  const temVariacoes = variacoes.length > 0;

  const [modalAberto, setModalAberto] = useState(false);
  const [variacaoSelecionada, setVariacaoSelecionada] = useState<VariacaoProduto | null>(
    variacoes.length === 1 ? variacoes[0] : null
  );

  const fotos =
    variacaoSelecionada?.fotos && variacaoSelecionada.fotos.length > 0
      ? variacaoSelecionada.fotos
      : produto.fotos && produto.fotos.length > 0
      ? produto.fotos
      : produto.foto_url
      ? [{ id: 0, produto_id: produto.id, url: produto.foto_url, ordem: 0 }]
      : [];

  const precoFinal = precoItem(produto, variacaoSelecionada);

  function selecionarVariacao(v: VariacaoProduto | null) {
    setVariacaoSelecionada(v);
  }

  function handleAdicionar() {
    // Escolher uma variação é opcional — o cliente pode adicionar o
    // produto "puro" (preço/foto base) mesmo quando ele tem variações.
    adicionar(produto, variacaoSelecionada ?? undefined);
    if (variacoes.length > 1) selecionarVariacao(null);
  }

  return (
    <>
      <div className="flex flex-col overflow-hidden rounded-2xl bg-superficie shadow-[0_2px_0_0_rgba(43,33,24,0.08)]">
        <button
          onClick={() => fotos.length > 0 && setModalAberto(true)}
          className="block aspect-square w-full overflow-hidden bg-fundo"
          aria-label="Ampliar foto"
        >
          {fotos.length > 0 ? (
            <img src={fotos[0].url} alt={produto.nome} className="h-full w-full object-cover transition hover:brightness-90" />
          ) : (
            <span className="flex h-full w-full items-center justify-center font-display text-3xl text-tinta/30">
              {produto.nome.charAt(0).toUpperCase()}
            </span>
          )}
        </button>

        <div className="flex flex-1 flex-col gap-1.5 p-3">
          <h3 className="line-clamp-2 font-display text-sm leading-tight text-tinta">{produto.nome}</h3>

          {temVariacoes && (
            <div className="flex flex-wrap gap-1">
              {variacoes.map((v) => (
                <button
                  key={v.id}
                  onClick={() => selecionarVariacao(variacaoSelecionada?.id === v.id ? null : v)}
                  className={`rounded-full border px-2 py-0.5 text-[11px] font-semibold transition ${
                    variacaoSelecionada?.id === v.id
                      ? 'border-acento bg-acento text-texto-claro'
                      : 'border-tinta/20 text-tinta hover:border-acento/50'
                  }`}
                >
                  {v.nome}
                </button>
              ))}
            </div>
          )}

          <div className="mt-auto flex items-center justify-between gap-2 pt-1">
            <span className="font-carimbo text-sm font-semibold text-acento">
              R$ {precoFinal.toFixed(2).replace('.', ',')}
            </span>
            <button
              onClick={handleAdicionar}
              className="rounded-full bg-acento px-3 py-1 text-xs font-semibold text-texto-claro transition active:scale-95 hover:bg-acento/90"
            >
              Adicionar
            </button>
          </div>
        </div>
      </div>

      {modalAberto && fotos.length > 0 && (
        <ImagemModal fotos={fotos} indiceInicial={0} onFechar={() => setModalAberto(false)} />
      )}
    </>
  );
}
