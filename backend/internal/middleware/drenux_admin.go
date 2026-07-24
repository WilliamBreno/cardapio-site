package middleware

import (
	"crypto/subtle"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	drenuxMaxTentativas = 5
	drenuxJanelaFalhas  = 15 * time.Minute
	drenuxTempoBloqueio = 15 * time.Minute
)

type tentativaDrenux struct {
	falhas        int
	primeiraFalha time.Time
	bloqueadoAte  time.Time
}

// drenuxTentativas guarda, por IP, quantas tentativas erradas seguidas
// aconteceram — em memória mesmo, não é robusto contra múltiplas
// instâncias do servidor nem sobrevive a um redeploy, mas já eleva bem
// o custo de uma tentativa de força bruta contra o secret (que sozinho,
// sem isso, podia ser tentado sem limite nenhum).
var (
	drenuxTentativasMu sync.Mutex
	drenuxTentativas   = map[string]*tentativaDrenux{}
)

// DrenuxAdminRequired protege as rotas internas /drenux/* (hoje só o
// controle de repasse de comissão de afiliado, Fase 5.5) — não existe
// sistema de login de staff da Drenux ainda, então é só um secret
// compartilhado no header X-Drenux-Admin-Secret, mesmo espírito do
// X-Cron-Secret já usado em /relatorio/semanal, mas com bloqueio por
// tentativa errada (o CronSecret não precisa disso — ninguém de fora
// tem motivo pra ficar tentando adivinhar o secret do cron; aqui expõe
// dado financeiro de afiliado, então o cálculo de risco é diferente).
//
// Diferente do CronSecret (que deixa a rota aberta se a env var não
// estiver definida — aceitável pra um cron interno), aqui o secret vazio
// SEMPRE bloqueia: essas rotas expõem dado financeiro de todos os
// afiliados da plataforma, então "sem secret configurado" tem que
// significar "rota fechada", nunca "rota aberta por engano".
//
// IMPORTANTE: esconder a tela no frontend (ver DrenuxAfiliados.tsx) é só
// camuflagem — o path /drenux/* aparece no bundle JS pra qualquer um que
// abrir as devtools, então a proteção de verdade é o secret em si (use
// um valor longo e aleatório) mais esse bloqueio, não o fato da tela
// estar escondida.
func DrenuxAdminRequired(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if secret == "" {
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"erro": "DRENUX_ADMIN_SECRET não configurada"})
			return
		}

		ip := c.ClientIP()
		agora := time.Now()

		drenuxTentativasMu.Lock()
		t := drenuxTentativas[ip]
		if t != nil && agora.Before(t.bloqueadoAte) {
			drenuxTentativasMu.Unlock()
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"erro": "muitas tentativas — tenta de novo mais tarde"})
			return
		}
		drenuxTentativasMu.Unlock()

		recebido := c.GetHeader("X-Drenux-Admin-Secret")
		if subtle.ConstantTimeCompare([]byte(recebido), []byte(secret)) != 1 {
			drenuxTentativasMu.Lock()
			if t == nil || agora.Sub(t.primeiraFalha) > drenuxJanelaFalhas {
				t = &tentativaDrenux{primeiraFalha: agora}
				drenuxTentativas[ip] = t
			}
			t.falhas++
			if t.falhas >= drenuxMaxTentativas {
				t.bloqueadoAte = agora.Add(drenuxTempoBloqueio)
			}
			drenuxTentativasMu.Unlock()

			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"erro": "não autorizado"})
			return
		}

		// Secret certo — limpa qualquer histórico de tentativa errada
		// desse IP, pra não deixar um bloqueio "encostado" atrapalhando o
		// próximo acesso legítimo.
		drenuxTentativasMu.Lock()
		delete(drenuxTentativas, ip)
		drenuxTentativasMu.Unlock()

		c.Next()
	}
}
