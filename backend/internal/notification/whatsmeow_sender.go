package notification

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/WilliamBreno/cardapio-backend/internal/domain"
	"github.com/mdp/qrterminal/v3"
	"go.mau.fi/whatsmeow"
	waE2E "go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"

	_ "github.com/lib/pq" // driver postgres registrado no database/sql, usado pelo sqlstore
)

// WhatsmeowSender implementa NotificationSender usando o protocolo do
// WhatsApp Web (via whatsmeow) — um único número da plataforma manda as
// mensagens de todas as lojas, cada uma identificada pelo nome no texto.
type WhatsmeowSender struct {
	client *whatsmeow.Client
}

// NewWhatsmeowSender conecta usando uma sessão JÁ pareada anteriormente
// (veja Pair e cmd/whatsapp-pair). Use no startup do serviço principal.
func NewWhatsmeowSender(ctx context.Context, connString string) (*WhatsmeowSender, error) {
	_, deviceStore, err := abrirDeviceStore(ctx, connString)
	if err != nil {
		return nil, err
	}

	if deviceStore.ID == nil {
		return nil, fmt.Errorf("nenhum número pareado ainda — rode antes o comando cmd/whatsapp-pair")
	}

	clientLog := waLog.Stdout("WhatsmeowClient", "ERROR", true)
	client := whatsmeow.NewClient(deviceStore, clientLog)
	client.AddEventHandler(logarEventoWhatsapp)
	if err := client.Connect(); err != nil {
		return nil, fmt.Errorf("conectando sessão existente: %w", err)
	}

	return &WhatsmeowSender{client: client}, nil
}

// logarEventoWhatsapp é só diagnóstico (não muda nenhum comportamento de
// envio) — registrado pra investigar mensagens que "saem" do número da
// plataforma mas não chegam no destinatário (ver Tarefa 0,
// docs/plano-melhorias-drenux.md). Sem isso, o serviço não tinha
// nenhuma visibilidade sobre: confirmação de entrega/leitura real (o
// SendMessage só confirma que o servidor do WhatsApp aceitou a
// mensagem, não que ela chegou no aparelho de quem recebe — são coisas
// diferentes), nem sobre a conta ter sido desconectada, banida
// temporariamente ou substituída por outra sessão — qualquer um desses
// explicaria o sintoma relatado sem aparecer como erro no envio em si.
func logarEventoWhatsapp(evt any) {
	switch e := evt.(type) {
	case *events.Receipt:
		log.Printf("whatsapp: recibo %s de %s pras mensagens %v (%s)", e.Type, e.Sender, e.MessageIDs, e.Timestamp.Format(time.RFC3339))
	case *events.Connected:
		log.Println("whatsapp: conectado")
	case *events.Disconnected:
		log.Println("whatsapp: desconectado (websocket fechado pelo servidor)")
	case *events.LoggedOut:
		log.Printf("whatsapp: ALERTA — sessão desconectada/deslogada (onConnect=%v, motivo=%v). Nenhuma mensagem vai sair até repareear com cmd/whatsapp-pair.", e.OnConnect, e.Reason)
	case *events.TemporaryBan:
		log.Printf("whatsapp: ALERTA — número banido temporariamente pelo WhatsApp: %s", e.String())
	case *events.ConnectFailure:
		log.Printf("whatsapp: ALERTA — falha de conexão (motivo=%s, msg=%s)", e.Reason.String(), e.Message)
	case *events.StreamReplaced:
		log.Println("whatsapp: ALERTA — sessão substituída por outra conexão com as mesmas credenciais (rodando o processo em duplicidade em algum lugar?)")
	}
}

// Pair faz o pareamento inicial via QR code. Rode uma única vez,
// localmente (não em produção), apontando pra mesma DATABASE_URL de
// produção — depois que a sessão é salva no banco, o serviço principal
// nunca mais pede QR code, mesmo após redeploy.
func Pair(ctx context.Context, connString string) error {
	_, deviceStore, err := abrirDeviceStore(ctx, connString)
	if err != nil {
		return err
	}
	if deviceStore.ID != nil {
		fmt.Println("Esse número já está pareado, nada a fazer.")
		return nil
	}

	clientLog := waLog.Stdout("WhatsmeowClient", "INFO", true)
	client := whatsmeow.NewClient(deviceStore, clientLog)

	qrChan, _ := client.GetQRChannel(ctx)
	if err := client.Connect(); err != nil {
		return fmt.Errorf("conectando pela primeira vez: %w", err)
	}

	for evt := range qrChan {
		if evt.Event == "code" {
			qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)
			fmt.Println("Escaneie o QR code acima com o WhatsApp do número da plataforma")
		} else {
			fmt.Println("Status do pareamento:", evt.Event)
		}
	}

	// Depois que o pareamento é confirmado, o WhatsApp ainda troca
	// mensagens internas (sincronização inicial, upload das chaves de
	// criptografia/prekeys). Desconectar rápido demais nesse momento
	// deixa esse processo pela metade e corrompe o estado salvo no
	// banco — foi exatamente isso que causou o erro de chave
	// estrangeira na primeira tentativa. Esperamos um tempo de
	// segurança antes de fechar a conexão.
	fmt.Println("Pareado! Aguardando a sincronização inicial terminar (não feche ainda)...")
	time.Sleep(15 * time.Second)
	fmt.Println("Sincronização concluída.")

	client.Disconnect()
	return nil
}

// Unpair remove a sessão pareada atual (se existir) — necessário antes
// de rodar Pair de novo pra reconectar, porque GetFirstDevice sempre
// devolve a sessão já salva se ela existir, mesmo deslogada/banida pelo
// WhatsApp (ver cmd/whatsapp-unpair). Rode localmente, apontando pra
// mesma DATABASE_URL de produção, e reinicie o serviço principal depois
// de repareear — ele só conecta ao WhatsApp uma vez, no boot.
func Unpair(ctx context.Context, connString string) error {
	container, deviceStore, err := abrirDeviceStore(ctx, connString)
	if err != nil {
		return err
	}
	if deviceStore.ID == nil {
		fmt.Println("Nenhuma sessão pareada encontrada — nada a remover. Já pode rodar cmd/whatsapp-pair direto.")
		return nil
	}

	numero := deviceStore.ID.User
	if err := container.DeleteDevice(ctx, deviceStore); err != nil {
		return fmt.Errorf("removendo sessão pareada: %w", err)
	}

	fmt.Printf("Sessão do número %s removida. Agora rode 'go run ./cmd/whatsapp-pair' pra escanear um QR code novo.\n", numero)
	return nil
}

func abrirDeviceStore(ctx context.Context, connString string) (*sqlstore.Container, *store.Device, error) {
	dbLog := waLog.Stdout("WhatsmeowDB", "ERROR", true)
	container, err := sqlstore.New(ctx, "postgres", connString, dbLog)
	if err != nil {
		return nil, nil, fmt.Errorf("conectando store do whatsmeow: %w", err)
	}
	deviceStore, err := container.GetFirstDevice(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("obtendo device store: %w", err)
	}
	return container, deviceStore, nil
}

func (s *WhatsmeowSender) EnviarConfirmacaoPedido(ctx context.Context, pedido *domain.Pedido, lojaNome string) error {
	return s.enviarTexto(ctx, pedido.ClienteTelefone, montarMensagemCliente(pedido, lojaNome))
}

func (s *WhatsmeowSender) EnviarNotificacaoAdmin(ctx context.Context, pedido *domain.Pedido, lojaNome, telefoneAdmin string) error {
	if telefoneAdmin == "" {
		return fmt.Errorf("loja %q não tem whatsapp configurado", lojaNome)
	}
	return s.enviarTexto(ctx, telefoneAdmin, montarMensagemAdmin(pedido, lojaNome))
}

func (s *WhatsmeowSender) EnviarTextoAdmin(ctx context.Context, telefoneAdmin, texto string) error {
	if telefoneAdmin == "" {
		return fmt.Errorf("telefone do admin não informado")
	}
	return s.enviarTexto(ctx, telefoneAdmin, texto)
}

// EnviarSaiuParaEntrega avisa o cliente que o pedido saiu pra entrega,
// com o link de rastreamento em tempo real.
func (s *WhatsmeowSender) EnviarSaiuParaEntrega(ctx context.Context, pedido *domain.Pedido, lojaNome, linkRastreamento string) error {
	return s.enviarTexto(ctx, pedido.ClienteTelefone, montarMensagemSaiuParaEntrega(pedido, lojaNome, linkRastreamento))
}

// enviarTexto resolve o número pelo IsOnWhatsApp antes de mandar a
// mensagem, em vez de montar o destinatário direto a partir do número
// puro. É essa consulta ao servidor da própria WhatsApp que resolve o
// identificador interno (LID) certo — sem isso, o envio falha
// silenciosamente com "no LID found", mesmo pra números válidos.
func (s *WhatsmeowSender) enviarTexto(ctx context.Context, telefone, texto string) error {
	// Defesa a mais, além da normalização na escrita (ver
	// service.NormalizarTelefone): número salvo com espaço/hífen/sem o 55
	// resolve pro JID errado (ou nenhum) sem gerar erro nenhum visível —
	// foi exatamente o sintoma investigado na Tarefa 0
	// (docs/plano-melhorias-drenux.md): mensagem "sai" no app mas não
	// chega em ninguém real.
	limpo := apenasDigitos(telefone)
	if !strings.HasPrefix(limpo, "55") {
		limpo = "55" + limpo
	}

	resultados, err := s.client.IsOnWhatsApp(ctx, []string{"+" + limpo})
	if err != nil {
		log.Printf("whatsapp: erro verificando número %q (normalizado de %q) no WhatsApp: %v", limpo, telefone, err)
		return fmt.Errorf("verificando número %s no WhatsApp: %w", limpo, err)
	}
	if len(resultados) == 0 || !resultados[0].IsIn {
		log.Printf("whatsapp: número %q (normalizado de %q) NÃO está registrado no WhatsApp — nenhuma mensagem foi enviada", limpo, telefone)
		return fmt.Errorf("número %s não está registrado no WhatsApp", limpo)
	}
	jid := resultados[0].JID

	msg := &waE2E.Message{Conversation: proto.String(texto)}
	resp, err := s.client.SendMessage(ctx, jid, msg)
	if err != nil {
		log.Printf("whatsapp: erro enviando mensagem pra %s (JID %s): %v", limpo, jid, err)
		return fmt.Errorf("enviando mensagem para %s: %w", limpo, err)
	}
	log.Printf("whatsapp: mensagem %s aceita pelo servidor pra %s (JID %s) às %s — isso confirma só que o WhatsApp recebeu, não que chegou no aparelho de quem recebe (ver eventos de Receipt no log pra confirmar entrega)", resp.ID, limpo, jid, resp.Timestamp.Format(time.RFC3339))
	return nil
}

// apenasDigitos remove tudo que não for número — mesmo critério de
// service.NormalizarTelefone, duplicado aqui porque o pacote
// notification não pode importar service (import cíclico: service já
// importa notification).
func apenasDigitos(telefone string) string {
	var sb strings.Builder
	for _, r := range telefone {
		if r >= '0' && r <= '9' {
			sb.WriteRune(r)
		}
	}
	return sb.String()
}

// Close encerra a conexão com o WhatsApp. Chame no shutdown do serviço.
func (s *WhatsmeowSender) Close() {
	s.client.Disconnect()
}
