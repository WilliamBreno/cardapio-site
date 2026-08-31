package service

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/WilliamBreno/cardapio-backend/internal/domain"
)

// DistanciaService cuida de duas coisas: transformar um endereço em
// coordenadas (geocodificação) e calcular a distância entre dois pontos.
//
// Usamos o Nominatim (OpenStreetMap) para geocodificação — é gratuito e não
// exige chave de API, mas pede que respeitemos um limite informal de ~1
// requisição por segundo (ver aguardarLimiteTaxaNominatim). Para o volume
// de um SaaS nessa fase, isso não é um problema.
type DistanciaService struct {
	httpClient *http.Client
}

func NewDistanciaService() *DistanciaService {
	return &DistanciaService{httpClient: &http.Client{}}
}

type Coordenada struct {
	Latitude  float64
	Longitude float64
}

// EnderecoEstruturado são os campos separados de um endereço (o mesmo
// formulário usado no frontend em EnderecoCampos.tsx) — usado pra montar
// consultas estruturadas ao Nominatim, que são muito mais confiáveis do
// que jogar o endereço inteiro como texto livre numa única query (ver
// comentário em tentativasEstruturadas).
type EnderecoEstruturado struct {
	Rua         string
	Numero      string
	Complemento string
	Bairro      string
	Cidade      string
	Estado      string
	CEP         string
}

type nominatimResultado struct {
	Lat     string            `json:"lat"`
	Lon     string            `json:"lon"`
	Address nominatimEndereco `json:"address"`
}

type nominatimEndereco struct {
	Cidade      string `json:"city"`
	Municipio   string `json:"municipality"`
	Vila        string `json:"village"`
	Cidadezinha string `json:"town"`
	Estado      string `json:"state"`
}

// GeocodificacaoDetalhada é o resultado de GeocodificarEstruturadoDetalhado
// — além das coordenadas, traz cidade/estado, usados pra decidir se um
// destino está na mesma região da loja ou fora dela.
type GeocodificacaoDetalhada struct {
	Coordenada
	Cidade string
	Estado string
}

// GeocodificarEstruturado transforma um endereço (em campos separados) em
// coordenadas de latitude/longitude. Retorna erro se o endereço não puder
// ser localizado.
func (s *DistanciaService) GeocodificarEstruturado(end EnderecoEstruturado) (*Coordenada, error) {
	resultado, err := s.geocodificarComFallback(end, false)
	if err != nil {
		return nil, err
	}
	return &resultado.Coordenada, nil
}

// GeocodificarEstruturadoDetalhado é igual a GeocodificarEstruturado, mas
// também pede o detalhamento do endereço (cidade, estado) ao Nominatim —
// usado pra decidir se um destino de entrega está na mesma cidade/estado
// da loja (frete por km) ou fora dela (frete estimado por peso+distância).
func (s *DistanciaService) GeocodificarEstruturadoDetalhado(end EnderecoEstruturado) (*GeocodificacaoDetalhada, error) {
	return s.geocodificarComFallback(end, true)
}

// tentativasEstruturadas monta, em ordem de precisão decrescente, os
// conjuntos de parâmetros estruturados a tentar no Nominatim.
//
// Por que estruturado em vez de uma única query de texto livre: testamos
// contra endereços reais (ex: "Rua Geraldo Menezes Carvalho, 161, Suíça,
// Aracaju - SE, 49050-360") e a busca de texto livre falha com frequência
// — o parser do Nominatim tenta casar cada trecho separado por vírgula
// contra o nome exato que o OpenStreetMap usa (nesse exemplo, o bairro é
// grafado "Suíssa" no OSM, não "Suíça" como no cadastro dos Correios — a
// diferença sozinha já derruba a busca de texto livre inteira). Os
// parâmetros estruturados (street/city/state/postalcode) casam cada campo
// de forma independente, então um bairro ou CEP que não bate exatamente
// não invalida o resto — e ainda evita o risco de "achar" um resultado
// completamente errado (ex: buscar só o CEP como texto livre já retornou,
// em teste, um endereço em outro estado).
//
// A cadeia de fallback (rua+cidade+estado+cep → rua+cidade+estado →
// cidade+estado+cep → cidade+estado) troca precisão por robustez: rua não
// mapeada ou CEP com pequena divergência do que o OSM registra não fazem o
// pedido falhar por completo — na pior das hipóteses cai pro centro da
// cidade, o que ainda é muito melhor do que recusar o frete.
func tentativasEstruturadas(end EnderecoEstruturado) []url.Values {
	rua := strings.TrimSpace(end.Rua)
	numero := strings.TrimSpace(end.Numero)
	cidade := strings.TrimSpace(end.Cidade)
	estado := strings.TrimSpace(end.Estado)
	cep := strings.TrimSpace(end.CEP)

	if cidade == "" || estado == "" {
		return nil
	}

	rua = strings.TrimSpace(strings.ReplaceAll(rua, ",", " "))
	ruaCompleta := rua
	if numero != "" {
		ruaCompleta = rua + ", " + numero
	}

	base := func() url.Values {
		v := url.Values{}
		v.Set("format", "json")
		v.Set("limit", "1")
		v.Set("countrycodes", "br")
		v.Set("country", "Brasil")
		v.Set("city", cidade)
		v.Set("state", estado)
		return v
	}

	var tentativas []url.Values

	if ruaCompleta != "" {
		if cep != "" {
			v := base()
			v.Set("street", ruaCompleta)
			v.Set("postalcode", cep)
			tentativas = append(tentativas, v)
		}
		v := base()
		v.Set("street", ruaCompleta)
		tentativas = append(tentativas, v)
	}

	if cep != "" {
		v := base()
		v.Set("postalcode", cep)
		tentativas = append(tentativas, v)
	}

	tentativas = append(tentativas, base())

	return tentativas
}

// GeocodificarTextoLivre transforma um endereço já formatado como uma
// única string (ex: "Rua X, 123, Bairro, Cidade - UF, CEP") em
// coordenadas — usado só pra pedido antigo, criado antes de o campo
// EnderecoEntregaGeo (estruturado) existir, cujos campos separados
// (rua/número/bairro/cidade/estado/CEP) nunca foram guardados, só o
// texto já achatado (ver PedidoService.PreencherDestinoGeoFaltantes).
//
// Menos confiável que GeocodificarEstruturado (ver o comentário de
// tentativasEstruturadas sobre por que texto livre falha mais) — é o
// melhor disponível pra esse pedido específico, não o caminho normal.
func (s *DistanciaService) GeocodificarTextoLivre(enderecoCompleto string) (*Coordenada, error) {
	texto := strings.TrimSpace(enderecoCompleto)
	if texto == "" {
		return nil, fmt.Errorf("endereço vazio")
	}

	params := url.Values{}
	params.Set("format", "json")
	params.Set("limit", "1")
	params.Set("countrycodes", "br")
	params.Set("q", texto)

	resultado, err := s.buscarNominatim(params)
	if err != nil {
		return nil, err
	}
	return &resultado.Coordenada, nil
}

func (s *DistanciaService) geocodificarComFallback(end EnderecoEstruturado, comEndereco bool) (*GeocodificacaoDetalhada, error) {
	tentativas := tentativasEstruturadas(end)
	if len(tentativas) == 0 {
		return nil, fmt.Errorf("endereço incompleto — informe ao menos cidade e estado")
	}

	var ultimoErr error
	for _, params := range tentativas {
		if comEndereco {
			params.Set("addressdetails", "1")
		}
		resultado, err := s.buscarNominatim(params)
		if err == nil {
			return resultado, nil
		}
		ultimoErr = err
	}

	return nil, ultimoErr
}

var (
	nominatimMu       sync.Mutex
	nominatimUltimaEm time.Time
)

// aguardarLimiteTaxaNominatim serializa TODAS as chamadas ao Nominatim
// feitas pelo processo (mesmo vindas de requisições concorrentes de
// clientes diferentes) com um espaçamento mínimo entre elas.
//
// Sem isso, um pico de tráfego (vários clientes cotando frete ao mesmo
// tempo) ou a própria cadeia de fallback em geocodificarComFallback
// disparando tentativas em sequência rápida ultrapassam o limite informal
// de ~1 requisição/segundo do Nominatim — e o Nominatim reage devolvendo
// HTTP 200 com corpo `[]` (lista vazia), indistinguível de "endereço não
// encontrado". Esse foi, na prática, a causa raiz de um endereço real e
// válido ser rejeitado em teste manual: as tentativas de fallback estavam
// sendo disparadas rápido demais uma atrás da outra.
func aguardarLimiteTaxaNominatim() {
	nominatimMu.Lock()
	defer nominatimMu.Unlock()

	const intervaloMinimo = 1100 * time.Millisecond
	espera := intervaloMinimo - time.Since(nominatimUltimaEm)
	if espera > 0 {
		time.Sleep(espera)
	}
	nominatimUltimaEm = time.Now()
}

func (s *DistanciaService) buscarNominatim(params url.Values) (*GeocodificacaoDetalhada, error) {
	aguardarLimiteTaxaNominatim()

	baseURL := "https://nominatim.openstreetmap.org/search"

	req, err := http.NewRequest("GET", baseURL+"?"+params.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("montando requisição de geocodificação: %w", err)
	}
	req.Header.Set("User-Agent", "Drenux/1.0 (contato: williamdevpy@gmail.com)")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("chamando Nominatim: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Nominatim retornou status %d", resp.StatusCode)
	}

	var resultados []nominatimResultado
	if err := json.NewDecoder(resp.Body).Decode(&resultados); err != nil {
		return nil, fmt.Errorf("lendo resposta do Nominatim: %w", err)
	}

	if len(resultados) == 0 {
		return nil, fmt.Errorf("endereço não encontrado (%s)", params.Encode())
	}

	lat, err := strconv.ParseFloat(resultados[0].Lat, 64)
	if err != nil {
		return nil, fmt.Errorf("interpretando latitude: %w", err)
	}
	lon, err := strconv.ParseFloat(resultados[0].Lon, 64)
	if err != nil {
		return nil, fmt.Errorf("interpretando longitude: %w", err)
	}

	endr := resultados[0].Address
	cidade := endr.Cidade
	if cidade == "" {
		cidade = endr.Municipio
	}
	if cidade == "" {
		cidade = endr.Cidadezinha
	}
	if cidade == "" {
		cidade = endr.Vila
	}

	return &GeocodificacaoDetalhada{
		Coordenada: Coordenada{Latitude: lat, Longitude: lon},
		Cidade:     cidade,
		Estado:     endr.Estado,
	}, nil
}

// DistanciaKm calcula a distância em linha reta (fórmula de Haversine)
// entre dois pontos, em quilômetros.
func (s *DistanciaService) DistanciaKm(origem, destino Coordenada) float64 {
	const raioTerraKm = 6371.0

	lat1 := origem.Latitude * math.Pi / 180
	lat2 := destino.Latitude * math.Pi / 180
	deltaLat := (destino.Latitude - origem.Latitude) * math.Pi / 180
	deltaLon := (destino.Longitude - origem.Longitude) * math.Pi / 180

	a := math.Sin(deltaLat/2)*math.Sin(deltaLat/2) +
		math.Cos(lat1)*math.Cos(lat2)*math.Sin(deltaLon/2)*math.Sin(deltaLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return raioTerraKm * c
}

// CalcularTaxaPorKm aplica a fórmula "taxa base + (distância × valor por km)".
func CalcularTaxaPorKm(distanciaKm, taxaBase, taxaPorKm float64) float64 {
	return taxaBase + (distanciaKm * taxaPorKm)
}

// Constantes da estimativa de frete fora da região da loja — uma
// aproximação inspirada nas faixas de peso do PAC/SEDEX, não uma
// integração real com os Correios (que exige contrato empresarial). São
// valores de partida, ajustáveis livremente conforme o custo real
// observado.
const (
	freteCorreiosBase300g   = 18.0
	freteCorreiosBase1kg    = 25.0
	freteCorreiosBase5kg    = 40.0
	freteCorreiosBase10kg   = 60.0
	freteCorreiosPorKgExtra = 6.0
	freteCorreiosPorKm      = 0.06
	freteCorreiosMinimo     = 20.0
)

// CalcularFreteEstimadoCorreios estima o frete de itens guardados quando o
// destino fica fora da cidade/estado da loja, combinando uma faixa de peso
// com uma tarifa por distância.
func CalcularFreteEstimadoCorreios(pesoGramas int, distanciaKm float64) float64 {
	var base float64
	switch {
	case pesoGramas <= 300:
		base = freteCorreiosBase300g
	case pesoGramas <= 1000:
		base = freteCorreiosBase1kg
	case pesoGramas <= 5000:
		base = freteCorreiosBase5kg
	case pesoGramas <= 10000:
		base = freteCorreiosBase10kg
	default:
		kgExtra := math.Ceil(float64(pesoGramas-10000) / 1000)
		base = freteCorreiosBase10kg + kgExtra*freteCorreiosPorKgExtra
	}

	total := base + distanciaKm*freteCorreiosPorKm
	if total < freteCorreiosMinimo {
		total = freteCorreiosMinimo
	}
	return total
}

// pesoPendenteEmItens diz se algum item mercadoria da lista está sem peso
// cadastrado — usado tanto pelo aviso preventivo do Pedido (modo
// "guardar") quanto pelo aviso definitivo da SolicitacaoEntrega, pra não
// duplicar esse critério em dois lugares.
func pesoPendenteEmItens(itens []domain.ItemPedido) bool {
	for _, item := range itens {
		if item.TipoProduto == domain.TipoProdutoMercadoria && item.PesoGramas <= 0 {
			return true
		}
	}
	return false
}

// nomesItensSemPeso lista os produtos (por nome, sem repetir) que estão
// sem peso cadastrado — usado pra deixar claro, no aviso de WhatsApp,
// exatamente qual produto o lojista precisa completar.
func nomesItensSemPeso(itens []domain.ItemPedido) []string {
	vistos := make(map[string]bool)
	var nomes []string
	for _, item := range itens {
		if item.TipoProduto == domain.TipoProdutoMercadoria && item.PesoGramas <= 0 && !vistos[item.ProdutoNome] {
			vistos[item.ProdutoNome] = true
			nomes = append(nomes, item.ProdutoNome)
		}
	}
	return nomes
}
