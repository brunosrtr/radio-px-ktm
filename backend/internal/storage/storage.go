// Package storage gerencia o pool de conexão com o Postgres e a aplicação
// idempotente das migrações versionadas em backend/migrations (Princípio V
// da constituição: SQL escrito à mão, sem ORM).
package storage

import (
	"context"
	"fmt"
	"io/fs"
	"sort"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/brunosrtr/radio-px-ktm/backend/migrations"
)

// Pool encapsula o pool de conexões pgx usado pelo restante do backend.
type Pool struct {
	*pgxpool.Pool
}

// Conectar abre o pool de conexões com o Postgres e confirma a conectividade
// com um ping antes de retornar.
func Conectar(ctx context.Context, databaseURL string) (*Pool, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("storage: erro ao criar pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("storage: erro ao conectar ao Postgres: %w", err)
	}

	return &Pool{Pool: pool}, nil
}

// AplicarMigracoes executa, em ordem, os arquivos .sql embutidos em
// backend/migrations que ainda não constam em schema_migrations. Cada
// migração roda dentro de uma transação; reexecutar sobre um banco já
// migrado não falha nem duplica estado.
func (p *Pool) AplicarMigracoes(ctx context.Context) error {
	if _, err := p.Exec(ctx, `
		create table if not exists schema_migrations (
			versao text primary key,
			aplicada_em timestamptz not null default now()
		)
	`); err != nil {
		return fmt.Errorf("storage: erro ao criar schema_migrations: %w", err)
	}

	entradas, err := fs.ReadDir(migrations.Arquivos, ".")
	if err != nil {
		return fmt.Errorf("storage: erro ao ler migrações embutidas: %w", err)
	}

	var versoes []string
	for _, entrada := range entradas {
		if !entrada.IsDir() {
			versoes = append(versoes, entrada.Name())
		}
	}
	sort.Strings(versoes)

	for _, versao := range versoes {
		var jaAplicada bool
		if err := p.QueryRow(ctx,
			`select exists(select 1 from schema_migrations where versao = $1)`,
			versao,
		).Scan(&jaAplicada); err != nil {
			return fmt.Errorf("storage: erro ao verificar migração %s: %w", versao, err)
		}
		if jaAplicada {
			continue
		}

		conteudo, err := migrations.Arquivos.ReadFile(versao)
		if err != nil {
			return fmt.Errorf("storage: erro ao ler migração %s: %w", versao, err)
		}

		tx, err := p.Begin(ctx)
		if err != nil {
			return fmt.Errorf("storage: erro ao iniciar transação da migração %s: %w", versao, err)
		}

		if _, err := tx.Exec(ctx, string(conteudo)); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("storage: erro ao aplicar migração %s: %w", versao, err)
		}

		if _, err := tx.Exec(ctx,
			`insert into schema_migrations (versao) values ($1)`,
			versao,
		); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("storage: erro ao registrar migração %s: %w", versao, err)
		}

		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("storage: erro ao confirmar migração %s: %w", versao, err)
		}
	}

	return nil
}
