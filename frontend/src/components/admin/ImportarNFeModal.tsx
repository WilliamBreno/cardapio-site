import { useRef, useState } from 'react';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import {
  listarInsumos, previewImportacaoNFe, confirmarImportacaoNFe,
  type PreviewNFe, type ConfirmarItemNFeInput,
} from '../../api/admin';

// Uma linha da tela de conferência — o admin decide, item a item, se
// vincula a um insumo já cadastrado, cria um novo, ou ignora a linha.
// Tudo em string porque são campos de formulário (evita o problema
// clássico de input number controlado apagando o "0" enquanto digita).
interface LinhaImportacao {
  nome: string;
  unidadeCompra: string;
  quantidade: string;
  valorUnitario: string;
  acao: 'vincular' | 'criar' | 'ignorar';
  insumoId: string;
  unidadeUso: string;
  fatorConversao: string;
}

// ImportarNFeModal implementa a Fase 9.2 (importação em massa via XML de
// NF-e — dado estruturado, sem OCR/IA). O XML nunca sai do navegador em
// arquivo bruto: é lido como texto (File.text()) e mandado como string
// dentro de um POST JSON normal, então não precisa de upload multipart
// no backend nem de nenhum serviço externo de armazenamento.
export function ImportarNFeModal({ onFechar }: { onFechar: () => void }) {
  const queryClient = useQueryClient();
  const { data: insumosExistentes } = useQuery({ queryKey: ['insumos'], queryFn: listarInsumos });
  const fileInputRef = useRef<HTMLInputElement>(null);

  const [preview, setPreview] = useState<PreviewNFe | null>(null);
  const [linhas, setLinhas] = useState<LinhaImportacao[]>([]);
  const [carregandoArquivo, setCarregandoArquivo] = useState(false);
  const [confirmando, setConfirmando] = useState(false);
  const [erro, setErro] = useState<string | null>(null);
  const [resultado, setResultado] = useState<string | null>(null);

  async function selecionarArquivo(e: React.ChangeEvent<HTMLInputElement>) {
    const arquivo = e.target.files?.[0];
    if (!arquivo) return;
    setErro(null);
    setResultado(null);
    setCarregandoArquivo(true);
    try {
      const xml = await arquivo.text();
      const dados = await previewImportacaoNFe(xml);
      setPreview(dados);
      setLinhas(dados.itens.map((item) => ({
        nome: item.nome,
        unidadeCompra: item.unidade,
        quantidade: String(item.quantidade),
        valorUnitario: String(item.valor_unitario),
        acao: item.insumo_sugerido ? 'vincular' : 'criar',
        insumoId: item.insumo_sugerido ? String(item.insumo_sugerido) : '',
        unidadeUso: item.unidade,
        fatorConversao: '1',
      })));
    } catch {
      setErro('Não foi possível ler esse arquivo como uma NF-e válida.');
    } finally {
      setCarregandoArquivo(false);
    }
  }

  function atualizarLinha(i: number, campo: keyof LinhaImportacao, valor: string) {
    setLinhas(linhas.map((linha, idx) => (idx === i ? { ...linha, [campo]: valor } : linha)));
  }

  async function confirmar() {
    const itens: ConfirmarItemNFeInput[] = [];
    for (const linha of linhas) {
      if (linha.acao === 'ignorar') { itens.push({ acao: 'ignorar', quantidade: 0, valor_unitario: 0 }); continue; }

      const quantidade = parseFloat(linha.quantidade);
      const valorUnitario = parseFloat(linha.valorUnitario);
      if (!quantidade || quantidade <= 0 || isNaN(valorUnitario) || valorUnitario < 0) {
        setErro(`Quantidade/valor inválido pra "${linha.nome}".`);
        return;
      }

      if (linha.acao === 'vincular') {
        if (!linha.insumoId) { setErro(`Escolhe um insumo pra vincular "${linha.nome}", ou muda pra "criar novo".`); return; }
        itens.push({ acao: 'vincular', insumo_id: Number(linha.insumoId), quantidade, valor_unitario: valorUnitario });
      } else {
        const fatorConversao = parseFloat(linha.fatorConversao);
        if (!linha.unidadeUso || !fatorConversao || fatorConversao <= 0) {
          setErro(`Preenche unidade de uso e fator de conversão pra "${linha.nome}".`);
          return;
        }
        itens.push({
          acao: 'criar', nome: linha.nome, unidade_compra: linha.unidadeCompra, unidade_uso: linha.unidadeUso,
          fator_conversao: fatorConversao, quantidade, valor_unitario: valorUnitario,
        });
      }
    }

    setErro(null);
    setConfirmando(true);
    try {
      const insumosAtualizados = await confirmarImportacaoNFe(preview?.numero_nota ?? '', itens);
      queryClient.invalidateQueries({ queryKey: ['insumos'] });
      setResultado(`${insumosAtualizados.length} insumo(s) atualizado(s)/criado(s) com sucesso.`);
      setPreview(null);
      setLinhas([]);
    } catch {
      setErro('Não foi possível confirmar a importação.');
    } finally {
      setConfirmando(false);
    }
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 px-4" onClick={onFechar}>
      <div
        className="max-h-[85vh] w-full max-w-2xl overflow-y-auto rounded-2xl bg-superficie p-5 shadow-lg"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="mb-4 flex items-start justify-between">
          <div>
            <h2 className="font-display text-lg tracking-wide text-tinta">Importar NF-e (XML)</h2>
            <p className="text-sm text-tinta-suave">
              Sobe o arquivo XML da nota de compra dos insumos — confere e ajusta cada linha antes
              de confirmar.
            </p>
          </div>
          <button onClick={onFechar} className="text-tinta-suave hover:text-tinta" aria-label="Fechar">✕</button>
        </div>

        {resultado ? (
          <div className="space-y-4">
            <p className="rounded-xl bg-douro/10 p-3 text-sm text-tinta">{resultado}</p>
            <button onClick={onFechar} className="rounded-full bg-acento px-4 py-2 text-sm font-semibold text-superficie">Fechar</button>
          </div>
        ) : !preview ? (
          <div className="space-y-3">
            <input
              ref={fileInputRef}
              type="file"
              accept=".xml"
              onChange={selecionarArquivo}
              disabled={carregandoArquivo}
              className="w-full rounded-lg border border-tinta/20 bg-fundo px-3 py-2 text-sm text-tinta outline-none file:mr-3 file:rounded-full file:border-0 file:bg-tinta file:px-3 file:py-1.5 file:text-xs file:font-semibold file:text-superficie"
            />
            {carregandoArquivo && <p className="text-sm text-tinta-suave">Lendo nota fiscal...</p>}
            {erro && <p className="text-sm text-acento">{erro}</p>}
          </div>
        ) : (
          <div className="space-y-4">
            <p className="text-xs text-tinta-suave">
              Nota nº {preview.numero_nota || '—'} · {preview.fornecedor || 'fornecedor não identificado'}
            </p>

            <div className="space-y-3">
              {linhas.map((linha, i) => (
                <div key={i} className="space-y-2 rounded-xl bg-fundo p-3">
                  <div className="flex items-center justify-between gap-2">
                    <p className="text-sm font-medium text-tinta">{linha.nome}</p>
                    <select
                      value={linha.acao}
                      onChange={(e) => atualizarLinha(i, 'acao', e.target.value)}
                      className="rounded-lg border border-tinta/20 bg-superficie px-2 py-1 text-xs text-tinta outline-none focus:border-acento"
                    >
                      <option value="vincular">Vincular a insumo existente</option>
                      <option value="criar">Criar novo insumo</option>
                      <option value="ignorar">Ignorar esta linha</option>
                    </select>
                  </div>

                  {linha.acao !== 'ignorar' && (
                    <div className="flex flex-wrap items-center gap-2">
                      <input
                        type="number" step="0.0001" min="0.0001" value={linha.quantidade}
                        onChange={(e) => atualizarLinha(i, 'quantidade', e.target.value)}
                        placeholder="quantidade"
                        className="w-24 rounded-lg border border-tinta/20 bg-superficie px-2 py-1 text-sm text-tinta outline-none focus:border-acento"
                      />
                      <span className="text-xs text-tinta-suave">{linha.unidadeCompra}</span>
                      <span className="text-xs text-tinta-suave">×</span>
                      <input
                        type="number" step="0.0001" min="0" value={linha.valorUnitario}
                        onChange={(e) => atualizarLinha(i, 'valorUnitario', e.target.value)}
                        placeholder="custo unit."
                        className="w-24 rounded-lg border border-tinta/20 bg-superficie px-2 py-1 text-sm text-tinta outline-none focus:border-acento"
                      />
                    </div>
                  )}

                  {linha.acao === 'vincular' && (
                    <select
                      value={linha.insumoId}
                      onChange={(e) => atualizarLinha(i, 'insumoId', e.target.value)}
                      className="w-full rounded-lg border border-tinta/20 bg-superficie px-2 py-1 text-sm text-tinta outline-none focus:border-acento"
                    >
                      <option value="">Selecione um insumo...</option>
                      {insumosExistentes?.map((ins) => (
                        <option key={ins.id} value={ins.id}>{ins.nome}</option>
                      ))}
                    </select>
                  )}

                  {linha.acao === 'criar' && (
                    <div className="flex flex-wrap items-center gap-2">
                      <span className="text-xs text-tinta-suave">Unidade de uso na receita:</span>
                      <input
                        value={linha.unidadeUso}
                        onChange={(e) => atualizarLinha(i, 'unidadeUso', e.target.value)}
                        placeholder="g, ml, unidade..."
                        className="w-24 rounded-lg border border-tinta/20 bg-superficie px-2 py-1 text-sm text-tinta outline-none focus:border-acento"
                      />
                      <span className="text-xs text-tinta-suave">1 {linha.unidadeCompra} =</span>
                      <input
                        type="number" step="0.01" min="0.01" value={linha.fatorConversao}
                        onChange={(e) => atualizarLinha(i, 'fatorConversao', e.target.value)}
                        className="w-20 rounded-lg border border-tinta/20 bg-superficie px-2 py-1 text-sm text-tinta outline-none focus:border-acento"
                      />
                      <span className="text-xs text-tinta-suave">{linha.unidadeUso || 'unidade de uso'}</span>
                    </div>
                  )}
                </div>
              ))}
            </div>

            {erro && <p className="text-sm text-acento">{erro}</p>}

            <div className="flex gap-3">
              <button type="button" onClick={onFechar} className="rounded-full border border-tinta/20 px-4 py-2 text-sm font-semibold text-tinta">Cancelar</button>
              <button onClick={confirmar} disabled={confirmando} className="rounded-full bg-acento px-4 py-2 text-sm font-semibold text-superficie disabled:opacity-60">
                {confirmando ? 'Confirmando...' : 'Confirmar importação'}
              </button>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
