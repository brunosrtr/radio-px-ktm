package usuario

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/brunosrtr/radio-px-ktm/backend/internal/auth"
	"github.com/brunosrtr/radio-px-ktm/backend/internal/email"
)

// Regras do código de recuperação de senha (FR-025, research.md §7).
const (
	ValidadeCodigoRecuperacao = 15 * time.Minute
	MaximoTentativasCodigo    = 5
)

// ErrCodigoInvalido cobre código errado, expirado, já usado ou com
// tentativas esgotadas — de propósito sem distinguir os casos.
var ErrCodigoInvalido = errors.New("usuario: código de recuperação inválido")

func gerarCodigo() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1_000_000))
	if err != nil {
		return "", fmt.Errorf("usuario: erro ao gerar código: %w", err)
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}

// SolicitarRecuperacao gera um código de 6 dígitos para a conta do e-mail
// informado, invalida os códigos anteriores e envia o novo por e-mail. Para
// um e-mail sem conta, não faz nada e não devolve erro: a resposta da API é
// sempre a mesma, para não revelar quais e-mails estão cadastrados.
func (r *Repositorio) SolicitarRecuperacao(ctx context.Context, emailInformado string, enviador email.Enviador) error {
	endereco := NormalizarEmail(emailInformado)
	if endereco == "" {
		return nil
	}
	u, err := r.BuscarPorEmail(ctx, endereco)
	if errors.Is(err, ErrNaoEncontrado) {
		return nil
	}
	if err != nil {
		return err
	}

	codigo, err := gerarCodigo()
	if err != nil {
		return err
	}
	hash, err := auth.HashSenha(codigo)
	if err != nil {
		return err
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("usuario: erro ao iniciar recuperação: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err = tx.Exec(ctx, `
		update recuperacao_senha set expira_em = now()
		where usuario_id = $1 and usado_em is null and expira_em > now()
	`, u.ID); err != nil {
		return fmt.Errorf("usuario: erro ao invalidar códigos anteriores: %w", err)
	}
	if _, err = tx.Exec(ctx, `
		insert into recuperacao_senha (usuario_id, codigo_hash, expira_em)
		values ($1, $2, now() + $3::interval)
	`, u.ID, hash, fmt.Sprintf("%d seconds", int(ValidadeCodigoRecuperacao.Seconds()))); err != nil {
		return fmt.Errorf("usuario: erro ao gravar código: %w", err)
	}
	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("usuario: erro ao confirmar recuperação: %w", err)
	}

	corpo := fmt.Sprintf(
		"Olá, %s!\n\nSeu código para redefinir a senha da Rádio PX KTM é:\n\n    %s\n\n"+
			"Ele vale por %d minutos. Se você não pediu a troca de senha, ignore este e-mail.\n",
		u.NomeNoRadio(), codigo, int(ValidadeCodigoRecuperacao.Minutes()),
	)
	return enviador.Enviar(ctx, u.Email, "Código para redefinir sua senha", corpo)
}

// RedefinirSenha troca a senha da conta do e-mail se o código for o último
// gerado, estiver dentro da validade, não tiver sido usado e ainda houver
// tentativas. Cada código errado consome uma tentativa.
func (r *Repositorio) RedefinirSenha(ctx context.Context, emailInformado, codigo, novaSenha string) error {
	if err := ValidarSenha(novaSenha); err != nil {
		return err
	}
	endereco := NormalizarEmail(emailInformado)
	if endereco == "" || len(codigo) != 6 {
		return ErrCodigoInvalido
	}
	u, err := r.BuscarPorEmail(ctx, endereco)
	if errors.Is(err, ErrNaoEncontrado) {
		return ErrCodigoInvalido
	}
	if err != nil {
		return err
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("usuario: erro ao iniciar redefinição: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var id, codigoHash string
	err = tx.QueryRow(ctx, `
		select id, codigo_hash from recuperacao_senha
		where usuario_id = $1 and usado_em is null and expira_em > now() and tentativas < $2
		order by criado_em desc
		limit 1
		for update
	`, u.ID, MaximoTentativasCodigo).Scan(&id, &codigoHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrCodigoInvalido
	}
	if err != nil {
		return fmt.Errorf("usuario: erro ao buscar código: %w", err)
	}

	if !auth.VerificarSenha(codigoHash, codigo) {
		if _, err = tx.Exec(ctx, `update recuperacao_senha set tentativas = tentativas + 1 where id = $1`, id); err != nil {
			return fmt.Errorf("usuario: erro ao registrar tentativa: %w", err)
		}
		if err = tx.Commit(ctx); err != nil {
			return fmt.Errorf("usuario: erro ao registrar tentativa: %w", err)
		}
		return ErrCodigoInvalido
	}

	novoHash, err := auth.HashSenha(novaSenha)
	if err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `update usuario set senha_hash = $2 where id = $1`, u.ID, novoHash); err != nil {
		return fmt.Errorf("usuario: erro ao trocar senha: %w", err)
	}
	if _, err = tx.Exec(ctx, `update recuperacao_senha set usado_em = now() where id = $1`, id); err != nil {
		return fmt.Errorf("usuario: erro ao marcar código como usado: %w", err)
	}
	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("usuario: erro ao confirmar redefinição: %w", err)
	}
	return nil
}
