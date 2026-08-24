import { useState, type ChangeEvent, type FormEvent } from 'react';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import {
  criarProduto, criarVariacao, adicionarFoto, deletarFoto, reordenarFotos,
  adicionarFotoVariacao, deletarFotoVariacao,
  type ProdutoInput, type VariacaoInput,
} from '../../api/admin';
import type { Categoria, Subcategoria, GrupoCor, Produto, TipoProduto, VariacaoProduto } from '../../api/types';
import { Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription, DialogFooter } from '../ui/dialog';
import { Button } from '../ui/button';
import { ProdutoFormFields } from './ProdutoFormFields';
import { VariacaoFormFields } from './VariacaoFormFields';
import { enviarImagem } from '../../api/upload';

interface Props {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  categorias?: Categoria[];
  subcategorias?: Subcategoria[];
  gruposCor?: GrupoCor[];
  segmentoPadrao: TipoProduto;
}

function produtoVazio(categorias: Categoria[] | undefined, segmento: TipoProduto): ProdutoInput {
  return {
    nome: '', descricao: '', preco: 0, foto_url: '', disponivel: true,
    categoria_id: categorias?.[0]?.id ?? 0, subcategoria_id: null, grupo_cor_id: null,
    estoque_atual: null, estoque_alerta: null,
    tipo_produto: segmento, peso_gramas: null,
  };
}

function variacaoVazia(segmento: TipoProduto): VariacaoInput {
  return {
    nome: '', preco_adicional: 0, disponivel: true, mostrar_valor_adicional: true,
    modo_preco: segmento === 'mercadoria' ? 'absoluto' : 'aditivo',
    estoque_atual: null, estoque_alerta: null, ordem: 0,
  };
}

// Wizard de cadastro em massa: cria um produto, deixa adicionar fotos e
// variações pra ele sem fechar, e permite emendar direto no próximo
// produto — pensado pra lojas de mercadoria com catálogos maiores, mas
// funciona pra qualquer segmento. Cada etapa é montada de novo (dialog é
// desmontado ao fechar, ver Produtos.tsx), então o estado interno nasce
// limpo a cada abertura.
export function CadastroEmMassaDialog({ open, onOpenChange, categorias, subcategorias, gruposCor, segmentoPadrao }: Props) {
  const queryClient = useQueryClient();
  const invalidar = () => queryClient.invalidateQueries({ queryKey: ['produtos'] });

  const [etapa, setEtapa] = useState<'produto' | 'variacoes'>('produto');
  const [form, setForm] = useState<ProdutoInput>(() => produtoVazio(categorias, segmentoPadrao));
  const [erro, setErro] = useState<string | null>(null);
  const [enviandoFoto, setEnviandoFoto] = useState(false);
  const [produtoCriado, setProdutoCriado] = useState<Produto | null>(null);

  // Todos os produtos já salvos nesta sessão do wizard — vira a lista de
  // conferência no rodapé do modal. Fica sincronizada com produtoCriado
  // sempre que ele muda (foto/variação adicionada), via atualizarProdutoCriado.
  const [produtosSalvos, setProdutosSalvos] = useState<Produto[]>([]);

  const [formVariacao, setFormVariacao] = useState<VariacaoInput>(() => variacaoVazia(segmentoPadrao));
  const [mostrarFormVariacao, setMostrarFormVariacao] = useState(false);
  const [erroVariacao, setErroVariacao] = useState<string | null>(null);
  const [enviandoFotoVariacao, setEnviandoFotoVariacao] = useState(false);
  const [enviandoFotoProduto, setEnviandoFotoProduto] = useState(false);

  // Aplica uma atualização tanto no produto "em edição" (produtoCriado)
  // quanto na entrada correspondente na lista de conferência — pra lista
  // nunca ficar mostrando uma foto/variação que já não existe mais.
  function atualizarProdutoCriado(atualizador: (p: Produto) => Produto) {
    if (!produtoCriado) return;
    const atualizado = atualizador(produtoCriado);
    setProdutoCriado(atualizado);
    setProdutosSalvos((lista) => lista.map((p) => (p.id === atualizado.id ? atualizado : p)));
  }

  const mutCriarProduto = useMutation({
    mutationFn: criarProduto,
    onSuccess: (produto) => {
      invalidar();
      setProdutoCriado(produto);
      setProdutosSalvos((lista) => [...lista, produto]);
      setEtapa('variacoes');
    },
    onError: () => setErro('Não foi possível salvar o produto.'),
  });

  const mutCriarVariacao = useMutation({
    mutationFn: ({ pid, input }: { pid: number; input: VariacaoInput }) => criarVariacao(pid, input),
    onSuccess: (variacao) => {
      atualizarProdutoCriado((atual) => ({ ...atual, variacoes: [...(atual.variacoes ?? []), variacao] }));
      setFormVariacao(variacaoVazia(segmentoPadrao));
      setMostrarFormVariacao(false);
      invalidar();
    },
    onError: () => setErroVariacao('Não foi possível salvar a variação.'),
  });

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

  function salvarProduto(e: FormEvent) {
    e.preventDefault();
    if (!form.nome.trim() || form.preco <= 0 || !form.categoria_id) { setErro('Preenche nome, preço e categoria.'); return; }
    setErro(null);
    mutCriarProduto.mutate(form);
  }

  function salvarVariacao(e: FormEvent) {
    e.preventDefault();
    if (!produtoCriado) return;
    if (!formVariacao.nome.trim()) { setErroVariacao('Digita um nome.'); return; }
    setErroVariacao(null);
    mutCriarVariacao.mutate({ pid: produtoCriado.id, input: formVariacao });
  }

  // Fotos do produto (galeria — múltiplas, com uma marcada como principal)
  async function adicionarFotoDoProduto(e: ChangeEvent<HTMLInputElement>) {
    if (!produtoCriado) return;
    const arquivo = e.target.files?.[0];
    if (!arquivo) return;
    setEnviandoFotoProduto(true);
    try {
      const url = await enviarImagem(arquivo);
      const ordem = produtoCriado.fotos?.length ?? 0;
      const foto = await adicionarFoto(produtoCriado.id, url, ordem);
      atualizarProdutoCriado((atual) => ({ ...atual, fotos: [...(atual.fotos ?? []), foto] }));
      invalidar();
    } finally {
      setEnviandoFotoProduto(false);
    }
  }

  async function removerFotoDoProduto(fotoId: number) {
    if (!produtoCriado) return;
    await deletarFoto(produtoCriado.id, fotoId);
    atualizarProdutoCriado((atual) => ({ ...atual, fotos: atual.fotos?.filter((f) => f.id !== fotoId) }));
    invalidar();
  }

  async function tornarFotoPrincipal(fotoId: number) {
    if (!produtoCriado?.fotos) return;
    const principal = produtoCriado.fotos.find((f) => f.id === fotoId);
    if (!principal) return;
    const outras = produtoCriado.fotos.filter((f) => f.id !== fotoId);
    await reordenarFotos(produtoCriado.id, [fotoId, ...outras.map((f) => f.id)]);
    atualizarProdutoCriado((atual) => ({
      ...atual,
      fotos: [{ ...principal, ordem: 0 }, ...outras.map((f, i) => ({ ...f, ordem: i + 1 }))],
    }));
    invalidar();
  }

  async function adicionarFotoDaVariacao(variacao: VariacaoProduto, e: ChangeEvent<HTMLInputElement>) {
    if (!produtoCriado) return;
    const arquivo = e.target.files?.[0];
    if (!arquivo) return;
    setEnviandoFotoVariacao(true);
    try {
      const url = await enviarImagem(arquivo);
      const ordem = variacao.fotos?.length ?? 0;
      const foto = await adicionarFotoVariacao(produtoCriado.id, variacao.id, url, ordem);
      atualizarProdutoCriado((atual) => ({
        ...atual,
        variacoes: atual.variacoes?.map((v) => v.id === variacao.id ? { ...v, fotos: [...(v.fotos ?? []), foto] } : v),
      }));
      invalidar();
    } finally {
      setEnviandoFotoVariacao(false);
    }
  }

  async function removerFotoDaVariacao(variacaoId: number, fotoId: number) {
    if (!produtoCriado) return;
    await deletarFotoVariacao(produtoCriado.id, variacaoId, fotoId);
    atualizarProdutoCriado((atual) => ({
      ...atual,
      variacoes: atual.variacoes?.map((v) => v.id === variacaoId ? { ...v, fotos: v.fotos?.filter((f) => f.id !== fotoId) } : v),
    }));
    invalidar();
  }

  function proximoProduto() {
    // Mantém o que costuma se repetir num lote (categoria, subcategoria,
    // grupo de cor, tipo e peso) e limpa o que quase sempre muda a cada
    // item (nome, descrição, preço, foto, estoque) — evita ter que
    // reconfigurar a categoria a cada produto novo e evita, ao mesmo
    // tempo, salvar um produto "duplicado" por esquecimento do nome.
    setForm((atual) => ({
      ...produtoVazio(categorias, segmentoPadrao),
      categoria_id: atual.categoria_id,
      subcategoria_id: atual.subcategoria_id,
      grupo_cor_id: atual.grupo_cor_id,
      tipo_produto: atual.tipo_produto,
      peso_gramas: atual.peso_gramas,
    }));
    setProdutoCriado(null);
    setErro(null);
    setMostrarFormVariacao(false);
    setEtapa('produto');
  }

  return (
    <Dialog open={open} onOpenChange={(v) => { if (!v) onOpenChange(false); }}>
      <DialogContent className="max-h-[85vh] overflow-y-auto bg-superficie text-tinta ring-tinta/10 sm:max-w-lg">
        <DialogHeader>
          <DialogTitle className="font-display tracking-wide text-tinta">
            Cadastro em massa{produtosSalvos.length > 0 && ` · ${produtosSalvos.length} produto${produtosSalvos.length > 1 ? 's' : ''} adicionado${produtosSalvos.length > 1 ? 's' : ''}`}
          </DialogTitle>
          <DialogDescription className="text-tinta-suave">
            {etapa === 'produto'
              ? 'Preenche os dados do produto. Depois de salvar dá pra adicionar fotos e variações antes de ir pro próximo.'
              : `Adicione fotos e variações de "${produtoCriado?.nome}" (opcional) ou siga pro próximo produto.`}
          </DialogDescription>
        </DialogHeader>

        {etapa === 'produto' ? (
          <form onSubmit={salvarProduto} className="space-y-4">
            <ProdutoFormFields form={form} onChange={setForm} categorias={categorias} subcategorias={subcategorias} gruposCor={gruposCor} enviandoFoto={enviandoFoto} onSelecionarFoto={selecionarFoto} segmentoLoja={segmentoPadrao} />
            {erro && <p className="text-sm text-acento">{erro}</p>}
            <DialogFooter className="border-tinta/10 bg-fundo">
              <Button type="button" variant="outline" className="border-tinta/20 bg-superficie text-tinta hover:bg-fundo" onClick={() => onOpenChange(false)}>Cancelar</Button>
              <Button type="submit" className="bg-acento text-texto-claro hover:bg-acento/90" disabled={mutCriarProduto.isPending}>
                {mutCriarProduto.isPending ? 'Salvando...' : 'Salvar e continuar'}
              </Button>
            </DialogFooter>
          </form>
        ) : (
          <div className="space-y-4">
            {/* Fotos do produto — galeria com várias fotos, uma marcada como principal */}
            <div className="space-y-2 rounded-xl border border-tinta/10 p-3">
              <div className="flex items-center justify-between">
                <p className="text-xs font-medium uppercase tracking-wide text-tinta-suave">Fotos do produto</p>
                <label className="cursor-pointer rounded-full bg-tinta px-3 py-1 text-xs font-semibold text-texto-claro">
                  {enviandoFotoProduto ? 'Enviando...' : '+ Adicionar'}
                  <input type="file" accept="image/*" className="hidden" disabled={enviandoFotoProduto}
                    onChange={adicionarFotoDoProduto} />
                </label>
              </div>
              {produtoCriado?.fotos && produtoCriado.fotos.length > 0 ? (
                <div className="flex flex-wrap gap-2">
                  {produtoCriado.fotos.map((foto, i) => (
                    <div key={foto.id} className="relative group">
                      <img
                        src={foto.url}
                        alt={`Foto ${i + 1}`}
                        onClick={() => i !== 0 && tornarFotoPrincipal(foto.id)}
                        title={i === 0 ? 'Foto principal' : 'Clique pra tornar principal'}
                        className={`h-16 w-16 rounded-xl object-cover ${i !== 0 ? 'cursor-pointer' : ''}`}
                      />
                      <button
                        type="button"
                        onClick={() => removerFotoDoProduto(foto.id)}
                        title="Remover foto"
                        className="absolute -right-1 -top-1 hidden h-5 w-5 items-center justify-center rounded-full bg-acento text-xs text-texto-claro group-hover:flex"
                      >×</button>
                      {i === 0 && (
                        <span className="pointer-events-none absolute bottom-0 left-0 right-0 rounded-b-xl bg-tinta/60 py-0.5 text-center text-[10px] text-texto-claro">
                          Principal
                        </span>
                      )}
                    </div>
                  ))}
                </div>
              ) : (
                <p className="text-xs text-tinta-suave">Nenhuma foto ainda. A primeira adicionada vira a principal.</p>
              )}
            </div>

            {produtoCriado?.variacoes && produtoCriado.variacoes.length > 0 && (
              <ul className="space-y-2">
                {produtoCriado.variacoes.map((v) => (
                  <li key={v.id} className="space-y-2 rounded-xl bg-fundo px-3 py-2">
                    <p className="text-sm">
                      <span className="font-medium text-tinta">{v.nome}</span>{' '}
                      <span className="text-tinta-suave">
                        {v.modo_preco === 'absoluto'
                          ? `R$ ${v.preco_adicional.toFixed(2).replace('.', ',')} (preço próprio)`
                          : v.preco_adicional > 0 ? `+R$ ${v.preco_adicional.toFixed(2).replace('.', ',')}` : 'sem adicional'}
                      </span>
                    </p>
                    {v.modo_preco === 'absoluto' && (
                      <div className="flex flex-wrap items-center gap-2">
                        {v.fotos?.map((foto) => (
                          <div key={foto.id} className="relative group">
                            <img src={foto.url} alt={v.nome} className="h-12 w-12 rounded-lg object-cover" />
                            <button
                              type="button"
                              onClick={() => removerFotoDaVariacao(v.id, foto.id)}
                              className="absolute -right-1 -top-1 hidden h-5 w-5 items-center justify-center rounded-full bg-acento text-xs text-texto-claro group-hover:flex"
                            >×</button>
                          </div>
                        ))}
                        <label className="cursor-pointer rounded-full border border-tinta/20 px-2.5 py-1 text-xs font-medium text-tinta-suave hover:border-acento">
                          {enviandoFotoVariacao ? 'Enviando...' : '+ Foto'}
                          <input type="file" accept="image/*" className="hidden" disabled={enviandoFotoVariacao}
                            onChange={(e) => adicionarFotoDaVariacao(v, e)} />
                        </label>
                      </div>
                    )}
                  </li>
                ))}
              </ul>
            )}

            {mostrarFormVariacao ? (
              <form onSubmit={salvarVariacao} className="space-y-3 rounded-xl border border-tinta/15 p-3">
                <VariacaoFormFields form={formVariacao} onChange={setFormVariacao} segmentoLoja={segmentoPadrao} />
                {erroVariacao && <p className="text-xs text-acento">{erroVariacao}</p>}
                <div className="flex gap-2">
                  <Button type="button" variant="outline" size="sm" className="border-tinta/20 bg-superficie text-tinta hover:bg-fundo" onClick={() => setMostrarFormVariacao(false)}>Cancelar</Button>
                  <Button type="submit" size="sm" className="bg-acento text-texto-claro hover:bg-acento/90" disabled={mutCriarVariacao.isPending}>
                    {mutCriarVariacao.isPending ? 'Salvando...' : 'Adicionar variação'}
                  </Button>
                </div>
              </form>
            ) : (
              <Button
                type="button"
                variant="outline"
                className="border-tinta/20 bg-superficie text-tinta hover:bg-fundo"
                onClick={() => { setFormVariacao(variacaoVazia(segmentoPadrao)); setMostrarFormVariacao(true); }}
              >
                + Adicionar variação
              </Button>
            )}

            <DialogFooter className="border-tinta/10 bg-fundo">
              <Button type="button" variant="secondary" className="bg-fundo text-tinta hover:bg-fundo/80" onClick={proximoProduto}>+ Adicionar outro produto</Button>
              <Button type="button" className="bg-acento text-texto-claro hover:bg-acento/90" onClick={() => onOpenChange(false)}>Concluir</Button>
            </DialogFooter>
          </div>
        )}

        {/* Lista de conferência: todos os produtos já salvos nesta sessão */}
        {produtosSalvos.length > 0 && (
          <div className="space-y-2 border-t border-tinta/10 pt-3">
            <p className="text-xs font-medium uppercase tracking-wide text-tinta-suave">
              Produtos adicionados nesta sessão ({produtosSalvos.length})
            </p>
            <ul className="max-h-40 space-y-1.5 overflow-y-auto">
              {produtosSalvos.map((p) => {
                const foto = p.fotos?.[0]?.url || p.foto_url;
                return (
                  <li key={p.id} className="flex items-center gap-2 rounded-lg bg-fundo px-2.5 py-1.5">
                    <div className="flex h-8 w-8 shrink-0 items-center justify-center overflow-hidden rounded-full bg-superficie">
                      {foto ? (
                        <img src={foto} alt={p.nome} className="h-full w-full object-cover" />
                      ) : (
                        <span className="text-xs text-tinta/30">{p.nome.charAt(0).toUpperCase()}</span>
                      )}
                    </div>
                    <div className="min-w-0 flex-1">
                      <p className="truncate text-xs font-medium text-tinta">{p.nome}</p>
                      <p className="text-[11px] text-tinta-suave">
                        R$ {p.preco.toFixed(2).replace('.', ',')}
                        {p.variacoes && p.variacoes.length > 0 && ` · ${p.variacoes.length} variação${p.variacoes.length > 1 ? 'ões' : ''}`}
                      </p>
                    </div>
                  </li>
                );
              })}
            </ul>
          </div>
        )}
      </DialogContent>
    </Dialog>
  );
}
