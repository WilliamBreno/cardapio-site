 package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// AfiliadoRequired é o equivalente de AuthRequired, mas valida o token
// de afiliado (claim "afiliado_id"), separado do token de dono de loja.
// Fica num arquivo próprio pra não mexer no middleware existente.
func AfiliadoRequired(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" || !strings.HasPrefix(header, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"erro": "token não informado"})
			return
		}
		tokenStr := strings.TrimPrefix(header, "Bearer ")

		token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
			return []byte(jwtSecret), nil
		})
		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"erro": "token inválido"})
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"erro": "token inválido"})
			return
		}

		afiliadoIDFloat, ok := claims["afiliado_id"].(float64)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"erro": "esse token não é de um afiliado"})
			return
		}

		c.Set("afiliado_id", uint(afiliadoIDFloat))
		c.Next()
	}
}