// Package posicao implementa a entidade e o repositório de localização dos
// motoristas. Nesta fase cobre apenas a leitura de posicao_atual, necessária
// para a checagem de geocerca (US3); ingestão em lote e histórico
// (posicao) chegam na User Story 4.
package posicao

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/brunosrtr/radio-px-ktm/backend/internal/storage"
)

// ErrNaoEncontrada indica que o usuário ainda não teve nenhuma posição
// registrada.
var ErrNaoEncontrada = errors.New("posicao: nenhuma posição registrada para o usuário")

// Atual é a última posição conhecida de um motorista (data-model.md).
type Atual struct {
	UsuarioID     string
	Latitude      float64
	Longitude     float64
	VelocidadeKmh *float64
	CapturadoEm   time.Time
	AtualizadoEm  time.Time
}

// Repositorio consulta e persiste posições.
type Repositorio struct {
	pool *storage.Pool
}

// NovoRepositorio cria um Repositorio sobre o pool de conexões informado.
func NovoRepositorio(pool *storage.Pool) *Repositorio {
	return &Repositorio{pool: pool}
}

// ObterAtual busca a última posição conhecida de um usuário.
func (r *Repositorio) ObterAtual(ctx context.Context, usuarioID string) (*Atual, error) {
	var p Atual
	err := r.pool.QueryRow(ctx, `
		select usuario_id, latitude, longitude, velocidade_kmh, capturado_em, atualizado_em
		from posicao_atual
		where usuario_id = $1
	`, usuarioID).Scan(&p.UsuarioID, &p.Latitude, &p.Longitude, &p.VelocidadeKmh, &p.CapturadoEm, &p.AtualizadoEm)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNaoEncontrada
	}
	if err != nil {
		return nil, fmt.Errorf("posicao: erro ao buscar posição atual: %w", err)
	}
	return &p, nil
}
