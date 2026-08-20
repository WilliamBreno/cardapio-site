import type { Pedido } from '../api/types';

// Impressão de comanda via Bluetooth (Fase 10.7) — impressora térmica de
// recibo/comanda comum (ESC/POS), não equipamento fiscal homologado.
// 100% client-side via Web Bluetooth API, sem nenhuma rota de backend.
//
// Limitação sem contorno possível: Web Bluetooth só existe em Chrome/Edge
// (desktop ou Android) — não existe em NENHUM navegador no iOS/iPadOS
// (restrição da própria Apple). bluetoothSuportado() deixa a tela avisar
// isso antes de tentar, em vez de falhar silenciosamente.

const SERVICO_IMPRESSORA = '000018f0-0000-1000-8000-00805f9b34fb';

const ESC = 0x1b;
const GS = 0x1d;

export function bluetoothSuportado(): boolean {
  return typeof navigator !== 'undefined' && 'bluetooth' in navigator;
}

// A Web Bluetooth API exige gesto do usuário pra parear e não permite
// persistir o dispositivo entre recarregamentos de página — mas dentro da
// mesma sessão (mesmo carregamento da SPA), reaproveita a característica já
// conectada em vez de pedir pareamento de novo a cada comanda impressa.
let caracteristicaAtual: BluetoothRemoteGATTCharacteristic | null = null;

async function obterCaracteristicaEscrita(): Promise<BluetoothRemoteGATTCharacteristic> {
  if (caracteristicaAtual && caracteristicaAtual.service.device.gatt?.connected) {
    return caracteristicaAtual;
  }

  const dispositivo = await navigator.bluetooth.requestDevice({
    filters: [{ services: [SERVICO_IMPRESSORA] }],
    optionalServices: [SERVICO_IMPRESSORA],
  });

  const servidor = await dispositivo.gatt?.connect();
  if (!servidor) throw new Error('Não foi possível conectar à impressora.');

  const servico = await servidor.getPrimaryService(SERVICO_IMPRESSORA);
  const caracteristicas = await servico.getCharacteristics();
  // A UUID exata da característica de escrita varia por fabricante — em
  // vez de fixar uma, procura a que anuncia suporte a escrita.
  const caracteristica = caracteristicas.find((c) => c.properties.writeWithoutResponse || c.properties.write);
  if (!caracteristica) throw new Error('Essa impressora não tem uma característica de escrita compatível.');

  dispositivo.addEventListener('gattserverdisconnected', () => {
    caracteristicaAtual = null;
  });

  caracteristicaAtual = caracteristica;
  return caracteristica;
}

// Impressoras térmicas variam de codepage (CP850, CP860, WCP1252 etc.) e
// sem saber o modelo exato do lojista não dá pra garantir qual delas está
// configurada — tira acento antes de imprimir pra evitar caractere quebrado
// (mojibake), preferindo "sem acento" a "ilegível".
function normalizarTexto(s: string): string {
  return s.normalize('NFD').replace(/[̀-ͯ]/g, '');
}

function formatarDataComanda(iso: string): string {
  return new Date(iso).toLocaleDateString('pt-BR', {
    day: '2-digit',
    month: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  });
}

function montarComandaESCPOS(pedido: Pedido, nomeLoja: string): Uint8Array {
  const bytes: number[] = [];
  const comando = (...codigos: number[]) => bytes.push(...codigos);
  const texto = (s: string) => bytes.push(...new TextEncoder().encode(normalizarTexto(s)));

  comando(ESC, 0x40); // inicializa a impressora

  comando(ESC, 0x61, 0x01); // centralizado
  comando(ESC, 0x45, 0x01); // negrito on
  texto(`${nomeLoja}\n`);
  comando(ESC, 0x45, 0x00); // negrito off
  texto(`Pedido #${pedido.id}\n`);
  texto(`${formatarDataComanda(pedido.data_retirada)}\n`);

  comando(ESC, 0x61, 0x00); // alinhado à esquerda
  texto('--------------------------------\n');
  texto(`Cliente: ${pedido.cliente_nome}\n`);
  texto(`Fone: ${pedido.cliente_telefone}\n`);
  texto(pedido.modo_entrega === 'entrega' ? 'Entrega\n' : 'Retirada\n');
  if (pedido.modo_entrega === 'entrega' && pedido.endereco_entrega) {
    texto(`${pedido.endereco_entrega}\n`);
  }
  texto('--------------------------------\n');

  for (const item of pedido.itens) {
    texto(`${item.quantidade}x ${item.produto_nome}\n`);
    if (item.variacao_nome) texto(`   (${item.variacao_nome})\n`);
  }
  for (const combo of pedido.combos ?? []) {
    texto(`${combo.quantidade}x ${combo.nome} (combo)\n`);
    for (const item of combo.itens) {
      texto(`   ${item.quantidade}x ${item.produto_nome}${item.variacao_nome ? ` (${item.variacao_nome})` : ''}\n`);
    }
  }

  texto('--------------------------------\n');
  comando(ESC, 0x45, 0x01);
  texto(`TOTAL: R$ ${pedido.total.toFixed(2).replace('.', ',')}\n`);
  comando(ESC, 0x45, 0x00);
  texto('\n\n\n');
  comando(GS, 0x56, 0x00); // corta o papel

  return new Uint8Array(bytes);
}

// BLE tem limite de tamanho por escrita (MTU) — quebra em blocos pequenos
// pra não estourar o buffer de característica em impressoras com MTU baixo
// (o padrão sem negociação é só 20 bytes; 100 é um meio-termo comum e
// seguro na prática pra esse tipo de impressora).
const TAMANHO_BLOCO = 100;

export async function imprimirComanda(pedido: Pedido, nomeLoja: string): Promise<void> {
  if (!bluetoothSuportado()) {
    throw new Error('Este navegador não suporta impressão via Bluetooth — funciona em Chrome ou Edge, no computador ou Android (não existe no iPhone/iPad).');
  }

  const caracteristica = await obterCaracteristicaEscrita();
  const bytes = montarComandaESCPOS(pedido, nomeLoja);

  for (let i = 0; i < bytes.length; i += TAMANHO_BLOCO) {
    const bloco = bytes.slice(i, i + TAMANHO_BLOCO);
    if (caracteristica.properties.writeWithoutResponse) {
      await caracteristica.writeValueWithoutResponse(bloco);
    } else {
      await caracteristica.writeValue(bloco);
    }
  }
}
