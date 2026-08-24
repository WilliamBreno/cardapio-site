import { useEffect, useState } from 'react';
import { Link, useParams, useSearchParams } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import { buscarCardapio } from '../api/catalogo';
import { ProdutoCard } from '../components/ProdutoCard';
import { ComboCard } from '../components/ComboCard';
import { AbasCategorias } from '../components/AbasCategorias';
import { CatalogoGrid } from '../components/CatalogoGrid';
import { CarrinhoFlutuante } from '../components/CarrinhoFlutuante';
import { CarrinhoDrawer } from '../components/CarrinhoDrawer';
import { HistoricoDrawer } from '../components/HistoricoDrawer';
import { GuardadosDrawer } from '../components/GuardadosDrawer';
import { rotuloCatalogo, rotuloCombo, cn } from '../lib/utils';
import type { TipoProduto, Produto, Subcategoria, GrupoCor, FotoBanner } from '../api/types';

// Lembra o segmento da loja (cardápio/catálogo) de uma visita anterior,
// só pra acertar a palavra certa na tela de "abrindo..." — nesse momento
// a resposta da API ainda não chegou, então não temos segmento_principal
// de verdade ainda. Numa primeira visita mesmo (sem nada salvo), cai no
// padrão "cardápio" de rotuloCatalogo(undefined), sem regressão nenhuma.
function segmentoLembrado(slug: string | undefined): TipoProduto | undefined {
  if (!slug) return undefined;
  const valor = localStorage.getItem(`drenux-segmento-${slug}`);
  return valor === 'mercadoria' || valor === 'alimenticio' ? valor : undefined;
}

function lojaEstaAberta(loja: {
  horario_abertura: string;
  horario_fechamento: string;
  margem_fechamento_minutos: number;
  pausado: boolean;
}): boolean {
  if (loja.pausado) return false;
  if (!loja.horario_abertura || !loja.horario_fechamento) return true;

  const agora = new Date();
  const hhmm = agora.toLocaleTimeString('pt-BR', {
    hour: '2-digit',
    minute: '2-digit',
    hour12: false,
    timeZone: 'America/Sao_Paulo',
  });

  let fechamento = loja.horario_fechamento;
  if (loja.margem_fechamento_minutos > 0) {
    const [h, m] = loja.horario_fechamento.split(':').map(Number);
    const total = h * 60 + m - loja.margem_fechamento_minutos;
    const hf = Math.floor(total / 60).toString().padStart(2, '0');
    const mf = (total % 60).toString().padStart(2, '0');
    fechamento = `${hf}:${mf}`;
  }

  return hhmm >= loja.horario_abertura && hhmm < fechamento;
}

// renderProdutosAgrupados exibe os produtos de UMA categoria já filtrada
// no cardápio público — se a loja cadastrou Subcategoria/Grupo pra essa
// categoria (Fase 6, agora disponível pra qualquer segmento, não só
// "mercadoria"), agrupa por subcategoria e, dentro dela, por grupo, com
// um cabeçalho pra cada nível. Produto sem subcategoria/grupo continua
// aparecendo solto. Categoria sem nenhuma subcategoria cadastrada
// degrada pra lista simples de sempre — mesmo espírito do agrupamento já
// usado no admin (ver Produtos.tsx, renderProdutosDaCategoria).
function renderProdutosAgrupados(produtos: Produto[], categoriaId: number, subcategorias: Subcategoria[], gruposCor: GrupoCor[]) {
  const subsDaCategoria = subcategorias.filter((s) => s.categoria_id === categoriaId);
  if (subsDaCategoria.length === 0) {
    return produtos.map((produto) => <ProdutoCard key={produto.id} produto={produto} />);
  }

  const semSubcategoria = produtos.filter((p) => p.subcategoria_id === null);

  return (
    <>
      {semSubcategoria.map((produto) => <ProdutoCard key={produto.id} produto={produto} />)}
      {subsDaCategoria.map((sub) => {
        const produtosDaSub = produtos.filter((p) => p.subcategoria_id === sub.id);
        if (produtosDaSub.length === 0) return null;

        const gruposDaSub = gruposCor.filter((g) => g.subcategoria_id === sub.id);
        const semGrupo = produtosDaSub.filter((p) => p.grupo_cor_id === null);

        return (
          <div key={sub.id} className="space-y-3 pt-2">
            <h3 className="font-display text-base tracking-wide text-tinta">{sub.nome}</h3>
            {gruposDaSub.length === 0 ? (
              produtosDaSub.map((produto) => <ProdutoCard key={produto.id} produto={produto} />)
            ) : (
              <>
                {semGrupo.map((produto) => <ProdutoCard key={produto.id} produto={produto} />)}
                {gruposDaSub.map((grupo) => {
                  const produtosDoGrupo = produtosDaSub.filter((p) => p.grupo_cor_id === grupo.id);
                  if (produtosDoGrupo.length === 0) return null;
                  return (
                    <div key={grupo.id} className="space-y-3 border-l-2 border-tinta/10 pl-3">
                      <h4 className="text-sm font-medium text-acento">{grupo.nome}</h4>
                      {produtosDoGrupo.map((produto) => <ProdutoCard key={produto.id} produto={produto} />)}
                    </div>
                  );
                })}
              </>
            )}
          </div>
        );
      })}
    </>
  );
}

// BannerCarrossel (redesign de 24/08/2026, substitui o BannerOferta de
// foto única): card dedicado abaixo do cabeçalho da loja, com largura
// proporcional mas limitada (max-w-4xl — não vai mais de ponta a ponta
// da tela em telas largas). A altura do card se ajusta à proporção de
// cada foto (w-full h-auto, sem object-cover nem altura fixa) — testado
// com object-cover antes e o corte cortava texto importante de peças
// promocionais (ex: título no topo da arte), então a prioridade agora é
// nunca cortar a imagem, mesmo que a altura do card varie de foto pra
// foto no carrossel. Com mais de uma foto, alterna sozinho a cada 5s e
// mostra os pontinhos de navegação; a troca usa fade-in (tw-animate-css)
// em vez do crossfade por opacidade empilhada de antes, que exigia
// altura fixa pra funcionar.
function BannerCarrossel({ fotos }: { fotos: FotoBanner[] }) {
  const [indice, setIndice] = useState(0);

  useEffect(() => {
    if (fotos.length <= 1) return;
    const intervalo = setInterval(() => setIndice((i) => (i + 1) % fotos.length), 5000);
    return () => clearInterval(intervalo);
  }, [fotos.length]);

  if (fotos.length === 0) return null;

  const fotoAtual = fotos[indice];

  return (
    <div className="mx-auto max-w-4xl px-4 sm:px-6">
      <div className="relative mt-4 overflow-hidden rounded-2xl shadow-sm">
        <img
          key={fotoAtual.id}
          src={fotoAtual.url}
          alt="Oferta em destaque"
          className="block h-auto w-full animate-in fade-in-0 duration-500"
        />
        {fotos.length > 1 && (
          <div className="absolute bottom-2 left-1/2 flex -translate-x-1/2 gap-1.5">
            {fotos.map((foto, i) => (
              <button
                key={foto.id}
                onClick={() => setIndice(i)}
                aria-label={`Foto ${i + 1}`}
                className={cn(
                  'h-1.5 w-1.5 rounded-full transition',
                  i === indice ? 'bg-superficie' : 'bg-superficie/40'
                )}
              />
            ))}
          </div>
        )}
      </div>
    </div>
  );
}

export function CardapioPublico() {
  const { slug } = useParams<{ slug: string }>();
  const [searchParams] = useSearchParams();
  const pagamentoConfirmado = searchParams.get('pago') === '1';
  const [categoriaAtiva, setCategoriaAtiva] = useState<number | null>(null);
  const [carrinhoAberto, setCarrinhoAberto] = useState(false);
  const [historicoAberto, setHistoricoAberto] = useState(false);
  const [guardadosAberto, setGuardadosAberto] = useState(false);

  const { data, isLoading, isError } = useQuery({
    queryKey: ['cardapio', slug],
    queryFn: () => buscarCardapio(slug!),
    enabled: !!slug,
    refetchInterval: 60_000, // revalida o status de aberta a cada minuto
  });

  // Guarda o segmento assim que a resposta chega — assim, numa próxima
  // visita a essa mesma loja, a tela de "abrindo..." já acerta a palavra
  // (cardápio/catálogo) antes mesmo da API responder de novo.
  useEffect(() => {
    if (slug && data?.loja.segmento_principal) {
      localStorage.setItem(`drenux-segmento-${slug}`, data.loja.segmento_principal);
    }
  }, [slug, data?.loja.segmento_principal]);

  if (isLoading) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-fundo">
        <p className="font-display tracking-wide text-tinta-suave">
          Abrindo o {rotuloCatalogo(segmentoLembrado(slug))}...
        </p>
      </div>
    );
  }

  if (isError || !data) {
    return (
      <div className="flex min-h-screen flex-col items-center justify-center gap-2 bg-fundo px-6 text-center">
        <p className="font-display text-2xl text-tinta">Loja não encontrada</p>
        <p className="text-tinta-suave">Confere se o link está certo.</p>
      </div>
    );
  }

  const aberta = lojaEstaAberta(data.loja);

  // Se a loja está pausada ou fechada, mostra só o aviso
  if (!aberta) {
    const mensagem = data.loja.pausado
      ? data.loja.mensagem_pausa || 'Loja temporariamente fechada.'
      : `Estamos fechados no momento. Funcionamos das ${data.loja.horario_abertura} às ${data.loja.horario_fechamento}.`;

    return (
      <div className="min-h-screen bg-fundo" data-tema={data.loja.tema || 'kraft'}>
        <header className="bg-acento px-6 py-8 text-center">
          {data.loja.logo_url && (
            <img src={data.loja.logo_url} alt={data.loja.nome}
              className="mx-auto mb-3 h-16 w-16 rounded-full border-2 border-superficie/40 object-cover" />
          )}
          <h1 className="font-display text-3xl tracking-wide text-superficie">{data.loja.nome}</h1>
        </header>
        <BannerCarrossel fotos={data.loja.banner_fotos} />
        <div className="flex flex-col items-center justify-center gap-3 px-6 py-16 text-center">
          <span className="rounded-full bg-tinta/10 px-4 py-1 font-carimbo text-xs uppercase tracking-widest text-tinta-suave">
            {data.loja.pausado ? 'Produção pausada' : 'Fechado'}
          </span>
          <p className="max-w-sm text-tinta-suave">{mensagem}</p>
        </div>
      </div>
    );
  }

  const ehMercadoria = data.loja.segmento_principal === 'mercadoria';

  const produtosFiltrados = categoriaAtiva
    ? data.produtos.filter((produto) => produto.categoria_id === categoriaAtiva)
    : data.produtos;

  return (
    <div className="min-h-screen bg-fundo pb-28" data-tema={data.loja.tema || 'kraft'}>
      {pagamentoConfirmado && (
        <div className="bg-emerald-600 px-6 py-3 text-center text-sm font-medium text-white">
          {data.loja.avisos_whatsapp_cliente
            ? 'Pedido confirmado! Você vai receber uma mensagem no WhatsApp com os detalhes.'
            : 'Pedido confirmado! Acompanhe o status em "Meus pedidos".'}
        </div>
      )}

      <header className="bg-acento px-6 py-8 text-center">
        {data.loja.logo_url && (
          <img src={data.loja.logo_url} alt={data.loja.nome}
            className="mx-auto mb-3 h-16 w-16 rounded-full border-2 border-superficie/40 object-cover" />
        )}
        <h1 className="font-display text-3xl tracking-wide text-superficie sm:text-4xl">
          {data.loja.nome}
        </h1>
        {data.loja.horario_abertura && data.loja.horario_fechamento && (
          <p className="mt-1 font-carimbo text-xs uppercase tracking-[0.2em] text-superficie/70">
            {data.loja.horario_abertura} – {data.loja.horario_fechamento}
          </p>
        )}
        <div className="mt-3 flex justify-center gap-2">
          <button
            onClick={() => setHistoricoAberto(true)}
            className="rounded-full border border-superficie/30 px-4 py-1.5 text-xs font-medium text-superficie/80 hover:bg-superficie/10"
          >
            Meus pedidos
          </button>
          {data.loja.aceita_guardar_entregar && (
            <button
              onClick={() => setGuardadosAberto(true)}
              className="rounded-full border border-superficie/30 px-4 py-1.5 text-xs font-medium text-superficie/80 hover:bg-superficie/10"
            >
              📦 Itens guardados
            </button>
          )}
        </div>
      </header>

      <BannerCarrossel fotos={data.loja.banner_fotos} />

      {ehMercadoria ? (
        <main className="mx-auto max-w-4xl">
          <CatalogoGrid
            produtos={data.produtos}
            categorias={data.categorias}
            subcategorias={data.subcategorias}
            gruposCor={data.grupos_cor}
          />
          {(data.combos?.length ?? 0) > 0 && (
            <section className="mx-auto max-w-2xl space-y-3 px-4 pb-4 pt-2">
              <h2 className="font-display text-lg tracking-wide text-tinta">{rotuloCombo(data.loja.segmento_principal)}s</h2>
              {data.combos.map((combo) => (
                <ComboCard key={combo.id} combo={combo} segmentoLoja={data.loja.segmento_principal} />
              ))}
            </section>
          )}
        </main>
      ) : (
        <>
          <div className="sticky top-0 z-10 bg-fundo/95 py-3 backdrop-blur">
            <AbasCategorias
              categorias={data.categorias}
              ativa={categoriaAtiva}
              onSelecionar={setCategoriaAtiva}
            />
          </div>

          <main className="mx-auto max-w-2xl space-y-3 px-4 pt-2">
            {(data.combos?.length ?? 0) > 0 && (
              <section className="space-y-3 pb-2">
                <h2 className="font-display text-lg tracking-wide text-tinta">{rotuloCombo(data.loja.segmento_principal)}s</h2>
                {data.combos.map((combo) => (
                  <ComboCard key={combo.id} combo={combo} segmentoLoja={data.loja.segmento_principal} />
                ))}
              </section>
            )}

            {produtosFiltrados.length === 0 ? (
              <p className="py-12 text-center text-tinta-suave">Nenhum produto por aqui ainda.</p>
            ) : categoriaAtiva !== null ? (
              // Só agrupa por Subcategoria/Grupo quando uma categoria
              // específica está selecionada — com "Tudo" ativo, os
              // produtos vêm de várias categorias diferentes e cada
              // subcategoria só faz sentido dentro da sua própria
              // categoria, então a lista continua solta nesse caso.
              renderProdutosAgrupados(produtosFiltrados, categoriaAtiva, data.subcategorias, data.grupos_cor)
            ) : (
              produtosFiltrados.map((produto) => <ProdutoCard key={produto.id} produto={produto} />)
            )}
          </main>
        </>
      )}

      <CarrinhoFlutuante onAbrir={() => setCarrinhoAberto(true)} />

      <CarrinhoDrawer
        aberto={carrinhoAberto}
        onFechar={() => setCarrinhoAberto(false)}
        slug={slug!}
        modoPedido={data.loja.modo_pedido}
        antecedenciaMinimaHoras={data.loja.antecedencia_minima_horas}
        aceitaRetirada={data.loja.aceita_retirada}
        aceitaEntrega={data.loja.aceita_entrega}
        aceitaGuardarEntregar={data.loja.aceita_guardar_entregar}
        taxaEntregaTipo={data.loja.taxa_entrega_tipo}
        taxaEntregaValor={data.loja.taxa_entrega_valor}
        valorMinimoPedido={data.loja.valor_minimo_pedido}
        sugestaoInteligenteAtiva={data.loja.sugestao_inteligente_ativa}
        produtos={data.produtos}
        segmentoLoja={data.loja.segmento_principal}
      />

      <HistoricoDrawer
        aberto={historicoAberto}
        onFechar={() => setHistoricoAberto(false)}
        slug={slug!}
        produtos={data.produtos}
        onAbrirCarrinho={() => setCarrinhoAberto(true)}
      />

      {data.loja.aceita_guardar_entregar && (
        <GuardadosDrawer
          aberto={guardadosAberto}
          onFechar={() => setGuardadosAberto(false)}
          slug={slug!}
        />
      )}

      {data.loja.mostrar_selo_drenux && (
        <p className="pb-24 pt-8 text-center text-xs text-tinta-suave/70">
          Feito com{' '}
          <Link to="/" className="font-medium text-tinta-suave underline underline-offset-2">
            Drenux
          </Link>
        </p>
      )}
    </div>
  );
}