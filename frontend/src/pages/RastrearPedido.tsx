import { useState, useEffect } from 'react';
import { useParams, useSearchParams, useLocation } from 'react-router-dom';
import { MapContainer, TileLayer, Marker, Popup, useMap } from 'react-leaflet';
import 'leaflet/dist/leaflet.css';
import { rastrearPedido, rastrearSolicitacao } from '../api/rastreamento';
import { iconePadrao } from '../lib/leafletIcone';

const INTERVALO_POLL_MS = 10_000; // atualiza o mapa a cada 10s

// InvalidarTamanhoMapa (27/08/2026) — achado testando essa tela num
// navegador real: o mapa nascia com altura 0 (invisível, sem nenhum
// erro no console) sempre que essa tela é a primeira coisa renderizada
// (link aberto direto, sem navegação prévia dentro da SPA). O layout
// atual já usa "fixed inset-0" (posição com bordas explícitas, sem
// depender de porcentagem de altura em cima de flex-grow — a cilada
// original), mas invalidateSize() continua aqui como segurança extra:
// não custa nada e cobre qualquer outro cenário de mount com o
// container ainda não totalmente assentado.
function InvalidarTamanhoMapa() {
  const map = useMap();
  useEffect(() => {
    const id = setTimeout(() => map.invalidateSize(), 100);
    return () => clearTimeout(id);
  }, [map]);
  return null;
}

// CodigoConfirmacaoEntrega (24/08/2026, redesenhada em 28/08/2026) —
// mostra o código de 4 dígitos que o cliente informa pro entregador na
// hora da entrega. Vive como um cartão flutuante ("bottom sheet") por
// cima do mapa em tela cheia, não mais empilhado acima dele — sempre
// visível assim que o pedido está "saiu para entrega", independente do
// plano ter mapa ao vivo ou não.
function CodigoConfirmacaoEntrega({ codigo }: { codigo: string }) {
  if (!codigo) return null;
  return (
    <div className="pointer-events-auto mx-auto max-w-xs rounded-2xl bg-superficie px-5 py-4 text-center shadow-xl ring-1 ring-tinta/5">
      <p className="text-xs text-tinta-suave">Mostre esse código pro entregador na entrega</p>
      <p className="font-carimbo text-3xl font-semibold tracking-[0.35em] text-acento">{codigo}</p>
    </div>
  );
}

// TelaCentralizada é o layout compartilhado pelos estados sem mapa
// (carregando, erro, ainda não saiu, entregue, sem rastreamento ao
// vivo) — um cartão único centralizado na tela, mesma linguagem visual
// do cartão flutuante do mapa, em vez de texto solto direto no fundo.
function TelaCentralizada({ children }: { children: React.ReactNode }) {
  return (
    <div className="flex min-h-screen items-center justify-center bg-fundo px-6">
      <div className="w-full max-w-sm space-y-2 rounded-2xl bg-superficie px-6 py-8 text-center shadow-xl ring-1 ring-tinta/5">
        {children}
      </div>
    </div>
  );
}

export function RastrearPedido() {
  const { slug, id } = useParams<{ slug: string; id: string }>();
  const [searchParams] = useSearchParams();
  const location = useLocation();
  const telefone = searchParams.get('telefone') || '';
  const pedidoId = Number(id);
  const ehSolicitacao = location.pathname.includes('/solicitacao/');
  const buscarRastreamento = ehSolicitacao ? rastrearSolicitacao : rastrearPedido;

  const [dados, setDados] = useState<{
    status_entrega: string;
    entregador_latitude: number;
    entregador_longitude: number;
    entregador_atualizado_em: string | null;
    disponivel: boolean;
    codigo_confirmacao: string;
  } | null>(null);
  const [erro, setErro] = useState<string | null>(null);
  const [carregando, setCarregando] = useState(true);

  useEffect(() => {
    if (!slug || !pedidoId || !telefone) {
      setErro('Link de rastreamento incompleto.');
      setCarregando(false);
      return;
    }

    let cancelado = false;

    async function buscar() {
      try {
        const resultado = await buscarRastreamento(slug!, pedidoId, telefone);
        if (!cancelado) {
          setDados(resultado);
          setErro(null);
        }
      } catch {
        if (!cancelado) setErro('Não conseguimos encontrar esse pedido pra esse telefone.');
      } finally {
        if (!cancelado) setCarregando(false);
      }
    }

    buscar();
    const intervalo = setInterval(buscar, INTERVALO_POLL_MS);
    return () => {
      cancelado = true;
      clearInterval(intervalo);
    };
  }, [slug, pedidoId, telefone]);

  if (carregando) {
    return (
      <TelaCentralizada>
        <p className="text-tinta-suave">Carregando rastreamento...</p>
      </TelaCentralizada>
    );
  }

  if (erro || !dados) {
    return (
      <TelaCentralizada>
        <p className="font-display text-xl text-tinta">Não foi possível rastrear</p>
        <p className="text-sm text-tinta-suave">{erro || 'Tenta abrir o link novamente.'}</p>
      </TelaCentralizada>
    );
  }

  if (dados.status_entrega === '' || dados.status_entrega === undefined) {
    return (
      <TelaCentralizada>
        <p className="font-display text-xl text-tinta">Ainda não saiu para entrega</p>
        <p className="text-sm text-tinta-suave">Assim que o pedido sair, o mapa aparece aqui automaticamente.</p>
      </TelaCentralizada>
    );
  }

  if (dados.status_entrega === 'entregue') {
    return (
      <TelaCentralizada>
        <p className="font-display text-xl text-tinta">Pedido entregue! 🎉</p>
        <p className="text-sm text-tinta-suave">Esperamos que aproveite.</p>
      </TelaCentralizada>
    );
  }

  if (!dados.disponivel) {
    return (
      <TelaCentralizada>
        <p className="font-display text-xl text-tinta">Pedido saiu para entrega 🛵</p>
        <p className="text-sm text-tinta-suave">
          O rastreamento em tempo real não está disponível pra essa loja no momento.
        </p>
        <div className="pt-2">
          <CodigoConfirmacaoEntrega codigo={dados.codigo_confirmacao} />
        </div>
      </TelaCentralizada>
    );
  }

  const posicao: [number, number] = [dados.entregador_latitude, dados.entregador_longitude];
  const semLocalizacaoAinda = dados.entregador_latitude === 0 && dados.entregador_longitude === 0;

  return (
    <div className="relative min-h-screen overflow-hidden bg-fundo">
      {/* Mapa em tela cheia, sempre no fundo — cabeçalho e código
          flutuam por cima dele, estilo apps de entrega/corrida (Uber,
          99). Fixed inset-0 (não flex-1/height:100%): dá ao mapa uma
          geometria explícita desde o primeiro frame, evitando a mesma
          cilada de altura zero já documentada em InvalidarTamanhoMapa. */}
      <div className="fixed inset-0">
        {semLocalizacaoAinda ? (
          <div className="flex h-full items-center justify-center bg-superficie px-6 text-center">
            <p className="text-tinta-suave">
              O entregador saiu, mas ainda não compartilhou a localização. Atualiza em alguns instantes.
            </p>
          </div>
        ) : (
          <MapContainer center={posicao} zoom={15} zoomControl={false} style={{ height: '100%', width: '100%' }}>
            <InvalidarTamanhoMapa />
            <TileLayer
              url="https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png"
              attribution='&copy; <a href="https://www.openstreetmap.org/copyright">OpenStreetMap</a>'
            />
            <Marker position={posicao} icon={iconePadrao}>
              <Popup>Localização do entregador</Popup>
            </Marker>
          </MapContainer>
        )}
      </div>

      {/* Cabeçalho flutuante — pílula centralizada no topo, por cima do
          mapa (z-[1000] fica acima de qualquer pane/controle do
          Leaflet, que não passa de ~650). */}
      <header className="pointer-events-none fixed inset-x-0 top-0 z-[1000] flex justify-center px-4 pt-4">
        <div className="pointer-events-auto flex items-center gap-2 rounded-full bg-acento px-5 py-2.5 shadow-lg">
          <span
            className={`h-2 w-2 shrink-0 rounded-full ${semLocalizacaoAinda ? 'bg-texto-claro/50' : 'animate-pulse bg-emerald-400'}`}
          />
          <div className="text-left">
            <p className="font-display text-sm leading-tight tracking-wide text-texto-claro">
              Pedido #{pedidoId}
            </p>
            <p className="text-[11px] leading-tight text-texto-claro/80">
              {semLocalizacaoAinda
                ? 'Aguardando localização do entregador...'
                : `Atualizado às ${new Date(dados.entregador_atualizado_em!).toLocaleTimeString('pt-BR')}`}
            </p>
          </div>
        </div>
      </header>

      {/* Código flutuante — "bottom sheet" fixo na base da tela. */}
      <div className="pointer-events-none fixed inset-x-0 bottom-0 z-[1000] px-4 pb-[max(1rem,env(safe-area-inset-bottom))]">
        <CodigoConfirmacaoEntrega codigo={dados.codigo_confirmacao} />
      </div>
    </div>
  );
}
