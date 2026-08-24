import { useState, type FormEvent } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { loginAfiliado } from '../../api/afiliado';
import { useAfiliadoAuthStore } from '../../store/afiliadoAuthStore';
import { Campo } from '../../components/Campo';

export function LoginAfiliado() {
  const navigate = useNavigate();
  const setToken = useAfiliadoAuthStore((state) => state.setToken);

  const [email, setEmail] = useState('');
  const [senha, setSenha] = useState('');
  const [erro, setErro] = useState<string | null>(null);
  const [enviando, setEnviando] = useState(false);

  async function entrar(e: FormEvent) {
    e.preventDefault();
    setEnviando(true);
    setErro(null);
    try {
      const { token } = await loginAfiliado({ email, senha });
      setToken(token);
      navigate('/afiliado/dashboard');
    } catch {
      setErro('Email ou senha inválidos.');
    } finally {
      setEnviando(false);
    }
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-fundo px-4">
      <form
        onSubmit={entrar}
        className="w-full max-w-sm space-y-5 rounded-2xl bg-superficie p-8 shadow-sm"
      >
        <div className="text-center">
          <h1 className="font-display text-2xl tracking-wide text-tinta">Painel do Afiliado</h1>
          <p className="mt-1 text-sm text-tinta-suave">Drenux</p>
        </div>

        <Campo label="Email">
          <input
            type="email"
            required
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            className="w-full rounded-lg border border-tinta/20 bg-fundo px-3 py-2 text-tinta outline-none focus:border-acento"
          />
        </Campo>

        <Campo label="Senha">
          <input
            type="password"
            required
            value={senha}
            onChange={(e) => setSenha(e.target.value)}
            className="w-full rounded-lg border border-tinta/20 bg-fundo px-3 py-2 text-tinta outline-none focus:border-acento"
          />
        </Campo>

        <div className="text-right">
          <Link to="/afiliado/esqueci-senha" className="text-sm text-tinta-suave hover:text-acento">
            Esqueceu a senha?
          </Link>
        </div>

        {erro && <p className="text-sm text-acento">{erro}</p>}

        <button
          type="submit"
          disabled={enviando}
          className="w-full rounded-full bg-acento py-3 font-semibold text-texto-claro transition disabled:opacity-60"
        >
          {enviando ? 'Entrando...' : 'Entrar'}
        </button>
      </form>
    </div>
  );
}