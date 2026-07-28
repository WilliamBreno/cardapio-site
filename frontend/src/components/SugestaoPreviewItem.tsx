interface Props {
  nome: string;
  preco: number;
  precoComDesconto: number;
  // Sem onAdicionar, o botão fica só decorativo — usado no admin como
  // exemplo estático de como a sugestão aparece pro cliente final (ver
  // SugestaoInteligente.tsx). No carrinho real (CarrinhoDrawer), sempre
  // vem preenchido e adiciona o produto de verdade.
  onAdicionar?: () => void;
}

// SugestaoPreviewItem é o mesmo componente visual usado tanto na seção
// "Quem pediu isso também levou" do carrinho real quanto na prévia
// estática da tela de admin — pra o lojista ver exatamente como o cliente
// final vai ver, sem duplicar o layout em dois lugares.
export function SugestaoPreviewItem({ nome, preco, precoComDesconto, onAdicionar }: Props) {
  const temDesconto = precoComDesconto < preco;

  return (
    <li className="flex items-center gap-3">
      <div className="flex-1">
        <p className="text-sm font-medium text-tinta">{nome}</p>
        <p className="font-carimbo text-xs text-tinta-suave">
          {temDesconto && (
            <span className="mr-1.5 line-through opacity-60">
              R$ {preco.toFixed(2).replace('.', ',')}
            </span>
          )}
          R$ {precoComDesconto.toFixed(2).replace('.', ',')}
        </p>
      </div>
      <button
        onClick={onAdicionar}
        disabled={!onAdicionar}
        className="rounded-full bg-acento px-3 py-1.5 text-xs font-semibold text-superficie disabled:opacity-70"
      >
        Adicionar
      </button>
    </li>
  );
}
