import { useEffect } from 'react';
import { useMap } from 'react-leaflet';

// InvalidarTamanhoMapa (27/08/2026) — achado testando RastrearPedido.tsx
// num navegador real: o mapa nascia com altura 0 (invisível, sem nenhum
// erro no console) sempre que a tela é a primeira coisa renderizada
// (link aberto direto, sem navegação prévia dentro da SPA). As telas de
// mapa já usam "fixed inset-0" (posição com bordas explícitas, sem
// depender de porcentagem de altura em cima de flex-grow — a cilada
// original), mas invalidateSize() continua aqui como segurança extra:
// não custa nada e cobre qualquer outro cenário de mount com o
// container ainda não totalmente assentado.
export function InvalidarTamanhoMapa() {
  const map = useMap();
  useEffect(() => {
    const id = setTimeout(() => map.invalidateSize(), 100);
    return () => clearTimeout(id);
  }, [map]);
  return null;
}

// AjustarBoundsMapa (28/08/2026) — com dois pontos no mapa (pessoa +
// destino), um "center"/"zoom" fixo não enquadra os dois. fitBounds
// reenquadra pra sempre mostrar ambos, com uma margem confortável.
// Compartilhado entre RastrearPedido.tsx (cliente vê o entregador +
// destino) e GerenciarEntrega.tsx (entregador vê a própria posição +
// destino) — mesma necessidade nas duas pontas.
export function AjustarBoundsMapa({ pontos }: { pontos: [number, number][] }) {
  const map = useMap();
  const chave = pontos.map((p) => p.join(',')).join('|');
  useEffect(() => {
    if (pontos.length < 2) return;
    const id = setTimeout(() => map.fitBounds(pontos, { padding: [60, 60], maxZoom: 16 }), 150);
    return () => clearTimeout(id);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [map, chave]);
  return null;
}
