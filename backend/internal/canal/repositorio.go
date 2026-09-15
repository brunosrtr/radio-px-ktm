package canal

import (
	"context"
	"fmt"

	"github.com/brunosrtr/radio-px-ktm/backend/internal/storage"
)

// Repositorio persiste o histórico de participação em canais (RF-16, RF-18,
// RF-23). O registro em memória de quem está no canal agora é
// responsabilidade do Gerenciador/Canal — esta tabela é o histórico
// auditável, não o estado ativo do hub.
type Repositorio struct {
	pool *storage.Pool
}

// NovoRepositorio cria um Repositorio sobre o pool de conexões informado.
func NovoRepositorio(pool *storage.Pool) *Repositorio {
	return &Repositorio{pool: pool}
}

// RegistrarEntrada grava uma nova participação ativa do usuário no canal. A
// constraint idx_participacao_ativa do banco impede duas participações
// ativas simultâneas do mesmo usuário no mesmo canal.
func (r *Repositorio) RegistrarEntrada(ctx context.Context, canalID, usuarioID string) error {
	_, err := r.pool.Exec(ctx, `
		insert into participacao_canal (canal_id, usuario_id)
		values ($1, $2)
	`, canalID, usuarioID)
	if err != nil {
		return fmt.Errorf("canal: erro ao registrar entrada: %w", err)
	}
	return nil
}

// RegistrarSaida encerra a participação ativa do usuário no canal com o
// motivo informado ('manual', 'geocerca', 'desconexao' ou 'encerramento').
func (r *Repositorio) RegistrarSaida(ctx context.Context, canalID, usuarioID, motivo string) error {
	_, err := r.pool.Exec(ctx, `
		update participacao_canal
		set saiu_em = now(), motivo_saida = $3
		where canal_id = $1 and usuario_id = $2 and saiu_em is null
	`, canalID, usuarioID, motivo)
	if err != nil {
		return fmt.Errorf("canal: erro ao registrar saída: %w", err)
	}
	return nil
}

// NomeUsuario busca o nome do usuário para identificar o remetente de uma
// transmissão (FR-21). Provisório até a User Story 2 introduzir um pacote
// usuario dedicado (T033) — aqui cobre só o necessário para a US1.
func (r *Repositorio) NomeUsuario(ctx context.Context, usuarioID string) (string, error) {
	var nome string
	err := r.pool.QueryRow(ctx, `select nome from usuario where id = $1`, usuarioID).Scan(&nome)
	if err != nil {
		return "", fmt.Errorf("canal: erro ao buscar nome do usuário: %w", err)
	}
	return nome, nil
}
