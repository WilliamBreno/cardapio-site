import { create } from 'zustand';
import type { Combo, ItemCarrinho, ItemCarrinhoCombo, Produto, SelecaoComboItem, SugestaoCarrinhoItem, VariacaoProduto } from '../api/types';
import { precoItem } from '../lib/utils';

// Chave única do item = produto_id + variacao_id (ou só produto_id se sem variação)
function chaveItem(produtoId: number, variacaoId?: number): string {
  return variacaoId ? `${produtoId}-${variacaoId}` : `${produtoId}`;
}

interface CartState {
  itens: ItemCarrinho[];
  combos: ItemCarrinhoCombo[];
  adicionar: (produto: Produto, variacao?: VariacaoProduto) => void;
  // adicionarSugestao entra pelo mesmo caminho de "adicionar", mas marca o
  // item com o vínculo de Sugestão Inteligente que originou ele e trava o
  // preço com desconto já calculado pelo backend (MontarSugestoesCarrinho)
  // — o backend revalida tudo de novo no checkout, então isso é só pra
  // exibir o total certo no carrinho antes de confirmar.
  adicionarSugestao: (sugestao: SugestaoCarrinhoItem, produto: Produto) => void;
  remover: (produtoId: number, variacaoId?: number) => void;
  alterarQuantidade: (produtoId: number, quantidade: number, variacaoId?: number) => void;
  // Combo é tratado como uma linha própria no carrinho (chave = combo_id)
  // — pra manter o V1 simples, adicionar o mesmo combo de novo só soma
  // quantidade, mantendo as variações escolhidas na primeira vez.
  adicionarCombo: (combo: Combo, selecoes: SelecaoComboItem[]) => void;
  removerCombo: (comboId: number) => void;
  alterarQuantidadeCombo: (comboId: number, quantidade: number) => void;
  limpar: () => void;
  total: () => number;
}

function precoItemCarrinho(item: ItemCarrinho): number {
  return item.precoComDesconto ?? precoItem(item.produto, item.variacao);
}

export const useCartStore = create<CartState>((set, get) => ({
  itens: [],
  combos: [],

  adicionar: (produto, variacao) => {
    set((state) => {
      const chave = chaveItem(produto.id, variacao?.id);
      const existente = state.itens.find(
        (item) => chaveItem(item.produto.id, item.variacao?.id) === chave
      );
      if (existente) {
        return {
          itens: state.itens.map((item) =>
            chaveItem(item.produto.id, item.variacao?.id) === chave
              ? { ...item, quantidade: item.quantidade + 1 }
              : item
          ),
        };
      }
      return { itens: [...state.itens, { produto, variacao, quantidade: 1 }] };
    });
  },

  adicionarSugestao: (sugestao, produto) => {
    set((state) => {
      const chave = chaveItem(produto.id);
      const existente = state.itens.find(
        (item) => chaveItem(item.produto.id, item.variacao?.id) === chave
      );
      if (existente) {
        return {
          itens: state.itens.map((item) =>
            chaveItem(item.produto.id, item.variacao?.id) === chave
              ? { ...item, quantidade: item.quantidade + 1 }
              : item
          ),
        };
      }
      return {
        itens: [
          ...state.itens,
          {
            produto,
            quantidade: 1,
            sugestaoProdutoId: sugestao.sugestao_id,
            precoComDesconto: sugestao.preco_com_desconto,
          },
        ],
      };
    });
  },

  remover: (produtoId, variacaoId) => {
    const chave = chaveItem(produtoId, variacaoId);
    set((state) => ({
      itens: state.itens.filter(
        (item) => chaveItem(item.produto.id, item.variacao?.id) !== chave
      ),
    }));
  },

  alterarQuantidade: (produtoId, quantidade, variacaoId) => {
    const chave = chaveItem(produtoId, variacaoId);
    if (quantidade <= 0) {
      get().remover(produtoId, variacaoId);
      return;
    }
    set((state) => ({
      itens: state.itens.map((item) =>
        chaveItem(item.produto.id, item.variacao?.id) === chave
          ? { ...item, quantidade }
          : item
      ),
    }));
  },

  adicionarCombo: (combo, selecoes) => {
    set((state) => {
      const existente = state.combos.find((item) => item.combo.id === combo.id);
      if (existente) {
        return {
          combos: state.combos.map((item) =>
            item.combo.id === combo.id ? { ...item, quantidade: item.quantidade + 1 } : item
          ),
        };
      }
      return { combos: [...state.combos, { combo, quantidade: 1, selecoes }] };
    });
  },

  removerCombo: (comboId) => {
    set((state) => ({ combos: state.combos.filter((item) => item.combo.id !== comboId) }));
  },

  alterarQuantidadeCombo: (comboId, quantidade) => {
    if (quantidade <= 0) {
      get().removerCombo(comboId);
      return;
    }
    set((state) => ({
      combos: state.combos.map((item) =>
        item.combo.id === comboId ? { ...item, quantidade } : item
      ),
    }));
  },

  limpar: () => set({ itens: [], combos: [] }),

  total: () => {
    const totalItens = get().itens.reduce(
      (soma, item) => soma + precoItemCarrinho(item) * item.quantidade,
      0
    );
    const totalCombos = get().combos.reduce(
      (soma, item) => soma + item.combo.preco * item.quantidade,
      0
    );
    return totalItens + totalCombos;
  },
}));
