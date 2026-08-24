import { useState } from 'react';
import { useCartStore } from '../store/cartStore';
import type { Produto, VariacaoProduto } from '../api/types';
import { ImagemModal } from './ImagemModal';
import { precoItem } from '../lib/utils';

interface Props {
  produto: Produto;
}

export function ProdutoCard({ produto }: Props) {
  const adicionar = useCartStore((state) => state.adicionar);
  const variacoes = produto.variacoes?.filter((v) => v.disponivel) ?? [];
  const temVariacoes = variacoes.length > 0;

  const [fotoAtiva, setFotoAtiva] = useState(0);
  const [modalAberto, setModalAberto] = useState(false);
  const [modalIndice, setModalIndice] = useState(0);

  const [variacaoSelecionada, setVariacaoSelecionada] = useState<VariacaoProduto | null>(
    variacoes.length === 1 ? variacoes[0] : null
  );

  // A variação escolhida pode ter fotos próprias (modo de preço
  // "absoluto") — nesse caso elas substituem a galeria do produto.
  const fotos =
    variacaoSelecionada?.fotos && variacaoSelecionada.fotos.length > 0
      ? variacaoSelecionada.fotos
      : produto.fotos && produto.fotos.length > 0
      ? produto.fotos
      : produto.foto_url
      ? [{ id: 0, produto_id: produto.id, url: produto.foto_url, ordem: 0 }]
      : [];

  function selecionarVariacao(v: VariacaoProduto | null) {
    setVariacaoSelecionada(v);
    setFotoAtiva(0);
  }

  const precoFinal = precoItem(produto, variacaoSelecionada);

  function abrirModal(indice: number) {
    if (fotos.length === 0) return;
    setModalIndice(indice);
    setModalAberto(true);
  }

  function handleAdicionar() {
    // Escolher uma variação é opcional — o cliente pode adicionar o
    // produto "puro" (preço/foto base) mesmo quando ele tem variações.
    adicionar(produto, variacaoSelecionada ?? undefined);
    if (variacoes.length > 1) selecionarVariacao(null);
  }

  return (
    <>
      <div className="group flex gap-4 rounded-2xl bg-superficie p-4 shadow-[0_2px_0_0_rgba(43,33,24,0.08)] transition hover:-translate-y-0.5 hover:shadow-[0_6px_0_0_rgba(43,33,24,0.08)]">

        {/* Foto com carrossel e clique pra ampliar */}
        <div className="shrink-0">
          {fotos.length > 0 ? (
            <div className="relative">
              <button
                onClick={() => abrirModal(fotoAtiva)}
                className="block overflow-hidden rounded-full focus:outline-none"
                aria-label="Ampliar foto"
              >
                <img
                  src={fotos[fotoAtiva]?.url}
                  alt={produto.nome}
                  className="h-20 w-20 rounded-full object-cover transition hover:brightness-90"
                />
              </button>
              {/* Dots pra navegar entre fotos sem abrir o modal */}
              {fotos.length > 1 && (
                <div className="mt-1 flex justify-center gap-1">
                  {fotos.map((_, i) => (
                    <button
                      key={i}
                      onClick={() => setFotoAtiva(i)}
                      className={`h-1.5 rounded-full transition-all ${i === fotoAtiva ? 'w-4 bg-acento' : 'w-1.5 bg-tinta/20'}`}
                      aria-label={`Ver foto ${i + 1}`}
                    />
                  ))}
                </div>
              )}
            </div>
          ) : (
            <div className="flex h-20 w-20 items-center justify-center rounded-full border-2 border-dashed border-tinta/25 bg-fundo">
              <span className="font-display text-2xl text-tinta/40">
                {produto.nome.charAt(0).toUpperCase()}
              </span>
            </div>
          )}
        </div>

        <div className="flex flex-1 flex-col justify-between gap-2">
          <div>
            <h3 className="font-display text-lg leading-none tracking-wide text-tinta">
              {produto.nome}
            </h3>
            {produto.descricao && (
              <p className="mt-1.5 text-sm text-tinta-suave">{produto.descricao}</p>
            )}
          </div>

          {/* Seletor de variações */}
          {temVariacoes && (
            <div className="flex flex-wrap gap-1.5">
              {variacoes.map((v) => (
                <button
                  key={v.id}
                  onClick={() => selecionarVariacao(variacaoSelecionada?.id === v.id ? null : v)}
                  className={`rounded-full border px-3 py-1 text-xs font-semibold transition ${
                    variacaoSelecionada?.id === v.id
                      ? 'border-acento bg-acento text-texto-claro'
                      : 'border-tinta/20 text-tinta hover:border-acento/50'
                  }`}
                >
                  {v.nome}
                  {v.mostrar_valor_adicional && (
                    v.modo_preco === 'absoluto'
                      ? <span className="ml-1 opacity-70">R${v.preco_adicional.toFixed(2).replace('.', ',')}</span>
                      : v.preco_adicional > 0 && (
                        <span className="ml-1 opacity-70">
                          +R${v.preco_adicional.toFixed(2).replace('.', ',')}
                        </span>
                      )
                  )}
                </button>
              ))}
            </div>
          )}

          <div className="flex items-end justify-between gap-2">
            <span className="inline-flex items-center rounded-full border-2 border-acento px-3 py-0.5 font-carimbo text-sm font-semibold text-acento">
              R$ {precoFinal.toFixed(2).replace('.', ',')}
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
      {modalAberto && fotos.length > 0 && (
        <ImagemModal
          fotos={fotos}
          indiceInicial={modalIndice}
          onFechar={() => setModalAberto(false)}
        />
      )}
    </>
  );
}