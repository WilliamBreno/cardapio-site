package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// entradaLimite guarda o limitador de um IP e quando ele foi usado pela
// última vez — o "último uso" é o que permite descartar IPs inativos
// (limparPeriodicamente) sem deixar o mapa crescer pra sempre num
// processo de vida longa.
type entradaLimite struct {
	limiter   *rate.Limiter
	ultimoUso time.Time
}

// RateLimiter é uma fábrica de middlewares de limite de requisições por
// IP (token bucket, via golang.org/x/time/rate). Cada instância tem seu
// próprio mapa de IPs — um limitador "geral" (todas as rotas) e um
// "estrito" (login/cadastro) não compartilham cota entre si, de
// propósito: passar no geral não dá passe livre no estrito.
type RateLimiter struct {
	mu                    sync.Mutex
	porIP                 map[string]*entradaLimite
	requisicoesPorSegundo rate.Limit
	burst                 int
}

// NovoRateLimiter cria um limitador por IP: até `burst` requisições de
// uma vez (rajada), repondo à taxa de `requisicoesPorMinuto` por minuto
// depois disso. Inicia uma goroutine de limpeza que descarta IPs sem
// requisição há mais de 10 minutos.
func NovoRateLimiter(requisicoesPorMinuto, burst int) *RateLimiter {
	rl := &RateLimiter{
		porIP:                 make(map[string]*entradaLimite),
		requisicoesPorSegundo: rate.Limit(float64(requisicoesPorMinuto) / 60.0),
		burst:                 burst,
	}
	go rl.limparPeriodicamente()
	return rl
}

func (rl *RateLimiter) limiterDoIP(ip string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	entrada, existe := rl.porIP[ip]
	if !existe {
		entrada = &entradaLimite{limiter: rate.NewLimiter(rl.requisicoesPorSegundo, rl.burst)}
		rl.porIP[ip] = entrada
	}
	entrada.ultimoUso = time.Now()
	return entrada.limiter
}

// limparPeriodicamente roda pra sempre num processo de vida longa — sem
// isso, cada IP diferente que já bateu na API uma vez só ficaria pra
// sempre no mapa, vazando memória aos poucos.
func (rl *RateLimiter) limparPeriodicamente() {
	for {
		time.Sleep(5 * time.Minute)
		rl.limparInativosAntesDe(time.Now().Add(-10 * time.Minute))
	}
}

// limparInativosAntesDe descarta os IPs cujo último uso foi antes do
// limite informado — extraída à parte de limparPeriodicamente só pra dar
// pra testar a lógica de limpeza sem precisar esperar 5 minutos de
// verdade.
func (rl *RateLimiter) limparInativosAntesDe(limite time.Time) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	for ip, entrada := range rl.porIP {
		if entrada.ultimoUso.Before(limite) {
			delete(rl.porIP, ip)
		}
	}
}

// Middleware devolve o gin.HandlerFunc pronto pra usar em router.Use(...)
// (rota única) ou num grupo de rotas. Usa c.ClientIP() — o IP real da
// conexão TCP, já que main.go chama router.SetTrustedProxies(nil) (não
// confia em X-Forwarded-For vindo do próprio cliente, que seria fácil de
// forjar pra burlar o limite).
func (rl *RateLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		limiter := rl.limiterDoIP(c.ClientIP())
		if !limiter.Allow() {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"erro": "muitas requisições — aguarde um instante e tente de novo"})
			return
		}
		c.Next()
	}
}
