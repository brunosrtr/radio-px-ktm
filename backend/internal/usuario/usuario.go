// Package usuario implementa a entidade e o repositório de usuários
// (motoristas e administradores) vinculados a uma empresa.
package usuario

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/brunosrtr/radio-px-ktm/backend/internal/storage"
)

// ErrNaoEncontrado indica que nenhum usuário existe com o ID ou login
// informado.
var ErrNaoEncontrado = errors.New("usuario: não encontrado")

// Papel do usuário dentro da sua empresa (FR-020, RF21).
const (
	PapelAdmin     = "admin"
	PapelMotorista = "motorista"
)

// Usuario é uma pessoa autenticada, vinculada a uma empresa.
type Usuario struct {
	ID        string
	EmpresaID string
	Nome      string
	Login     string
	SenhaHash string
	Papel     string
	Ativo     bool
	CriadoEm  time.Time
}

// Repositorio persiste e consulta usuários.
type Repositorio struct {
	pool *storage.Pool
}

// NovoRepositorio cria um Repositorio sobre o pool de conexões informado.
func NovoRepositorio(pool *storage.Pool) *Repositorio {
	return &Repositorio{pool: pool}
}

const colunas = `id, empresa_id, nome, login, senha_hash, papel, ativo, criado_em`

func escanear(linha interface{ Scan(...any) error }, u *Usuario) error {
	return linha.Scan(&u.ID, &u.EmpresaID, &u.Nome, &u.Login, &u.SenhaHash, &u.Papel, &u.Ativo, &u.CriadoEm)
}

// ObterPorID busca um usuário pelo ID.
func (r *Repositorio) ObterPorID(ctx context.Context, id string) (*Usuario, error) {
	var u Usuario
	err := escanear(r.pool.QueryRow(ctx, `select `+colunas+` from usuario where id = $1`, id), &u)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNaoEncontrado
	}
	if err != nil {
		return nil, fmt.Errorf("usuario: erro ao buscar usuário %s: %w", id, err)
	}
	return &u, nil
}

// BuscarPorLogin busca um usuário ativo pelo login, usado na autenticação.
func (r *Repositorio) BuscarPorLogin(ctx context.Context, login string) (*Usuario, error) {
	var u Usuario
	err := escanear(r.pool.QueryRow(ctx, `
		select `+colunas+`
		from usuario
		where login = $1 and ativo
	`, login), &u)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNaoEncontrado
	}
	if err != nil {
		return nil, fmt.Errorf("usuario: erro ao buscar usuário por login: %w", err)
	}
	return &u, nil
}

// Criar insere um novo usuário. senhaHash já deve estar hasheada (bcrypt).
func (r *Repositorio) Criar(ctx context.Context, u Usuario) (*Usuario, error) {
	err := r.pool.QueryRow(ctx, `
		insert into usuario (empresa_id, nome, login, senha_hash, papel)
		values ($1, $2, $3, $4, $5)
		returning id, ativo, criado_em
	`, u.EmpresaID, u.Nome, u.Login, u.SenhaHash, u.Papel).Scan(&u.ID, &u.Ativo, &u.CriadoEm)
	if err != nil {
		return nil, fmt.Errorf("usuario: erro ao criar usuário: %w", err)
	}
	return &u, nil
}

// RegistrarDispositivo grava (ou atualiza, se já existir um dispositivo do
// usuário na mesma plataforma) o dispositivo móvel usado no login —
// push_token é usado para avisos de remoção por geocerca (FR-017).
func (r *Repositorio) RegistrarDispositivo(ctx context.Context, usuarioID, plataforma, versaoApp string, pushToken *string) error {
	_, err := r.pool.Exec(ctx, `
		insert into dispositivo (usuario_id, plataforma, push_token, versao_app)
		values ($1, $2, $3, nullif($4, ''))
		on conflict (usuario_id, plataforma) do update set
			push_token = excluded.push_token,
			versao_app = excluded.versao_app,
			visto_em = now()
	`, usuarioID, plataforma, pushToken, versaoApp)
	if err != nil {
		return fmt.Errorf("usuario: erro ao registrar dispositivo: %w", err)
	}
	return nil
}
