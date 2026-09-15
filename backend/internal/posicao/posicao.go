// Package posicao implementa a coleta e consulta de localização dos
// motoristas: ingestão em lote do histórico bruto (posicao) e atualização
// condicional da última posição conhecida (posicao_atual), usada tanto pela
// checagem de geocerca (US3) quanto pelo painel da empresa (US5).
package posicao

import (
	"errors"
	"time"
)

// ErrNaoEncontrada indica que o usuário ainda não teve nenhuma posição
// registrada.
var ErrNaoEncontrada = errors.New("posicao: nenhuma posição registrada para o usuário")

// ErrLoteInvalido indica que o lote de posições enviado é vazio ou tem um
// ponto sem capturado_em (contracts/rest-api.md).
var ErrLoteInvalido = errors.New("posicao: lote inválido")

// ErrSemPermissao indica uma tentativa de consultar posições de outra
// empresa, ou de um motorista que não pertence à empresa do admin (FR-026).
var ErrSemPermissao = errors.New("posicao: sem permissão")

// Ponto é uma posição bruta recebida do app (FR-024, FR-028).
type Ponto struct {
	Latitude       float64
	Longitude      float64
	PrecisaoMetros *float64
	VelocidadeKmh  *float64
	CapturadoEm    time.Time
}

// Atual é a última posição conhecida de um motorista (data-model.md).
type Atual struct {
	UsuarioID     string
	Latitude      float64
	Longitude     float64
	VelocidadeKmh *float64
	CapturadoEm   time.Time
	AtualizadoEm  time.Time
}

// MotoristaPosicaoAtual é a posição atual de um motorista, já com o nome
// para exibição no mapa do painel (FR-026).
type MotoristaPosicaoAtual struct {
	UsuarioID     string
	Nome          string
	Latitude      float64
	Longitude     float64
	VelocidadeKmh *float64
	CapturadoEm   time.Time
}

// PontoTrajeto é um ponto do histórico de um motorista num período (FR-027).
type PontoTrajeto struct {
	Latitude    float64
	Longitude   float64
	CapturadoEm time.Time
}
