package usuario

import (
	"context"
	"fmt"
)

// ExcluirConta remove os dados pessoais da conta (FR-026). A linha de
// usuario é mantida, anonimizada e inativa, porque participações em canais
// e consentimentos apontam para ela; o mesmo e-mail pode criar uma conta nova
// depois. Também apaga o histórico e a posição atual, dispositivos,
// preferências e códigos de recuperação, e encerra participações abertas.
func (r *Repositorio) ExcluirConta(ctx context.Context, usuarioID string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("usuario: erro ao iniciar exclusão: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	tag, err := tx.Exec(ctx, `
		update usuario set
			nome = 'Conta excluída',
			apelido = null,
			email = null,
			login = 'excluido-' || id::text,
			primeiro_nome = null,
			sobrenome = null,
			cpf = null,
			senha_hash = '',
			ativo = false,
			excluido_em = now()
		where id = $1 and excluido_em is null
	`, usuarioID)
	if err != nil {
		return fmt.Errorf("usuario: erro ao anonimizar conta: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNaoEncontrado
	}

	limpezas := []string{
		`delete from posicao where usuario_id = $1`,
		`delete from posicao_atual where usuario_id = $1`,
		`delete from dispositivo where usuario_id = $1`,
		`delete from preferencia_canal where usuario_id = $1`,
		`delete from recuperacao_senha where usuario_id = $1`,
		`update participacao_canal set saiu_em = now(), motivo_saida = 'desconexao'
			where usuario_id = $1 and saiu_em is null`,
	}
	for _, comando := range limpezas {
		if _, err = tx.Exec(ctx, comando, usuarioID); err != nil {
			return fmt.Errorf("usuario: erro na exclusão da conta: %w", err)
		}
	}

	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("usuario: erro ao confirmar exclusão: %w", err)
	}
	return nil
}
