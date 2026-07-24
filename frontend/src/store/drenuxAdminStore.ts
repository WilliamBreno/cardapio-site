import { create } from 'zustand';
import { persist } from 'zustand/middleware';

interface DrenuxAdminState {
  secret: string | null;
  setSecret: (secret: string) => void;
  logout: () => void;
}

// Guarda o secret de /drenux/* (Fase 5.5 — controle de repasse de
// afiliado) — não é um JWT, é o mesmo valor fixo configurado no backend
// via DRENUX_ADMIN_SECRET (ver middleware.DrenuxAdminRequired). Separado
// dos outros stores de auth de propósito: essa não é uma identidade, é
// só uma senha compartilhada de uso interno.
export const useDrenuxAdminStore = create<DrenuxAdminState>()(
  persist(
    (set) => ({
      secret: null,
      setSecret: (secret) => set({ secret }),
      logout: () => set({ secret: null }),
    }),
    { name: 'drenux-admin-storage' }
  )
);
