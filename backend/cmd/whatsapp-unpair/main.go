// Comando standalone: remove a sessão de WhatsApp pareada atualmente —
// use antes de rodar cmd/whatsapp-pair de novo pra reconectar (ex: sessão
// deslogada remotamente, número banido temporariamente, ou troca pra
// outro número). Sem isso, cmd/whatsapp-pair encontra a sessão salva
// (mesmo quebrada) e desiste sem mostrar QR code nenhum.
//
// Uso (apontando pra mesma DATABASE_URL de produção):
//
//	go run ./cmd/whatsapp-unpair
//	go run ./cmd/whatsapp-pair
//
// Depois de escanear o QR code novo, reinicie o serviço principal — ele
// só conecta ao WhatsApp uma vez, no boot.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/WilliamBreno/cardapio-backend/internal/notification"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

	connString := os.Getenv("DATABASE_URL")
	if connString == "" {
		fmt.Println("Defina a variável de ambiente DATABASE_URL antes de rodar")
		os.Exit(1)
	}

	if err := notification.Unpair(context.Background(), connString); err != nil {
		fmt.Println("Erro ao remover sessão pareada:", err)
		os.Exit(1)
	}
}
