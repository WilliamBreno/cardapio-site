import { useState, type FormEvent } from 'react';
import type { Categoria, Subcategoria, GrupoCor } from '../../api/types';
import { Campo } from '../Campo';

interface Props {
  categoria: Categoria;
  subcategorias: Subcategoria[];
  gruposCor: GrupoCor[];
  onCriarSub: (nome: string) => void;
  onAtualizarSub: (id: number, nome: string) => void;
  onDeletarSub: (id: number) => void;
  onCriarGrupo: (subcategoriaId: number, nome: string) => void;
  onAtualizarGrupo: (id: number, nome: string) => void;
  onDeletarGrupo: (id: number) => void;
}

// Gerencia a hierarquia Categoria → Subcategoria → Grupo de uma categoria
// específica — disponível pra qualquer segmento (Fase 6). Cada nível é
// opcional: o lojista pode deixar uma categoria sem nenhuma subcategoria,
// ou uma subcategoria sem nenhum grupo.
export function HierarquiaCategoria({
  categoria, subcategorias, gruposCor,
  onCriarSub, onAtualizarSub, onDeletarSub,
  onCriarGrupo, onAtualizarGrupo, onDeletarGrupo,
}: Props) {
  const [aberta, setAberta] = useState(false);
  const [mostrarFormSub, setMostrarFormSub] = useState(false);
  const [editandoSubId, setEditandoSubId] = useState<number | null>(null);
  const [nomeSub, setNomeSub] = useState('');

  const [subExpandida, setSubExpandida] = useState<number | null>(null);
  const [mostrarFormGrupo, setMostrarFormGrupo] = useState(false);
  const [editandoGrupoId, setEditandoGrupoId] = useState<number | null>(null);
  const [nomeGrupo, setNomeGrupo] = useState('');

  const subsDaCategoria = subcategorias.filter((s) => s.categoria_id === categoria.id);

  function abrirNovaSub() {
    setEditandoSubId(null);
    setNomeSub('');
    setMostrarFormSub(true);
  }

  function abrirEdicaoSub(sub: Subcategoria) {
    setEditandoSubId(sub.id);
    setNomeSub(sub.nome);
    setMostrarFormSub(true);
  }

  function salvarSub(e: FormEvent) {
    e.preventDefault();
    if (!nomeSub.trim()) return;
    if (editandoSubId) onAtualizarSub(editandoSubId, nomeSub.trim());
    else onCriarSub(nomeSub.trim());
    setMostrarFormSub(false);
    setEditandoSubId(null);
  }

  function abrirNovoGrupo(subId: number) {
    setSubExpandida(subId);
    setEditandoGrupoId(null);
    setNomeGrupo('');
    setMostrarFormGrupo(true);
  }

  function abrirEdicaoGrupo(subId: number, grupo: GrupoCor) {
    setSubExpandida(subId);
    setEditandoGrupoId(grupo.id);
    setNomeGrupo(grupo.nome);
    setMostrarFormGrupo(true);
  }

  function salvarGrupo(e: FormEvent, subId: number) {
    e.preventDefault();
    if (!nomeGrupo.trim()) return;
    if (editandoGrupoId) onAtualizarGrupo(editandoGrupoId, nomeGrupo.trim());
    else onCriarGrupo(subId, nomeGrupo.trim());
    setMostrarFormGrupo(false);
    setEditandoGrupoId(null);
  }

  return (
    <div className="border-t border-tinta/10 px-4 pb-4 pt-3">
      <button
        onClick={() => setAberta(!aberta)}
        className="text-xs font-medium uppercase tracking-wide text-tinta-suave hover:text-acento"
      >
        {aberta ? '▾' : '▸'} Subcategorias {subsDaCategoria.length > 0 && `(${subsDaCategoria.length})`}
      </button>

      {aberta && (
        <div className="mt-3 space-y-3">
          {subsDaCategoria.length > 0 ? (
            <ul className="space-y-2">
              {subsDaCategoria.map((sub) => {
                const gruposDaSub = gruposCor.filter((g) => g.subcategoria_id === sub.id);
                return (
                  <li key={sub.id} className="rounded-xl bg-fundo px-3 py-2">
                    <div className="flex items-center justify-between gap-2">
                      <p className="text-sm font-medium text-tinta">{sub.nome}</p>
                      <div className="flex items-center gap-2">
                        <button
                          onClick={() => setSubExpandida(subExpandida === sub.id ? null : sub.id)}
                          className="text-xs font-medium text-tinta-suave hover:text-acento"
                        >
                          Grupos {gruposDaSub.length > 0 && `(${gruposDaSub.length})`}
                        </button>
                        <button onClick={() => abrirEdicaoSub(sub)} className="text-xs font-medium text-acento hover:underline">Editar</button>
                        <button
                          onClick={() => { if (confirm(`Excluir "${sub.nome}"?`)) onDeletarSub(sub.id); }}
                          className="text-xs text-tinta-suave hover:text-acento"
                        >
                          Excluir
                        </button>
                      </div>
                    </div>

                    {subExpandida === sub.id && (
                      <div className="mt-2 space-y-2 border-t border-tinta/10 pt-2">
                        {gruposDaSub.length > 0 ? (
                          <ul className="space-y-1">
                            {gruposDaSub.map((grupo) => (
                              <li key={grupo.id} className="flex items-center justify-between gap-2 rounded-lg bg-superficie px-2 py-1">
                                <span className="text-xs text-tinta">{grupo.nome}</span>
                                <div className="flex items-center gap-2">
                                  <button onClick={() => abrirEdicaoGrupo(sub.id, grupo)} className="text-xs font-medium text-acento hover:underline">Editar</button>
                                  <button
                                    onClick={() => { if (confirm(`Excluir "${grupo.nome}"?`)) onDeletarGrupo(grupo.id); }}
                                    className="text-xs text-tinta-suave hover:text-acento"
                                  >
                                    Excluir
                                  </button>
                                </div>
                              </li>
                            ))}
                          </ul>
                        ) : (
                          <p className="text-xs text-tinta-suave">Nenhum grupo ainda.</p>
                        )}

                        {mostrarFormGrupo && subExpandida === sub.id ? (
                          <form onSubmit={(e) => salvarGrupo(e, sub.id)} className="flex items-center gap-2">
                            <input
                              autoFocus
                              required
                              value={nomeGrupo}
                              onChange={(e) => setNomeGrupo(e.target.value)}
                              placeholder="Ex: Tons escuros"
                              className="flex-1 rounded-lg border border-tinta/20 bg-fundo px-2 py-1 text-xs text-tinta outline-none focus:border-acento"
                            />
                            <button type="submit" className="rounded-full bg-acento px-2 py-1 text-xs font-semibold text-superficie">Salvar</button>
                            <button type="button" onClick={() => setMostrarFormGrupo(false)} className="rounded-full border border-tinta/20 px-2 py-1 text-xs text-tinta">Cancelar</button>
                          </form>
                        ) : (
                          <button onClick={() => abrirNovoGrupo(sub.id)} className="text-xs font-semibold text-acento hover:underline">
                            + Novo grupo
                          </button>
                        )}
                      </div>
                    )}
                  </li>
                );
              })}
            </ul>
          ) : (
            <p className="text-xs text-tinta-suave">Nenhuma subcategoria ainda — organize por tamanho, por exemplo.</p>
          )}

          {mostrarFormSub ? (
            <form onSubmit={salvarSub} className="flex items-center gap-2">
              <Campo label="" className="flex-1">
                <input
                  autoFocus
                  required
                  value={nomeSub}
                  onChange={(e) => setNomeSub(e.target.value)}
                  placeholder="Ex: 42"
                  className="w-full rounded-lg border border-tinta/20 bg-fundo px-2 py-1 text-xs text-tinta outline-none focus:border-acento"
                />
              </Campo>
              <button type="submit" className="rounded-full bg-acento px-3 py-1 text-xs font-semibold text-superficie">Salvar</button>
              <button type="button" onClick={() => setMostrarFormSub(false)} className="rounded-full border border-tinta/20 px-3 py-1 text-xs text-tinta">Cancelar</button>
            </form>
          ) : (
            <button onClick={abrirNovaSub} className="text-xs font-semibold text-acento hover:underline">
              + Nova subcategoria
            </button>
          )}
        </div>
      )}
    </div>
  );
}
