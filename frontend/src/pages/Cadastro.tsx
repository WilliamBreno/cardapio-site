import { useState, type FormEvent } from 'react';
import { Link, useNavigate, useSearchParams } from 'react-router-dom';
import { cadastrar } from '../api/auth';
import { useAuthStore } from '../store/authStore';
import { Campo } from '../components/Campo';
import { rotuloCatalogo } from '../lib/utils';
import { NOME_PLANO } from '../lib/planos';

export function Cadastro() {
  const navigate = useNavigate();
  const setToken = useAuthStore((state) => state.setToken);
  const [searchParams] = useSearchParams();

  const codigoAfiliado = searchParams.get('ref') || undefined;
  const tokenAssinatura = searchParams.get('token_assinatura') || undefined;
  const emailPreenchido = searchParams.get('email') || '';
  const planoConfirmado = searchParams.get('plano');
  // Vem de Planos.tsx quando o dono escolhe Start ou Basic (planos sem
  // mensalidade) — nenhum pagamento envolvido, por isso um parâmetro
  // separado de `plano` (que é só pro banner de pagamento confirmado).
  const planoDesejadoParam = searchParams.get('plano_desejado');
  const planoDesejado = planoDesejadoParam === 'basic' ? 'basic' : undefined;

  const [nome, setNome] = useState('');
  const [email, setEmail] = useState(emailPreenchido);
  const [senha, setSenha] = useState('');
  const [nomeLoja, setNomeLoja] = useState('');
  const [segmento, setSegmento] = useState<'alimenticio' | 'mercadoria'>('alimenticio');
  const [erro, setErro] = useState<string | null>(null);
  const [enviando, setEnviando] = useState(false);

  async function criarConta(e: FormEvent) {
    e.preventDefault();
    setEnviando(true);
    setErro(null);
    try {
      const { token } = await cadastrar({
        nome,
        email,
        senha,
        nome_loja: nomeLoja,
        segmento_principal: segmento,
        codigo_afiliado: codigoAfiliado,
        token_assinatura: tokenAssinatura,
        plano: planoDesejado,
      });
      setToken(token);
      navigate('/admin');
    } catch {
      setErro('Não foi possível criar a conta. Esse email já pode estar em uso.');
    } finally {
      setEnviando(false);
    }
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-fundo px-4 py-10">
      <form
        onSubmit={criarConta}
        className="w-full max-w-sm space-y-5 rounded-2xl bg-superficie p-8 shadow-sm"
      >
        <div className="text-center">
          <h1 className="font-display text-2xl tracking-wide text-tinta">Crie sua loja</h1>
          <p className="mt-1 text-sm text-tinta-suave">Seu {rotuloCatalogo(segmento)} online em poucos minutos</p>
        </div>

        {planoConfirmado && NOME_PLANO[planoConfirmado] && (
          <div className="rounded-lg bg-acento/10 px-3 py-2 text-center text-sm font-medium text-acento">
            ✓ Pagamento do plano {NOME_PLANO[planoConfirmado]} confirmado
          </div>
        )}

        <Campo label="Seu nome">
          <input
            required
            value={nome}
            onChange={(e) => setNome(e.target.value)}
            className="w-full rounded-lg border border-tinta/20 bg-fundo px-3 py-2 text-tinta outline-none focus:border-acento"
          />
        </Campo>

        <Campo label="Nome da loja">
          <input
            required
            value={nomeLoja}
            onChange={(e) => setNomeLoja(e.target.value)}
            placeholder="Padaria da Maria"
            className="w-full rounded-lg border border-tinta/20 bg-fundo px-3 py-2 text-tinta outline-none focus:border-acento"
          />
        </Campo>

        <Campo label="Email">
          <input
            type="email"
            required
            readOnly={!!tokenAssinatura}
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            className={`w-full rounded-lg border border-tinta/20 px-3 py-2 text-tinta outline-none focus:border-acento ${
              tokenAssinatura ? 'bg-fundo/50 text-tinta-suave' : 'bg-fundo'
            }`}
          />
          {tokenAssinatura && (
            <span className="mt-1 block text-xs text-tinta-suave">
              Email do pagamento — não pode ser alterado aqui.
            </span>
          )}
        </Campo>

        <Campo label="Senha">
          <input
            type="password"
            required
            minLength={6}
            value={senha}
            onChange={(e) => setSenha(e.target.value)}
            className="w-full rounded-lg border border-tinta/20 bg-fundo px-3 py-2 text-tinta outline-none focus:border-acento"
          />
        </Campo>

        <Campo label="O que sua loja vende principalmente?">
          <div className="grid grid-cols-2 gap-2">
            <button
              type="button"
              onClick={() => setSegmento('alimenticio')}
              className={`rounded-lg border px-3 py-2 text-sm font-medium transition ${
                segmento === 'alimenticio'
                  ? 'border-acento bg-acento/10 text-acento'
                  : 'border-tinta/20 text-tinta-suave hover:border-tinta/40'
              }`}
            >
              Comida e bebida
            </button>
            <button
              type="button"
              onClick={() => setSegmento('mercadoria')}
              className={`rounded-lg border px-3 py-2 text-sm font-medium transition ${
                segmento === 'mercadoria'
                  ? 'border-acento bg-acento/10 text-acento'
                  : 'border-tinta/20 text-tinta-suave hover:border-tinta/40'
              }`}
            >
              Outros produtos
            </button>
          </div>
        </Campo>

        {erro && <p className="text-sm text-acento">{erro}</p>}

        <button
          type="submit"
          disabled={enviando}
          className="w-full rounded-full bg-acento py-3 font-semibold text-superficie transition disabled:opacity-60"
        >
          {enviando ? 'Criando...' : 'Criar minha loja'}
        </button>

        <p className="text-center text-sm text-tinta-suave">
          Já tem conta?{' '}
          <Link to="/login" className="font-medium text-acento">
            Entrar
          </Link>
        </p>
      </form>
    </div>
  );
}