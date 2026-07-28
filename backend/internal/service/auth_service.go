package service

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/WilliamBreno/cardapio-backend/internal/domain"
	"github.com/WilliamBreno/cardapio-backend/internal/notification"
	"github.com/WilliamBreno/cardapio-backend/internal/repository"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type AuthService struct {
	db          *gorm.DB
	jwtSecret   string
	emailSender *notification.EmailSender
	frontendURL string
}

func NewAuthService(db *gorm.DB, jwtSecret string, emailSender *notification.EmailSender, frontendURL string) *AuthService {
	return &AuthService{db: db, jwtSecret: jwtSecret, emailSender: emailSender, frontendURL: frontendURL}
}

type CadastroInput struct {
	Nome              string
	Email             string
	Senha             string
	NomeLoja          string
	SegmentoPrincipal string // "alimenticio" | "mercadoria" — define tipo padrão de produto e categorias iniciais
	CodigoAfiliado    string // opcional — vem do ?ref=CODIGO capturado no frontend
	TokenAssinatura   string // opcional — vem de /cadastro/finalizar?token=XXX, só existe pra planos Pro/Scale
}

// Cadastrar cria o usuário, a loja dele e as categorias padrão tudo numa
// transação. Se vier um TokenAssinatura válido (pagamento de mensalidade
// já confirmado), a loja já nasce vinculada ao plano Pro/Scale e à
// assinatura Stripe correspondente; sem token, nasce no plano Start.
func (s *AuthService) Cadastrar(input CadastroInput) (string, error) {
	senhaHash, err := bcrypt.GenerateFromPassword([]byte(input.Senha), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("gerando hash da senha: %w", err)
	}

	usuario := domain.Usuario{
		Nome:      input.Nome,
		Email:     input.Email,
		SenhaHash: string(senhaHash),
	}
	var loja domain.Loja

	err = s.db.Transaction(func(tx *gorm.DB) error {
		usuarioRepo := repository.NewUsuarioRepository(tx)
		lojaRepo := repository.NewLojaRepository(tx)
		categoriaRepo := repository.NewCategoriaRepository(tx)
		afiliadoRepo := repository.NewAfiliadoRepository(tx)
		assinaturaRepo := repository.NewAssinaturaPendenteRepository(tx)

		if err := usuarioRepo.Criar(&usuario); err != nil {
			return fmt.Errorf("não foi possível criar o usuário (email já cadastrado?): %w", err)
		}

		slug, err := gerarSlugUnico(lojaRepo, input.NomeLoja)
		if err != nil {
			return fmt.Errorf("gerando slug da loja: %w", err)
		}

		segmento := domain.TipoProduto(input.SegmentoPrincipal)
		if segmento != domain.TipoProdutoMercadoria {
			segmento = domain.TipoProdutoAlimenticio
		}

		loja = domain.Loja{
			UsuarioID:         usuario.ID,
			Nome:              input.NomeLoja,
			Slug:              slug,
			Plano:             "start",
			SegmentoPrincipal: segmento,
		}

		// Se veio um token de assinatura (pagamento de Pro/Scale já
		// confirmado), valida de novo aqui dentro da transação — evita
		// condição de corrida se o cliente abrir o link em duas abas ao
		// mesmo tempo — e vincula o plano pago à loja recém-criada.
		if input.TokenAssinatura != "" {
			assinatura, err := assinaturaRepo.BuscarPorToken(input.TokenAssinatura)
			if err != nil {
				return errors.New("link de finalização inválido ou já utilizado")
			}
			loja.Plano = assinatura.Plano
			loja.StripeCustomerID = assinatura.StripeCustomerID
			loja.StripeSubscriptionID = assinatura.StripeSubscriptionID
			loja.MercadoPagoPreapprovalIDPlano = assinatura.MercadoPagoPreapprovalID

			if err := assinaturaRepo.MarcarUsado(assinatura.ID); err != nil {
				return fmt.Errorf("marcando assinatura como usada: %w", err)
			}
		}

		// Se veio um código de afiliado, resolve e vincula. Um código
		// inválido/inexistente não deve travar o cadastro — só ignora
		// silenciosamente e loga.
		if input.CodigoAfiliado != "" {
			afiliado, err := afiliadoRepo.BuscarPorCodigo(input.CodigoAfiliado)
			if err != nil {
				log.Printf("aviso: código de afiliado %q não encontrado no cadastro da loja %q", input.CodigoAfiliado, input.NomeLoja)
			} else {
				loja.AfiliadoID = &afiliado.ID
			}
		}

		if err := lojaRepo.Criar(&loja); err != nil {
			return fmt.Errorf("criando loja: %w", err)
		}

		if err := categoriaRepo.CriarVarias(categoriasPadrao(segmento, loja.ID)); err != nil {
			return fmt.Errorf("criando categorias padrão: %w", err)
		}

		return nil
	})
	if err != nil {
		return "", err
	}

	return s.gerarToken(usuario.ID, loja.ID)
}

func (s *AuthService) Login(email, senha string) (string, error) {
	usuarioRepo := repository.NewUsuarioRepository(s.db)
	lojaRepo := repository.NewLojaRepository(s.db)

	usuario, err := usuarioRepo.BuscarPorEmail(email)
	if err != nil {
		return "", errors.New("email ou senha inválidos")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(usuario.SenhaHash), []byte(senha)); err != nil {
		return "", errors.New("email ou senha inválidos")
	}

	loja, err := lojaRepo.BuscarPorUsuarioID(usuario.ID)
	if err != nil {
		return "", errors.New("loja não encontrada para esse usuário")
	}

	return s.gerarToken(usuario.ID, loja.ID)
}

func (s *AuthService) gerarToken(usuarioID, lojaID uint) (string, error) {
	claims := jwt.MapClaims{
		"usuario_id": usuarioID,
		"loja_id":    lojaID,
		"exp":        time.Now().Add(7 * 24 * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.jwtSecret))
}

// categoriasPadrao devolve as categorias iniciais de uma loja recém-criada,
// variando conforme o segmento — lojas de mercadoria tendem a não usar as
// categorias "Salgados"/"Doces", que fazem sentido só pro alimentício.
func categoriasPadrao(segmento domain.TipoProduto, lojaID uint) []domain.Categoria {
	if segmento == domain.TipoProdutoMercadoria {
		return []domain.Categoria{
			{LojaID: lojaID, Nome: "Mais vendidos"},
			{LojaID: lojaID, Nome: "Novidades"},
		}
	}
	return []domain.Categoria{
		{LojaID: lojaID, Nome: "Salgados"},
		{LojaID: lojaID, Nome: "Doces"},
	}
}

func gerarSlugUnico(lojaRepo *repository.LojaRepository, nomeLoja string) (string, error) {
	base := gerarSlug(nomeLoja)
	slug := base
	contador := 1

	for {
		existe, err := lojaRepo.SlugExiste(slug)
		if err != nil {
			return "", err
		}
		if !existe {
			return slug, nil
		}
		contador++
		slug = fmt.Sprintf("%s-%d", base, contador)
	}
}

func (s *AuthService) EsqueciSenha(email string) error {
	usuarioRepo := repository.NewUsuarioRepository(s.db)

	usuario, err := usuarioRepo.BuscarPorEmail(email)
	if err != nil {
		return nil
	}

	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return fmt.Errorf("gerando token de reset: %w", err)
	}
	token := hex.EncodeToString(tokenBytes)
	expira := time.Now().Add(1 * time.Hour)

	if err := usuarioRepo.SalvarResetToken(usuario.ID, token, expira); err != nil {
		return fmt.Errorf("salvando token de reset: %w", err)
	}

	if s.emailSender == nil {
		return nil
	}

	link := fmt.Sprintf("%s/redefinir-senha?token=%s", s.frontendURL, token)
	if err := s.emailSender.EnviarResetSenha(usuario.Email, usuario.Nome, link); err != nil {
		return fmt.Errorf("enviando email de reset: %w", err)
	}

	return nil
}

func (s *AuthService) RedefinirSenha(token, novaSenha string) error {
	usuarioRepo := repository.NewUsuarioRepository(s.db)

	usuario, err := usuarioRepo.BuscarPorResetToken(token)
	if err != nil {
		return errors.New("link inválido ou expirado")
	}

	if usuario.ResetTokenExpira == nil || time.Now().After(*usuario.ResetTokenExpira) {
		return errors.New("link inválido ou expirado")
	}

	senhaHash, err := bcrypt.GenerateFromPassword([]byte(novaSenha), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("gerando hash da nova senha: %w", err)
	}

	if err := usuarioRepo.AtualizarSenha(usuario.ID, string(senhaHash)); err != nil {
		return fmt.Errorf("atualizando senha: %w", err)
	}

	return nil
}
