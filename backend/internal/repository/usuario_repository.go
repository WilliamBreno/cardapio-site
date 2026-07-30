package repository

import (
	"time"

	"github.com/WilliamBreno/cardapio-backend/internal/domain"
	"gorm.io/gorm"
)

type UsuarioRepository struct {
	db *gorm.DB
}

// NewUsuarioRepository recebe um *gorm.DB — pode ser a conexão principal
// ou uma transação (tx). Como uma transação do GORM também é um *gorm.DB,
// esse mesmo repositório funciona dos dois jeitos sem precisar duplicar
// código.
func NewUsuarioRepository(db *gorm.DB) *UsuarioRepository {
	return &UsuarioRepository{db: db}
}

func (r *UsuarioRepository) Criar(usuario *domain.Usuario) error {
	return r.db.Create(usuario).Error
}

func (r *UsuarioRepository) BuscarPorID(id uint) (*domain.Usuario, error) {
	var usuario domain.Usuario
	if err := r.db.First(&usuario, id).Error; err != nil {
		return nil, err
	}
	return &usuario, nil
}

func (r *UsuarioRepository) BuscarPorEmail(email string) (*domain.Usuario, error) {
	var usuario domain.Usuario
	if err := r.db.Where("email = ?", email).First(&usuario).Error; err != nil {
		return nil, err
	}
	return &usuario, nil
}

// SalvarResetToken grava o token de redefinição de senha e sua data de
// expiração no usuário. Chamado quando alguém pede "esqueci minha senha".
func (r *UsuarioRepository) SalvarResetToken(usuarioID uint, token string, expira time.Time) error {
	return r.db.Model(&domain.Usuario{}).
		Where("id = ?", usuarioID).
		Updates(map[string]interface{}{
			"reset_token":        token,
			"reset_token_expira": expira,
		}).Error
}

// BuscarPorResetToken encontra o usuário dono de um token de reset
// específico. Não confere expiração aqui — isso é responsabilidade do
// service, que decide o que fazer com um token vencido.
func (r *UsuarioRepository) BuscarPorResetToken(token string) (*domain.Usuario, error) {
	var usuario domain.Usuario
	if err := r.db.Where("reset_token = ?", token).First(&usuario).Error; err != nil {
		return nil, err
	}
	return &usuario, nil
}

// AtualizarSenha troca o hash da senha e limpa o token de reset (uso
// único — depois de trocar a senha, esse link não deve funcionar de novo).
func (r *UsuarioRepository) AtualizarSenha(usuarioID uint, novaSenhaHash string) error {
	return r.db.Model(&domain.Usuario{}).
		Where("id = ?", usuarioID).
		Updates(map[string]interface{}{
			"senha_hash":         novaSenhaHash,
			"reset_token":        nil,
			"reset_token_expira": nil,
		}).Error
}
