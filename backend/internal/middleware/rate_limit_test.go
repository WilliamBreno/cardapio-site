package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func novoContextoTeste(ip string) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/qualquer", nil)
	c.Request.RemoteAddr = ip + ":12345"
	return c, w
}

func TestRateLimiterPermiteAteORajadaEDepoisBloqueia(t *testing.T) {
	rl := NovoRateLimiter(60, 3) // 1/seg, rajada de 3
	handler := rl.Middleware()

	for i := 0; i < 3; i++ {
		c, _ := novoContextoTeste("1.2.3.4")
		handler(c)
		if c.IsAborted() {
			t.Fatalf("requisição %d dentro da rajada foi bloqueada sem necessidade", i+1)
		}
	}

	c, w := novoContextoTeste("1.2.3.4")
	handler(c)
	if !c.IsAborted() || w.Code != http.StatusTooManyRequests {
		t.Fatalf("esperava 429 depois de estourar a rajada, veio código %d (abortado=%v)", w.Code, c.IsAborted())
	}
}

func TestRateLimiterIsolaIPsDiferentes(t *testing.T) {
	rl := NovoRateLimiter(60, 1) // rajada de 1 só, pra forçar bloqueio rápido
	handler := rl.Middleware()

	c1, _ := novoContextoTeste("10.0.0.1")
	handler(c1)
	if c1.IsAborted() {
		t.Fatal("primeira requisição do IP 10.0.0.1 não deveria ser bloqueada")
	}

	// Mesmo IP, segunda requisição imediata: estoura a rajada de 1.
	c1b, w1b := novoContextoTeste("10.0.0.1")
	handler(c1b)
	if !c1b.IsAborted() || w1b.Code != http.StatusTooManyRequests {
		t.Fatal("segunda requisição imediata do mesmo IP deveria ser bloqueada")
	}

	// IP diferente não deveria ser afetado pela cota do primeiro.
	c2, _ := novoContextoTeste("10.0.0.2")
	handler(c2)
	if c2.IsAborted() {
		t.Fatal("IP diferente (10.0.0.2) não deveria compartilhar cota com 10.0.0.1")
	}
}

func TestRateLimiterLimpaIPsInativos(t *testing.T) {
	rl := NovoRateLimiter(60, 1)
	c, _ := novoContextoTeste("192.168.0.1")
	rl.Middleware()(c)

	rl.mu.Lock()
	if len(rl.porIP) != 1 {
		rl.mu.Unlock()
		t.Fatalf("esperava 1 IP registrado, tem %d", len(rl.porIP))
	}
	// Simula inatividade: empurra o último uso pro passado, sem esperar
	// os 5 minutos reais do ticker de limpeza.
	for _, entrada := range rl.porIP {
		entrada.ultimoUso = time.Now().Add(-11 * time.Minute)
	}
	rl.mu.Unlock()

	rl.limparInativosAntesDe(time.Now().Add(-10 * time.Minute))

	rl.mu.Lock()
	restantes := len(rl.porIP)
	rl.mu.Unlock()

	if restantes != 0 {
		t.Fatalf("esperava mapa vazio depois de descartar IP inativo, sobrou %d", restantes)
	}
}
