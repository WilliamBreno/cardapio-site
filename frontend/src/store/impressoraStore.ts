import { create } from 'zustand';
import { persist } from 'zustand/middleware';

// autoBuscar (Fase de redesign, 24/08/2026) — fica salvo por navegador/
// computador (persist), não por loja no backend: pareamento Bluetooth é
// inerentemente local à máquina que tem a impressora fisicamente ligada,
// não faz sentido sincronizar essa preferência entre dispositivos.
interface ImpressoraState {
  autoBuscar: boolean;
  definirAutoBuscar: (valor: boolean) => void;
}

export const useImpressoraStore = create<ImpressoraState>()(
  persist(
    (set) => ({
      autoBuscar: false,
      definirAutoBuscar: (autoBuscar) => set({ autoBuscar }),
    }),
    { name: 'drenux-impressora' }
  )
);
