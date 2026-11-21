// Package usuario implementa a entidade e o repositório de usuários: contas
// do público (sem empresa) e motoristas/administradores vinculados a uma
// empresa (FR-033 da 003-rede-orbit).
package usuario

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/brunosrtr/radio-px-ktm/backend/internal/storage"
)

// ErrNaoEncontrado indica que nenhum usuário existe com o ID ou login
// informado.
var ErrNaoEncontrado = errors.New("usuario: não encontrado")

// Papéis de usuário. motorista e admin exigem empresa; os demais nunca têm
// (constraint usuario_papel_empresa_coerentes).
const (
	PapelAdmin     = "admin"
	PapelMotorista = "motorista"
	PapelUsuario   = "usuario"
	PapelAdminKTM  = "admin_ktm"
	PapelDemo      = "demo"
)

// Usuario é uma pessoa autenticada. EmpresaID, Login e Email usam "" para
// "não informado" (as colunas aceitam nulo no banco), para não espalhar
// ponteiros pelos pacotes que já usavam string.
type Usuario struct {
	ID        string
	EmpresaID string
	Nome      string
	Apelido   string
	Email     string
	Login     string
	SenhaHash string
	Papel     string
	Ativo     bool
	CriadoEm  time.Time
}

// TemEmpresa informa se o usuário está vinculado a uma empresa.
func (u Usuario) TemEmpresa() bool {
	return u.EmpresaID != ""
}

// NomeNoRadio é o nome mostrado aos demais participantes de um canal
// (FR-027): o apelido, ou o nome quando a conta ainda não tem apelido.
func (u Usuario) NomeNoRadio() string {
	if u.Apelido != "" {
		return u.Apelido
	}
	return u.Nome
}

// Repositorio persiste e consulta usuários.
type Repositorio struct {
	pool *storage.Pool
}

// NovoRepositorio cria um Repositorio sobre o pool de conexões informado.
func NovoRepositorio(pool *storage.Pool) *Repositorio {
	return &Repositorio{pool: pool}
}

const colunas = `id, coalesce(empresa_id::text, ''), nome, coalesce(apelido, ''), coalesce(email::text, ''),
	coalesce(login, ''), senha_hash, papel, ativo, criado_em`

func escanear(linha interface{ Scan(...any) error }, u *Usuario) error {
	return linha.Scan(&u.ID, &u.EmpresaID, &u.Nome, &u.Apelido, &u.Email, &u.Login, &u.SenhaHash, &u.Papel, &u.Ativo, &u.CriadoEm)
}

// ObterPorID busca um usuário ativo (não excluído) pelo ID. Um token emitido
// antes da exclusão da conta continua assinado até expirar, então quem
// depende do usuário (GET /me, WebSocket) recebe ErrNaoEncontrado.
func (r *Repositorio) ObterPorID(ctx context.Context, id string) (*Usuario, error) {
	var u Usuario
	err := escanear(r.pool.QueryRow(ctx, `
		select `+colunas+` from usuario
		where id = $1 and ativo and excluido_em is null
	`, id), &u)
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
		where login = $1 and ativo and excluido_em is null
	`, login), &u)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNaoEncontrado
	}
	if err != nil {
		return nil, fmt.Errorf("usuario: erro ao buscar usuário por login: %w", err)
	}
	return &u, nil
}

// BuscarPorEmail busca um usuário ativo pelo e-mail, sem diferenciar
// maiúsculas de minúsculas (coluna citext).
func (r *Repositorio) BuscarPorEmail(ctx context.Context, email string) (*Usuario, error) {
	var u Usuario
	err := escanear(r.pool.QueryRow(ctx, `
		select `+colunas+`
		from usuario
		where email = $1 and ativo and excluido_em is null
	`, email), &u)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNaoEncontrado
	}
	if err != nil {
		return nil, fmt.Errorf("usuario: erro ao buscar usuário por e-mail: %w", err)
	}
	return &u, nil
}

// BuscarPorIdentificador aceita o que o usuário digita no login: e-mail
// (contas do público) ou login/CPF (contas de empresa da 001).
func (r *Repositorio) BuscarPorIdentificador(ctx context.Context, identificador string) (*Usuario, error) {
	identificador = strings.TrimSpace(identificador)
	if strings.Contains(identificador, "@") {
		return r.BuscarPorEmail(ctx, identificador)
	}
	return r.BuscarPorLogin(ctx, NormalizarLogin(identificador))
}

// Criar insere um novo usuário. senhaHash já deve estar hasheada (bcrypt).
func (r *Repositorio) Criar(ctx context.Context, u Usuario) (*Usuario, error) {
	err := r.pool.QueryRow(ctx, `
		insert into usuario (empresa_id, nome, apelido, email, login, senha_hash, papel)
		values (nullif($1, '')::uuid, $2, nullif($3, ''), nullif($4, ''), nullif($5, ''), $6, $7)
		returning id, ativo, criado_em
	`, u.EmpresaID, u.Nome, u.Apelido, u.Email, u.Login, u.SenhaHash, u.Papel).Scan(&u.ID, &u.Ativo, &u.CriadoEm)
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

// PushTokens retorna os push_token não nulos de todos os dispositivos
// registrados do usuário (um usuário pode ter Android e iOS ao mesmo
// tempo), usados para avisos como remoção por geocerca (FR-017).
func (r *Repositorio) PushTokens(ctx context.Context, usuarioID string) ([]string, error) {
	linhas, err := r.pool.Query(ctx, `
		select push_token from dispositivo
		where usuario_id = $1 and push_token is not null
	`, usuarioID)
	if err != nil {
		return nil, fmt.Errorf("usuario: erro ao buscar push tokens: %w", err)
	}
	defer linhas.Close()

	var tokens []string
	for linhas.Next() {
		var token string
		if err := linhas.Scan(&token); err != nil {
			return nil, fmt.Errorf("usuario: erro ao ler push token: %w", err)
		}
		tokens = append(tokens, token)
	}
	if err := linhas.Err(); err != nil {
		return nil, fmt.Errorf("usuario: erro ao percorrer push tokens: %w", err)
	}
	return tokens, nil
}
