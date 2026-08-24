import { useState, type FormEvent } from 'react';
import { Link } from 'react-router-dom';
import { solicitarResetSenhaAfiliado } from '../../api/afiliado';
import { Campo } from '../../components/Campo';

export function EsqueciSenhaAfiliado() {
  const [email, setEmail] = useState('');
  const [enviado, setEnviado] = useState(false);
  const [enviando, setEnviando] = useState(false);
  const [erro, setErro] = useState<string | null>(null);

  async function enviar(e: FormEvent) {
    e.preventDefault();
    setEnviando(true);
    setErro(null);
    try {
      await solicitarResetSenhaAfiliado(email);
      setEnviado(true);
    } catch {
      setErro('Não foi possível processar sua solicitação. Tente novamente.');
    } finally {
      setEnviando(false);
    }
  }

  if (enviado) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-fundo px-4">
        <div className="w-full max-w-sm space-y-4 rounded-2xl bg-superficie p-8 text-center shadow-sm">
          <h1 className="font-display text-2xl tracking-wide text-tinta">Verifique seu email</h1>
          <p className="text-sm text-tinta-suave">
            Se existir uma conta de afiliado com o email <strong>{email}</strong>, você vai
            receber um link para redefinir sua senha em instantes.
          </p>
          <Link to="/afiliado/login" className="inline-block font-medium text-acento">
            Voltar para o login
          </Link>
        </div>
      </div>
    );
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-fundo px-4">
      <form
        onSubmit={enviar}
        className="w-full max-w-sm space-y-5 rounded-2xl bg-superficie p-8 shadow-sm"
      >
        <div className="text-center">
          <h1 className="font-display text-2xl tracking-wide text-tinta">Esqueceu a senha?</h1>
          <p className="mt-1 text-sm text-tinta-suave">
            Digite o email da sua conta de afiliado.
          </p>
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

        {erro && <p className="text-sm text-acento">{erro}</p>}

        <button
          type="submit"
          disabled={enviando}
          className="w-full rounded-full bg-acento py-3 font-semibold text-texto-claro transition disabled:opacity-60"
        >
          {enviando ? 'Enviando...' : 'Enviar link de redefinição'}
        </button>

        <p className="text-center text-sm text-tinta-suave">
          <Link to="/afiliado/login" className="font-medium text-acento">
            Voltar para o login
          </Link>
        </p>
      </form>
    </div>
  );
}