import { useEffect, useRef, useState } from 'react';
import QRCode from 'qrcode';
import type { TipoProduto } from '../api/types';
import { rotuloCatalogo } from '../lib/utils';

interface Props {
  slug: string;
  nomeLoja: string;
  segmentoLoja?: TipoProduto;
}

export function QRCodeCardapio({ slug, nomeLoja, segmentoLoja }: Props) {
  const rotulo = rotuloCatalogo(segmentoLoja);
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const [gerado, setGerado] = useState(false);

  const baseUrl = import.meta.env.VITE_PUBLIC_URL || window.location.origin;
  const url = `${baseUrl}/${slug}`;

  useEffect(() => {
    if (!canvasRef.current) return;

    QRCode.toCanvas(canvasRef.current, url, {
      width: 240,
      margin: 2,
      color: {
        dark: '#2B2118',  // tinta — mesma cor do texto da paleta
        light: '#FAF3E4', // superficie — fundo claro da paleta
      },
    }).then(() => setGerado(true));
  }, [url]);

  function baixar() {
    if (!canvasRef.current) return;
    const link = document.createElement('a');
    link.download = `qrcode-${slug}.png`;
    link.href = canvasRef.current.toDataURL('image/png');
    link.click();
  }

  return (
    <div className="space-y-4 rounded-2xl bg-superficie p-5 shadow-sm">
      <div>
        <h2 className="font-display text-lg tracking-wide text-tinta">QR Code do {rotulo}</h2>
        <p className="mt-1 text-sm text-tinta-suave">
          Coloca em embalagens, cartazes ou na bio das redes sociais — quem escanear vai direto pro
          {' '}{rotulo} da <strong>{nomeLoja}</strong>.
        </p>
      </div>

      <div className="flex flex-col items-center gap-4 sm:flex-row sm:items-start">
        <div className="rounded-xl border border-tinta/10 p-3">
          <canvas ref={canvasRef} />
        </div>

        <div className="space-y-3 text-sm">
          <div className="rounded-lg bg-fundo px-3 py-2">
            <p className="text-xs font-medium uppercase tracking-wide text-tinta-suave">Link do {rotulo}</p>
            <p className="mt-0.5 break-all font-carimbo text-sm text-tinta">{url}</p>
          </div>

          {gerado && (
            <button
              onClick={baixar}
              className="w-full btn-neu-primario transition hover:bg-acento/90"
            >
              Baixar imagem (PNG)
            </button>
          )}
        </div>
      </div>
    </div>
  );
}