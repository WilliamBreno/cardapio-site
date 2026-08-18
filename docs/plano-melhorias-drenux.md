# Plano de melhorias — Drenux

> Este arquivo documenta o roadmap combinado numa conversa com o Claude (chat) em julho/2026.
> Ele existe pra você (Claude Code) saber exatamente o que fazer quando o William disser algo como
> **"Pode atualizar"**, sem precisar que ele reexplique o contexto a cada sessão.

## Instruções de execução

Quando o William pedir pra seguir o plano ("pode atualizar", "bora pra próxima fase", etc.):

1. Releia este arquivo do início. Ache a primeira fase com status `[ ] pendente`.
2. Implemente **só essa fase** — não pule fases, não faça mais de uma por vez, a menos que ele peça
   explicitamente.
3. Depois de implementar, valide antes de dizer que terminou:
   - Backend: `cd backend && go build ./...` (e `go vet ./...` se der tempo)
   - Frontend: `cd frontend && npx tsc -b` (typecheck) — o build completo (`npm run build`) se quiser
     ir além
4. Corrija qualquer erro que aparecer antes de considerar a fase concluída.
5. Marque a fase como `[x] concluída` neste arquivo (edite o status abaixo) e resuma pro William, em
   poucas linhas, o que mudou e em quais arquivos.
6. Pare e espere ele revisar antes de seguir pra próxima fase, a menos que ele quando peça
   explicitamente pra fazer tudo de uma vez.

Se qualquer instrução deste arquivo conflitar com o que o William pedir na hora, o pedido dele na
hora vale mais — este arquivo é o plano combinado, não uma trava.

## Contexto do produto

Drenux (antigo nome: Brenvo/Cardápio Site) é um SaaS multi-tenant de cardápio/loja online com
pagamento integrado. Atende dois perfis de loja: **alimentício** (comida/bebida) e **mercadoria**
(produtos não perecíveis, ex: roupas, artesanato). Monetização por assinatura combinada com a taxa
por pedido, que varia por plano — **planos e fórmula de comissão em processo de reestruturação, ver
Fase 7 mais abaixo**. Até a Fase 7 ser implementada, o código ainda reflete o modelo antigo (3
planos: Start/Pro/Scale, ver `MeuPlano.tsx`).

**Processador de pagamento em migração: Stripe Connect → Mercado Pago (decisão final, fechada em
23/07/2026, depois de avaliar Mercado Pago contra Asaas).** Ainda nenhuma empresa real está
cadastrada em produção, então a troca está sendo feita antes do lançamento real — motivo principal é
o Pix (na Stripe, hoje só disponível por convite pra empresas brasileiras), e no Mercado Pago
especificamente: sem teto de quantidade de contas vinculadas (diferente da Asaas, que tem limite de
10 subcontas até homologação regulatória) e Pix cobrado em percentual (0,99%), o que favorece o
ticket baixo típico do segmento alimentício. Ver a especificação completa da migração de processador
na Fase 5, mais abaixo.

**Fórmula de comissão — estado legado, em código até a Fase 7 ser implementada**: Start
`max(pedido × 6,5%, R$2,50)` com a Drenux absorvendo a taxa do processador; Pro (R$129/mês+4%) e
Scale (R$349/mês+1,5%), com a loja absorvendo a taxa do processador. **Isso está sendo substituído
pela Fase 7** (planos Start/Basic/Pro/Scale, comissão escalonada por faixa de GMV mensal, e a Drenux
sempre absorvendo a taxa do Mercado Pago em qualquer plano pago — sem distinção de "quem absorve" por
plano como no modelo antigo). Ver Fase 7 pra especificação completa; este parágrafo documenta só o
que está em produção agora, não o alvo.

Stack: backend Go (Gin/GORM/PostgreSQL) em `backend/`, frontend React 19 + TypeScript + Vite +
Tailwind v3 + TanStack Query + Zustand em `frontend/`.

## Bibliotecas de UI — onde usar cada uma

| Área | Biblioteca | Observação |
|---|---|---|
| Painel Admin (formulários, tabelas, dialogs) | **shadcn/ui** | Já instalado (`components.json`, style `base-nova`). Base de tudo. |
| Fluxos novos de cadastro em massa / variações | shadcn + **21st.dev** | 21st.dev não é dependência — é diretório de blocos prontos pra copiar (formulário multi-step, data table). |
| Meu Plano (cards de plano, alerta de upgrade) | 21st.dev (layout) + **Magic UI** (destaque) | Já em uso: `ShimmerButton`, `NumberTicker` em `Planos.tsx` e `MeuPlano.tsx`. |
| Cardápio público (cliente final) | **Magic UI** como padrão | React Bits só como reserva pontual — não instalar como base pra não duplicar dependência de animação com a Magic UI. |

## Fases

### Fase 1 — Tipo de loja (`SegmentoPrincipal`)
Status: `[x] concluída` — **revisão em 23/07/2026**: ao reler o arquivo antes de começar, encontrei
a fase inteira já implementada (backend e frontend, incluindo o bug do `CodigoAfiliado`/
`TokenAssinatura` e o texto final "O que sua loja vende principalmente?" / "Comida e bebida" /
"Outros produtos"). Não sobrou nenhum item da checklist abaixo em aberto. Não refiz nada — só
conferi arquivo por arquivo e validei com `go build ./...` e `npx tsc -b`, os dois passando limpo.
Checklist original mantida abaixo como referência do que foi coberto.

**Pedido novo do William (22/07/2026):** o rótulo dessa escolha pro lojista precisa de um nome
melhor que "alimentício"/"mercadoria" cru na tela — esses continuam sendo os valores internos do
enum (não mudar o `oneof=alimenticio mercadoria` no backend), só o texto exibido na interface
(pergunta + rótulo dos botões) deve ficar mais claro pro lojista leigo entender na hora do
cadastro. Sugestão a validar com o William antes de implementar (não travar nisso sozinho): algo
como "O que sua loja vende?" com opções "Comida e bebida" vs. "Outros produtos" — mas confirmar
com ele antes de fixar o texto final.

Objetivo: cada loja declara se vende principalmente produtos **alimentícios** ou **mercadoria**
(reaproveitando o enum `TipoProduto` que já existe em `domain.Produto` — não criar um vocabulário
novo). Isso: (1) define o tipo padrão de produtos novos, (2) mais pra frente vai alimentar a categoria
de negócio sugerida no onboarding da loja no processador de pagamento (Mercado Pago, ver Fase 5 e a
seção "Contexto do produto" acima; não amarrar essa lógica a campos específicos da API da Stripe),
(3) decide qual fluxo de catálogo mostrar nas Fases 2/3.

**Backend:**
- `domain/loja.go` — campo novo `SegmentoPrincipal TipoProduto` (gorm `default:'alimenticio'`,
  json `segmento_principal`). Migra sozinho via `AutoMigrate`, sem SQL manual.
- `handler/auth_handler.go` — `cadastroRequest` ganha `SegmentoPrincipal string` (`binding:"required,oneof=alimenticio mercadoria"`), repassado pro `CadastroInput`.
  - **Bug pra corrigir de quebra**: `CodigoAfiliado` e `TokenAssinatura` chegam no JSON do
    `cadastroRequest` mas hoje **não são repassados** pro `service.CadastroInput` na chamada de
    `Cadastrar()` — atribuição de afiliado e finalização de assinatura paga no cadastro estão
    quebradas silenciosamente. Adicionar os dois campos na struct e na chamada.
- `service/auth_service.go` — `CadastroInput` ganha `SegmentoPrincipal string`; usar na criação da
  `domain.Loja{}`; criar helper `categoriasPadrao(segmento, lojaID)` que devolve categorias diferentes
  por segmento (ex: alimentício → "Salgados"/"Doces"; mercadoria → "Mais vendidos"/"Novidades") em vez
  do slice fixo atual.
- `handler/loja_handler.go` + `repository/loja_repository.go` — `SegmentoPrincipal` também editável
  depois via `PUT /admin/loja` (mesmo padrão do campo `AceitaGuardarEntregar`, que já existe e serve
  de referência de como um campo de configuração é adicionado ponta a ponta).

**Frontend:**
- `api/types.ts`, `api/auth.ts`, `api/admin.ts` — adicionar `segmento_principal: 'alimenticio' | 'mercadoria'`.
- `pages/Cadastro.tsx` — seletor visual (dois botões) pra escolher o segmento no cadastro, obrigatório.
- `pages/admin/Configuracoes.tsx` — mesmo seletor, editável depois (mesmo padrão visual do bloco
  "Guardar e entregar depois" que já existe nesse arquivo).
- `pages/admin/Produtos.tsx` — buscar a `loja` (`useQuery(['loja'], buscarLoja)`) e usar
  `loja?.segmento_principal` como valor inicial do campo `tipo_produto` ao abrir "novo produto".

**Bug bônus pra corrigir (achado à parte, sem relação com a fase, mas trava build em clone novo):**
Se existir uma pasta literal `frontend/@/lib/utils.ts`, é o `lib/utils.ts` do shadcn no lugar errado
(deveria ser `frontend/src/lib/utils.ts` — confere o alias `@` em `vite.config.ts` e `tsconfig.app.json`,
os dois já apontam pra `src/`). Mover pro lugar certo com `git mv` e apagar a pasta `@` vazia.

### Fase 2 — Variações de produto (só segmento alimentício)
Status: `[x] concluída` — **revisão em 23/07/2026**: igual a Fase 1, já estava tudo implementado
(`domain.VariacaoProduto.MostrarValorAdicional`, `variacao_handler.go`/`variacao_service.go`
repassando o campo, `VariacaoFormFields.tsx` com o checkbox, e `ProdutoCard.tsx` só mostrando o
valor adicional no cardápio público quando `mostrar_valor_adicional` é `true`). Não refiz nada,
só validei com `go build ./...` e `npx tsc -b`. **Atenção, divergência encontrada**: o código foi
além do que essa fase pedia — `VariacaoProduto` ganhou também `ModoPreco` ("aditivo"/"absoluto")
e fotos próprias por variação, e o modo "absoluto" já é usado pra produtos `mercadoria` (preço/
foto por variação, ex: modelos de um tênis). Isso contraria a decisão registrada na Fase 3 de que
`mercadoria` não usa variação, só Subcategoria/Grupo de Cor. Ver nota de divergência antes da Fase 3.

**Importante, decisão do William em 23/07/2026**: variação (`domain.VariacaoProduto`, aditiva sobre o
preço base) é um recurso de **cardápio**, não de catálogo de varejo. Essa fase se aplica **só** a
lojas com `SegmentoPrincipal = alimenticio`. Lojas `mercadoria` **não têm essa opção** — pra elas,
o mecanismo próprio é a Fase 3 (Subcategoria + Grupo de Cor), não variação. Não confundir os dois
nem tentar reaproveitar a mesma estrutura de dados entre os dois segmentos.

Objetivo, restrito a alimentício: um único ajuste no sistema de variações que hoje só tem
`PrecoAdicional` (valor somado ao preço base) — adicionar um **toggle de visibilidade do valor
adicional**: campo novo (ex: `MostrarValorAdicional bool`) pra decidir se o preço extra da variação
aparece pro cliente no cardápio público ou fica escondido.

### Fase 3 — Catálogo de varejo (só segmento "mercadoria"/outros produtos)
Status: `[x] concluída` — **revisão em 23/07/2026, confirmada com o William**: as quatro partes já
estavam implementadas de ponta a ponta. 3.1: `domain/subcategoria.go`, `domain/grupo_cor.go`,
`subcategoria_service.go`/`grupo_cor_service.go` (com checagem de dono e bloqueio de exclusão se
ainda tiver produto vinculado), rotas em `main.go`, `Categorias.tsx` + `HierarquiaCategoria.tsx` no
admin. 3.2: botão "Cadastro em massa" em `Produtos.tsx` abrindo `CadastroEmMassaDialog.tsx` (wizard
produto → fotos/variações → próximo produto, só aparece pra `mercadoria`). 3.3: `Produtos.tsx`
agrupando por Categoria → Subcategoria → Grupo de Cor pra lojas `mercadoria` (lista plana continua
pra `alimenticio`). 3.4: `CatalogoGrid.tsx` no `CardapioPublico.tsx`, grid com filtro em cascata,
layout de lista/abas mantido pra `alimenticio`. Validado com `go build ./...` e `npx tsc -b`, sem
nenhuma mudança de código — só confirmação.

**Reescrita em 23/07/2026** a partir de feedback do William — essa fase deixou de ser só "cadastro em
massa" e virou uma reestruturação de como o catálogo funciona pra lojas de varejo (roupa, sapato,
produtos do gênero). São quatro partes, todas exclusivas de `SegmentoPrincipal = mercadoria` (não
aparecem/não fazem sentido pra loja alimentício):

**3.1 — Hierarquia Categoria → Subcategoria → Grupo de Cor**
- Uma `Categoria` existente (ex: "Tênis") ganha **Subcategorias opcionais**, pensadas pra representar
  tamanho (ex: "40", "41", "42").
- Dentro de uma Subcategoria, opcionalmente também um **Grupo de Cor** (ex: "Tons escuros", "Branco").
- Leitura confirmada com o William: isso é um drill-down (Categoria → Subcategoria → Grupo de Cor),
  **não** o mesmo conceito de variação da Fase 2 — os dois sistemas não se misturam. Pra
  `mercadoria`, a Fase 2 (variação) nem aparece como opção.
- **Cardinalidade**: uma combinação de Subcategoria + Grupo de Cor pode conter **vários produtos
  diferentes** (ex: várias camisas diferentes que são todas tamanho 42 e todas de cor escura) — não é
  uma relação 1:1 produto↔combinação.
- **Tudo opcional**: o lojista decide se usa Subcategoria, se usa Grupo de Cor dentro dela, ou nenhum
  dos dois — cadastro simples de produto sem essa estrutura continua funcionando normalmente.
- **Confirmado com o William (23/07/2026)**: Grupo de Cor é sempre aninhado dentro de uma
  Subcategoria — Categoria → Subcategoria → Grupo de Cor é uma cadeia só, não duas facetas
  paralelas independentes. Implementar o schema já nessa estrutura, sem alternativa a considerar.

**3.2 — Cadastro em massa**
- Botão de "adicionar produtos em sequência" (cadastro rápido, um atrás do outro, sem fechar o
  formulário) — vive **dentro da própria tela de Produtos** (não em tela separada), e só aparece
  quando a loja é `mercadoria`.

**3.3 — Exibição organizada no admin**
- A lista de produtos no admin, pra lojas `mercadoria`, precisa refletir visualmente a hierarquia
  Categoria/Subcategoria/Grupo de Cor de forma organizada — não é só uma lista plana como hoje.

**3.4 — Catálogo público em formato de e-commerce**
- Pra lojas `mercadoria`, a página pública do cardápio muda de layout: sai do formato lista-por-
  categoria (estilo cardápio de comida) e vira algo mais parecido com catálogo de loja online (grid
  de produtos, navegação/filtro por categoria → subcategoria → grupo de cor). Loja `alimenticio`
  mantém o layout atual, sem mudança.

### Fase 4 — Meu Plano: alerta proativo
Status: `[x] concluída` — **revisão em 23/07/2026**: já implementado em `pages/admin/Inicio.tsx`,
reaproveitando `PLANOS`/`custoPlano`/`planoMaisBarato` de `lib/planos.ts` (a mesma lógica de
`MeuPlano.tsx`). Mostra o alerta só quando há economia real (`economiaMensal > 0`) e o plano
recomendado é diferente do atual, com link direto pra "Meu Plano". Validado com `npx tsc -b`, sem
nenhuma mudança de código.

O essencial de "Meu Plano" **já existe** em `pages/admin/MeuPlano.tsx`: planos Start/Pro/Scale reais,
troca de plano funcionando (com downgrade agendado pra renovação + cancelamento), e uma recomendação
de "mais barato pra você" já calculada com o faturamento real do mês (`dashboard.total_mes`).

O que falta: essa recomendação só aparece se o lojista entrar na tela "Meu Plano". Objetivo da fase:
expor um alerta proativo em `pages/admin/Inicio.tsx` (ou `Dashboard.tsx`) reaproveitando a mesma lógica
de cálculo que já existe em `MeuPlano.tsx`, avisando quando o faturamento do mês ultrapassa o ponto de
equilíbrio pra outro plano.

### Fase 5 — Integração real com o Mercado Pago (decisão fechada em 23/07/2026)
Status: `[~] 5.1–5.5 implementadas — checkout/webhook (5.1–5.4) testados em produção em 24/07/2026
com correções aplicadas ao longo do teste; repasse de afiliado (5.5) é controle manual, não split
automático — ver detalhe de cada subfase abaixo`

**O que foi implementado (código novo, revisar antes de confiar em produção):**
- `domain/loja.go`: `MercadoPagoAccessToken`/`MercadoPagoRefreshToken`/`MercadoPagoUserID`/
  `MercadoPagoTokenExpiraEm`. `domain/pedido.go`: `MercadoPagoPreferenceID`.
- `config.go`: `MERCADOPAGO_CLIENT_ID`, `MERCADOPAGO_CLIENT_SECRET`, `MERCADOPAGO_WEBHOOK_SECRET`
  (chave separada, configurada no painel do Mercado Pago, usada só pra validar a assinatura do
  webhook) e `API_PUBLIC_URL` (endereço público desta API — precisa bater com o redirect_uri
  cadastrado na aplicação do Mercado Pago).
- `service/mercadopago_service.go` (novo): onboarding OAuth (`state` é um JWT curto assinado com o
  mesmo `JWT_SECRET`, carregando o `loja_id` — o callback do Mercado Pago não manda nenhum header
  nosso, é um redirect de navegador puro), troca de código por token, renovação via refresh_token,
  checkout de pedido via `POST /checkout/preferences` com `marketplace_fee` usando o access_token da
  própria loja, e processamento do webhook (`GET /v1/payments/:id` com token de aplicação
  `client_credentials`, cross-check do `collector_id` contra a loja do pedido).
- `service/pos_pagamento_service.go` (novo): a lógica de desconto de estoque + notificação
  WhatsApp que antes vivia só dentro do `StripeService` foi extraída pra cá, porque agora é chamada
  tanto pelo pagamento de pedido via Mercado Pago quanto (indiretamente) pela Stripe — evita ter a
  mesma lógica de estoque duplicada e divergindo com o tempo entre os dois processadores.
- `handler/mercadopago_handler.go` (novo) + rotas em `main.go`: `GET /admin/mercadopago/onboarding`
  e `GET /admin/mercadopago/status` (autenticadas), `GET /admin/mercadopago/callback` (pública —
  não dá pra proteger com JWT porque quem chama é o redirect do navegador vindo do Mercado Pago, sem
  nenhum header nosso), `POST /webhooks/mercadopago` (pública, valida assinatura), e
  `POST /mercadopago/renovar-tokens` (pública, protegida por `X-Cron-Secret`, mesmo padrão do
  `/relatorio/semanal` — precisa de um cron externo tipo cron-job.org batendo nela periodicamente,
  igual já existe pro relatório).
- **`POST /pedidos/:id/checkout` agora chama o Mercado Pago, não mais a Stripe** — só essa rota.
  `POST /solicitacoes/:id/checkout` (frete de itens guardados) e `/planos/checkout` (assinatura
  Pro/Scale) continuam na Stripe, como o texto original da Fase 5.2 pedia (só menciona
  `/pedidos/:id/checkout`) — não presumi que o resto também devesse migrar. O código da Stripe
  inteiro continua no repositório, só parou de ser chamado nessa rota específica.
- Frontend: `Configuracoes.tsx`, bloco "Pagamento" trocado de Stripe pra Mercado Pago (mesmo padrão
  visual, `api/admin.ts` com `iniciarOnboardingMercadoPago`/`statusMercadoPago`).

**Correções em `mercadopago_service.go` (24/07/2026), a pedido do William:**
1. **Piso de R$2,50 no plano Start.** `CriarCheckout` calculava a comissão só como percentual
   (`TaxaPlataformaPercentual`, hoje 8%), sem aplicar nenhum mínimo. Adicionada
   `calcularMarketplaceFee` (substitui a antiga `taxaPlataformaPercentualPedido`), que aplica
   `max(base × 8%, R$2,50)` pro Start; Pro (4%) e Scale (1,5%) continuam só percentuais, sem piso.
   **Atenção**: apesar do pedido dizer "mesma regra do stripe_service.go", conferi e esse piso **não
   existe** no `stripe_service.go` — lá a comissão do Start é só o percentual puro
   (`TaxaPlataformaPercentual = 8.0`), sem `max(..., R$2,50)` em lugar nenhum. Também não é 6,5% como
   a seção "Contexto do produto" deste documento registra — tanto o backend (`stripe_service.go`)
   quanto o que já é mostrado pro lojista no frontend (`lib/planos.ts`, `PLANOS[0].taxa = 0.08`) usam
   8% pro Start hoje. Ou seja: a decisão "Start = 6,5%" registrada na seção de contexto ainda não foi
   implementada em lugar nenhum do código — só apliquei o piso de R$2,50 sobre a taxa que já existe
   de fato (8%), sem mudar o percentual. Se a intenção é 6,5% de verdade, isso precisa de mais uma
   rodada — trocar o `TaxaPlataformaPercentual` e o `PLANOS[0].taxa` do frontend juntos, pra não
   ficar cobrando um valor e mostrando outro pro lojista.
   **Resolvido em 04/08/2026, a pedido do William**: confirmado com ele que hoje, nesse split via
   Mercado Pago, quem paga tanto a taxa do processador quanto a comissão da Drenux é a loja (o
   cliente final só paga o valor do pedido) — e que os 8% cheios, sem nenhum desconto pela taxa do
   processador, contrariavam a decisão original de a Drenux absorver essa taxa no Start. Trocado
   `TaxaPlataformaPercentual` (`stripe_service.go`) de `8.0` pra `6.5` e `PLANOS[0].taxa`
   (`frontend/src/lib/planos.ts`) de `0.08` pra `0.065`, juntos, pra loja e lojista verem o mesmo
   número. `ComissaoAfiliadoPercentual` também ajustada de `3.01` pra `2.44` (mantém a proporção de
   `ProporcaoComissaoAfiliadoPadrao = 0.376` sobre a nova taxa: 6,5% × 0,376 ≈ 2,44%) — essa
   constante está sem nenhum uso no código hoje (só documentativa, quem manda de verdade é
   `Afiliado.ComissaoPercentual` por afiliado), mas foi corrigida pra não ficar um número
   inconsistente largado no arquivo. `calcularComissaoAfiliado` e `calcularMarketplaceFee`
   (Mercado Pago) leem `TaxaPlataformaPercentual` direto, então já refletem os 6,5% sem precisar de
   mudança própria neles. **Não implementado, e não foi pedido**: uma lógica que calcule a taxa real
   do processador por transação (varia por método de pagamento — Pix, cartão, parcelamento) e
   desconte isso dinamicamente da comissão — os 6,5% são um número fixo de negócio, a mesma
   abordagem simples que já existia pros outros dois planos, não uma conta em tempo real contra a
   taxa efetiva do Mercado Pago/Stripe. Validado com `go build ./...`, `go vet ./...`, `gofmt -l`,
   `npx tsc -b` e `npm run build`, todos limpos.
2. **Comissão não incide sobre frete.** Confirmado em `pedido_service.go` (`CriarPorSlug`):
   `pedido.Total = total (subtotal dos itens) + taxaEntrega`, ou seja, **`Total` já inclui o
   frete**. `CriarCheckout` agora calcula a comissão sobre `pedido.Total - pedido.TaxaEntrega` (só o
   subtotal dos itens), não mais sobre `pedido.Total` inteiro. Mesma ressalva do item acima: essa
   exclusão do frete também **não existe** no `stripe_service.go` hoje (nem no checkout de pedido nem
   no repasse de comissão de afiliado, `transferirComissaoAfiliado`) — ambos ali usam
   `pedido.Total` cheio, com frete incluído na base de comissão. Não toquei nesses dois pontos da
   Stripe porque não foi pedido e o checkout de pedido da Stripe está fora de uso desde a Fase 5.2
   (rota `/pedidos/:id/checkout` migrou pro Mercado Pago) — mas fica registrado que os dois
   processadores calculariam a comissão de forma diferente se o checkout Stripe voltasse a ser usado
   pra pedido.
Validado com `go build ./...`, `go vet ./...` e `gofmt -l`, todos limpos.

**Correção 3 (24/07/2026), achada em teste real com pedido de R$1**: o Mercado Pago recusou o
checkout com `invalid_marketplace_fee` ("marketplace_fee must not be greater than total amount") —
o piso de R$2,50 da Correção 1 sozinho já passava do total de um pedido de R$1. `CriarCheckout`
agora limita `marketplace_fee` ao total realmente cobrado na preference (soma dos itens + frete);
se o piso ultrapassar isso, a comissão vira só o total do pedido em vez de travar o checkout.
**Achado à parte, não corrigido**: os itens enviados pro Mercado Pago usam o preço cheio de cada
item (`item.PrecoUnit`) — se o pedido tiver desconto de cupom aplicado, `pedido.Total` (com
desconto já subtraído) fica menor que a soma dos itens enviados na preference, cobrando o cliente
a mais do que o pedido realmente vale. Não toquei nisso porque não foi pedido e não apareceu no
teste (sem cupom), mas fica registrado — precisa de uma correção própria antes de qualquer loja
usar cupom com Mercado Pago de verdade.

**Correção 4 (24/07/2026), a mais importante das quatro — achada em teste real de ponta a ponta**:
o webhook só processava notificações `type=payment`. Só que, na prática, testando com a Loja
conectada de verdade, o Mercado Pago mandou só notificações `topic=merchant_order` pros dois
pagamentos de teste (um deles recusado pelo próprio Mercado Pago com a mensagem genérica "não foi
possível processar seu pagamento" — recusa do lado do Mercado Pago/banco, não bug nosso). Como o
código ignorava qualquer coisa que não fosse `type=payment`, **nenhum pagamento seria processado
de verdade nessa integração**, aprovado ou não — o webhook devolvia 200 sem fazer nada.
`MercadoPagoHandler.Webhook` agora trata os dois tipos: `merchant_order` busca
`GET /merchant_orders/:id`, pega o(s) pagamento(s) associados, e delega cada um pro mesmo
`ProcessarNotificacaoPagamento` de sempre (que já é idempotente). Ainda não confirmamos com um
pagamento **aprovado** de verdade que o fluxo completa (desconto de estoque, WhatsApp, `pedido`
marcado como pago) — só que a notificação agora chega até o processamento em vez de ser
descartada na porta. Validado com `go build`/`go vet`/`gofmt`, ainda precisa de um teste aprovado
de ponta a ponta pra fechar a Fase 5 de vez.

**Ressalvas importantes antes de ir pra produção:**
1. **Nada disso foi testado contra a API real do Mercado Pago** — não há credenciais de sandbox
   nesse ambiente. Antes de confiar: criar a aplicação "drenux-marketplace" no Mercado Pago
   Developers, pegar `MERCADOPAGO_CLIENT_ID`/`MERCADOPAGO_CLIENT_SECRET`/`MERCADOPAGO_WEBHOOK_SECRET`
   de teste, testar o fluxo de onboarding OAuth ponta a ponta, criar um pagamento de teste e conferir
   se o webhook chega e é validado corretamente.
2. A validação de assinatura do webhook (`ValidarAssinaturaWebhook` em `mercadopago_service.go`)
   segue o algoritmo documentado publicamente pelo Mercado Pago (`x-signature` com manifest
   `id:...;request-id:...;ts:...;`), mas isso é exatamente o tipo de detalhe que costuma ter
   pegadinha na prática — validar contra uma notificação real antes de confiar que pagamentos vão
   ser aceitos/rejeitados corretamente.
3. **Fase 5.5 (repasse de comissão de afiliado) não foi implementada** — nem deveria, o texto
   original já pedia decisão do William antes. Hoje, um pedido pago via Mercado Pago numa loja com
   afiliado vinculado só gera um log de aviso (`ProcessarNotificacaoPagamento` em
   `mercadopago_service.go`) — o repasse precisa ser feito manualmente até isso ser resolvido.
4. `GET /admin/mercadopago/onboarding` devolve JSON (`{ url }`) pro frontend redirecionar, em vez de
   um redirect HTTP direto do backend — diferente do texto original da Fase 5.1 ("redireciona a loja
   pra lá"), mas necessário: essa rota exige o Bearer token do dono (só ele pode iniciar o onboarding
   da própria loja), e um redirect de navegador puro não carrega esse header. É o mesmo padrão que já
   existia pro onboarding da Stripe nesse projeto.
5. Validado só com `go build ./...`, `go vet ./...`, `npx tsc -b` e `npm run build` — nenhum teste de
   integração real.

**Decisão fechada**: Mercado Pago, não Asaas. Motivo resumido (contexto completo na seção
"Contexto do produto" acima e no histórico de decisões): sem teto de quantidade de contas
vinculadas (cada Loja usa a própria conta Mercado Pago via OAuth, não uma subconta criada pela
Drenux), Pix cobrado em percentual (0,99%) o que favorece o ticket baixo típico do segmento
alimentício, e Split 1:1 já validado tecnicamente em Sandbox (preferência e pagamento aceitos com
`marketplace_fee`/`application_fee`, apontando o `collector_id` certo pro vendedor).

**Confirmado com o William (23/07/2026)**: como nenhuma loja real
está em produção ainda, a integração da Stripe deve ser **substituída por completo** pelo Mercado
Pago (não manter os dois rodando em paralelo) — mas não apagar o código da Stripe do histórico do
git, só parar de chamá-lo ativamente. Se o William quiser manter a Stripe como opção secundária por
algum motivo, avisar antes de remover qualquer rota/handler dela.

**5.1 — Backend: conexão da Loja com o Mercado Pago (equivalente ao onboarding Stripe)**
- Novo campo em `domain.Loja`: dados da autorização OAuth — `MercadoPagoAccessToken`,
  `MercadoPagoRefreshToken`, `MercadoPagoUserID` (o `collector_id`), `MercadoPagoTokenExpiraEm`
  (data, pra saber quando precisa renovar — token válido por 6 meses).
- Novo handler `mercadopago_handler.go`, espelhando o padrão de `stripe_handler.go`:
  - `GET /admin/mercadopago/onboarding` — monta a URL de autorização OAuth
    (`https://auth.mercadopago.com.br/authorization?client_id=...&response_type=code&platform_id=mp&redirect_uri=...`)
    e redireciona a loja pra lá.
  - `GET /admin/mercadopago/callback` — recebe o `code` de volta, troca pelo `access_token` via
    `POST https://api.mercadopago.com/oauth/token`, salva os dados na `Loja`.
  - `GET /admin/mercadopago/status` — equivalente ao `/admin/stripe/status` já existente.
- Novo `mercadopago_service.go` com essa lógica de troca de token e chamadas à API.
- **Variáveis de ambiente novas**: `MERCADOPAGO_CLIENT_ID`, `MERCADOPAGO_CLIENT_SECRET` (da
  aplicação "drenux-marketplace" — usar as de produção quando for a hora, não as de teste que já
  usamos nessa conversa).

**5.2 — Backend: checkout e split**
- Trocar a criação de cobrança que hoje usa a Stripe (`/pedidos/:id/checkout`) pra usar a API do
  Mercado Pago, com `application_fee` calculado pela mesma fórmula de plano que já existe (Start:
  `max(pedido × 6,5%, R$2,50)`; Pro/Scale: percentuais já definidos) — só troca o processador por
  trás, a lógica de cálculo de comissão não muda.
- Usar o `access_token` da própria Loja (salvo em 5.1) pra criar o pagamento, não o token da
  plataforma.

**5.3 — Backend: webhook**
- Novo endpoint `POST /webhooks/mercadopago`, substituindo/complementando `/webhooks/stripe`.
- Validar a assinatura do webhook (o Mercado Pago manda uma assinatura no header — verificar antes
  de processar qualquer evento, mesmo padrão de segurança que já fizemos com o `whsec_` da Stripe).
- Escutar pelo menos o evento de pagamento aprovado, pra disparar o mesmo fluxo que já existe hoje
  (desconto de estoque, notificação WhatsApp, incremento de uso de cupom).

**5.4 — Renovação automática do token (a cada 6 meses)**
- Como o `access_token` de cada loja expira em 6 meses, criar uma rotina (cron ou verificação no
  login do admin) que renova via `refresh_token` **antes** de expirar — evitar que uma loja perca a
  capacidade de receber pagamento silenciosamente por token vencido.

**5.5 — Repasse de comissão do afiliado — decisão fechada em 24/07/2026, implementada**
Status: `[x] concluída` (controle manual — não é split automático). Decisão: hipótese 2 do que
estava em aberto (repasse separado, fora do split) — o split 1:N do Mercado Pago exigiria contato
comercial sem prazo garantido, então optamos por controlar isso internamente em vez de esperar.

O que foi implementado:
- `domain/repasse_afiliado.go` (novo): `RepasseAfiliado` — um lançamento por pedido pago via
  Mercado Pago com afiliado vinculado (`PedidoID` com `uniqueIndex`, trava duplicata se a
  notificação do Mercado Pago repetir), `Valor`, `Status` (`pendente`/`pago`), `PagoEm`.
- Fórmula de comissão **não mudou** — extraída de dentro de `transferirComissaoAfiliado`
  (Stripe) pra uma função só, `calcularComissaoAfiliado` (em `stripe_service.go`, ~37,6% da taxa
  de plataforma do plano da loja, igual já era). Usada tanto pelo repasse automático via Stripe
  Transfer quanto pelo registro manual do Mercado Pago — os dois processadores usam exatamente a
  mesma conta. **Nota**: não existe em lugar nenhum do código uma fórmula de "30% do lucro
  líquido" pro Start mencionada num pedido anterior — a fórmula real, em produção, sempre foi
  ~37,6% da taxa de plataforma, igual em qualquer plano; usei essa (a que já existe de verdade),
  não inventei a de "lucro líquido".
- `MercadoPagoService.ProcessarNotificacaoPagamento`: onde antes só logava aviso, agora calcula e
  registra o `RepasseAfiliado` como `pendente` via `RepasseAfiliadoService.RegistrarPendente`.
- `GET /afiliado/repasses` (novo, autenticado pelo token do próprio afiliado): extrato — histórico
  completo + total pendente. Seção nova em `DashboardAfiliado` (frontend), abaixo de "Lojas
  indicadas".
- **Área admin da Drenux — decisão do William**: não existe login de staff da plataforma nesse
  projeto (só dono de loja e afiliado, cada um só vê os próprios dados). Perguntei como proteger a
  tela de "marcar como pago"; decisão foi um secret compartilhado (`DRENUX_ADMIN_SECRET`, header
  `X-Drenux-Admin-Secret`), mesmo padrão do `CronSecret` já usado em `/relatorio/semanal` — mas
  fechando por padrão se a variável não estiver definida (diferente do CronSecret, que abre a rota
  se não tiver secret configurado; aqui expõe dado financeiro de todos os afiliados, então "sem
  secret" tem que significar "fechado", não "aberto por engano"). Rotas novas: `GET
  /drenux/afiliados/pendentes` (visão geral), `GET /drenux/afiliados/:id/repasses` (detalhe), `POST
  /drenux/repasses/marcar-pago` (lote). Frontend: `/drenux/afiliados`, pede o secret uma vez
  (guardado em localStorage via `drenuxAdminStore`), lista afiliados com saldo pendente, expande
  pra ver os lançamentos, marca em lote como pago.
- **Nenhuma chamada de API de pagamento** foi adicionada — o repasse via Pix continua 100% manual,
  fora do sistema, como pedido. Essa tela só registra a confirmação depois do Pix já ter sido
  feito.

Validado com `go build ./...`, `go vet ./...`, `gofmt -l`, `npx tsc -b` e `npm run build`, todos
limpos.

**Correção (24/07/2026): comissão incidindo sobre frete, achada num pedido de auditoria do
William.** Confirmado: `pedido.Total` inclui `TaxaEntrega` (`pedido_service.go`,
`pedido.Total = total + taxaEntrega`). O `marketplace_fee` do Mercado Pago já excluía frete desde
a correção do teste de R$1 (Fase 5, `CriarCheckout`) — isso já estava certo. **Mas a comissão do
afiliado (Fase 5.5, `ProcessarNotificacaoPagamento`) não excluía** — usava `pedido.Total` cheio,
entrou assim junto com a implementação da Fase 5.5 na mesma sessão. Corrigido pra
`pedido.Total - pedido.TaxaEntrega`, mesma base do marketplace_fee. Também corrigidos, por
consistência (mas confirmados como **código morto hoje**, sem rota que os alcance desde a Fase
5.2): `StripeService.CriarCheckout` (application_fee) e `StripeService.transferirComissaoAfiliado`
— os dois também usavam `pedido.Total` cheio. Fórmula da comissão em si não mudou em nenhum dos
três — só a base sobre a qual ela é aplicada. Validado com `go build`/`go vet`/`gofmt`.

### Fase 6 — Combos/Kits e Sugestão Inteligente de produtos
Status: `[x] implementada — validada com go build/go vet e tsc -b/npm run build, sem teste manual
em navegador real ainda`

**Parte 1 — Combos/Kits:**
- `domain/combo.go` (`Combo`, `ComboItem`) e `domain/pedido_combo.go` (`PedidoCombo`,
  `PedidoComboItem` — cópia dos itens do combo no momento da compra, sem `TipoProduto`/
  `PesoGramas` por componente de propósito: combo não suporta o fluxo "guardar e entregar
  depois", ver decisão abaixo). `repository/combo_repository.go`, `service/combo_service.go`
  (CRUD com checagem de dono), `handler/combo_handler.go`, rotas `admin.GET/POST/PUT/DELETE
  /combos`.
- `PedidoService.CriarPorSlug` ganhou `input.Combos []ComboPedidoInput` — valida dono/
  disponibilidade do combo, resolve variação escolhida por componente (por `ComboItemID`, não
  `ProdutoID`, pra não dar ambiguidade se o mesmo produto aparecer duas vezes no combo), checa
  estoque de cada componente (sem decrementar — desconto real só depois do pagamento, mesmo
  padrão dos itens avulsos) e soma `combo.Preço × quantidade` ao total.
- **Decisão de escopo**: pedido em modo "guardar" com combo é bloqueado explicitamente
  (`"combos não podem ser guardados pra entrega depois"`) — estender o fluxo de guardados pra
  combo exigiria levar `TipoProduto`/`PesoGramas` até `PedidoComboItem` também, e isso não foi
  pedido.
- `PosPagamentoService.ProcessarPedidoPago` ganhou um segundo laço pra descontar estoque dos
  componentes de cada `PedidoCombo` (reaproveitando a mesma função de desconto+alerta dos itens
  avulsos, extraída pra `descontarEstoque`/`notificarAlertaEstoque`).
- Cardápio público: `GET /lojas/:slug` agora devolve `combos` (só os disponíveis);
  `ComboCard.tsx` novo (cliente escolhe variação de cada componente, igual comprando avulso),
  seção "Combos" em `CardapioPublico.tsx` acima do cardápio normal, com selo "Combo".
  `Combos.tsx` novo no admin (mesmo padrão visual de `Cupons.tsx`).

**Parte 2 — Sugestão Inteligente:**
- `domain/sugestao_produto.go` (`SugestaoProduto`, vínculo manual `ProdutoOrigemID` →
  `ProdutoSugeridoID` por loja, com desconto opcional `percentual`/`fixo` que só se aplica via
  sugestão). `service/sugestao_produto_service.go`: `Criar` bloqueia produto-sugere-a-si-mesmo e
  uma segunda sugestão da mesma categoria pra mesma origem (mesma regra que o carrinho aplica na
  exibição). `MontarSugestoesCarrinho(lojaID, produtosNoCarrinho)` monta a seção consolidada:
  nunca sugere produto já no carrinho (avulso ou componente de combo), nunca duplica o mesmo
  produto sugerido por origens diferentes, e no máximo uma sugestão por categoria — sempre a de
  maior desconto em R$ quando há disputa pela mesma vaga (mesmo produto ou mesma categoria).
- `GET /lojas/:slug/sugestoes-carrinho?produtos=1,2,3` (rota pública) devolve `[]` sem erro se a
  loja não tiver a Sugestão Inteligente ativa — o frontend não precisa checar o flag antes de
  chamar. Consumida em `CarrinhoDrawer.tsx` na revisão do carrinho (etapa "carrinho", antes de
  ir pra "dados"), não em popup por produto.
- `ItemPedidoInput.SugestaoProdutoID` — ao criar o pedido, o backend revalida o vínculo (pertence
  à loja, aponta pra esse produto sugerido, e o produto de origem realmente está no pedido) antes
  de aplicar o desconto; se não bater, ignora silenciosamente e cobra o preço cheio — proteção
  contra alguém forjar esse campo direto na API pra ganhar desconto indevido.
- `SugestaoInteligente.tsx` novo no admin: por produto, configura até 5 sugestões (limite fixo
  V1), bloqueando na UI o próprio produto e produtos de categoria já usada por outra sugestão
  daquela origem, com desconto opcional. O toggle geral "Sugestão Inteligente" (liga/desliga as
  sugestões no carrinho do cliente) vive em `Configuracoes.tsx`, não nessa tela — motivo: `PUT
  /admin/loja` substitui a configuração inteira da loja de uma vez (mesmo padrão de todo o resto
  dessa tela), então o toggle precisa fazer o round-trip do valor atual junto com as outras
  configurações pra não ser resetado pra `false` toda vez que o lojista salva qualquer outra
  coisa — só faz sentido morar onde esse round-trip já acontece.

**Parte 3 — Cobrança do recurso pago:**
- `domain/configuracao_plataforma.go`: `ConfiguracaoPlataforma` — linha única (`ID: 1`,
  garantida no boot via `FirstOrCreate` em `main.go`, sem sobrescrever se já existir) com
  `SugestaoInteligentePrecoMensal` (padrão `19.90`). Lido em `GET /admin/configuracao-plataforma`
  — nunca hardcoded no frontend, ajustável direto no console do Neon sem redeploy, como pedido.
- `Loja.SugestaoInteligenteContratada`/`SugestaoInteligenteContratadaEm`/
  `SugestaoInteligenteAtiva` — o toggle de ativar (Parte 2) só é aceito se `Contratada` for
  `true` (`LojaService.AtualizarConfiguracoes` força `false` se não for, relendo o estado atual
  da loja em vez de confiar no que veio do formulário).
- **Mecanismo de cobrança da assinatura Drenux existente, conforme pedido**: a assinatura de
  plano (Pro/Scale) já usa cobrança recorrente de verdade via Stripe Subscriptions
  (`StripeService.CriarCheckoutAssinatura`/`MudarPlano`, webhook de renovação atualizando
  `Loja.Plano` a cada ciclo). Esse mecanismo **poderia** ser reaproveitado pra Sugestão
  Inteligente (um segundo Price/Subscription na mesma conta Stripe do lojista), mas isso não foi
  implementado nessa fase, conforme pedido explicitamente. **Hoje, `SugestaoInteligenteContratada`
  não tem nenhuma tela ou fluxo que o ligue** — é um campo puro no banco, pensado pra ser
  ligado manualmente (ex: direto no Neon) até uma decisão sobre reaproveitar o Stripe de
  assinatura ou cobrar separado. Nenhuma cobrança automática foi criada.

**Ressalvas antes de confiar em produção:**
1. Nenhum teste manual em navegador real foi feito — só `go build`/`go vet`/`tsc -b`/`npm run
   build`, todos limpos. Testar o fluxo completo (montar combo com variação, aplicar sugestão no
   carrinho, checkout com combo+sugestão juntos) antes de anunciar pro lojista.
2. Cupom de desconto aplicado sobre pedido com combo: a lógica de cupom opera sobre a variável
   `total` (que já soma itens avulsos + combos antes do cupom ser aplicado) — não foi criado
   nenhum caso especial pra combo, deveria funcionar igual a um pedido só de itens avulsos, mas
   não foi testado com combinação real.
3. `SugestaoInteligenteContratada` não tem fluxo de contratação nenhum (nem manual documentado
   além de "mexer direto no banco") — William precisa decidir como/quando uma loja passa a ter
   esse campo `true` antes de vender o recurso de verdade.

### Fase 7 — Reestruturação de planos (Start/Basic/Pro/Scale) e afiliados
Status: `[x] concluída — 7.1 a 7.5 implementadas (12/08/2026)`. **Confirmado em produção em
17/08/2026** (auditoria pontual, fora do fluxo normal de fase): 7.1 (Basic existe e aparece no
site), 7.2 (matriz de recursos/comissão escalonada refletida na página de planos) e 7.4 (gates de
plano — avisos de WhatsApp por plano, selo "Feito com Drenux") verificados direto no
drenux.com.br. 7.3 (limite do Start) e 7.5 (fórmula de afiliado/bônus) **não foram reverificados
nessa auditoria** — seguem só com a validação de build da implementação original, sem teste contra
um mês real batendo o limite nem uma indicação de afiliado com transação de verdade.

**O que foi feito nessa sessão (7.1 + 7.2):**

- **Planos renomeados/redefinidos em todo lugar que lia `Loja.Plano`**: `ordemPlano` (agora
  `start:0, basic:1, pro:2, scale:3`), validação nos dois handlers (`plano_handler.go` —
  `mudarPlanoRequest` ganhou `basic` no `oneof`; `checkoutAssinaturaRequest` continua só
  `pro scale`, de propósito — Basic não tem mensalidade, não passa por checkout), `domain/loja.go`
  (comentário do campo `Plano`), `lib/planos.ts`, `MeuPlano.tsx`, `Planos.tsx`, `Inicio.tsx`,
  `Cadastro.tsx` (as três cópias separadas do mapa de nome de plano — `NOME_PLANO`/`NOMES_PLANO` —
  foram consolidadas num export só em `lib/planos.ts`). Mensalidade do Pro caiu de R$129 pra R$99
  (`valoresMensalidadePlano`, `stripe_service.go`), Scale continua R$349.
- **Start deixou de ter comissão** — removida a `pisoComissaoStart`/`calcularMarketplaceFee`
  antiga (piso de R$2,50 + 6,5% flat). Start agora não passa pela tabela de comissão nenhuma (sem
  split de pagamento, por definição da Fase 7.1).
- **Comissão escalonada por faixa de GMV mensal** (`calcularComissaoEscalonada`, nova, em
  `mercadopago_service.go`): tabela `faixasComissaoPorPlano` com as faixas exatas de
  `docs/drenux-planos-comissoes-definido.md` § 2 (Basic 2,4%/1,5%/1,3%; Pro 1,8%/1,2%/1,05%; Scale
  1,1% flat). A comissão é **marginal** (igual imposto progressivo) — se o valor de um pedido
  cruza o teto de uma faixa dentro do mês, a fatia de cada lado paga o percentual da própria
  faixa, não um "tudo ou nada" pela faixa mais alta atingida.
- **GMV do mês corrente, novo**: `PedidoRepository.SomarGMVMesAtual` (novo) — soma
  `total - taxa_entrega` de pedidos pagos desde o dia 1º do mês corrente (fuso
  `America/Sao_Paulo`). **Achado importante durante a implementação**: `DashboardData.TotalMes`
  (usado em Início/Meu Plano) é uma **janela rolante de 30 dias**, não um mês-calendário — não dá
  pra reaproveitar pra decidir faixa de comissão (que precisa resetar todo dia 1º, por definição da
  Fase 7.2) nem pro contador de pedidos do Start (Fase 7.3, ainda não implementada). Ficam como
  números diferentes de propósito, os dois continuam existindo.
- **Ordem de leitura do GMV corrigida no webhook**: `MercadoPagoService.ProcessarNotificacaoPagamento`
  lia (teria lido, se implementado ingenuamente) o GMV do mês DEPOIS de marcar o próprio pedido
  como pago, contando-o a mais nele mesmo. Corrigido lendo o GMV antes de `AtualizarStatus`.
- **`calcularComissaoAfiliado` reescrita** pra aplicar `proporcaoAfiliado` sobre a comissão real
  calculada por `calcularComissaoEscalonada`, em vez de recalcular um percentual próprio — nunca
  mais pode divergir do valor cobrado da loja. Ganhou o parâmetro `gmvAntes`; os dois call sites
  (`repasse_afiliado_service.go`, e o `transferirComissaoAfiliado` morto da Stripe) foram
  atualizados. **Fórmula do afiliado em si não mudou** (continua ~37,6% da comissão, por decisão —
  isso é Fase 7.5, não essa).
- **`MudarPlano` (troca de plano pelo dono, `stripe_service.go`) ganhou um caminho novo**: troca
  entre Start↔Basic (os dois sem mensalidade) agora é aplicada na hora, sem passar pelo Checkout
  da Stripe — só Pro/Scale continuam exigindo cartão. Downgrade de um plano pago pra Basic (ou pra
  Start) continua agendado pro fim do ciclo, igual já era pro Start; `processarRenovacaoAssinatura`
  e `LimparAssinatura` foram generalizados pra aceitar qualquer plano-sem-mensalidade como destino
  (antes só reconheciam "start" hardcoded).
- **Cadastro pode nascer direto em Basic**: `cadastroRequest`/`CadastroInput` ganharam `Plano`
  opcional (`"start"` por padrão, ou `"basic"`) — sem isso, o botão "Escolher Basic" da Planos.tsx
  levaria pro cadastro normal e a loja nasceria silenciosamente no Start. Usa um parâmetro de URL
  próprio (`?plano_desejado=`), separado do `?plano=` que já existia pro banner de "pagamento
  confirmado" do checkout de Pro/Scale — são fluxos diferentes (um tem pagamento, o outro não).
- **Frontend**: `lib/planos.ts` reescrito — `PlanoInfo.taxa` (número único) virou
  `PlanoInfo.faixas` (array de faixas), com `custoPlano`/nova `taxaEfetivaPlano` fazendo o mesmo
  cálculo marginal do backend. `planoMaisBarato` e o "mais barato"/"recomendado" dos cards agora
  **excluem o Start da comparação** — como ele não tem comissão nem processa pagamento, ele sempre
  "venceria" a comparação por ser sempre R$0, o que não faz sentido (não é uma alternativa
  comparável de custo de processamento, é uma ausência de processamento).

**Ressalvas antes de confiar em produção:**
1. Nenhum teste manual em navegador — só `go build`/`go vet`/`gofmt -l` (limpos) e
   `npx tsc -b`/`npm run build` (limpos) — validar o fluxo completo (simular troca de plano,
   conferir a comissão de um pedido real que cruza um teto de faixa) antes de anunciar.
2. Preço da assinatura Pro na Stripe é cacheado por `lookup_key` (`drenux_pro_mensal`) — se algum
   Price de teste com o valor antigo (R$129) já existir numa conta Stripe usada nessa integração,
   ele **não atualiza sozinho** (Price é imutável na Stripe); precisaria criar um novo Price/
   lookup_key ou arquivar o antigo manualmente antes de usar em produção. Não é um problema de
   código, é operacional.
3. `LojaRepository.CancelarAssinaturaPlano` (chamado quando uma assinatura MP de Pro/Scale é
   cancelada ou pausada pelo próprio Mercado Pago) continua revertendo pra **Start**, não Basic —
   decisão conservadora, não especificada em nenhum dos dois documentos de planejamento; se o
   William preferir que caia em Basic (mantém split ativo mesmo sem mensalidade), é uma troca de
   uma linha em `loja_repository.go`.
4. Fase 7.5 (nova fórmula de afiliado) — ver detalhe abaixo, implementada em 12/08/2026.

**Fase 7.5 — nova fórmula de afiliado + bônus de ativação (implementada em 12/08/2026):**

- **Fórmula: 45% do LUCRO LÍQUIDO** (era 37,6% do bruto) — `calcularComissaoAfiliado`
  (`stripe_service.go`) agora calcula `lucroLiquido = max(0, comissaoEscalonada − custo real do
  Mercado Pago sobre o mesmo valor)` e só depois aplica a proporção do afiliado. Constante nova
  `custoRealMercadoPagoPercentual = 0.99`. `ProporcaoComissaoAfiliadoPadrao` (valor sugerido no
  cadastro de afiliado novo) foi de `0.376` pra `0.45` — cada afiliado continua podendo ter a
  própria proporção negociada (`Afiliado.ComissaoPercentual`), isso não mudou, só o padrão sugerido
  e a base de cálculo. `ComissaoAfiliadoPercentual` (constante puramente documentativa, sem uso no
  código, baseada no modelo antigo) foi removida — não fazia mais sentido nenhum número ilustrativo
  único depois da comissão virar escalonada (Fase 7.2) E a base virar líquido (Fase 7.5) ao mesmo
  tempo.
- **Bônus de ativação** (pagamento único: Basic R$60 · Pro R$150 · Scale R$400, só se a loja
  indicada atingir 30 pedidos nos primeiros 60 dias desde o cadastro): `domain.RepasseAfiliado`
  ganhou `Tipo` (`recorrente` | `bonus_ativacao`) e `PedidoID` virou ponteiro (nulo pra bônus, que
  não é atrelado a um pedido — o Postgres trata cada `NULL` como distinto numa `uniqueIndex`, então
  isso não quebra a trava de duplicata dos lançamentos recorrentes). `RepasseAfiliadoService.
  VerificarBonusAtivacao` (novo) é chamado a cada pedido pago via Mercado Pago de loja com afiliado
  (mesmo ponto de `RegistrarPendente`, em `MercadoPagoService.ProcessarNotificacaoPagamento`) —
  idempotente via `RepasseAfiliadoRepository.ExisteBonusAtivacao`. Novo
  `PedidoRepository.ContarPedidosDesde` (conta pedidos a partir de uma data arbitrária, diferente do
  `ContarPedidosMesAtual` da Fase 7.3, que é sempre mês-calendário).
- **Start nunca gera nada disso** (nem recorrente nem bônus) — já era assim pro recorrente (sem
  split, comissão sempre zero) e o bônus só é alcançável depois que a loja tem pelo menos um pedido
  pago via Mercado Pago, o que por definição exige ter saído do Start.
- **Painel `/drenux/afiliados` e `/afiliado/dashboard`**: não precisaram recalcular nada — os dois
  só leem `RepasseAfiliado.Valor`, já gravado com a fórmula nova no momento do registro. Só
  atualizada a exibição da lista pra mostrar "bônus de ativação" em vez de "pedido #null" quando
  `tipo === 'bonus_ativacao'`. Formulário de cadastro de afiliado (`Afiliados.tsx`) — valor padrão
  sugerido e o texto do rótulo atualizados de 37,6%/bruto pra 45%/líquido.
- **Não implementado, como o roadmap já previa que ficasse pra depois**: bônus de upgrade (loja
  indicada migra de plano) — valor ainda não definido, fora de escopo desta fase.

Validado com `go build ./...`, `go vet ./...`, `gofmt -l` (limpo nos arquivos tocados),
`npx tsc -b` e `npm run build`, todos limpos. Nenhum teste de ponta a ponta contra um pagamento
real do Mercado Pago — o cenário completo (loja nova, afiliado vinculado, 30 pedidos em 60 dias)
não foi simulado nessa sessão. Como o roadmap já registrava, hoje só existe 1 afiliado cadastrado e
nenhuma indicação ativa, então não há risco de a mudança de fórmula quebrar cálculo de comissão já
em andamento pra ninguém.

Com isso, as cinco sub-fases da Fase 7 (7.1–7.5) estão implementadas.

**Fase 7.4 — gates de funcionalidade por plano (implementada em 12/08/2026):**

Antes de implementar, investiguei o estado real do código (o roadmap tinha suposições desatualizadas,
igual já tinha acontecido nas fases anteriores) e confirmei com o William dois pontos em aberto:

- **"Gestão de entregadores"**: hoje não existe entidade `Entregador` nenhuma (login, cadastro,
  atribuição) — é um link de compartilhamento de localização por pedido (`AtualizarLocalizacao`/
  `CompartilharLocalizacao.tsx`), já funcional de ponta a ponta antes desta fase. **Confirmado com o
  William**: esse mecanismo já conta como a "gestão de entregadores" da matriz — não construir uma
  entidade própria agora.
- **Achado à parte, corrigido junto**: `leaflet`/`react-leaflet` são usados em `RastrearPedido.tsx`
  mas não estavam declarados no `frontend/package.json` (só sobravam no cache do Vite) — um `npm
  install` limpo quebraria essa página. Adicionados `leaflet`, `react-leaflet` e `@types/leaflet`
  como dependências reais.

**O que foi implementado:**

1. **Avisos automáticos de status via WhatsApp (Start ❌, Basic+ ✅)** — `PosPagamentoService.
   notificarPagamento` (confirmação de pagamento) e `PedidoHandler.notificarSaiuParaEntrega` (saiu
   pra entrega) agora pulam o aviso ao CLIENTE se `loja.Plano == "start"`. O aviso ao DONO da loja
   (`EnviarNotificacaoAdmin`) não é gateado — não é um recurso de plano, é operação essencial da
   loja, roda sempre.
2. **Rastreamento em tempo real (Pro/Scale)** — já existia pronto (mapa Leaflet, polling a cada
   10s, compartilhamento de GPS a cada 25s), só faltava o gate:
   - Backend: `AtualizarLocalizacao` (Pedido e Solicitação) recusa com 403 se o plano não for
     Pro/Scale. `Rastrear` (rota pública) devolve `disponivel: bool` em vez de erro — permite o
     frontend mostrar uma mensagem, não uma tela de erro. Helper `rastreamentoDisponivel(plano)`
     compartilhado entre `PedidoHandler` e `SolicitacaoHandler`.
   - A mesma mensagem de WhatsApp de "saiu pra entrega" (item 1) agora só inclui o link de
     rastreamento se o plano for Pro/Scale — Basic recebe o aviso de status sem o link.
   - Frontend: `RastrearPedido.tsx` mostra "rastreamento não disponível" em vez do mapa quando
     `disponivel` é `false`. `CompartilharLocalizacao.tsx` — pra quem não é Pro/Scale, o botão vira
     "Marcar como saiu para entrega" (só o status, sem tentar GPS, que o backend recusaria mesmo).
3. **Limite de 30 produtos (Start)** — não existia nenhuma checagem antes. `ProdutoRepository.
   ContarPorLoja` (novo) + `ProdutoService.validarLimiteProdutosStart`, chamada no início de
   `Criar`. Basic/Pro/Scale continuam ilimitados. **Decisão confirmada com o William**: a matriz
   fala em "produtos/categorias" como uma linha só, mas o limite vale só pra **produtos** —
   categorias continuam ilimitadas em qualquer plano, de propósito (categoria é estrutura do
   cardápio, não um item que "ocupa espaço" do mesmo jeito; limitar ela tende a atrapalhar
   organização sem ganho real). Confirmado via `CadastroEmMassaDialog.tsx` (Fase 3.2) que não existe
   nenhum caminho de criação de produto que passe por fora de `ProdutoService.Criar` — o cadastro em
   massa chama a mesma mutation produto a produto, então o limite vale igual lá.
4. **Marca "Feito com Drenux" (visível até Basic, removível a partir do Pro)** — não existia antes.
   `CatalogoHandler.BuscarCardapio` agora computa `mostrar_selo_drenux` (`plano != "pro" && plano !=
   "scale"`) na resposta pública; `CardapioPublico.tsx` mostra um rodapé discreto linkando pra `/`
   quando `true`.

Validado com `go build ./...`, `go vet ./...`, `gofmt -l` (limpo nos arquivos tocados),
`npx tsc -b` e `npm run build`, todos limpos. Nenhum teste manual em navegador (em especial, o mapa
Leaflet e o compartilhamento de GPS real não foram testados num device de verdade nessa sessão).

**Fase 7.3 — contador, aviso e bloqueio do limite do Start (implementada em 12/08/2026):**

- **`domain/loja.go`**: dois campos novos, `AvisoLimitePedidosEm` (quando o aviso de "passou de 30"
  foi mandado) e `PedidosBloqueadosDesde` (quando o bloqueio de fato passou a valer — `nil` =
  liberado). Os dois só valem pro mês em que foram gravados — se a data for de um mês anterior,
  quem lê trata como se não existisse, sem precisar de nenhuma limpeza manual quando a cota reseta
  no dia 1º. Constantes `LimitePedidosStart` (30) e `LimiteToleranciaBloqueioPedidos` (3 dias)
  também ficaram em `domain` (não em `service`/`repository`) porque as duas camadas precisam do
  mesmo valor.
- **`PedidoRepository.ContarPedidosMesAtual`** (novo): conta pedidos não cancelados do mês
  corrente por `created_at` (não `updated_at`/status pago — o Start não tem pagamento integrado
  nenhum, então "pedido recebido" é o que importa, não "pedido pago"). Reaproveitei/limpei o
  cálculo de início de mês que já existia em `SomarGMVMesAtual` (Fase 7.2), agora numa função só
  (`inicioMesCalendario`).
- **Aviso inicial, reativo, em `PedidoService.CriarPorSlug`** (`verificarLimiteStart`): só se
  aplica a lojas Start. Se a loja já estiver bloqueada nesse mês, recusa o pedido (mensagem neutra
  pro cliente final, sem menção a plano). Senão, se esse pedido é o que estoura os 30 do mês e
  ainda não tinha avisado nesse mês, dispara o aviso por WhatsApp (`EnviarTextoAdmin`, mesmo padrão
  de `avisarPagamentoNaoConfigurado`) e deixa o pedido passar normalmente — não bloqueia na hora,
  como pedido.
- **Bloqueio proativo, por rotina agendada** (`LojaService.VerificarLimiteStart`, novo endpoint
  `POST /drenux/lojas/verificar-limite-start`, protegido por `X-Cron-Secret` igual
  `/mercadopago/renovar-tokens` — aberto se o secret não estiver configurado, mesmo padrão dos
  outros crons): pensada pra rodar 1x/dia via cron externo (cron-job.org). Só essa rotina bloqueia
  de verdade (marca `PedidosBloqueadosDesde`) e avisa o dono — o motivo de precisar de uma rotina
  agendada em vez de só checar tudo na hora do pedido é que o dono precisa saber que os pedidos
  pararam mesmo que nenhum cliente novo tente pedir nos 3 dias seguintes ao aviso (senão ele só
  descobre quando um cliente reclamar).
- **Contador exposto em `GET /admin/loja`** (`LojaHandler.Buscar`/`lojaResponse`): `pedidos_mes_atual`
  e `limite_start_bloqueado` (computados em `LojaService.LimitePedidosStart`, sempre, não só pra
  Start — o frontend decide o que mostrar). `limite_start_bloqueado` já vem com a checagem de "é do
  mês corrente" aplicada no backend — o frontend não precisa (nem deve) recalcular isso sozinho.
- **Frontend (`Inicio.tsx`)**: contador "X/30 pedidos este mês" sempre visível pra loja Start (não só
  perto do limite, como pedido), com aviso em tom de lembrete (não punição) quando passa de 30, e
  banner mais forte quando `limite_start_bloqueado` é `true` — mesma frase exata do WhatsApp, pra
  não ter uma mensagem no painel e outra no WhatsApp.

Validado com `go build ./...`, `go vet ./...`, `gofmt -l` (limpo nos arquivos tocados) e
`npx tsc -b`/`npm run build`, todos limpos. Nenhum teste manual em navegador nem contra um cron
real — não dá pra simular "3 dias depois" sem esperar de verdade ou mexer no relógio do banco.

Combinado numa conversa separada com o Claude (chat) em ago/2026, fechada em
`docs/drenux-planos-comissoes-definido.md` (referência completa de números e posicionamento
competitivo — este bloco aqui é a tradução pra tarefa de implementação). Objetivo geral: sair de 3
planos (Start/Pro/Scale, comissão fixa sem teto) pra 4 planos, com comissão escalonada por faixa de
GMV mensal, nunca abaixo do custo real do Mercado Pago (~0,99%), e desacoplando "cardápio" de
"pagamento integrado" — quem não quer split de pagamento nenhum tem opção 100% grátis pra sempre.

**Antes de começar**: reler `docs/drenux-planos-comissoes-definido.md` inteiro — ele tem o raciocínio
completo (por que cada número foi escolhido, comparação com concorrentes, guardrails financeiros) que
não foi todo reproduzido aqui.

**7.1 — Renomear e redefinir os planos**

| Plano | Mensalidade | Split de pagamento | Pedidos |
|---|---|---|---|
| Start | R$0 | Não — loja recebe Pix na própria chave | 30/mês, recorrente (renova todo ciclo) |
| Basic | R$0 | Sim | Ilimitado |
| Pro | R$99 | Sim | Ilimitado |
| Scale | R$349 | Sim | Ilimitado |

- Onde hoje existe `Start/Pro/Scale` (`lib/planos.ts`, `MeuPlano.tsx`, `Planos.tsx`, e qualquer
  lugar que leia `Loja.Plano`), vira `Start/Basic/Pro/Scale`. Verificar todo lugar que faz
  `switch`/`if` no valor do plano antes de assumir que são só esses três arquivos.
- `Loja.Plano` (ou equivalente) precisa aceitar o valor novo `basic`. Migração de dado: nenhuma loja
  real em produção ainda (confirmado nas Fases 1 e 5), então não precisa de script de migração de
  lojas existentes — só ajustar o enum/validação.
- **Start deixa de ser o plano "com comissão"** — hoje `Start = max(pedido × 6,5%, R$2,50)`; no
  modelo novo, Start não tem comissão nenhuma porque não tem split de pagamento nenhum. Quem hoje
  seria "Start" no sentido antigo (comissão sem mensalidade) vira o novo **Basic**.

**7.2 — Comissão escalonada por faixa de GMV mensal**

Substitui o percentual fixo atual. Faixas (sobre o GMV mensal da loja, resetando a cada mês —
confirmar com o código de fechamento mensal/relatório que já existe, tipo o que gera
`dashboard.total_mes`, se dá pra reaproveitar a mesma lógica de "GMV do mês corrente"):

- **Basic**: R$0–5.000 → 2,4% · R$5.000–20.000 → 1,5% · acima de R$20.000 → 1,3%
- **Pro**: R$0–5.000 → 1,8% · R$5.000–20.000 → 1,2% · acima de R$20.000 → 1,05%
- **Scale**: 1,1% flat sobre todo o GMV, sem faixa

Guardrail obrigatório: nenhuma faixa pode ficar abaixo do custo real do Mercado Pago (~0,99%) — as
faixas acima já respeitam isso com margem, não alterar sem recalcular contra
`docs/drenux-planos-comissoes-definido.md` § 4.

- `marketplace_fee`/`application_fee` calculado no checkout (Fase 5.2, `mercadopago_service.go`)
  precisa mudar de percentual fixo por plano pra essa tabela escalonada — provavelmente uma função
  `calcularComissaoEscalonada(plano string, gmvMesAcumulado float64, valorPedido float64) float64`
  que decide em qual faixa o pedido cai dado o GMV já acumulado no mês antes desse pedido.
- **Simplificação importante**: a distinção antiga "Start absorve a taxa do processador / Pro-Scale a
  loja absorve" deixa de existir — em todos os planos pagos (Basic/Pro/Scale), a Drenux absorve a
  taxa do Mercado Pago de dentro da própria comissão (é por isso que as faixas têm o guardrail acima
  do custo). Se existir algum branch de código condicionando isso por plano, remover.

**7.3 — Limite do Start: contador, aviso e bloqueio**

- Contador de pedidos do mês corrente, visível no painel do lojista Start (ex: "18/30 pedidos este
  mês") — sempre visível, não só quando perto do limite.
- Ao passar de 30 pedidos no mês: aviso no painel + notificação (WhatsApp, reaproveitando o
  whatsmeow já usado no projeto) sugerindo ativar o Basic. Pedidos continuam sendo aceitos
  normalmente nesse momento — não bloqueia na hora.
- 3 dias corridos depois do aviso, se a loja não tiver ativado o Basic: bloquear novos pedidos (não
  afeta pedidos já em andamento) até (a) a loja ativar o Basic, ou (b) o mês virar e a cota resetar
  — o que vier primeiro. Se a loja bate o limite perto do fim do mês, o reset natural pode chegar
  antes dos 3 dias — comportamento esperado, não é bug.
- Precisa de alguma rotina agendada pra checar "loja passou de 30 há mais de 3 dias e ainda não
  ativou Basic" — ver se dá pra reaproveitar o mecanismo de cron que já existe no projeto
  (cron-job.org, mencionado no contexto de outras rotinas) em vez de criar um novo.
- Mensagem de bloqueio (painel + WhatsApp) precisa soar como lembrete, não penalidade — ex: "Sua
  loja atingiu o limite do plano grátis — ative o Basic agora (sem custo) e volte a receber pedidos
  na hora." Nunca "conta bloqueada" ou tom de punição.

**7.4 — Gates de funcionalidade por plano**

Matriz completa em `docs/drenux-planos-comissoes-definido.md` § "Matriz de funcionalidades". Os que
mexem em comportamento novo (o resto da matriz é feature já existente, só precisa checar o plano
antes de mostrar/permitir):
- Avisos automáticos via WhatsApp pro cliente (status do pedido): Start não tem, Basic em diante tem.
  Se esse recurso já existe e hoje roda pra todo mundo, precisa virar condicional por plano.
- Rastreamento de entrega em tempo real (cliente vê no mapa) e gestão de entregadores: **recurso
  novo, não existe ainda no sistema** — Pro e Scale. Isso é maior que os outros itens dessa fase
  (envolve localização ao vivo, provavelmente reaproveitando Leaflet/Nominatim que já está no
  stack) — se o William quiser, vale quebrar isso numa sub-fase própria (7.4.1) em vez de fazer
  junto com o resto, dado o tamanho.
- Start: até 30 produtos/categorias (hoje sem esse limite pra ninguém — checar onde cadastro de
  produto valida hoje e adicionar a checagem por plano).
- Marca "Feito com Drenux": removível a partir do Pro (checar se esse selo já existe no cardápio
  público hoje ou se precisa ser criado do zero).

**7.5 — Afiliados: nova fórmula (corrige a fórmula real, não uma hipotética)**

**Importante**: a Fase 5.5 registra que a fórmula em produção hoje é `calcularComissaoAfiliado`
(originalmente em `stripe_service.go`) = **~37,6% da taxa de plataforma (comissão bruta), igual pra
qualquer plano**. Uma conversa anterior com o William partiu do pressuposto de que o Start usava uma
fórmula diferente ("30% do lucro líquido") — isso nunca existiu no código, é engano de contexto, não
alterar com base nele. A mudança abaixo é a partir da fórmula real (37,6% do bruto), não dessa
hipotética.

- Nova fórmula, substituindo `calcularComissaoAfiliado`: **45% do lucro líquido** (comissão da loja
  no mês − custo real do Mercado Pago, ~0,99% do GMV), aplicada igual pra Basic/Pro/Scale. Mensalidade
  fixa (Pro/Scale) continua **fora** da base de cálculo do afiliado, como já era.
  - Start não gera comissão nenhuma (sem split), então não gera repasse recorrente de afiliado
    enquanto a loja ficar só no Start — só passa a gerar quando a loja indicada ativar Basic/Pro/Scale.
  - Motivo da mudança: com a comissão bruta caindo (Fase 7.2), pagar 37,6% do bruto passou a poder
    exceder a margem real da Drenux em faixas de comissão mais finas (Pro/Scale nas faixas
    superiores) — % do líquido nunca deixa isso acontecer, em nenhum volume, por construção.
- **Novo**: bônus de ativação, pagamento único, quando a loja indicada completa onboarding + atinge
  um mínimo de pedidos nos primeiros 60 dias (sugestão: 30 pedidos — mesmo número do limite do Start,
  ajustar se fizer mais sentido outro valor). Só é pago se a loja ativar Basic, Pro ou Scale (Start
  puro não gera bônus, pelo mesmo motivo do item acima).
  - Basic: R$60 · Pro: R$150 · Scale: R$400
  - Precisa de um jeito de registrar isso — provavelmente estender `RepasseAfiliado` com um campo de
    tipo (`recorrente` vs `bonus_ativacao`) em vez de criar tabela nova, mas confirmar se o modelo
    atual comporta isso de forma limpa antes de decidir.
- **Em aberto, não implementar ainda**: bônus de upgrade (loja indicada migra de Start/Basic pra
  Pro/Scale) — ideia validada com o William, valor ainda não definido. Não travar o resto da fase
  nisso.
- Área `/drenux/afiliados` (painel interno, Fase 5.5) precisa refletir a fórmula nova no cálculo
  exibido — conferir se o valor é calculado ali ou só lido do que já foi gravado em
  `RepasseAfiliado.Valor` no momento do pedido (nesse caso a UI não muda, só a função que gera o
  registro).

**Status dos afiliados no momento desta fase**: 1 afiliado cadastrado, nenhuma indicação iniciada
ainda — sem risco de quebrar cálculo em produção pra afiliado ativo. Avaliar com o William se vale
lançar com bônus dobrado só pro primeiro lote ("condição de fundador"), decisão dele, não bloqueia
a implementação técnica.

**Sugestão de quebra em sessões**, dado o tamanho da fase (o William pode pedir tudo de uma vez, mas
por padrão seguir o mesmo critério de "uma fase por vez" das instruções lá em cima, tratando
7.1–7.5 como sub-fases nesse espírito):
1. 7.1 + 7.2 (nomes e taxas) — maior impacto financeiro, testar isolado primeiro.
2. 7.3 (contador/aviso/bloqueio do Start) — depende de 7.1 existir.
3. 7.4 (gates de funcionalidade) — rastreamento de entrega pode virar sub-fase própria (7.4.1) por
   causa do tamanho, ver nota acima.
4. 7.5 (afiliados) — independente das outras, pode ser feita em paralelo/antes se o William preferir.

### Fase 9 — Controle de estoque (Pro relatório / Scale completo)
Status: `[x] 9.1, 9.2 (só XML de NF-e) e 9.3 concluídas e testadas em 18/08/2026 — fase encerrada
nesse escopo. Duas coisas ficam pendentes de propósito, cada uma esperando decisão do William antes
de qualquer código: a via "PDF via IA" da 9.2 (provedor/credenciais) e a 9.4 completa (multi-loja —
achado que não existe nem como base hoje, precisa de spec própria, ver nota na 9.4)`

**Importante, ler antes de mexer em qualquer coisa nessa fase**: quando essa fase foi rascunhada
(ago/2026), o Claude (chat) supôs que controle de estoque seria construído do zero. Auditoria
confirmou que **isso já existe e roda em produção**, só nunca foi documentado em nenhum arquivo do
roadmap — mesmo tipo de desconexão que já aconteceu com a Fase 7 antes. A partir de agora, o que
já existe fica registrado abaixo; não reconstruir.

**O que já existe (confirmado por auditoria, NÃO reconstruir)**
- `frontend/src/pages/admin/Estoque.tsx` — aba no menu, visível só pra Pro/Scale.
- **Pro**: relatório somente leitura — lista de itens com estoque controlado, status "Esgotado" /
  "Estoque baixo — N unidade(s)" / "N unidade(s)". Não edita por lá.
- **Scale**: tudo do Pro, mais botão "Gerenciar" → modal com "Repor estoque" (soma um delta, motivo
  opcional) e "Ajustar pra um valor" (define valor absoluto, motivo **obrigatório**), mais aba de
  histórico de movimentação.
- `domain.Produto` e `domain.VariacaoProduto` já têm `EstoqueAtual *int` + `EstoqueAlerta *int`
  (nil = sem controle/ilimitado). Se a variação tiver valor preenchido, tem precedência sobre o
  estoque geral do produto. `Combo` não tem campo de estoque — desconta dos produtos-componente na
  hora da venda.
- Desconto na venda é **atômico via SQL** (`GREATEST(estoque_atual - ?, 0)`) — já evita race
  condition entre pagamentos concorrentes. O ponto de atenção que eu tinha levantado antes
  ("dois clientes pegam a última unidade ao mesmo tempo") **já está resolvido**, não é mais item
  em aberto.
- Ao zerar: `Disponivel = false` automático no produto/variação, roda pra loja de qualquer plano
  (o campo existe geral, só a tela de gestão é que é gateada Pro/Scale). Ao repor acima de zero
  (via Scale): `Disponivel = true` automático.
- Alerta é **WhatsApp**, não e-mail — `notificarAlertaEstoque`, mensagem diferente pra "estoque
  baixo" (abaixo do limiar) vs "esgotado" (zerou e pausou sozinho).
- `MovimentacaoEstoque` — tabela de auditoria já migrada, registra cada venda e cada
  reposição/ajuste manual, com estoque resultante e referência ao pedido quando aplicável. A
  "contagem física / auditoria formal" que eu tinha desenhado como sub-fase nova **já está
  coberta** por isso — não precisa de sub-fase própria.

**O que falta de verdade — únicos itens que são construção nova**

**9.1 — Ficha técnica + CMV automático, plano Scale — implementada e testada em 18/08/2026**

O que foi construído (backend):
- `domain.Insumo` (tabela `insumos`, nova): `UnidadeCompra`/`UnidadeUso` livres (texto — "kg"/"g",
  "fardo"/"unidade" etc., sem lista fechada), `FatorConversao` (quantas `UnidadeUso` equivalem a 1
  `UnidadeCompra`) e `CustoUnidadeCompra`. `CustoPorUnidadeUso()` é sempre **derivado na hora**
  (`CustoUnidadeCompra / FatorConversao`), nunca guardado — é assim que o CMV fica automático de
  verdade: não existe um valor cacheado de custo pra invalidar/recalcular quando o preço do insumo
  muda, o cálculo já lê o valor atual sempre. Estoque próprio do insumo (`EstoqueAtual`/
  `EstoqueAlerta`, `*float64` — insumo se mede em fração, diferente do estoque de produto que é
  `*int`), opcional, mesmo espírito nil=sem controle de `domain.Produto`.
- `domain.FichaTecnicaItem` (tabela `ficha_tecnica_itens`, nova): insumo + quantidade por produto.
  **Escopo v1, decisão consciente**: a ficha técnica é do **produto**, não da variação — todo
  pedido desse produto consome a mesma receita, independente da variação escolhida. Ficha técnica
  por variação não foi pedida agora, fica pra depois se precisar.
- `domain.MovimentacaoInsumo` (tabela `movimentacoes_insumo`, nova, própria — não reaproveita
  `MovimentacaoEstoque` porque lá `ProdutoID` é `NOT NULL`): mesmo padrão de auditoria da Fase 8
  (venda/reposição/ajuste, reaproveitando o enum `TipoMovimentoEstoque` já existente).
- `InsumoService`/`FichaTecnicaService` (+ handlers/rotas, `GET/POST/PUT/DELETE /admin/insumos`,
  `GET/PUT /admin/produtos/:id/ficha-tecnica`) — CRUD com checagem de dono, mesmo padrão de
  `SubcategoriaService`/`ComboService`. Insumo usado em alguma ficha técnica não pode ser excluído
  (mensagem amigável, mesmo padrão de `ComboRepository.ExisteComComponente`). Salvar a ficha
  técnica substitui a lista inteira (apaga+recria numa transação), mesmo padrão de
  `ComboRepository.Atualizar`.
- **Integração no momento da venda** (`PosPagamentoService.descontarInsumosSeFichaTecnica`, novo,
  chamado antes de `descontarEstoque` tanto pra item avulso quanto pra componente de combo): se o
  produto tem ficha técnica, desconta cada insumo (quantidade da receita × quantidade vendida) —
  **atômico via SQL** (`GREATEST`, mesmo padrão de `ProdutoRepository.SubtrairEstoque`, evita race
  condition entre pagamentos concorrentes) — e **não** cai mais no desconto de estoque simples do
  produto/variação (os dois caminhos são mutuamente exclusivos por produto). Alerta de estoque
  baixo/esgotado por WhatsApp, mesmo padrão do alerta de produto (`notificarAlertaInsumo`, mensagem
  própria porque "produto" não faria sentido no texto pra um insumo).
- `GET /admin/produtos/:id/ficha-tecnica` devolve `{itens, cmv, preco, margem}` — CMV somado a
  partir do custo por unidade de uso de cada insumo, margem = preço − CMV.

Frontend:
- `pages/admin/Insumos.tsx` (rota `/admin/insumos`, nova) — CRUD de insumo, mesmo padrão visual de
  `Cupons.tsx`. Bloqueado com upsell fora do Scale (mesmo padrão de `Estoque.tsx`).
- `components/admin/FichaTecnicaModal.tsx` (novo) — aberto por um botão "Ficha técnica" no card do
  produto em `Produtos.tsx` (só aparece pra loja Scale). Lista insumo+quantidade editável, CMV/
  margem **recalculados no cliente a cada tecla** (mesma fórmula do backend, sem round-trip) e
  reconciliados com a resposta real do servidor só ao salvar.
- Link "Insumos" no menu (`Dashboard.tsx`), só pra loja Scale, logo depois de "Estoque".

**Ressalvas de escopo, decididas conscientemente, não esquecidas:**
1. Insumo achando estoque insuficiente **não bloqueia o checkout nem pausa o produto
   automaticamente** — o consumo é só descontado (podendo ficar negativo pro GREATEST truncar em
   zero) depois do pagamento confirmado, igual produto simples. Reservar estoque de insumo no
   carrinho, ou pausar produto quando falta insumo, não foi pedido nessa fase — ficaria pra uma
   fase própria se o William quiser.
2. CMV só aparece dentro do modal de ficha técnica — não foi adicionado à listagem de produtos nem
   a nenhum relatório novo (isso seria mais próximo de "9.3 — relatórios avançados").

**Testado ao vivo (loja de teste temporária, promovida pra Scale direto no banco, removida depois
do teste):**
- CRUD de insumo via navegador: criar "Carne bovina moída" (kg → g, fator 1000, R$32/kg) — custo
  por grama calculado e exibido corretamente (R$0,03/g) tanto no formulário quanto na listagem.
- Ficha técnica de um produto ("Burguer Bacon", R$28): adicionar 100g de carne — CMV ao vivo no
  modal bateu R$3,20, margem R$24,80, idêntico ao que o backend devolveu no `PUT`. Fechar e reabrir
  o modal confirma que persiste certinho.
- **Desconto automático na venda**, testado chamando `PosPagamentoService.ProcessarPedidoPago` de
  ponta a ponta (o caminho real de pós-pagamento, não uma simulação isolada) num pedido de 2x
  Burguer Bacon: insumo com 500g em estoque foi pra 300g (200g = 100g × 2, confirmado pelo SQL
  `GREATEST(estoque_atual - 200, 0)` no log), e a movimentação ficou registrada
  (`tipo=venda quantidade=-200 estoque_resultante=300 pedido_id=<o pedido de teste>`). Confirma que
  o produto com ficha técnica realmente pulou o desconto de estoque simples (nenhuma query ao
  estoque de `produtos` pra esse item) e foi só pelo caminho de insumo.
- Não foi possível testar o alerta de WhatsApp de insumo baixo/esgotado nem o auto-pause (não
  existe pra insumo, ver ressalva 1 acima) contra um número real — o mecanismo é idêntico ao já
  comprovado em produção pra alerta de produto (Fase 8), risco baixo.

Validado com `go build ./...`, `go vet ./...`, `gofmt -l` (limpo nos arquivos tocados) e
`npx tsc -b`/`npm run build`, todos limpos.

**9.2 — Importação em massa, plano Scale — via XML implementada e testada em 18/08/2026; via PDF+IA
deixada de fora conscientemente**

**Decisão de escopo, confirmada com o William antes de implementar**: essa fase tinha duas vias. A
via PDF+IA exigiria escolher um provedor de IA e configurar credenciais novas (nada disso existe
hoje — o projeto não tem NENHUMA integração de IA/LLM real; "Sugestão Inteligente" é só um nome
comercial pra um recurso configurado manualmente, não chama nenhum modelo) e teria custo recorrente
por PDF processado — mesmo tipo de decisão que travou a integração do Mercado Pago até ter
credenciais de verdade. Implementada só a via XML (autocontida, sem dependência externa); a via
PDF+IA fica pendente até o William decidir provedor/orçamento.

O que foi construído:
- `NFeImportService` (`nfe_import_service.go`, novo) — parseia o XML da NF-e via `encoding/xml`
  (structs próprias, só os campos usados: `nNF`, `xNome` do emitente, e por item `xProd`/`uCom`/
  `qCom`/`vUnCom`). Aceita tanto `<nfeProc>` (nota + protocolo, formato mais comum ao baixar do
  portal do fornecedor) quanto `<NFe>` solto — tenta o primeiro, cai pro segundo.
- **Sem upload multipart**: o XML é texto, então o navegador lê o arquivo com `File.text()` e manda
  o conteúdo como string dentro de um `POST` JSON normal — esse backend não tem nenhum mecanismo de
  upload multipart em lugar nenhum (fotos vão direto do navegador pro Cloudinary, nunca passam pelo
  backend), não fazia sentido introduzir um só pra isso.
- **Fluxo em duas etapas, com tela de conferência sempre** (decisão: apliquei a mesma exigência de
  "sempre conferir antes de confirmar" do texto original (que falava só da via PDF) também pro XML —
  mesmo sendo dado estruturado, a interpretação de "isso é um insumo novo ou já existe?" e "qual a
  unidade de uso na receita?" não dá pra saber só pelo XML, tem que ter humano decidindo):
  1. `POST /admin/insumos/importar-nfe/preview` — só leitura, nenhuma escrita no banco. Devolve
     número da nota, fornecedor, e cada item com uma sugestão de insumo já cadastrado (match exato
     por nome, sem diferenciar maiúsculo/minúsculo) ou `null` se não achou.
  2. `POST /admin/insumos/importar-nfe/confirmar` — aplica o que o admin decidiu linha a linha:
     `vincular` (soma estoque + atualiza custo de um insumo existente), `criar` (cria um insumo
     novo, exige que o admin informe a unidade de uso na receita e o fator de conversão — o XML só
     diz a unidade de *compra*, não tem como inferir isso), ou `ignorar`.
- **Bug achado e corrigido durante o teste ao vivo**: a quantidade da NF-e vem na unidade de
  *compra* (`qCom`, ex: 5 KG), mas `Insumo.EstoqueAtual` é sempre guardado na unidade de *uso* (ex:
  gramas) — a primeira versão somava a quantidade crua sem converter pelo fator, fazendo "comprei
  5kg" virar "5g" no estoque (1000x errado). Corrigido multiplicando pelo fator de conversão nos
  dois caminhos (`criar` e `vincular`) antes de gravar — confirmado depois com um teste real: 5kg
  importados de um insumo com fator 1000 e 10g já em estoque resultaram em 5010g, não 15g.
- Cada confirmação registra `MovimentacaoInsumo` (tipo `reposicao`, motivo "Importação NF-e nº X"),
  reaproveitando a mesma tabela de auditoria da Fase 9.1.

Frontend: `components/admin/ImportarNFeModal.tsx` (novo) — botão "Importar NF-e" em
`Insumos.tsx`, seletor de arquivo `.xml`, tela de conferência com uma linha por item do XML
(nome, quantidade/unidade/custo editáveis, seletor vincular/criar/ignorar, campos extras de
unidade de uso + fator quando "criar").

**Testado ao vivo** (loja de teste temporária, mesmo padrão da Fase 9.1) com um XML de NF-e de
exemplo (2 itens: um batendo com um insumo já cadastrado por nome em caixa diferente — confirma
o match sem diferenciar maiúsculo/minúsculo —, outro sem match nenhum): prévia mostrou os dois
itens corretamente classificados (`vincular`/`criar` sugeridos certos), ajuste manual de unidade
de uso funcionou, e a confirmação gravou os valores certos — só depois de achar e corrigir o bug
de conversão de unidade acima (a primeira tentativa gravou valores 1000x menores, seria um erro
sério em produção se não fosse pego no teste).

Validado com `go build ./...`, `go vet ./...`, `gofmt -l` (limpo nos arquivos tocados) e
`npx tsc -b`/`npm run build`, todos limpos.

**9.3 — Lista de compras automática + relatórios avançados, plano Scale — implementada e testada em
18/08/2026**

**Confirmado antes de implementar**: nenhum dos 4 relatórios pedidos já existia. O único relatório
de estoque/venda que já existia era `DashboardService.BuscarDados` (`TopProdutos`, top 5 mais
vendidos em 30 dias) — adjacente, mas não é nenhum dos 4 pedidos aqui, não foi reaproveitado como
"já pronto" por engano.

O que foi construído — `EstoqueAvancadoService` (`estoque_avancado_service.go`, novo), 2 rotas
novas (`GET /admin/estoque/lista-compras`, `GET /admin/estoque/relatorios`), mesmo gate de plano
Scale de reposição/ajuste/histórico (Fase 8, nível 2):

- **Lista de compras** (`ListaDeCompras`): insumos com `EstoqueAtual <= EstoqueAlerta` (só entra
  quem tem os dois campos configurados — sem `EstoqueAlerta` não tem como saber o que é "pouco"
  pra aquele insumo específico). Sugere comprar até o alerta (não inventei uma margem de segurança
  maior — não foi pedida), convertido também pra unidade de compra (é nela que o lojista compra de
  verdade). Cruza com a ficha técnica (Fase 9.1) pra listar quais produtos dependem de cada insumo
  em falta — o "cruza ficha técnica" do texto original.
- **Produtos parados**: produtos disponíveis sem nenhuma venda paga nos últimos 30 dias.
- **Giro de estoque**: só itens com controle de estoque ativo (mesmo universo do relatório simples
  da Fase 8) — quantidade vendida em 30 dias ÷ estoque atual.
- **Valor parado em estoque**: soma só o estoque de **insumo** ao custo cadastrado — decisão
  documentada explicitamente na resposta da API (campo `valor_parado_observacao`), não escondida:
  `Produto` não tem campo de custo (só preço de venda), e só teria custo conhecido através da ficha
  técnica dos produtos que o usam, o que não é o mesmo conceito de "quanto vale o estoque parado
  desse produto pronto" — não inventei um custo estimado pra produto só pra preencher o número.
- **Insumos que mais saem**: soma o consumo (`MovimentacaoInsumo` tipo `venda`) dos últimos 30 dias
  por insumo, top 10.

**Dois bugs achados e corrigidos durante o teste ao vivo** (nenhum dos dois aparecia no `go build`/
`go vet` — só testando com dado real):
1. **Giro sempre saía 0, mesmo com venda real no período.** A query de "vendido nos últimos 30
   dias" passava `variacaoID` (`*uint`, `nil` pra produto sem variação) direto como parâmetro `?`
   de um `Raw()` só, esperando que o driver tratasse `nil` como `NULL` numa comparação `variacao_id
   = ?`. Não tratou de forma confiável — testado ao vivo, sempre devolvia 0. Corrigido separando em
   duas queries (uma pra `variacao_id IS NULL`, outra pra `variacao_id = valor`), decidido em Go em
   vez de deixar pro SQL decidir com bind param nulo.
2. **`insumos_que_mais_saem` sempre saía com `consumido_30d: 0`**, mesmo com a query certa e o dado
   certo no banco. Causa: o `GORM` mapeia coluna de `Raw().Scan()` pro campo da struct convertendo o
   nome do campo Go pra snake_case sozinho quando não tem tag — pra um campo chamado `Consumido30d`,
   a conversão automática não bate com o alias `consumido_30d` da query (a fronteira letra→dígito
   confunde o conversor). Corrigido com `gorm:"column:consumido_30d"` explícito no campo.

Frontend: duas seções novas em `Estoque.tsx` (Lista de compras, Relatórios avançados), visíveis só
Scale, mesmo padrão visual do resto da página — sem tela nova, a Fase 8 já tinha estabelecido
"Estoque" como o hub certo pra isso.

**Testado ao vivo** (loja de teste temporária, removida depois — mesmo padrão das fases 9.1/9.2):
cenário com 2 produtos (um vendido 3x via `ProcessarPedidoPago` real, outro sem nenhuma venda) e 1
insumo abaixo do alerta ligado por ficha técnica ao produto vendido. Confirmado, batendo exatamente
com o esperado: lista de compras mostrou o insumo com o déficit certo e "Usado em: Produto
Vendido"; giro do produto vendido saiu `0.15` (3 vendidos ÷ 20 em estoque); produtos parados listou
só o produto sem venda; valor parado em estoque bateu R$0,40 (20g restantes × R$0,02/g); insumos
que mais saem mostrou os 30g consumidos. Confirmado visualmente também no navegador (screenshot),
todos os números batendo com a API.

Validado com `go build ./...`, `go vet ./...`, `gofmt -l` (limpo nos arquivos tocados) e
`npx tsc -b`/`npm run build`, todos limpos.

**9.4 — Multi-loja consolidado, plano Scale — adiada em 18/08/2026, precisa de spec própria**

**Achado antes de começar, mudou o entendimento da fase**: o texto original supõe que "a gestão de
rede que o Scale já tem (Fase 7)" existe de verdade — não existe. Conferido no código:
`domain.Loja.UsuarioID` tem `unique` no banco (`backend/internal/domain/loja.go:28`) — cada login
só pode ter **uma** loja, sempre, sem exceção hoje. "Multi-loja / gestão de rede consolidada" nunca
saiu da matriz de recursos do Scale (só texto de marketing, `docs/drenux-planos-comissoes-definido.md`
§5) — não existe nenhum código, nem parcial, de um usuário dono de mais de uma loja.

Isso muda o tamanho real da 9.4: não é "um relatório consolidado em cima de algo que já existe"
(como 9.1–9.3 foram), é uma mudança estrutural no produto inteiro — precisa decidir como um login
passa a enxergar várias lojas (loja "matriz" com filiais? grupo solto de lojas independentes?),
como cobrança/plano funciona nesse caso (uma assinatura Scale cobre a rede toda, ou cada loja paga
a própria?), e revisar todo handler do sistema, que hoje assume `loja_id` único por token
(`middleware.AuthRequired` seta um `loja_id` só na claim do JWT). Decisão do William: **adiar** —
fica pendente de especificação própria antes de qualquer código, mesmo padrão já usado com a via
PDF+IA da Fase 9.2 (não travar o resto do roadmap nisso).

## Backlog mais antigo, fora de escopo por enquanto (não iniciar sem o William pedir)

Esses itens já existiam antes do roadmap atual e não fazem parte da sequência das 4 fases — só
ficam registrados aqui pra não sumirem do radar:
- **Carteira Drenux** (cashback cross-loja, 1% opt-in por loja, saldo global por telefone) — já
  desenhada em detalhe em outra conversa, zero código ainda.
- **Ciclo de vida de assinatura mais robusto** — cartão recusado na renovação ainda não é tratado.
- **Resto do admin migrando pra shadcn** — só "Meu Plano" e a página pública de Planos usam shadcn
  até agora; o restante do admin ainda está no estilo antigo.

## Decisões já tomadas (não reabrir sem o William pedir)

- `SegmentoPrincipal` reaproveita os valores de `TipoProduto` (`alimenticio` / `mercadoria`) — não usar
  outros nomes como `nao_pereciveis`.
- Cada fase é implementada e validada isoladamente, não tudo de uma vez.
- shadcn é a base de tudo; Magic UI só onde faz sentido destacar algo pro cliente final ou no
  dashboard; React Bits fica de reserva, não como padrão.
- **Fase 7**: limite do Start é 30 pedidos **por mês, recorrente** (renova todo ciclo, igual ao
  concorrente Goomer) — não é um limite vitalício/total. Já foi discutido e revertido de vitalício
  pra mensal antes de fechar a fase; não reabrir essa discussão sem o William pedir.
- **Fase 7**: a fórmula real de afiliado em produção antes dessa fase é 37,6% da taxa de plataforma
  (comissão bruta), não "30% do lucro líquido pro Start" — essa segunda fórmula nunca existiu no
  código, é um engano de contexto de conversa anterior. Não usar como referência do "antes".
