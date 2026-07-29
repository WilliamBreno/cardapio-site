import { useState, type FormEvent } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import axios from 'axios';
import {
  listarCategorias,
  criarCategoria,
  atualizarCategoria,
  deletarCategoria,
  listarSubcategorias,
  criarSubcategoria,
  atualizarSubcategoria,
  deletarSubcategoria,
  listarGruposCor,
  criarGrupoCor,
  atualizarGrupoCor,
  deletarGrupoCor,
  buscarLoja,
} from '../../api/admin';
import type { Categoria } from '../../api/types';
import { Campo } from '../../components/Campo';
import { HierarquiaCategoria } from '../../components/admin/HierarquiaCategoria';

export function Categorias() {
  const queryClient = useQueryClient();

  const { data: categorias, isLoading } = useQuery({
    queryKey: ['categorias'],
    queryFn: listarCategorias,
  });
  const { data: loja } = useQuery({ queryKey: ['loja'], queryFn: buscarLoja });
  // Subcategoria/Grupo já foi restrito ao segmento "mercadoria" (Fase 3),
  // mas o William pediu pra liberar como ferramenta de organização pra
  // qualquer segmento (Fase 6) — o cardápio público do alimentício
  // continua no formato lista de sempre, isso aqui é só admin.
  const ehMercadoria = loja?.segmento_principal === 'mercadoria';

  const { data: subcategorias } = useQuery({
    queryKey: ['subcategorias'],
    queryFn: listarSubcategorias,
  });
  const { data: gruposCor } = useQuery({
    queryKey: ['grupos-cor'],
    queryFn: listarGruposCor,
  });

  const invalidarSubcategorias = () => queryClient.invalidateQueries({ queryKey: ['subcategorias'] });
  const invalidarGruposCor = () => queryClient.invalidateQueries({ queryKey: ['grupos-cor'] });

  const mutCriarSub = useMutation({
    mutationFn: ({ categoriaId, nome }: { categoriaId: number; nome: string }) => criarSubcategoria(categoriaId, nome),
    onSuccess: invalidarSubcategorias,
  });
  const mutAtualizarSub = useMutation({
    mutationFn: ({ id, nome }: { id: number; nome: string }) => atualizarSubcategoria(id, nome),
    onSuccess: invalidarSubcategorias,
  });
  const mutDeletarSub = useMutation({ mutationFn: deletarSubcategoria, onSuccess: invalidarSubcategorias });

  const mutCriarGrupo = useMutation({
    mutationFn: ({ subcategoriaId, nome }: { subcategoriaId: number; nome: string }) => criarGrupoCor(subcategoriaId, nome),
    onSuccess: invalidarGruposCor,
  });
  const mutAtualizarGrupo = useMutation({
    mutationFn: ({ id, nome }: { id: number; nome: string }) => atualizarGrupoCor(id, nome),
    onSuccess: invalidarGruposCor,
  });
  const mutDeletarGrupo = useMutation({ mutationFn: deletarGrupoCor, onSuccess: invalidarGruposCor });

  const [editandoId, setEditandoId] = useState<number | null>(null);
  const [mostrarForm, setMostrarForm] = useState(false);
  const [nome, setNome] = useState('');
  const [erro, setErro] = useState<string | null>(null);
  const [erroExclusao, setErroExclusao] = useState<string | null>(null);

  const invalidar = () => queryClient.invalidateQueries({ queryKey: ['categorias'] });

  const mutCriar = useMutation({
    mutationFn: criarCategoria,
    onSuccess: () => {
      invalidar();
      fecharForm();
    },
    onError: () =>
      setErro('Não foi possível criar — talvez já exista uma categoria com esse nome.'),
  });

  const mutAtualizar = useMutation({
    mutationFn: ({ id, nome }: { id: number; nome: string }) => atualizarCategoria(id, nome),
    onSuccess: () => {
      invalidar();
      fecharForm();
    },
    onError: () => setErro('Não foi possível salvar.'),
  });

  const mutDeletar = useMutation({
    mutationFn: deletarCategoria,
    onSuccess: () => {
      invalidar();
      setErroExclusao(null);
    },
    onError: (err) => {
      // O backend manda uma mensagem específica quando a categoria ainda
      // tem produto dentro — mostra ela de verdade, não um erro genérico.
      const mensagem =
        axios.isAxiosError(err) && err.response?.data?.erro
          ? err.response.data.erro
          : 'Não foi possível excluir.';
      setErroExclusao(mensagem);
    },
  });

  function abrirNovo() {
    setEditandoId(null);
    setNome('');
    setErro(null);
    setMostrarForm(true);
  }

  function abrirEdicao(categoria: Categoria) {
    setEditandoId(categoria.id);
    setNome(categoria.nome);
    setErro(null);
    setMostrarForm(true);
  }

  function fecharForm() {
    setMostrarForm(false);
    setEditandoId(null);
  }

  function salvar(e: FormEvent) {
    e.preventDefault();
    if (!nome.trim()) {
      setErro('Digita um nome.');
      return;
    }
    setErro(null);
    if (editandoId) {
      mutAtualizar.mutate({ id: editandoId, nome: nome.trim() });
    } else {
      mutCriar.mutate(nome.trim());
    }
  }

  const salvando = mutCriar.isPending || mutAtualizar.isPending;

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="font-display text-2xl tracking-wide text-tinta">Categorias</h1>
        {!mostrarForm && (
          <button
            onClick={abrirNovo}
            className="rounded-full bg-acento px-4 py-2 text-sm font-semibold text-superficie"
          >
            + Nova categoria
          </button>
        )}
      </div>

      <p className="text-sm text-tinta-suave">
        {ehMercadoria
          ? 'São as categorias que organizam seu catálogo (ex: Tênis, Camisetas) — o cliente navega por elas no catálogo público, e você pode detalhar cada uma em subcategorias e grupos logo abaixo.'
          : 'São as abas que o cliente vê no cardápio (ex: Salgados, Doces). Toda loja já nasce com essas duas — adiciona mais se quiser organizar diferente. Se quiser mais detalhe (ex: tamanhos, tipos), dá pra usar subcategorias e grupos logo abaixo — é só uma ferramenta de organização interna, não muda nada no que o cliente vê.'}
      </p>

      {mostrarForm && (
        <form onSubmit={salvar} className="space-y-4 rounded-2xl bg-superficie p-5 shadow-sm">
          <h2 className="font-display text-lg tracking-wide text-tinta">
            {editandoId ? 'Editar categoria' : 'Nova categoria'}
          </h2>

          <Campo label="Nome">
            <input
              required
              autoFocus
              value={nome}
              onChange={(e) => setNome(e.target.value)}
              placeholder="Ex: Bebidas"
              className="w-full rounded-lg border border-tinta/20 bg-fundo px-3 py-2 text-tinta outline-none focus:border-acento"
            />
          </Campo>

          {erro && <p className="text-sm text-acento">{erro}</p>}

          <div className="flex gap-3">
            <button
              type="button"
              onClick={fecharForm}
              className="rounded-full border border-tinta/20 px-4 py-2 text-sm font-semibold text-tinta"
            >
              Cancelar
            </button>
            <button
              type="submit"
              disabled={salvando}
              className="rounded-full bg-acento px-4 py-2 text-sm font-semibold text-superficie disabled:opacity-60"
            >
              {salvando ? 'Salvando...' : 'Salvar'}
            </button>
          </div>
        </form>
      )}

      {erroExclusao && (
        <p className="rounded-lg bg-acento/10 px-4 py-2 text-sm text-acento">{erroExclusao}</p>
      )}

      {isLoading ? (
        <p className="text-tinta-suave">Carregando categorias...</p>
      ) : categorias && categorias.length > 0 ? (
        <ul className="space-y-3">
          {categorias.map((categoria) => (
            <li key={categoria.id} className="rounded-2xl bg-superficie shadow-sm">
              <div className="flex items-center justify-between gap-3 p-4">
                <p className="font-medium text-tinta">{categoria.nome}</p>

                <div className="flex shrink-0 items-center gap-3">
                  <button
                    onClick={() => abrirEdicao(categoria)}
                    className="text-sm font-medium text-acento hover:underline"
                  >
                    Editar
                  </button>
                  <button
                    onClick={() => {
                      if (confirm(`Excluir "${categoria.nome}"?`)) mutDeletar.mutate(categoria.id);
                    }}
                    className="text-sm text-tinta-suave hover:text-acento"
                  >
                    Excluir
                  </button>
                </div>
              </div>

              <HierarquiaCategoria
                categoria={categoria}
                subcategorias={subcategorias ?? []}
                gruposCor={gruposCor ?? []}
                onCriarSub={(nome) => mutCriarSub.mutate({ categoriaId: categoria.id, nome })}
                onAtualizarSub={(id, nome) => mutAtualizarSub.mutate({ id, nome })}
                onDeletarSub={(id) => mutDeletarSub.mutate(id)}
                onCriarGrupo={(subcategoriaId, nome) => mutCriarGrupo.mutate({ subcategoriaId, nome })}
                onAtualizarGrupo={(id, nome) => mutAtualizarGrupo.mutate({ id, nome })}
                onDeletarGrupo={(id) => mutDeletarGrupo.mutate(id)}
              />
            </li>
          ))}
        </ul>
      ) : (
        <p className="text-tinta-suave">Nenhuma categoria ainda.</p>
      )}
    </div>
  );
}