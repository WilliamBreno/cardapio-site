import { useState, type FormEvent, type ChangeEvent } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import {
  listarProdutos, criarProduto, atualizarProduto, deletarProduto,
  listarCategorias, listarSubcategorias, listarGruposCor,
  criarVariacao, atualizarVariacao, deletarVariacao,
  adicionarFoto, deletarFoto, reordenarFotos, adicionarFotoVariacao, deletarFotoVariacao, buscarLoja,
  type ProdutoInput, type VariacaoInput,
} from '../../api/admin';
import type { Produto, VariacaoProduto } from '../../api/types';
import { ProdutoFormFields } from '../../components/admin/ProdutoFormFields';
import { VariacaoFormFields } from '../../components/admin/VariacaoFormFields';
import { CadastroEmMassaDialog } from '../../components/admin/CadastroEmMassaDialog';
import { FichaTecnicaModal } from '../../components/admin/FichaTecnicaModal';
import { InfoTooltip } from '../../components/InfoTooltip';
import { enviarImagem, logoMiniatura } from '../../api/upload';
import { rotuloCatalogo } from '../../lib/utils';

const formVazio: ProdutoInput = {
  nome: '',
  descricao: '',
  preco: 0,
  foto_url: '',
  disponivel: true,
  categoria_id: 0,
  subcategoria_id: null,
  grupo_cor_id: null,
  estoque_atual: null,
  estoque_alerta: null,
  tipo_produto: 'alimenticio',
  peso_gramas: null,
};

const variacaoVazia: VariacaoInput = {
  nome: '',
  preco_adicional: 0,
  disponivel: true,
  mostrar_valor_adicional: true,
  modo_preco: 'aditivo',
  estoque_atual: null,
  estoque_alerta: null,
  ordem: 0,
};

export function Produtos() {
  const queryClient = useQueryClient();

  const { data: produtos, isLoading } = useQuery({ queryKey: ['produtos'], queryFn: listarProdutos });
  const { data: categorias } = useQuery({ queryKey: ['categorias'], queryFn: listarCategorias });
  const { data: loja } = useQuery({ queryKey: ['loja'], queryFn: buscarLoja });
  const ehMercadoria = loja?.segmento_principal === 'mercadoria';
  // Subcategoria/Grupo (Fase 6): liberado como ferramenta de organização
  // pra qualquer segmento — o agrupamento na lista abaixo só aparece de
  // fato se a categoria tiver subcategoria cadastrada (ver
  // renderProdutosDaCategoria), então não precisa mais gate por segmento
  // aqui na busca.
  const { data: subcategorias } = useQuery({ queryKey: ['subcategorias'], queryFn: listarSubcategorias });
  const { data: gruposCor } = useQuery({ queryKey: ['grupos-cor'], queryFn: listarGruposCor });

  const [editandoId, setEditandoId] = useState<number | null>(null);
  const [mostrarForm, setMostrarForm] = useState(false);
  const [mostrarCadastroEmMassa, setMostrarCadastroEmMassa] = useState(false);
  const [form, setForm] = useState<ProdutoInput>(formVazio);
  const [erro, setErro] = useState<string | null>(null);
  const [enviandoFoto, setEnviandoFoto] = useState(false);

  // Variações
  const [produtoExpandido, setProdutoExpandido] = useState<number | null>(null);
  const [editandoVariacaoId, setEditandoVariacaoId] = useState<number | null>(null);
  const [mostrarFormVariacao, setMostrarFormVariacao] = useState(false);
  const [formVariacao, setFormVariacao] = useState<VariacaoInput>(variacaoVazia);
  const [erroVariacao, setErroVariacao] = useState<string | null>(null);

  const invalidar = () => queryClient.invalidateQueries({ queryKey: ['produtos'] });

  const mutCriar = useMutation({ mutationFn: criarProduto, onSuccess: () => { invalidar(); fecharForm(); }, onError: () => setErro('Não foi possível salvar o produto.') });
  const mutAtualizar = useMutation({ mutationFn: ({ id, input }: { id: number; input: ProdutoInput }) => atualizarProduto(id, input), onSuccess: () => { invalidar(); fecharForm(); }, onError: () => setErro('Não foi possível salvar o produto.') });
  const mutDeletar = useMutation({ mutationFn: deletarProduto, onSuccess: invalidar });

  const mutCriarVar = useMutation({ mutationFn: ({ pid, input }: { pid: number; input: VariacaoInput }) => criarVariacao(pid, input), onSuccess: () => { invalidar(); fecharFormVariacao(); }, onError: () => setErroVariacao('Não foi possível salvar.') });
  const mutAtualizarVar = useMutation({ mutationFn: ({ pid, vid, input }: { pid: number; vid: number; input: VariacaoInput }) => atualizarVariacao(pid, vid, input), onSuccess: () => { invalidar(); fecharFormVariacao(); }, onError: () => setErroVariacao('Não foi possível salvar.') });
  const mutDeletarVar = useMutation({ mutationFn: ({ pid, vid }: { pid: number; vid: number }) => deletarVariacao(pid, vid), onSuccess: invalidar });

  function abrirNovo() {
    setEditandoId(null);
    setForm({ ...formVazio, categoria_id: categorias?.[0]?.id ?? 0, tipo_produto: loja?.segmento_principal ?? 'alimenticio' });
    setErro(null);
    setMostrarForm(true);
  }

  function abrirEdicao(produto: Produto) {
    setEditandoId(produto.id);
    setForm({ nome: produto.nome, descricao: produto.descricao, preco: produto.preco, foto_url: produto.foto_url, disponivel: produto.disponivel, categoria_id: produto.categoria_id, subcategoria_id: produto.subcategoria_id, grupo_cor_id: produto.grupo_cor_id, estoque_atual: produto.estoque_atual, estoque_alerta: produto.estoque_alerta, tipo_produto: produto.tipo_produto, peso_gramas: produto.peso_gramas });
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

  function salvar(e: FormEvent) {
    e.preventDefault();
    if (!form.nome.trim() || form.preco <= 0 || !form.categoria_id) { setErro('Preenche nome, preço e categoria.'); return; }
    setErro(null);
    if (editandoId) { mutAtualizar.mutate({ id: editandoId, input: form }); }
    else { mutCriar.mutate(form); }
  }

  function alternarDisponibilidade(produto: Produto) {
    mutAtualizar.mutate({ id: produto.id, input: { nome: produto.nome, descricao: produto.descricao, preco: produto.preco, foto_url: produto.foto_url, categoria_id: produto.categoria_id, subcategoria_id: produto.subcategoria_id, grupo_cor_id: produto.grupo_cor_id, disponivel: !produto.disponivel, estoque_atual: produto.estoque_atual, estoque_alerta: produto.estoque_alerta, tipo_produto: produto.tipo_produto, peso_gramas: produto.peso_gramas } });
  }

  // Variações
  function abrirNovaVariacao(produtoId: number) {
    setProdutoExpandido(produtoId);
    setEditandoVariacaoId(null);
    // Sugere o modo de preço conforme o segmento da loja — mercadoria
    // tende a ter variações com preço/foto próprios, alimentício soma
    // acréscimo sobre o preço base. É só um padrão, o lojista pode trocar.
    setFormVariacao({ ...variacaoVazia, modo_preco: loja?.segmento_principal === 'mercadoria' ? 'absoluto' : 'aditivo' });
    setErroVariacao(null);
    setMostrarFormVariacao(true);
  }

  function abrirEdicaoVariacao(produtoId: number, variacao: VariacaoProduto) {
    setProdutoExpandido(produtoId);
    setEditandoVariacaoId(variacao.id);
    setFormVariacao({
      nome: variacao.nome,
      preco_adicional: variacao.preco_adicional,
      disponivel: variacao.disponivel,
      mostrar_valor_adicional: variacao.mostrar_valor_adicional,
      modo_preco: variacao.modo_preco,
      estoque_atual: variacao.estoque_atual,
      estoque_alerta: variacao.estoque_alerta,
      ordem: variacao.ordem,
    });
    setErroVariacao(null);
    setMostrarFormVariacao(true);
  }

  function fecharFormVariacao() { setMostrarFormVariacao(false); setEditandoVariacaoId(null); }

  function salvarVariacao(e: FormEvent, produtoId: number) {
    e.preventDefault();
    if (!formVariacao.nome.trim()) { setErroVariacao('Digita um nome.'); return; }
    setErroVariacao(null);
    if (editandoVariacaoId) { mutAtualizarVar.mutate({ pid: produtoId, vid: editandoVariacaoId, input: formVariacao }); }
    else { mutCriarVar.mutate({ pid: produtoId, input: formVariacao }); }
  }

  // Fotos
  const [produtoFotos, setProdutoFotos] = useState<number | null>(null);
  const [produtoFichaTecnica, setProdutoFichaTecnica] = useState<Produto | null>(null);
  const [enviandoFotoProduto, setEnviandoFotoProduto] = useState(false);

  const salvando = mutCriar.isPending || mutAtualizar.isPending;

  async function adicionarFotoProduto(produtoId: number, e: React.ChangeEvent<HTMLInputElement>) {
    const arquivo = e.target.files?.[0];
    if (!arquivo) return;
    setEnviandoFotoProduto(true);
    try {
      const url = await enviarImagem(arquivo);
      const produto = produtos?.find(p => p.id === produtoId);
      const ordem = produto?.fotos?.length ?? 0;
      await adicionarFoto(produtoId, url, ordem);
      invalidar();
    } finally {
      setEnviandoFotoProduto(false);
    }
  }

  async function removerFotoProduto(produtoId: number, fotoId: number) {
    if (!confirm('Remover esta foto?')) return;
    await deletarFoto(produtoId, fotoId);
    invalidar();
  }

  async function tornarFotoPrincipal(produtoId: number, fotoId: number) {
    const produto = produtos?.find((p) => p.id === produtoId);
    if (!produto?.fotos) return;
    const outras = produto.fotos.filter((f) => f.id !== fotoId);
    await reordenarFotos(produtoId, [fotoId, ...outras.map((f) => f.id)]);
    invalidar();
  }

  // Fotos de variação (modo de preço "absoluto")
  const [enviandoFotoVariacao, setEnviandoFotoVariacao] = useState(false);

  async function adicionarFotoDaVariacao(produtoId: number, variacao: VariacaoProduto, e: ChangeEvent<HTMLInputElement>) {
    const arquivo = e.target.files?.[0];
    if (!arquivo) return;
    setEnviandoFotoVariacao(true);
    try {
      const url = await enviarImagem(arquivo);
      const ordem = variacao.fotos?.length ?? 0;
      await adicionarFotoVariacao(produtoId, variacao.id, url, ordem);
      invalidar();
    } finally {
      setEnviandoFotoVariacao(false);
    }
  }

  async function removerFotoDaVariacao(produtoId: number, variacaoId: number, fotoId: number) {
    if (!confirm('Remover esta foto?')) return;
    await deletarFotoVariacao(produtoId, variacaoId, fotoId);
    invalidar();
  }

  // Renderiza o card de um produto — extraído numa função (em vez de um
  // componente à parte) só pra reaproveitar a mesma marcação tanto na
  // lista plana (alimentício) quanto na lista agrupada por
  // Categoria → Subcategoria → Grupo de Cor (mercadoria), sem duplicar
  // esse bloco grande em dois lugares.
  function renderProduto(produto: Produto) {
    return (
      <li key={produto.id} className="rounded-2xl bg-superficie shadow-sm">
        {/* Card do produto */}
        <div className="p-4 space-y-2">
          {/* Linha 1: foto + nome */}
          <div className={`flex items-center gap-3 ${!produto.disponivel ? 'opacity-50' : ''}`}>
            <div className="flex h-10 w-10 shrink-0 items-center justify-center overflow-hidden rounded-full bg-fundo">
              {/* Prioriza a galeria (fotos[0], já ordenada por "ordem" no
                  backend) sobre foto_url — foto_url é o campo legado,
                  mantido só pra produto antigo sem galeria. Sem isso, "tornar
                  principal" na galeria parecia não ter efeito nenhum aqui,
                  mesmo já refletindo certo no cardápio público. */}
              {(() => {
                const miniatura = produto.fotos?.[0]?.url ?? produto.foto_url;
                return miniatura ? (
                  <img src={logoMiniatura(miniatura)} alt={produto.nome} className="h-full w-full object-cover" />
                ) : (
                  <span className="font-display text-sm text-tinta/30">{produto.nome.charAt(0).toUpperCase()}</span>
                );
              })()}
            </div>
            <p className="font-medium text-tinta truncate">{produto.nome}</p>
          </div>

          {/* Linha 2: preço + botões */}
          <div className="flex items-center justify-between gap-2">
            <div className="min-w-0">
              <p className="font-carimbo text-sm text-tinta-suave">
                R$ {produto.preco.toFixed(2).replace('.', ',')} · {produto.categoria?.nome ?? 'sem categoria'}
              </p>
              {produto.estoque_atual !== null && produto.variacoes?.length === 0 && (
                <p className={`text-xs ${produto.estoque_atual === 0 ? 'text-acento' : 'text-tinta-suave'}`}>
                  {produto.estoque_atual === 0 ? 'Esgotado' : `${produto.estoque_atual} em estoque`}
                </p>
              )}
            </div>
            <div className="grid grid-cols-2 gap-1 shrink-0">
              <button
                onClick={() => setProdutoFotos(produtoFotos === produto.id ? null : produto.id)}
                className="rounded-full border border-tinta/15 px-2 py-1 text-xs font-semibold text-tinta-suave hover:border-acento hover:text-acento"
              >
                Fotos {produto.fotos && produto.fotos.length > 0 && `(${produto.fotos.length})`}
              </button>
              <button
                onClick={() => setProdutoExpandido(produtoExpandido === produto.id ? null : produto.id)}
                className="rounded-full border border-tinta/15 px-2 py-1 text-xs font-semibold text-tinta-suave hover:border-acento hover:text-acento"
              >
                Variações
              </button>
              {loja?.plano === 'scale' && (
                <button
                  onClick={() => setProdutoFichaTecnica(produto)}
                  className="rounded-full border border-tinta/15 px-2 py-1 text-xs font-semibold text-tinta-suave hover:border-acento hover:text-acento"
                >
                  Ficha técnica
                </button>
              )}
              <button onClick={() => alternarDisponibilidade(produto)} className={`rounded-full px-2 py-1 text-xs font-semibold ${produto.disponivel ? 'bg-douro/20 text-douro' : 'bg-tinta/10 text-tinta-suave'}`}>
                {produto.disponivel ? 'Disponível' : 'Pausado'}
              </button>
              <button onClick={() => abrirEdicao(produto)} className="rounded-full border border-acento/30 px-2 py-1 text-xs font-medium text-acento hover:bg-acento/5">Editar</button>
              <button onClick={() => { if (confirm(`Excluir "${produto.nome}"?`)) mutDeletar.mutate(produto.id); }} className="rounded-full border border-tinta/15 px-2 py-1 text-xs text-tinta-suave hover:text-acento col-span-2">Excluir</button>
            </div>
          </div>
        </div>

        {/* Painel de fotos */}
        {produtoFotos === produto.id && (
          <div className="border-t border-tinta/10 px-4 pb-4 pt-3 space-y-3">
            <div className="flex items-center justify-between">
              <p className="text-xs font-medium uppercase tracking-wide text-tinta-suave">Galeria de fotos</p>
              <label className="cursor-pointer rounded-full bg-tinta px-3 py-1 text-xs font-semibold text-superficie">
                {enviandoFotoProduto ? 'Enviando...' : '+ Adicionar'}
                <input type="file" accept="image/*" className="hidden" disabled={enviandoFotoProduto}
                  onChange={(e) => adicionarFotoProduto(produto.id, e)} />
              </label>
            </div>
            {produto.fotos && produto.fotos.length > 0 ? (
              <div className="flex flex-wrap gap-2">
                {produto.fotos.map((foto, i) => (
                  <div key={foto.id} className="relative group">
                    <img
                      src={foto.url}
                      alt={`Foto ${i + 1}`}
                      onClick={() => i !== 0 && tornarFotoPrincipal(produto.id, foto.id)}
                      title={i === 0 ? 'Foto principal' : 'Clique pra tornar principal'}
                      className={`h-16 w-16 rounded-xl object-cover ${i !== 0 ? 'cursor-pointer' : ''}`}
                    />
                    <button
                      onClick={() => removerFotoProduto(produto.id, foto.id)}
                      title="Remover foto"
                      className="absolute -right-1 -top-1 hidden h-5 w-5 items-center justify-center rounded-full bg-acento text-xs text-superficie group-hover:flex"
                    >×</button>
                    {i === 0 && (
                      <span className="pointer-events-none absolute bottom-0 left-0 right-0 rounded-b-xl bg-tinta/60 py-0.5 text-center text-xs text-superficie">
                        Principal
                      </span>
                    )}
                  </div>
                ))}
              </div>
            ) : (
              <p className="text-xs text-tinta-suave">Nenhuma foto ainda. A primeira foto adicionada será a principal.</p>
            )}
          </div>
        )}
        {produtoExpandido === produto.id && (
          <div className="border-t border-tinta/10 px-4 pb-4 pt-3 space-y-3">
            <div className="flex items-center justify-between">
              <p className="text-xs font-medium uppercase tracking-wide text-tinta-suave">Variações (tamanhos, sabores, etc.)</p>
              {!mostrarFormVariacao && (
                <button onClick={() => abrirNovaVariacao(produto.id)} className="rounded-full bg-tinta px-3 py-1 text-xs font-semibold text-superficie">
                  + Adicionar
                </button>
              )}
            </div>

            {/* Lista de variações existentes */}
            {produto.variacoes && produto.variacoes.length > 0 ? (
              <ul className="space-y-2">
                {produto.variacoes.map((v) => (
                  <li key={v.id} className="flex items-center justify-between rounded-xl bg-fundo px-3 py-2">
                    <div className={!v.disponivel ? 'opacity-50' : ''}>
                      <p className="text-sm font-medium text-tinta">{v.nome}</p>
                      <p className="text-xs text-tinta-suave">
                        {v.modo_preco === 'absoluto'
                          ? `R$ ${v.preco_adicional.toFixed(2).replace('.', ',')} (preço próprio)`
                          : v.preco_adicional > 0 ? `+R$ ${v.preco_adicional.toFixed(2).replace('.', ',')}` : 'Sem adicional'}
                        {!v.mostrar_valor_adicional && ` · valor oculto no ${rotuloCatalogo(loja?.segmento_principal)}`}
                        {v.estoque_atual !== null && ` · ${v.estoque_atual === 0 ? 'Esgotada' : `${v.estoque_atual} em estoque`}`}
                      </p>
                    </div>
                    <div className="flex items-center gap-2">
                      <button onClick={() => abrirEdicaoVariacao(produto.id, v)} className="text-xs font-medium text-acento hover:underline">Editar</button>
                      <button onClick={() => { if (confirm(`Excluir "${v.nome}"?`)) mutDeletarVar.mutate({ pid: produto.id, vid: v.id }); }} className="text-xs text-tinta-suave hover:text-acento">Excluir</button>
                    </div>
                  </li>
                ))}
              </ul>
            ) : (
              <p className="text-xs text-tinta-suave">Nenhuma variação — produto funciona sem seleção de opção.</p>
            )}

            {/* Formulário de variação */}
            {mostrarFormVariacao && produtoExpandido === produto.id && (
              <form onSubmit={(e) => salvarVariacao(e, produto.id)} className="space-y-3 rounded-xl border border-tinta/15 p-3">
                <p className="text-xs font-medium text-tinta">{editandoVariacaoId ? 'Editar variação' : 'Nova variação'}</p>

                <VariacaoFormFields form={formVariacao} onChange={setFormVariacao} segmentoLoja={loja?.segmento_principal} />

                {editandoVariacaoId && formVariacao.modo_preco === 'absoluto' && (() => {
                  const variacaoAtual = produto.variacoes?.find((v) => v.id === editandoVariacaoId);
                  return (
                    <div className="space-y-2 rounded-lg border border-tinta/10 bg-superficie p-2">
                      <div className="flex items-center justify-between">
                        <p className="text-xs font-medium uppercase tracking-wide text-tinta-suave">Fotos da variação</p>
                        <label className="cursor-pointer rounded-full bg-tinta px-3 py-1 text-xs font-semibold text-superficie">
                          {enviandoFotoVariacao ? 'Enviando...' : '+ Adicionar'}
                          <input type="file" accept="image/*" className="hidden" disabled={enviandoFotoVariacao}
                            onChange={(e) => variacaoAtual && adicionarFotoDaVariacao(produto.id, variacaoAtual, e)} />
                        </label>
                      </div>
                      {variacaoAtual?.fotos && variacaoAtual.fotos.length > 0 ? (
                        <div className="flex flex-wrap gap-2">
                          {variacaoAtual.fotos.map((foto) => (
                            <div key={foto.id} className="relative group">
                              <img src={foto.url} alt={variacaoAtual.nome} className="h-14 w-14 rounded-lg object-cover" />
                              <button
                                onClick={() => removerFotoDaVariacao(produto.id, editandoVariacaoId, foto.id)}
                                className="absolute -right-1 -top-1 hidden h-5 w-5 items-center justify-center rounded-full bg-acento text-xs text-superficie group-hover:flex"
                              >×</button>
                            </div>
                          ))}
                        </div>
                      ) : (
                        <p className="text-xs text-tinta-suave">Nenhuma foto ainda.</p>
                      )}
                    </div>
                  );
                })()}

                {erroVariacao && <p className="text-xs text-acento">{erroVariacao}</p>}
                <div className="flex gap-2">
                  <button type="button" onClick={fecharFormVariacao} className="rounded-full border border-tinta/20 px-3 py-1 text-xs font-semibold text-tinta">Cancelar</button>
                  <button type="submit" disabled={mutCriarVar.isPending || mutAtualizarVar.isPending} className="rounded-full bg-acento px-3 py-1 text-xs font-semibold text-superficie disabled:opacity-60">Salvar</button>
                </div>
              </form>
            )}
          </div>
        )}
      </li>
    );
  }

  // Agrupa os produtos de uma categoria pela hierarquia Subcategoria →
  // Grupo (Fase 6: liberado pra qualquer segmento, não só "mercadoria").
  // Cada nível só aparece se a loja realmente cadastrou subcategorias/
  // grupos; produtos sem subcategoria/grupo continuam aparecendo soltos,
  // e uma categoria sem nenhuma subcategoria cadastrada degrada pra lista
  // plana normal — por isso não precisa de nenhum gate de segmento aqui.
  function renderProdutosDaCategoria(produtosDaCategoria: Produto[], categoriaId: number) {
    const subsDaCategoria = (subcategorias ?? []).filter((s) => s.categoria_id === categoriaId);
    if (subsDaCategoria.length === 0) {
      return <ul className="space-y-3">{produtosDaCategoria.map(renderProduto)}</ul>;
    }

    const semSubcategoria = produtosDaCategoria.filter((p) => p.subcategoria_id === null);

    return (
      <div className="space-y-4">
        {semSubcategoria.length > 0 && <ul className="space-y-3">{semSubcategoria.map(renderProduto)}</ul>}
        {subsDaCategoria.map((sub) => {
          const produtosDaSub = produtosDaCategoria.filter((p) => p.subcategoria_id === sub.id);
          if (produtosDaSub.length === 0) return null;

          const gruposDaSub = (gruposCor ?? []).filter((g) => g.subcategoria_id === sub.id);
          const semGrupo = produtosDaSub.filter((p) => p.grupo_cor_id === null);

          return (
            <div key={sub.id} className="space-y-2 border-l-2 border-tinta/10 pl-3">
              <h3 className="text-xs font-semibold uppercase tracking-wide text-tinta-suave">{sub.nome}</h3>
              {gruposDaSub.length === 0 ? (
                <ul className="space-y-3">{produtosDaSub.map(renderProduto)}</ul>
              ) : (
                <div className="space-y-3">
                  {semGrupo.length > 0 && <ul className="space-y-3">{semGrupo.map(renderProduto)}</ul>}
                  {gruposDaSub.map((grupo) => {
                    const produtosDoGrupo = produtosDaSub.filter((p) => p.grupo_cor_id === grupo.id);
                    if (produtosDoGrupo.length === 0) return null;
                    return (
                      <div key={grupo.id} className="space-y-2 border-l-2 border-tinta/10 pl-3">
                        <h4 className="text-xs font-medium text-acento">{grupo.nome}</h4>
                        <ul className="space-y-3">{produtosDoGrupo.map(renderProduto)}</ul>
                      </div>
                    );
                  })}
                </div>
              )}
            </div>
          );
        })}
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="font-display text-2xl tracking-wide text-tinta">Produtos</h1>
        {!mostrarForm && (
          <div className="flex items-center gap-2">
            {ehMercadoria && (
              <div className="flex items-center gap-1.5">
                <button onClick={() => setMostrarCadastroEmMassa(true)} className="btn-neu-secundario hover:border-acento hover:text-acento">
                  Cadastro em massa
                </button>
                <InfoTooltip texto='Cadastra vários produtos em sequência sem fechar a tela: você preenche um, salva, adiciona fotos e variações se quiser, e já parte pro próximo — sem abrir o formulário de novo a cada item. Bom pra catálogos grandes.' />
              </div>
            )}
            <button onClick={abrirNovo} className="btn-neu-primario">
              + Novo produto
            </button>
          </div>
        )}
      </div>

      {mostrarCadastroEmMassa && (
        <CadastroEmMassaDialog
          open={mostrarCadastroEmMassa}
          onOpenChange={setMostrarCadastroEmMassa}
          categorias={categorias}
          subcategorias={subcategorias}
          gruposCor={gruposCor}
          segmentoPadrao={loja?.segmento_principal ?? 'alimenticio'}
        />
      )}

      {mostrarForm && (
        <form onSubmit={salvar} className="space-y-4 rounded-2xl bg-superficie p-5 shadow-sm">
          <h2 className="font-display text-lg tracking-wide text-tinta">{editandoId ? 'Editar produto' : 'Novo produto'}</h2>

          <ProdutoFormFields form={form} onChange={setForm} categorias={categorias} subcategorias={subcategorias} gruposCor={gruposCor} enviandoFoto={enviandoFoto} onSelecionarFoto={selecionarFoto} segmentoLoja={loja?.segmento_principal} />

          {erro && <p className="text-sm text-acento">{erro}</p>}
          <div className="flex gap-3">
            <button type="button" onClick={fecharForm} className="btn-neu-secundario">Cancelar</button>
            <button type="submit" disabled={salvando} className="btn-neu-primario">{salvando ? 'Salvando...' : 'Salvar'}</button>
          </div>
        </form>
      )}

      {isLoading ? (
        <p className="text-tinta-suave">Carregando produtos...</p>
      ) : produtos && produtos.length > 0 ? (
        <div className="space-y-6">
          {categorias?.map((categoria) => {
            const produtosDaCategoria = produtos.filter((p) => p.categoria_id === categoria.id);
            if (produtosDaCategoria.length === 0) return null;
            return (
              <div key={categoria.id} className="space-y-3">
                <h2 className="font-display text-base tracking-wide text-tinta">{categoria.nome}</h2>
                {renderProdutosDaCategoria(produtosDaCategoria, categoria.id)}
              </div>
            );
          })}
        </div>
      ) : (
        <p className="text-tinta-suave">Nenhum produto cadastrado ainda.</p>
      )}

      {produtoFichaTecnica && (
        <FichaTecnicaModal produto={produtoFichaTecnica} onFechar={() => setProdutoFichaTecnica(null)} />
      )}
    </div>
  );
}
