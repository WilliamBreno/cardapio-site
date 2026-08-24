import { create } from 'zustand';

// Diferente de authStore/temaAdminStore, esse store NÃO usa persist — é
// estado efêmero de UI (qual visão de Pedidos está aberta agora), não uma
// preferência durável do usuário. Recarregar a página deve voltar pro
// padrão (largura normal), não lembrar a última visão.
interface LayoutAdminState {
  larguraCompleta: boolean;
  definirLarguraCompleta: (valor: boolean) => void;
}

export const useLayoutAdminStore = create<LayoutAdminState>((set) => ({
  larguraCompleta: false,
  definirLarguraCompleta: (larguraCompleta) => set({ larguraCompleta }),
}));
