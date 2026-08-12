import { api } from './client';

interface CadastroInput {
  nome: string;
  email: string;
  senha: string;
  nome_loja: string;
  segmento_principal: 'alimenticio' | 'mercadoria';
  codigo_afiliado?: string;
  token_assinatura?: string;
  /** "basic" pra nascer direto no Basic (sem mensalidade); omitido = Start. */
  plano?: 'start' | 'basic';
}

interface LoginInput {
  email: string;
  senha: string;
}

interface AuthResponse {
  token: string;
}

interface MensagemResponse {
  mensagem: string;
}

export async function cadastrar(input: CadastroInput): Promise<AuthResponse> {
  const { data } = await api.post<AuthResponse>('/auth/cadastro', input);
  return data;
}

export async function login(input: LoginInput): Promise<AuthResponse> {
  const { data } = await api.post<AuthResponse>('/auth/login', input);
  return data;
}

export async function solicitarResetSenha(email: string): Promise<MensagemResponse> {
  const { data } = await api.post<MensagemResponse>('/auth/esqueci-senha', { email });
  return data;
}

export async function redefinirSenha(token: string, senha: string): Promise<MensagemResponse> {
  const { data } = await api.post<MensagemResponse>('/auth/redefinir-senha', { token, senha });
  return data;
}