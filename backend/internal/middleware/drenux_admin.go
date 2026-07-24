package middleware

import (
	"crypto/subtle"
	"net/http"

	"github.com/gin-gonic/gin"
)

// DrenuxAdminRequired protege as rotas internas /drenux/* (hoje só o
// controle de repasse de comissão de afiliado, Fase 5.5) — não existe
// sistema de login de staff da Drenux ainda, então é só um secret
// compartilhado no header X-Drenux-Admin-Secret, mesmo espírito do
// X-Cron-Secret já usado em /relatorio/semanal.
//
// Diferente do CronSecret (que deixa a rota aberta se a env var não
// estiver definida — aceitável pra um cron interno), aqui o secret vazio
// SEMPRE bloqueia: essas rotas expõem dado financeiro de todos os
// afiliados da plataforma, então "sem secret configurado" tem que
// significar "rota fechada", nunca "rota aberta por engano".
func DrenuxAdminRequired(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if secret == "" {
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"erro": "DRENUX_ADMIN_SECRET não configurada"})
			return
		}

		recebido := c.GetHeader("X-Drenux-Admin-Secret")
		if subtle.ConstantTimeCompare([]byte(recebido), []byte(secret)) != 1 {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"erro": "não autorizado"})
			return
		}

		c.Next()
	}
}
