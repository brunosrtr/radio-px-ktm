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
