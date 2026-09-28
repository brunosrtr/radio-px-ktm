package canal

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"

	"github.com/brunosrtr/radio-px-ktm/backend/internal/storage"
)

// ErrNaoEncontrado indica que o canal não existe ou está inativo.
var ErrNaoEncontrado = errors.New("canal: não encontrado")

// CanalDB é a representação persistida de um canal (data-model.md), distinta
// do hub em memória Canal.
type CanalDB struct {
	ID                  string
	EmpresaID           string
	Nome                string
	Descricao           string
	TipoAcesso          string
	LimiteParticipantes int
	GeocercaAtiva       bool
	CentroLatitude      *float64
	CentroLongitude     *float64
	RaioMetros          *int
	Ativo               bool
}

// Repositorio persiste canais e o histórico de participação, liberação de
// empresas parceiras e preferências de silenciamento.
type Repositorio struct {
	pool *storage.Pool
}

// NovoRepositorio cria um Repositorio sobre o pool de conexões informado.
func NovoRepositorio(pool *storage.Pool) *Repositorio {
	return &Repositorio{pool: pool}
}

const colunasCanal = `id, empresa_id, nome, coalesce(descricao, ''), tipo_acesso,
	limite_participantes, geocerca_ativa, centro_latitude, centro_longitude,
	raio_metros, ativo`

func escanearCanal(linha interface{ Scan(...any) error }, c *CanalDB) error {
	return linha.Scan(
		&c.ID, &c.EmpresaID, &c.Nome, &c.Descricao, &c.TipoAcesso,
		&c.LimiteParticipantes, &c.GeocercaAtiva, &c.CentroLatitude, &c.CentroLongitude,
		&c.RaioMetros, &c.Ativo,
	)
}

// CriarCanal insere um novo canal.
func (r *Repositorio) CriarCanal(ctx context.Context, c CanalDB) (*CanalDB, error) {
	err := escanearCanal(r.pool.QueryRow(ctx, `
		insert into canal (
			empresa_id, nome, descricao, tipo_acesso, limite_participantes,
			geocerca_ativa, centro_latitude, centro_longitude, raio_metros
		)
		values ($1, $2, nullif($3, ''), $4, $5, $6, $7, $8, $9)
		returning `+colunasCanal,
		c.EmpresaID, c.Nome, c.Descricao, c.TipoAcesso, c.LimiteParticipantes,
		c.GeocercaAtiva, c.CentroLatitude, c.CentroLongitude, c.RaioMetros,
	), &c)
	if err != nil {
		return nil, fmt.Errorf("canal: erro ao criar canal: %w", err)
	}
	return &c, nil
}

// ObterCanal busca um canal ativo pelo ID.
func (r *Repositorio) ObterCanal(ctx context.Context, id string) (*CanalDB, error) {
	var c CanalDB
	err := escanearCanal(r.pool.QueryRow(ctx, `
		select `+colunasCanal+`
		from canal
		where id = $1 and ativo
	`, id), &c)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNaoEncontrado
	}
	if err != nil {
		return nil, fmt.Errorf("canal: erro ao buscar canal %s: %w", id, err)
	}
	return &c, nil
}

// AtualizarCanal grava os campos editáveis do canal.
func (r *Repositorio) AtualizarCanal(ctx context.Context, c CanalDB) (*CanalDB, error) {
	err := escanearCanal(r.pool.QueryRow(ctx, `
		update canal set
			nome = $2,
			descricao = nullif($3, ''),
			tipo_acesso = $4,
			limite_participantes = $5,
			geocerca_ativa = $6,
			centro_latitude = $7,
			centro_longitude = $8,
			raio_metros = $9
		where id = $1 and ativo
		returning `+colunasCanal,
		c.ID, c.Nome, c.Descricao, c.TipoAcesso, c.LimiteParticipantes,
		c.GeocercaAtiva, c.CentroLatitude, c.CentroLongitude, c.RaioMetros,
	), &c)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNaoEncontrado
	}
	if err != nil {
		return nil, fmt.Errorf("canal: erro ao atualizar canal %s: %w", c.ID, err)
	}
	return &c, nil
}

// ListarAutorizados retorna os canais ativos que a empresa pode acessar:
// os privados de que é dona e os compartilhados liberados para ela (FR-010,
// FR-012, FR-029, RNF12).
func (r *Repositorio) ListarAutorizados(ctx context.Context, empresaID string) ([]CanalDB, error) {
	linhas, err := r.pool.Query(ctx, `
		select `+colunasCanal+`
		from canal
		where ativo and (
			empresa_id = $1
			or id in (select canal_id from canal_empresa where empresa_id = $1)
		)
		order by nome
	`, empresaID)
	if err != nil {
		return nil, fmt.Errorf("canal: erro ao listar canais autorizados: %w", err)
	}
	defer linhas.Close()

	var canais []CanalDB
	for linhas.Next() {
		var c CanalDB
		if err := escanearCanal(linhas, &c); err != nil {
			return nil, fmt.Errorf("canal: erro ao ler canal listado: %w", err)
		}
		canais = append(canais, c)
	}
	if err := linhas.Err(); err != nil {
		return nil, fmt.Errorf("canal: erro ao percorrer canais listados: %w", err)
	}
	return canais, nil
}

// EstaAutorizada indica se a empresa pode acessar o canal — por ser a dona
// ou por ter sido liberada como parceira (RNF12).
func (r *Repositorio) EstaAutorizada(ctx context.Context, canalID, empresaID string) (bool, error) {
	var autorizada bool
	err := r.pool.QueryRow(ctx, `
		select exists(
			select 1 from canal where id = $1 and empresa_id = $2
			union
			select 1 from canal_empresa where canal_id = $1 and empresa_id = $2
		)
	`, canalID, empresaID).Scan(&autorizada)
	if err != nil {
		return false, fmt.Errorf("canal: erro ao verificar autorização: %w", err)
	}
	return autorizada, nil
}

// LiberarEmpresa concede a uma empresa parceira acesso a um canal
// compartilhado (FR-012/FR-013).
func (r *Repositorio) LiberarEmpresa(ctx context.Context, canalID, empresaID string) error {
	_, err := r.pool.Exec(ctx, `
		insert into canal_empresa (canal_id, empresa_id)
		values ($1, $2)
		on conflict (canal_id, empresa_id) do nothing
	`, canalID, empresaID)
	if err != nil {
		return fmt.Errorf("canal: erro ao liberar empresa %s no canal %s: %w", empresaID, canalID, err)
	}
	return nil
}

// RevogarEmpresa remove o acesso de uma empresa parceira a um canal.
func (r *Repositorio) RevogarEmpresa(ctx context.Context, canalID, empresaID string) error {
	_, err := r.pool.Exec(ctx, `
		delete from canal_empresa where canal_id = $1 and empresa_id = $2
	`, canalID, empresaID)
	if err != nil {
		return fmt.Errorf("canal: erro ao revogar empresa %s no canal %s: %w", empresaID, canalID, err)
	}
	return nil
}

// DefinirPreferencia grava se o usuário silenciou (ou reativou) um canal
// (FR-019), sem sair dele.
func (r *Repositorio) DefinirPreferencia(ctx context.Context, usuarioID, canalID string, silenciado bool) error {
	_, err := r.pool.Exec(ctx, `
		insert into preferencia_canal (usuario_id, canal_id, silenciado)
		values ($1, $2, $3)
		on conflict (usuario_id, canal_id) do update set silenciado = excluded.silenciado
	`, usuarioID, canalID, silenciado)
	if err != nil {
		return fmt.Errorf("canal: erro ao definir preferência: %w", err)
	}
	return nil
}

// ObterPreferencia retorna se o usuário silenciou o canal informado.
func (r *Repositorio) ObterPreferencia(ctx context.Context, usuarioID, canalID string) (bool, error) {
	var silenciado bool
	err := r.pool.QueryRow(ctx, `
		select silenciado from preferencia_canal where usuario_id = $1 and canal_id = $2
	`, usuarioID, canalID).Scan(&silenciado)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("canal: erro ao buscar preferência: %w", err)
	}
	return silenciado, nil
}

// PreferenciasSilenciadas retorna o conjunto de canais que o usuário
// silenciou, para montar a listagem de GET /canais.
func (r *Repositorio) PreferenciasSilenciadas(ctx context.Context, usuarioID string) (map[string]bool, error) {
	linhas, err := r.pool.Query(ctx, `
		select canal_id, silenciado from preferencia_canal where usuario_id = $1
	`, usuarioID)
	if err != nil {
		return nil, fmt.Errorf("canal: erro ao listar preferências: %w", err)
	}
	defer linhas.Close()

	preferencias := make(map[string]bool)
	for linhas.Next() {
		var canalID string
		var silenciado bool
		if err := linhas.Scan(&canalID, &silenciado); err != nil {
			return nil, fmt.Errorf("canal: erro ao ler preferência: %w", err)
		}
		preferencias[canalID] = silenciado
	}
	if err := linhas.Err(); err != nil {
		return nil, fmt.Errorf("canal: erro ao percorrer preferências: %w", err)
	}
	return preferencias, nil
}

// RegistrarEntrada grava uma nova participação ativa do usuário no canal. A
// constraint idx_participacao_ativa do banco impede duas participações
// ativas simultâneas do mesmo usuário no mesmo canal.
// RegistrarEntrada fecha qualquer participação ativa anterior do usuário no
// canal antes de abrir uma nova. Isso é necessário porque uma conexão
// anterior pode ter caído sem o servidor rodar RegistrarSaida (motorista
// perdeu sinal, aba fechada abruptamente, processo reiniciado) — sem fechar
// essa linha órfã primeiro, o índice único idx_participacao_ativa
// (canal_id, usuario_id) where saiu_em is null recusaria a reentrada com um
// erro de conflito, bloqueando o motorista de voltar ao canal.
func (r *Repositorio) RegistrarEntrada(ctx context.Context, canalID, usuarioID string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("canal: erro ao iniciar transação de entrada: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `
		update participacao_canal
		set saiu_em = now(), motivo_saida = 'desconexao'
		where canal_id = $1 and usuario_id = $2 and saiu_em is null
	`, canalID, usuarioID); err != nil {
		return fmt.Errorf("canal: erro ao encerrar participação anterior: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		insert into participacao_canal (canal_id, usuario_id)
		values ($1, $2)
	`, canalID, usuarioID); err != nil {
		return fmt.Errorf("canal: erro ao registrar entrada: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("canal: erro ao confirmar entrada: %w", err)
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
