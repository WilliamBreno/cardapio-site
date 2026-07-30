import { useState, useRef } from 'react';
import { buscarCep } from '../api/cep';
import { Campo } from './Campo';

export interface EnderecoValor {
  cep: string;
  rua: string;
  numero: string;
  complemento: string;
  bairro: string;
  cidade: string;
  estado: string;
}

export const enderecoVazio: EnderecoValor = {
  cep: '', rua: '', numero: '', complemento: '', bairro: '', cidade: '', estado: '',
};

// Monta o endereço nessa ordem — rua+número, bairro, cidade-estado, cep —
// porque é o formato que o Nominatim (motor de geocodificação gratuito
// que usamos pro cálculo de frete) reconhece com mais precisão. Endereço
// digitado livre, sem essa estrutura, é a causa mais comum de busca não
// encontrar nada.
export function enderecoParaTexto(v: EnderecoValor): string {
  let linha = v.rua.trim();
  if (v.numero.trim()) linha = `${linha}, ${v.numero.trim()}`;
  if (v.complemento.trim()) linha = `${linha} - ${v.complemento.trim()}`;

  const cidadeEstado = [v.cidade.trim(), v.estado.trim()].filter(Boolean).join(' - ');

  return [linha, v.bairro.trim(), cidadeEstado, v.cep.trim()].filter(Boolean).join(', ');
}

export function enderecoPreenchido(v: EnderecoValor): boolean {
  return v.rua.trim() !== '' && v.cidade.trim() !== '';
}

// Campos estruturados (rua/numero/bairro/cidade/estado/cep separados) pra
// mandar na cotação/criação de frete — a geocodificação no backend usa
// esses campos direto numa busca estruturada do Nominatim, muito mais
// confiável do que tentar quebrar de volta o texto livre montado por
// enderecoParaTexto.
export function enderecoParaCampos(v: EnderecoValor) {
  return {
    rua: v.rua.trim(),
    numero: v.numero.trim(),
    complemento: v.complemento.trim(),
    bairro: v.bairro.trim(),
    cidade: v.cidade.trim(),
    estado: v.estado.trim(),
    cep: v.cep.trim(),
  };
}

const ESTADOS = [
  'AC', 'AL', 'AP', 'AM', 'BA', 'CE', 'DF', 'ES', 'GO', 'MA', 'MT', 'MS', 'MG', 'PA', 'PB',
  'PR', 'PE', 'PI', 'RJ', 'RN', 'RS', 'RO', 'RR', 'SC', 'SP', 'SE', 'TO',
];

interface Props {
  valor: EnderecoValor;
  onChange: (v: EnderecoValor) => void;
}

const campoClasse = 'w-full rounded-lg border border-tinta/20 bg-fundo px-3 py-2 text-tinta outline-none focus:border-acento';

export function EnderecoCampos({ valor, onChange }: Props) {
  const [buscandoCep, setBuscandoCep] = useState(false);
  const [erroCep, setErroCep] = useState<string | null>(null);
  const ultimoCepBuscado = useRef('');

  async function buscarPorCep(cepDigitado: string) {
    const digitos = cepDigitado.replace(/\D/g, '');
    if (digitos.length !== 8 || digitos === ultimoCepBuscado.current) return;
    ultimoCepBuscado.current = digitos;
    setBuscandoCep(true);
    setErroCep(null);
    try {
      const dados = await buscarCep(digitos);
      if (dados.erro) {
        setErroCep('CEP não encontrado — preenche o endereço manualmente.');
        return;
      }
      onChange({
        ...valor,
        cep: cepDigitado,
        rua: dados.logradouro || valor.rua,
        bairro: dados.bairro || valor.bairro,
        cidade: dados.localidade || valor.cidade,
        estado: dados.uf || valor.estado,
      });
    } catch {
      setErroCep('Não conseguimos consultar esse CEP agora — preenche o endereço manualmente.');
    } finally {
      setBuscandoCep(false);
    }
  }

  function handleCepChange(v: string) {
    onChange({ ...valor, cep: v });
    buscarPorCep(v);
  }

  return (
    <div className="space-y-3">
      <Campo label="CEP (opcional — se souber, preenche o resto sozinho)">
        <input
          value={valor.cep}
          onChange={(e) => handleCepChange(e.target.value)}
          placeholder="49000-000"
          maxLength={9}
          inputMode="numeric"
          className={campoClasse}
        />
        {buscandoCep && <span className="mt-1 block text-xs text-tinta-suave">Buscando endereço pelo CEP...</span>}
        {erroCep && <span className="mt-1 block text-xs text-acento">{erroCep}</span>}
      </Campo>

      <div className="flex gap-2">
        <Campo label="Rua" className="flex-[2]">
          <input value={valor.rua} onChange={(e) => onChange({ ...valor, rua: e.target.value })} placeholder="Rua / Avenida" className={campoClasse} />
        </Campo>
        <Campo label="Número" className="w-24">
          <input value={valor.numero} onChange={(e) => onChange({ ...valor, numero: e.target.value })} placeholder="123" className={campoClasse} />
        </Campo>
      </div>

      <Campo label="Complemento (opcional)">
        <input value={valor.complemento} onChange={(e) => onChange({ ...valor, complemento: e.target.value })} placeholder="Apto, bloco, ponto de referência..." className={campoClasse} />
      </Campo>

      <Campo label="Bairro">
        <input value={valor.bairro} onChange={(e) => onChange({ ...valor, bairro: e.target.value })} className={campoClasse} />
      </Campo>

      <div className="flex gap-2">
        <Campo label="Cidade" className="flex-[2]">
          <input value={valor.cidade} onChange={(e) => onChange({ ...valor, cidade: e.target.value })} className={campoClasse} />
        </Campo>
        <Campo label="Estado" className="w-24">
          <select value={valor.estado} onChange={(e) => onChange({ ...valor, estado: e.target.value })} className={campoClasse}>
            <option value="">--</option>
            {ESTADOS.map((uf) => <option key={uf} value={uf}>{uf}</option>)}
          </select>
        </Campo>
      </div>
    </div>
  );
}
