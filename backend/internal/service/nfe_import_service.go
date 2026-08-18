package service

import (
	"encoding/xml"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/WilliamBreno/cardapio-backend/internal/domain"
	"github.com/WilliamBreno/cardapio-backend/internal/repository"
	"gorm.io/gorm"
)

// --- Structs de parsing do XML da NF-e (só os campos que usamos) ---
//
// O layout oficial da NF-e (SEFAZ) tem duas formas de arquivo: o
// "nfeProc" (nota + protocolo de autorização, o mais comum quando a
// nota é baixada do portal do fornecedor) e o "NFe" solto (só a nota,
// sem protocolo). Os dois têm a mesma estrutura interna a partir de
// <infNFe> — tentamos decodificar como nfeProc primeiro, e caímos pro
// NFe solto se não bater.
type nfeProcXML struct {
	XMLName xml.Name `xml:"nfeProc"`
	NFe     nfeXML   `xml:"NFe"`
}

type nfeXML struct {
	XMLName xml.Name  `xml:"NFe"`
	InfNFe  infNFeXML `xml:"infNFe"`
}

type infNFeXML struct {
	Ide  ideXML   `xml:"ide"`
	Emit emitXML  `xml:"emit"`
	Det  []detXML `xml:"det"`
}

type ideXML struct {
	NNF string `xml:"nNF"`
}

type emitXML struct {
	XNome string `xml:"xNome"`
}

type detXML struct {
	Prod prodXML `xml:"prod"`
}

type prodXML struct {
	XProd  string `xml:"xProd"`
	UCom   string `xml:"uCom"`
	QCom   string `xml:"qCom"`
	VUnCom string `xml:"vUnCom"`
}

// --- DTOs de resposta pro admin ---

// ItemNFeImportado é uma linha do XML já interpretada — devolvida na
// prévia pro admin decidir, na tela de conferência, se vincula a um
// insumo existente, cria um novo, ou ignora a linha.
type ItemNFeImportado struct {
	Nome           string  `json:"nome"`
	Unidade        string  `json:"unidade"`
	Quantidade     float64 `json:"quantidade"`
	ValorUnitario  float64 `json:"valor_unitario"`
	InsumoSugerido *uint   `json:"insumo_sugerido"`
}

// PreviewNFe é a prévia completa devolvida antes de qualquer escrita no
// banco — nenhum insumo/estoque é alterado nessa etapa.
type PreviewNFe struct {
	NumeroNota string             `json:"numero_nota"`
	Fornecedor string             `json:"fornecedor"`
	Itens      []ItemNFeImportado `json:"itens"`
}

// ConfirmarItemNFeInput é o que o admin decidiu, linha a linha, na tela
// de conferência.
type ConfirmarItemNFeInput struct {
	Acao           string // "vincular" | "criar" | "ignorar"
	InsumoID       *uint
	Nome           string
	UnidadeCompra  string
	UnidadeUso     string
	FatorConversao float64
	Quantidade     float64
	ValorUnitario  float64
}

type NFeImportService struct {
	insumoRepo             *repository.InsumoRepository
	movimentacaoInsumoRepo *repository.MovimentacaoInsumoRepository
}

func NewNFeImportService(db *gorm.DB) *NFeImportService {
	return &NFeImportService{
		insumoRepo:             repository.NewInsumoRepository(db),
		movimentacaoInsumoRepo: repository.NewMovimentacaoInsumoRepository(db),
	}
}

// Preview interpreta o XML e sugere, pra cada item, um insumo já
// cadastrado com o mesmo nome (comparação sem diferenciar
// maiúsculo/minúsculo) — não escreve nada no banco, é só leitura.
func (s *NFeImportService) Preview(lojaID uint, xmlContent string) (*PreviewNFe, error) {
	inf, err := parseNFeXML(xmlContent)
	if err != nil {
		return nil, err
	}

	itens := make([]ItemNFeImportado, 0, len(inf.Det))
	for _, det := range inf.Det {
		quantidade, err := strconv.ParseFloat(strings.TrimSpace(det.Prod.QCom), 64)
		if err != nil {
			return nil, fmt.Errorf("quantidade inválida no item %q: %w", det.Prod.XProd, err)
		}
		valorUnitario, err := strconv.ParseFloat(strings.TrimSpace(det.Prod.VUnCom), 64)
		if err != nil {
			return nil, fmt.Errorf("valor unitário inválido no item %q: %w", det.Prod.XProd, err)
		}

		item := ItemNFeImportado{
			Nome:          det.Prod.XProd,
			Unidade:       det.Prod.UCom,
			Quantidade:    quantidade,
			ValorUnitario: valorUnitario,
		}
		if insumo, err := s.insumoRepo.BuscarPorNome(lojaID, det.Prod.XProd); err == nil {
			item.InsumoSugerido = &insumo.ID
		}
		itens = append(itens, item)
	}

	return &PreviewNFe{
		NumeroNota: inf.Ide.NNF,
		Fornecedor: inf.Emit.XNome,
		Itens:      itens,
	}, nil
}

// Confirmar aplica o que o admin decidiu na tela de conferência: cria
// insumo novo, ou atualiza custo + soma estoque de um insumo já
// existente — sempre registrando a movimentação (Fase 9.1) pra manter o
// mesmo rastro de auditoria de uma reposição manual.
func (s *NFeImportService) Confirmar(lojaID uint, numeroNota string, itens []ConfirmarItemNFeInput) ([]domain.Insumo, error) {
	resultado := make([]domain.Insumo, 0, len(itens))
	motivo := "Importação NF-e"
	if numeroNota != "" {
		motivo = fmt.Sprintf("Importação NF-e nº %s", numeroNota)
	}

	for _, item := range itens {
		switch item.Acao {
		case "ignorar":
			continue

		case "criar":
			if err := validarInsumoInput(InsumoInput{
				Nome: item.Nome, UnidadeCompra: item.UnidadeCompra, UnidadeUso: item.UnidadeUso,
				FatorConversao: item.FatorConversao, CustoUnidadeCompra: item.ValorUnitario,
			}); err != nil {
				return nil, err
			}
			if item.Quantidade <= 0 {
				return nil, fmt.Errorf("quantidade inválida pro insumo novo %q", item.Nome)
			}
			// A quantidade da NF-e vem na unidade de COMPRA (qCom, mesma
			// unidade de item.UnidadeCompra) — o estoque é sempre guardado
			// na unidade de USO (ver domain.Insumo), então precisa
			// converter pelo fator antes de gravar (5kg comprados, fator
			// 1000, viram 5000g em estoque — nunca "5").
			estoqueInicial := item.Quantidade * item.FatorConversao
			insumo := domain.Insumo{
				LojaID:             lojaID,
				Nome:               item.Nome,
				UnidadeCompra:      item.UnidadeCompra,
				UnidadeUso:         item.UnidadeUso,
				FatorConversao:     item.FatorConversao,
				CustoUnidadeCompra: item.ValorUnitario,
				EstoqueAtual:       &estoqueInicial,
			}
			if err := s.insumoRepo.Criar(&insumo); err != nil {
				return nil, fmt.Errorf("criando insumo %q: %w", item.Nome, err)
			}
			s.registrarMovimentacao(lojaID, insumo.ID, estoqueInicial, estoqueInicial, motivo)
			resultado = append(resultado, insumo)

		case "vincular":
			if item.InsumoID == nil {
				return nil, errors.New("insumo_id é obrigatório pra vincular um item")
			}
			if item.Quantidade <= 0 {
				return nil, fmt.Errorf("quantidade inválida pro insumo %d", *item.InsumoID)
			}
			insumo, err := s.insumoRepo.BuscarPorID(*item.InsumoID)
			if err != nil {
				return nil, fmt.Errorf("insumo %d não encontrado", *item.InsumoID)
			}
			if insumo.LojaID != lojaID {
				return nil, fmt.Errorf("insumo %q não pertence a essa loja", insumo.Nome)
			}

			// Mesma conversão do caso "criar" acima: item.Quantidade está
			// na unidade de compra do insumo já cadastrado (não
			// necessariamente a mesma unidade escrita na NF-e — o insumo
			// já tem o fator dele próprio, é esse que vale), o estoque é
			// somado na unidade de uso.
			quantidadeConvertida := item.Quantidade * insumo.FatorConversao
			estoqueAnterior := 0.0
			if insumo.EstoqueAtual != nil {
				estoqueAnterior = *insumo.EstoqueAtual
			}
			novoEstoque := estoqueAnterior + quantidadeConvertida

			insumo.CustoUnidadeCompra = item.ValorUnitario
			insumo.EstoqueAtual = &novoEstoque
			if err := s.insumoRepo.Atualizar(insumo); err != nil {
				return nil, fmt.Errorf("atualizando insumo %q: %w", insumo.Nome, err)
			}
			s.registrarMovimentacao(lojaID, insumo.ID, quantidadeConvertida, novoEstoque, motivo)
			resultado = append(resultado, *insumo)

		default:
			return nil, fmt.Errorf("ação inválida %q", item.Acao)
		}
	}

	return resultado, nil
}

func (s *NFeImportService) registrarMovimentacao(lojaID, insumoID uint, quantidade, estoqueResultante float64, motivo string) {
	mov := domain.MovimentacaoInsumo{
		LojaID:            lojaID,
		InsumoID:          insumoID,
		Tipo:              domain.MovimentoEstoqueReposicao,
		Quantidade:        quantidade,
		EstoqueResultante: estoqueResultante,
		Motivo:            motivo,
	}
	// Mesmo espírito de EstoqueService.registrarMovimentacao: falha ao
	// registrar o histórico não deve desfazer a importação em si, só
	// perde a rastreabilidade desse evento específico.
	_ = s.movimentacaoInsumoRepo.Criar(&mov)
}

// parseNFeXML tenta decodificar como nfeProc (nota + protocolo, o
// formato mais comum) e cai pro NFe solto se não bater.
func parseNFeXML(xmlContent string) (*infNFeXML, error) {
	data := []byte(xmlContent)

	var proc nfeProcXML
	if err := xml.Unmarshal(data, &proc); err == nil && len(proc.NFe.InfNFe.Det) > 0 {
		return &proc.NFe.InfNFe, nil
	}

	var nfe nfeXML
	if err := xml.Unmarshal(data, &nfe); err != nil {
		return nil, errors.New("XML não parece ser uma NF-e válida (esperado <nfeProc> ou <NFe>)")
	}
	if len(nfe.InfNFe.Det) == 0 {
		return nil, errors.New("nenhum item (<det>) encontrado na nota fiscal")
	}
	return &nfe.InfNFe, nil
}
