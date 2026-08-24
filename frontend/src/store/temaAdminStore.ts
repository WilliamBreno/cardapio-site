import { create } from 'zustand';
import { persist } from 'zustand/middleware';

type PreferenciaTema = 'claro' | 'escuro';

function preferenciaSistema(): PreferenciaTema {
  if (typeof window === 'undefined' || !window.matchMedia) return 'claro';
  return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'escuro' : 'claro';
}

interface TemaAdminState {
  preferencia: PreferenciaTema;
  definirPreferencia: (preferencia: PreferenciaTema) => void;
  alternar: () => void;
}

// Detecta a preferência do sistema operacional só na primeira visita (valor
// inicial do store, antes de qualquer persistência existir) — a partir do
// primeiro clique no toggle, a escolha manual fica salva (via persist) e
// nunca mais é sobrescrita por prefers-color-scheme.
export const useTemaAdminStore = create<TemaAdminState>()(
  persist(
    (set, get) => ({
      preferencia: preferenciaSistema(),
      definirPreferencia: (preferencia) => set({ preferencia }),
      alternar: () => set({ preferencia: get().preferencia === 'claro' ? 'escuro' : 'claro' }),
    }),
    { name: 'drenux-tema-admin' }
  )
);
