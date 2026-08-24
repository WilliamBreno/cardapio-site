import { useState, useEffect } from 'react';
import { useParams, useSearchParams, useLocation } from 'react-router-dom';
import { MapContainer, TileLayer, Marker, Popup } from 'react-leaflet';
import L from 'leaflet';
import 'leaflet/dist/leaflet.css';
import { rastrearPedido, rastrearSolicitacao } from '../api/rastreamento';

// O react-leaflet quebra o ícone padrão do marcador por causa de como o
// Vite/webpack lida com os caminhos dos assets — esse ajuste manual é o
// jeito padrão de contornar isso.
const iconePadrao = L.icon({
  iconUrl: 'https://unpkg.com/leaflet@1.9.4/dist/images/marker-icon.png',
  iconRetinaUrl: 'https://unpkg.com/leaflet@1.9.4/dist/images/marker-icon-2x.png',
  shadowUrl: 'https://unpkg.com/leaflet@1.9.4/dist/images/marker-shadow.png',
  iconSize: [25, 41],
  iconAnchor: [12, 41],
});

const INTERVALO_POLL_MS = 10_000; // atualiza o mapa a cada 10s

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
      <div className="flex min-h-screen flex-col items-center justify-center gap-2 bg-fundo px-6 text-center">
        <p className="font-display text-xl text-tinta">Pedido saiu para entrega 🛵</p>
        <p className="text-sm text-tinta-suave">
          O rastreamento em tempo real não está disponível pra essa loja no momento.
        </p>
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

      <div className="flex-1">
        {semLocalizacaoAinda ? (
          <div className="flex h-full items-center justify-center px-6 text-center">
            <p className="text-tinta-suave">
              O entregador saiu, mas ainda não compartilhou a localização. Atualiza em alguns instantes.
            </p>
          </div>
        ) : (
          <MapContainer center={posicao} zoom={15} style={{ height: '100%', width: '100%' }}>
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