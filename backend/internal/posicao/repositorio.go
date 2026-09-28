package posicao

import (
	"context"
	"errors"
	"fmt"
	"time"

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

// PosicoesAtuaisPorEmpresa retorna a última posição conhecida de cada
// motorista ativo da empresa (FR-026) — nunca consulta o histórico
// (posicao) para montar o mapa, só posicao_atual.
func (r *Repositorio) PosicoesAtuaisPorEmpresa(ctx context.Context, empresaID string) ([]MotoristaPosicaoAtual, error) {
	linhas, err := r.pool.Query(ctx, `
		select u.id, u.nome, pa.latitude, pa.longitude, pa.velocidade_kmh, pa.capturado_em
		from posicao_atual pa
		join usuario u on u.id = pa.usuario_id
		where u.empresa_id = $1 and u.ativo
		order by u.nome
	`, empresaID)
	if err != nil {
		return nil, fmt.Errorf("posicao: erro ao consultar posições atuais da empresa: %w", err)
	}
	defer linhas.Close()

	var motoristas []MotoristaPosicaoAtual
	for linhas.Next() {
		var m MotoristaPosicaoAtual
		if err := linhas.Scan(&m.UsuarioID, &m.Nome, &m.Latitude, &m.Longitude, &m.VelocidadeKmh, &m.CapturadoEm); err != nil {
			return nil, fmt.Errorf("posicao: erro ao ler posição atual: %w", err)
		}
		motoristas = append(motoristas, m)
	}
	if err := linhas.Err(); err != nil {
		return nil, fmt.Errorf("posicao: erro ao percorrer posições atuais: %w", err)
	}
	return motoristas, nil
}

// AtualComEmpresa busca a posição atual de um usuário junto com seu nome e
// empresa — usado para montar o evento de broadcast em tempo real após
// IngerirLote, sem exigir que o chamador já tenha esses dados em mãos.
func (r *Repositorio) AtualComEmpresa(ctx context.Context, usuarioID string) (empresaID string, atual MotoristaPosicaoAtual, err error) {
	err = r.pool.QueryRow(ctx, `
		select u.empresa_id, u.id, u.nome, pa.latitude, pa.longitude, pa.velocidade_kmh, pa.capturado_em
		from posicao_atual pa
		join usuario u on u.id = pa.usuario_id
		where pa.usuario_id = $1
	`, usuarioID).Scan(&empresaID, &atual.UsuarioID, &atual.Nome, &atual.Latitude, &atual.Longitude, &atual.VelocidadeKmh, &atual.CapturadoEm)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", MotoristaPosicaoAtual{}, ErrNaoEncontrada
	}
	if err != nil {
		return "", MotoristaPosicaoAtual{}, fmt.Errorf("posicao: erro ao buscar posição atual com empresa: %w", err)
	}
	return empresaID, atual, nil
}

// MotoristaPertenceAEmpresa indica se o usuário informado pertence à
// empresa informada — usado para restringir a consulta de trajeto.
func (r *Repositorio) MotoristaPertenceAEmpresa(ctx context.Context, usuarioID, empresaID string) (bool, error) {
	var pertence bool
	err := r.pool.QueryRow(ctx, `
		select exists(select 1 from usuario where id = $1 and empresa_id = $2)
	`, usuarioID, empresaID).Scan(&pertence)
	if err != nil {
		return false, fmt.Errorf("posicao: erro ao verificar empresa do motorista: %w", err)
	}
	return pertence, nil
}

// Trajeto retorna o histórico de posições de um usuário no período
// informado (inclusive em ambas as pontas), do mais antigo para o mais
// recente (FR-027).
func (r *Repositorio) Trajeto(ctx context.Context, usuarioID string, de, ate time.Time) ([]PontoTrajeto, error) {
	linhas, err := r.pool.Query(ctx, `
		select latitude, longitude, capturado_em
		from posicao
		where usuario_id = $1 and capturado_em between $2 and $3
		order by capturado_em asc
	`, usuarioID, de, ate)
	if err != nil {
		return nil, fmt.Errorf("posicao: erro ao consultar trajeto: %w", err)
	}
	defer linhas.Close()

	pontos := make([]PontoTrajeto, 0)
	for linhas.Next() {
		var p PontoTrajeto
		if err := linhas.Scan(&p.Latitude, &p.Longitude, &p.CapturadoEm); err != nil {
			return nil, fmt.Errorf("posicao: erro ao ler ponto de trajeto: %w", err)
		}
		pontos = append(pontos, p)
	}
	if err := linhas.Err(); err != nil {
		return nil, fmt.Errorf("posicao: erro ao percorrer trajeto: %w", err)
	}
	return pontos, nil
}
