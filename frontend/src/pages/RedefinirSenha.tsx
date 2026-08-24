import { useState, type FormEvent } from 'react';
import { useSearchParams, useNavigate, Link } from 'react-router-dom';
import { redefinirSenha } from '../api/auth';
import { Campo } from '../components/Campo';

export function RedefinirSenha() {
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();
  const token = searchParams.get('token') || '';

  const [novaSenha, setNovaSenha] = useState('');
  const [confirmarSenha, setConfirmarSenha] = useState('');
  const [enviando, setEnviando] = useState(false);
  const [erro, setErro] = useState<string | null>(null);
  const [sucesso, setSucesso] = useState(false);

  async function redefinir(e: FormEvent) {
    e.preventDefault();
    setErro(null);

    if (novaSenha.length < 6) {
      setErro('A senha precisa ter pelo menos 6 caracteres.');
      return;
    }
    if (novaSenha !== confirmarSenha) {
      setErro('As senhas não coincidem.');
      return;
    }

    setEnviando(true);
    try {
      await redefinirSenha(token, novaSenha);
      setSucesso(true);
      setTimeout(() => navigate('/login'), 2500);
    } catch {
      setErro('Link inválido ou expirado. Solicite um novo link de redefinição.');
    } finally {
      setEnviando(false);
    }
  }

  if (!token) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-fundo px-4">
        <div className="w-full max-w-sm space-y-4 rounded-2xl bg-superficie p-8 text-center shadow-sm">
          <h1 className="font-display text-2xl tracking-wide text-tinta">Link inválido</h1>
          <p className="text-sm text-tinta-suave">Esse link de redefinição está incompleto.</p>
          <Link to="/esqueci-senha" className="inline-block font-medium text-acento">
            Solicitar novo link
          </Link>
        </div>
      </div>
    );
  }

  if (sucesso) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-fundo px-4">
        <div className="w-full max-w-sm space-y-2 rounded-2xl bg-superficie p-8 text-center shadow-sm">
          <h1 className="font-display text-2xl tracking-wide text-tinta">Senha redefinida!</h1>
          <p className="text-sm text-tinta-suave">Redirecionando para o login...</p>
        </div>
      </div>
    );
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-fundo px-4">
      <form
        onSubmit={redefinir}
        className="w-full max-w-sm space-y-5 rounded-2xl bg-superficie p-8 shadow-sm"
      >
        <h1 className="text-center font-display text-2xl tracking-wide text-tinta">
          Criar nova senha
        </h1>

        <Campo label="Nova senha">
          <input
            type="password"
            required
            minLength={6}
            value={novaSenha}
            onChange={(e) => setNovaSenha(e.target.value)}
            className="w-full rounded-lg border border-tinta/20 bg-fundo px-3 py-2 text-tinta outline-none focus:border-acento"
          />
        </Campo>

        <Campo label="Confirmar senha">
          <input
            type="password"
            required
            minLength={6}
            value={confirmarSenha}
            onChange={(e) => setConfirmarSenha(e.target.value)}
            className="w-full rounded-lg border border-tinta/20 bg-fundo px-3 py-2 text-tinta outline-none focus:border-acento"
          />
        </Campo>

        {erro && <p className="text-sm text-acento">{erro}</p>}

        <button
          type="submit"
          disabled={enviando}
          className="w-full rounded-full bg-acento py-3 font-semibold text-texto-claro transition disabled:opacity-60"
        >
          {enviando ? 'Salvando...' : 'Redefinir senha'}
        </button>
      </form>
    </div>
  );
}