import { useState, useEffect, useRef } from 'react';
import { useParams, useNavigate, useLocation } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import {
  atualizarStatusEntrega, atualizarLocalizacao,
  atualizarStatusEntregaSolicitacao, atualizarLocalizacaoSolicitacao,
} from '../../api/rastreamento';
import { buscarLoja } from '../../api/admin';

const INTERVALO_MS = 25_000; // 25 segundos, conforme definido

export function CompartilharLocalizacao() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const location = useLocation();
  const pedidoId = Number(id);

  // Rastreamento em tempo real é Pro/Scale (Fase 7.4) — sem isso, a tela
  // ainda deixa marcar "saiu para entrega"/"entregue" (isso não é
  // exclusivo de plano), só não tenta compartilhar GPS, que o backend ia
  // recusar de qualquer forma.
  const { data: loja } = useQuery({ queryKey: ['loja'], queryFn: buscarLoja });
  const temRastreamento = loja?.plano === 'pro' || loja?.plano === 'scale';

  // Mesma tela serve tanto pra entrega de um pedido normal quanto pra
  // entrega de itens guardados (SolicitacaoEntrega) — os dois têm os
  // mesmos campos de rastreamento, só muda qual endpoint chamar.
  const ehSolicitacao = location.pathname.includes('/solicitacoes/');
  const atualizarStatus = ehSolicitacao ? atualizarStatusEntregaSolicitacao : atualizarStatusEntrega;
  const enviarLocalizacao = ehSolicitacao ? atualizarLocalizacaoSolicitacao : atualizarLocalizacao;
  const rotaVoltar = ehSolicitacao ? '/admin/solicitacoes' : '/admin/pedidos';

  const [compartilhando, setCompartilhando] = useState(false);
  const [ultimaAtualizacao, setUltimaAtualizacao] = useState<Date | null>(null);
  const [erro, setErro] = useState<string | null>(null);
  const [finalizando, setFinalizando] = useState(false);
  const watchIdRef = useRef<number | null>(null);
  const intervalRef = useRef<ReturnType<typeof setInterval> | null>(null);
  const posicaoAtualRef = useRef<{ lat: number; lng: number } | null>(null);

  useEffect(() => {
    return () => {
      // Limpa tudo se a pessoa sair da página sem clicar em "Parar" —
      // evita continuar mandando localização em segundo plano à toa.
      if (watchIdRef.current !== null) navigator.geolocation.clearWatch(watchIdRef.current);
      if (intervalRef.current) clearInterval(intervalRef.current);
    };
  }, []);

  async function iniciarCompartilhamento() {
    if (temRastreamento && !navigator.geolocation) {
      setErro('Esse navegador não suporta geolocalização.');
      return;
    }
    setErro(null);

    try {
      await atualizarStatus(pedidoId, 'saiu_para_entrega');
    } catch {
      setErro('Não foi possível marcar o pedido como "saiu para entrega". Tenta de novo.');
      return;
    }

    if (!temRastreamento) {
      // Plano sem rastreamento em tempo real (Fase 7.4) — só marca o
      // status, sem tentar compartilhar GPS (o backend recusaria mesmo).
      setCompartilhando(true);
      return;
    }

    // watchPosition mantém a posição sempre atualizada localmente; o
    // envio pro backend acontece separado, a cada 25s, pra não sobrecarregar
    // com uma requisição a cada pequeno movimento do GPS.
    watchIdRef.current = navigator.geolocation.watchPosition(
      (posicao) => {
        posicaoAtualRef.current = {
          lat: posicao.coords.latitude,
          lng: posicao.coords.longitude,
        };
      },
      () => setErro('Não conseguimos acessar sua localização. Verifica se a permissão foi concedida.'),
      { enableHighAccuracy: true, maximumAge: 10_000 }
    );

    async function enviar() {
      if (!posicaoAtualRef.current) return;
      try {
        await enviarLocalizacao(pedidoId, posicaoAtualRef.current.lat, posicaoAtualRef.current.lng);
        setUltimaAtualizacao(new Date());
      } catch {
        // Falha silenciosa num ciclo só — tenta de novo no próximo, sem
        // travar a experiência de quem está dirigindo.
      }
    }

    enviar(); // primeira vez já dispara, sem esperar os 25s iniciais
    intervalRef.current = setInterval(enviar, INTERVALO_MS);
    setCompartilhando(true);
  }

  function pararCompartilhamento() {
    if (watchIdRef.current !== null) navigator.geolocation.clearWatch(watchIdRef.current);
    if (intervalRef.current) clearInterval(intervalRef.current);
    setCompartilhando(false);
  }

  async function marcarEntregue() {
    setFinalizando(true);
    try {
      await atualizarStatus(pedidoId, 'entregue');
      pararCompartilhamento();
      navigate(rotaVoltar);
    } catch {
      setErro('Não foi possível marcar como entregue. Tenta de novo.');
      setFinalizando(false);
    }
  }

  return (
    <div className="mx-auto max-w-md space-y-6">
      <h1 className="font-display text-2xl tracking-wide text-tinta">
        {ehSolicitacao ? 'Entrega de itens guardados' : 'Entrega do pedido'} #{pedidoId}
      </h1>

      <div className="rounded-2xl bg-superficie p-5 shadow-sm">
        {!compartilhando ? (
          <>
            <p className="text-sm text-tinta-suave">
              {temRastreamento
                ? 'Ao iniciar, o cliente vai poder acompanhar sua localização em tempo real até você marcar a entrega como concluída. Mantenha essa aba aberta enquanto estiver a caminho.'
                : 'Isso marca o pedido como "saiu para entrega" e avisa o cliente. Rastreamento em tempo real (mapa ao vivo) é um recurso do plano Pro.'}
            </p>
            <button
              onClick={iniciarCompartilhamento}
              className="mt-4 w-full rounded-full bg-acento py-3 font-semibold text-superficie"
            >
              {temRastreamento ? '🛵 Iniciar entrega e compartilhar localização' : '🛵 Marcar como saiu para entrega'}
            </button>
          </>
        ) : (
          <>
            {temRastreamento ? (
              <div className="flex items-center gap-2">
                <span className="h-2.5 w-2.5 animate-pulse rounded-full bg-emerald-500" />
                <p className="text-sm font-medium text-tinta">Compartilhando localização...</p>
              </div>
            ) : (
              <p className="text-sm font-medium text-tinta">Pedido marcado como saiu para entrega.</p>
            )}
            {ultimaAtualizacao && (
              <p className="mt-1 text-xs text-tinta-suave">
                Última atualização: {ultimaAtualizacao.toLocaleTimeString('pt-BR')}
              </p>
            )}
            <p className="mt-3 text-xs text-tinta-suave">
              Mantenha essa aba aberta e a tela do celular ligada até finalizar a entrega.
            </p>
            <button
              onClick={marcarEntregue}
              disabled={finalizando}
              className="mt-4 w-full rounded-full bg-acento py-3 font-semibold text-superficie disabled:opacity-60"
            >
              {finalizando ? 'Finalizando...' : '✅ Marcar como entregue'}
            </button>
          </>
        )}
        {erro && <p className="mt-3 text-sm text-acento">{erro}</p>}
      </div>
    </div>
  );
}