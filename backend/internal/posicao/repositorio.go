package posicao

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/brunosrtr/radio-px-ktm/backend/internal/storage"
)

// Repositorio persiste e consulta posições.
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

// InserirLote grava todos os pontos do lote no histórico (posicao) numa
// única operação e atualiza posicao_atual com o ponto de capturado_em mais
// recente do lote — só se for mais novo que o já gravado, para que um lote
// atrasado nunca regrida a posição exibida (research.md §8).
func (r *Repositorio) InserirLote(ctx context.Context, usuarioID string, pontos []Ponto) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("posicao: erro ao iniciar transação: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	linhas := make([][]any, len(pontos))
	maisRecente := pontos[0]
	for i, p := range pontos {
		linhas[i] = []any{usuarioID, p.Latitude, p.Longitude, p.PrecisaoMetros, p.VelocidadeKmh, p.CapturadoEm}
		if p.CapturadoEm.After(maisRecente.CapturadoEm) {
			maisRecente = p
		}
	}

	if _, err := tx.CopyFrom(ctx,
		pgx.Identifier{"posicao"},
		[]string{"usuario_id", "latitude", "longitude", "precisao_metros", "velocidade_kmh", "capturado_em"},
		pgx.CopyFromRows(linhas),
	); err != nil {
		return fmt.Errorf("posicao: erro ao inserir lote: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		insert into posicao_atual (usuario_id, latitude, longitude, velocidade_kmh, capturado_em)
		values ($1, $2, $3, $4, $5)
		on conflict (usuario_id) do update set
			latitude = excluded.latitude,
			longitude = excluded.longitude,
			velocidade_kmh = excluded.velocidade_kmh,
			capturado_em = excluded.capturado_em,
			atualizado_em = now()
		where excluded.capturado_em > posicao_atual.capturado_em
	`, usuarioID, maisRecente.Latitude, maisRecente.Longitude, maisRecente.VelocidadeKmh, maisRecente.CapturadoEm); err != nil {
		return fmt.Errorf("posicao: erro ao atualizar posição atual: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("posicao: erro ao confirmar lote: %w", err)
	}
	return nil
}
