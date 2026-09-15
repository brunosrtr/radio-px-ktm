// Package empresa implementa a entidade e o repositório de empresas
// clientes do serviço.
package empresa

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/brunosrtr/radio-px-ktm/backend/internal/storage"
)

// ErrNaoEncontrada indica que nenhuma empresa existe com o ID informado.
var ErrNaoEncontrada = errors.New("empresa: não encontrada")

// Empresa é a organização cliente do serviço.
type Empresa struct {
	ID          string
	RazaoSocial string
	CNPJ        string
	Ativa       bool
	CriadoEm    time.Time
}

// Repositorio persiste e consulta empresas.
type Repositorio struct {
	pool *storage.Pool
}

// NovoRepositorio cria um Repositorio sobre o pool de conexões informado.
func NovoRepositorio(pool *storage.Pool) *Repositorio {
	return &Repositorio{pool: pool}
}

// ObterPorID busca uma empresa pelo ID, retornando ErrNaoEncontrada se não
// existir.
func (r *Repositorio) ObterPorID(ctx context.Context, id string) (*Empresa, error) {
	var e Empresa
	err := r.pool.QueryRow(ctx, `
		select id, razao_social, cnpj, ativa, criado_em
		from empresa
		where id = $1
	`, id).Scan(&e.ID, &e.RazaoSocial, &e.CNPJ, &e.Ativa, &e.CriadoEm)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNaoEncontrada
	}
	if err != nil {
		return nil, fmt.Errorf("empresa: erro ao buscar empresa %s: %w", id, err)
	}
	return &e, nil
}

// Criar insere uma nova empresa.
func (r *Repositorio) Criar(ctx context.Context, razaoSocial, cnpj string) (*Empresa, error) {
	e := Empresa{RazaoSocial: razaoSocial, CNPJ: cnpj}
	err := r.pool.QueryRow(ctx, `
		insert into empresa (razao_social, cnpj)
		values ($1, $2)
		returning id, ativa, criado_em
	`, razaoSocial, cnpj).Scan(&e.ID, &e.Ativa, &e.CriadoEm)
	if err != nil {
		return nil, fmt.Errorf("empresa: erro ao criar empresa: %w", err)
	}
	return &e, nil
}
