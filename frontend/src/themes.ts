export interface Tema {
  id: string;
  nome: string;
  descricao: string;
  acento: string;    // hex — usado no preview do seletor
  fundo: string;
  superficie: string;
}

export const TEMAS: Tema[] = [
  {
    id: 'kraft',
    nome: 'Kraft',
    descricao: 'Padaria, salgados, comida caseira',
    acento: '#A8362A',
    fundo: '#EDE0C8',
    superficie: '#FAF3E4',
  },
  {
    id: 'oceano',
    nome: 'Oceano',
    descricao: 'Açaí, frutos do mar, smoothies',
    acento: '#0077B6',
    fundo: '#E4F0F6',
    superficie: '#F0F8FC',
  },
  {
    id: 'floresta',
    nome: 'Floresta',
    descricao: 'Natural, orgânico, saudável',
    acento: '#2D6A4F',
    fundo: '#E6EDE0',
    superficie: '#F2F7EE',
  },
  {
    id: 'rosa',
    nome: 'Rosa',
    descricao: 'Confeitaria, bolos, doces finos',
    acento: '#C9184A',
    fundo: '#FDF0F5',
    superficie: '#FFF5F9',
  },
  {
    id: 'noite',
    nome: 'Noite',
    descricao: 'Dark · hamburguer, pizza, bar',
    acento: '#7C3AED',
    fundo: '#121212',
    superficie: '#1E1E2E',
  },
  {
    id: 'carvao',
    nome: 'Carvão',
    descricao: 'Minimalista · delivery sofisticado',
    acento: '#374151',
    fundo: '#F5F5F3',
    superficie: '#FFFFFF',
  },
  {
    id: 'brasa',
    nome: 'Brasa',
    descricao: 'Churrasco, espeto, cervejaria',
    acento: '#BF360C',
    fundo: '#F5EDE6',
    superficie: '#FBF6F1',
  },
  {
    id: 'hortela',
    nome: 'Hortelã',
    descricao: 'Sorvete, açaí, tropical, fresco',
    acento: '#00796B',
    fundo: '#E0F5F0',
    superficie: '#F0FDFB',
  },
  {
    id: 'preto',
    nome: 'Preto',
    descricao: 'Dark premium · elegante',
    acento: '#E8B86D',
    fundo: '#0D0D0D',
    superficie: '#1C1C1C',
  },
  {
    id: 'cinza',
    nome: 'Cinza',
    descricao: 'Neutro · moderno e limpo',
    acento: '#404040',
    fundo: '#ECECEC',
    superficie: '#F8F8F8',
  },
  {
    id: 'marinho',
    nome: 'Azul Marinho',
    descricao: 'Sofisticado · delivery premium',
    acento: '#1B3A6B',
    fundo: '#E8EEF4',
    superficie: '#F2F6FA',
  },
  {
    id: 'amarelo',
    nome: 'Amarelo',
    descricao: 'Alegre · lanchonete e café',
    acento: '#F59E0B',
    fundo: '#FFF8DC',
    superficie: '#FFFEF0',
  },
  {
    id: 'vinho',
    nome: 'Vinho',
    descricao: 'Chocolateria, bistrô, enoteca',
    acento: '#7B1535',
    fundo: '#F5E8EC',
    superficie: '#FAF0F3',
  },
  {
    id: 'pessego',
    nome: 'Pêssego',
    descricao: 'Café moderno, smoothie, açaí gourmet',
    acento: '#E8734A',
    fundo: '#FDF0E8',
    superficie: '#FFF7F2',
  },
  {
    id: 'lavanda',
    nome: 'Lavanda',
    descricao: 'Floral, chá, brunch, doceria delicada',
    acento: '#8A5CB8',
    fundo: '#EEE8F6',
    superficie: '#F8F5FC',
  },
  {
    id: 'terracota',
    nome: 'Terracota',
    descricao: 'Pizzaria artesanal, forno a lenha, trattoria',
    acento: '#B05230',
    fundo: '#EED2BC',
    superficie: '#F8E8DA',
  },
];