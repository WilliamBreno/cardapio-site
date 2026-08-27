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
// (link aberto direto, sem navegação prévia dentro da SPA). Causa: o
// `<MapContainer style={{height:'100%'}}>` fica dentro de um
// `<div className="flex-1">` — a altura dele só existe DEPOIS do
// layout flex terminar de se resolver, mas o Leaflet mede o tamanho do
// próprio container de forma síncrona, no exato instante do mount, e
// nesse instante a altura ainda podia estar zerada. invalidateSize()
// força o Leaflet a reler o tamanho real do container logo em seguida
// (mesmo problema/solução documentados pela própria biblioteca pra
// mapas dentro de layout flex/grid).
function InvalidarTamanhoMapa() {
  const map = useMap();
  useEffect(() => {
    const id = setTimeout(() => map.invalidateSize(), 100);
    return () => clearTimeout(id);
  }, [map]);
  return null;
}

// CodigoConfirmacaoEntrega (24/08/2026) — mostra o código de 4 dígitos
// que o cliente informa pro entregador na hora da entrega, pra ele
// confirmar no painel dele que a entrega aconteceu de verdade (ver
// CompartilharLocalizacao.tsx). Sempre visível assim que o pedido está
// "saiu para entrega", independente do plano ter mapa ao vivo ou não —
// esse mecanismo não depende de rastreamento em tempo real.
function CodigoConfirmacaoEntrega({ codigo }: { codigo: string }) {
  if (!codigo) return null;
  return (
    <div className="mx-auto max-w-xs rounded-xl bg-superficie px-4 py-3 text-center shadow-sm">
      <p className="text-xs text-tinta-suave">Mostre esse código pro entregador na entrega</p>
      <p className="font-carimbo text-2xl font-semibold tracking-[0.3em] text-acento">{codigo}</p>
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
      <div className="flex min-h-screen items-center justify-center bg-fundo">
        <p className="text-tinta-suave">Carregando rastreamento...</p>
      </div>
    );
  }

  if (erro || !dados) {
    return (
      <div className="flex min-h-screen flex-col items-center justify-center gap-2 bg-fundo px-6 text-center">
        <p className="font-display text-xl text-tinta">Não foi possível rastrear</p>
        <p className="text-sm text-tinta-suave">{erro || 'Tenta abrir o link novamente.'}</p>
      </div>
    );
  }

  if (dados.status_entrega === '' || dados.status_entrega === undefined) {
    return (
      <div className="flex min-h-screen flex-col items-center justify-center gap-2 bg-fundo px-6 text-center">
        <p className="font-display text-xl text-tinta">Ainda não saiu para entrega</p>
        <p className="text-sm text-tinta-suave">Assim que o pedido sair, o mapa aparece aqui automaticamente.</p>
      </div>
    );
  }

  if (dados.status_entrega === 'entregue') {
    return (
      <div className="flex min-h-screen flex-col items-center justify-center gap-2 bg-fundo px-6 text-center">
        <p className="font-display text-xl text-tinta">Pedido entregue! 🎉</p>
        <p className="text-sm text-tinta-suave">Esperamos que aproveite.</p>
      </div>
    );
  }

  if (!dados.disponivel) {
    return (
      <div className="flex min-h-screen flex-col items-center justify-center gap-3 bg-fundo px-6 text-center">
        <p className="font-display text-xl text-tinta">Pedido saiu para entrega 🛵</p>
        <p className="text-sm text-tinta-suave">
          O rastreamento em tempo real não está disponível pra essa loja no momento.
        </p>
        <CodigoConfirmacaoEntrega codigo={dados.codigo_confirmacao} />
      </div>
    );
  }

  const posicao: [number, number] = [dados.entregador_latitude, dados.entregador_longitude];
  const semLocalizacaoAinda = dados.entregador_latitude === 0 && dados.entregador_longitude === 0;

  return (
    <div className="flex min-h-screen flex-col bg-fundo">
      <header className="bg-acento px-6 py-4 text-center">
        <h1 className="font-display text-lg tracking-wide text-texto-claro">
          Acompanhando seu pedido #{pedidoId}
        </h1>
        <p className="text-xs text-texto-claro/80">
          {semLocalizacaoAinda
            ? 'Aguardando a primeira atualização de localização...'
            : `Atualizado às ${new Date(dados.entregador_atualizado_em!).toLocaleTimeString('pt-BR')}`}
        </p>
      </header>

      <div className="px-6 pt-3">
        <CodigoConfirmacaoEntrega codigo={dados.codigo_confirmacao} />
      </div>

      {/* relative + MapContainer "absolute inset-0" (em vez de
          height:100%): altura em % dentro de um pai dimensionado só por
          flex-grow (flex-1) é uma cilada clássica de CSS com Leaflet —
          o container media 0px de altura mesmo com o pai já tendo
          altura real (ver InvalidarTamanhoMapa acima, que sozinho não
          bastou pra corrigir; achado testando essa tela num navegador
          real). Posicionamento absoluto contorna o problema porque não
          depende da resolução de porcentagem-em-altura, só da caixa
          geométrica real do ancestral posicionado. */}
      <div className="relative flex-1">
        {semLocalizacaoAinda ? (
          <div className="flex h-full items-center justify-center px-6 text-center">
            <p className="text-tinta-suave">
              O entregador saiu, mas ainda não compartilhou a localização. Atualiza em alguns instantes.
            </p>
          </div>
        ) : (
          <MapContainer center={posicao} zoom={15} style={{ position: 'absolute', inset: 0 }}>
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
    </div>
  );
}