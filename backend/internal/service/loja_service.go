package service

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/WilliamBreno/cardapio-backend/internal/domain"
	"github.com/WilliamBreno/cardapio-backend/internal/notification"
	"github.com/WilliamBreno/cardapio-backend/internal/repository"
	"gorm.io/gorm"
)

// contasComPermissaoTrocarSegmento lista os únicos emails que podem
// alterar o SegmentoPrincipal de uma loja depois do cadastro — pra
// qualquer outra conta, a escolha feita no cadastro é definitiva (pedido
// do William: sem isso, a pergunta do cadastro perde o sentido, já que
// dela dependem regras de negócio inteiras — Variação x Subcategoria/
// Grupo, categoria sugerida no onboarding do Mercado Pago, etc). São
// contas de teste/administração, não lojistas reais.
var contasComPermissaoTrocarSegmento = map[string]bool{
	"teste-mp-assinatura@example.com": true,
	"drenuxtest@test.com":             true,
	"lojasapato@test.com":             true,
}

type LojaService struct {
	lojaRepo           *repository.LojaRepository
	usuarioRepo        *repository.UsuarioRepository
	pedidoRepo         *repository.PedidoRepository
	notificationSender notification.NotificationSender
}

func NewLojaService(db *gorm.DB, notificationSender notification.NotificationSender) *LojaService {
	return &LojaService{
		lojaRepo:           repository.NewLojaRepository(db),
		usuarioRepo:        repository.NewUsuarioRepository(db),
		pedidoRepo:         repository.NewPedidoRepository(db),
		notificationSender: notificationSender,
	}
}

func (s *LojaService) Buscar(lojaID uint) (*domain.Loja, error) {
	return s.lojaRepo.BuscarPorID(lojaID)
}

func (s *LojaService) BuscarPorSlug(slug string) (*domain.Loja, error) {
	return s.lojaRepo.BuscarPorSlug(slug)
}

// PodeEditarSegmento diz se o usuário logado pode alterar o
// SegmentoPrincipal da própria loja depois do cadastro — ver
// contasComPermissaoTrocarSegmento.
func (s *LojaService) PodeEditarSegmento(usuarioID uint) bool {
	usuario, err := s.usuarioRepo.BuscarPorID(usuarioID)
	if err != nil {
		return false
	}
	return contasComPermissaoTrocarSegmento[usuario.Email]
}

func (s *LojaService) AtualizarConfiguracoes(lojaID, usuarioID uint, cfg repository.ConfiguracoesLoja) error {
	// Achado investigando entrega de WhatsApp (ver Tarefa 0): esse campo
	// nunca era normalizado — um número salvo com formatação (espaço,
	// hífen, sem o 55) resolve pro JID errado na hora de notificar o
	// lojista, sem erro nenhum visível pra ele. Ver NormalizarTelefone.
	cfg.WhatsappNumero = NormalizarTelefone(cfg.WhatsappNumero)

	// Sugestão Inteligente (Fase 6): o toggle liga/desliga a exibição no
	// carrinho do cliente livremente, contratada ou não — mesmo sem
	// contratar, a loja tem direito a 1 vínculo grátis que precisa
	// aparecer de verdade se o toggle estiver ligado (ver
	// SugestaoProdutoService, que já limita a QUANTIDADE de vínculos, não
	// a exibição). Não há mais trava aqui.

	// SegmentoPrincipal é definitivo desde o cadastro — só as contas em
	// contasComPermissaoTrocarSegmento podem mudar depois. Pra qualquer
	// outra conta, ignora silenciosamente o valor que veio do formulário
	// e mantém o segmento atual, sem travar o resto do salvamento (mesmo
	// espírito da trava que existia pra Sugestão Inteligente: nunca falha
	// o form inteiro por causa de um campo só). O frontend já deixa esse
	// campo travado pra quem não tem permissão — isso aqui é defesa
	// contra uma chamada direta na API.
	lojaAtual, err := s.lojaRepo.BuscarPorID(lojaID)
	if err != nil {
		return err
	}
	if cfg.SegmentoPrincipal != string(lojaAtual.SegmentoPrincipal) && !s.PodeEditarSegmento(usuarioID) {
		cfg.SegmentoPrincipal = string(lojaAtual.SegmentoPrincipal)
	}

	return s.lojaRepo.AtualizarConfiguracoes(lojaID, cfg)
}

// LimitePedidosStart devolve quantos pedidos a loja já recebeu no mês
// corrente e se ela está bloqueada pro limite do plano Start (Fase 7.3).
// Sempre calcula o contador (útil de exibir mesmo antes do limite), mas
// só considera "bloqueada" se a loja ainda estiver no Start — trocar pro
// Basic destrava na hora, mesmo sem nenhum campo ser limpo no banco (ver
// domain.Loja.PedidosBloqueadosDesde).
func (s *LojaService) LimitePedidosStart(loja *domain.Loja) (pedidosNoMes int, bloqueada bool, err error) {
	pedidosNoMes, err = s.pedidoRepo.ContarPedidosMesAtual(loja.ID)
	if err != nil {
		return 0, false, err
	}

	if loja.Plano != "start" || loja.PedidosBloqueadosDesde == nil {
		return pedidosNoMes, false, nil
	}

	fusoBrasil, _ := time.LoadLocation("America/Sao_Paulo")
	agora := time.Now().In(fusoBrasil)
	inicioMes := time.Date(agora.Year(), agora.Month(), 1, 0, 0, 0, 0, fusoBrasil)
	bloqueada = !loja.PedidosBloqueadosDesde.Before(inicioMes)

	return pedidosNoMes, bloqueada, nil
}

// VerificarLimiteStart aplica a escalada final do limite de pedidos/mês
// do plano Start (Fase 7.3): lojas avisadas do limite há pelo menos 3 dias
// e que ainda não migraram pro Basic são bloqueadas (novos pedidos passam
// a ser recusados, ver PedidoService.verificarLimiteStart) e avisadas por
// WhatsApp. O aviso INICIAL de "passou de 30" já é disparado antes disso,
// direto em PedidoService — essa rotina só cobre o passo seguinte, que
// precisa avisar o dono mesmo que nenhum cliente novo tente pedir nos 3
// dias seguintes ao aviso. Pensada pra rodar via cron externo (mesmo
// padrão de MercadoPagoService.RenovarTokensExpirando).
func (s *LojaService) VerificarLimiteStart(ctx context.Context) (bloqueadas int, erros []string) {
	lojas, err := s.lojaRepo.ListarStartParaBloquearLimite(time.Now())
	if err != nil {
		return 0, []string{fmt.Sprintf("erro listando lojas: %v", err)}
	}

	for i := range lojas {
		loja := &lojas[i]
		if err := s.lojaRepo.AtualizarBloqueioLimitePedidos(loja.ID, time.Now()); err != nil {
			erros = append(erros, fmt.Sprintf("loja %d: %v", loja.ID, err))
			log.Printf("falha ao bloquear pedidos da loja %d por limite do Start: %v", loja.ID, err)
			continue
		}
		s.avisarBloqueioLimitePedidos(ctx, loja)
		bloqueadas++
	}
	return bloqueadas, erros
}

func (s *LojaService) avisarBloqueioLimitePedidos(ctx context.Context, loja *domain.Loja) {
	if s.notificationSender == nil || loja.WhatsappNumero == "" {
		return
	}
	texto := "Sua loja atingiu o limite do plano grátis — ative o Basic agora (sem custo) e volte a receber pedidos na hora.\n\nAtiva em Meu Plano, no painel da Drenux."
	if err := s.notificationSender.EnviarTextoAdmin(ctx, loja.WhatsappNumero, texto); err != nil {
		log.Printf("falha ao enviar aviso de bloqueio de pedidos da loja %d: %v", loja.ID, err)
	}
}
