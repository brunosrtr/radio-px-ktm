package usuario

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"unicode/utf8"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/brunosrtr/radio-px-ktm/backend/internal/auth"
)

// VersaoTermosAtual é a versão dos termos de uso e da política de
// privacidade servidos em /termos e /privacidade. O cadastro só aceita o
// aceite da versão vigente.
const VersaoTermosAtual = "2026-11"

// Limites do cadastro público (contracts/rest-api.md).
const (
	TamanhoMinimoSenha   = 8
	TamanhoMinimoApelido = 2
	TamanhoMaximoApelido = 24
	tamanhoMaximoNome    = 80
	tamanhoMaximoSenha   = 72 // bcrypt ignora o que passa de 72 bytes
)

var (
	// ErrEmailEmUso indica que já existe uma conta com o e-mail informado.
	ErrEmailEmUso = errors.New("usuario: e-mail já cadastrado")
	// ErrDadosInvalidos indica um campo de cadastro ou perfil fora das regras.
	ErrDadosInvalidos = errors.New("usuario: dados inválidos")
)

// DadosCadastroPublico é o que uma pessoa informa para criar a própria conta
// pelo aplicativo (FR-023).
type DadosCadastroPublico struct {
	Nome                string `json:"nome"`
	Apelido             string `json:"apelido"`
	Email               string `json:"email"`
	Senha               string `json:"senha"`
	AceitouTermosVersao string `json:"aceitou_termos_versao"`
}

// NormalizarEmail remove espaços e valida o formato do e-mail, devolvendo ""
// quando ele é inválido. A comparação sem diferenciar maiúsculas fica com a
// coluna citext.
func NormalizarEmail(valor string) string {
	valor = strings.TrimSpace(valor)
	endereco, err := mail.ParseAddress(valor)
	if err != nil || endereco.Address != valor || !strings.Contains(valor[strings.LastIndex(valor, "@"):], ".") {
		return ""
	}
	return valor
}

// ValidarApelido aplica as regras do apelido de rádio (FR-027).
func ValidarApelido(apelido string) (string, error) {
	apelido = strings.Join(strings.Fields(apelido), " ")
	n := utf8.RuneCountInString(apelido)
	if n < TamanhoMinimoApelido || n > TamanhoMaximoApelido {
		return "", fmt.Errorf("%w: o apelido deve ter de %d a %d caracteres", ErrDadosInvalidos, TamanhoMinimoApelido, TamanhoMaximoApelido)
	}
	return apelido, nil
}

// ValidarNome aplica as regras do nome completo.
func ValidarNome(nome string) (string, error) {
	nome = strings.Join(strings.Fields(nome), " ")
	if nome == "" || utf8.RuneCountInString(nome) > tamanhoMaximoNome {
		return "", fmt.Errorf("%w: informe seu nome (até %d caracteres)", ErrDadosInvalidos, tamanhoMaximoNome)
	}
	return nome, nil
}

// ValidarSenha aplica as regras de senha das contas do público.
func ValidarSenha(senha string) error {
	if utf8.RuneCountInString(senha) < TamanhoMinimoSenha || len(senha) > tamanhoMaximoSenha {
		return fmt.Errorf("%w: a senha deve ter de %d a %d caracteres", ErrDadosInvalidos, TamanhoMinimoSenha, tamanhoMaximoSenha)
	}
	return nil
}

// ValidarCadastroPublico normaliza e valida os dados do cadastro.
func ValidarCadastroPublico(d DadosCadastroPublico) (DadosCadastroPublico, error) {
	var err error
	if d.Nome, err = ValidarNome(d.Nome); err != nil {
		return d, err
	}
	if d.Apelido, err = ValidarApelido(d.Apelido); err != nil {
		return d, err
	}
	if d.Email = NormalizarEmail(d.Email); d.Email == "" {
		return d, fmt.Errorf("%w: e-mail inválido", ErrDadosInvalidos)
	}
	if err = ValidarSenha(d.Senha); err != nil {
		return d, err
	}
	if d.AceitouTermosVersao != VersaoTermosAtual {
		return d, fmt.Errorf("%w: é preciso aceitar os termos de uso e a política de privacidade", ErrDadosInvalidos)
	}
	return d, nil
}

// CriarUsuarioPublico cria uma conta do público (papel "usuario", sem
// empresa) e registra, na mesma transação, o aceite dos termos (FR-023,
// FR-028).
func (r *Repositorio) CriarUsuarioPublico(ctx context.Context, d DadosCadastroPublico) (*Usuario, error) {
	d, err := ValidarCadastroPublico(d)
	if err != nil {
		return nil, err
	}
	hash, err := auth.HashSenha(d.Senha)
	if err != nil {
		return nil, err
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("usuario: erro ao iniciar cadastro: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	u := Usuario{Nome: d.Nome, Apelido: d.Apelido, Email: d.Email, SenhaHash: hash, Papel: PapelUsuario}
	err = tx.QueryRow(ctx, `
		insert into usuario (nome, apelido, email, senha_hash, papel)
		values ($1, $2, $3, $4, $5)
		returning id, ativo, criado_em
	`, u.Nome, u.Apelido, u.Email, u.SenhaHash, u.Papel).Scan(&u.ID, &u.Ativo, &u.CriadoEm)
	if err != nil {
		var pg *pgconn.PgError
		if errors.As(err, &pg) && pg.Code == "23505" {
			return nil, ErrEmailEmUso
		}
		return nil, fmt.Errorf("usuario: erro ao criar conta: %w", err)
	}

	if _, err = tx.Exec(ctx, `
		insert into consentimento (usuario_id, tipo, versao) values ($1, $2, $3)
	`, u.ID, ConsentimentoTermos, d.AceitouTermosVersao); err != nil {
		return nil, fmt.Errorf("usuario: erro ao registrar aceite dos termos: %w", err)
	}

	if err = tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("usuario: erro ao confirmar cadastro: %w", err)
	}
	return &u, nil
}

// AtualizarPerfil altera nome e/ou apelido (PATCH /me). Campos vazios ficam
// como estão.
func (r *Repositorio) AtualizarPerfil(ctx context.Context, usuarioID, nome, apelido string) (*Usuario, error) {
	var err error
	if nome != "" {
		if nome, err = ValidarNome(nome); err != nil {
			return nil, err
		}
	}
	if apelido != "" {
		if apelido, err = ValidarApelido(apelido); err != nil {
			return nil, err
		}
	}
	tag, err := r.pool.Exec(ctx, `
		update usuario
		set nome = coalesce(nullif($2, ''), nome),
		    apelido = coalesce(nullif($3, ''), apelido)
		where id = $1 and ativo and excluido_em is null
	`, usuarioID, nome, apelido)
	if err != nil {
		return nil, fmt.Errorf("usuario: erro ao atualizar perfil: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return nil, ErrNaoEncontrado
	}
	return r.ObterPorID(ctx, usuarioID)
}
