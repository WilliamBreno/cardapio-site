package service

import (
	"errors"
	"fmt"

	"github.com/WilliamBreno/cardapio-backend/internal/domain"
	"github.com/WilliamBreno/cardapio-backend/internal/repository"
	"gorm.io/gorm"
)

// GuardadosService cuida do fluxo pós-compra de "guardar e entregar
// depois": listar o que um cliente tem guardado, cotar o frete de uma
// entrega desses itens e efetivar a solicitação de entrega.
type GuardadosService struct {
	db               *gorm.DB
	lojaRepo         *repository.LojaRepository
	itemRepo         *repository.ItemPedidoRepository
	solicitacaoRepo  *repository.SolicitacaoEntregaRepository
	distanciaService *DistanciaService
}

func NewGuardadosService(db *gorm.DB, distanciaService *DistanciaService) *GuardadosService {
	return &GuardadosService{
		db:               db,
		lojaRepo:         repository.NewLojaRepository(db),
		itemRepo:         repository.NewItemPedidoRepository(db),
		solicitacaoRepo:  repository.NewSolicitacaoEntregaRepository(db),
		distanciaService: distanciaService,
	}
}

// ListarPorSlugETelefone devolve os itens guardados e ainda disponíveis
// de um cliente numa loja, identificado pelo slug e telefone — mesmo
// padrão de identificação por telefone já usado no histórico público.
func (s *GuardadosService) ListarPorSlugETelefone(slug, telefone string) ([]repository.ItemGuardado, error) {
	loja, err := s.lojaRepo.BuscarPorSlug(slug)
	if err != nil {
		return nil, errors.New("loja não encontrada")
	}
	return s.itemRepo.ListarGuardadosPorTelefone(loja.ID, telefone)
}

// CotacaoFrete é o resultado de uma cotação de frete de itens guardados —
// só uma prévia, o valor final é sempre recalculado no servidor na hora
// de efetivar a solicitação de entrega.
type CotacaoFrete struct {
	DistanciaKm float64
	MesmaRegiao bool
	ValorFrete  float64
}

// CotarFrete calcula o frete pra entregar um conjunto de itens guardados
// num endereço, sem reivindicar nada ainda. Decide entre a fórmula
// regional (por km, mesma cidade/estado da loja) e a fórmula estimada
// (peso + distância, fora da região) comparando a cidade/estado
// geocodificados do destino com os já salvos na loja.
func (s *GuardadosService) CotarFrete(slug, telefone string, endereco EnderecoEstruturado, itemIDs []uint) (*CotacaoFrete, error) {
	loja, err := s.lojaRepo.BuscarPorSlug(slug)
	if err != nil {
		return nil, errors.New("loja não encontrada")
	}
	if loja.Latitude == 0 && loja.Longitude == 0 {
		return nil, errors.New("essa loja ainda não configurou o endereço de origem pra calcular o frete")
	}

	itens, err := s.itemRepo.BuscarGuardadosPorIDs(loja.ID, telefone, itemIDs)
	if err != nil {
		return nil, fmt.Errorf("buscando itens guardados: %w", err)
	}
	if len(itens) != len(itemIDs) {
		return nil, errors.New("um ou mais itens selecionados não estão disponíveis pra entrega")
	}

	pesoTotal := 0
	for _, item := range itens {
		pesoTotal += item.PesoGramas * item.Quantidade
	}

	return s.calcularFrete(*loja, endereco, pesoTotal)
}

func (s *GuardadosService) calcularFrete(loja domain.Loja, endereco EnderecoEstruturado, pesoGramas int) (*CotacaoFrete, error) {
	destino, err := s.distanciaService.GeocodificarEstruturadoDetalhado(endereco)
	if err != nil {
		return nil, errors.New("não conseguimos localizar esse endereço")
	}

	origem := Coordenada{Latitude: loja.Latitude, Longitude: loja.Longitude}
	distancia := s.distanciaService.DistanciaKm(origem, destino.Coordenada)

	mesmaRegiao := destino.Cidade != "" && destino.Estado != "" &&
		destino.Cidade == loja.Cidade && destino.Estado == loja.Estado

	var valor float64
	if mesmaRegiao {
		valor = CalcularTaxaPorKm(distancia, loja.TaxaEntregaBase, loja.TaxaEntregaPorKm)
	} else {
		valor = CalcularFreteEstimadoCorreios(pesoGramas, distancia)
	}

	return &CotacaoFrete{DistanciaKm: distancia, MesmaRegiao: mesmaRegiao, ValorFrete: valor}, nil
}

// SolicitarEntregaInput é o que o cliente informa na hora de pedir a
// entrega dos itens que tem guardados.
type SolicitarEntregaInput struct {
	ClienteNome     string
	ClienteTelefone string
	Endereco        string
	EnderecoGeo     EnderecoEstruturado
	ItemIDs         []uint
}

// SolicitarEntrega reivindica os itens guardados selecionados e cria a
// SolicitacaoEntrega correspondente, com o frete recalculado no servidor
// (nunca confia no valor cotado antes). Tudo dentro de uma transação: se
// outra solicitação já reivindicou algum dos itens nesse meio-tempo, a
// operação inteira é desfeita.
func (s *GuardadosService) SolicitarEntrega(slug string, input SolicitarEntregaInput) (*domain.SolicitacaoEntrega, error) {
	loja, err := s.lojaRepo.BuscarPorSlug(slug)
	if err != nil {
		return nil, errors.New("loja não encontrada")
	}
	if len(input.ItemIDs) == 0 {
		return nil, errors.New("selecione ao menos um item guardado pra entregar")
	}
	if input.Endereco == "" {
		return nil, errors.New("endereço de entrega é obrigatório")
	}

	var solicitacao domain.SolicitacaoEntrega

	err = s.db.Transaction(func(tx *gorm.DB) error {
		itemRepo := repository.NewItemPedidoRepository(tx)
		solicitacaoRepo := repository.NewSolicitacaoEntregaRepository(tx)

		itens, err := itemRepo.BuscarGuardadosPorIDs(loja.ID, input.ClienteTelefone, input.ItemIDs)
		if err != nil {
			return fmt.Errorf("buscando itens guardados: %w", err)
		}
		if len(itens) != len(input.ItemIDs) {
			return errors.New("um ou mais itens selecionados não estão mais disponíveis")
		}

		pesoTotal := 0
		for _, item := range itens {
			pesoTotal += item.PesoGramas * item.Quantidade
		}

		cotacao, err := s.calcularFrete(*loja, input.EnderecoGeo, pesoTotal)
		if err != nil {
			return err
		}

		tipoCalculo := "correios_estimado"
		if cotacao.MesmaRegiao {
			tipoCalculo = "regional"
		}

		// Peso só importa de verdade quando o frete saiu da fórmula
		// estimada (fora da região) — na fórmula regional (por km) o peso
		// nunca é usado, então não faz sentido marcar como pendente.
		pesoPendente := !cotacao.MesmaRegiao && pesoPendenteEmItens(itens)

		solicitacao = domain.SolicitacaoEntrega{
			LojaID:          loja.ID,
			ClienteNome:     input.ClienteNome,
			ClienteTelefone: input.ClienteTelefone,
			EnderecoEntrega: input.Endereco,
			DistanciaKm:     cotacao.DistanciaKm,
			TipoCalculo:     tipoCalculo,
			PesoTotalGramas: pesoTotal,
			ValorFrete:      cotacao.ValorFrete,
			PesoPendente:    pesoPendente,
			Status:          domain.StatusSolicitacaoAguardandoPagamento,
		}
		if err := solicitacaoRepo.Criar(&solicitacao); err != nil {
			return fmt.Errorf("criando solicitação de entrega: %w", err)
		}

		afetados, err := itemRepo.MarcarComoReivindicados(input.ItemIDs, solicitacao.ID)
		if err != nil {
			return fmt.Errorf("reivindicando itens guardados: %w", err)
		}
		if int(afetados) != len(input.ItemIDs) {
			return errors.New("um dos itens selecionados acabou de ser reivindicado por outra solicitação — tente novamente")
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return &solicitacao, nil
}
