import { useState, useEffect, useRef } from 'react';
import { useParams, useSearchParams } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import { MapContainer, TileLayer, Marker, Popup } from 'react-leaflet';
import 'leaflet/dist/leaflet.css';
import axios from 'axios';
import {
  buscarParaEntregador, atualizarLocalizacaoEntregador, atualizarStatusEntregador,
} from '../api/rastreamento';
import { iconePadrao } from '../lib/leafletIcone';

const INTERVALO_MS = 25_000; // 25 segundos, mesmo padrão de CompartilharLocalizacao.tsx

// GerenciarEntrega (26/08/2026) é a versão pública do que antes só
// existia em CompartilharLocalizacao.tsx (admin, atrás de login) — o
// botão "Gerar link" em Pedidos.tsx manda pra cá, com um token na URL em
// vez de exigir a conta do dono. Pensada pra ser aberta por quem está
// entregando de verdade (funcionário, motoboy terceirizado), que não tem
// (e não precisa ter) login nesse sistema.
export function GerenciarEntrega() {
  const { slug, id } = useParams<{ slug: string; id: string }>();
  const [searchParams] = useSearchParams();
  const token = searchParams.get('token') || '';
  const pedidoId = Number(id);

  const { data: pedido, isLoading, error, refetch } = useQuery({
    queryKey: ['entregador', slug, pedidoId, token],
    queryFn: () => buscarParaEntregador(slug!, pedidoId, token),
    enabled: Boolean(slug && pedidoId && token),
    refetchOnWindowFocus: false,
  });

  const [compartilhando, setCompartilhando] = useState(false);
  const [ultimaAtualizacao, setUltimaAtualizacao] = useState<Date | null>(null);
  const [erro, setErro] = useState<string | null>(null);
  const [finalizando, setFinalizando] = useState(false);
  const [codigo, setCodigo] = useState('');
  const watchIdRef = useRef<number | null>(null);
  const intervalRef = useRef<ReturnType<typeof setInterval> | null>(null);
  const posicaoAtualRef = useRef<{ lat: number; lng: number } | null>(null);

  useEffect(() => {
    return () => {
      if (watchIdRef.current !== null) navigator.geolocation.clearWatch(watchIdRef.current);
      if (intervalRef.current) clearInterval(intervalRef.current);
    };
  }, []);

  // abrirNavegacaoExterna manda o entregador pro Google Maps (app, se
  // instalado, senão o site) já com a rota até o destino calculada —
  // decisão consciente de não desenhar rota/turn-by-turn dentro do
  // Drenux (esforço grande, ver auditoria), delegando isso pro app de
  // navegação que a pessoa já usa. Usa coordenada quando a geocodificação
  // em segundo plano (PedidoService) já terminou; cai pro endereço em
  // texto (o Google Maps também aceita endereço como destino) enquanto
  // isso não acontece ou se a geocodificação falhou.
  function abrirNavegacaoExterna(pedidoAtual: NonNullable<typeof pedido>) {
    const temCoordenada = pedidoAtual.destino_latitude !== 0 || pedidoAtual.destino_longitude !== 0;
    const destino = temCoordenada
      ? `${pedidoAtual.destino_latitude},${pedidoAtual.destino_longitude}`
      : encodeURIComponent(pedidoAtual.endereco_entrega);
    if (!destino) return;
    window.open(`https://www.google.com/maps/dir/?api=1&destination=${destino}`, '_blank', 'noopener,noreferrer');
  }

  async function iniciarCompartilhamento() {
    if (!slug || !pedido) return;
    const temRastreamento = pedido.disponivel;
    if (temRastreamento && !navigator.geolocation) {
      setErro('Esse navegador não suporta geolocalização.');
      return;
    }
    setErro(null);

    // Se o pedido já estiver "saiu_para_entrega" e o plano não tiver
    // rastreamento ao vivo, esse clique não é "iniciar corrida" de
    // verdade — é só avançar pra tela de digitar o código (ver rótulo
    // "✅ Confirmar entrega" abaixo). Não abre navegação de novo nesse
    // caso nem reenvia o status.
    const soConfirmando = pedido.status_entrega === 'saiu_para_entrega' && !temRastreamento;

    if (!soConfirmando) {
      // Abre a navegação ANTES de qualquer `await` — precisa continuar
      // síncrono com o clique do botão, senão navegadores tratam como
      // popup não solicitado e bloqueiam a aba nova.
      abrirNavegacaoExterna(pedido);
    }

    // Se o pedido já estiver "saiu_para_entrega" (ex: o dono já avançou a
    // etapa pelo Kanban/Lista antes de gerar o link, ou o entregador só
    // recarregou essa página no meio da entrega), não manda o status de
    // novo — evita reenviar o aviso de WhatsApp de "saiu para entrega"
    // pro cliente uma segunda vez. Só o compartilhamento de GPS (abaixo)
    // precisa ser retomado, já que ele vive em memória local e se perde
    // ao recarregar a página.
    if (pedido.status_entrega !== 'saiu_para_entrega') {
      try {
        await atualizarStatusEntregador(slug, pedidoId, token, 'saiu_para_entrega');
      } catch {
        setErro('Não foi possível marcar o pedido como "saiu para entrega". Tenta de novo.');
        return;
      }
    }

    if (!temRastreamento) {
      setCompartilhando(true);
      return;
    }

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
      if (!posicaoAtualRef.current || !slug) return;
      try {
        await atualizarLocalizacaoEntregador(slug, pedidoId, token, posicaoAtualRef.current.lat, posicaoAtualRef.current.lng);
        setUltimaAtualizacao(new Date());
      } catch {
        // Falha silenciosa num ciclo só — tenta de novo no próximo.
      }
    }

    enviar();
    intervalRef.current = setInterval(enviar, INTERVALO_MS);
    setCompartilhando(true);
  }

  function pararCompartilhamento() {
    if (watchIdRef.current !== null) navigator.geolocation.clearWatch(watchIdRef.current);
    if (intervalRef.current) clearInterval(intervalRef.current);
    setCompartilhando(false);
  }

  async function marcarEntregue() {
    if (!slug) return;
    setFinalizando(true);
    setErro(null);
    try {
      await atualizarStatusEntregador(slug, pedidoId, token, 'entregue', codigo);
      pararCompartilhamento();
      refetch();
    } catch (e) {
      setErro(
        axios.isAxiosError(e) && e.response?.data?.erro
          ? e.response.data.erro
          : 'Não foi possível marcar como entregue. Tenta de novo.'
      );
      setFinalizando(false);
    }
  }

  if (!slug || !pedidoId || !token) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-fundo px-6 text-center">
        <p className="text-tinta-suave">Link de entrega inválido.</p>
      </div>
    );
  }

  if (isLoading) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-fundo">
        <p className="text-tinta-suave">Carregando...</p>
      </div>
    );
  }

  if (error || !pedido) {
    return (
      <div className="flex min-h-screen flex-col items-center justify-center gap-2 bg-fundo px-6 text-center">
        <p className="font-display text-xl text-tinta">Não foi possível abrir esse link</p>
        <p className="text-sm text-tinta-suave">Confere se o link está completo e tenta de novo.</p>
      </div>
    );
  }

  if (pedido.status_entrega === 'entregue') {
    return (
      <div className="flex min-h-screen flex-col items-center justify-center gap-2 bg-fundo px-6 text-center">
        <p className="font-display text-xl text-tinta">Pedido já entregue ✅</p>
        <p className="text-sm text-tinta-suave">Não tem mais nada pra fazer por aqui.</p>
      </div>
    );
  }

  const temDestino = pedido.destino_latitude !== 0 || pedido.destino_longitude !== 0;
  // painelDeCompartilhar só troca de tela depois que o GPS foi de fato
  // iniciado NESSA sessão (compartilhando) — se depender só do status do
  // servidor (pedido.status_entrega === 'saiu_para_entrega'), a página
  // mostraria "compartilhando localização" sem o watchPosition realmente
  // ligado sempre que carregada/recarregada com o pedido já nessa etapa
  // (ex: dono avançou pelo Kanban antes de gerar o link, ou o entregador
  // só deu F5 no meio da entrega). statusJaSaiuParaEntrega só serve pra
  // adaptar o texto do botão inicial — ver iniciarCompartilhamento, que
  // usa o mesmo campo pra não reenviar o aviso de WhatsApp ao cliente.
  const statusJaSaiuParaEntrega = pedido.status_entrega === 'saiu_para_entrega';

  return (
    <div className="flex min-h-screen flex-col bg-fundo">
      <header className="bg-acento px-6 py-4 text-center">
        <h1 className="font-display text-lg tracking-wide text-texto-claro">Entrega do pedido #{pedidoId}</h1>
        <p className="text-xs text-texto-claro/80">{pedido.cliente_nome}</p>
      </header>

      <div className="h-64 shrink-0">
        {temDestino ? (
          <MapContainer
            center={[pedido.destino_latitude, pedido.destino_longitude]}
            zoom={15}
            style={{ height: '100%', width: '100%' }}
          >
            <TileLayer
              url="https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png"
              attribution='&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a>'
            />
            <Marker position={[pedido.destino_latitude, pedido.destino_longitude]} icon={iconePadrao}>
              <Popup>Destino da entrega</Popup>
            </Marker>
          </MapContainer>
        ) : (
          <div className="flex h-full items-center justify-center bg-superficie px-6 text-center">
            <p className="text-sm text-tinta-suave">Localização exata do destino ainda não disponível.</p>
          </div>
        )}
      </div>

      <div className="mx-auto w-full max-w-md flex-1 space-y-4 p-4">
        <div className="rounded-2xl bg-superficie p-4 shadow-sm">
          <p className="text-xs font-medium uppercase tracking-wide text-tinta-suave">Endereço</p>
          <p className="text-sm text-tinta">{pedido.endereco_entrega || 'Endereço não informado'}</p>
        </div>

        <div className="rounded-2xl bg-superficie p-5 shadow-sm">
          {!compartilhando ? (
            <>
              <p className="text-sm text-tinta-suave">
                {statusJaSaiuParaEntrega && pedido.disponivel
                  ? 'Esse pedido já está a caminho. Toque abaixo pra reabrir a navegação e retomar o compartilhamento de localização nesse aparelho.'
                  : statusJaSaiuParaEntrega
                  ? 'Esse pedido já está a caminho. Toque abaixo pra confirmar a entrega com o código do cliente.'
                  : pedido.disponivel
                  ? 'Ao iniciar a corrida, abrimos a navegação até o destino e o cliente já pode acompanhar sua localização em tempo real. Mantenha essa aba aberta enquanto estiver a caminho.'
                  : 'Isso abre a navegação até o destino e marca o pedido como "saiu para entrega", avisando o cliente.'}
              </p>
              <button
                onClick={iniciarCompartilhamento}
                className="mt-4 w-full rounded-full bg-acento py-3 font-semibold text-texto-claro"
              >
                {statusJaSaiuParaEntrega && pedido.disponivel
                  ? '🗺️ Reabrir navegação e retomar corrida'
                  : statusJaSaiuParaEntrega
                  ? '✅ Confirmar entrega'
                  : '🗺️ Iniciar corrida'}
              </button>
            </>
          ) : (
            <>
              {pedido.disponivel ? (
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
                onClick={() => abrirNavegacaoExterna(pedido)}
                className="mt-2 text-xs font-medium text-acento underline"
              >
                🗺️ Reabrir navegação até o destino
              </button>

              <div className="mt-4">
                <label className="mb-1 block text-xs font-medium uppercase tracking-wide text-tinta-suave">
                  Código de confirmação
                </label>
                <p className="mb-2 text-xs text-tinta-suave">
                  Peça pro cliente o código que aparece na tela de rastreamento dele.
                </p>
                <input
                  type="text"
                  inputMode="numeric"
                  maxLength={4}
                  value={codigo}
                  onChange={(e) => setCodigo(e.target.value.replace(/\D/g, '').slice(0, 4))}
                  placeholder="0000"
                  className="w-full rounded-xl border border-tinta/15 bg-fundo px-3 py-2 text-center font-carimbo text-lg tracking-[0.3em] text-tinta outline-none focus:border-acento"
                />
              </div>

              <button
                onClick={marcarEntregue}
                disabled={finalizando || codigo.length !== 4}
                className="mt-4 w-full rounded-full bg-acento py-3 font-semibold text-texto-claro disabled:opacity-60"
              >
                {finalizando ? 'Finalizando...' : '✅ Marcar como entregue'}
              </button>
            </>
          )}
          {erro && <p className="mt-3 text-sm text-acento">{erro}</p>}
        </div>
      </div>
    </div>
  );
}
