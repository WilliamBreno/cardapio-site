import type { ChangeEvent } from 'react';
import type { Categoria, Subcategoria, GrupoCor, TipoProduto } from '../../api/types';
import type { ProdutoInput } from '../../api/admin';
import { Campo } from '../Campo';
import { logoMiniatura } from '../../api/upload';
import { rotuloCatalogo } from '../../lib/utils';

interface Props {
  form: ProdutoInput;
  onChange: (form: ProdutoInput) => void;
  categorias?: Categoria[];
  subcategorias?: Subcategoria[];
  gruposCor?: GrupoCor[];
  enviandoFoto: boolean;
  onSelecionarFoto: (e: ChangeEvent<HTMLInputElement>) => void;
  // Segmento da LOJA (não do produto) — decide se o rótulo da página
  // pública é "cardápio" ou "catálogo" (ver Fase 3.4, CatalogoGrid).
  segmentoLoja?: TipoProduto;
}

// Campos do formulário de produto — usado tanto na edição/criação inline
// (Produtos.tsx) quanto no wizard de cadastro em massa, pra não duplicar
// essa lógica nos dois lugares.
export function ProdutoFormFields({ form, onChange, categorias, subcategorias, gruposCor, enviandoFoto, onSelecionarFoto, segmentoLoja }: Props) {
  // Subcategoria/Grupo (Fase 6: disponível pra qualquer segmento, não só
  // "mercadoria") formam uma cadeia — trocar a categoria ou a
  // subcategoria limpa o que vinha "embaixo" na hierarquia, pra nunca
  // ficar um produto com subcategoria/grupo de outra categoria. Só
  // aparecem se a categoria escolhida realmente tiver subcategoria
  // cadastrada — não depende do tipo do produto.
  const subcategoriasDaCategoria = subcategorias?.filter((s) => s.categoria_id === form.categoria_id) ?? [];
  const gruposCorDaSubcategoria = gruposCor?.filter((g) => g.subcategoria_id === form.subcategoria_id) ?? [];

  return (
    <>
      <Campo label="Nome">
        <input required value={form.nome} onChange={(e) => onChange({ ...form, nome: e.target.value })} className="w-full rounded-lg border border-tinta/20 bg-fundo px-3 py-2 text-tinta outline-none focus:border-acento" />
      </Campo>
      <Campo label="Descrição">
        <textarea value={form.descricao} onChange={(e) => onChange({ ...form, descricao: e.target.value })} rows={2} className="w-full rounded-lg border border-tinta/20 bg-fundo px-3 py-2 text-tinta outline-none focus:border-acento" />
      </Campo>
      <div className="flex gap-3">
        <Campo label="Preço (R$)" className="flex-1">
          <input type="number" step="0.01" min="0.01" required value={form.preco || ''} onChange={(e) => onChange({ ...form, preco: parseFloat(e.target.value) || 0 })} className="w-full rounded-lg border border-tinta/20 bg-fundo px-3 py-2 text-tinta outline-none focus:border-acento" />
        </Campo>
        <Campo label="Categoria" className="flex-1">
          <select required value={form.categoria_id || ''} onChange={(e) => onChange({ ...form, categoria_id: Number(e.target.value), subcategoria_id: null, grupo_cor_id: null })} className="w-full rounded-lg border border-tinta/20 bg-fundo px-3 py-2 text-tinta outline-none focus:border-acento">
            <option value="" disabled>Escolhe...</option>
            {categorias?.map((c) => <option key={c.id} value={c.id}>{c.nome}</option>)}
          </select>
        </Campo>
      </div>

      {subcategoriasDaCategoria.length > 0 && (
        <div className="flex gap-3">
          <Campo label="Subcategoria (opcional)" className="flex-1">
            <select
              value={form.subcategoria_id ?? ''}
              onChange={(e) => onChange({ ...form, subcategoria_id: e.target.value === '' ? null : Number(e.target.value), grupo_cor_id: null })}
              className="w-full rounded-lg border border-tinta/20 bg-fundo px-3 py-2 text-tinta outline-none focus:border-acento"
            >
              <option value="">Nenhuma</option>
              {subcategoriasDaCategoria.map((s) => <option key={s.id} value={s.id}>{s.nome}</option>)}
            </select>
          </Campo>
          {form.subcategoria_id !== null && gruposCorDaSubcategoria.length > 0 && (
            <Campo label="Grupo (opcional)" className="flex-1">
              <select
                value={form.grupo_cor_id ?? ''}
                onChange={(e) => onChange({ ...form, grupo_cor_id: e.target.value === '' ? null : Number(e.target.value) })}
                className="w-full rounded-lg border border-tinta/20 bg-fundo px-3 py-2 text-tinta outline-none focus:border-acento"
              >
                <option value="">Nenhum</option>
                {gruposCorDaSubcategoria.map((g) => <option key={g.id} value={g.id}>{g.nome}</option>)}
              </select>
            </Campo>
          )}
        </div>
      )}

      <div className="flex gap-3">
        <Campo label="Tipo de produto" className="flex-1">
          <select
            value={form.tipo_produto}
            onChange={(e) => onChange({ ...form, tipo_produto: e.target.value as 'alimenticio' | 'mercadoria', peso_gramas: e.target.value === 'alimenticio' ? null : form.peso_gramas })}
            className="w-full rounded-lg border border-tinta/20 bg-fundo px-3 py-2 text-tinta outline-none focus:border-acento"
          >
            <option value="alimenticio">Alimentício</option>
            <option value="mercadoria">Mercadoria (roupas, artesanato...)</option>
          </select>
        </Campo>
        {form.tipo_produto === 'mercadoria' && (
          <Campo label="Peso (g) — opcional" className="flex-1">
            <input type="number" min="1" value={form.peso_gramas ?? ''} onChange={(e) => onChange({ ...form, peso_gramas: e.target.value === '' ? null : parseInt(e.target.value) })} placeholder="Ex: 300" className="w-full rounded-lg border border-tinta/20 bg-fundo px-3 py-2 text-tinta outline-none focus:border-acento" />
          </Campo>
        )}
      </div>
      {form.tipo_produto === 'mercadoria' && (
        <p className="text-xs text-tinta-suave">
          Produtos "Mercadoria" podem ser guardados pelo cliente e entregues depois (se a loja tiver essa opção ativada em Configurações). Alimentícios nunca podem, por segurança.
          {' '}O peso não é obrigatório, mas só é usado pra calcular o frete quando a entrega for pra fora da sua região — sem ele, um pedido assim fica marcado como "peso pendente" até você completar.
        </p>
      )}

      {/* Upload de foto */}
      <div>
        <span className="mb-2 block text-xs font-medium uppercase tracking-wide text-tinta-suave">Foto (opcional)</span>
        <div className="flex items-center gap-4">
          <div className="flex h-16 w-16 shrink-0 items-center justify-center overflow-hidden rounded-full border-2 border-dashed border-tinta/25 bg-fundo">
            {form.foto_url ? <img src={logoMiniatura(form.foto_url)} alt="Foto" className="h-full w-full object-cover" /> : <span className="font-display text-xl text-tinta/30">{form.nome.charAt(0).toUpperCase() || '?'}</span>}
          </div>
          <label className="cursor-pointer rounded-full border border-tinta/20 px-4 py-2 text-sm font-semibold text-tinta hover:border-acento">
            {enviandoFoto ? 'Enviando...' : form.foto_url ? 'Trocar foto' : 'Enviar foto'}
            <input type="file" accept="image/*" onChange={onSelecionarFoto} disabled={enviandoFoto} className="hidden" />
          </label>
          {form.foto_url && !enviandoFoto && (
            <button
              type="button"
              onClick={() => onChange({ ...form, foto_url: '' })}
              className="rounded-full border border-tinta/20 px-4 py-2 text-sm font-semibold text-tinta-suave hover:border-acento hover:text-acento"
            >
              Remover foto
            </button>
          )}
        </div>
      </div>

      <label className="flex items-center gap-2 text-sm text-tinta">
        <input type="checkbox" checked={form.disponivel} onChange={(e) => onChange({ ...form, disponivel: e.target.checked })} className="h-4 w-4 accent-acento" />
        Disponível no {rotuloCatalogo(segmentoLoja)}
      </label>

      {/* Estoque */}
      <div className="space-y-3 rounded-xl border border-tinta/10 bg-fundo p-4">
        <p className="text-xs font-medium uppercase tracking-wide text-tinta-suave">Controle de estoque <span className="normal-case font-normal">(opcional)</span></p>
        <div className="flex gap-3">
          <Campo label="Quantidade em estoque" className="flex-1">
            <input type="number" min="0" value={form.estoque_atual ?? ''} onChange={(e) => onChange({ ...form, estoque_atual: e.target.value === '' ? null : parseInt(e.target.value) })} placeholder="Sem limite" className="w-full rounded-lg border border-tinta/20 bg-superficie px-3 py-2 text-tinta outline-none focus:border-acento" />
          </Campo>
          <Campo label="Alerta quando restar" className="flex-1">
            <input type="number" min="0" value={form.estoque_alerta ?? ''} onChange={(e) => onChange({ ...form, estoque_alerta: e.target.value === '' ? null : parseInt(e.target.value) })} placeholder="Não avisar" className="w-full rounded-lg border border-tinta/20 bg-superficie px-3 py-2 text-tinta outline-none focus:border-acento" />
          </Campo>
        </div>
        <p className="text-xs text-tinta-suave">Se o produto tem variações, o estoque é gerenciado por variação, não aqui.</p>
      </div>
    </>
  );
}
