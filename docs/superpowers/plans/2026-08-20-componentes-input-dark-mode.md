# Componentes de Input Sofisticados + Modo Noturno — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Construir 13 componentes de input reutilizáveis (padrão shadcn/Base UI) e o modo noturno do admin, aplicando cada componente a pelo menos um campo real do sistema.

**Architecture:** Cada componente novo vive em `frontend/src/components/ui/`, usa o primitivo Base UI certo por baixo (`@base-ui/react/input`, `/field`, `/switch`, `/number-field`, `/toggle-group`) e só classes Tailwind da camada de cor de produto (`bg-fundo`, `text-tinta`, `border-tinta/20`, `bg-acento` etc.) — nunca cor fixa. O modo noturno redefine essas mesmas variáveis CSS dentro de `.dark`, escopado por um wrapper na raiz do layout do admin (`Dashboard.tsx`), controlado por um store Zustand novo.

**Tech Stack:** React 19, TypeScript, Vite, Tailwind v3, `@base-ui/react` 1.6 (já instalado), `class-variance-authority`, `clsx`+`tailwind-merge` (via `cn()`), `lucide-react` (ícones), Zustand 5 (`persist` middleware). Nenhuma dependência nova — tudo já está no `package.json`.

## Global Constraints

- Spec de referência: `docs/superpowers/specs/2026-08-20-componentes-input-dark-mode-design.md` — ler antes de implementar qualquer task.
- **Sem suíte de teste de componente React no projeto** (confirmado: só o backend Go tem testes automatizados). Por isso, o ciclo "escreve teste falhando → roda → implementa → roda passando" do padrão TDD não se aplica aqui — cada task termina com `npx tsc -b` limpo (substituindo "rodar os testes") e uma nota do que verificar visualmente no navegador (substituindo "verificar que o teste passa"). Isso é uma adaptação deliberada ao que a spec já define como validação ("sem suíte de teste... validação manual no navegador"), não uma omissão.
- Só classes Tailwind da paleta de produto (`fundo`, `superficie`, `tinta`, `tinta-suave`, `acento`, `douro`) nos componentes novos — nunca hex fixo, nunca os tokens shadcn (`bg-primary`, `text-foreground` etc., usados só em `Planos.tsx`/`MeuPlano.tsx`, fora do escopo aqui).
- Import de ícones sempre de `lucide-react` (já instalado, já usado no projeto).
- Fontes: `font-carimbo` (IBM Plex Mono) em qualquer valor monetário/numérico; `font-display` (Anton) não se aplica a nenhum componente desta lista (é só pra títulos). Nenhuma fonte nova.
- `cn()` importado de `@/lib/utils` nos componentes de `components/ui/` (padrão dos arquivos existentes ali, que já usam o alias `@/`); nos arquivos de `pages/`/`components/` fora de `ui/`, seguir o padrão relativo já usado nesse arquivo específico (ex: `Dashboard.tsx` usa `'../../lib/utils'`, não `@/`).
- Cada task, ao final: `cd frontend && npx tsc -b` deve sair limpo (sem erros) antes do commit.
- Mensagens de commit em português, seguindo o estilo já usado no repo (presente, descrevendo o que mudou e por quê quando não for óbvio).

---

### Task 1: `input.tsx` + `field.tsx` — base de todos os outros componentes

**Files:**
- Create: `frontend/src/components/ui/field.tsx`
- Create: `frontend/src/components/ui/input.tsx`

**Interfaces:**
- Produces: `Field`, `FieldLabel`, `FieldDescription`, `FieldError` (de `field.tsx`); `Input`, `inputVariants` (de `input.tsx`, com `status?: 'default' | 'success' | 'error'`).

- [x] **Passo 1: Criar `field.tsx`**

```tsx
import { Field as FieldPrimitive } from "@base-ui/react/field"

import { cn } from "@/lib/utils"

function Field({ className, ...props }: FieldPrimitive.Root.Props) {
  return (
    <FieldPrimitive.Root
      data-slot="field"
      className={cn("flex flex-col gap-1", className)}
      {...props}
    />
  )
}

function FieldLabel({ className, ...props }: FieldPrimitive.Label.Props) {
  return (
    <FieldPrimitive.Label
      data-slot="field-label"
      className={cn(
        "text-xs font-medium uppercase tracking-wide text-tinta-suave",
        className
      )}
      {...props}
    />
  )
}

function FieldDescription({ className, ...props }: FieldPrimitive.Description.Props) {
  return (
    <FieldPrimitive.Description
      data-slot="field-description"
      className={cn("text-xs text-tinta-suave", className)}
      {...props}
    />
  )
}

function FieldError({ className, ...props }: FieldPrimitive.Error.Props) {
  return (
    <FieldPrimitive.Error
      data-slot="field-error"
      className={cn("text-xs text-acento", className)}
      {...props}
    />
  )
}

export { Field, FieldLabel, FieldDescription, FieldError }
```

- [x] **Passo 2: Criar `input.tsx`**

```tsx
import { Input as InputPrimitive } from "@base-ui/react/input"
import { cva, type VariantProps } from "class-variance-authority"
import { Check, X } from "lucide-react"

import { cn } from "@/lib/utils"

const inputVariants = cva(
  "w-full rounded-lg border bg-fundo px-3 py-2 text-sm text-tinta outline-none transition placeholder:text-tinta-suave/70 disabled:cursor-not-allowed disabled:opacity-60",
  {
    variants: {
      status: {
        default: "border-tinta/20 focus:border-acento",
        success: "border-emerald-500 bg-emerald-500/5 focus:border-emerald-500",
        error: "border-acento bg-acento/5 focus:border-acento",
      },
    },
    defaultVariants: {
      status: "default",
    },
  }
)

export interface InputProps
  extends Omit<InputPrimitive.Props, "className">,
    VariantProps<typeof inputVariants> {
  className?: string
}

function Input({ className, status, ...props }: InputProps) {
  const temIcone = status === "success" || status === "error"
  return (
    <div className="relative">
      <InputPrimitive
        data-slot="input"
        className={cn(inputVariants({ status }), temIcone && "pr-9", className)}
        {...props}
      />
      {status === "success" && (
        <Check className="pointer-events-none absolute right-3 top-1/2 size-4 -translate-y-1/2 text-emerald-500" />
      )}
      {status === "error" && (
        <X className="pointer-events-none absolute right-3 top-1/2 size-4 -translate-y-1/2 text-acento" />
      )}
    </div>
  )
}

export { Input, inputVariants }
```

- [x] **Passo 3: Validar**

Rodar: `cd frontend && npx tsc -b`
Esperado: sem erros. `Input`/`Field` ainda não são usados em lugar nenhum nessa task — normal não ter nada visual pra conferir ainda.

- [x] **Passo 4: Commit**

```bash
git add frontend/src/components/ui/field.tsx frontend/src/components/ui/input.tsx
git commit -m "feat: base Field/Input (Base UI) pra biblioteca de componentes de formulário"
```

---

### Task 2: Componente A (`input-price.tsx`) — aplicado em 3 campos de preço reais

**Files:**
- Create: `frontend/src/components/ui/input-price.tsx`
- Modify: `frontend/src/components/admin/ProdutoFormFields.tsx:43-45`
- Modify: `frontend/src/pages/admin/Combos.tsx` (campo "Preço final do {rotuloMin} (R$)")
- Modify: `frontend/src/pages/admin/Cupons.tsx:163-173`

**Interfaces:**
- Consumes: `inputVariants` de `./input` (Task 1).
- Produces: `InputPrice` (aceita as mesmas props de um `<input>` nativo, mais `type`/`step`/`min` já com default).

- [x] **Passo 1: Criar `input-price.tsx`**

```tsx
import { Input as InputPrimitive } from "@base-ui/react/input"
import type { ComponentProps } from "react"

import { cn } from "@/lib/utils"
import { inputVariants } from "./input"

function InputPrice({ className, ...props }: ComponentProps<typeof InputPrimitive>) {
  return (
    <div className="relative">
      <span className="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 font-carimbo text-sm font-semibold text-tinta-suave">
        R$
      </span>
      <InputPrimitive
        type="number"
        step="0.01"
        min="0"
        data-slot="input-price"
        className={cn(inputVariants({ status: "default" }), "pl-10 font-carimbo", className)}
        {...props}
      />
    </div>
  )
}

export { InputPrice }
```

- [x] **Passo 2: Aplicar em `ProdutoFormFields.tsx`**

Trocar (linhas 43-45):
```tsx
<Campo label="Preço (R$)" className="flex-1">
  <input type="number" step="0.01" min="0.01" required value={form.preco || ''} onChange={(e) => onChange({ ...form, preco: parseFloat(e.target.value) || 0 })} className="w-full rounded-lg border border-tinta/20 bg-fundo px-3 py-2 text-tinta outline-none focus:border-acento" />
</Campo>
```
por:
```tsx
<Campo label="Preço" className="flex-1">
  <InputPrice min="0.01" required value={form.preco || ''} onChange={(e) => onChange({ ...form, preco: parseFloat(e.target.value) || 0 })} />
</Campo>
```
Adicionar `import { InputPrice } from '../ui/input-price';` no topo do arquivo (junto dos outros imports relativos já existentes).

- [x] **Passo 3: Aplicar em `Combos.tsx`**

Achar o campo `<Campo label={`Preço final do ${rotuloMin} (R$)`}>` (por volta da linha 190-199) e trocar o `<input>` interno pelo mesmo padrão do Passo 2 — `<InputPrice min="0.01" required value={form.preco || ''} onChange={(e) => setForm({ ...form, preco: arredondarCentavos(parseFloat(e.target.value) || 0) })} />`, ajustando o label pra `Preço final do ${rotuloMin}` (sem o "(R$)" redundante). Adicionar o import de `InputPrice`.

- [x] **Passo 4: Aplicar em `Cupons.tsx`**

Trocar (linhas 163-173):
```tsx
<Campo label="Pedido mínimo pra usar (R$)">
  <input
    type="number"
    step="0.50"
    min="0"
    value={form.valor_minimo_pedido || ''}
    onChange={(e) => setForm({ ...form, valor_minimo_pedido: parseFloat(e.target.value) || 0 })}
    placeholder="Sem mínimo"
    className="w-full rounded-lg border border-tinta/20 bg-fundo px-3 py-2 text-tinta outline-none focus:border-acento"
  />
</Campo>
```
por:
```tsx
<Campo label="Pedido mínimo pra usar">
  <InputPrice
    step="0.50"
    value={form.valor_minimo_pedido || ''}
    onChange={(e) => setForm({ ...form, valor_minimo_pedido: parseFloat(e.target.value) || 0 })}
    placeholder="Sem mínimo"
  />
</Campo>
```
Adicionar o import.

- [x] **Passo 5: Validar**

Rodar: `cd frontend && npx tsc -b`
Esperado: sem erros.
Verificação manual no navegador: abrir Produtos (novo produto), Combos (novo combo) e Cupons (novo cupom) — os três campos de preço devem mostrar "R$" fixo dentro do campo, à esquerda, com o número em `font-carimbo`.

- [x] **Passo 6: Commit**

```bash
git add frontend/src/components/ui/input-price.tsx frontend/src/components/admin/ProdutoFormFields.tsx frontend/src/pages/admin/Combos.tsx frontend/src/pages/admin/Cupons.tsx
git commit -m "feat: componente A (preço com prefixo R\$), aplicado em produto/combo/cupom"
```

---

### Task 3: Componente B (`stepper.tsx`) — aplicado no carrinho do cliente

**Files:**
- Create: `frontend/src/components/ui/stepper.tsx`
- Modify: `frontend/src/components/CarrinhoDrawer.tsx:359-377` (item de produto)
- Modify: `frontend/src/components/CarrinhoDrawer.tsx:404-422` (item de combo)

**Interfaces:**
- Consumes: nada de tasks anteriores.
- Produces: `Stepper` (`value: number`, `onValueChange: (v: number) => void`, `min?: number`).

- [x] **Passo 1: Criar `stepper.tsx`**

```tsx
import { NumberField } from "@base-ui/react/number-field"
import { Minus, Plus } from "lucide-react"

import { cn } from "@/lib/utils"

export interface StepperProps {
  value: number
  onValueChange: (value: number) => void
  min?: number
  className?: string
}

function Stepper({ value, onValueChange, min = 0, className }: StepperProps) {
  return (
    <NumberField.Root
      value={value}
      min={min}
      onValueChange={(novoValor) => {
        if (novoValor !== null) onValueChange(novoValor)
      }}
      className={cn("inline-flex", className)}
    >
      <NumberField.Group
        data-slot="stepper"
        className="flex items-center gap-1 rounded-full border border-tinta/15 bg-superficie px-1 py-1"
      >
        <NumberField.Decrement
          aria-label="Diminuir quantidade"
          className="flex size-6 items-center justify-center rounded-full text-tinta transition hover:bg-fundo disabled:opacity-40"
        >
          <Minus className="size-3.5" />
        </NumberField.Decrement>
        <NumberField.Input
          readOnly
          className="w-6 border-none bg-transparent text-center font-carimbo text-sm text-tinta outline-none"
        />
        <NumberField.Increment
          aria-label="Aumentar quantidade"
          className="flex size-6 items-center justify-center rounded-full text-tinta transition hover:bg-fundo disabled:opacity-40"
        >
          <Plus className="size-3.5" />
        </NumberField.Increment>
      </NumberField.Group>
    </NumberField.Root>
  )
}

export { Stepper }
```

- [x] **Passo 2: Aplicar nos dois lugares de `CarrinhoDrawer.tsx`**

Trocar o bloco do item de produto (linhas 359-377):
```tsx
<div className="flex items-center gap-2 rounded-full border border-tinta/15 px-2 py-1">
  <button
    onClick={() => alterarQuantidade(item.produto.id, item.quantidade - 1, item.variacao?.id)}
    className="h-6 w-6 rounded-full text-tinta hover:bg-fundo"
    aria-label="Diminuir quantidade"
  >
    −
  </button>
  <span className="w-5 text-center font-carimbo text-sm">
    {item.quantidade}
  </span>
  <button
    onClick={() => alterarQuantidade(item.produto.id, item.quantidade + 1, item.variacao?.id)}
    className="h-6 w-6 rounded-full text-tinta hover:bg-fundo"
    aria-label="Aumentar quantidade"
  >
    +
  </button>
</div>
```
por:
```tsx
<Stepper
  value={item.quantidade}
  min={1}
  onValueChange={(nova) => alterarQuantidade(item.produto.id, nova, item.variacao?.id)}
/>
```
E o bloco equivalente do item de combo (linhas 404-422) por:
```tsx
<Stepper
  value={item.quantidade}
  min={1}
  onValueChange={(nova) => alterarQuantidadeCombo(item.combo.id, nova)}
/>
```
Adicionar `import { Stepper } from './ui/stepper';` no topo (ajustar o caminho relativo conforme a localização real de `CarrinhoDrawer.tsx` dentro de `components/`).

**Nota de comportamento**: com `min={1}`, o stepper não deixa mais decrementar até 0 (antes disso era tecnicamente possível, calculando `item.quantidade - 1` sem limite inferior). O botão "Remover" ao lado de cada item já existe e continua sendo o caminho pra tirar o item do carrinho — não é uma funcionalidade perdida, só deixou de ser possível zerar via o stepper.

- [x] **Passo 3: Validar**

Rodar: `cd frontend && npx tsc -b`
Esperado: sem erros.
Verificação manual: abrir o carrinho público de uma loja de teste com itens e combos, conferir que +/- funcionam, e que decrementar de 1 não desce pra 0 (botão "−" fica desabilitado/sem efeito nesse ponto).

- [x] **Passo 4: Commit**

```bash
git add frontend/src/components/ui/stepper.tsx frontend/src/components/CarrinhoDrawer.tsx
git commit -m "feat: componente B (stepper de quantidade), aplicado no carrinho"
```

---

### Task 4: Componente C (`input-floating.tsx`) — aplicado no nome do produto

**Files:**
- Create: `frontend/src/components/ui/input-floating.tsx`
- Modify: `frontend/src/components/admin/ProdutoFormFields.tsx:36-38`

**Interfaces:**
- Produces: `InputFloating` (`label: string` + props de `<input>`).

- [x] **Passo 1: Criar `input-floating.tsx`**

```tsx
import { Field as FieldPrimitive } from "@base-ui/react/field"
import { Input as InputPrimitive } from "@base-ui/react/input"
import type { ComponentProps } from "react"

import { cn } from "@/lib/utils"

interface InputFloatingProps extends ComponentProps<typeof InputPrimitive> {
  label: string
  id: string
}

function InputFloating({ label, className, id, ...props }: InputFloatingProps) {
  return (
    <FieldPrimitive.Root className="relative">
      <InputPrimitive
        id={id}
        placeholder=" "
        data-slot="input-floating"
        className={cn(
          "peer w-full rounded-lg border border-tinta/20 bg-fundo px-3 pt-5 pb-1.5 text-sm text-tinta outline-none transition focus:border-acento",
          className
        )}
        {...props}
      />
      <FieldPrimitive.Label
        htmlFor={id}
        className="pointer-events-none absolute left-3 top-4 -translate-y-1/2 bg-fundo px-1 text-sm text-tinta-suave transition-all peer-focus:top-2.5 peer-focus:translate-y-0 peer-focus:text-[11px] peer-focus:font-semibold peer-focus:text-acento peer-[:not(:placeholder-shown)]:top-2.5 peer-[:not(:placeholder-shown)]:translate-y-0 peer-[:not(:placeholder-shown)]:text-[11px] peer-[:not(:placeholder-shown)]:font-semibold peer-[:not(:placeholder-shown)]:text-acento"
      >
        {label}
      </FieldPrimitive.Label>
    </FieldPrimitive.Root>
  )
}

export { InputFloating }
```

- [x] **Passo 2: Aplicar em `ProdutoFormFields.tsx`**

Trocar (linhas 36-38):
```tsx
<Campo label="Nome">
  <input required value={form.nome} onChange={(e) => onChange({ ...form, nome: e.target.value })} className="w-full rounded-lg border border-tinta/20 bg-fundo px-3 py-2 text-tinta outline-none focus:border-acento" />
</Campo>
```
por (sem o wrapper `<Campo>` — o label já vem embutido no componente):
```tsx
<InputFloating id="produto-nome" label="Nome do produto" required value={form.nome} onValueChange={(nome) => onChange({ ...form, nome })} />
```
Adicionar o import.

- [x] **Passo 3: Validar**

Rodar: `cd frontend && npx tsc -b`
Esperado: sem erros.
Verificação manual: abrir "Novo produto" — o campo "Nome do produto" deve mostrar o label dentro do campo (grande, cinza) quando vazio, e subir/encolher (pequeno, cor de destaque) ao focar ou digitar.

- [x] **Passo 4: Commit**

```bash
git add frontend/src/components/ui/input-floating.tsx frontend/src/components/admin/ProdutoFormFields.tsx
git commit -m "feat: componente C (label flutuante), aplicado no nome do produto"
```

---

### Task 5: Componente D (`input-search.tsx`) — busca funcional em Produtos.tsx

**Files:**
- Create: `frontend/src/components/ui/input-search.tsx`
- Modify: `frontend/src/pages/admin/Produtos.tsx` (novo estado `busca` + filtro + inserção do campo)

**Interfaces:**
- Produces: `InputSearch` (`value: string`, `onValueChange: (v: string) => void`, `placeholder?: string`).

- [x] **Passo 1: Criar `input-search.tsx`**

```tsx
import { Input as InputPrimitive } from "@base-ui/react/input"
import { Search, X } from "lucide-react"

import { cn } from "@/lib/utils"

export interface InputSearchProps {
  value: string
  onValueChange: (value: string) => void
  placeholder?: string
  className?: string
}

function InputSearch({ value, onValueChange, placeholder = "Buscar...", className }: InputSearchProps) {
  return (
    <div className={cn("relative", className)}>
      <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-tinta-suave" />
      <InputPrimitive
        value={value}
        onValueChange={onValueChange}
        placeholder={placeholder}
        data-slot="input-search"
        className="w-full rounded-full border border-tinta/20 bg-fundo py-2 pl-9 pr-9 text-sm text-tinta outline-none transition focus:border-acento"
      />
      {value.length > 0 && (
        <button
          type="button"
          onClick={() => onValueChange("")}
          aria-label="Limpar busca"
          className="absolute right-2.5 top-1/2 flex size-5 -translate-y-1/2 items-center justify-center rounded-full bg-tinta/10 text-tinta-suave transition hover:bg-tinta/20"
        >
          <X className="size-3" />
        </button>
      )}
    </div>
  )
}

export { InputSearch }
```

- [x] **Passo 2: Aplicar em `Produtos.tsx`**

Adicionar estado novo, logo junto dos outros `useState` já existentes no topo de `Produtos()`:
```tsx
const [busca, setBusca] = useState('');
```

Adicionar a derivação da lista filtrada, logo antes do `return (`:
```tsx
const produtosVisiveis = busca.trim()
  ? produtos?.filter((p) => p.nome.toLowerCase().includes(busca.trim().toLowerCase()))
  : produtos;
```

No JSX, logo abaixo do bloco `<div className="flex items-center justify-between">...</div>` (linhas 478-495) e antes do bloco `{mostrarCadastroEmMassa && (...)}`, inserir (só quando não estiver com o form aberto, mesma condição já usada pros botões):
```tsx
{!mostrarForm && (
  <InputSearch value={busca} onValueChange={setBusca} placeholder="Buscar produto..." className="max-w-sm" />
)}
```

Trocar as duas ocorrências de `produtos` pela variável filtrada na renderização da lista (linhas 524 e 526-527):
```tsx
) : produtosVisiveis && produtosVisiveis.length > 0 ? (
  <div className="space-y-6">
    {categorias?.map((categoria) => {
      const produtosDaCategoria = produtosVisiveis.filter((p) => p.categoria_id === categoria.id);
```
Adicionar `import { InputSearch } from '../../components/ui/input-search';`.

- [x] **Passo 3: Validar**

Rodar: `cd frontend && npx tsc -b`
Esperado: sem erros.
Verificação manual: abrir Produtos, digitar parte do nome de um produto existente — a lista deve filtrar em tempo real, agrupada por categoria como já era; o "x" só aparece com texto digitado e limpa a busca ao clicar.

- [x] **Passo 4: Commit**

```bash
git add frontend/src/components/ui/input-search.tsx frontend/src/pages/admin/Produtos.tsx
git commit -m "feat: componente D (busca com limpar), campo novo de busca em Produtos"
```

---

### Task 6: Componente E (`switch.tsx`) — aplicado no "Disponível" do produto

**Files:**
- Create: `frontend/src/components/ui/switch.tsx`
- Modify: `frontend/src/components/admin/ProdutoFormFields.tsx:128-131`

**Interfaces:**
- Produces: `Switch` (`checked`, `onCheckedChange`, mesma API do Base UI `Switch.Root`).

- [x] **Passo 1: Criar `switch.tsx`**

```tsx
import { Switch as SwitchPrimitive } from "@base-ui/react/switch"

import { cn } from "@/lib/utils"

function Switch({ className, ...props }: SwitchPrimitive.Root.Props) {
  return (
    <SwitchPrimitive.Root
      data-slot="switch"
      className={cn(
        "relative inline-flex h-6 w-11 shrink-0 items-center rounded-full bg-tinta/20 transition-colors data-[checked]:bg-acento disabled:cursor-not-allowed disabled:opacity-50",
        className
      )}
      {...props}
    >
      <SwitchPrimitive.Thumb
        data-slot="switch-thumb"
        className="block size-5 translate-x-0.5 rounded-full bg-superficie shadow transition-transform data-[checked]:translate-x-[22px]"
      />
    </SwitchPrimitive.Root>
  )
}

export { Switch }
```

- [x] **Passo 2: Aplicar em `ProdutoFormFields.tsx`**

Trocar (linhas 128-131):
```tsx
<label className="flex items-center gap-2 text-sm text-tinta">
  <input type="checkbox" checked={form.disponivel} onChange={(e) => onChange({ ...form, disponivel: e.target.checked })} className="h-4 w-4 accent-acento" />
  Disponível no {rotuloCatalogo(segmentoLoja)}
</label>
```
por:
```tsx
<div className="flex items-center gap-2">
  <Switch checked={form.disponivel} onCheckedChange={(disponivel) => onChange({ ...form, disponivel })} />
  <span className="text-sm text-tinta">Disponível no {rotuloCatalogo(segmentoLoja)}</span>
</div>
```
Adicionar o import.

- [x] **Passo 3: Validar**

Rodar: `cd frontend && npx tsc -b`
Esperado: sem erros.
Verificação manual: abrir edição de um produto, alternar "Disponível" — o switch desliza e muda de cor (cinza↔terracota).

- [x] **Passo 4: Commit**

```bash
git add frontend/src/components/ui/switch.tsx frontend/src/components/admin/ProdutoFormFields.tsx
git commit -m "feat: componente E (toggle switch), aplicado em Disponível do produto"
```

---

### Task 7: Componente F (`segmented.tsx`) — aplicado no modo de entrega do carrinho

**Files:**
- Create: `frontend/src/components/ui/segmented.tsx`
- Modify: `frontend/src/components/CarrinhoDrawer.tsx:461-474`

**Interfaces:**
- Produces: `Segmented<T>` (`opcoes: {valor: T, label: string}[]`, `valor: T`, `onValorChange: (v: T) => void`).

- [x] **Passo 1: Criar `segmented.tsx`**

```tsx
import { Toggle } from "@base-ui/react/toggle"
import { ToggleGroup } from "@base-ui/react/toggle-group"

import { cn } from "@/lib/utils"

export interface SegmentedOption<T extends string> {
  valor: T
  label: string
}

interface SegmentedProps<T extends string> {
  opcoes: SegmentedOption<T>[]
  valor: T
  onValorChange: (valor: T) => void
  className?: string
}

function Segmented<T extends string>({ opcoes, valor, onValorChange, className }: SegmentedProps<T>) {
  return (
    <ToggleGroup
      value={[valor]}
      onValueChange={(novoValor) => {
        const escolhido = novoValor[novoValor.length - 1]
        if (escolhido) onValorChange(escolhido as T)
      }}
      data-slot="segmented"
      className={cn("inline-flex gap-1 rounded-full border border-tinta/15 bg-fundo p-1", className)}
    >
      {opcoes.map((opcao) => (
        <Toggle
          key={opcao.valor}
          value={opcao.valor}
          className="flex-1 rounded-full px-4 py-1.5 text-sm font-semibold text-tinta-suave transition data-[pressed]:bg-acento data-[pressed]:text-superficie"
        >
          {opcao.label}
        </Toggle>
      ))}
    </ToggleGroup>
  )
}

export { Segmented }
```

- [x] **Passo 2: Aplicar em `CarrinhoDrawer.tsx`**

Trocar (linhas 461-474):
```tsx
{opcoesModoEntrega.length > 1 && (
  <div className="flex gap-2">
    {opcoesModoEntrega.map((opcao) => (
      <button
        key={opcao.valor}
        type="button"
        onClick={() => setModoEntrega(opcao.valor)}
        className={`flex-1 rounded-full border-2 py-2 text-sm font-semibold transition ${modoEntrega === opcao.valor ? 'border-acento bg-acento text-superficie' : 'border-tinta/20 text-tinta'}`}
      >
        {opcao.label}
      </button>
    ))}
  </div>
)}
```
por:
```tsx
{opcoesModoEntrega.length > 1 && (
  <Segmented opcoes={opcoesModoEntrega} valor={modoEntrega} onValorChange={setModoEntrega} className="w-full" />
)}
```
Adicionar o import. `opcoesModoEntrega` já é tipado como `{ valor: 'retirada' | 'entrega' | 'guardar'; label: string }[]`, compatível com `SegmentedOption<'retirada' | 'entrega' | 'guardar'>` sem mudança de tipo.

- [x] **Passo 3: Validar**

Rodar: `cd frontend && npx tsc -b`
Esperado: sem erros.
Verificação manual: abrir o carrinho público na etapa de dados — o segmentado Entrega/Retirada (e Guardar quando aplicável) deve alternar com pill preenchida no selecionado.

- [x] **Passo 4: Commit**

```bash
git add frontend/src/components/ui/segmented.tsx frontend/src/components/CarrinhoDrawer.tsx
git commit -m "feat: componente F (segmentado), aplicado no modo de entrega do carrinho"
```

---

### Task 8: Componente G/L (validação inline) — telefone e CEP

**Files:**
- Modify: `frontend/src/pages/admin/Configuracoes.tsx` (campo WhatsApp, ~linha 300-309)
- Modify: `frontend/src/components/EnderecoCampos.tsx:106-117` (campo CEP)

**Interfaces:**
- Consumes: `Input` + `status` prop de `input.tsx` (Task 1) — não cria arquivo novo, G e L reaproveitam o que já existe.

- [x] **Passo 1: Aplicar em `Configuracoes.tsx` (telefone)**

Trocar (linhas ~300-309):
```tsx
<Campo label="WhatsApp pra receber avisos de pedido">
  <input
    required
    value={whatsapp}
    onChange={(e) => setWhatsapp(e.target.value)}
    placeholder="5579999999999"
    className="w-full rounded-lg border border-tinta/20 bg-fundo px-3 py-2 text-tinta outline-none focus:border-acento"
  />
  <span className="mt-1 block text-xs text-tinta-suave">DDI + DDD + número (ex: 5579999999999).</span>
</Campo>
```
por:
```tsx
<Campo label="WhatsApp pra receber avisos de pedido">
  <Input
    required
    status={whatsapp.length === 0 ? 'default' : /^\d{12,13}$/.test(whatsapp) ? 'success' : 'error'}
    value={whatsapp}
    onChange={(e) => setWhatsapp(e.target.value)}
    placeholder="5579999999999"
  />
  <span className="mt-1 block text-xs text-tinta-suave">DDI + DDD + número (ex: 5579999999999).</span>
</Campo>
```
Adicionar `import { Input } from '../../components/ui/input';` (ajustar caminho relativo conforme localização real de `Configuracoes.tsx`).

- [x] **Passo 2: Aplicar em `EnderecoCampos.tsx` (CEP)**

Trocar (linhas 106-117):
```tsx
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
```
por:
```tsx
<Campo label="CEP (opcional — se souber, preenche o resto sozinho)">
  <Input
    status={erroCep ? 'error' : valor.cep.replace(/\D/g, '').length === 8 && valor.rua.trim() !== '' ? 'success' : 'default'}
    value={valor.cep}
    onChange={(e) => handleCepChange(e.target.value)}
    placeholder="49000-000"
    maxLength={9}
    inputMode="numeric"
  />
  {buscandoCep && <span className="mt-1 block text-xs text-tinta-suave">Buscando endereço pelo CEP...</span>}
  {erroCep && <span className="mt-1 block text-xs text-acento">{erroCep}</span>}
</Campo>
```
Adicionar `import { Input } from './ui/input';` e remover `campoClasse` da chamada (o componente `Input` já tem seu próprio estilo — pode manter a constante `campoClasse` se outros campos do mesmo arquivo ainda a usam, só não passar ela pro `Input` novo).

- [x] **Passo 3: Validar**

Rodar: `cd frontend && npx tsc -b`
Esperado: sem erros.
Verificação manual: em Configurações, digitar um número de WhatsApp incompleto (borda vermelha + ícone X) e depois completo (borda verde + ícone check). No endereço da loja, digitar um CEP válido (borda verde após a busca preencher o resto) e um CEP inexistente (borda vermelha, mensagem de erro já existente).

- [x] **Passo 4: Commit**

```bash
git add frontend/src/pages/admin/Configuracoes.tsx frontend/src/components/EnderecoCampos.tsx
git commit -m "feat: componentes G/L (validação inline), aplicados em telefone e CEP"
```

---

### Task 9: Componente H (`textarea.tsx`) — aplicado na descrição do produto

**Files:**
- Create: `frontend/src/components/ui/textarea.tsx`
- Modify: `frontend/src/components/admin/ProdutoFormFields.tsx:39-41`

**Interfaces:**
- Produces: `Textarea` (`value: string`, `maxLength: number`, + demais props de `<textarea>`).

- [x] **Passo 1: Criar `textarea.tsx`**

```tsx
import type { ComponentProps } from "react"

import { cn } from "@/lib/utils"

interface TextareaProps extends ComponentProps<"textarea"> {
  maxLength: number
  value: string
}

function Textarea({ className, maxLength, value, ...props }: TextareaProps) {
  const restantes = maxLength - value.length
  return (
    <div className="relative">
      <textarea
        value={value}
        maxLength={maxLength}
        data-slot="textarea"
        className={cn(
          "min-h-[90px] w-full resize-y rounded-lg border border-tinta/20 bg-fundo px-3 py-2 pb-6 text-sm text-tinta outline-none transition focus:border-acento",
          className
        )}
        {...props}
      />
      <span className="pointer-events-none absolute bottom-2 right-3 rounded bg-fundo px-1 font-carimbo text-[11px] text-tinta-suave">
        {restantes} restantes
      </span>
    </div>
  )
}

export { Textarea }
```

- [x] **Passo 2: Aplicar em `ProdutoFormFields.tsx`**

Trocar (linhas 39-41):
```tsx
<Campo label="Descrição">
  <textarea value={form.descricao} onChange={(e) => onChange({ ...form, descricao: e.target.value })} rows={2} className="w-full rounded-lg border border-tinta/20 bg-fundo px-3 py-2 text-tinta outline-none focus:border-acento" />
</Campo>
```
por:
```tsx
<Campo label="Descrição">
  <Textarea maxLength={160} value={form.descricao} onChange={(e) => onChange({ ...form, descricao: e.target.value })} />
</Campo>
```
Adicionar o import. Limite de 160 caracteres escolhido como tamanho razoável pra descrição de item de cardápio/catálogo — se algum produto existente já tiver descrição mais longa que isso, o `maxLength` do HTML não apaga texto já salvo, só impede digitar além do limite dali em diante.

- [x] **Passo 3: Validar**

Rodar: `cd frontend && npx tsc -b`
Esperado: sem erros.
Verificação manual: abrir edição de produto, digitar na descrição — contador de caracteres restantes deve descer em tempo real.

- [x] **Passo 4: Commit**

```bash
git add frontend/src/components/ui/textarea.tsx frontend/src/components/admin/ProdutoFormFields.tsx
git commit -m "feat: componente H (textarea com contador), aplicado na descrição do produto"
```

---

### Task 10: Componente I (`dropzone.tsx`) — aplicado na foto do produto

**Files:**
- Create: `frontend/src/components/ui/dropzone.tsx`
- Modify: `frontend/src/components/admin/ProdutoFormFields.tsx:105-126`

**Interfaces:**
- Produces: `Dropzone` (`onFileChange: (e: ChangeEvent<HTMLInputElement>) => void`, `previewUrl?`, `disabled?`, `texto?`) — reaproveita a MESMA assinatura de evento que `onSelecionarFoto` já usa hoje, pra não precisar mudar nada em `Produtos.tsx`/`CadastroEmMassaDialog.tsx` (que passam esse handler).

- [x] **Passo 1: Criar `dropzone.tsx`**

```tsx
import { useRef, useState, type ChangeEvent, type DragEvent } from "react"
import { Camera } from "lucide-react"

import { cn } from "@/lib/utils"

interface DropzoneProps {
  onFileChange: (e: ChangeEvent<HTMLInputElement>) => void
  previewUrl?: string
  className?: string
  disabled?: boolean
  texto?: string
}

function Dropzone({ onFileChange, previewUrl, className, disabled, texto = "Arraste a foto ou clique pra escolher" }: DropzoneProps) {
  const [sobre, setSobre] = useState(false)
  const inputRef = useRef<HTMLInputElement>(null)

  function onDrop(e: DragEvent<HTMLDivElement>) {
    e.preventDefault()
    setSobre(false)
    const arquivo = e.dataTransfer.files?.[0]
    if (!arquivo || !inputRef.current) return
    // Reconstrói um FileList no input escondido e dispara um evento
    // "change" nativo nele, pra reaproveitar o MESMO handler que já
    // processa o upload via seleção manual de arquivo — evita duplicar a
    // lógica de envio pra Cloudinary em dois lugares.
    const dt = new DataTransfer()
    dt.items.add(arquivo)
    inputRef.current.files = dt.files
    inputRef.current.dispatchEvent(new Event("change", { bubbles: true }))
  }

  return (
    <div
      role="button"
      tabIndex={0}
      onClick={() => !disabled && inputRef.current?.click()}
      onKeyDown={(e) => {
        if (e.key === "Enter" || e.key === " ") inputRef.current?.click()
      }}
      onDragOver={(e) => {
        e.preventDefault()
        setSobre(true)
      }}
      onDragLeave={() => setSobre(false)}
      onDrop={onDrop}
      data-slot="dropzone"
      className={cn(
        "flex cursor-pointer flex-col items-center gap-2 rounded-xl border-2 border-dashed border-tinta/20 bg-fundo px-6 py-8 text-center transition",
        sobre && "border-acento bg-acento/5",
        disabled && "cursor-not-allowed opacity-60",
        className
      )}
    >
      {previewUrl ? (
        <img src={previewUrl} alt="Prévia" className="size-16 rounded-full object-cover" />
      ) : (
        <Camera className="size-6 text-tinta-suave" />
      )}
      <p className="text-sm text-tinta-suave">{texto}</p>
      <input ref={inputRef} type="file" accept="image/*" onChange={onFileChange} disabled={disabled} className="hidden" />
    </div>
  )
}

export { Dropzone }
```

- [x] **Passo 2: Aplicar em `ProdutoFormFields.tsx`**

Trocar o bloco de upload de foto (linhas 105-126):
```tsx
<div>
  <span className="mb-2 block text-xs font-medium uppercase tracking-wide text-tinta-suave">Foto (opcional)</span>
  <div className="flex items-center gap-4">
    <div className="flex h-16 w-16 shrink-0 items-center justify-center overflow-hidden rounded-full border-2 border-dashed border-tinta/25 bg-fundo">
      {form.foto_url ? <img src={logoMiniatura(form.foto_url)} alt="Foto" className="h-full w-full object-cover" /> : <span className="font-display text-xl text-tinta/30">{form.nome.charAt(0).toUpperCase() || '?'}</span>}
    </div>
    <label className="cursor-pointer btn-neu-secundario hover:border-acento">
      {enviandoFoto ? 'Enviando...' : form.foto_url ? 'Trocar foto' : 'Enviar foto'}
      <input type="file" accept="image/*" onChange={onSelecionarFoto} disabled={enviandoFoto} className="hidden" />
    </label>
    {form.foto_url && !enviandoFoto && (
      <button
        type="button"
        onClick={() => onChange({ ...form, foto_url: '' })}
        className="btn-neu-secundario-suave hover:border-acento hover:text-acento"
      >
        Remover foto
      </button>
    )}
  </div>
</div>
```
por:
```tsx
<div>
  <span className="mb-2 block text-xs font-medium uppercase tracking-wide text-tinta-suave">Foto (opcional)</span>
  <Dropzone
    onFileChange={onSelecionarFoto}
    previewUrl={form.foto_url ? logoMiniatura(form.foto_url) : undefined}
    disabled={enviandoFoto}
    texto={enviandoFoto ? 'Enviando...' : form.foto_url ? 'Trocar foto (ou arraste uma nova)' : 'Arraste a foto ou clique pra escolher'}
  />
  {form.foto_url && !enviandoFoto && (
    <button
      type="button"
      onClick={() => onChange({ ...form, foto_url: '' })}
      className="btn-neu-secundario-suave mt-2 hover:border-acento hover:text-acento"
    >
      Remover foto
    </button>
  )}
</div>
```
Adicionar o import. `onSelecionarFoto` continua com a mesma assinatura que já vinha de `Produtos.tsx`/`CadastroEmMassaDialog.tsx` — nenhuma mudança precisa nesses dois arquivos.

- [x] **Passo 3: Validar**

Rodar: `cd frontend && npx tsc -b`
Esperado: sem erros.
Verificação manual: abrir "Novo produto", clicar na área de foto (deve abrir o seletor de arquivo do sistema) e também testar arrastar um arquivo de imagem do explorador de arquivos até a área — os dois caminhos devem disparar o mesmo upload.

- [x] **Passo 4: Commit**

```bash
git add frontend/src/components/ui/dropzone.tsx frontend/src/components/admin/ProdutoFormFields.tsx
git commit -m "feat: componente I (dropzone), aplicado na foto do produto"
```

---

### Task 11: Componente J (`input-date.tsx`) — aplicado na validade do cupom

**Files:**
- Create: `frontend/src/components/ui/input-date.tsx`
- Modify: `frontend/src/pages/admin/Cupons.tsx:143-150`

**Interfaces:**
- Produces: `InputDate` (props de `<input type="date">`).

- [ ] **Passo 1: Criar `input-date.tsx`**

```tsx
import { useRef, type ComponentProps } from "react"
import { Calendar } from "lucide-react"

import { cn } from "@/lib/utils"

function InputDate({ className, ...props }: ComponentProps<"input">) {
  const ref = useRef<HTMLInputElement>(null)

  function abrirSeletor() {
    // showPicker() garante que clicar em QUALQUER parte do campo abra o
    // calendário — o comportamento nativo do <input type="date"> só abre
    // isso clicando exatamente no ícone embutido, dependendo do navegador.
    ref.current?.showPicker?.()
  }

  return (
    <div
      onClick={abrirSeletor}
      data-slot="input-date"
      className={cn(
        "flex cursor-pointer items-center gap-2 rounded-lg border border-tinta/20 bg-fundo px-3 py-2 transition focus-within:border-acento",
        className
      )}
    >
      <Calendar className="size-4 shrink-0 text-tinta-suave" />
      <input
        ref={ref}
        type="date"
        className="w-full cursor-pointer border-none bg-transparent font-carimbo text-sm text-tinta outline-none [&::-webkit-calendar-picker-indicator]:hidden"
        {...props}
      />
    </div>
  )
}

export { InputDate }
```

- [ ] **Passo 2: Aplicar em `Cupons.tsx`**

Trocar (linhas 143-150):
```tsx
<Campo label="Validade (opcional)">
  <input
    type="date"
    value={form.validade ?? ''}
    onChange={(e) => setForm({ ...form, validade: e.target.value || null })}
    className="w-full rounded-lg border border-tinta/20 bg-fundo px-3 py-2 text-tinta outline-none focus:border-acento"
  />
</Campo>
```
por:
```tsx
<Campo label="Validade (opcional)">
  <InputDate value={form.validade ?? ''} onChange={(e) => setForm({ ...form, validade: e.target.value || null })} />
</Campo>
```
Adicionar o import.

- [ ] **Passo 3: Validar**

Rodar: `cd frontend && npx tsc -b`
Esperado: sem erros.
Verificação manual: abrir "Novo cupom", clicar em qualquer parte do campo de validade (não só no ícone) — o seletor de data do navegador deve abrir.

- [ ] **Passo 4: Commit**

```bash
git add frontend/src/components/ui/input-date.tsx frontend/src/pages/admin/Cupons.tsx
git commit -m "feat: componente J (campo de data com showPicker no clique), aplicado na validade do cupom"
```

---

### Task 12: Componente K (`theme-picker.tsx`) — substitui o seletor de tema em Configurações

**Files:**
- Create: `frontend/src/components/ui/theme-picker.tsx`
- Modify: `frontend/src/pages/admin/Configuracoes.tsx:619-646` (bloco do seletor de tema)

**Interfaces:**
- Consumes: `Tema` de `frontend/src/themes.ts` (`{id, nome, descricao, acento, fundo, superficie}`).
- Produces: `ThemePicker` (`temas: Tema[]`, `valor: string`, `onValorChange: (id: string) => void`).

- [ ] **Passo 1: Criar `theme-picker.tsx`**

```tsx
import type { Tema } from "@/themes"
import { cn } from "@/lib/utils"

interface ThemePickerProps {
  temas: Tema[]
  valor: string
  onValorChange: (id: string) => void
  className?: string
}

function ThemePicker({ temas, valor, onValorChange, className }: ThemePickerProps) {
  const temaAtual = temas.find((t) => t.id === valor)
  return (
    <div className={cn("space-y-2", className)}>
      <div className="flex flex-wrap gap-2">
        {temas.map((tema) => (
          <button
            key={tema.id}
            type="button"
            onClick={() => onValorChange(tema.id)}
            aria-label={tema.nome}
            aria-pressed={valor === tema.id}
            data-slot="theme-dot"
            className={cn(
              "size-7 shrink-0 rounded-full border-2 transition",
              valor === tema.id ? "scale-110 border-tinta" : "border-transparent hover:scale-105"
            )}
            style={{ background: tema.acento }}
          />
        ))}
      </div>
      {temaAtual && <p className="text-xs text-tinta-suave">{temaAtual.descricao}</p>}
    </div>
  )
}

export { ThemePicker }
```

- [ ] **Passo 2: Aplicar em `Configuracoes.tsx`**

Trocar o bloco (linhas 619-646):
```tsx
{/* Seletor de tema */}
<div className="space-y-3">
  <p className="text-xs font-medium uppercase tracking-wide text-tinta-suave">
    Tema do {rotuloCatalogo(loja?.segmento_principal)}
  </p>
  <div className="grid grid-cols-4 gap-2">
    {TEMAS.map((t) => (
      <button
        key={t.id}
        type="button"
        onClick={() => setTema(t.id)}
        className={`rounded-xl border-2 p-2 text-left transition ${
          tema === t.id ? 'border-acento' : 'border-tinta/10 hover:border-tinta/25'
        }`}
      >
        <div
          className="mb-1.5 h-6 w-full rounded-lg"
          style={{ background: t.acento }}
        />
        <div
          className="mb-1 h-1.5 w-full rounded"
          style={{ background: t.fundo }}
        />
        <p className="truncate text-xs font-medium text-tinta">{t.nome}</p>
      </button>
    ))}
  </div>
</div>
```
por:
```tsx
{/* Seletor de tema */}
<div className="space-y-3">
  <p className="text-xs font-medium uppercase tracking-wide text-tinta-suave">
    Tema do {rotuloCatalogo(loja?.segmento_principal)}
  </p>
  <ThemePicker temas={TEMAS} valor={tema} onValorChange={setTema} />
</div>
```
Adicionar o import.

- [ ] **Passo 3: Validar**

Rodar: `cd frontend && npx tsc -b`
Esperado: sem erros.
Verificação manual: abrir Configurações — o seletor de tema deve mostrar as 16 bolinhas de cor lado a lado (bem mais compacto que o grid de 4 colunas anterior), com a descrição do tema selecionado embaixo.

- [ ] **Passo 4: Commit**

```bash
git add frontend/src/components/ui/theme-picker.tsx frontend/src/pages/admin/Configuracoes.tsx
git commit -m "feat: componente K (seletor de tema compacto), substitui o grid em Configurações"
```

---

### Task 13: Componente M (`tag-pill.tsx`) — aplicado na seleção de categoria/sub/grupo do produto

**Files:**
- Create: `frontend/src/components/ui/tag-pill.tsx`
- Modify: `frontend/src/components/admin/ProdutoFormFields.tsx` (logo após o bloco de selects de Categoria/Subcategoria/Grupo, linhas ~46-79)

**Interfaces:**
- Produces: `TagPill` (`children: ReactNode`, `onRemove: () => void`).

- [ ] **Passo 1: Criar `tag-pill.tsx`**

```tsx
import type { ReactNode } from "react"
import { X } from "lucide-react"

import { cn } from "@/lib/utils"

interface TagPillProps {
  children: ReactNode
  onRemove: () => void
  className?: string
}

function TagPill({ children, onRemove, className }: TagPillProps) {
  return (
    <span
      data-slot="tag-pill"
      className={cn(
        "inline-flex items-center gap-1.5 rounded-full bg-acento px-3 py-1 text-xs font-semibold text-superficie",
        className
      )}
    >
      {children}
      <button
        type="button"
        onClick={onRemove}
        aria-label={`Remover ${typeof children === "string" ? children : "filtro"}`}
        className="opacity-80 transition hover:opacity-100"
      >
        <X className="size-3" />
      </button>
    </span>
  )
}

export { TagPill }
```

- [ ] **Passo 2: Aplicar em `ProdutoFormFields.tsx`**

Logo depois do bloco de selects de Categoria (linha 52, fechamento da primeira `<div className="flex gap-3">`) e do bloco condicional de Subcategoria/Grupo (linhas 54-79), inserir, antes do bloco "Tipo de produto":
```tsx
{(form.categoria_id > 0 || form.subcategoria_id !== null || form.grupo_cor_id !== null) && (
  <div className="flex flex-wrap gap-2">
    {form.categoria_id > 0 && (
      <TagPill onRemove={() => onChange({ ...form, categoria_id: 0, subcategoria_id: null, grupo_cor_id: null })}>
        {categorias?.find((c) => c.id === form.categoria_id)?.nome ?? 'Categoria'}
      </TagPill>
    )}
    {form.subcategoria_id !== null && (
      <TagPill onRemove={() => onChange({ ...form, subcategoria_id: null, grupo_cor_id: null })}>
        {subcategorias?.find((s) => s.id === form.subcategoria_id)?.nome ?? 'Subcategoria'}
      </TagPill>
    )}
    {form.grupo_cor_id !== null && (
      <TagPill onRemove={() => onChange({ ...form, grupo_cor_id: null })}>
        {gruposCor?.find((g) => g.id === form.grupo_cor_id)?.nome ?? 'Grupo'}
      </TagPill>
    )}
  </div>
)}
```
Adicionar o import. Os `<select>` de Categoria/Subcategoria/Grupo continuam exatamente como estão — essa linha de pills é só uma exibição adicional da seleção atual, com atalho de remover que já reaproveita a mesma lógica de cascata (`subcategoria_id`/`grupo_cor_id` zerados junto) que os `onChange` dos selects já usam.

- [ ] **Passo 3: Validar**

Rodar: `cd frontend && npx tsc -b`
Esperado: sem erros.
Verificação manual: abrir "Novo produto" de uma loja com categoria/subcategoria/grupo cadastrados, escolher os três níveis nos selects — as pills devem aparecer refletindo a seleção; clicar no X de uma pill deve resetar aquele nível (e os de baixo).

- [ ] **Passo 4: Commit**

```bash
git add frontend/src/components/ui/tag-pill.tsx frontend/src/components/admin/ProdutoFormFields.tsx
git commit -m "feat: componente M (tags removíveis), aplicado na seleção de categoria/sub/grupo"
```

---

### Task 14: Modo noturno — store + paleta escura

**Files:**
- Create: `frontend/src/store/temaAdminStore.ts`
- Modify: `frontend/src/index.css` (bloco `.dark` já existente, linhas 196-228)

**Interfaces:**
- Produces: `useTemaAdminStore` (hook Zustand: `{ preferencia: 'claro' | 'escuro', definirPreferencia, alternar }`).

- [ ] **Passo 1: Criar `temaAdminStore.ts`**

```tsx
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
```

- [ ] **Passo 2: Adicionar a paleta escura em `index.css`**

Dentro do bloco `.dark { ... }` já existente (linhas 196-228, tokens shadcn), adicionar estas linhas logo antes do `}` de fechamento (depois de `--sidebar-ring:        oklch(0.556 0 0);`):
```css
    /* Modo noturno do admin (20/08/2026) — redefine a MESMA camada de cor
       de produto usada em toda tela do admin (bg-fundo/text-tinta/etc),
       em vez de migrar pros tokens shadcn acima (que exigiria editar
       dezenas de arquivos existentes). --color-acento (terracota) fica
       IDÊNTICO ao modo claro, de propósito — mantém a identidade visual. */
    --color-fundo:        26 20 15;
    --color-superficie:   38 30 23;
    --color-tinta:        236 226 209;
    --color-tinta-suave:  162 146 124;
    --color-douro:        214 158 74;
```

- [ ] **Passo 3: Validar**

Rodar: `cd frontend && npx tsc -b`
Esperado: sem erros (essa task ainda não tem nenhum consumidor visual do store — isso vem na Task 15).

- [ ] **Passo 4: Commit**

```bash
git add frontend/src/store/temaAdminStore.ts frontend/src/index.css
git commit -m "feat: store de preferência de tema do admin + paleta escura"
```

---

### Task 15: Modo noturno — toggle no cabeçalho + wrapper `.dark`

**Files:**
- Modify: `frontend/src/pages/admin/Dashboard.tsx`

**Interfaces:**
- Consumes: `useTemaAdminStore` (Task 14).

- [ ] **Passo 1: Editar `Dashboard.tsx`**

Adicionar os imports novos, junto dos já existentes no topo do arquivo:
```tsx
import { Moon, Sun } from 'lucide-react';
import { useTemaAdminStore } from '../../store/temaAdminStore';
import { cn } from '../../lib/utils';
```

Dentro do componente `Dashboard()`, junto das outras chamadas de hook já existentes (`useAuthStore`, `useQuery`):
```tsx
const preferencia = useTemaAdminStore((state) => state.preferencia);
const alternarTema = useTemaAdminStore((state) => state.alternar);
```

Trocar a `<div>` raiz do JSX:
```tsx
<div className="min-h-screen bg-fundo">
```
por:
```tsx
<div className={cn('min-h-screen bg-fundo', preferencia === 'escuro' && 'dark')}>
```

Trocar o botão "Sair" isolado no cabeçalho:
```tsx
<button onClick={sair} className="text-sm font-medium text-tinta-suave hover:text-acento">
  Sair
</button>
```
por (toggle + Sair, agrupados):
```tsx
<div className="flex items-center gap-3">
  <button
    onClick={alternarTema}
    aria-label={preferencia === 'escuro' ? 'Mudar pro modo claro' : 'Mudar pro modo escuro'}
    className="flex size-8 items-center justify-center rounded-full text-tinta-suave transition hover:bg-tinta/10 hover:text-tinta"
  >
    {preferencia === 'escuro' ? <Sun className="size-4" /> : <Moon className="size-4" />}
  </button>
  <button onClick={sair} className="text-sm font-medium text-tinta-suave hover:text-acento">
    Sair
  </button>
</div>
```

- [ ] **Passo 2: Validar**

Rodar: `cd frontend && npx tsc -b`
Esperado: sem erros.
Verificação manual — a mais importante desta task: logar no admin, clicar no ícone de sol/lua no cabeçalho e conferir que TODA a tela (fundo, cards, texto, bordas) muda pra paleta escura instantaneamente, em qualquer página do admin (não só Início). Recarregar a página (F5) e confirmar que a preferência escolhida persiste (não volta pro claro sozinha). Conferir visualmente que a cor de destaque (terracota, usada nos botões `btn-neu-primario`, no link ativo do menu, etc.) continua a mesma cor nos dois modos.

- [ ] **Passo 3: Commit**

```bash
git add frontend/src/pages/admin/Dashboard.tsx
git commit -m "feat: toggle de modo noturno no cabeçalho do admin"
```

---

## Self-Review (rodado antes de entregar este plano)

**Cobertura da spec**: A (Task 2), B (Task 3), C (Task 4), D (Task 5), E (Task 6), F (Task 7), G (Task 8), H (Task 9), I (Task 10), J (Task 11), K (Task 12), L (Task 8, junto com G), M (Task 13), modo noturno (Tasks 14-15). N fica de fora, como já decidido na spec (sem fluxo de verificação no sistema). Todos os 13 componentes + modo noturno têm task própria.

**Placeholders**: nenhum "TBD"/"implementar depois" — todo passo tem código completo e caminho de arquivo exato.

**Consistência de tipos**: `Stepper.onValueChange` sempre recebe `number` (não `number | null` — o componente já filtra o `null` internamente antes de chamar o callback do consumidor); `Segmented<T>` usa o mesmo tipo genérico em `opcoes`/`valor`/`onValorChange`; `InputPrice`/`InputSearch`/`InputDate`/`Dropzone` documentados com a assinatura exata que cada arquivo consumidor já espera (`onSelecionarFoto` do Dropzone, por exemplo, mantém a assinatura de evento nativo já usada em `Produtos.tsx`/`CadastroEmMassaDialog.tsx`, sem exigir mudança nesses dois arquivos fora do escopo desta spec).
