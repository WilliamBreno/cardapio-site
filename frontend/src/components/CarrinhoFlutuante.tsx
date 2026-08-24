import { useCartStore } from '../store/cartStore';

interface Props {
  onAbrir: () => void;
}

export function CarrinhoFlutuante({ onAbrir }: Props) {
  const itens = useCartStore((state) => state.itens);
  const combos = useCartStore((state) => state.combos);
  const total = useCartStore((state) => state.total());

  // Precisa somar itens E combos — só contar `itens` fazia o botão ficar
  // escondido (quantidadeTotal === 0) quando o carrinho tinha só um
  // combo/kit dentro, mesmo com o combo já lá. Só aparecia depois de
  // adicionar também um item avulso, quando então mostrava os dois juntos.
  const quantidadeTotal =
    itens.reduce((soma, item) => soma + item.quantidade, 0) +
    combos.reduce((soma, item) => soma + item.quantidade, 0);

  if (quantidadeTotal === 0) return null;

  return (
    <button
      onClick={onAbrir}
      className="fixed inset-x-4 bottom-4 z-20 flex items-center justify-between rounded-2xl bg-tinta px-5 py-4 text-texto-claro shadow-lg transition active:scale-[0.98] sm:inset-x-auto sm:right-6 sm:w-80"
    >
      <span className="flex items-center gap-2 text-sm font-medium">
        <span className="flex h-6 w-6 items-center justify-center rounded-full bg-douro font-carimbo text-xs font-semibold text-tinta">
          {quantidadeTotal}
        </span>
        Ver carrinho
      </span>
      <span className="font-carimbo text-sm font-semibold">
        R$ {total.toFixed(2).replace('.', ',')}
      </span>
    </button>
  );
}