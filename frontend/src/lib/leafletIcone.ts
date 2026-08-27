import L from 'leaflet';

// O react-leaflet quebra o ícone padrão do marcador por causa de como o
// Vite/webpack lida com os caminhos dos assets — esse ajuste manual é o
// jeito padrão de contornar isso. Extraído aqui porque mais de uma tela
// usa mapa (RastrearPedido.tsx e GerenciarEntrega.tsx) — sem isso, cada
// uma duplicaria a mesma configuração de ícone.
export const iconePadrao = L.icon({
  iconUrl: 'https://unpkg.com/leaflet@1.9.4/dist/images/marker-icon.png',
  iconRetinaUrl: 'https://unpkg.com/leaflet@1.9.4/dist/images/marker-icon-2x.png',
  shadowUrl: 'https://unpkg.com/leaflet@1.9.4/dist/images/marker-shadow.png',
  iconSize: [25, 41],
  iconAnchor: [12, 41],
});
