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
(produtos não perecíveis, ex: roupas, artesanato). Monetização por assinatura (planos Start/Pro/Scale,
ver `MeuPlano.tsx`) combinada com a taxa por pedido, que varia por plano.

**Processador de pagamento em migração: Stripe Connect → Mercado Pago (decisão final, fechada em
23/07/2026, depois de avaliar Mercado Pago contra Asaas).** Ainda nenhuma empresa real está
cadastrada em produção, então a troca está sendo feita antes do lançamento real — motivo principal é
o Pix (na Stripe, hoje só disponível por convite pra empresas brasileiras), e no Mercado Pago
especificamente: sem teto de quantidade de contas vinculadas (diferente da Asaas, que tem limite de
10 subcontas até homologação regulatória) e Pix cobrado em percentual (0,99%), o que favorece o
ticket baixo típico do segmento alimentício. Fórmula de comissão mantida como já decidido: Start
`max(pedido × 6,5%, R$2,50)` com a Drenux absorvendo a taxa do processador; Pro (R$129/mês+4%) e
Scale (R$349/mês+1,5%) mantêm os mesmos números, com a loja absorvendo a taxa do processador — só
troca o processador por trás, a lógica de quem paga a taxa em cada plano não muda. Ver a
especificação completa da migração na Fase 5, mais abaixo.

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
