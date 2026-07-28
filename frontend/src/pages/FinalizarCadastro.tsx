import { useEffect, useState } from 'react';
import { useSearchParams, useNavigate, Link } from 'react-router-dom';
import { verificarToken, verificarSessao } from '../api/planos';

const TENTATIVAS_MAX = 8; // ~16s de espera total, tempo do webhook processar
const INTERVALO_MS = 2000;

export function FinalizarCadastro() {
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();
  const [erro, setErro] = useState<string | null>(null);
  const [aviso, setAviso] = useState<string | null>(null);

  const token = searchParams.get('token');
  const sessionId = searchParams.get('session_id');

  useEffect(() => {
    let cancelado = false;

    async function irParaCadastro(email: string, plano: string, tokenFinal: string) {
      if (cancelado) return;
      navigate(
        `/cadastro?token_assinatura=${encodeURIComponent(tokenFinal)}&email=${encodeURIComponent(email)}&plano=${plano}`,
        { replace: true }
      );
    }

    async function viaToken(t: string, tentativa = 1) {
      try {
        const dados = await verificarToken(t);
        if ('pendente' in dados) {
          // Assinatura ainda não confirmada pelo webhook do Mercado Pago
          // (assíncrono) — tenta de novo por um tempo curto antes de
          // desistir, mesmo padrão já usado no fluxo por session_id.
          if (cancelado) return;
          if (tentativa < TENTATIVAS_MAX) {
            setTimeout(() => viaToken(t, tentativa + 1), INTERVALO_MS);
          } else {
            setErro('Confirmação demorou mais que o esperado. Verifique seu email — mandamos o link de finalização por lá também.');
          }
          return;
        }
        await irParaCadastro(dados.email, dados.plano, dados.token);
      } catch {
        if (!cancelado) setErro('Esse link já foi usado ou é inválido.');
      }
    }

    async function viaSessao(sid: string, tentativa = 1) {
      try {
        const dados = await verificarSessao(sid);
        await irParaCadastro(dados.email, dados.plano, dados.token);
      } catch {
        if (cancelado) return;
        if (tentativa < TENTATIVAS_MAX) {
          setTimeout(() => viaSessao(sid, tentativa + 1), INTERVALO_MS);
        } else {
          setErro('Confirmação demorou mais que o esperado. Verifique seu email — mandamos o link de finalização por lá também.');
        }
      }
    }

    if (token) {
      viaToken(token);
    } else if (sessionId) {
      viaSessao(sessionId);
    } else {
      // Pode acontecer de verdade com o checkout de assinatura do
      // Mercado Pago (diferente do antigo checkout da Stripe, o retorno
      // pro site não necessariamente carrega um identificador na URL) —
      // não é obrigatoriamente um link quebrado, o pagamento pode ter
      // sido aprovado mesmo assim. O email com o link de finalização é
      // sempre enviado pelo webhook, então é a fonte confiável aqui.
      setAviso('Se você acabou de pagar, confirmamos por email — o link de finalização de cadastro chega em instantes na sua caixa de entrada.');
    }

    return () => {
      cancelado = true;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [token, sessionId]);

  return (
    <div className="flex min-h-screen items-center justify-center bg-fundo px-4">
      <div className="w-full max-w-sm space-y-3 rounded-2xl bg-superficie p-8 text-center shadow-sm">
        {erro || aviso ? (
          <>
            <h1 className="font-display text-xl tracking-wide text-tinta">{erro ? 'Ops' : 'Quase lá'}</h1>
            <p className="text-sm text-tinta-suave">{erro ?? aviso}</p>
            <Link to="/" className="inline-block pt-2 text-sm font-medium text-acento">
              Voltar pra página de planos
            </Link>
          </>
        ) : (
          <>
            <h1 className="font-display text-xl tracking-wide text-tinta">Confirmando seu pagamento...</h1>
            <p className="text-sm text-tinta-suave">Isso leva só alguns segundos.</p>
          </>
        )}
      </div>
    </div>
  );
}