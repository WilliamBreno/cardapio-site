// buscarRota (28/08/2026) — pede o trajeto real (por rua, não linha
// reta) entre dois pontos ao OSRM (Open Source Routing Machine),
// servidor de demonstração público do próprio projeto OSM — mesmo
// espírito do Nominatim já usado no backend: gratuito, sem chave, uso
// direto do navegador. Decisão consciente de não montar isso no
// backend: é só desenho no mapa (não afeta nenhum cálculo de frete/
// comissão), então não precisava de mais uma chamada de rede
// atravessando o nosso servidor.
//
// Falha graciosamente (retorna null) se o serviço estiver fora, sem
// rota encontrada, ou bloqueado por CORS — o mapa continua funcionando
// só com os pinos, sem a linha do trajeto, nunca quebra a tela por
// causa disso.
export interface PontoRota {
  lat: number;
  lng: number;
}

export async function buscarRota(origem: PontoRota, destino: PontoRota): Promise<[number, number][] | null> {
  try {
    const url = `https://router.project-osrm.org/route/v1/driving/${origem.lng},${origem.lat};${destino.lng},${destino.lat}?overview=full&geometries=geojson`;
    const resposta = await fetch(url);
    if (!resposta.ok) return null;
    const dados = await resposta.json();
    const coordenadas = dados?.routes?.[0]?.geometry?.coordinates;
    if (!Array.isArray(coordenadas) || coordenadas.length === 0) return null;
    // GeoJSON vem como [longitude, latitude] — Leaflet espera [latitude, longitude].
    return coordenadas.map((par: [number, number]) => [par[1], par[0]]);
  } catch {
    return null;
  }
}
