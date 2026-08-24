import { NavLink, Outlet, useNavigate } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import { Moon, Sun } from 'lucide-react';
import { buscarLoja, statusMercadoPago } from '../../api/admin';
import { useAuthStore } from '../../store/authStore';
import { useTemaAdminStore } from '../../store/temaAdminStore';
import { useLayoutAdminStore } from '../../store/layoutAdminStore';
import { rotuloCatalogo, rotuloCombo, cn } from '../../lib/utils';

const linksBase = [
  { to: '/admin', label: 'Início' },
  { to: '/admin/pedidos', label: 'Pedidos' },
  { to: '/admin/produtos', label: 'Produtos' },
  { to: '/admin/categorias', label: 'Categorias' },
  { to: '/admin/cupons', label: 'Cupons' },
  { to: '/admin/combos', label: 'Combos' },
  { to: '/admin/sugestao-inteligente', label: 'Sugestão Inteligente' },
  { to: '/admin/configuracoes', label: 'Configurações' },
  { to: '/admin/meu-plano', label: 'Meu Plano' },
];

export function Dashboard() {
  const navigate = useNavigate();
  const logout = useAuthStore((state) => state.logout);

  const { data: loja } = useQuery({ queryKey: ['loja'], queryFn: buscarLoja });
  const { data: mercadoPagoStatus } = useQuery({ queryKey: ['mercadopago-status'], queryFn: statusMercadoPago });
  const preferencia = useTemaAdminStore((state) => state.preferencia);
  const alternarTema = useTemaAdminStore((state) => state.alternar);
  const larguraCompleta = useLayoutAdminStore((state) => state.larguraCompleta);

  // Só aparece pra loja que ativou o recurso — pra maioria (lojas de
  // comida) esse link não faz sentido.
  const comGuardados = loja?.aceita_guardar_entregar
    ? [...linksBase.slice(0, 2), { to: '/admin/solicitacoes', label: 'Guardados' }, ...linksBase.slice(2)]
    : linksBase;

  // Estoque (Fase 8) — relatório é Pro/Scale, direto no menu só a partir
  // desses planos; quem é Start/Basic não vê o link (a tela em si também
  // bloqueia, isso aqui é só não mostrar caminho pra algo que vai barrar).
  const comEstoque =
    loja?.plano === 'pro' || loja?.plano === 'scale'
      ? (() => {
          const indiceProdutos = comGuardados.findIndex((l) => l.to === '/admin/produtos');
          const copia = [...comGuardados];
          copia.splice(indiceProdutos + 1, 0, { to: '/admin/estoque', label: 'Estoque' });
          return copia;
        })()
      : comGuardados;

  // Insumos (Fase 9.1) — ficha técnica + CMV automático, exclusivo do
  // Scale — mesmo espírito acima, entra logo depois de "Estoque".
  const comInsumos =
    loja?.plano === 'scale'
      ? (() => {
          const indiceEstoque = comEstoque.findIndex((l) => l.to === '/admin/estoque');
          const posicao = indiceEstoque === -1 ? comEstoque.findIndex((l) => l.to === '/admin/produtos') : indiceEstoque;
          const copia = [...comEstoque];
          copia.splice(posicao + 1, 0, { to: '/admin/insumos', label: 'Insumos' });
          return copia;
        })()
      : comEstoque;

  // "Combo" vira "Kit" pra loja "mercadoria" (ver rotuloCombo).
  const rotuloCombos = `${rotuloCombo(loja?.segmento_principal)}s`;
  const links = comInsumos.map((link) =>
    link.to === '/admin/combos' ? { ...link, label: rotuloCombos } : link
  );

  function sair() {
    logout();
    navigate('/login');
  }

  return (
    <div className={cn('min-h-screen bg-fundo', preferencia === 'escuro' && 'dark')}>
      <header className="flex items-center justify-between border-b border-tinta/10 bg-superficie px-6 py-4">
        <div>
          <p className="font-display text-lg tracking-wide text-tinta">
            {loja?.nome ?? 'Sua loja'}
          </p>
          {loja && (
            <a
              href={`/${loja.slug}`}
              target="_blank"
              rel="noreferrer"
              className="text-xs text-acento hover:underline"
            >
              Ver {rotuloCatalogo(loja?.segmento_principal)} público ↗
            </a>
          )}
        </div>
        <div className="flex items-center gap-3">
          <button
            onClick={alternarTema}
            aria-label={preferencia === 'escuro' ? 'Mudar pro modo claro' : 'Mudar pro modo escuro'}
            className="flex size-8 items-center justify-center rounded-full text-tinta-suave transition hover:bg-tinta/10 hover:text-tinta"
          >
            {preferencia === 'escuro' ? <Sun className="size-4" /> : <Moon className="size-4" />}
          </button>
          <button onClick={sair} className="text-sm font-medium text-tinta-suave hover:text-acento">
            Sair
          </button>
        </div>
      </header>

      {mercadoPagoStatus && !mercadoPagoStatus.mercadopago_conectado && (
        <NavLink
          to="/admin/configuracoes"
          className="block bg-acento px-6 py-2 text-center text-sm font-medium text-superficie hover:bg-acento/90"
        >
          ⚠️ Pagamento não configurado — seus clientes não conseguem finalizar pedidos até você conectar sua conta do Mercado Pago. Clica aqui pra resolver.
        </NavLink>
      )}

      <nav className="flex gap-1 overflow-x-auto border-b border-tinta/10 bg-superficie px-6">
        {links.map((link) => (
          <NavLink
            key={link.to}
            to={link.to}
            end={link.to === '/admin'}
            className={({ isActive }) =>
              `whitespace-nowrap border-b-2 px-3 py-3 text-sm font-medium transition ${
                isActive
                  ? 'border-acento text-acento'
                  : 'border-transparent text-tinta-suave hover:text-tinta'
              }`
            }
          >
            {link.label}
          </NavLink>
        ))}
      </nav>

      <main className={cn('mx-auto px-6 py-6', larguraCompleta ? 'max-w-none' : 'max-w-3xl')}>
        <Outlet />
      </main>
    </div>
  );
}