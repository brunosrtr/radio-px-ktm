// Package orbit implementa a Rede Orbit (003-rede-orbit): ingestão das
// leituras feitas pelos pontos de transmissão, última posição de cada Orbit,
// pontos fixos de leitura e o encaminhamento para a KTM.
//
// Princípio VII: nada neste pacote grava ou publica quem fez a leitura. O
// celular é identificado só por um ponto_id opaco e renovado a cada 24 h.
package orbit

import (
	"errors"
	"fmt"
	"math"
	"regexp"
	"time"
)

// Origens possíveis de uma leitura.
const (
	OrigemCelular = "celular"
	OrigemFixo    = "fixo"
)

// Limites do lote (contracts/rest-api.md, FR-004/FR-005).
const (
	MaximoLeiturasPorLote = 100
	ToleranciaFuturo      = 2 * time.Minute
	IdadeMaximaLeitura    = 7 * 24 * time.Hour
)

var (
	// ErrLoteInvalido indica um lote fora do contrato.
	ErrLoteInvalido = errors.New("orbit: lote inválido")
	// ErrNaoEncontrado indica Orbit ou ponto fixo inexistente.
	ErrNaoEncontrado = errors.New("orbit: não encontrado")
	// ErrDadosInvalidos indica um ponto fixo fora das regras.
	ErrDadosInvalidos = errors.New("orbit: dados inválidos")

	formatoOrbitID = regexp.MustCompile(`^ORBIT-[0-9A-F]{6}$`)
)

// LeituraRecebida é uma leitura como o app envia em POST /leituras.
type LeituraRecebida struct {
	OrbitID        string    `json:"orbit_id"`
	Latitude       float64   `json:"latitude"`
	Longitude      float64   `json:"longitude"`
	PrecisaoMetros *float64  `json:"precisao_metros"`
	RSSI           *int      `json:"rssi"`
	DetectadoEm    time.Time `json:"detectado_em"`
}

// LeituraRegistrada é uma leitura gravada — o que vai para o painel e para a
// integração com a KTM. Não tem nenhum dado do usuário.
type LeituraRegistrada struct {
	ID             int64     `json:"id"`
	OrbitID        string    `json:"orbit_id"`
	Origem         string    `json:"origem"`
	PontoID        string    `json:"ponto_id"`
	Latitude       float64   `json:"latitude"`
	Longitude      float64   `json:"longitude"`
	PrecisaoMetros *float64  `json:"precisao_metros,omitempty"`
	RSSI           *int      `json:"rssi,omitempty"`
	DetectadoEm    time.Time `json:"detectado_em"`
	RecebidoEm     time.Time `json:"recebido_em"`
}

// OrbitResumo é a última posição conhecida de um Orbit, como o painel mostra.
type OrbitResumo struct {
	ID                 string     `json:"id"`
	Virtual            bool       `json:"virtual"`
	Rotulo             *string    `json:"rotulo"`
	Latitude           *float64   `json:"latitude"`
	Longitude          *float64   `json:"longitude"`
	UltimaLeituraEm    *time.Time `json:"ultima_leitura_em"`
	Leituras24h        int        `json:"leituras_24h"`
	PontosDistintos24h int        `json:"pontos_distintos_24h"`
}

// PontoFixo é um ponto de leitura existente da KTM (simulado na demo).
type PontoFixo struct {
	ID         string  `json:"id"`
	Nome       string  `json:"nome"`
	Latitude   float64 `json:"latitude"`
	Longitude  float64 `json:"longitude"`
	RaioMetros int     `json:"raio_metros"`
	Simulado   bool    `json:"simulado"`
}

// ResultadoIngestao resume um lote gravado.
type ResultadoIngestao struct {
	Aceitas    int
	Duplicadas int
	// Novas são as leituras efetivamente inseridas (sem as duplicadas).
	Novas []LeituraRegistrada
	// OrbitsAtualizados são os Orbits cuja última posição mudou.
	OrbitsAtualizados []OrbitResumo
}

// OrbitIDValido informa se o ID segue o formato `ORBIT-` + 6 hexadecimais.
func OrbitIDValido(id string) bool {
	return formatoOrbitID.MatchString(id)
}

func coordenadasValidas(lat, lng float64) bool {
	return !math.IsNaN(lat) && !math.IsNaN(lng) && lat >= -90 && lat <= 90 && lng >= -180 && lng <= 180 && !(lat == 0 && lng == 0)
}

// ValidarLote confere um lote de leituras antes da gravação.
func ValidarLote(leituras []LeituraRecebida, agora time.Time) error {
	if len(leituras) == 0 || len(leituras) > MaximoLeiturasPorLote {
		return fmt.Errorf("%w: o lote deve ter de 1 a %d leituras", ErrLoteInvalido, MaximoLeiturasPorLote)
	}
	for i, l := range leituras {
		switch {
		case !OrbitIDValido(l.OrbitID):
			return fmt.Errorf("%w: leituras[%d].orbit_id fora do formato ORBIT-XXXXXX", ErrLoteInvalido, i)
		case !coordenadasValidas(l.Latitude, l.Longitude):
			return fmt.Errorf("%w: leituras[%d] com coordenadas inválidas", ErrLoteInvalido, i)
		case l.DetectadoEm.IsZero():
			return fmt.Errorf("%w: leituras[%d].detectado_em é obrigatório", ErrLoteInvalido, i)
		case l.DetectadoEm.After(agora.Add(ToleranciaFuturo)):
			return fmt.Errorf("%w: leituras[%d].detectado_em está no futuro", ErrLoteInvalido, i)
		case l.DetectadoEm.Before(agora.Add(-IdadeMaximaLeitura)):
			return fmt.Errorf("%w: leituras[%d].detectado_em tem mais de 7 dias", ErrLoteInvalido, i)
		case l.PrecisaoMetros != nil && (*l.PrecisaoMetros < 0 || *l.PrecisaoMetros > 9999):
			return fmt.Errorf("%w: leituras[%d].precisao_metros inválida", ErrLoteInvalido, i)
		case l.RSSI != nil && (*l.RSSI < -127 || *l.RSSI > 20):
			return fmt.Errorf("%w: leituras[%d].rssi inválido", ErrLoteInvalido, i)
		}
	}
	return nil
}

// ValidarPontoFixo confere os dados de um ponto fixo.
func ValidarPontoFixo(p PontoFixo) error {
	switch {
	case p.Nome == "" || len(p.Nome) > 80:
		return fmt.Errorf("%w: informe um nome de até 80 caracteres", ErrDadosInvalidos)
	case !coordenadasValidas(p.Latitude, p.Longitude):
		return fmt.Errorf("%w: coordenadas inválidas", ErrDadosInvalidos)
	case p.RaioMetros < 50 || p.RaioMetros > 5000:
		return fmt.Errorf("%w: o raio deve ficar entre 50 e 5000 metros", ErrDadosInvalidos)
	}
	return nil
}
