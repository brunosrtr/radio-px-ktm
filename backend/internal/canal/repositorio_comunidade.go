package canal

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// ErrConviteRepetido indica colisão do código de convite sorteado (raro);
// quem chama sorteia outro.
var ErrConviteRepetido = errors.New("canal: código de convite repetido")

func erroConvite(err error) error {
	var pg *pgconn.PgError
	if errors.As(err, &pg) && pg.Code == "23505" && pg.ConstraintName == "canal_codigo_convite_unico" {
		return ErrConviteRepetido
	}
	return err
}

// CanalDoUsuario é um canal listado para um usuário, com o que é dele.
type CanalDoUsuario struct {
	CanalDB
	Favorito bool
}

// ListarParaUsuario devolve os canais que o usuário pode usar (FR-029/030):
// os da empresa dele (regra da 001), os que criou, os privados em que entrou
// por convite e os públicos que favoritou — nunca os de que foi removido.
func (r *Repositorio) ListarParaUsuario(ctx context.Context, usuarioID, empresaID string) ([]CanalDoUsuario, error) {
	linhas, err := r.pool.Query(ctx, `
		select `+colunasCanalComPrefixo+`, f.canal_id is not null as favorito
		from canal c
		left join favorito_canal f on f.canal_id = c.id and f.usuario_id = $1
		left join acesso_canal a on a.canal_id = c.id and a.usuario_id = $1
		where c.ativo
		  and coalesce(a.situacao, '') <> 'banido'
		  and (
			($2 <> '' and (
				c.empresa_id::text = $2
				or c.id in (select canal_id from canal_empresa where empresa_id::text = $2)
			))
			or c.criador_id = $1
			or a.situacao = 'liberado'
			or (f.canal_id is not null and c.empresa_id is null and c.visibilidade = 'publico')
		  )
		order by favorito desc, c.nome
	`, usuarioID, empresaID)
	if err != nil {
		return nil, fmt.Errorf("canal: erro ao listar canais do usuário: %w", err)
	}
	return pgx.CollectRows(linhas, func(l pgx.CollectableRow) (CanalDoUsuario, error) {
		var c CanalDoUsuario
		err := escanearCanalCom(l, &c.CanalDB, &c.Favorito)
		return c, err
	})
}

// colunasCanalComPrefixo é colunasCanal com o alias "c." (para joins).
const colunasCanalComPrefixo = `c.id, coalesce(c.empresa_id::text, ''), coalesce(c.criador_id::text, ''), c.visibilidade,
	coalesce(c.codigo_convite, ''), c.nome, coalesce(c.descricao, ''), c.tipo_acesso,
	c.limite_participantes, c.geocerca_ativa, c.centro_latitude, c.centro_longitude,
	c.raio_metros, c.ativo`

func escanearCanalCom(linha interface{ Scan(...any) error }, c *CanalDB, extras ...any) error {
	destinos := []any{
		&c.ID, &c.EmpresaID, &c.CriadorID, &c.Visibilidade, &c.CodigoConvite, &c.Nome, &c.Descricao, &c.TipoAcesso,
		&c.LimiteParticipantes, &c.GeocercaAtiva, &c.CentroLatitude, &c.CentroLongitude,
		&c.RaioMetros, &c.Ativo,
	}
	return linha.Scan(append(destinos, extras...)...)
}

// ListarPublicos devolve até 200 canais públicos da comunidade, filtrados
// por nome quando [busca] não é vazia (FR-030), sem os de que o usuário foi
// removido.
func (r *Repositorio) ListarPublicos(ctx context.Context, usuarioID, busca string) ([]CanalDoUsuario, error) {
	linhas, err := r.pool.Query(ctx, `
		select `+colunasCanalComPrefixo+`, f.canal_id is not null as favorito
		from canal c
		left join favorito_canal f on f.canal_id = c.id and f.usuario_id = $1
		left join acesso_canal a on a.canal_id = c.id and a.usuario_id = $1
		where c.ativo and c.empresa_id is null and c.visibilidade = 'publico'
		  and coalesce(a.situacao, '') <> 'banido'
		  and ($2 = '' or lower(c.nome) like '%' || lower($2) || '%' or similarity(lower(c.nome), lower($2)) > 0.3)
		order by c.nome
		limit 200
	`, usuarioID, busca)
	if err != nil {
		return nil, fmt.Errorf("canal: erro ao buscar canais públicos: %w", err)
	}
	return pgx.CollectRows(linhas, func(l pgx.CollectableRow) (CanalDoUsuario, error) {
		var c CanalDoUsuario
		err := escanearCanalCom(l, &c.CanalDB, &c.Favorito)
		return c, err
	})
}

// SituacaoAcesso devolve "liberado", "banido" ou "" para o usuário no canal.
func (r *Repositorio) SituacaoAcesso(ctx context.Context, canalID, usuarioID string) (string, error) {
	var situacao string
	err := r.pool.QueryRow(ctx, `
		select situacao from acesso_canal where canal_id = $1 and usuario_id = $2
	`, canalID, usuarioID).Scan(&situacao)
	if errors.Is(err, pgx.ErrNoRows) {
		return SituacaoNenhuma, nil
	}
	if err != nil {
		return "", fmt.Errorf("canal: erro ao buscar acesso: %w", err)
	}
	return situacao, nil
}

// LiberarAcesso registra a entrada por convite. Nunca desfaz uma remoção:
// quem foi banido continua banido.
func (r *Repositorio) LiberarAcesso(ctx context.Context, canalID, usuarioID string) error {
	_, err := r.pool.Exec(ctx, `
		insert into acesso_canal (canal_id, usuario_id, situacao) values ($1, $2, 'liberado')
		on conflict (canal_id, usuario_id) do nothing
	`, canalID, usuarioID)
	if err != nil {
		return fmt.Errorf("canal: erro ao liberar acesso: %w", err)
	}
	return nil
}

// Banir marca o usuário como removido do canal e tira o canal dos favoritos
// dele.
func (r *Repositorio) Banir(ctx context.Context, canalID, usuarioID string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("canal: erro ao iniciar remoção: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err = tx.Exec(ctx, `
		insert into acesso_canal (canal_id, usuario_id, situacao) values ($1, $2, 'banido')
		on conflict (canal_id, usuario_id) do update set situacao = 'banido', atualizado_em = now()
	`, canalID, usuarioID); err != nil {
		return fmt.Errorf("canal: erro ao remover participante: %w", err)
	}
	if _, err = tx.Exec(ctx, `delete from favorito_canal where canal_id = $1 and usuario_id = $2`, canalID, usuarioID); err != nil {
		return fmt.Errorf("canal: erro ao remover favorito: %w", err)
	}
	return tx.Commit(ctx)
}

// ObterPorConvite busca o canal privado ativo com o código informado.
func (r *Repositorio) ObterPorConvite(ctx context.Context, codigo string) (*CanalDB, error) {
	var c CanalDB
	err := escanearCanal(r.pool.QueryRow(ctx, `
		select `+colunasCanal+` from canal
		where codigo_convite = $1 and ativo and visibilidade = 'privado'
	`, codigo), &c)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNaoEncontrado
	}
	if err != nil {
		return nil, fmt.Errorf("canal: erro ao buscar convite: %w", err)
	}
	return &c, nil
}

// DefinirConvite troca o código de convite do canal (o antigo deixa de
// valer).
func (r *Repositorio) DefinirConvite(ctx context.Context, canalID, codigo string) error {
	tag, err := r.pool.Exec(ctx, `update canal set codigo_convite = $2 where id = $1 and ativo`, canalID, codigo)
	if err != nil {
		return erroConvite(err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNaoEncontrado
	}
	return nil
}

// Apagar desativa o canal e invalida o convite. Participações e histórico
// continuam no banco.
func (r *Repositorio) Apagar(ctx context.Context, canalID string) error {
	tag, err := r.pool.Exec(ctx, `update canal set ativo = false, codigo_convite = null where id = $1 and ativo`, canalID)
	if err != nil {
		return fmt.Errorf("canal: erro ao apagar canal: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNaoEncontrado
	}
	return nil
}

// DefinirFavorito marca ou desmarca o canal como favorito do usuário.
func (r *Repositorio) DefinirFavorito(ctx context.Context, usuarioID, canalID string, favorito bool) error {
	var err error
	if favorito {
		_, err = r.pool.Exec(ctx, `
			insert into favorito_canal (usuario_id, canal_id) values ($1, $2) on conflict do nothing
		`, usuarioID, canalID)
	} else {
		_, err = r.pool.Exec(ctx, `delete from favorito_canal where usuario_id = $1 and canal_id = $2`, usuarioID, canalID)
	}
	if err != nil {
		return fmt.Errorf("canal: erro ao definir favorito: %w", err)
	}
	return nil
}

// EhFavorito informa se o usuário favoritou o canal.
func (r *Repositorio) EhFavorito(ctx context.Context, usuarioID, canalID string) (bool, error) {
	var fav bool
	err := r.pool.QueryRow(ctx, `
		select exists(select 1 from favorito_canal where usuario_id = $1 and canal_id = $2)
	`, usuarioID, canalID).Scan(&fav)
	return fav, err
}
