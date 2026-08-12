import { BrowserRouter, Routes, Route } from 'react-router-dom';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { EsqueciSenha } from './pages/EsqueciSenha';
import { RedefinirSenha } from './pages/RedefinirSenha';
import { Home } from './pages/Home';
import { Planos } from './pages/Planos';
import { CardapioPublico } from './pages/CardapioPublico';
import { RastrearPedido } from './pages/RastrearPedido';
import { Login } from './pages/Login';
import { Cadastro } from './pages/Cadastro';
import { FinalizarCadastro } from './pages/FinalizarCadastro';
import { RotaProtegida } from './components/RotaProtegida';
import { Dashboard } from './pages/admin/Dashboard';
import { Inicio } from './pages/admin/Inicio';
import { Produtos } from './pages/admin/Produtos';
import { Estoque } from './pages/admin/Estoque';
import { Categorias } from './pages/admin/Categorias';
import { Pedidos } from './pages/admin/Pedidos';
import { CompartilharLocalizacao } from './pages/admin/CompartilharLocalizacao';
import { Cupons } from './pages/admin/Cupons';
import { Combos } from './pages/admin/Combos';
import { SugestaoInteligente } from './pages/admin/SugestaoInteligente';
import { Configuracoes } from './pages/admin/Configuracoes';
import { MeuPlano } from './pages/admin/MeuPlano';
import { LoginAfiliado } from './pages/afiliado/Login';
import { DashboardAfiliado } from './pages/afiliado/Dashboard';
import { EsqueciSenhaAfiliado } from './pages/afiliado/EsqueciSenha';
import { RedefinirSenhaAfiliado } from './pages/afiliado/RedefinirSenha';
import { AfiliadoRotaProtegida } from './components/AfiliadoRotaProtegida';
import { DrenuxAfiliados } from './pages/drenux/Afiliados';

const queryClient = new QueryClient();

function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <BrowserRouter>
        <Routes>
          {/* "/" agora é a porta de entrada comercial: mostra os planos.
              O que antes era a home (entrar / cadastrar) virou "/inicio",
              separado de propósito pra manter cada URL com um papel claro. */}
          <Route path="/" element={<Planos />} />
          <Route path="/inicio" element={<Home />} />

          <Route path="/login" element={<Login />} />
          <Route path="/cadastro" element={<Cadastro />} />
          <Route path="/cadastro/finalizar" element={<FinalizarCadastro />} />
          <Route path="/esqueci-senha" element={<EsqueciSenha />} />
          <Route path="/redefinir-senha" element={<RedefinirSenha />} />

          <Route element={<RotaProtegida />}>
            <Route path="/admin" element={<Dashboard />}>
              <Route index element={<Inicio />} />
              <Route path="inicio" element={<Inicio />} />
              <Route path="produtos" element={<Produtos />} />
              <Route path="estoque" element={<Estoque />} />
              <Route path="categorias" element={<Categorias />} />
              <Route path="pedidos" element={<Pedidos />} />
              <Route path="pedidos/:id/localizacao" element={<CompartilharLocalizacao />} />
              <Route path="cupons" element={<Cupons />} />
              <Route path="combos" element={<Combos />} />
              <Route path="sugestao-inteligente" element={<SugestaoInteligente />} />
              <Route path="configuracoes" element={<Configuracoes />} />
              <Route path="meu-plano" element={<MeuPlano />} />
            </Route>
          </Route>

          {/* Painel do afiliado — autenticação própria, separada de loja */}
          <Route path="/afiliado/login" element={<LoginAfiliado />} />
          <Route path="/afiliado/esqueci-senha" element={<EsqueciSenhaAfiliado />} />
          <Route path="/afiliado/redefinir-senha" element={<RedefinirSenhaAfiliado />} />
          <Route element={<AfiliadoRotaProtegida />}>
            <Route path="/afiliado/dashboard" element={<DashboardAfiliado />} />
          </Route>

          {/* Interno — repasse de comissão de afiliado (Fase 5.5), sem
              login de staff próprio: protegido por secret compartilhado
              (ver DrenuxAfiliados/drenuxAdminStore). */}
          <Route path="/drenux/afiliados" element={<DrenuxAfiliados />} />

          {/* Precisa vir ANTES de "/:slug" — senão o React Router tentaria
              casar "/loja-x/pedido/5/rastrear" com a rota do cardápio,
              tratando "pedido" como se fosse um slug de loja. */}
          <Route path="/:slug/pedido/:id/rastrear" element={<RastrearPedido />} />

          <Route path="/:slug" element={<CardapioPublico />} />
        </Routes>
      </BrowserRouter>
    </QueryClientProvider>
  );
}

export default App;