# Drenux — Planos, Comissões e Programa de Afiliados (Definido)
*Revisão: ago/2026*

## 1. Contexto da mudança
O modelo anterior (Start único: R$0 + 6,5% flat sem teto) ficava mais caro que
qualquer concorrente de mensalidade fixa (Goomer, Anota AI, Cardápio Web, Rei
do Delivery) a partir de ~R$1-5 mil/mês de GMV por loja — só era competitivo
contra o iFood (12%-27%). A mudança desacopla software de processamento de
pagamento e usa comissão escalonada com piso de segurança acima do custo real
do Mercado Pago (~0,99%), pra ficar competitivo em qualquer volume sem operar
no prejuízo.

## 2. Planos

### Start — R$0/mês
- Cardápio digital + pedidos via WhatsApp
- Até 30 produtos/categorias
- Até 30 pedidos/mês, recorrente (renova todo ciclo, igual ao Goomer — não é limite vitalício)
- Sem avisos automáticos via WhatsApp
- Sem pagamento integrado — loja recebe Pix na própria chave
- Marca "Feito com Drenux" visível
- Ao ultrapassar 30 pedidos: aviso + 3 dias de tolerância recebendo pedidos normalmente; sem
  migrar nesse prazo, pausa novos pedidos até o mês virar (quota renova) ou até ativar o Basic

### Basic — R$0/mês
- Tudo do Start, sem limite de produtos ou pedidos
- Avisos automáticos via WhatsApp (status do pedido pro cliente)
- Pagamento integrado (split via Mercado Pago) ativado
- Comissão escalonada:
  - R$0 – R$5.000 GMV/mês: **2,4%**
  - R$5.000 – R$20.000: **1,5%**
  - Acima de R$20.000: **1,3%**
- Marca "Feito com Drenux" visível

### Pro — R$99/mês
- Tudo do Basic
- Comissão escalonada:
  - R$0 – R$5.000: **1,8%**
  - R$5.000 – R$20.000: **1,2%**
  - Acima de R$20.000: **1,05%**
- Rastreamento de entrega em tempo real (cliente acompanha no mapa)
- Gestão de entregadores (atribuição de pedidos)
- Domínio próprio
- Marca Drenux removível
- Relatórios avançados/exportação
- Múltiplos usuários/equipe
- Automação (carrinho abandonado etc.)
- Suporte prioritário

### Scale — R$149,90/mês
*(revisado em 18/08/2026, ver nota abaixo)*
- Tudo do Pro
- Comissão flat: **0,99%** sobre todo o GMV (piso de custo real do Mercado Pago)
- Multi-loja / gestão de rede consolidada
- Suporte dedicado

**Nota de revisão (18/08/2026)**: o valor original desta seção (R$349/mês + 1,1% flat) fazia o
Scale só compensar em custo a partir de ~R$353 mil/mês de GMV — faturamento alto demais pra a
imensa maioria das lojas ver esse cruzamento na prática (William: "só compensa pra empresas
enormes"). A taxa já estava no piso possível (guardrail §4), então a correção foi na mensalidade:
caiu pra R$149,90. Consequência aceita conscientemente: com a mensalidade do Pro (R$99) tão perto
da do Scale, o **Pro deixou de ter qualquer faixa de faturamento em que é o mais barato dos três**
— ele segue existindo como "os recursos do Scale com mensalidade menor e sem compromisso de alto
volume", não como opção de menor custo. A alternativa (encarecer o Basic o bastante pra abrir uma
janela de preço pro Pro) foi avaliada e descartada — exigiria quase dobrar a taxa mais alta do
Basic, o que piora sua posição competitiva por uma janela de Pro pequena demais pra valer a pena.
Ver `docs/plano-melhorias-drenux.md`, Fase 7, pro detalhe completo da simulação.

## 3. Sugestão inteligente (IA) — cota por consumo
Trava por custo real da chamada de API, não por nível de plano:
- Start/Basic: cota paga avulsa
- Pro: cota mensal inclusa + excedente pago
- Scale: cota mensal maior inclusa + excedente pago

## 4. Guardrails financeiros — não alterar sem recalcular
- Custo real do Mercado Pago (Pix): **~0,99%** do GMV processado
- Nenhuma faixa de comissão pode ficar abaixo desse custo
- Margem mínima recomendada: ~0,3 p.p. acima do custo em planos sem
  mensalidade (Basic); Pro/Scale podem operar mais perto do custo porque a
  mensalidade fixa cobre a diferença
- A Drenux sempre absorve a taxa do Mercado Pago em qualquer plano pago — não existe mais a
  distinção antiga "Start absorve / Pro-Scale a loja absorve"
- Referência viva: `drenux-simulador-planos.html` (recalcula tudo ao vivo)

## 5. Matriz de funcionalidades

| Funcionalidade | Start | Basic | Pro | Scale |
|---|:---:|:---:|:---:|:---:|
| Cardápio digital (link + QR code) | ✅ | ✅ | ✅ | ✅ |
| Pedidos via WhatsApp | ✅ | ✅ | ✅ | ✅ |
| Nº de produtos/categorias | até 30 itens | ilimitado | ilimitado | ilimitado |
| Pedidos/mês sem custo | até 30 | ilimitado | ilimitado | ilimitado |
| Avisos automáticos de status via WhatsApp | ❌ | ✅ | ✅ | ✅ |
| Pagamento integrado (Pix automático/split) | ❌ | ✅ (comissão) | ✅ (comissão menor) | ✅ (comissão menor ainda) |
| Marca "Feito com Drenux" | visível | visível | removível | removível |
| Rastreamento de entrega em tempo real (mapa) | ❌ | ❌ | ✅ | ✅ |
| Gestão de entregadores (atribuição de pedidos) | ❌ | ❌ | ✅ | ✅ multi-loja |
| Domínio próprio | ❌ | ❌ | ✅ | ✅ |
| Relatórios avançados/exportação | ❌ | básico | ✅ | ✅ |
| Múltiplos usuários/equipe | ❌ | ❌ | ✅ | ✅ |
| Automação (carrinho abandonado etc.) | ❌ | ❌ | ✅ | ✅ |
| Multi-loja / rede consolidada | ❌ | ❌ | ❌ | ✅ |
| Suporte | padrão | padrão | prioritário | dedicado |
| Sugestão inteligente (IA) | cota paga avulsa | cota paga avulsa | cota inclusa + excedente | cota maior + excedente |

## 6. Programa de afiliados
- **Correção de contexto**: a fórmula real em produção antes desta revisão é ~37,6% da taxa de
  plataforma (comissão bruta), igual pra qualquer plano — não "30% do lucro líquido pro Start"
  (essa nunca existiu no código, foi engano de conversa anterior)
- Nova base: **% do lucro líquido** (comissão − custo real do Mercado Pago), nunca da comissão
  bruta — garante que a Drenux nunca opera no negativo numa loja indicada, em qualquer volume
- Percentual: **45%** do lucro líquido, recorrente enquanto a loja indicada ficar ativa em
  Basic/Pro/Scale (Start puro, sem split, não gera repasse)
- Mensalidades fixas (Pro/Scale) **não** entram na base de cálculo — ficam 100% com a Drenux
- Bônus de ativação (pago ao completar onboarding + mínimo de pedidos nos primeiros 60 dias, ex:
  30 pedidos; só pago se a loja ativar Basic, Pro ou Scale):
  - Basic: R$60
  - Pro: R$150
  - Scale: R$400
- Bônus de upgrade (loja indicada migra de plano): validado como ideia, **valor ainda não definido**
- Status (ago/2026): 1 afiliado cadastrado, nenhuma indicação iniciada — avaliar condição
  "fundador" (ex: bônus dobrado, por tempo limitado) pro primeiro lote, já que o risco de caixa é
  zero enquanto não há indicação ativa

## 7. Posicionamento vs. concorrência (referência, ago/2026)
| Concorrente | Modelo | Faixa de preço |
|---|---|---|
| Goomer | Mensalidade fixa | R$0 (até 30 pedidos/mês) a R$224,93/mês |
| Anota AI | Mensalidade fixa | R$219,99 a R$329,99/mês |
| Cardápio Web | Mensalidade fixa | R$169,99 a R$269,99/mês |
| Rei do Delivery | Mensalidade fixa | R$49,90/mês |
| iFood | Comissão (marketplace) | 12% a 27% por pedido |
| **Drenux** | Freemium + comissão escalonada | R$0 a R$149,90/mês + 0,99%–2,4% |

## 8. Comunicação / UX — obrigatório
- Nunca usar "comissão" isolado — ancorar em "taxa de processamento de pagamento", pra não puxar
  associação automática com iFood
- Headline pública: "Comece de graça, sem cartão, taxa só quando ativar pagamento automático — e
  ela cai conforme você cresce"
- Matriz completa de funcionalidades fica como calculadora/detalhe, não como primeira tela
- Contador de pedidos do Start visível no painel (ex: "18/30 pedidos este mês") — nunca deixar o
  dono descobrir o limite só ao travar; mensagem de bloqueio em tom de lembrete, nunca de punição

## 9. Em aberto
- Fase 5.5 (repasse de afiliado): repasse continua manual (controle interno via
  `RepasseAfiliado`/`/drenux/afiliados`), não split automático — decisão já fechada, ver Fase 5.5
  no roadmap. A mudança desta revisão é só a fórmula (item 6), não o mecanismo de pagamento.
- Valor exato do bônus de upgrade de plano do afiliado
- Mecanismo técnico do cap de 30 pedidos/mês do Start (bloqueio direto vs. aviso + upgrade guiado)
  — especificação em `docs/plano-melhorias-drenux.md`, Fase 7.3
