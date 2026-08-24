import { Link } from 'react-router-dom';

export function Home() {
  return (
    <div className="flex min-h-screen flex-col items-center justify-center bg-fundo px-6 text-center">
      <p className="font-carimbo text-xs uppercase tracking-[0.2em] text-tinta-suave">
        Drenux
      </p>
      <h1 className="mt-2 max-w-md font-display text-4xl tracking-wide text-tinta">
        Sua loja online, do pedido ao pagamento
      </h1>
      <p className="mt-4 max-w-sm text-tinta-suave">
        Crie sua loja, monte sua estrutura e receba pedidos com pagamento de verdade — sem
        complicação.
      </p>

      <div className="mt-8 flex flex-col gap-3 sm:flex-row">
        <Link
          to="/cadastro"
          className="rounded-full bg-acento px-6 py-3 font-semibold text-texto-claro transition hover:bg-acento/90"
        >
          Criar minha loja
        </Link>
        <Link
          to="/login"
          className="rounded-full border-2 border-tinta/20 px-6 py-3 font-semibold text-tinta transition hover:border-acento"
        >
          Já tenho loja
        </Link>
      </div>
    </div>
  );
}