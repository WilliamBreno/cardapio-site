package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/WilliamBreno/cardapio-backend/internal/domain"
	"github.com/WilliamBreno/cardapio-backend/internal/notification"
	"github.com/WilliamBreno/cardapio-backend/internal/repository"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stripe/stripe-go/v86"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type AfiliadoService struct {
	db           *gorm.DB
	jwtSecret    string
	afiliadoRepo *repository.AfiliadoRepository
	stripeClient *stripe.Client
	emailSender  *notification.EmailSender
	frontendURL  string
}

func NewAfiliadoService(db *gorm.DB, jwtSecret, stripeSecretKey string, emailSender *notification.EmailSender, frontendURL string) *AfiliadoService {
	return &AfiliadoService{
		db:           db,
		jwtSecret:    jwtSecret,
		afiliadoRepo: repository.NewAfiliadoRepository(db),
		stripeClient: stripe.NewClient(stripeSecretKey),
		emailSender:  emailSender,
		frontendURL:  frontendURL,
	}
}

// CriarAfiliado cadastra uma conta de afiliado — não existe autocadastro
// (ver domain.Afiliado), então isso só é chamado pela tela interna
// /drenux/afiliados (secret-gated, ver middleware.DrenuxAdminRequired).
// O código de indicação (usado em ?ref=CODIGO) é gerado a partir do nome,
// reaproveitando o mesmo gerador de slug já usado pra loja.
//
// comissaoPercentual é a fração da taxa de plataforma que ESSE afiliado
// recebe (ex: 0.376 = 37,6%) — definida aqui no cadastro, negociada por
// afiliado (ver domain.Afiliado.ComissaoPercentual, Fase 5.5).
func (s *AfiliadoService) CriarAfiliado(nome, email, senha string, comissaoPercentual float64) (*domain.Afiliado, error) {
	if comissaoPercentual <= 0 || comissaoPercentual > 1 {
		return nil, errors.New("comissão precisa ser maior que 0 e no máximo 1 (100% da taxa de plataforma)")
	}

	senhaHash, err := bcrypt.GenerateFromPassword([]byte(senha), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("gerando hash da senha: %w", err)
	}

	codigo, err := s.gerarCodigoUnico(nome)
	if err != nil {
		return nil, fmt.Errorf("gerando código de indicação: %w", err)
	}

	afiliado := domain.Afiliado{
		Nome:               nome,
		Email:              email,
		SenhaHash:          string(senhaHash),
		Codigo:             codigo,
		ComissaoPercentual: comissaoPercentual,
	}
	if err := s.afiliadoRepo.Criar(&afiliado); err != nil {
		return nil, fmt.Errorf("não foi possível criar o afiliado (email já cadastrado?): %w", err)
	}
	return &afiliado, nil
}

// gerarCodigoUnico monta o código de indicação a partir do nome (mesmo
// normalizador usado pro slug de loja — minúsculo, sem acento, só
// [a-z0-9-]) e desambigua com um sufixo numérico se já existir.
func (s *AfiliadoService) gerarCodigoUnico(nome string) (string, error) {
	base := gerarSlug(nome)
	if base == "" {
		base = "afiliado"
	}
	codigo := base
	contador := 1
	for {
		_, err := s.afiliadoRepo.BuscarPorCodigo(codigo)
		if err != nil {
			return codigo, nil // não encontrado — código livre
		}
		contador++
		codigo = fmt.Sprintf("%s-%d", base, contador)
	}
}

// Login confere email/senha do afiliado e devolve um token JWT próprio,
// separado do token de dono de loja (claim "afiliado_id" em vez de
// "usuario_id"/"loja_id").
func (s *AfiliadoService) Login(email, senha string) (string, error) {
	afiliado, err := s.afiliadoRepo.BuscarPorEmail(email)
	if err != nil {
		return "", errors.New("email ou senha inválidos")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(afiliado.SenhaHash), []byte(senha)); err != nil {
		return "", errors.New("email ou senha inválidos")
	}
	return s.gerarToken(afiliado.ID)
}

func (s *AfiliadoService) gerarToken(afiliadoID uint) (string, error) {
	claims := jwt.MapClaims{
		"afiliado_id": afiliadoID,
		"exp":         time.Now().Add(7 * 24 * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.jwtSecret))
}

// DashboardAfiliado agrega o que o painel do afiliado precisa mostrar:
// lojas indicadas, total já ganho, e se a conta Stripe já foi conectada.
type DashboardAfiliado struct {
	Codigo          string        `json:"codigo"`
	Lojas           []domain.Loja `json:"lojas"`
	TotalGanho      float64       `json:"total_ganho"`
	StripeConectado bool          `json:"stripe_conectado"`
}

func (s *AfiliadoService) Dashboard(afiliadoID uint) (*DashboardAfiliado, error) {
	afiliado, err := s.afiliadoRepo.BuscarPorID(afiliadoID)
	if err != nil {
		return nil, errors.New("afiliado não encontrado")
	}
	lojas, err := s.afiliadoRepo.ListarLojasIndicadas(afiliadoID)
	if err != nil {
		return nil, err
	}
	total, err := s.afiliadoRepo.SomarComissoes(afiliadoID)
	if err != nil {
		return nil, err
	}
	return &DashboardAfiliado{
		Codigo:          afiliado.Codigo,
		Lojas:           lojas,
		TotalGanho:      total,
		StripeConectado: afiliado.StripeAccountID != "",
	}, nil
}

// IniciarOnboarding cria (se ainda não existir) a conta Stripe Connect
// Express do afiliado e devolve o link de onboarding.
func (s *AfiliadoService) IniciarOnboarding(ctx context.Context, afiliadoID uint, returnURL, refreshURL string) (string, error) {
	afiliado, err := s.afiliadoRepo.BuscarPorID(afiliadoID)
	if err != nil {
		return "", errors.New("afiliado não encontrado")
	}

	accountID := afiliado.StripeAccountID
	if accountID == "" {
		accountParams := &stripe.AccountCreateParams{
			Type:    stripe.String(string(stripe.AccountTypeExpress)),
			Country: stripe.String("BR"),
		}
		account, err := s.stripeClient.V1Accounts.Create(ctx, accountParams)
		if err != nil {
			return "", fmt.Errorf("criando conta Stripe do afiliado: %w", err)
		}
		accountID = account.ID
		if err := s.afiliadoRepo.AtualizarStripeAccountID(afiliadoID, accountID); err != nil {
			return "", fmt.Errorf("salvando conta Stripe do afiliado: %w", err)
		}
	}

	linkParams := &stripe.AccountLinkCreateParams{
		Account:    stripe.String(accountID),
		Type:       stripe.String("account_onboarding"),
		ReturnURL:  stripe.String(returnURL),
		RefreshURL: stripe.String(refreshURL),
	}
	link, err := s.stripeClient.V1AccountLinks.Create(ctx, linkParams)
	if err != nil {
		return "", fmt.Errorf("gerando link de onboarding do afiliado: %w", err)
	}
	return link.URL, nil
}

// EsqueciSenha gera um token de redefinição e envia por email. Nunca
// retorna erro por "email não encontrado" — sempre parece ter dado
// certo, evitando que alguém descubra quais emails de afiliado existem.
func (s *AfiliadoService) EsqueciSenha(email string) error {
	afiliado, err := s.afiliadoRepo.BuscarPorEmail(email)
	if err != nil {
		return nil
	}

	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return fmt.Errorf("gerando token de reset: %w", err)
	}
	token := hex.EncodeToString(tokenBytes)
	expira := time.Now().Add(1 * time.Hour)

	if err := s.afiliadoRepo.SalvarResetToken(afiliado.ID, token, expira); err != nil {
		return fmt.Errorf("salvando token de reset: %w", err)
	}

	if s.emailSender == nil {
		return nil
	}

	link := fmt.Sprintf("%s/afiliado/redefinir-senha?token=%s", s.frontendURL, token)
	if err := s.emailSender.EnviarResetSenha(afiliado.Email, afiliado.Nome, link); err != nil {
		return fmt.Errorf("enviando email de reset: %w", err)
	}

	return nil
}

// RedefinirSenha valida o token (existe e não expirou) e troca a senha.
func (s *AfiliadoService) RedefinirSenha(token, novaSenha string) error {
	afiliado, err := s.afiliadoRepo.BuscarPorResetToken(token)
	if err != nil {
		return errors.New("link inválido ou expirado")
	}

	if afiliado.ResetTokenExpira == nil || time.Now().After(*afiliado.ResetTokenExpira) {
		return errors.New("link inválido ou expirado")
	}

	senhaHash, err := bcrypt.GenerateFromPassword([]byte(novaSenha), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("gerando hash da nova senha: %w", err)
	}

	if err := s.afiliadoRepo.AtualizarSenha(afiliado.ID, string(senhaHash)); err != nil {
		return fmt.Errorf("atualizando senha: %w", err)
	}

	return nil
}
