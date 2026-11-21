package orbit

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/brunosrtr/radio-px-ktm/backend/internal/storage"
)

// Repositorio persiste leituras, Orbits, pontos fixos e contadores de
// contribuição, com SQL escrito à mão (Princípio V).
type Repositorio struct {
	pool *storage.Pool
}

// NovoRepositorio cria o Repositorio sobre o pool informado.
func NovoRepositorio(pool *storage.Pool) *Repositorio {
	return &Repositorio{pool: pool}
}

// IngerirLote grava um lote numa única transação:
//   - cria os Orbits ainda desconhecidos;
//   - insere as leituras ignorando as já gravadas (mesmo orbit, ponto e
//     horário), para que reenviar um lote após falha de rede não duplique;
//   - atualiza a última posição de cada Orbit só quando a leitura nova é mais
//     recente que a registrada — um lote atrasado nunca faz o Orbit "voltar";
//   - soma em contribuicao_diaria só as leituras novas, quando usuarioID não
//     é vazio (leituras de ponto fixo não têm usuário).
func (r *Repositorio) IngerirLote(ctx context.Context, origem, pontoID, usuarioID string, leituras []LeituraRecebida) (ResultadoIngestao, error) {
	var resultado ResultadoIngestao

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return resultado, fmt.Errorf("orbit: erro ao iniciar ingestão: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	ids := make([]string, 0, len(leituras))
	for _, l := range leituras {
		ids = append(ids, l.OrbitID)
	}
	if _, err = tx.Exec(ctx, `
		insert into orbit (id) select distinct unnest($1::text[]) on conflict (id) do nothing
	`, ids); err != nil {
		return resultado, fmt.Errorf("orbit: erro ao registrar Orbits: %w", err)
	}

	idsNovos := make([]int64, 0, len(leituras))
	porDia := map[string]int{}
	for _, l := range leituras {
		registrada := LeituraRegistrada{
			OrbitID: l.OrbitID, Origem: origem, PontoID: pontoID,
			Latitude: l.Latitude, Longitude: l.Longitude,
			PrecisaoMetros: l.PrecisaoMetros, RSSI: l.RSSI, DetectadoEm: l.DetectadoEm.UTC(),
		}
		err = tx.QueryRow(ctx, `
			insert into leitura_orbit (orbit_id, origem, ponto_id, latitude, longitude, precisao_metros, rssi, detectado_em)
			values ($1, $2, $3, $4, $5, $6, $7, $8)
			on conflict on constraint leitura_orbit_unica do nothing
			returning id, recebido_em
		`, l.OrbitID, origem, pontoID, l.Latitude, l.Longitude, l.PrecisaoMetros, l.RSSI, l.DetectadoEm).
			Scan(&registrada.ID, &registrada.RecebidoEm)
		if errors.Is(err, pgx.ErrNoRows) {
			resultado.Duplicadas++
			continue
		}
		if err != nil {
			return resultado, fmt.Errorf("orbit: erro ao gravar leitura: %w", err)
		}
		resultado.Aceitas++
		resultado.Novas = append(resultado.Novas, registrada)
		idsNovos = append(idsNovos, registrada.ID)
		porDia[Dia(l.DetectadoEm)]++
	}

	if len(idsNovos) > 0 {
		linhas, err := tx.Query(ctx, `
			update orbit o set
				ultima_latitude = l.latitude,
				ultima_longitude = l.longitude,
				ultima_leitura_em = l.detectado_em
			from (
				select distinct on (orbit_id) orbit_id, latitude, longitude, detectado_em
				from leitura_orbit
				where id = any($1)
				order by orbit_id, detectado_em desc
			) l
			where o.id = l.orbit_id
			  and (o.ultima_leitura_em is null or l.detectado_em > o.ultima_leitura_em)
			returning o.id
		`, idsNovos)
		if err != nil {
			return resultado, fmt.Errorf("orbit: erro ao atualizar última posição: %w", err)
		}
		atualizados, err := pgx.CollectRows(linhas, pgx.RowTo[string])
		if err != nil {
			return resultado, fmt.Errorf("orbit: erro ao ler Orbits atualizados: %w", err)
		}

		if usuarioID != "" {
			for dia, total := range porDia {
				if _, err = tx.Exec(ctx, `
					insert into contribuicao_diaria (usuario_id, dia, leituras) values ($1, $2, $3)
					on conflict (usuario_id, dia) do update set leituras = contribuicao_diaria.leituras + excluded.leituras
				`, usuarioID, dia, total); err != nil {
					return resultado, fmt.Errorf("orbit: erro ao somar contribuição: %w", err)
				}
			}
		}

		if len(atualizados) > 0 {
			resultado.OrbitsAtualizados, err = listarOrbits(ctx, tx, atualizados)
			if err != nil {
				return resultado, err
			}
		}
	}

	if err = tx.Commit(ctx); err != nil {
		return resultado, fmt.Errorf("orbit: erro ao confirmar ingestão: %w", err)
	}
	return resultado, nil
}

type consultador interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

// listarOrbits devolve o resumo dos Orbits informados (todos, se ids for
// nil), com a contagem de leituras e de pontos distintos nas últimas 24 h.
func listarOrbits(ctx context.Context, db consultador, ids []string) ([]OrbitResumo, error) {
	linhas, err := db.Query(ctx, `
		select o.id, o.virtual, o.rotulo, o.ultima_latitude::float8, o.ultima_longitude::float8, o.ultima_leitura_em,
		       coalesce(r.leituras, 0), coalesce(r.pontos, 0)
		from orbit o
		left join lateral (
			select count(*)::int as leituras, count(distinct ponto_id)::int as pontos
			from leitura_orbit l
			where l.orbit_id = o.id and l.detectado_em > now() - interval '24 hours'
		) r on true
		where $1::text[] is null or o.id = any($1)
		order by o.ultima_leitura_em desc nulls last, o.id
	`, ids)
	if err != nil {
		return nil, fmt.Errorf("orbit: erro ao listar Orbits: %w", err)
	}
	orbits, err := pgx.CollectRows(linhas, func(l pgx.CollectableRow) (OrbitResumo, error) {
		var o OrbitResumo
		err := l.Scan(&o.ID, &o.Virtual, &o.Rotulo, &o.Latitude, &o.Longitude, &o.UltimaLeituraEm, &o.Leituras24h, &o.PontosDistintos24h)
		return o, err
	})
	if err != nil {
		return nil, fmt.Errorf("orbit: erro ao ler Orbits: %w", err)
	}
	return orbits, nil
}

// Orbits devolve o resumo de todos os Orbits conhecidos (painel).
func (r *Repositorio) Orbits(ctx context.Context) ([]OrbitResumo, error) {
	return listarOrbits(ctx, r.pool, nil)
}

// DefinirOrbit cria ou atualiza um Orbit com rótulo e marca de virtual
// (seed da demonstração).
func (r *Repositorio) DefinirOrbit(ctx context.Context, id string, virtual bool, rotulo string) error {
	if !OrbitIDValido(id) {
		return fmt.Errorf("%w: id de Orbit inválido", ErrDadosInvalidos)
	}
	_, err := r.pool.Exec(ctx, `
		insert into orbit (id, virtual, rotulo) values ($1, $2, nullif($3, ''))
		on conflict (id) do update set virtual = excluded.virtual, rotulo = excluded.rotulo
	`, id, virtual, rotulo)
	if err != nil {
		return fmt.Errorf("orbit: erro ao definir Orbit: %w", err)
	}
	return nil
}

// HistoricoOrbit devolve as leituras de um Orbit no período, em ordem
// cronológica (FR-019), limitadas às 5000 mais recentes.
func (r *Repositorio) HistoricoOrbit(ctx context.Context, orbitID string, de, ate time.Time) ([]LeituraRegistrada, error) {
	var existe bool
	if err := r.pool.QueryRow(ctx, `select exists(select 1 from orbit where id = $1)`, orbitID).Scan(&existe); err != nil {
		return nil, fmt.Errorf("orbit: erro ao buscar Orbit: %w", err)
	}
	if !existe {
		return nil, ErrNaoEncontrado
	}
	linhas, err := r.pool.Query(ctx, `
		select * from (
			select id, orbit_id, origem, ponto_id, latitude::float8, longitude::float8, precisao_metros::float8, rssi, detectado_em, recebido_em
			from leitura_orbit
			where orbit_id = $1 and detectado_em between $2 and $3
			order by detectado_em desc
			limit 5000
		) recentes order by detectado_em
	`, orbitID, de, ate)
	if err != nil {
		return nil, fmt.Errorf("orbit: erro ao buscar histórico: %w", err)
	}
	historico, err := pgx.CollectRows(linhas, func(l pgx.CollectableRow) (LeituraRegistrada, error) {
		var h LeituraRegistrada
		var rssi *int16
		err := l.Scan(&h.ID, &h.OrbitID, &h.Origem, &h.PontoID, &h.Latitude, &h.Longitude, &h.PrecisaoMetros, &rssi, &h.DetectadoEm, &h.RecebidoEm)
		if rssi != nil {
			v := int(*rssi)
			h.RSSI = &v
		}
		return h, err
	})
	if err != nil {
		return nil, fmt.Errorf("orbit: erro ao ler histórico: %w", err)
	}
	return historico, nil
}

// Contribuicao devolve os contadores de leituras do usuário (FR-040): hoje,
// últimos 7 dias (incluindo hoje) e total.
func (r *Repositorio) Contribuicao(ctx context.Context, usuarioID string, agora time.Time) (hoje, semana, total int, err error) {
	hojeDia := Dia(agora)
	inicioSemana := Dia(agora.AddDate(0, 0, -6))
	err = r.pool.QueryRow(ctx, `
		select coalesce(sum(leituras) filter (where dia = $2::date), 0)::int,
		       coalesce(sum(leituras) filter (where dia >= $3::date), 0)::int,
		       coalesce(sum(leituras), 0)::int
		from contribuicao_diaria where usuario_id = $1
	`, usuarioID, hojeDia, inicioSemana).Scan(&hoje, &semana, &total)
	if err != nil {
		err = fmt.Errorf("orbit: erro ao calcular contribuição: %w", err)
	}
	return hoje, semana, total, err
}

// PontosFixos lista os pontos fixos cadastrados.
func (r *Repositorio) PontosFixos(ctx context.Context) ([]PontoFixo, error) {
	linhas, err := r.pool.Query(ctx, `
		select id, nome, latitude::float8, longitude::float8, raio_metros, simulado from ponto_fixo order by nome
	`)
	if err != nil {
		return nil, fmt.Errorf("orbit: erro ao listar pontos fixos: %w", err)
	}
	return pgx.CollectRows(linhas, func(l pgx.CollectableRow) (PontoFixo, error) {
		var p PontoFixo
		err := l.Scan(&p.ID, &p.Nome, &p.Latitude, &p.Longitude, &p.RaioMetros, &p.Simulado)
		return p, err
	})
}

// ObterPontoFixo busca um ponto fixo pelo ID.
func (r *Repositorio) ObterPontoFixo(ctx context.Context, id string) (*PontoFixo, error) {
	var p PontoFixo
	err := r.pool.QueryRow(ctx, `
		select id, nome, latitude::float8, longitude::float8, raio_metros, simulado from ponto_fixo where id::text = $1
	`, id).Scan(&p.ID, &p.Nome, &p.Latitude, &p.Longitude, &p.RaioMetros, &p.Simulado)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNaoEncontrado
	}
	if err != nil {
		return nil, fmt.Errorf("orbit: erro ao buscar ponto fixo: %w", err)
	}
	return &p, nil
}

// CriarPontoFixo cadastra um ponto fixo (FR-020).
func (r *Repositorio) CriarPontoFixo(ctx context.Context, p PontoFixo) (*PontoFixo, error) {
	if err := ValidarPontoFixo(p); err != nil {
		return nil, err
	}
	err := r.pool.QueryRow(ctx, `
		insert into ponto_fixo (nome, latitude, longitude, raio_metros, simulado)
		values ($1, $2, $3, $4, $5) returning id
	`, p.Nome, p.Latitude, p.Longitude, p.RaioMetros, p.Simulado).Scan(&p.ID)
	if err != nil {
		return nil, fmt.Errorf("orbit: erro ao criar ponto fixo: %w", err)
	}
	return &p, nil
}

// ApagarPontoFixo remove um ponto fixo. As leituras feitas por ele ficam no
// histórico.
func (r *Repositorio) ApagarPontoFixo(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx, `delete from ponto_fixo where id::text = $1`, id)
	if err != nil {
		return fmt.Errorf("orbit: erro ao apagar ponto fixo: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNaoEncontrado
	}
	return nil
}
