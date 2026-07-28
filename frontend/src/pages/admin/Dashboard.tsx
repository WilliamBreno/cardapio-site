import { NavLink, Outlet, useNavigate } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import { buscarLoja } from '../../api/admin';
import { useAuthStore } from '../../store/authStore';
import { rotuloCatalogo } from '../../lib/utils';

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

  // Só aparece pra loja que ativou o recurso — pra maioria (lojas de
  // comida) esse link não faz sentido.
  const links = loja?.aceita_guardar_entregar
    ? [...linksBase.slice(0, 2), { to: '/admin/solicitacoes', label: 'Guardados' }, ...linksBase.slice(2)]
    : linksBase;

  function sair() {
    logout();
    navigate('/login');
  }

  return (
    <div className="min-h-screen bg-fundo">
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
        <button onClick={sair} className="text-sm font-medium text-tinta-suave hover:text-acento">
          Sair
        </button>
      </header>

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

      <main className="mx-auto max-w-3xl px-6 py-6">
        <Outlet />
      </main>
    </div>
  );
}